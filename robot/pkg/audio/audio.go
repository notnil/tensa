package audio

import (
	"embed"
	"io"
)

//go:embed assets/*.wav assets/*.mp3
var Assets embed.FS

// Player defines a simple audio playback interface. In test and headless
// environments, the default NoopPlayer is used and requires no system audio.
type Player interface {
	Play(r io.Reader) error
}

// NoopPlayer satisfies Player but performs no playback.
type NoopPlayer struct{}

func (n NoopPlayer) Play(r io.Reader) error { return nil }
