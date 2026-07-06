package drillutil

import (
	"fmt"
	"math"
	"testing"

	"github.com/notnil/tensa/pkg/util/numeric"
	"github.com/notnil/tensa/pkg/util/rotation"
)

func TestSubjectives(t *testing.T) {
	angle := rotation.FromDegrees(15.0)
	settings := Subjective(Speed7, SpinNegThree, angle)
	distance := PredictV1(settings.Top, settings.Bottom, angle)
	fmt.Printf("Distance: %.2f\n", distance)

	config := FindConfig{
		TargetDistance: 20.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: Speed5, Max: Speed8},
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: SpinOne, Max: SpinThree},
		AngleRange:     numeric.Range[float64]{Min: rotation.FromDegrees(1), Max: rotation.FromDegrees(30)},
		Threshold:      0.1,
		DistanceFunc:   PredictV1,
	}

	result := Find(config)
	fmt.Printf("Result: %+v\n", result)

	if !result.Found {
		t.Errorf("Find() should have found a solution for 20m target with Speed5-8 and Spin1-3")
	}

	// Verify the result is within threshold
	error := math.Abs(result.Distance - config.TargetDistance)
	if error > config.Threshold {
		t.Errorf("Find() error = %v, exceeds threshold %v", error, config.Threshold)
	}

	fmt.Printf("Found solution: TopSpeed=%.2f, BottomSpeed=%.2f, Angle=%.2f deg, Distance=%.2f m\n",
		result.TopSpeed, result.BottomSpeed, result.Angle*180.0/math.Pi, result.Distance)

	speed, spin := InverseSubjective(result.TopSpeed, result.BottomSpeed)
	fmt.Printf("Speed: %v, Spin: %v\n", speed, spin)
}

