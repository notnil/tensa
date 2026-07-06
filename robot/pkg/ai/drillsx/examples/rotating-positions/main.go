//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

package main

import (
	"bytes"
	"context"
	_ "embed"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
	"github.com/notnil/tensa/pkg/util/numeric"
	"github.com/notnil/tensa/pkg/util/rotation"
)

const (
	defaultThrowDuration = 3 * time.Second
	feedWaitDuration     = 3 * time.Second
)

//tts:filename=intro.mp3 This drill will feed balls from three rotating positions on the court to a random target.

//go:embed intro.mp3
var introBytes []byte

type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting drill")

	// Initialize shot tracker
	tracker := drillutil.NewShotTracker(
		rt.Events,
		rt.Metrics,
		rt.Log,
	)
	defer tracker.Stop()

	buf := bytes.NewBuffer(introBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play instructions audio", "error", err)
	}

	pts := []api.Point{
		{X: -2, Y: -10},
		{X: 0, Y: -10},
		{X: -2, Y: -10},
	}

	roundNum := 0
	for {
		roundNum++
		rt.Log.Info("Starting new round", "round", roundNum)

		for posIdx, pt := range pts {
			rt.Log.Info("Moving to position", "round", roundNum, "position", posIdx+1, "x", pt.X, "y", pt.Y)

			for i := 0; i < 3; i++ {
				shotNum := i + 1
				rt.Log.Info("Preparing shot from position", "round", roundNum, "position", posIdx+1, "shot", shotNum)

				targetPt, err := drillutil.SampleBox(rt.Rnd, api.Point{X: -3, Y: 7}, api.Point{X: 3, Y: 11})
				if err != nil {
					rt.Log.Error("Failed to sample target box", "error", err)
					return err
				}
				rt.Log.Info("Target sampled", "target_x", targetPt.X, "target_y", targetPt.Y)
				target := drillutil.Target{
					CurrentLocation: pt,
					TargetLocation:  targetPt,
					SpeedRange:      numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed5, Max: drillutil.Speed7},
					SpinRange:       numeric.Range[drillutil.SubjectiveSpin]{Min: drillutil.SpinZero, Max: drillutil.SpinThree},
					AngleRange:      numeric.Range[float64]{Min: rotation.FromDegrees(10), Max: rotation.FromDegrees(30)},
				}
				settings, err := target.Find()
				if err != nil {
					rt.Log.Error("Failed to find target settings", "error", err)
					return err
				}
				rt.Log.Info("Target settings calculated", "top_speed", settings.Top, "bottom_speed", settings.Bottom, "angle", settings.Angle)

				loc := api.Location{
					Point:    pt,
					Rotation: pt.Angle(targetPt),
				}
				if err := drillutil.PrepareThrow(ctx, rt, drillutil.PrepareParams{
					RobotLocation:          loc,
					MinimumPreparationTime: defaultThrowDuration,
					ThrowerSettings:        settings,
				}, rt.Log); err != nil {
					rt.Log.Error("Failed to prepare throw", "error", err)
					return err
				}
				tracker.Feed(nil, nil)

				// Execute throw
				if err := rt.Thrower.Throw(ctx); err != nil {
					rt.Log.Error("Failed to throw ball", "error", err)
					return err
				}
				rt.Log.Info("Shot completed successfully", "round", roundNum, "position", posIdx+1, "shot", shotNum)

				// Wait between feeds
				select {
				case <-time.After(feedWaitDuration):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
