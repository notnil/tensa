package zltech

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/notnil/tensa/pkg/hware/wheels"
	"golang.org/x/sync/errgroup"
)

const (
	// Bus 001 Device 017: ID 1d50:606f OpenMoko, Inc. Geschwister Schneider CAN adapter
	DefaultVendorID = "1d50"

	NodeID1 = 0x01
	NodeID2 = 0x02

	Bitrate    = 500000
	RestartMs  = 100
	TxQueueLen = 2048
)

// radsToRPM converts wheel speeds from rad/s to RPM and vice-versa
const radsToRPM = 9.549297

// Wheels implements wheels.Wheels using two ZLAC8015D (ZLTech) dual-motor drivers.
// Each client controls two motors; we map the first client to the left-side wheels
// (FrontLeft, RearLeft) and the second client to the right-side wheels (FrontRight, RearRight).
// Within a driver, Side Left corresponds to the front wheel and Side Right to the rear wheel.
type Wheels struct {
	c1        *Client
	c2        *Client
	log       *slog.Logger
	moveAcc   uint32 // acceleration in ms
	moveDec   uint32 // deceleration in ms
	rotateAcc uint32 // acceleration in ms
	rotateDec uint32 // deceleration in ms
}

// WheelsOption customizes the zltech Wheels behavior.
type WheelsOption func(*Wheels)

// WithWheelsLogger sets the logger for the zltech Wheels wrapper.
func WithWheelsLogger(l *slog.Logger) WheelsOption {
	return func(w *Wheels) {
		if l != nil {
			w.log = l
		}
	}
}

// WithMoveAccelDecelMs configures the acceleration/deceleration used when sending velocity profiles.
func WithMoveAccelDecelMs(acc, dec time.Duration) WheelsOption {
	return func(w *Wheels) {
		accMs := uint32(acc / time.Millisecond)
		decMs := uint32(dec / time.Millisecond)
		if accMs > 0 {
			w.moveAcc = accMs
		}
		if decMs > 0 {
			w.moveDec = decMs
		}
	}
}

// WithRotateAccelDecelMs configures the acceleration/deceleration used when sending velocity profiles.
func WithRotateAccelDecelMs(acc, dec time.Duration) WheelsOption {
	return func(w *Wheels) {
		accMs := uint32(acc / time.Millisecond)
		decMs := uint32(dec / time.Millisecond)
		if accMs > 0 {
			w.rotateAcc = accMs
		}
		if decMs > 0 {
			w.rotateDec = decMs
		}
	}
}

// NewWheels constructs a zltech-backed Wheels using two already-initialized clients.
// Caller owns the lifecycle of the provided clients, but this wrapper also provides Close().
// Defaults: acc/dec = 10ms unless overridden via WithAccelDecelMs.
func NewWheels(c1, c2 *Client, opts ...WheelsOption) (*Wheels, error) {
	w := &Wheels{
		c1:        c1,
		c2:        c2,
		log:       slog.Default(),
		moveAcc:   10,
		moveDec:   10,
		rotateAcc: 10,
		rotateDec: 10,
	}
	for _, opt := range opts {
		opt(w)
	}
	if err := w.c1.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("init client 1: %w", err)
	}
	if err := w.c2.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("init client 2: %w", err)
	}
	return w, nil
}

// NewWheelsAuto is implemented for Linux only in wheels_linux.go.

// Move applies mecanum translation at direction (rad) and wheel speed (rad/s).
func (w *Wheels) Move(dir, speed float64) error {
	speeds := wheels.MacanumTranslate(dir, speed)
	return w.setWheels(speeds)
}

// Rotate spins in place at the given angular speed (as wheel rad/s sign convention).
func (w *Wheels) Rotate(speed float64) error {
	speeds := wheels.MacanumRotate(speed)
	return w.setWheels(speeds)
}

// Stop halts motion. Uses quick-stop (Halt) on both drivers for immediate stop.
func (w *Wheels) Stop() error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return w.c1.Halt(context.Background()) })
	g.Go(func() error { return w.c2.Halt(context.Background()) })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("zltech stop: %w", err)
	}
	return nil
}

// Disable disables the drivers so wheels freewheel.
func (w *Wheels) Disable() error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return w.c1.Disable(context.Background()) })
	g.Go(func() error { return w.c2.Disable(context.Background()) })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("zltech disable: %w", err)
	}
	return nil
}

