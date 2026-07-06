package tensax

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/ai/navigation"
	"github.com/notnil/tensa/pkg/ai/player"
	"github.com/notnil/tensa/pkg/audio"
	"github.com/notnil/tensa/pkg/hware/controller"
	"github.com/notnil/tensa/pkg/hware/tegra"
	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/notnil/tensa/pkg/hware/thrower/clearcore"
	"github.com/notnil/tensa/pkg/hware/wheels"
	"github.com/notnil/tensa/pkg/hware/zed"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
)

var (
	_ controller.Hardware  = (*Tensa)(nil)
	_ navigation.Navigator = (*Tensa)(nil)
	_ navigation.Mover     = (*Tensa)(nil)
)

type Tensa struct {
	c              Config
	logger         *slog.Logger
	logPrefix      string
	wheels         wheels.Wheels
	thrower        thrower.Thrower
	zedArray       zed.Array
	navigator      navigation.Navigator
	locSub         func() pubsubx.Sub[metrics.Metric[location.Loc]]
	audioPlayer    audio.Player
	playerProvider player.Provider
	closers        []io.Closer
	recordCancel   context.CancelFunc
	recordMu       sync.Mutex
}

func New(ctx context.Context, c Config) (*Tensa, error) {
	t := &Tensa{
		c:       c,
		logger:  nil,
		closers: []io.Closer{},
	}
	if err := t.setupLogging(); err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}
	t.logger.Info("Booting Tensa")

	if err := t.setupAudio(); err != nil {
		return nil, fmt.Errorf("failed to setup audio: %w", err)
	}
	if err := t.setupThrower(); err != nil {
		return nil, fmt.Errorf("failed to setup thrower: %w", err)
	}
	if err := t.setupWheels(); err != nil {
		return nil, fmt.Errorf("failed to setup wheels: %w", err)
	}
	if err := t.setupZed(); err != nil {
		return nil, fmt.Errorf("failed to setup zed: %w", err)
	}
	if err := t.setupLocation(); err != nil {
		return nil, fmt.Errorf("failed to setup location: %w", err)
	}
	if err := t.setupNavigation(); err != nil {
		return nil, fmt.Errorf("failed to setup navigation: %w", err)
	}
	if err := t.setupStats(); err != nil {
		return nil, fmt.Errorf("failed to setup stats: %w", err)
	}
	if err := t.setupPlayer(); err != nil {
		return nil, fmt.Errorf("failed to setup player: %w", err)
	}
	f, err := audio.Assets.Open("assets/tensa-boot.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to open boot sound: %w", err)
	}
	// Play asynchronously to prevent blocking during startup
	go func() {
		defer f.Close()
		if err := t.audioPlayer.Play(f); err != nil {
			t.logger.Error("failed to play boot sound", "error", err)
		}
	}()
	return t, nil
}

func (t *Tensa) Wheels() wheels.Wheels {
	return t.wheels
}

func (t *Tensa) Thrower() thrower.Thrower {
	return t.thrower
}

func (t *Tensa) ZedArray() zed.Array {
	return t.zedArray
}

func (t *Tensa) AudioPlayer() audio.Player {
	return t.audioPlayer
}

func (t *Tensa) Stop() error {
	var errs []error
	if err := t.wheels.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop wheels: %w", err))
	}
	if err := t.thrower.Set(thrower.Stop); err != nil {
		errs = append(errs, fmt.Errorf("failed to set thrower to stop: %w", err))
	}
	if err := t.thrower.Spin(0); err != nil {
		errs = append(errs, fmt.Errorf("failed to spin thrower: %w", err))
	}
	f, err := audio.Assets.Open("assets/tensa-stop.wav")
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to open stop sound: %w", err))
	} else {
		// Play asynchronously to prevent blocking during shutdown
		go func() {
			defer f.Close()
			if err := t.audioPlayer.Play(f); err != nil {
				t.logger.Error("failed to play stop sound", "error", err)
			}
		}()
	}
	return errors.Join(errs...)
}

func (t *Tensa) Navigate(ctx context.Context, dest location.Loc) error {
	return t.navigator.Navigate(ctx, dest)
}

// Mover interface implementation
func (t *Tensa) Move(dir, speed float64) error {
	return t.wheels.Move(dir, speed)
}

func (t *Tensa) Rotate(speed float64) error {
	return t.wheels.Rotate(speed)
}

func (t *Tensa) Logger() *slog.Logger {
	return t.logger
}

func (t *Tensa) PlayerProvider() player.Provider {
	return t.playerProvider
}

func (t *Tensa) StartRecording(ctx context.Context) error {
	t.recordMu.Lock()
	defer t.recordMu.Unlock()

	if t.recordCancel != nil {
		return fmt.Errorf("recording already in progress")
	}

	recordCtx, cancel := context.WithCancel(ctx)
	t.recordCancel = cancel

	go func() {
		t.logger.Info("Starting ZED array recording")
		err := t.zedArray.Record(recordCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.logger.Error("ZED recording failed", "error", err)
		}
		t.recordMu.Lock()
		t.recordCancel = nil
		t.recordMu.Unlock()
	}()

	return nil
}

