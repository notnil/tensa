package fusion

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
	"github.com/notnil/tensa/pkg/tennis/court2d"
)

// FusedProvider implements a location provider that fuses absolute and relative feeds.
type FusedProvider struct {
	logger     *slog.Logger
	absSub     pubsubx.Sub[metrics.Metric[location.Loc]]
	relSub     pubsubx.Sub[metrics.Metric[location.Loc]]
	kf         *SimpleKalmanFilter
	mu         sync.Mutex
	lastRel    *location.Loc
	lastUpdate time.Time
}

// NewFusedProvider creates a new FusedProvider.
func NewFusedProvider(
	logger *slog.Logger,
	absSub pubsubx.Sub[metrics.Metric[location.Loc]],
	relSub pubsubx.Sub[metrics.Metric[location.Loc]],
) *FusedProvider {
	return &FusedProvider{
		logger: logger.With("component", "fused_provider"),
		absSub: absSub,
		relSub: relSub,
		kf:     NewSimpleKalmanFilter(),
	}
}

// Subscribe starts the fusion process and sends updates to the provided channel.
func (p *FusedProvider) Subscribe(ctx context.Context, ch chan<- metrics.Metric[location.Loc]) error {
	absCh := make(chan metrics.Metric[location.Loc], 10)
	relCh := make(chan metrics.Metric[location.Loc], 100) // Higher buffer for high freq

	// Start subscribers
	// We need a way to start them. If they block, we need goroutines.
	// Assuming Subscribe blocks until context done, we run them in goroutines.

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Absolute feed (AI)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.absSub.Subscribe(ctx, absCh); err != nil {
			p.logger.Error("Absolute feed subscription ended", "error", err)
		}
	}()

	// Relative feed (Zed)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.relSub.Subscribe(ctx, relCh); err != nil {
			p.logger.Error("Relative feed subscription ended", "error", err)
		}
	}()

	// Process loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-absCh:
				if !ok {
					return
				}
				p.handleAbsolute(m, ch)
			case m, ok := <-relCh:
				if !ok {
					return
				}
				p.handleRelative(m, ch)
			}
		}
	}()

	wg.Wait()
	return nil
}

func (p *FusedProvider) handleAbsolute(m metrics.Metric[location.Loc], outCh chan<- metrics.Metric[location.Loc]) {
	// Skip invalid measurements (zero location is likely invalid)
	if m.Value.IsZero() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Measurement update
	// Z = [x, y, cos(theta), sin(theta)]
	zX, zY := m.Value.Location.X, m.Value.Location.Y
	theta := m.Value.Rotation
	zCos, zSin := math.Cos(theta), math.Sin(theta)

	p.kf.Correct(zX, zY, zCos, zSin)

	// Optionally publish on correction too, but usually high freq feed drives output
}

func (p *FusedProvider) handleRelative(m metrics.Metric[location.Loc], outCh chan<- metrics.Metric[location.Loc]) {
	p.mu.Lock()

	currentLoc := m.Value
	now := m.Timestamp

	if p.lastRel == nil {
		p.lastRel = &currentLoc
		p.lastUpdate = now
		p.mu.Unlock()
		return
	}

	// Calculate deltas from relative location feed
	dx := currentLoc.Location.X - p.lastRel.Location.X
	dy := currentLoc.Location.Y - p.lastRel.Location.Y
	dYaw := currentLoc.Rotation - p.lastRel.Rotation

	// Calculate dt for time-scaled noise
	dt := now.Sub(p.lastUpdate).Seconds()
	if dt <= 0 {
		dt = 0.001 // Prevent division by zero, assume 1ms minimum
	}

	// Predict with time-scaled process noise
	p.kf.Predict(dx, dy, dYaw, dt)

	// Get estimated state
	x, y, rot := p.kf.GetState()

	p.lastRel = &currentLoc
	p.lastUpdate = now

	p.mu.Unlock()

	// Non-blocking send to output channel
	select {
	case outCh <- metrics.Metric[location.Loc]{
		Timestamp: now,
		Value: location.Loc{
			Location: court2d.Point{X: x, Y: y},
			Rotation: rot,
		},
	}:
	default:
		p.logger.Warn("Output channel full, dropping fused location")
	}
}

