//go:build hardware

package audio_test

import (
	"os"
	"testing"

	"github.com/notnil/tensa/pkg/audio"
)

func TestPlay(t *testing.T) {
	player := audio.BeepPlayer{}

	files := []string{
		"assets/tensa-boot.wav",
		"assets/tensa-start.wav",
		"assets/tensa-feed.wav",
		"assets/tensa-success.wav",
		"assets/tensa-failure.wav",
		"assets/tensa-failure-2.wav",
		"assets/tensa-pause.wav",
		"assets/tensa-stop.wav",
	}

	for _, file := range files {
		f, err := audio.Assets.Open(file)
		if err != nil {
			t.Fatalf("failed to open %s: %v", file, err)
		}
		if err := player.Play(f); err != nil {
			t.Errorf("failed to play %s: %v", file, err)
		}
		f.Close()
	}
}

func TestPlaySpeech(t *testing.T) {
	player := audio.BeepPlayer{}
	f, err := os.Open("assets/speech.mp3")
	if err != nil {
		t.Fatalf("failed to open speech file: %v", err)
	}
	defer f.Close()
	if err := player.Play(f); err != nil {
		t.Errorf("failed to play speech: %v", err)
	}
}

func TestPlaySoundThenSpeech(t *testing.T) {
	player := audio.BeepPlayer{}

	f1, err := audio.Assets.Open("assets/tensa-boot.wav")
	if err != nil {
		t.Fatalf("failed to open boot sound: %v", err)
	}
	if err := player.Play(f1); err != nil {
		t.Errorf("failed to play boot sound: %v", err)
	}
	f1.Close()

	f2, err := audio.Assets.Open("assets/speech.mp3")
	if err != nil {
		t.Fatalf("failed to open speech file: %v", err)
	}
	defer f2.Close()
	if err := player.Play(f2); err != nil {
		t.Errorf("failed to play speech: %v", err)
	}
}
