package drillutil

import (
	"errors"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/util/numeric"
)

var ErrTargetNotFound = errors.New("target not found")

type Target struct {
	CurrentLocation api.Point
	TargetLocation  api.Point
	SpeedRange      numeric.Range[SubjectiveSpeed]
	SpinRange       numeric.Range[SubjectiveSpin]
	AngleRange      numeric.Range[float64]
	// Threshold is the acceptable distance error in meters.
	// If zero, defaults to 0.1 meters.
	Threshold float64
}

func (t *Target) Find() (api.Settings, error) {
	dist := t.CurrentLocation.Distance(t.TargetLocation)
	threshold := t.Threshold
	if threshold == 0 {
		threshold = 0.1 // Default threshold
	}
	findResult := Find(FindConfig{
		TargetDistance: dist,
		SpeedRange:     t.SpeedRange,
		SpinRange:      t.SpinRange,
		AngleRange:     t.AngleRange,
		Threshold:      threshold,
		DistanceFunc:   PredictV1,
	})
	if !findResult.Found {
		return api.Settings{}, ErrTargetNotFound
	}
	return api.Settings{
		Top:    findResult.TopSpeed,
		Bottom: findResult.BottomSpeed,
		Angle:  findResult.Angle,
	}, nil
}
