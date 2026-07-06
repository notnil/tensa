// Package rotation provides utilities for handling angle rotations and orientation.
// This package uses robotics conventions for direction, where:
//   - West  = 0
//   - North = π/2
//   - East  = π
//   - South = 3π/2
//
// All angles are in radians.
package rotation

import (
	"math"
	"math/rand/v2"
)

// Cardinal directions in radians using robotics conventions.
const (
	East      = 0.0
	NorthEast = math.Pi / 4
	North     = math.Pi / 2
	NorthWest = 3 * math.Pi / 4
	West      = math.Pi
	SouthWest = 5 * math.Pi / 4
	South     = 3 * math.Pi / 2
	SouthEast = 7 * math.Pi / 4
)

// CardinalDirections returns a slice of all cardinal directions in radians.
func CardinalDirections() []float64 {
	return []float64{
		East,
		NorthEast,
		North,
		NorthWest,
		West,
		SouthWest,
		South,
		SouthEast,
	}
}

type Convention int

const (
	Robotics Convention = iota
	ComputerVision
	Godot
)

func Convert(from Convention, angle float64) float64 {
	switch from {
	case Robotics:
		return Normalize(angle)
	case ComputerVision:
		return Normalize(angle + (math.Pi / 2.0))
	case Godot:
		return Normalize(angle + (math.Pi*3.0)/2.0)
	}
	return angle
}

// Direction represents the direction of rotation.
type Direction int

const (
	// Clockwise rotation direction.
	Clockwise Direction = -1
	// CounterClockwise rotation direction.
	CounterClockwise Direction = 1
)

// Normalize takes an angle (in radians) and returns an equivalent angle in the range [0, 2π).
func Normalize(angle float64) float64 {
	mod := math.Mod(angle, 2*math.Pi)
	if mod < 0 {
		mod += 2 * math.Pi
	}
	return mod
}

// Add returns the sum of two angles (in radians).
func Add(a, b float64) float64 {
	return Normalize(a + b)
}

// Diff returns the minimal angle difference (in radians) and rotation direction
// needed to rotate from 'from' to 'to'. The returned difference is always non-negative.
// The direction is CounterClockwise if a positive rotation is optimal, or Clockwise if the optimal rotation is negative.
func Diff(from, to float64) (Direction, float64) {
	// Normalize both angles.
	a := Normalize(from)
	b := Normalize(to)
	diff := b - a

	// Wrap the difference into the range [-π, π).
	if diff < -math.Pi {
		diff += 2 * math.Pi
	} else if diff >= math.Pi {
		diff -= 2 * math.Pi
	}

	// Determine the optimal rotation direction.
	var dir Direction
	if diff >= 0 {
		dir = CounterClockwise
	} else {
		dir = Clockwise
	}

	return dir, math.Abs(diff)
}

// Avg computes the average (mean) of a slice of angles (in radians).
// It uses vector averaging (i.e., averaging sine and cosine components) to correctly
// handle wrap-around issues. If the input slice is empty, it returns 0.
func Avg(angles []float64) float64 {
	if len(angles) == 0 {
		return 0
	}

	var sumSin, sumCos float64
	for _, angle := range angles {
		sumSin += math.Sin(angle)
		sumCos += math.Cos(angle)
	}

	avg := math.Atan2(sumSin, sumCos)
	return Normalize(avg)
}

// Closest returns the angle in rotations that is closest to target.
func Closest(target float64, rotations []float64) float64 {
	minDiff := math.Inf(1)
	var closest float64
	for _, angle := range rotations {
		_, diff := Diff(target, angle)
		if diff < minDiff {
			minDiff = diff
			closest = angle
		}
	}
	return closest
}

// Random returns a random angle in radians.
func Random() float64 {
	return rand.Float64() * 2 * math.Pi
}

// FromDegrees converts a degree angle to a radian angle.
func FromDegrees(degrees float64) float64 {
	return degrees * math.Pi / 180
}
