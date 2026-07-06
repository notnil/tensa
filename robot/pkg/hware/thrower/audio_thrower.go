package thrower

import (
	"context"

	"github.com/notnil/tensa/pkg/audio"
)

// AudioThrower is a decorator for the Thrower interface that adds sound effects to throw operations.
// It wraps an existing Thrower implementation and plays appropriate audio cues during operations.
// This is typically used when sound effects are enabled in the configuration.
type AudioThrower struct {
	thrower     Thrower
	audioPlayer audio.Player
}

// NewAudioThrower creates a new AudioThrower that wraps the provided Thrower implementation.
// The audioPlayer parameter should be a configured audio.Player instance that can play sound effects.
func NewAudioThrower(thrower Thrower, audioPlayer audio.Player) *AudioThrower {
	return &AudioThrower{
		thrower:     thrower,
		audioPlayer: audioPlayer,
	}
}

// Set configures the throw system motors with the provided settings.
// This method delegates directly to the wrapped Thrower implementation.
func (t *AudioThrower) Set(s Settings) error {
	return t.thrower.Set(s)
}

// Throw activates the dispenser to throw a ball while playing a sound effect.
// The sound is played asynchronously before delegating to the wrapped Thrower.
// Returns any error from the throw operation (audio errors are logged but don't block throwing).
func (t *AudioThrower) Throw(ctx context.Context) error {
	f, err := audio.Assets.Open("assets/tensa-feed.wav")
	if err != nil {
		// Don't block throwing if audio file can't be opened
		return t.thrower.Throw(ctx)
	}

	// Play asynchronously by starting in a goroutine
	// Close the file after playback completes
	go func() {
		defer f.Close()
		if err := t.audioPlayer.Play(f); err != nil {
			// Audio errors are non-fatal, so we just ignore them
			// The throw operation should proceed regardless
		}
	}()

	return t.thrower.Throw(ctx)
}

// Load waits for a ball to be loaded into the throwing position.
// This method delegates directly to the wrapped Thrower implementation.
func (t *AudioThrower) Load(ctx context.Context) error {
	return t.thrower.Load(ctx)
}

// Spin spins the dispenser motor at the given speed.
// This method delegates directly to the wrapped Thrower implementation.
func (t *AudioThrower) Spin(speed float64) error {
	return t.thrower.Spin(speed)
}

// Info returns the current state of the thrower.
// This method delegates directly to the wrapped Thrower implementation.
func (t *AudioThrower) Info() (Info, error) {
	return t.thrower.Info()
}
