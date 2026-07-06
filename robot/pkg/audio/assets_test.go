package audio_test

import (
	"io"
	"testing"

	"github.com/notnil/tensa/pkg/audio"
)

func TestEmbeddedAssetsOpen(t *testing.T) {
	files := []string{
		"assets/tensa-boot.wav",
		"assets/tensa-start.wav",
		"assets/tensa-feed.wav",
		"assets/tensa-success.wav",
		"assets/tensa-failure.wav",
		"assets/tensa-failure-2.wav",
		"assets/tensa-pause.wav",
		"assets/tensa-stop.wav",
		"assets/speech.mp3",
	}

	player := audio.NoopPlayer{}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			f, err := audio.Assets.Open(file)
			if err != nil {
				t.Fatalf("failed to open embedded asset: %v", err)
			}
			defer f.Close()

			if _, err := io.Copy(io.Discard, f); err != nil {
				t.Fatalf("failed to read embedded asset: %v", err)
			}
			if err := player.Play(f); err != nil {
				t.Fatalf("noop player returned error: %v", err)
			}
		})
	}
}
