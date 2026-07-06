// rotation_test.go
package rotation_test

import (
	"math"
	"testing"

	"github.com/notnil/tensa/pkg/util/rotation"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
	}{
		{-math.Pi / 2, 3 * math.Pi / 2},
		{0, 0},
		{2 * math.Pi, 0},
		{3 * math.Pi, math.Pi},
		{-3 * math.Pi, math.Pi},
		{5 * math.Pi / 2, math.Pi / 2},
	}

	for _, tc := range testCases {
		got := rotation.Normalize(tc.input)
		if math.Abs(got-tc.expected) > 1e-9 {
			t.Errorf("Normalize(%v) = %v; expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestFromComputerVisionConvention(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
	}{
		// Adding π/2 should rotate computer vision 0 (typically rightward) to robotics North.
		{0, rotation.Normalize(0 + math.Pi/2)},                         // Expected: North (π/2)
		{math.Pi / 2, rotation.Normalize(math.Pi/2 + math.Pi/2)},       // Expected: West (π)
		{math.Pi, rotation.Normalize(math.Pi + math.Pi/2)},             // Expected: South (3π/2)
		{3 * math.Pi / 2, rotation.Normalize(3*math.Pi/2 + math.Pi/2)}, // Expected: East (0)
	}

	for _, tc := range testCases {
		got := rotation.Convert(rotation.ComputerVision, tc.input)
		if math.Abs(got-tc.expected) > 1e-9 {
			t.Errorf("FromComputerVisionConvention(%v) = %v; expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestDiff(t *testing.T) {
	// Case 1: from 0 to π/2 should yield a counterclockwise rotation of π/2.
	dir, diff := rotation.Diff(0, math.Pi/2)
	if dir != rotation.CounterClockwise || math.Abs(diff-math.Pi/2) > 1e-9 {
		t.Errorf("Diff(0, π/2) = (dir=%v, diff=%v); expected (dir=%v, diff=%v)",
			dir, diff, rotation.CounterClockwise, math.Pi/2)
	}

	// Case 2: from 0 to 7π/4 should yield a clockwise rotation of π/4.
	dir, diff = rotation.Diff(0, 7*math.Pi/4)
	if dir != rotation.Clockwise || math.Abs(diff-math.Pi/4) > 1e-9 {
		t.Errorf("Diff(0, 7π/4) = (dir=%v, diff=%v); expected (dir=%v, diff=%v)",
			dir, diff, rotation.Clockwise, math.Pi/4)
	}

	// Case 3: from 7π/4 to π/4 should yield a counterclockwise rotation of π/2.
	dir, diff = rotation.Diff(7*math.Pi/4, math.Pi/4)
	if dir != rotation.CounterClockwise || math.Abs(diff-math.Pi/2) > 1e-9 {
		t.Errorf("Diff(7π/4, π/4) = (dir=%v, diff=%v); expected (dir=%v, diff=%v)",
			dir, diff, rotation.CounterClockwise, math.Pi/2)
	}
}

func TestAverage(t *testing.T) {
	// Case 1: average of [0, π/2] should be π/4.
	angles := []float64{0, math.Pi / 2}
	avg := rotation.Avg(angles)
	expected := math.Pi / 4
	if _, diff := rotation.Diff(avg, expected); diff > 1e-9 {
		t.Errorf("Average(%v) = %v; expected %v", angles, avg, expected)
	}

	// Case 2: average of [pi/4, 7pi/4] should be 0.
	angles = []float64{math.Pi / 4, 7 * math.Pi / 4}
	avg = rotation.Avg(angles)
	expected = 0
	if _, diff := rotation.Diff(avg, expected); diff > 1e-9 {
		t.Errorf("Average(%v) = %v; expected %v", angles, avg, expected)
	}

	// Case 2: average of an empty slice should return 0.
	avg = rotation.Avg([]float64{})
	if avg != 0 {
		t.Errorf("Average([]) = %v; expected 0", avg)
	}
}

func TestClosest(t *testing.T) {
	// Case 1: For target π/2 and rotations [0, π/2, π] the closest should be π/2.
	rotations := []float64{0, math.Pi / 2, math.Pi}
	target := math.Pi / 2
	got := rotation.Closest(target, rotations)
	if _, diff := rotation.Diff(got, math.Pi/2); diff > 1e-9 {
		t.Errorf("Closest(%v, %v) = %v; expected %v", target, rotations, got, math.Pi/2)
	}

	// Case 2: For target π/4 and rotations [0, π/2, π],
	// both 0 and π/2 are equidistant (π/4 apart); the function should return the first encountered (0).
	target = math.Pi / 4
	got = rotation.Closest(target, rotations)
	if _, diff := rotation.Diff(got, 0); diff > 1e-9 {
		t.Errorf("Closest(%v, %v) = %v; expected %v", target, rotations, got, 0.0)
	}
}
