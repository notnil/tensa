// Package api provides the core interfaces and types for authoring drill plugins.
// This package serves as the stability boundary between the host runtime and drill plugins.
//
// All interfaces and types are redeclared here to minimize shared dependencies between
// the main application and plugins. This allows plugins to be built and updated independently
// without requiring exact version matching of all dependencies.
package api

import (
	"context"
	"log/slog"
	"math/rand"
)

// Runtime aggregates the core subsystems and utilities exposed to drills.
// It provides access to navigation, feeding, audio, logging, randomness, and metrics.
type Runtime struct {
	// Nav provides navigation capabilities for moving to specific locations and rotations
	Nav Navigator
	// Mover provides low-level movement control for direct motor commands
	Mover Mover
	// Thrower provides ball feeding capabilities with configurable speed and angle
	Thrower Thrower
	// Audio provides audio playback for announcements and cues
	Audio AudioPlayer
	// Events provides access to system events for shot tracking and other telemetry
	Events EventSub
	// Metrics provides shot metric collection and writing capabilities
	Metrics ShotMetricWriter
	// PlayerProvider provides access to players last location and timestamp
	PlayerProvider PlayerProvider
	// Log provides structured logging capabilities
	Log *slog.Logger
	// Rnd provides a seeded random number generator for sampling
	Rnd *rand.Rand
}

// Drill defines the interface that all drill plugins must implement.
// The host runtime loads plugins and invokes the Run method to execute the drill.
type Drill interface {
	// Run executes the drill with the provided runtime context.
	// The ctx parameter can be used for cancellation and timeouts.
	// The rt parameter provides access to all runtime capabilities including
	// navigation, thrower, audio, events, metrics, logging, and randomness.
	// Return an error to abort execution.
	Run(ctx context.Context, rt Runtime) error
}
