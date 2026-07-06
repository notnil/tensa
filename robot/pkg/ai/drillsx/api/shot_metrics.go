package api

import (
	"context"
	"time"
)

// ShotOutcome represents the result of a shot based on raw event data.
// The tracker collects data for later analysis by higher-level systems.
type ShotOutcome string

const (
	// OutcomeUnknown indicates no information was received about what happened
	// to the ball after it was fed (no hit, no bounce, no net collision detected).
	OutcomeUnknown ShotOutcome = "unknown"

	// OutcomeNet indicates the ball hit the net during the shot.
	// This is detected via EventTypeNetCollision events.
	OutcomeNet ShotOutcome = "net"

	// OutcomeBounce indicates at least one bounce was detected and recorded.
	// The bounce locations are stored in the Bounces field for later analysis
	// to determine accuracy and whether the shot was in or out of target zones.
	OutcomeBounce ShotOutcome = "bounce"
)

// BounceRecord captures a single bounce event with its location and time.
type BounceRecord struct {
	Location  Point     `json:"location"`
	Timestamp time.Time `json:"timestamp"`
}

// ShotMetric contains comprehensive data about a completed shot.
// The Targets and Zones are recorded for later analysis by higher-level systems
// to determine accuracy and whether shots landed in/out.
type ShotMetric struct {
	ShotCount      int            `json:"shot_count"`
	Targets        []Point        `json:"targets"`
	Zones          []Polygon      `json:"zones"`
	FeedTime       time.Time      `json:"feed_time"`
	HitTime        *time.Time     `json:"hit_time,omitempty"`
	PlayerPosition *Point         `json:"player_position,omitempty"`
	Bounces        []BounceRecord `json:"bounces"`
	Outcome        ShotOutcome    `json:"outcome"`
}

// ShotMetricWriter defines the interface for writing shot metrics.
type ShotMetricWriter interface {
	WriteShotMetric(ctx context.Context, metric ShotMetric) error
}
