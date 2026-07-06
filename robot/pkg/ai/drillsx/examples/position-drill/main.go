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
	positionCheckTicker  = 250 * time.Millisecond
	positionThreshold    = 2.0 // meters - distance within which player is considered "at" the location
)

//tts:filename=intro.mp3 Starting position drill. The machine will navigate to the baseline and feed you balls at different positions.
//tts:filename=service_line.mp3 Please stand in the middle of the service line.
//tts:filename=baseline.mp3 Please stand in the middle of the baseline.
//tts:filename=ready.mp3 Ready. Get ready for the ball.

//go:embed intro.mp3
var introBytes []byte

//go:embed service_line.mp3
var serviceLineBytes []byte

//go:embed baseline.mp3
var baselineBytes []byte

//go:embed ready.mp3
var readyBytes []byte

type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting position drill")

	// Initialize shot tracker
	tracker := drillutil.NewShotTracker(
		rt.Events,
		rt.Metrics,
		rt.Log,
	)
	defer tracker.Stop()

	// Play intro
	buf := bytes.NewBuffer(introBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play instructions audio", "error", err)
	}

	// Define keypoints
	kp19 := court2d.KP19.Point() // {X: 0.0, Y: -11.8872} - near baseline center
	kp7 := court2d.KP7.Point()   // {X: 0.0, Y: 6.4008} - far service line center
	kp3 := court2d.KP3.Point()   // {X: 0.0, Y: 11.8872} - far baseline center

	// Navigate to keypoint 19
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

		// Tell user to stand at service line (keypoint 7)
		rt.Log.Info("Instructing player to move to service line")
		buf = bytes.NewBuffer(serviceLineBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play service line audio", "error", err)
		}

		// Check every 250ms if player is at keypoint 7
		if err := waitForPlayerAtPosition(ctx, rt, kp7, "KP7 (service line)"); err != nil {
			return err
		}

		// Player is at service line, feed a ball
		rt.Log.Info("Player is at service line, preparing to feed ball")
		buf = bytes.NewBuffer(readyBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play ready audio", "error", err)
		}

		// Calculate shot to keypoint 7
		if err := feedBallToTarget(ctx, rt, tracker, kp19, kp7); err != nil {
			return err
		}

		// Wait between feeds
		select {
		case <-time.After(feedWaitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Tell user to stand at baseline (keypoint 3)
		rt.Log.Info("Instructing player to move to baseline")
		buf = bytes.NewBuffer(baselineBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play baseline audio", "error", err)
		}

		// Check every 250ms if player is at keypoint 3
		if err := waitForPlayerAtPosition(ctx, rt, kp3, "KP3 (baseline)"); err != nil {
			return err
		}

		// Player is at baseline, feed a ball
		rt.Log.Info("Player is at baseline, preparing to feed ball")
		buf = bytes.NewBuffer(readyBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play ready audio", "error", err)
		}

		// Calculate shot to keypoint 3
		if err := feedBallToTarget(ctx, rt, tracker, kp19, kp3); err != nil {
			return err
		}

		// Wait between feeds
		select {
		case <-time.After(feedWaitDuration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// waitForPlayerAtPosition checks every 250ms if any player is at the target position.
func waitForPlayerAtPosition(ctx context.Context, rt api.Runtime, targetPos court2d.Point, positionName string) error {
	ticker := time.NewTicker(positionCheckTicker)
	defer ticker.Stop()

	target := api.Point{X: targetPos.X, Y: targetPos.Y}
	rt.Log.Info("Waiting for player to reach position", "position", positionName, "x", target.X, "y", target.Y)

	for {
		select {
		case <-ticker.C:
			players, err := rt.PlayerProvider.Players()
			if err != nil {
				rt.Log.Error("Failed to get player positions", "error", err)
				continue // Keep trying
			}

			// Check if any player is at the target position
			for i, player := range players {
				dist := player.Point.Distance(target)
				rt.Log.Debug("Checking player position",
					"player", i,
					"player_x", player.Point.X,
					"player_y", player.Point.Y,
					"target_x", target.X,
					"target_y", target.Y,
					"distance", dist)

				if dist <= positionThreshold {
					rt.Log.Info("Player reached target position",
						"position", positionName,
						"player", i,
						"distance", dist)
					return nil
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// feedBallToTarget feeds a ball from the robot's current position to the target.
func feedBallToTarget(ctx context.Context, rt api.Runtime, tracker *drillutil.ShotTracker, robotPos court2d.Point, targetPos court2d.Point) error {
	robotPoint := api.Point{X: robotPos.X, Y: robotPos.Y}
	targetPoint := api.Point{X: targetPos.X, Y: targetPos.Y}

	// Calculate throw settings
	target := drillutil.Target{
		CurrentLocation: robotPoint,
		TargetLocation:  targetPoint,
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

	// Calculate rotation to face target
	loc := api.Location{
		Point:    robotPoint,
		Rotation: robotPoint.Angle(targetPoint),
	}

	// Prepare throw
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

	rt.Log.Info("Shot completed successfully")
	return nil
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
