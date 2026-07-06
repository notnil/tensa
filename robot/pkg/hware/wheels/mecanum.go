package wheels

import "math"

// MacanumTranslate calculates the individual wheel speeds for mecanum wheel translation (movement without rotation).
// It takes a direction in radians (where 0 is East, π/2 is North, etc.) and a desired overall speed.
// The speed is the target speed of wheel rotation in radians per second.
// It returns a Set[float64] where each value is the calculated speed for the corresponding wheel.
// Note: This version is a simplified kinematic model. For more advanced control, see MacanumTranslate2.
func MacanumTranslate(direction float64, speed float64) Set[float64] {
	vx := speed * math.Cos(direction)
	vy := speed * math.Sin(direction)
	return Set[float64]{
		FrontLeft:  vy + vx,
		FrontRight: vy - vx,
		RearLeft:   vy - vx,
		RearRight:  vy + vx,
	}
}

// MacanumRotate calculates the individual wheel speeds for in-place mecanum wheel rotation.
// Positive speed values result in counter-clockwise rotation, negative values in clockwise rotation.
// The speed is the target speed of wheel rotation in radians per second.
// It returns a Set[float64] where each value is the calculated speed for the corresponding wheel.
func MacanumRotate(speed float64) Set[float64] {
	return Set[float64]{
		FrontLeft:  -speed,
		FrontRight: speed,
		RearLeft:   -speed,
		RearRight:  speed,
	}
}
