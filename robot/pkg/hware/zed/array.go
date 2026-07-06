package zed

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CameraPosition represents the physical position of a camera in the array.
type CameraPosition int

const (
	// CameraPositionBack is the rear-facing camera.
	CameraPositionBack CameraPosition = iota
	// CameraPositionRight is the right-side camera.
	CameraPositionRight
	// CameraPositionFront is the front-facing camera.
	CameraPositionFront
	// CameraPositionLeft is the left-side camera.
	CameraPositionLeft
)

// String returns the string representation of the camera position.
func (p CameraPosition) String() string {
	switch p {
	case CameraPositionBack:
		return "back"
	case CameraPositionRight:
		return "right"
	case CameraPositionFront:
		return "front"
	case CameraPositionLeft:
		return "left"
	default:
		return fmt.Sprintf("unknown(%d)", p)
	}
}

// AllCameraPositions returns all camera positions in order.
func AllCameraPositions() []CameraPosition {
	return []CameraPosition{
		CameraPositionBack,
		CameraPositionRight,
		CameraPositionFront,
		CameraPositionLeft,
	}
}

// Array defines the interface for controlling multiple ZED cameras as a synchronized array.
type Array interface {
	// Start initializes and opens all cameras in the array.
	// Returns an error if any camera fails to open.
	Start(ctx context.Context) error

	// Record starts recording on all cameras and runs grab loops until context is cancelled.
	// Creates a timestamped directory with SVO2 files named by position (back.svo2, etc.).
	// Blocks until the context is cancelled, then stops recording.
	Record(ctx context.Context) error

	// Image grabs frames from all cameras and returns a 2x2 grid image of left views.
	// Layout: top-left=back, top-right=right, bottom-left=front, bottom-right=left.
	Image() (*Image, error)
}

// ArrayConfig contains configuration for creating a camera array.
type ArrayConfig struct {
	// OutputDir is the base directory for recordings. Defaults to current directory.
	OutputDir string

	// Cameras maps camera positions to Camera instances. Must contain all 4 positions.
	Cameras map[CameraPosition]Camera

	// RecordingParams contains shared recording parameters for all cameras.
	RecordingParams RecordingParameters

	// Logger for array operations. Optional, defaults to slog.Default().
	Logger *slog.Logger
}

// ProdArray is the production implementation of the Array interface.
type ProdArray struct {
	config ArrayConfig
	logger *slog.Logger
}

var _ Array = (*ProdArray)(nil)

// NewArray creates a new camera array with the given configuration.
func NewArray(cfg ArrayConfig) (*ProdArray, error) {
	positions := AllCameraPositions()
	for _, pos := range positions {
		if _, ok := cfg.Cameras[pos]; !ok {
			return nil, fmt.Errorf("missing camera for position %s", pos)
		}
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = "."
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &ProdArray{
		config: cfg,
		logger: cfg.Logger.With("system", "zed-array"),
	}, nil
}

// Start initializes and opens all cameras in the array.
func (a *ProdArray) Start(ctx context.Context) error {
	positions := AllCameraPositions()
	a.logger.Info("Starting camera array", "cameras", len(positions))

	var opened []CameraPosition
	for _, pos := range positions {
		select {
		case <-ctx.Done():
			// Context cancelled during startup, close already opened cameras
			a.closePositions(opened)
			return ctx.Err()
		default:
		}

		cam := a.config.Cameras[pos]
		a.logger.Info("Opening camera", "position", pos.String())
		if err := cam.Open(); err != nil {
			// Close already opened cameras
			a.closePositions(opened)
			return fmt.Errorf("failed to open camera %s: %w", pos, err)
		}
		opened = append(opened, pos)
	}

	a.logger.Info("All cameras opened successfully")
	return nil
}

// Record starts recording on all cameras and runs grab loops until context is cancelled.
func (a *ProdArray) Record(ctx context.Context) error {
	positions := AllCameraPositions()

	// Create timestamped output directory
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	recordingDir := filepath.Join(a.config.OutputDir, timestamp)

	a.logger.Info("Creating recording directory", "path", recordingDir)
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		return fmt.Errorf("failed to create recording directory: %w", err)
	}

	// Enable recording for each camera
	var enabled []CameraPosition
	for _, pos := range positions {
		cam := a.config.Cameras[pos]
		params := a.config.RecordingParams
		params.Filename = filepath.Join(recordingDir, fmt.Sprintf("%s.svo2", pos.String()))

		a.logger.Info("Enabling recording", "position", pos.String(), "file", params.Filename)
		if err := cam.EnableRecording(params); err != nil {
			a.disableRecordingPositions(enabled)
			return fmt.Errorf("failed to enable recording on camera %s: %w", pos, err)
		}
		enabled = append(enabled, pos)
	}

	a.logger.Info("Recording started on all cameras", "dir", recordingDir)

	// Start grab loops for each camera
	var wg sync.WaitGroup
	errChan := make(chan error, len(positions))

	for _, pos := range positions {
		cam := a.config.Cameras[pos]
		wg.Add(1)
		go func(p CameraPosition, c Camera) {
			defer wg.Done()
			a.grabLoop(ctx, p, c, errChan)
		}(pos, cam)
	}

	// Wait for context cancellation
	<-ctx.Done()
	a.logger.Info("Context cancelled, stopping recording")

	// Wait for all grab loops to finish
	wg.Wait()

	// Disable recording on all cameras
	a.disableRecordingPositions(positions)

	// Check for any errors from grab loops
	close(errChan)
	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err
		}
		a.logger.Error("Grab loop error", "error", err)
	}

	a.logger.Info("Recording stopped", "dir", recordingDir)

	if firstErr != nil {
		return fmt.Errorf("recording completed with errors: %w", firstErr)
	}
	return nil
}

