// Package drillutil provides utility functions for drill plugins.
package drillutil

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

// ShotTracker automatically tracks shot metrics by collecting system events
// and analyzing them when shots conclude.
// The bounce window is implicitly until the next feed, and max bounces is 2.
type ShotTracker struct {
	writer api.ShotMetricWriter
	log    *slog.Logger

	// Event subscription
	eventSub api.EventSub
	eventCtx context.Context
	eventCh  chan api.Event
	cancelFn context.CancelFunc
	stopOnce sync.Once

	// Event collection
	mu              sync.Mutex
	collectedEvents []api.Event

	// Current shot state
	shotCount       int
	currentTargets  []api.Point
	currentZones    []api.Polygon
	feedTime        time.Time
	hasPreviousShot bool
}

// NewShotTracker creates and starts a new ShotTracker.
// The writer is used to persist completed shot metrics.
// The tracker starts a background goroutine to listen for events; call Stop() to clean up.
// The bounce window is implicitly until the next feed, and max bounces is 2.
// Both eventSub and writer may be nil; if eventSub is nil, the tracker will operate
// without collecting event data (all shots will have OutcomeUnknown).
func NewShotTracker(
	eventSub api.EventSub,
	writer api.ShotMetricWriter,
	log *slog.Logger,
) *ShotTracker {
	eventCtx, cancelFn := context.WithCancel(context.Background())
	eventCh := make(chan api.Event, 100)

	st := &ShotTracker{
		writer:          writer,
		log:             log,
		eventSub:        eventSub,
		eventCtx:        eventCtx,
		eventCh:         eventCh,
		cancelFn:        cancelFn,
		collectedEvents: make([]api.Event, 0, 100),
		shotCount:       0,
	}

	// Start event listener goroutine
	go st.listenEvents()

	return st
}

// Feed marks the start of a new shot and finalizes the previous shot (if any).
// The targets and zones parameters define the intent for the upcoming shot.
func (st *ShotTracker) Feed(targets []api.Point, zones []api.Polygon) {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Finalize the previous shot if one exists
	if st.hasPreviousShot {
		st.finalizeLocked()
	}

	// Increment shot counter and set up new shot
	st.shotCount++
	st.currentTargets = append([]api.Point(nil), targets...)
	st.currentZones = append([]api.Polygon(nil), zones...)
	st.feedTime = time.Now()
	st.hasPreviousShot = true

	st.log.Debug("Shot tracker: new feed",
		"shot_count", st.shotCount,
		"target_count", len(targets),
		"zone_count", len(zones),
		"feed_time", st.feedTime,
	)
}

// Stop finalizes the last shot (if any) and stops the event listener.
// This should be called when the drill completes, typically via defer.
// It's safe to call multiple times.
func (st *ShotTracker) Stop() {
	st.stopOnce.Do(func() {
		st.mu.Lock()

		// Finalize last shot if exists
		if st.hasPreviousShot {
			st.finalizeLocked()
		}

		shotCount := st.shotCount
		st.mu.Unlock()

		// Cancel event listener (don't hold lock during cancel)
		st.cancelFn()

		st.log.Debug("Shot tracker stopped", "total_shots", shotCount)
	})
}

// finalizeLocked finalizes the current shot by analyzing collected events
// and writing the metric. The bounce window is implicitly until now.
// Must be called with st.mu held.
func (st *ShotTracker) finalizeLocked() {
	shotStartTime := st.feedTime
	shotEndTime := time.Now()

	// Analyze events for this shot
	metric := st.analyzeShot(shotStartTime, shotEndTime)

	st.log.Debug("Shot finalized",
		"shot_count", metric.ShotCount,
		"outcome", metric.Outcome,
		"bounce_count", len(metric.Bounces),
		"had_hit", metric.HitTime != nil,
	)

	// Write metric (release lock first to avoid holding it during write)
	st.mu.Unlock()
	if st.writer != nil {
		ctx := context.Background()
		if err := st.writer.WriteShotMetric(ctx, metric); err != nil {
			st.log.Error("Failed to write shot metric",
				"shot_count", metric.ShotCount,
				"error", err,
			)
		}
	}
	st.mu.Lock()

	st.hasPreviousShot = false
}

// analyzeShot processes collected events for the current shot and builds the metric.
// Must be called with st.mu held.
func (st *ShotTracker) analyzeShot(shotStartTime time.Time, shotEndTime time.Time) api.ShotMetric {
	var playerPosition *api.Point
	var hitTime *time.Time
	var bounces []api.BounceRecord
	netDetected := false

	// Process events in the time window
	for _, evt := range st.collectedEvents {
		// Skip events before shot start
		if evt.Timestamp.Before(shotStartTime) {
			continue
		}
		// Skip events after shot end
		if evt.Timestamp.After(shotEndTime) {
			continue
		}

		switch evt.Type {
		case api.EventTypeShot:
			// Capture hit time and player position from shot event
			if hitTime == nil {
				t := evt.Timestamp
				hitTime = &t
			}
			if evt.Position != nil && playerPosition == nil {
				pos := *evt.Position
				playerPosition = &pos
			}

		case api.EventTypeNetCollision:
			netDetected = true

		case api.EventTypeBounce:
			// Collect bounces up to max (hardcoded to 2)
			if len(bounces) < 2 && evt.Position != nil {
				bounces = append(bounces, api.BounceRecord{
					Location:  *evt.Position,
					Timestamp: evt.Timestamp,
				})
			}
		}
	}

	// Resolve outcome
	outcome := st.resolveOutcome(bounces, netDetected)

	return api.ShotMetric{
		ShotCount:      st.shotCount,
		Targets:        st.currentTargets,
		Zones:          st.currentZones,
		FeedTime:       shotStartTime,
		HitTime:        hitTime,
		PlayerPosition: playerPosition,
		Bounces:        bounces,
		Outcome:        outcome,
	}
}

// resolveOutcome determines the shot outcome based on raw event data.
// - OutcomeNet: ball hit the net
// - OutcomeBounce: at least one bounce was detected (raw data for later analysis)
// - OutcomeUnknown: no information about what happened to the ball
// Must be called with st.mu held.
func (st *ShotTracker) resolveOutcome(bounces []api.BounceRecord, netDetected bool) api.ShotOutcome {
	if netDetected {
		return api.OutcomeNet
	}

	if len(bounces) == 0 {
		return api.OutcomeUnknown
	}

	return api.OutcomeBounce
}

// listenEvents runs in a background goroutine and processes events from the subscription.
func (st *ShotTracker) listenEvents() {
	// If no event subscription is configured, just wait for cancellation
	if st.eventSub == nil {
		st.log.Debug("Shot tracker started without event subscription")
		<-st.eventCtx.Done()
		return
	}

	// Subscribe to events
	go func() {
		if err := st.eventSub.Subscribe(st.eventCtx, st.eventCh); err != nil {
			st.log.Error("Shot tracker event subscription failed", "error", err)
		}
	}()

	// Process events
	for {
		select {
		case <-st.eventCtx.Done():
			return
		case event, ok := <-st.eventCh:
			if !ok {
				return
			}
			st.handleEvent(event)
		}
	}
}

// handleEvent collects a single event for later analysis.
func (st *ShotTracker) handleEvent(event api.Event) {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Collect all events for later analysis
	st.collectedEvents = append(st.collectedEvents, event)

	st.log.Debug("Shot tracker: event collected",
		"type", event.Type,
		"timestamp", event.Timestamp,
		"has_position", event.Position != nil,
	)
}
