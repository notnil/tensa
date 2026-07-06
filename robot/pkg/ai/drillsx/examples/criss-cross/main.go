//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

package main

import (
	"bytes"
	"context"
	_ "embed"
	"math"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
)

const (
	defaultThrowDuration = 3 * time.Second
	feedWaitDuration     = 500 * time.Millisecond
)

//tts:filename=intro.mp3 Starting criss cross drill. Get ready to move at the baseline.

//go:embed intro.mp3
var introBytes []byte

// crissCross implements the criss-cross drill.
// This drill alternates between different positions and targets to create a crossing pattern.
type crissCross struct{}

// Run executes the criss-cross drill.
func (d *crissCross) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting criss-cross drill")

	// Initialize shot tracker
	tracker := drillutil.NewShotTracker(
		rt.Events,
		rt.Metrics,
		rt.Log,
	)
	defer tracker.Stop()

	go func() {
		buf := bytes.NewBuffer(introBytes)
		if err := rt.Audio.Play(buf); err != nil {
			rt.Log.Error("failed to play instructions audio", "error", err)
		}
	}()

	// Define robot positions (alternating between different baseline positions)
	// KP19 = {0.0, -11.8872}, KP20 = {4.1148, -11.8872}, KP18 = {-4.1148, -11.8872}
	kp19 := api.Point{X: 0.0, Y: -11.8872}
	kp20 := api.Point{X: 4.1148, Y: -11.8872}
	kp18 := api.Point{X: -4.1148, Y: -11.8872}

	points := []api.Point{
		kp19.Halfway(kp20),
		kp19.Halfway(kp20),
		kp19.Halfway(kp20),
		kp19,
		kp19,
		kp19,
		kp19.Halfway(kp18),
		kp19.Halfway(kp18),
		kp19.Halfway(kp18),
	}

	// Define targets (alternating between different court positions)
	// KP3 = {0.0, 11.8872}, KP4 = {4.1148, 11.8872}, KP2 = {-4.1148, 11.8872}
	kp3 := api.Point{X: 0.0, Y: 11.8872}
	kp4 := api.Point{X: 4.1148, Y: 11.8872}
	kp2 := api.Point{X: -4.1148, Y: 11.8872}

	facing := []api.Point{
		kp3.Halfway(kp4),
		kp3,
		kp3.Halfway(kp2),
		kp3.Halfway(kp4),
		kp3,
		kp3.Halfway(kp2),
		kp3.Halfway(kp4),
		kp3,
		kp3.Halfway(kp2),
	}

	angleRange := api.Range{
		Min: 0,
		Max: 5 * math.Pi / 180, // 5 degrees in radians
	}

	// Execute the sequence
	for i := 0; i < len(points); i++ {
		select {
		case <-ctx.Done():
			rt.Log.Info("Drill terminated by context", "shots_completed", i)
			return ctx.Err()
		default:
			rt.Log.Info("Starting shot", "number", i+1, "total", len(points))

			// Calculate rotation to face target
			rot := points[i].Angle(facing[i])
			robotLoc := drillutil.MakeLoc(points[i], rot)

			// Sample speed and angle
			speed, err := drillutil.SampleRange(rt.Rnd, api.Range{Min: 210, Max: 240})
			if err != nil {
				rt.Log.Error("Failed to sample speed", "error", err)
				return err
			}
			angle, err := drillutil.SampleRange(rt.Rnd, angleRange)
			if err != nil {
				rt.Log.Error("Failed to sample angle", "error", err)
				return err
			}
			rt.Log.Info("Sampled throw parameters", "speed", speed, "angle_deg", angle*180/math.Pi)

			// Prepare throw
			if err := drillutil.PrepareThrow(ctx, rt, drillutil.PrepareParams{
				RobotLocation:          robotLoc,
				MinimumPreparationTime: defaultThrowDuration,
				ThrowerSettings:        api.Settings{Top: speed, Bottom: speed, Angle: angle},
			}, rt.Log); err != nil {
				rt.Log.Error("Failed to prepare throw", "error", err)
				return err
			}

			// Mark feed start for shot tracking
			tracker.Feed([]api.Point{facing[i]}, nil)

			// Execute throw
			if err := rt.Thrower.Throw(ctx); err != nil {
				rt.Log.Error("Failed to throw ball", "error", err)
				return err
			}

			rt.Log.Info("Shot completed", "number", i+1)

			// Wait between feeds
			select {
			case <-time.After(feedWaitDuration):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	rt.Log.Info("Criss-cross drill completed", "total_shots", len(points))
	return nil
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &crissCross{}
