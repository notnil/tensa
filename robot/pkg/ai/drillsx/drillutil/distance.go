// Package drillutil provides distance prediction for tennis ball bounce distance
// based on wheel speeds and launch angle. It includes both forward prediction
// and inverse optimization to find configurations that achieve target distances.
package drillutil

import (
	"math"

	"github.com/notnil/tensa/pkg/util/numeric"
	"gonum.org/v1/gonum/optimize"
)

// DistanceFunc is a function type that predicts bounce distance based on wheel speeds and launch angle.
//
// Parameters:
//   - topSpeed: top wheel speed in rad/s
//   - bottomSpeed: bottom wheel speed in rad/s
//   - angleRad: launch angle in radians
//
// Returns:
//   - distance in meters
type DistanceFunc func(topSpeed, bottomSpeed, angleRad float64) float64

// Model V1 coefficients from fitted polynomial model (no pure-angle terms; intercept=0)
// d = C1*P + C2*(P*cosA) + C3*(P*sinA) + C4*S + C5*(S*cosA) + C6*(S*sinA) + C7*P^2 + C8*(P*S)
// where: P = top+bottom, S = top-bottom, A in radians
const (
	c1V1 = 0.0076819491
	c2V1 = 0.0001187791
	c3V1 = 0.0782092311
	c4V1 = 1.0476067212
	c5V1 = -0.9667515125
	c6V1 = -0.3659700820
	c7V1 = 0.0000213569
	c8V1 = -0.0001047804
)

// PredictV1 is the V1 distance model that calculates forward bounce distance
// based on the original fitted polynomial coefficients.
var PredictV1 DistanceFunc = func(topSpeed, bottomSpeed, angleRad float64) float64 {
	P := topSpeed + bottomSpeed
	S := topSpeed - bottomSpeed

	cosA := math.Cos(angleRad)
	sinA := math.Sin(angleRad)

	d := (c1V1 * P) +
		(c2V1 * (P * cosA)) +
		(c3V1 * (P * sinA)) +
		(c4V1 * S) +
		(c5V1 * (S * cosA)) +
		(c6V1 * (S * sinA)) +
		(c7V1 * (P * P)) +
		(c8V1 * (P * S))

	return math.Max(0.0, d)
}

// FindConfig specifies the parameters for finding a configuration that achieves
// a target distance.
type FindConfig struct {
	// TargetDistance is the desired bounce distance in meters
	TargetDistance float64

	// SpeedRange is the range for subjective speed (0-10 scale).
	// This will be converted to actual rad/s values internally.
	SpeedRange numeric.Range[SubjectiveSpeed]

	// SpinRange is the range for subjective spin (-5 to 5 scale).
	// This will be converted to actual rad/s values internally.
	SpinRange numeric.Range[SubjectiveSpin]

	// AngleRange is the range for launch angle in radians
	AngleRange numeric.Range[float64]

	// Threshold is the acceptable distance error in meters
	Threshold float64

	// DistanceFunc is the distance model to use for predictions.
	// If nil, PredictV1 will be used as the default.
	DistanceFunc DistanceFunc
}

// FindResult contains the result of a configuration search.
type FindResult struct {
	// TopSpeed is the found top wheel speed in rad/s
	TopSpeed float64

	// BottomSpeed is the found bottom wheel speed in rad/s
	BottomSpeed float64

	// Angle is the found launch angle in radians
	Angle float64

	// Distance is the actual predicted distance in meters
	Distance float64

	// Found indicates whether a valid solution was found within the threshold
	Found bool
}

// subjectiveToRadS converts subjective speed and spin values to actual rad/s values.
// Returns (P, S) where P = top + bottom and S = top - bottom in rad/s.
func subjectiveToRadS(speed, spin float64) (float64, float64) {
	// Use Subjective to get top and bottom speeds, then calculate P and S
	settings := Subjective(SubjectiveSpeed(speed), SubjectiveSpin(spin), 0)
	P := settings.Top + settings.Bottom
	S := settings.Top - settings.Bottom
	return P, S
}