func TestPredict(t *testing.T) {
	tests := []struct {
		name        string
		topSpeed    float64
		bottomSpeed float64
		angleRad    float64
		wantMin     float64 // minimum expected distance
		wantMax     float64 // maximum expected distance
	}{
		{
			name:        "Example from Python code",
			topSpeed:    180.0,
			bottomSpeed: 140.0,
			angleRad:    math.Pi / 9.0, // 20 degrees
			wantMin:     12.0,          // reasonable bounds based on model output
			wantMax:     13.0,
		},
		{
			name:        "Zero speeds",
			topSpeed:    0.0,
			bottomSpeed: 0.0,
			angleRad:    math.Pi / 4.0,
			wantMin:     0.0,
			wantMax:     0.0,
		},
		{
			name:        "High speeds with backspin",
			topSpeed:    100.0,
			bottomSpeed: 200.0,
			angleRad:    math.Pi / 6.0,
			wantMin:     0.0,
			wantMax:     50.0,
		},
		{
			name:        "High speeds with topspin",
			topSpeed:    200.0,
			bottomSpeed: 100.0,
			angleRad:    math.Pi / 6.0,
			wantMin:     0.0,
			wantMax:     50.0,
		},
		{
			name:        "Low angle",
			topSpeed:    150.0,
			bottomSpeed: 150.0,
			angleRad:    math.Pi / 18.0, // 10 degrees
			wantMin:     0.0,
			wantMax:     20.0,
		},
		{
			name:        "High angle",
			topSpeed:    150.0,
			bottomSpeed: 150.0,
			angleRad:    math.Pi / 3.0, // 60 degrees
			wantMin:     0.0,
			wantMax:     30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PredictV1(tt.topSpeed, tt.bottomSpeed, tt.angleRad)

			if got < 0 {
				t.Errorf("Predict() returned negative distance: %v", got)
			}

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Predict() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestPredictNegativeClamping(t *testing.T) {
	// Test configurations that might produce negative distances are clamped to 0
	tests := []struct {
		name        string
		topSpeed    float64
		bottomSpeed float64
		angleRad    float64
	}{
		{"Very low speeds", 1.0, 1.0, 0.1},
		{"Extreme backspin", 10.0, 200.0, 0.1},
		{"Negative angle", 100.0, 100.0, -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PredictV1(tt.topSpeed, tt.bottomSpeed, tt.angleRad)
			if got < 0 {
				t.Errorf("Predict() = %v, should be clamped to >= 0", got)
			}
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name           string
		config         FindConfig
		wantFound      bool
		checkTopBottom bool // whether to validate top/bottom are non-negative
	}{
		{
			name: "Find configuration for 10m distance",
			config: FindConfig{
				TargetDistance: 10.0,
				SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
				SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
				AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.7},
				Threshold:      0.5,
			},
			wantFound:      true,
			checkTopBottom: true,
		},
		{
			name: "Find configuration for medium distance",
			config: FindConfig{
				TargetDistance: 8.0,
				SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 3.0}, // Subjective speed 0-3
				SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
				AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.5},
				Threshold:      0.5,
			},
			wantFound:      true,
			checkTopBottom: true,
		},
		{
			name: "Impossible target with tight constraints",
			config: FindConfig{
				TargetDistance: 50.0,                                               // Very long distance
				SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 1.0}, // Subjective speed 0-1 (too low)
				SpinRange:      numeric.Range[SubjectiveSpin]{Min: 0.0, Max: 0.5},  // Subjective spin 0-0.5
				AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.2},
				Threshold:      0.1, // Tight threshold
			},
			wantFound:      false,
			checkTopBottom: true,
		},
		{
			name: "Find with wide ranges",
			config: FindConfig{
				TargetDistance: 8.0,
				SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 6.0}, // Subjective speed 0-6
				SpinRange:      numeric.Range[SubjectiveSpin]{Min: -5.0, Max: 5.0}, // Subjective spin -5 to 5 (full range)
				AngleRange:     numeric.Range[float64]{Min: 0.0, Max: 1.0},
				Threshold:      1.0,
			},
			wantFound:      true,
			checkTopBottom: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Find(tt.config)

			if result.Found != tt.wantFound {
				t.Errorf("Find() Found = %v, want %v", result.Found, tt.wantFound)
			}

			if tt.checkTopBottom {
				if result.TopSpeed < 0 {
					t.Errorf("Find() TopSpeed = %v, should be non-negative", result.TopSpeed)
				}
				if result.BottomSpeed < 0 {
					t.Errorf("Find() BottomSpeed = %v, should be non-negative", result.BottomSpeed)
				}
			}

			// Verify the result is consistent with Predict
			if result.Found || result.TopSpeed > 0 || result.BottomSpeed > 0 {
				predictedDistance := PredictV1(result.TopSpeed, result.BottomSpeed, result.Angle)
				tolerance := 0.001 // Allow small floating point errors
				if math.Abs(predictedDistance-result.Distance) > tolerance {
					t.Errorf("Find() Distance = %v, but Predict() = %v, inconsistent",
						result.Distance, predictedDistance)
				}
			}

			// If found, verify it's within threshold
			if result.Found {
				error := math.Abs(result.Distance - tt.config.TargetDistance)
				if error > tt.config.Threshold {
					t.Errorf("Find() error = %v, exceeds threshold %v",
						error, tt.config.Threshold)
				}
			}

			// Verify angle is within range if a result was returned
			if result.TopSpeed > 0 || result.BottomSpeed > 0 {
				if result.Angle < tt.config.AngleRange.Min || result.Angle > tt.config.AngleRange.Max {
					t.Errorf("Find() Angle = %v, outside range [%v, %v]",
						result.Angle, tt.config.AngleRange.Min, tt.config.AngleRange.Max)
				}
			}
		})
	}
}

func TestFindPhysicallyFeasible(t *testing.T) {
	// Test that Find only returns physically feasible configurations
	// (both top and bottom speeds must be non-negative)
	config := FindConfig{
		TargetDistance: 7.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 2.0}, // Subjective speed 0-2
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -5.0, Max: 5.0}, // Subjective spin -5 to 5
		AngleRange:     numeric.Range[float64]{Min: 0.2, Max: 0.6},
		Threshold:      0.5,
	}

	result := Find(config)

	if result.TopSpeed < 0 {
		t.Errorf("Find() returned negative TopSpeed: %v", result.TopSpeed)
	}
	if result.BottomSpeed < 0 {
		t.Errorf("Find() returned negative BottomSpeed: %v", result.BottomSpeed)
	}

	// Verify P and S relationship
	P := result.TopSpeed + result.BottomSpeed
	S := result.TopSpeed - result.BottomSpeed

	// Convert subjective ranges to rad/s for validation
	minP := 200.0 + 50.0*float64(config.SpeedRange.Min)
	maxP := 200.0 + 50.0*float64(config.SpeedRange.Max)
	minS := 25.7 * float64(config.SpinRange.Min)
	maxS := 25.7 * float64(config.SpinRange.Max)

	if P < minP || P > maxP {
		t.Errorf("Find() P = %v, outside converted SpeedRange [%v, %v]", P, minP, maxP)
	}

	if S < minS || S > maxS {
		t.Errorf("Find() S = %v, outside converted SpinRange [%v, %v]", S, minS, maxS)
	}
}

