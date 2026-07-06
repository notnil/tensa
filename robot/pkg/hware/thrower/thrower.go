// Package thrower provides interfaces and implementations for controlling a tennis ball throwing.
// The package includes both a hardware implementation (ClearCore) for actual device control
// and a mock implementation for testing purposes.
package thrower

import (
	"context"
	"errors"
	"math"

	"github.com/notnil/tensa/pkg/util/numeric"
	"github.com/notnil/tensa/pkg/util/rotation"
)

const (
	// MinAngle is the minimum allowed throw angle in radians (0 degrees)
	MinAngle = 0

	// maxAngleSafetyMargin is a small safety factor to prevent the angle from reaching its mechanical limit
	// due to floating-point inaccuracies and to provide a buffer from the absolute maximum value.
	maxAngleSafetyMargin = 0.98

	// MaxAngle is the maximum allowed throw angle in radians (π/4 radians or 45 degrees), with a safety margin.
	MaxAngle = (math.Pi / 4) * maxAngleSafetyMargin
)

var (
	// MinUsefulAngle is the minimum angle that is useful for throwing a ball.
	// This is the angle that is used for the thrower's default settings.
	MinUsefulAngle = rotation.FromDegrees(5.0)

	// HalfAngle is halfway between MinAngle and MaxAngle
	HalfAngle = (MaxAngle - MinAngle) / 2

	// StandardAngleRange is the default angle range for the thrower.
	StandardAngleRange = numeric.Range[float64]{Min: MinUsefulAngle, Max: HalfAngle}

	// ErrLoadTimeout is returned when a ball fails to load within the configured timeout period.
	// This typically indicates a mechanical issue with the ball loading mechanism.
	ErrLoadTimeout = errors.New("thrower: load timeout")
)

// Thrower defines the interface for controlling a tennis ball throwing machine.
// Implementations must be safe for concurrent use.
type Thrower interface {
	// Set configures the throw system motors with the provided settings.
	// Returns an error if any setting is invalid or if the hardware fails to respond.
	Set(s Settings) error

	// Throw activates the dispenser to throw a ball.
	// This includes loading a ball if necessary and activating the throwing mechanism.
	// Returns an error if the throw sequence fails at any point.
	// The operation can be canceled via the provided context.
	Throw(ctx context.Context) error

	// Load waits for a ball to be loaded into the throwing position.
	// This may involve activating the ball loading mechanism and waiting for confirmation.
	// Returns ErrLoadTimeout if a ball is not loaded within the configured timeout period.
	// The operation can be canceled via the provided context.
	Load(ctx context.Context) error

	// Spin spins the dispenser motor at the given speed.
	// Returns an error if the hardware fails to respond.
	Spin(speed float64) error

	// Info returns the current state of the thrower.
	Info() (Info, error)
}