// Enable enables both drivers.
func (w *Wheels) Enable() error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		ctx := context.Background()
		if err := w.c1.ClearFault(ctx); err != nil {
			return fmt.Errorf("clear fault client1: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
		return w.c1.Enable(ctx)
	})
	g.Go(func() error {
		ctx := context.Background()
		if err := w.c2.ClearFault(ctx); err != nil {
			return fmt.Errorf("clear fault client2: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
		return w.c2.Enable(ctx)
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("zltech enable: %w", err)
	}
	return nil
}

// Status reports per-wheel status derived from both drivers.
func (w *Wheels) Status() (wheels.Status, error) {
	// node 1 first slot is back left (spins backwards)
	// node 1 second slot is front left (spins backwards)
	// node 2 first slot is front right (spins forward)
	// node 2 second slot is back right (spins forward)
	ctx := context.Background()
	// Collect for client 1
	st1, l1, r1, sL1, sR1, cL1, cR1, err := w.readClientStatus(ctx, w.c1)
	if err != nil {
		return wheels.Status{}, fmt.Errorf("zltech status (client1): %w", err)
	}
	// Collect for client 2
	st2, l2, r2, sL2, sR2, cL2, cR2, err := w.readClientStatus(ctx, w.c2)
	if err != nil {
		return wheels.Status{}, fmt.Errorf("zltech status (client2): %w", err)
	}
	// Build wheel statuses (node 1: Left=RearLeft, Right=FrontLeft)
	wsFL := wheels.WheelStatus{Enabled: st1, Speed: sR1, Current: cR1, Error: r1.String()}
	wsRL := wheels.WheelStatus{Enabled: st1, Speed: sL1, Current: cL1, Error: l1.String()}
	wsFR := wheels.WheelStatus{Enabled: st2, Speed: sL2, Current: cL2, Error: l2.String()}
	wsRR := wheels.WheelStatus{Enabled: st2, Speed: sR2, Current: cR2, Error: r2.String()}
	return wheels.Status{
		FrontLeft:  wsFL,
		RearLeft:   wsRL,
		FrontRight: wsFR,
		RearRight:  wsRR,
	}, nil
}

// Close closes both underlying clients.
func (w *Wheels) Close() error {
	var errs []error
	if err := w.c1.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close zltech client 1: %w", err))
	}
	if err := w.c2.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close zltech client 2: %w", err))
	}
	return errors.Join(errs...)
}

// setWheels converts desired wheel speeds (rad/s) to RPM and dispatches to both drivers.
func (w *Wheels) setWheels(s wheels.Set[float64]) error {
	// node 1 first slot is back left (spins backwards)
	// node 1 second slot is front left (spins backwards)
	// node 2 first slot is front right (spins forward)
	// node 2 second slot is back right (spins forward)
	// Convert to RPM for driver commands
	toRPM := func(radPerSec float64) int32 { return int32(radPerSec * radsToRPM) }

	rpmFL := toRPM(s.FrontLeft)
	rpmRL := toRPM(s.RearLeft)
	rpmFR := toRPM(s.FrontRight)
	rpmRR := toRPM(s.RearRight)

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		// node 1: Left slot (first) = RearLeft, Right slot (second) = FrontLeft
		// reverse the direction of the left wheel
		return w.c1.SetProfileVelocity(context.Background(), rpmRL*-1, rpmFL*-1, w.moveAcc, w.moveDec)
	})
	g.Go(func() error {
		// node 2: Left slot (first) = FrontRight, Right slot (second) = RearRight
		return w.c2.SetProfileVelocity(context.Background(), rpmFR, rpmRR, w.moveAcc, w.moveDec)
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("zltech set wheels: %w", err)
	}
	return nil
}

// readClientStatus gathers Enabled, faults, speeds and currents for a driver.
// Returns: enabled, leftFault, rightFault, leftSpeedRad, rightSpeedRad, leftCurrentA, rightCurrentA
func (w *Wheels) readClientStatus(ctx context.Context, c *Client) (bool, FaultCode, FaultCode, float64, float64, float64, float64, error) {
	var (
		st  StatusWord
		lF  FaultCode
		rF  FaultCode
		sL  float64 // rpm
		sR  float64 // rpm
		cL  float64 // A
		cR  float64 // A
		err error
	)

	// Serialize SDO reads to avoid multiplexing races on a single client's SDO channel.
	if st, err = c.StatusWord(ctx); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	if lF, rF, err = c.FaultCodes(ctx); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	if sL, err = c.Speed(ctx, Left); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	if sR, err = c.Speed(ctx, Right); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	if cL, err = c.Current(ctx, Left); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	if cR, err = c.Current(ctx, Right); err != nil {
		return false, 0, 0, 0, 0, 0, 0, err
	}
	// Convert speeds from RPM to rad/s
	toRad := func(rpm float64) float64 { return rpm / radsToRPM }
	enabled := st.State() == StateOperationEnabled || st.State() == StateSwitchedOn
	return enabled, lF, rF, toRad(sL), toRad(sR), cL, cR, nil
}
