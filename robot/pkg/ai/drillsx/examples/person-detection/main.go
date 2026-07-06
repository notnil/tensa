//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

package main

import (
	"bytes"
	"context"
	_ "embed"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

const (
	personCheckInterval = 1 * time.Second
)

//tts:filename=person_detected.mp3 Person detected.

//go:embed person_detected.mp3
var personDetectedBytes []byte

type drill struct{}

func (d *drill) Run(ctx context.Context, rt api.Runtime) error {
	rt.Log.Info("Starting person detection drill")

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			rt.Log.Info("Person detection drill stopped")
			return ctx.Err()
		default:
		}

		// Check for players/people
		players, err := rt.PlayerProvider.Players()
		if err != nil {
			rt.Log.Error("Failed to get player positions", "error", err)
			time.Sleep(personCheckInterval)
			continue
		}

		// If any players are detected, play the audio
		if len(players) > 0 {
			rt.Log.Info("Person detected", "num_players", len(players))
			buf := bytes.NewBuffer(personDetectedBytes)
			if err := rt.Audio.Play(buf); err != nil {
				rt.Log.Error("Failed to play person detected audio", "error", err)
			}
			// Sleep after playing audio
			time.Sleep(personCheckInterval)
		} else {
			rt.Log.Debug("No person detected")
			time.Sleep(personCheckInterval)
		}
	}
}

// Drill is the exported symbol that the plugin host loads.
var Drill api.Drill = &drill{}