// SimpleKalmanFilter handles 2D position and rotation fusion.
type SimpleKalmanFilter struct {
	// State: [x, y]
	x, y float64

	// Rotation state: [cos, sin] (not normalized in storage to allow averaging, but normalized on read)
	rc, rs float64

	// Covariance or gain - kept simple here (constant gain or simplified)
	// For a full KF, we need P matrix.
	// Let's implement a simplified KF with fixed Process Noise and Measurement Noise.

	pxx, pyy float64 // Variance for x, y
	prr      float64 // Variance for rotation (scalar for combined cos/sin approx)

	// Configuration
	qPos float64 // Process noise per second (motion uncertainty)
	rPos float64 // Measurement noise (AI uncertainty)

	qRot float64 // Rotation process noise per second
	rRot float64

	// Initialization tracking
	initialized bool
}

func NewSimpleKalmanFilter() *SimpleKalmanFilter {
	return &SimpleKalmanFilter{
		// Initialize with some defaults
		rc: 1.0, rs: 0.0, // 0 angle

		pxx: 1.0, pyy: 1.0, prr: 0.1,

		// Tune these
		qPos: 0.01, // Low process noise (trust dead reckoning between updates)
		rPos: 0.5,  // High measurement noise (trust AI less/smooth it)

		qRot: 0.001,
		rRot: 0.1,
	}
}

func (kf *SimpleKalmanFilter) Predict(dx, dy, dTheta, dt float64) {
	// Update state with control input (odometry/relative)

	// If the relative feed provides deltas in world frame (like SLAM/VIO),
	// add them directly. If they're in body frame, they should be rotated first.
	// For this implementation, we assume world-frame deltas.
	kf.x += dx
	kf.y += dy

	// Rotation update
	// dTheta is change in angle.
	// New angle = Old angle + dTheta
	cd, sd := math.Cos(dTheta), math.Sin(dTheta)

	// Rotate [rc, rs]
	newRc := kf.rc*cd - kf.rs*sd
	newRs := kf.rc*sd + kf.rs*cd

	kf.rc = newRc
	kf.rs = newRs

	// Normalize to keep it stable (it's a direction vector)
	norm := math.Hypot(kf.rc, kf.rs)
	if norm > 1e-9 {
		kf.rc /= norm
		kf.rs /= norm
	}

	// Increase uncertainty scaled by time
	kf.pxx += kf.qPos * dt
	kf.pyy += kf.qPos * dt
	kf.prr += kf.qRot * dt
}

func (kf *SimpleKalmanFilter) Correct(zx, zy, zc, zs float64) {
	// On first absolute measurement, initialize state directly
	if !kf.initialized {
		kf.x = zx
		kf.y = zy
		kf.rc = zc
		kf.rs = zs
		// Normalize
		norm := math.Hypot(kf.rc, kf.rs)
		if norm > 1e-9 {
			kf.rc /= norm
			kf.rs /= norm
		}
		kf.initialized = true
		return
	}

	// 1. Position Update
	// K = P / (P + R)
	kx := kf.pxx / (kf.pxx + kf.rPos)
	ky := kf.pyy / (kf.pyy + kf.rPos)

	kf.x = kf.x + kx*(zx-kf.x)
	kf.y = kf.y + ky*(zy-kf.y)

	// Update covariance
	kf.pxx = (1 - kx) * kf.pxx
	kf.pyy = (1 - ky) * kf.pyy

	// 2. Rotation Update
	// We treat [rc, rs] as state. Measurement is [zc, zs].
	// A standard KF on x,y of the unit vector works fine if we normalize.

	// Gain
	kr := kf.prr / (kf.prr + kf.rRot)

	kf.rc = kf.rc + kr*(zc-kf.rc)
	kf.rs = kf.rs + kr*(zs-kf.rs)

	// Update covariance
	kf.prr = (1 - kr) * kf.prr

	// Normalize
	norm := math.Hypot(kf.rc, kf.rs)
	if norm > 1e-9 {
		kf.rc /= norm
		kf.rs /= norm
	}
}

func (kf *SimpleKalmanFilter) GetState() (float64, float64, float64) {
	return kf.x, kf.y, math.Atan2(kf.rs, kf.rc)
}