// Find searches for wheel speeds and launch angle that achieve a target distance
// within the specified ranges and threshold. It uses Gonum's optimization library
// with the Nelder-Mead algorithm for efficient convergence.
//
// The function searches in the space of P (speed = top+bottom) and S (spin = top-bottom)
// and converts back to individual wheel speeds. Only configurations where both
// top and bottom speeds are non-negative are considered valid.
//
// SpeedRange and SpinRange should be provided in subjective units (0-10 for speed,
// -5 to 5 for spin), and will be converted to rad/s internally.
func Find(config FindConfig) FindResult {
	// Use the provided distance function, or default to PredictV1
	distFunc := config.DistanceFunc
	if distFunc == nil {
		distFunc = PredictV1
	}

	// Convert subjective speed range to P range (rad/s)
	minP, _ := subjectiveToRadS(float64(config.SpeedRange.Min), 0)
	maxP, _ := subjectiveToRadS(float64(config.SpeedRange.Max), 0)
	speedRange := numeric.Range[float64]{Min: minP, Max: maxP}

	// Convert subjective spin range to S range (rad/s)
	_, minS := subjectiveToRadS(0, float64(config.SpinRange.Min))
	_, maxS := subjectiveToRadS(0, float64(config.SpinRange.Max))
	spinRange := numeric.Range[float64]{Min: minS, Max: maxS}

	// Define the objective function: minimize absolute error from target distance
	objective := func(x []float64) float64 {
		P := x[0] // speed = top + bottom
		S := x[1] // spin = top - bottom
		A := x[2] // angle

		// Convert P and S to top and bottom speeds
		top := (P + S) / 2.0
		bottom := (P - S) / 2.0

		// Apply penalty for physically infeasible configurations
		if top < 0 || bottom < 0 {
			return 1e10 // Large penalty
		}

		// Predict distance and calculate error
		distance := distFunc(top, bottom, A)
		error := math.Abs(distance - config.TargetDistance)

		return error
	}

	// Try multiple starting points for better convergence (sequentially with early exit)
	startingPoints := [][]float64{
		{speedRange.Middle(), spinRange.Middle(), config.AngleRange.Middle()}, // Center
		{speedRange.Min + (speedRange.Max-speedRange.Min)*0.3,
			spinRange.Min + (spinRange.Max-spinRange.Min)*0.3,
			config.AngleRange.Min + (config.AngleRange.Max-config.AngleRange.Min)*0.3}, // Lower third
		{speedRange.Min + (speedRange.Max-speedRange.Min)*0.7,
			spinRange.Min + (spinRange.Max-spinRange.Min)*0.7,
			config.AngleRange.Min + (config.AngleRange.Max-config.AngleRange.Min)*0.7}, // Upper third
	}

	bestResult := FindResult{Found: false}
	bestError := math.MaxFloat64

	for _, initial := range startingPoints {
		// Define the optimization problem
		problem := optimize.Problem{
			Func: objective,
		}

		// Use Nelder-Mead method (derivative-free, good for non-linear problems)
		method := &optimize.NelderMead{}

		// Configure optimization settings
		settings := &optimize.Settings{
			FuncEvaluations: 1000,
			MajorIterations: 500,
		}

		// Run the optimizer
		result, err := optimize.Minimize(problem, initial, settings, method)
		if err != nil {
			continue // Try next starting point
		}

		// Extract the optimal parameters
		P := result.X[0]
		S := result.X[1]
		A := result.X[2]

		// Clamp to valid ranges
		P = speedRange.Clamp(P)
		S = spinRange.Clamp(S)
		A = config.AngleRange.Clamp(A)

		// Convert back to top and bottom speeds
		top := (P + S) / 2.0
		bottom := (P - S) / 2.0

		// Validate physical feasibility
		if top < 0 || bottom < 0 {
			continue
		}

		// Calculate actual distance
		distance := distFunc(top, bottom, A)
		error := math.Abs(distance - config.TargetDistance)

		// Update best result if this is better
		if error < bestError {
			bestError = error
			bestResult = FindResult{
				TopSpeed:    top,
				BottomSpeed: bottom,
				Angle:       A,
				Distance:    distance,
				Found:       error <= config.Threshold,
			}
		}

		// Early exit if we found a solution within threshold
		if error <= config.Threshold {
			return bestResult
		}
	}

	return bestResult
}
