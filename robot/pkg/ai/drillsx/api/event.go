// Package api provides types and interfaces for tracking tennis game events,
// player positions, and shot metrics during drill sessions.
package api

import (
	"time"
)

// Player represents a tennis player's current state including their position
// and the last time they were detected or updated.
type Player struct {
	// ID is the unique identifier for the player
	ID string `json:"id"`
	// LastPosition is the player's most recently recorded position on the court
	LastPosition Point `json:"last_position"`
	// LastTimestamp is when the player's position was last updated
	LastTimestamp time.Time `json:"last_timestamp"`
}

// PlayerProvider is an interface for components that can supply information
// about active players in the tennis session.
type PlayerProvider interface {
	// Players returns a slice of all currently tracked players
	Players() ([]Location, error)
}

// EventType identifies the kind of tennis game event that occurred.
type EventType string

const (
	// EventTypeBounce indicates the ball bounced on the court surface
	EventTypeBounce EventType = "bounce"
	// EventTypeShot indicates a player hit the ball
	EventTypeShot EventType = "shot"
	// EventTypeNetCollision indicates the ball collided with the net
	EventTypeNetCollision EventType = "net"
)

// Event represents a significant occurrence during a tennis drill or game,
// such as a ball bounce, player shot, or net collision.
type Event struct {
	// Type specifies what kind of event occurred
	Type EventType `json:"type"`
	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Position is the location where the event occurred (optional)
	Position *Point `json:"position,omitempty"`
}

// EventSub is a subscription type for receiving Event updates.
type EventSub Sub[Event]
