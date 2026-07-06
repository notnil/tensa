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
	defaultThrowDuration = 2 * time.Second
	feedWaitDuration     = 2 * time.Second
	machineMoveRange     = 1   // meters - small random movement range
	targetTolerance      = 0.5 // meters - acceptable distance error for target finding
)

//tts:filename=intro.mp3 Starting inside out forehand drill. We'll alternate between cross court and inside out forehands. Remember to recover to the center after each ball. For the first three pairs we will go a bit slower to warm up.
//tts:filename=middle.mp3 Okay picking up the pace. For the next three pairs, give maximum effort!
//tts:filename=completion.mp3 Great job! You are ready for Wimbledon! Your stats are available in the application.

//go:embed intro.mp3
var introBytes []byte

//go:embed middle.mp3
var middleBytes []byte

//go:embed completion.mp3
var completionBytes []byte

type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting inside-out forehand drill")

	// Initialize shot tracker
	tracker := drillutil.NewShotTracker(
		rt.Events,
		rt.Metrics,
		rt.Log,
	)
	defer tracker.Stop()

	// Play intro audio
	buf := bytes.NewBuffer(introBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play intro audio", "error", err)
	}

	// Define keypoints
	kp14 := court2d.KP14.Point() // {X: 0.0, Y: -6.4008} - near service line center
	kp15 := court2d.KP15.Point() // {X: 0.0, Y: -6.4008} - near service line center
	kp18 := court2d.KP18.Point() // {X: 0.0, Y: -11.8872} - near baseline center
	kp19 := court2d.KP19.Point() // {X: 0.0, Y: -11.8872} - near baseline center
	kp4 := court2d.KP4.Point()   // {X: 4.1148, Y: 11.8872} - far right baseline (ad court)
	kp3 := court2d.KP3.Point()   // {X: 0.0, Y: 11.8872} - far right baseline (ad court)
	kp7 := court2d.KP7.Point()   // {X: 0.0, Y: -6.4008} - near service line center
	kp8 := court2d.KP8.Point()   // {X: 0.0, Y: 6.4008} - far service line center

	// Machine starts halfway between KP15, KP14, KP18 and KP19
	machineStartX := (kp15.X + kp14.X + kp18.X + kp19.X) / 4
	machineStartY := (kp15.Y + kp14.Y + kp18.Y + kp19.Y) / 4
	machineStartAPI := api.Point{X: machineStartX, Y: machineStartY}
	rt.Log.Info("Machine starting position calculated", "x", machineStartAPI.X, "y", machineStartAPI.Y)

	// Target locations - deep in ad court (inside-out) and deuce court (cross-court)
	// make the target point the center of 3,4,7,8
	adTarget := api.Point{X: (kp3.X + kp4.X + kp7.X + kp8.X) / 4, Y: (kp3.Y + kp4.Y + kp7.Y + kp8.Y) / 4}
	// Deuce target is mirror of ad target (x multiplied by -1)
	deuceTarget := api.Point{X: -adTarget.X, Y: adTarget.Y}

	// Navigate to starting position, facing toward center of both targets
	avgTarget := api.Point{X: (adTarget.X + deuceTarget.X) / 2, Y: (adTarget.Y + deuceTarget.Y) / 2}
	rt.Log.Info("Navigating to starting position", "x", machineStartAPI.X, "y", machineStartAPI.Y)
	if err := rt.Nav.Navigate(ctx, api.Location{
		Point:    machineStartAPI,
		Rotation: machineStartAPI.Angle(avgTarget), // Face toward center of both targets
	}); err != nil {
		rt.Log.Error("Failed to navigate to starting position", "error", err)
		return err
	}
	rt.Log.Info("Navigation complete")

	// Helper function to execute a single throw
	throwBall := func(currentPos api.Point, target api.Point, shotType string, speedRange numeric.Range[drillutil.SubjectiveSpeed], shotNum int) error {
		// Navigate to position facing target
		if err := rt.Nav.Navigate(ctx, api.Location{
			Point:    currentPos,
			Rotation: currentPos.Angle(target),
		}); err != nil {
			rt.Log.Error("Failed to navigate to position", "error", err)
			return err
		}

		// Calculate throw settings
		targetCalc := drillutil.Target{
			CurrentLocation: currentPos,
			TargetLocation:  target,
			SpeedRange:      speedRange,
			SpinRange:       numeric.Range[drillutil.SubjectiveSpin]{Min: drillutil.SpinZero, Max: drillutil.SpinThree},
			AngleRange:      numeric.Range[float64]{Min: rotation.FromDegrees(10), Max: rotation.FromDegrees(30)},
			Threshold:       targetTolerance,
		}

		settings, err := targetCalc.Find()
		if err != nil {
			rt.Log.Error("Failed to find target settings", "error", err)
			return err
		}
		rt.Log.Info("Target settings calculated", "shot_type", shotType, "top_speed", settings.Top, "bottom_speed", settings.Bottom, "angle", settings.Angle)

		// Prepare throw
		if err := drillutil.PrepareThrow(ctx, rt, drillutil.PrepareParams{
			RobotLocation: api.Location{
				Point:    currentPos,
				Rotation: currentPos.Angle(target),
			},
			MinimumPreparationTime: defaultThrowDuration,
			ThrowerSettings:        settings,
		}, rt.Log); err != nil {
			rt.Log.Error("Failed to prepare throw", "error", err)
			return err
		}

		tracker.Feed([]api.Point{target}, nil)

		// Execute throw
		if err := rt.Thrower.Throw(ctx); err != nil {
			rt.Log.Error("Failed to throw ball", "error", err)
			return err
		}

		rt.Log.Info("Shot completed", "shot_type", shotType, "shot_number", shotNum)
		return nil
	}

	// First set: 3 pairs at slower speed (warmup) - alternating cross-court and inside-out
	rt.Log.Info("Starting first set - warmup (3 pairs: 6 shots)")
	for pair := 0; pair < 3; pair++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Get current robot position with slight random movement
			offsetX, err := drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample X offset", "error", err)
				return err
			}
			offsetY, err := drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample Y offset", "error", err)
				return err
			}
			currentPos := api.Point{
				X: machineStartAPI.X + offsetX,
				Y: machineStartAPI.Y + offsetY,
			}

			// Cross-court forehand (deuce to deuce)
			if err := throwBall(currentPos, deuceTarget, "cross-court", numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed5, Max: drillutil.Speed6}, pair*2+1); err != nil {
				return err
			}

			// Wait between shots in pair
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}

			// Update position for second shot
			offsetX, err = drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample X offset", "error", err)
				return err
			}
			offsetY, err = drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample Y offset", "error", err)
				return err
			}
			currentPos = api.Point{
				X: machineStartAPI.X + offsetX,
				Y: machineStartAPI.Y + offsetY,
			}

			// Inside-out forehand (ad to ad)
			if err := throwBall(currentPos, adTarget, "inside-out", numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed5, Max: drillutil.Speed6}, pair*2+2); err != nil {
				return err
			}

			// Wait between pairs
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Play middle instruction audio
	rt.Log.Info("Playing middle instruction audio")
	buf = bytes.NewBuffer(middleBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play middle audio", "error", err)
	}

	// Second set: 3 pairs at faster speed (maximum effort) - alternating cross-court and inside-out
	rt.Log.Info("Starting second set - maximum effort (3 pairs: 6 shots)")
	for pair := 0; pair < 3; pair++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Get current robot position with slight random movement
			offsetX, err := drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample X offset", "error", err)
				return err
			}
			offsetY, err := drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample Y offset", "error", err)
				return err
			}
			currentPos := api.Point{
				X: machineStartAPI.X + offsetX,
				Y: machineStartAPI.Y + offsetY,
			}

			// Cross-court forehand (deuce to deuce) - maximum effort
			if err := throwBall(currentPos, deuceTarget, "cross-court", numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed6, Max: drillutil.Speed7}, pair*2+1); err != nil {
				return err
			}

			// Wait between shots in pair
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}

			// Update position for second shot
			offsetX, err = drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample X offset", "error", err)
				return err
			}
			offsetY, err = drillutil.SampleRange(rt.Rnd, api.Range{Min: -machineMoveRange, Max: machineMoveRange})
			if err != nil {
				rt.Log.Error("Failed to sample Y offset", "error", err)
				return err
			}
			currentPos = api.Point{
				X: machineStartAPI.X + offsetX,
				Y: machineStartAPI.Y + offsetY,
			}

			// Inside-out forehand (ad to ad) - maximum effort
			if err := throwBall(currentPos, adTarget, "inside-out", numeric.Range[drillutil.SubjectiveSpeed]{Min: drillutil.Speed7, Max: drillutil.Speed8}, pair*2+2); err != nil {
				return err
			}

			// Wait between pairs
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Play completion audio
	rt.Log.Info("Playing completion audio")
	buf = bytes.NewBuffer(completionBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play completion audio", "error", err)
	}

	return nil
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
