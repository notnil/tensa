package clearcore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/notnil/tensa/pkg/util/jsonx"
)

const (
	// RadsToRPM conversion factor
	RadsToRPM = 9.5493
)

var (
	// ErrLoadTimeout is returned when a ball fails to load within the configured timeout period.
	// This typically indicates a mechanical issue with the ball loading mechanism.
	ErrLoadTimeout = errors.New("thrower: load timeout")
)

// Config contains configuration parameters for the ClearCore microcontroller implementation.
// All durations are specified using jsonx.Duration for JSON serialization compatibility.
type Config struct {
	// DispenserSpeed is the speed of the dispenser motor in radians per second
	DispenserSpeed float64 `json:"dispenser_speed"`

	// ThrowDuration is the amount of time the dispenser motor runs during a throw
	ThrowDuration jsonx.Duration `json:"throw_duration"`

	// LoadTimeoutDuration is the maximum time to wait for a ball to be loaded
	LoadTimeoutDuration jsonx.Duration `json:"load_timeout_duration"`

	// LoadPollInterval is how frequently to check if a ball is loaded
	LoadPollInterval jsonx.Duration `json:"load_poll_interval"`

	// MaxThrowSpeed is the maximum speed of the throw motor in radians per second
	MaxThrowSpeed float64 `json:"max_throw_speed"`

	// LoadBeforeThrow determines whether to verify a ball is loaded before throwing
	LoadBeforeThrow bool `json:"load_before_throw"`

	// PreventJams determines whether to prevent spinning the dispenser if the throw system
	// isn't non-zero. This is to prevent jams in the thrower.
	PreventJams bool `json:"prevent_jams"`
}

// ClearCore implements the Thrower interface using the ClearCore microcontroller.
// It includes thread-safe state management for loading and throwing operations.
type ClearCore struct {
	client *Client // Client for communicating with the ClearCore hardware
	cfg    Config  // Configuration parameters

	mu             sync.RWMutex
	set            thrower.Settings
	dispenserSpeed float64
}

// New creates a new thrower that uses the ClearCore microcontroller.
// Returns a pointer to a ClearCore instance initialized with the provided client and configuration.
func New(client *Client, cfg Config) *ClearCore {
	return &ClearCore{
		client: client,
		cfg:    cfg,
	}
}

// MinAngle is the minimum allowed throw angle in radians (0 degrees)
const MinAngle = 0

// maxAngleSafetyMargin is a small safety factor to prevent the angle from reaching its mechanical limit
// due to floating-point inaccuracies and to provide a buffer from the absolute maximum value.
const maxAngleSafetyMargin = 0.98

// MaxAngle is the maximum allowed throw angle in radians (π/4 radians or 45 degrees), with a safety margin.
const MaxAngle = (math.Pi / 4) * maxAngleSafetyMargin

// Set configures the throw system motors with the provided settings.
// Returns an error if any setting is invalid (out of range) or if the hardware fails to respond.
// The method validates that:
// - Angle is between MinAngle and MaxAngle
// - Top and Bottom speeds are positive and don't exceed MaxThrowSpeed
func (t *ClearCore) Set(s thrower.Settings) error {
	if s.Angle < MinAngle || s.Angle > MaxAngle {
		return fmt.Errorf("ClearCore: invalid angle: %f", s.Angle)
	}
	if s.Top > t.cfg.MaxThrowSpeed {
		return fmt.Errorf("ClearCore: top speed too high: %f", s.Top)
	}
	if s.Bottom > t.cfg.MaxThrowSpeed {
		return fmt.Errorf("ClearCore: bottom speed too high: %f", s.Bottom)
	}
	if s.Top < 0 {
		return fmt.Errorf("ClearCore: top speed too low: %f", s.Top)
	}
	if s.Bottom < 0 {
		return fmt.Errorf("ClearCore: bottom speed too low: %f", s.Bottom)
	}
	top := toRPM(s.Top)
	bottom := toRPM(s.Bottom)
	if err := t.client.SetThrow(top, bottom, s.Angle); err != nil {
		return fmt.Errorf("ClearCore: failed to set throw: %w", err)
	}
	t.mu.Lock()
	t.set = s
	t.mu.Unlock()
	return nil
}