func (t *Tensa) StopRecording() error {
	t.recordMu.Lock()
	defer t.recordMu.Unlock()

	if t.recordCancel == nil {
		return nil // Not recording
	}

	t.logger.Info("Stopping ZED array recording")
	t.recordCancel()
	t.recordCancel = nil
	return nil
}

func (t *Tensa) Close() error {
	t.logger.Info("Closing Tensa")

	var errs []error
	for _, closer := range t.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close all resources: %v", errs)
	}
	return nil
}

func (t *Tensa) addCloser(i interface{}) {
	if closer, ok := i.(io.Closer); ok {
		t.closers = append(t.closers, closer)
	}
}

func (t *Tensa) setupThrower() error {
	t.logger.Info("Setting up thrower")
	switch t.c.Thrower.Type {
	case "mock":
		t.logger.Info("Using Mock Thrower")
		t.thrower = thrower.MockThrower(t.logger)
		if t.c.Audio.SoundEffectsEnabled {
			t.thrower = thrower.NewAudioThrower(t.thrower, t.audioPlayer)
		}
		return nil
	case "clear_core":
		t.logger.Info("Using ClearCore Thrower (TCP)")
		conn, err := net.DialTimeout("tcp", t.c.Thrower.TCPAddress, time.Second*2)
		if err != nil {
			return fmt.Errorf("failed to connect to top motor TCP: %w", err)
		}
		client := clearcore.NewClient(conn, time.Second*2)
		t.thrower = clearcore.New(client, t.c.Thrower.ClearCore)
		t.addCloser(conn)
		if t.c.Audio.SoundEffectsEnabled {
			t.thrower = thrower.NewAudioThrower(t.thrower, t.audioPlayer)
		}
		return nil
	default:
		return fmt.Errorf("invalid thrower type: %s", t.c.Thrower.Type)
	}
}

func (t *Tensa) setupZed() error {
	t.logger.Info("Setting up ZED camera array", "type", t.c.Zed.Type)

	// Handle mock type first
	switch t.c.Zed.Type {
	case "mock":
		t.logger.Info("Using Mock ZED Array")
		t.zedArray = zed.MockArray(t.logger)
		return nil
	}

	// Helper to create a camera config
	createConfig := func(cameraID int, serial uint) zed.Config {
		return zed.Config{
			Resolution:      zed.Resolution(t.c.Zed.Resolution),
			FPS:             t.c.Zed.FPS,
			DepthMode:       zed.DepthMode(t.c.Zed.DepthMode),
			CameraID:        cameraID,
			SerialNumber:    serial,
			MaxExposureTime: time.Duration(t.c.Zed.MaxExposureTime),
			Logger:          t.logger,
		}
	}

	if len(t.c.Zed.SerialNumbers) == 0 {
		return fmt.Errorf("no camera serial numbers configured")
	}

	t.logger.Info("Setting up ZED camera array", "count", len(t.c.Zed.SerialNumbers))

	cameras := make(map[zed.CameraPosition]zed.Camera)
	positions := map[string]zed.CameraPosition{
		"back":  zed.CameraPositionBack,
		"right": zed.CameraPositionRight,
		"front": zed.CameraPositionFront,
		"left":  zed.CameraPositionLeft,
	}

	i := 0
	for name, serial := range t.c.Zed.SerialNumbers {
		pos, ok := positions[name]
		if !ok {
			return fmt.Errorf("invalid camera position: %s", name)
		}

		cfg := createConfig(i, serial)
		cam := zed.NewCamera(cfg)
		cameras[pos] = cam
		t.addCloser(cam)
		i++
	}

	if len(cameras) != 4 {
		return fmt.Errorf("camera array requires exactly 4 cameras, got %d", len(cameras))
	}

	arrayCfg := zed.ArrayConfig{
		Cameras:   cameras,
		OutputDir: t.c.Zed.RecordingDirectory,
		RecordingParams: zed.RecordingParameters{
			CompressionMode: zed.SVOCompressionH265,
		},
		Logger: t.logger,
	}
	arr, err := zed.NewArray(arrayCfg)
	if err != nil {
		return fmt.Errorf("failed to create camera array: %w", err)
	}
	t.zedArray = arr

	return nil
}

func (t *Tensa) Locator() pubsubx.Sub[metrics.Metric[location.Loc]] {
	return t.locSub()
}

