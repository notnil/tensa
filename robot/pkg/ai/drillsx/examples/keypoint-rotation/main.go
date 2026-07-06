//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

// Package main implements the criss-cross drill as a plugin.
package main

import (
	"bytes"
	"context"
	_ "embed"
	"math"
	"math/rand"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/rotation"
)

const (
	feedWaitDuration = 3 * time.Second
)

//tts:filename=intro.mp3 In this drill, the machine will drive to a random keypoint and rotate to a random angle.

//go:embed intro.mp3
var introBytes []byte

// crissCross implements the criss-cross drill.
// This drill alternates between different positions and targets to create a crossing pattern.
type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting keypoint rotation drill")

	buf := bytes.NewBuffer(introBytes)
	if err := rt.Audio.Play(buf); err != nil {
		rt.Log.Error("failed to play instructions audio", "error", err)
	}

	kps := []court2d.KeyPoint{
		court2d.KP14,
		court2d.KP15,
		court2d.KP16,
		court2d.KP17,
		court2d.KP18,
		court2d.KP19,
		court2d.KP20,
		court2d.KP21,
	}

	rotations := []float64{
		rotation.FromDegrees(0),
		rotation.FromDegrees(45),
		rotation.FromDegrees(90),
		rotation.FromDegrees(135),
		rotation.FromDegrees(180),
		rotation.FromDegrees(225),
		rotation.FromDegrees(270),
	}

	moveNum := 0
	for {
		select {
		case <-ctx.Done():
			rt.Log.Info("Drill terminated by context", "total_moves", moveNum)
			return ctx.Err()
		case <-time.After(feedWaitDuration):
			moveNum++
			kp := kps[rand.Intn(len(kps))]
			rotation := rotations[rand.Intn(len(rotations))]

			rt.Log.Info("Moving to random position", "move", moveNum, "keypoint", kp, "x", kp.Point().X, "y", kp.Point().Y, "rotation_deg", rotation*180/math.Pi)

			loc := api.Location{
				Point:    api.Point{X: kp.Point().X, Y: kp.Point().Y},
				Rotation: rotation,
			}
			if err := rt.Nav.Navigate(ctx, loc); err != nil {
				rt.Log.Error("Failed to navigate", "error", err, "move", moveNum)
				return err
			}
			rt.Log.Info("Navigation complete", "move", moveNum)
		}
	}
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