// Throw executes the ball throwing sequence. It first ensures a ball is loaded by calling Load(),
// then activates the dispenser motor at the configured speed (DispenserSpeed) for the specified
// duration (ThrowDuration), and finally stops the motor by setting speed to 0. If any step fails,
// it returns an error with context about the failure.
// The method is thread-safe and will return immediately if a throw operation is already in progress.
// If LoadBeforeThrow is true, the method will preemptively load a ball before throwing.
func (t *ClearCore) Throw(ctx context.Context) error {
	if t.cfg.LoadBeforeThrow {
		// preemptively load a ball
		if err := t.Load(ctx); err != nil {
			return fmt.Errorf("ClearCore: failed to throw by preemptively loading a ball: %w", err)
		}
	}
	// set the dispenser motor to the configured speed
	speed := toRPM(t.cfg.DispenserSpeed)
	if err := t.client.SetDispenser(speed); err != nil {
		return fmt.Errorf("ClearCore: failed to throw by setting the dispenser motor to the configured speed: %w", err)
	}
	// run the dispenser motor for the configured duration with context cancellation support
	select {
	case <-time.After(time.Duration(t.cfg.ThrowDuration)):
		// Normal completion
	case <-ctx.Done():
		// Context canceled, stop the motor and return
		t.client.SetDispenser(0)
		return ctx.Err()
	}
	// stop the dispenser motor
	return t.client.SetDispenser(0)
}

// Load waits for a ball to be loaded into the throwing position.
// It polls the ClearCore microcontroller at regular intervals to check if a ball is loaded.
// Returns ErrLoadTimeout if a ball is not loaded within the configured timeout period.
// If Load is already being called by another goroutine, this method will return immediately
// without performing any action.
// The method is thread-safe and includes proper state management to prevent concurrent loading.
func (t *ClearCore) Load(ctx context.Context) error {
	// check if a ball is loaded before moving dispenser motor
	loaded, err := t.client.GetLoaded()
	if err != nil {
		return fmt.Errorf("ClearCore: failed to load: %w", err)
	}
	if loaded {
		return nil
	}
	// set the dispenser motor to the configured speed
	if err := t.setDispenserSpeed(t.cfg.DispenserSpeed); err != nil {
		return fmt.Errorf("ClearCore: failed to throw by setting the dispenser motor to the configured speed: %w", err)
	}
	defer t.setDispenserSpeed(0)

	timeout := time.After(time.Duration(t.cfg.LoadTimeoutDuration))
	ticker := time.NewTicker(time.Duration(t.cfg.LoadPollInterval))
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return ErrLoadTimeout
		case <-ticker.C:
			loaded, err := t.client.GetLoaded()
			if err != nil {
				return fmt.Errorf("ClearCore: failed to load: %w", err)
			}
			if loaded {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Spin spins the dispenser motor at the given speed.
// Returns an error if the hardware fails to respond.
func (t *ClearCore) Spin(speed float64) error {
	return t.client.SetDispenser(toRPM(speed))
}

// Info returns the current state of the thrower.
func (t *ClearCore) Info() (thrower.Info, error) {
	loaded, err := t.client.GetLoaded()
	if err != nil {
		return thrower.Info{}, fmt.Errorf("ClearCore: failed to get info: %w", err)
	}
	t.mu.RLock()
	dispenserSpeed := t.dispenserSpeed
	set := t.set
	t.mu.RUnlock()
	return thrower.Info{
		Loaded:         loaded,
		DispenserSpeed: dispenserSpeed,
		ThrowSettings:  set,
	}, nil
}

// setDispenserSpeed sets the dispenser motor speed and updates the internal state.
func (t *ClearCore) setDispenserSpeed(speed float64) error {
	if t.cfg.PreventJams && speed != 0 {
		info, err := t.Info()
		if err != nil {
			return fmt.Errorf("ClearCore: failed to set dispenser speed due to info error: %w", err)
		}
		// if the throw system is not moving, don't set the dispenser speed
		// this is to prevent jams in the thrower
		if !info.ThrowSettings.IsMoving() {
			return nil
		}
	}
	if err := t.client.SetDispenser(toRPM(speed)); err != nil {
		return fmt.Errorf("ClearCore: failed to set dispenser speed: %w", err)
	}
	t.mu.Lock()
	t.dispenserSpeed = speed
	t.mu.Unlock()
	return nil
}

func toRPM(speed float64) int {
	return int(math.Round(speed * RadsToRPM))
}
