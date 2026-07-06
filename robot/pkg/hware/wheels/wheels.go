// Package wheels provides interfaces and implementations for controlling
// wheeled robot movement systems, particularly for mecanum wheel configurations.
package wheels

// Wheels defines the interface for controlling a wheeled robot.
// Implementations handle the specific motor controllers and wheel configurations.
type Wheels interface {
	// Move translates the robot in the specified direction and speed.
	// Direction is in radians (0 is East, π/2 is North, π is West, 3π/2 is South).
	// Speed is given in rad/s.
	Move(dir, speed float64) error

	// Rotate rotates the tensa at the given speed in rad/s.
	// A positive speed rotates the tensa counterclockwise, while a negative speed rotates it clockwise.
	Rotate(speed float64) error

	// Status returns the current state of the wheels.
	Status() (Status, error)

	// Stop halts all wheel movement.
	Stop() error

	// Disable deactivates the motor controllers, allowing wheels to spin freely.
	Disable() error

	// Enable activates the motor controllers, allowing them to control the wheels.
	Enable() error
}

// Position identifies a wheel's physical location on the four-corner
// chassis. It is used throughout the package to map configuration and
// status values to an actual wheel on the robot.
//
// The positions are ordered clockwise starting from the front-left corner
// when the robot is viewed from above.
type Position int

const (
	FrontLeft Position = iota
	FrontRight
	RearRight
	RearLeft
)

// Positions returns the slice of all wheel positions in
// deterministic order: FrontLeft, FrontRight, RearRight, RearLeft.
// This is useful for iterating over every wheel without having to
// remember the ordering convention.
func Positions() []Position {
	return []Position{FrontLeft, FrontRight, RearRight, RearLeft}
}

// String implements fmt.Stringer, returning a human-readable
// representation of the wheel Position.
func (p Position) String() string {
	return []string{"FrontLeft", "FrontRight", "RearRight", "RearLeft"}[p]
}

// Set represents a generic collection of four wheel-specific values.
// It is used for configurations, statuses, or any data that needs to be
// specified or reported on a per-wheel basis (FrontLeft, FrontRight, RearRight, RearLeft).
type Set[T any] struct {
	FrontLeft  T `json:"front_left"`
	FrontRight T `json:"front_right"`
	RearRight  T `json:"rear_right"`
	RearLeft   T `json:"rear_left"`
}

// SetFromMap constructs a Set from a map keyed by Position. Any
// positions that are missing from the map will contain the zero value
// for T in the returned Set.
func SetFromMap[T any](m map[Position]T) Set[T] {
	return Set[T]{
		FrontLeft:  m[FrontLeft],
		FrontRight: m[FrontRight],
		RearRight:  m[RearRight],
		RearLeft:   m[RearLeft],
	}
}

// Get retrieves the element associated with the provided Position.
// If an unknown position is supplied the value for FrontLeft is
// returned as a sensible default.
func (s *Set[T]) Get(pos Position) T {
	switch pos {
	case FrontLeft:
		return s.FrontLeft
	case FrontRight:
		return s.FrontRight
	case RearRight:
		return s.RearRight
	case RearLeft:
		return s.RearLeft
	default:
		return s.FrontLeft
	}
}
