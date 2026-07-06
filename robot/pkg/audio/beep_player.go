package audio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

var (
	speakerOnce sync.Once
)

const standardSampleRate = beep.SampleRate(48000)

func initSpeaker() {
	speakerOnce.Do(func() {
		ensureALSAConfig()
		speaker.Init(standardSampleRate, standardSampleRate.N(time.Second/10))
	})
}

// ensureALSAConfig sets a per-process ALSA config to route the default PCM to a
// specific hardware device when running on Linux and no ALSA config path is
// already provided. This helps in environments like SSH where the default
// device may not be the desired one. The device can be overridden via the
// TENSA_ALSA_PLUGHW environment variable in the form "card,device" (e.g.,
// "0,0").
func ensureALSAConfig() {
	if runtime.GOOS != "linux" {
		return
	}
	if v := os.Getenv("ALSA_CONFIG_PATH"); v != "" {
		return
	}

	// Allow overriding the card,device via env; default to 0,0
	card := "0"
	device := "0"
	if v := strings.TrimSpace(os.Getenv("TENSA_ALSA_PLUGHW")); v != "" {
		parts := strings.Split(v, ",")
		if len(parts) == 2 {
			if strings.TrimSpace(parts[0]) != "" {
				card = strings.TrimSpace(parts[0])
			}
			if strings.TrimSpace(parts[1]) != "" {
				device = strings.TrimSpace(parts[1])
			}
		}
	}

	conf := fmt.Sprintf("pcm.usb {\n\ttype hw\n\tcard %s\n\tdevice %s\n}\n\npcm.!default {\n\ttype plug\n\tslave.pcm \"usb\"\n}\n\nctl.!default {\n\ttype hw\n\tcard %s\n}\n", card, device, card)
	path := filepath.Join(os.TempDir(), fmt.Sprintf("tensa-alsa-%d.conf", os.Getpid()))
	if err := os.WriteFile(path, []byte(conf), 0644); err != nil {
		return
	}
	_ = os.Setenv("ALSA_CONFIG_PATH", path)
}

// BeepPlayer uses the Beep library to play sounds.
type BeepPlayer struct{}

func (b BeepPlayer) Play(r io.Reader) error {
	initSpeaker()
	streamer, err := b.getStreamer(r)
	if err != nil {
		return err
	}
	defer streamer.Close()

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() { done <- true })))
	<-done
	return nil
}

func MonoToStereo(s beep.Streamer) beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = s.Stream(samples)
		for i := 0; i < n; i++ {
			mono := samples[i][0]
			samples[i][0] = mono
			samples[i][1] = mono
		}
		return n, ok
	})
}

func EnsureStereo(s beep.Streamer, format beep.Format) beep.Streamer {
	if format.NumChannels == 2 {
		return s
	}
	return MonoToStereo(s)
}

func (b BeepPlayer) getStreamer(r io.Reader) (beep.StreamSeekCloser, error) {
	// Read all data to detect MIME type and create a seekable reader
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Detect MIME type
	mtype := mimetype.Detect(data)
	reader := bytes.NewReader(data)

	var streamer beep.StreamSeekCloser
	var format beep.Format
	switch mtype.String() {
	case "audio/wav", "audio/x-wav", "audio/wave":
		streamer, format, err = wav.Decode(io.NopCloser(reader))
		if err != nil {
			return nil, fmt.Errorf("failed to decode wav: %w", err)
		}
	case "audio/mpeg", "audio/mp3":
		streamer, format, err = mp3.Decode(io.NopCloser(reader))
		if err != nil {
			return nil, fmt.Errorf("failed to decode mp3: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported audio MIME type: %s", mtype.String())
	}

	stereoStreamer := EnsureStereo(streamer, format)
	if seeker, ok := streamer.(beep.StreamSeekCloser); ok {
		return &stereoStreamSeekCloser{Streamer: stereoStreamer, seeker: seeker}, nil
	}
	return &stereoStreamSeekCloser{Streamer: stereoStreamer, seeker: streamer}, nil
}

type stereoStreamSeekCloser struct {
	beep.Streamer
	seeker beep.StreamSeekCloser
}

func (s *stereoStreamSeekCloser) Len() int         { return s.seeker.Len() }
func (s *stereoStreamSeekCloser) Position() int    { return s.seeker.Position() }
func (s *stereoStreamSeekCloser) Seek(p int) error { return s.seeker.Seek(p) }
func (s *stereoStreamSeekCloser) Close() error     { return s.seeker.Close() }
