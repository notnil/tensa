// Package main implements the progressive volley drill as a plugin.
package main

import (
	"context"
	"math"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
)

const (
	defaultThrowDuration = 4 * time.Second
	initialThrowDuration = 3 * time.Second
)

// progressiveVolley implements the progressive volley drill.
// This drill progressively increases speed and moves the robot forward.
type progressiveVolley struct{}

// Run executes the progressive volley drill.
func (d *progressiveVolley) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting progressive volley drill")

	// Initialize shot tracker
	tracker := drillutil.NewShotTracker(
		rt.Events,
		rt.Metrics,
		rt.Log,
	)
	defer tracker.Stop()

	// Audio commented out for now
	// go func() {
	// 	f, err := rt.Assets.Open("assets/speech/00002.mp3")
	// 	if err != nil {
	// 		rt.Log.Error("failed to open instructions audio", "error", err)
	// 		return
	// 	}
	// 	defer f.Close()
	// 	if err := rt.Audio.Play(f); err != nil {
	// 		rt.Log.Error("failed to play instructions audio", "error", err)
	// 	}
	// }()

	// Phase 1: Progressive speed increase from fixed position
	// KP19 = {0.0, -11.8872}, North = π/2 radians
	kp19 := api.Point{X: 0.0, Y: -11.8872}
	robotLoc := drillutil.MakeLoc(kp19, math.Pi/2)
	speed := 200.0

	rt.Log.Info("Phase 1: Progressive speed increase", "starting_speed", speed)
	for i := 0; i < 4; i++ {
		select {
		case <-ctx.Done():
			rt.Log.Info("Drill terminated by context in phase 1", "shots_completed", i)
			return ctx.Err()
		default:
			rt.Log.Info("Phase 1 shot", "number", i+1, "speed", speed)

			// Prepare throw
			if err := drillutil.PrepareThrow(ctx, rt, drillutil.PrepareParams{
				RobotLocation:          robotLoc,
				MinimumPreparationTime: initialThrowDuration,
				ThrowerSettings:        api.Settings{Top: speed, Bottom: speed, Angle: 0},
			}, rt.Log); err != nil {
				rt.Log.Error("Failed to prepare throw", "error", err)
				return err
			}

			// Mark feed start for shot tracking (targeting net area)
			target := api.Point{X: 0, Y: 0}
			tracker.Feed([]api.Point{target}, nil)

			// Execute throw
			if err := rt.Thrower.Throw(ctx); err != nil {
				rt.Log.Error("Failed to throw ball", "error", err)
				return err
			}

			speed += 10 // Increase speed for next shot
		}
	}

	// Phase 2: Progressive forward movement
	rt.Log.Info("Phase 2: Progressive forward movement")
	for i := 1; i < 5; i++ {
		select {
		case <-ctx.Done():
			rt.Log.Info("Drill terminated by context in phase 2", "shots_completed", i-1)
			return ctx.Err()
		default:
			rt.Log.Info("Phase 2 shot", "number", i, "position_offset", i)

			// Move forward progressively
			y := -11.8872 + float64(i)
			pt := api.Point{X: 0.0, Y: y}
			robotLoc := drillutil.MakeLoc(pt, math.Pi/2)

			// Prepare throw
			if err := drillutil.PrepareThrow(ctx, rt, drillutil.PrepareParams{
				RobotLocation:          robotLoc,
				MinimumPreparationTime: defaultThrowDuration,
				ThrowerSettings:        api.Settings{Top: speed, Bottom: speed, Angle: 0},
			}, rt.Log); err != nil {
				rt.Log.Error("Failed to prepare throw", "error", err)
				return err
			}

			// Mark feed start for shot tracking (targeting net area)
			target := api.Point{X: 0, Y: 0}
			tracker.Feed([]api.Point{target}, nil)

			// Execute throw
			if err := rt.Thrower.Throw(ctx); err != nil {
				rt.Log.Error("Failed to throw ball", "error", err)
				return err
			}
		}
	}

	rt.Log.Info("Progressive volley drill completed", "total_shots", 8)
	return nil
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &progressiveVolley{}