func TestPredictConsistency(t *testing.T) {
	// Test that Predict gives consistent results
	topSpeed := 180.0
	bottomSpeed := 140.0
	angle := math.Pi / 9.0

	d1 := PredictV1(topSpeed, bottomSpeed, angle)
	d2 := PredictV1(topSpeed, bottomSpeed, angle)

	if d1 != d2 {
		t.Errorf("Predict() not consistent: first call = %v, second call = %v", d1, d2)
	}
}

func TestFindWithCustomDistanceFunc(t *testing.T) {
	// Create a custom distance function that always returns a fixed distance
	customFunc := DistanceFunc(func(topSpeed, bottomSpeed, angleRad float64) float64 {
		return 5.0 // Always return 5 meters
	})

	config := FindConfig{
		TargetDistance: 5.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -2.0, Max: 2.0}, // Subjective spin -2 to 2
		AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.5},
		Threshold:      0.1,
		DistanceFunc:   customFunc,
	}

	result := Find(config)

	if !result.Found {
		t.Errorf("Find() should have found a solution with custom function")
	}

	if result.Distance != 5.0 {
		t.Errorf("Find() Distance = %v, want 5.0 (from custom function)", result.Distance)
	}
}

func TestFindWithPredictV1Explicit(t *testing.T) {
	// Test explicitly passing PredictV1 vs relying on nil default
	configWithV1 := FindConfig{
		TargetDistance: 10.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
		AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.7},
		Threshold:      0.5,
		DistanceFunc:   PredictV1,
	}

	configWithNil := FindConfig{
		TargetDistance: 10.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
		AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.7},
		Threshold:      0.5,
		DistanceFunc:   nil,
	}

	resultV1 := Find(configWithV1)
	resultNil := Find(configWithNil)

	// Both should find the same result since nil defaults to PredictV1
	if resultV1.Found != resultNil.Found {
		t.Errorf("Explicit PredictV1 and nil should produce same Found result")
	}

	if resultV1.Distance != resultNil.Distance {
		t.Errorf("Explicit PredictV1 and nil should produce same Distance")
	}
}

func BenchmarkPredict(b *testing.B) {
	topSpeed := 180.0
	bottomSpeed := 140.0
	angle := math.Pi / 9.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PredictV1(topSpeed, bottomSpeed, angle)
	}
}

func BenchmarkPredictV1(b *testing.B) {
	topSpeed := 180.0
	bottomSpeed := 140.0
	angle := math.Pi / 9.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PredictV1(topSpeed, bottomSpeed, angle)
	}
}

func TestFindAccuracy(t *testing.T) {
	// Test that the optimizer finds highly accurate solutions
	config := FindConfig{
		TargetDistance: 10.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
		AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.7},
		Threshold:      0.1, // Very tight threshold
	}

	result := Find(config)

	if !result.Found {
		t.Errorf("Find() should find a solution with Gonum optimizer")
	}

	error := math.Abs(result.Distance - config.TargetDistance)
	if error > config.Threshold {
		t.Errorf("Find() error = %v, exceeds tight threshold %v", error, config.Threshold)
	}

	// Verify the result is highly accurate
	if error > 0.05 { // Expect better than 5cm accuracy
		t.Errorf("Find() error = %v meters, expected < 0.05 meters with optimizer", error)
	}
}

func BenchmarkFind(b *testing.B) {
	config := FindConfig{
		TargetDistance: 10.0,
		SpeedRange:     numeric.Range[SubjectiveSpeed]{Min: 0.0, Max: 4.0}, // Subjective speed 0-4
		SpinRange:      numeric.Range[SubjectiveSpin]{Min: -3.0, Max: 3.0}, // Subjective spin -3 to 3
		AngleRange:     numeric.Range[float64]{Min: 0.1, Max: 0.7},
		Threshold:      0.1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Find(config)
	}
}
