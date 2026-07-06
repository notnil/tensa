package roboct

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
	// radsToRPM converts wheel speeds from rad/s to RPM.
	// 1 rad/s = 60 / (2 * pi) RPM ≈ 9.549297
	radsToRPM = 9.549297
)

// Wheels implements wheels.Wheels using 4 RoboCT servo drives.
type Wheels struct {
	fl, fr, rl, rr *Client
	log            *slog.Logger

	// Acceleration and Deceleration in RPS^2 (internally units of 1..1000)
	accel uint16
	decel uint16
}

// WheelsOption customizes Wheels behavior.
type WheelsOption func(*Wheels)

// WithWheelsLogger sets the logger.
func WithWheelsLogger(l *slog.Logger) WheelsOption {
	return func(w *Wheels) {
		w.log = l
	}
}

// WithAccelDecel sets the acceleration and deceleration values (T-curve).
// Units are internal RPS^2 (1..1000). Default is 100.
func WithAccelDecel(acc, dec uint16) WheelsOption {
	return func(w *Wheels) {
		if acc > 0 {
			w.accel = acc
		}
		if dec > 0 {
			w.decel = dec
		}
	}
}

// NewWheels creates a new Wheels instance controlling 4 RoboCT clients.
// The clients must be initialized (or will be initialized by Enable/Move implicitly if implemented,
// but typically Init() is called during startup).
func NewWheels(fl, fr, rl, rr *Client, opts ...WheelsOption) *Wheels {
	w := &Wheels{
		fl:    fl,
		fr:    fr,
		rl:    rl,
		rr:    rr,
		log:   slog.Default(),
		accel: 100, // Default from manual
		decel: 100, // Default from manual
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Init initializes all 4 motor controllers.
func (w *Wheels) Init(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, c := range []*Client{w.fl, w.fr, w.rl, w.rr} {
		client := c
		g.Go(func() error {
			return client.Init(ctx)
		})
	}
	return g.Wait()
}

// Move translates the robot.
func (w *Wheels) Move(dir, speed float64) error {
	speeds := wheels.MacanumTranslate(dir, speed)
	return w.setWheels(speeds)
}

// Rotate rotates the robot.
func (w *Wheels) Rotate(speed float64) error {
	speeds := wheels.MacanumRotate(speed)
	return w.setWheels(speeds)
}

// Stop halts all wheels.
func (w *Wheels) Stop() error {
	g, ctx := errgroup.WithContext(context.Background())
	for _, c := range []*Client{w.fl, w.fr, w.rl, w.rr} {
		client := c
		g.Go(func() error {
			return client.Halt(ctx)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("roboct stop: %w", err)
	}
	return nil
}

// Disable disables all motors.
func (w *Wheels) Disable() error {
	g, ctx := errgroup.WithContext(context.Background())
	for _, c := range []*Client{w.fl, w.fr, w.rl, w.rr} {
		client := c
		g.Go(func() error {
			return client.Disable(ctx)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("roboct disable: %w", err)
	}
	return nil
}

// Enable enables all motors.
func (w *Wheels) Enable() error {
	g, ctx := errgroup.WithContext(context.Background())
	for _, c := range []*Client{w.fl, w.fr, w.rl, w.rr} {
		client := c
		g.Go(func() error {
			return client.Enable(ctx)
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("roboct enable: %w", err)
	}
	return nil
}

// Status returns the collective status of the wheels.
func (w *Wheels) Status() (wheels.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var s wheels.Status
	var errs []error

	// Helper to read status safely
	read := func(pos wheels.Position, c *Client) wheels.WheelStatus {
		st, err := c.Status(ctx)
		ws := wheels.WheelStatus{}
		if err != nil {
			ws.Error = err.Error()
			// Don't append error to global errs unless critical, just report in status
			// But wheels.Status expects us to return error if something is wrong?
			// Usually we want best-effort status.
			w.log.Warn("status read failed", "node", c.nodeID, "err", err)
		} else {
			// Check if enabled (StatusWord bit 1 "Switched On" and bit 2 "Operation Enabled")
			// Actually Operation Enabled is bit 2 (0x0007 = Switched On + Ready).
			// Operation Enabled is state x01x 0111 (binary) -> 0x0027 mask check?
			// StatusWord bits:
			// 0: Ready to switch on
			// 1: Switched on
			// 2: Operation enabled
			// ...
			// Operation Enabled state usually has bits 0, 1, 2 set.
			ws.Enabled = (st.StatusWord & 0x07) == 0x07

			ws.Speed = float64(st.RPM) / radsToRPM
			ws.Current = float64(st.Current) / 1000.0 // mA -> A
			if st.FaultCode != 0 {
				ws.Error = fmt.Sprintf("Fault: 0x%X", st.FaultCode)
			}
		}
		return ws
	}

	s.FrontLeft = read(wheels.FrontLeft, w.fl)
	s.FrontRight = read(wheels.FrontRight, w.fr)
	s.RearLeft = read(wheels.RearLeft, w.rl)
	s.RearRight = read(wheels.RearRight, w.rr)

	if len(errs) > 0 {
		return s, errors.Join(errs...)
	}
	return s, nil
}

func (w *Wheels) setWheels(s wheels.Set[float64]) error {
	g, ctx := errgroup.WithContext(context.Background())

	set := func(c *Client, rads float64) {
		g.Go(func() error {
			rpm := int16(rads * radsToRPM)
			if err := c.SetAcceleration(ctx, w.accel, w.decel); err != nil {
				return err
			}
			return c.SetVelocity(ctx, rpm)
		})
	}

	set(w.fl, s.FrontLeft)
	set(w.fr, s.FrontRight)
	set(w.rl, s.RearLeft)
	set(w.rr, s.RearRight)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("roboct set wheels: %w", err)
	}
	return nil
}
