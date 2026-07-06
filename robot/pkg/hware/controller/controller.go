// Package controller provides interfaces and a basic implementation for controlling
// Tensa hardware components. It defines abstractions to interact with wheels and
// thrower mechanisms, allowing different controller implementations to update hardware
// based on input events.
package controller

import (
	"context"
	"log/slog"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/ai/player"
	"github.com/notnil/tensa/pkg/audio"
	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/notnil/tensa/pkg/hware/wheels"
	"github.com/notnil/tensa/pkg/hware/zed"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
)

type LocatorType uint8

const (
	LocatorTypeMock LocatorType = iota
	LocatorTypeRawLocation
	LocatorTypeFilteredLocation
)

// Hardware defines the interface for accessing Tensa hardware components.
// It provides methods for retrieving the wheel controller and the thrower mechanism.
type Hardware interface {
	// Logger returns the logger for the hardware.
	Logger() *slog.Logger
	// Stop stops all hardware components.
	Stop() error
	// Wheels returns the wheels controller interface for movement operations.
	Wheels() wheels.Wheels
	// Thrower returns the thrower interface for operating the throwing mechanism.
	Thrower() thrower.Thrower
	// Navigate moves the hardware to a specific location using the configured navigator.
	Navigate(ctx context.Context, dest location.Loc) error
	// Locator returns a subscription for localization metrics.
	Locator() pubsubx.Sub[metrics.Metric[location.Loc]]
	// AudioPlayer returns the audio player for playing sound effects.
	AudioPlayer() audio.Player
	// PlayerProvider returns the player tracking provider.
	PlayerProvider() player.Provider
	// ZedArray returns the ZED camera array interface for recording.
	ZedArray() zed.Array
	// StartRecording starts recording on all cameras.
	StartRecording(ctx context.Context) error
	// StopRecording stops recording on all cameras.
	StopRecording() error
}

// Controller defines the interface that all hardware controllers must implement.
// The Control method should continuously process input events and update the provided
// hardware accordingly until the context is canceled or an error occurs.
type Controller interface {
	// Control starts the controller to process input and update hardware.
	Control(ctx context.Context, h Hardware) error
}

// Mock is a dummy implementation of the Controller interface.
// It is useful for testing or when no real controller is needed.
type Mock struct{}

// Control for Mock simply waits until the context is canceled, then returns the context's error.
func (m Mock) Control(ctx context.Context, h Hardware) error {
	<-ctx.Done()
	return ctx.Err()
}
