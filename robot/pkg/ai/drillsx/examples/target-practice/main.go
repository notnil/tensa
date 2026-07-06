//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

package main

import (
	"bytes"
	"context"
	_ "embed"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/numeric"
	"github.com/notnil/tensa/pkg/util/rotation"
)

const (
	defaultThrowDuration = 3 * time.Second
	feedWaitDuration     = 3 * time.Second
)

//tts:filename=intro.mp3 This drill is target practice. The machine goes to keypoint 19 and shoots at keypoints in order from 1 to 8.
//tts:filename=starting.mp3 Starting with keypoint 1.

//go:embed intro.mp3
var introBytes []byte

//go:embed starting.mp3
var startingBytes []byte

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

	targetKPs := []court2d.KeyPoint{
		court2d.KP1,
		court2d.KP2,
		court2d.KP3,
		court2d.KP4,
		court2d.KP5,
		court2d.KP6,
		court2d.KP7,
		court2d.KP8,
	}

	kp19 := court2d.KP19.Point()

	rt.Log.Info("Navigating to starting position", "keypoint", "KP19", "x", kp19.X, "y", kp19.Y)
	if err := rt.Nav.Navigate(ctx, api.Location{
		Point:    api.Point{X: kp19.X, Y: kp19.Y},
		Rotation: rotation.FromDegrees(90),
	}); err != nil {
		rt.Log.Error("Failed to navigate to starting position", "error", err)
		return err
	}
	rt.Log.Info("Navigation complete")

	roundNum := 0
	for {
		roundNum++
		rt.Log.Info("Starting new round", "round", roundNum)

		// announce starting keypoint
		buf = bytes.NewBuffer(startingBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play instructions audio", "error", err)
		}
		shotNum := 0
		for _, pt := range targetKPs {
			shotNum++
			rt.Log.Info("Preparing shot", "round", roundNum, "shot", shotNum, "target_x", pt.Point().X, "target_y", pt.Point().Y)
			targetPt := api.Point{X: pt.Point().X, Y: pt.Point().Y}
			target := drillutil.Target{
				CurrentLocation: api.Point{X: kp19.X, Y: kp19.Y},
				TargetLocation:  targetPt,
				SpeedRange:      numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed5, Max: drillutil.Speed8},
				SpinRange:       numeric.Range[drillutil.SubjectiveSpin]{Min: drillutil.SpinZero, Max: drillutil.SpinThree},
				AngleRange:      numeric.Range[float64]{Min: rotation.FromDegrees(10), Max: rotation.FromDegrees(35)},
			}
			settings, err := target.Find()
			if err != nil {
				rt.Log.Error("Failed to find target settings", "error", err)
				return err
			}
			rt.Log.Info("Target settings calculated", "top_speed", settings.Top, "bottom_speed", settings.Bottom, "angle", settings.Angle)

			loc := api.Location{
				Point:    api.Point{X: kp19.X, Y: kp19.Y},
				Rotation: api.Point{X: kp19.X, Y: kp19.Y}.Angle(targetPt),
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
			rt.Log.Info("Shot completed successfully", "round", roundNum, "shot", shotNum)

			// Wait between feeds
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
