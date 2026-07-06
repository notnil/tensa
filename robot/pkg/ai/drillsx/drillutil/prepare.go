// Package drillutil provides utility functions for drill plugins.
package drillutil

import (
	"context"
	"log/slog"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"golang.org/x/sync/errgroup"
)

// PrepareParams contains all parameters needed for throw preparation.
type PrepareParams struct {
	// RobotLocation is the target position and rotation for the robot
	RobotLocation api.Location
	// MinimumPreparationTime is the minimum time to wait before completing preparation
	MinimumPreparationTime time.Duration
	// ThrowerSettings is the settings for the thrower
	ThrowerSettings api.Settings
}

// PrepareThrow handles navigation and feeder preparation concurrently.
// It navigates the robot to the target position, prepares the feeder with the given config,
// and waits for the specified throw duration before returning.
// All operations are performed concurrently using errgroup for efficient execution.
//
// This function can be used by any drill implementation that needs to prepare for a throw.
func PrepareThrow(ctx context.Context, rt api.Runtime, params PrepareParams, log *slog.Logger) error {
	log.Debug("Preparing throw",
		"robot_x", params.RobotLocation.Point.X,
		"robot_y", params.RobotLocation.Point.Y,
		"robot_rotation", params.RobotLocation.Rotation,
		"minimum_preparation_time", params.MinimumPreparationTime,
	)

	eg, ctx := errgroup.WithContext(ctx)

	// Navigate robot to position
	eg.Go(func() error {
		log.Debug("Starting robot navigation",
			"target_x", params.RobotLocation.Point.X,
			"target_y", params.RobotLocation.Point.Y,
			"target_rotation", params.RobotLocation.Rotation,
		)

		if err := rt.Nav.Navigate(ctx, params.RobotLocation); err != nil {
			log.Error("Robot navigation failed",
				"target_x", params.RobotLocation.Point.X,
				"target_y", params.RobotLocation.Point.Y,
				"target_rotation", params.RobotLocation.Rotation,
				"error", err.Error(),
			)
			return err
		}

		log.Debug("Robot navigation completed successfully",
			"final_x", params.RobotLocation.Point.X,
			"final_y", params.RobotLocation.Point.Y,
			"final_rotation", params.RobotLocation.Rotation,
		)
		return nil
	})

	// Wait for throw duration to allow setup time
	eg.Go(func() error {
		log.Debug("Starting throw duration timer",
			"minimum_preparation_time", params.MinimumPreparationTime,
		)

		select {
		case <-time.After(params.MinimumPreparationTime):
			log.Debug("Throw duration timer completed")
			return nil
		case <-ctx.Done():
			log.Debug("Throw duration timer cancelled by context")
			return ctx.Err()
		}
	})

	// Prepare feeder with configuration
	eg.Go(func() error {
		log.Debug("Preparing feeder",
			"top_throw_speed", params.ThrowerSettings.Top,
			"bottom_throw_speed", params.ThrowerSettings.Bottom,
			"angle", params.ThrowerSettings.Angle,
		)

		if err := rt.Thrower.Set(params.ThrowerSettings); err != nil {
			log.Error("Feeder preparation failed",
				"top_throw_speed", params.ThrowerSettings.Top,
				"bottom_throw_speed", params.ThrowerSettings.Bottom,
				"angle", params.ThrowerSettings.Angle,
				"error", err.Error(),
			)
			return err
		}

		log.Debug("Feeder preparation completed successfully")
		return nil
	})

	eg.Go(func() error {
		log.Debug("Loading ball into thrower")
		err := rt.Thrower.Load(ctx)
		if err != nil {
			log.Error("Failed to load ball into thrower", "error", err.Error())
			return err
		}
		log.Debug("Ball loaded into thrower successfully")
		return nil
	})

	return eg.Wait()
}
