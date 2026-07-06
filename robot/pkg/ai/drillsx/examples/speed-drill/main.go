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

//tts:filename=intro.mp3 The speed drill will annouce the speed of the subjective speed of each shot.
//tts:filename=speed5.mp3 The speed is set to 5
//tts:filename=speed6.mp3 The speed is set to 6
//tts:filename=speed7.mp3 The speed is set to 7

//go:embed intro.mp3
var introBytes []byte

//go:embed speed5.mp3
var speed5Bytes []byte

//go:embed speed6.mp3
var speed6Bytes []byte

//go:embed speed7.mp3
var speed7Bytes []byte

type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting speed drill")

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

	loc := api.Location{
		Point:    api.Point{X: 0, Y: -12},
		Rotation: rotation.FromDegrees(90),
	}
	rt.Log.Info("Navigating to starting position", "x", loc.Point.X, "y", loc.Point.Y, "rotation_deg", 90)
	if err := rt.Nav.Navigate(ctx, loc); err != nil {
		rt.Log.Error("Failed to navigate to starting position", "error", err)
		return err
	}
	rt.Log.Info("Navigation complete")

	type speedPair struct {
		speed drillutil.SubjectiveSpeed
		bytes []byte
	}

	speedPairs := []speedPair{
		{speed: drillutil.Speed5, bytes: speed5Bytes},
		{speed: drillutil.Speed6, bytes: speed6Bytes},
		{speed: drillutil.Speed7, bytes: speed7Bytes},
	}

	for speedIdx, speedPair := range speedPairs {
		rt.Log.Info("Starting speed level", "level", speedIdx+1, "speed", speedPair.speed)

		buf = bytes.NewBuffer(speedPair.bytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play instructions audio", "error", err)
		}
		speed := speedPair.speed
		for i := 0; i < 3; i++ {
			shotNum := i + 1
			rt.Log.Info("Preparing shot", "speed_level", speedIdx+1, "shot", shotNum)
			target := drillutil.Target{
				CurrentLocation: loc.Point,
				TargetLocation:  api.Point{X: 0, Y: 9},
				SpeedRange:      numeric.Range[drillutil.SubjectiveSpeed]{Min: speed, Max: speed},
				SpinRange:       numeric.Range[drillutil.SubjectiveSpin]{Min: drillutil.SpinZero, Max: drillutil.SpinThree},
				AngleRange:      numeric.Range[float64]{Min: rotation.FromDegrees(0), Max: rotation.FromDegrees(40)},
			}
			settings, err := target.Find()
			if err != nil {
				rt.Log.Error("Failed to find target settings", "error", err)
				return err
			}
			rt.Log.Info("Target settings calculated", "top_speed", settings.Top, "bottom_speed", settings.Bottom, "angle", settings.Angle)

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
			rt.Log.Info("Shot completed successfully", "speed_level", speedIdx+1, "shot", shotNum)

			// Wait between feeds
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				rt.Log.Info("Drill terminated by context")
				return ctx.Err()
			}
		}
	}

	rt.Log.Info("Speed drill completed successfully", "total_speed_levels", len(speedPairs), "total_shots", len(speedPairs)*3)
	return nil
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