// grabLoop continuously grabs frames from a camera until context is cancelled.
func (a *ProdArray) grabLoop(ctx context.Context, pos CameraPosition, cam Camera, errChan chan<- error) {
	a.logger.Info("Starting grab loop", "position", pos.String())
	frameCount := 0

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Grab loop stopped", "position", pos.String(), "frames", frameCount)
			return
		default:
		}

		if err := cam.Grab(); err != nil {
			errChan <- fmt.Errorf("camera %s grab failed: %w", pos, err)
			return
		}
		frameCount++
	}
}

// closePositions closes cameras at the given positions.
func (a *ProdArray) closePositions(positions []CameraPosition) {
	for _, pos := range positions {
		a.logger.Info("Closing camera", "position", pos.String())
		if err := a.config.Cameras[pos].Close(); err != nil {
			a.logger.Error("Failed to close camera", "position", pos.String(), "error", err)
		}
	}
}

// disableRecordingPositions disables recording on cameras at the given positions.
func (a *ProdArray) disableRecordingPositions(positions []CameraPosition) {
	for _, pos := range positions {
		a.logger.Info("Disabling recording", "position", pos.String())
		if err := a.config.Cameras[pos].DisableRecording(); err != nil {
			a.logger.Error("Failed to disable recording", "position", pos.String(), "error", err)
		}
	}
}

// Image grabs frames from all cameras and returns a 2x2 grid image of left views.
// Layout: top-left=back, top-right=right, bottom-left=front, bottom-right=left.
func (a *ProdArray) Image() (*Image, error) {
	positions := AllCameraPositions()

	// Grab frames from all cameras in parallel
	var wg sync.WaitGroup
	grabErrors := make([]error, len(positions))

	for i, pos := range positions {
		wg.Add(1)
		go func(idx int, p CameraPosition) {
			defer wg.Done()
			if err := a.config.Cameras[p].Grab(); err != nil {
				grabErrors[idx] = fmt.Errorf("camera %s grab failed: %w", p, err)
			}
		}(i, pos)
	}
	wg.Wait()

	// Check for grab errors
	for _, err := range grabErrors {
		if err != nil {
			return nil, err
		}
	}

	// Retrieve images from all cameras in parallel
	images := make(map[CameraPosition]*Image)
	var mu sync.Mutex
	retrieveErrors := make([]error, len(positions))

	for i, pos := range positions {
		wg.Add(1)
		go func(idx int, p CameraPosition) {
			defer wg.Done()
			img, err := a.config.Cameras[p].RetrieveImage()
			if err != nil {
				retrieveErrors[idx] = fmt.Errorf("camera %s retrieve failed: %w", p, err)
				return
			}
			mu.Lock()
			images[p] = img
			mu.Unlock()
		}(i, pos)
	}
	wg.Wait()

	// Check for retrieve errors
	for _, err := range retrieveErrors {
		if err != nil {
			return nil, err
		}
	}

	// Get dimensions from one of the images (all should be same size)
	singleImg := images[CameraPositionBack]
	width := singleImg.Width
	height := singleImg.Height

	// Create output image (2x2 grid)
	gridWidth := width * 2
	gridHeight := height * 2
	gridData := make([]byte, gridWidth*gridHeight*4)

	// Copy each image to its position in the grid
	// Top-left = back, Top-right = right, Bottom-left = front, Bottom-right = left
	gridPositions := map[CameraPosition]struct{ offsetX, offsetY int }{
		CameraPositionBack:  {0, 0},          // top-left
		CameraPositionRight: {width, 0},      // top-right
		CameraPositionFront: {0, height},     // bottom-left
		CameraPositionLeft:  {width, height}, // bottom-right
	}

	for pos, offset := range gridPositions {
		img := images[pos]
		for y := 0; y < height; y++ {
			srcStart := y * width * 4
			srcEnd := srcStart + width*4
			dstY := offset.offsetY + y
			dstStart := (dstY*gridWidth + offset.offsetX) * 4
			copy(gridData[dstStart:dstStart+width*4], img.Data[srcStart:srcEnd])
		}
	}

	return &Image{
		Width:  gridWidth,
		Height: gridHeight,
		Data:   gridData,
	}, nil
}