func (t *Tensa) setupLocation() error {
	t.logger.Info("Setting up location")
	switch t.c.Location.Type {
	case "mock":
		t.logger.Info("Using Mock Location (random walk)")
		t.locSub = func() pubsubx.Sub[metrics.Metric[location.Loc]] {
			return location.NewRandomWalkSub(location.RandomWalkConfig{
				Polygon:         navigation.DefaultSafeZone(),
				Interval:        time.Millisecond * 100,
				Seed:            time.Now().UnixNano(),
				PositionDelta:   location.DefaultPositionDelta,
				RotationDelta:   location.DefaultRotationDelta,
				InitialPoint:    nil,
				InitialRotation: nil,
			})
		}
		return nil
	default:
		return fmt.Errorf("invalid location type: %s", t.c.Location.Type)
	}
}

func (t *Tensa) setupNavigation() error {
	t.logger.Info("Setting up navigation")
	switch t.c.Navigation.Type {
	case "two_stage":
		t.logger.Info("Using TwoStage Navigator")
		cfg := t.c.Navigation.TwoStage
		translation := navigation.Translation{
			FarSpeed:    cfg.Translation.FarSpeed,
			NearSpeed:   cfg.Translation.NearSpeed,
			OnThreshold: cfg.Translation.OnThreshold,
			SafeZone:    cfg.Translation.SafeZone,
			Timeout:     cfg.Translation.Timeout,
		}
		rotation := navigation.Rotation{
			MaxSpeed:    cfg.Rotation.MaxSpeed,
			MinSpeed:    cfg.Rotation.MinSpeed,
			OnThreshold: cfg.Rotation.OnThreshold,
			Timeout:     cfg.Rotation.Timeout,
		}
		nav := navigation.NewTwoStage(t, t.locSub(), t.logger, translation, rotation, cfg.RestDuration, cfg.Timeout)
		t.navigator = nav
		return nil
	default:
		return fmt.Errorf("invalid navigation type: %s", t.c.Navigation.Type)
	}
}

func (t *Tensa) setupPlayer() error {
	t.logger.Info("Setting up player tracking")
	switch t.c.Player.Type {
	case "mock":
		t.logger.Info("Using Mock Player Provider")
		t.playerProvider = player.NewMockProvider()
		return nil
	default:
		return fmt.Errorf("invalid player type: %s", t.c.Player.Type)
	}
}

func (t *Tensa) setupAudio() error {
	t.logger.Info("Setting up audio")
	if t.c.Audio.SoundEffectsEnabled {
		t.logger.Info("Using BeepPlayer for audio")
		t.audioPlayer = audio.BeepPlayer{}
	} else {
		t.logger.Info("Using NoopPlayer for audio (sound effects disabled)")
		t.audioPlayer = audio.NoopPlayer{}
	}
	return nil
}

func (t *Tensa) setupLogging() error {
	// --- Logging Setup ---
	if err := os.MkdirAll(t.c.Logging.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create timestamped log filename
	timestampStr := time.Now().Format("20060102_150405")
	t.logPrefix = timestampStr
	logFilename := fmt.Sprintf("%s_tensa.log", timestampStr)
	logPath := filepath.Join(t.c.Logging.Directory, logFilename)

	// Open the log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file '%s': %w", logPath, err)
	}
	t.closers = append(t.closers, logFile)

	// Create a multi-writer that writes to both stdout and the file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logOpts := &slog.HandlerOptions{Level: slog.Level(t.c.Logging.Level)}
	logHandler := slog.NewTextHandler(multiWriter, logOpts)
	logger := slog.New(logHandler)
	t.logger = logger
	return nil
}

func (t *Tensa) setupStats() error {
	var parser tegra.Parser
	switch t.c.Stats.Type {
	case "orin":
		parser = tegra.OrinParser{}
	case "nano":
		parser = tegra.NanoParser{}
	default:
		// If not configured or unknown, don't start stats
		return nil
	}

	t.logger.Info("Setting up Tegra stats", "type", t.c.Stats.Type)
	sub := tegra.NewStatsSub(t.logger, parser)

	if t.c.Logging.LogStats {
		ch := make(chan metrics.Metric[tegra.Stats])
		// Use a dedicated context for stats that will be cancelled when Tensa closes
		statsCtx, cancel := context.WithCancel(context.Background())
		t.addCloser(io.Closer(funcCloser(cancel)))

		go func() {
			if err := sub.Subscribe(statsCtx, ch); err != nil && !errors.Is(err, context.Canceled) {
				t.logger.Error("Stats subscription failed", "error", err)
			}
		}()

		go func() {
			for {
				select {
				case <-statsCtx.Done():
					return
				case m := <-ch:
					t.logger.Info("Tegra Stats",
						"ram_used_mb", m.Value.RAM_MB.Used,
						"ram_total_mb", m.Value.RAM_MB.Total,
						"gpu_util", m.Value.GPUUtil,
						"cpu_util", m.Value.CPUUtil,
						"temperatures", m.Value.TemperaturesC,
						"power_mw", m.Value.PowerMW,
					)
				}
			}
		}()
	}

	return nil
}

type funcCloser func()

func (f funcCloser) Close() error {
	f()
	return nil
}
