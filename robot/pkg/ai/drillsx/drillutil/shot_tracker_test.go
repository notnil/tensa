package drillutil

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

// mockEventSub is a mock implementation of EventSub for testing.
type mockEventSub struct {
	events []api.Event
	mu     sync.Mutex
}

func newMockEventSub() *mockEventSub {
	return &mockEventSub{
		events: make([]api.Event, 0),
	}
}

func (m *mockEventSub) Subscribe(ctx context.Context, ch chan<- api.Event) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			m.mu.Lock()
			if len(m.events) > 0 {
				evt := m.events[0]
				m.events = m.events[1:]
				m.mu.Unlock()
				ch <- evt
			} else {
				m.mu.Unlock()
			}
		}
	}
}

func (m *mockEventSub) PublishEvent(evt api.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, evt)
}

// mockWriter collects written metrics for verification.
type mockWriter struct {
	metrics []api.ShotMetric
	mu      sync.Mutex
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		metrics: make([]api.ShotMetric, 0),
	}
}

func (w *mockWriter) WriteShotMetric(ctx context.Context, metric api.ShotMetric) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.metrics = append(w.metrics, metric)
	return nil
}

func (w *mockWriter) GetMetrics() []api.ShotMetric {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]api.ShotMetric(nil), w.metrics...)
}

func TestShotTracker_OutcomeBounce(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	// Define a target zone
	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	target := api.Point{X: 1, Y: 1}

	// Start tracking a shot
	tracker.Feed([]api.Point{target}, []api.Polygon{zone})

	// Simulate events: shot and bounce
	now := time.Now()
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeShot,
		Timestamp: now,
		Position:  &api.Point{X: 0.5, Y: 0.5},
	})
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeBounce,
		Timestamp: now.Add(10 * time.Millisecond),
		Position:  &api.Point{X: 1, Y: 1},
	})

	// Wait for events to be processed (mock polls every 10ms)
	time.Sleep(200 * time.Millisecond)

	// Finalize by calling Stop
	tracker.Stop()

	// Verify metric
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]
	if metric.Outcome != api.OutcomeBounce {
		t.Errorf("Expected outcome Bounce, got %v", metric.Outcome)
	}
	if metric.ShotCount != 1 {
		t.Errorf("Expected shot count 1, got %d", metric.ShotCount)
	}
	if len(metric.Bounces) != 1 {
		t.Errorf("Expected 1 bounce, got %d", len(metric.Bounces))
	}
	if metric.HitTime == nil {
		t.Error("Expected hit time to be recorded")
	}
	if metric.PlayerPosition == nil {
		t.Error("Expected player position to be recorded")
	}
}

func TestShotTracker_OutcomeNet(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	target := api.Point{X: 1, Y: 1}

	// Start tracking a shot
	tracker.Feed([]api.Point{target}, []api.Polygon{zone})

	// Simulate net collision
	now := time.Now()
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeShot,
		Timestamp: now,
		Position:  &api.Point{X: 0.5, Y: 0.5},
	})
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeNetCollision,
		Timestamp: now.Add(10 * time.Millisecond),
	})

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Finalize
	tracker.Stop()

	// Verify metric
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]
	if metric.Outcome != api.OutcomeNet {
		t.Errorf("Expected outcome Net, got %v", metric.Outcome)
	}
}

func TestShotTracker_OutcomeUnknown(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	target := api.Point{X: 1, Y: 1}

	// Start tracking a shot with no bounces
	tracker.Feed([]api.Point{target}, []api.Polygon{zone})

	// Simulate shot but no bounces
	now := time.Now()
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeShot,
		Timestamp: now,
		Position:  &api.Point{X: 0.5, Y: 0.5},
	})

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Finalize
	tracker.Stop()

	// Verify metric
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]
	if metric.Outcome != api.OutcomeUnknown {
		t.Errorf("Expected outcome Unknown, got %v", metric.Outcome)
	}
}

func TestShotTracker_MultipleShots(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	// Shot 1
	tracker.Feed([]api.Point{{X: 1, Y: 1}}, []api.Polygon{zone})
	now := time.Now()
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeBounce,
		Timestamp: now,
		Position:  &api.Point{X: 1, Y: 1},
	})
	time.Sleep(100 * time.Millisecond)

	// Shot 2 (this finalizes shot 1)
	tracker.Feed([]api.Point{{X: 1.5, Y: 1.5}}, []api.Polygon{zone})
	eventSub.PublishEvent(api.Event{
		Type:      api.EventTypeBounce,
		Timestamp: time.Now(),
		Position:  &api.Point{X: 5, Y: 5}, // Outside
	})
	time.Sleep(100 * time.Millisecond)

	// Shot 3 (this finalizes shot 2)
	tracker.Feed([]api.Point{{X: 1.2, Y: 1.2}}, []api.Polygon{zone})
	time.Sleep(100 * time.Millisecond)

	// Stop finalizes shot 3
	tracker.Stop()

	// Verify metrics
	metrics := writer.GetMetrics()
	if len(metrics) != 3 {
		t.Fatalf("Expected 3 metrics, got %d", len(metrics))
	}

	// Check shot counts
	for i, metric := range metrics {
		expectedCount := i + 1
		if metric.ShotCount != expectedCount {
			t.Errorf("Shot %d: expected count %d, got %d", i, expectedCount, metric.ShotCount)
		}
	}

	// Check outcomes
	if metrics[0].Outcome != api.OutcomeBounce {
		t.Errorf("Shot 1: expected Bounce, got %v", metrics[0].Outcome)
	}
	if metrics[1].Outcome != api.OutcomeBounce {
		t.Errorf("Shot 2: expected Bounce, got %v", metrics[1].Outcome)
	}
	if metrics[2].Outcome != api.OutcomeUnknown {
		t.Errorf("Shot 3: expected Unknown, got %v", metrics[2].Outcome)
	}
}

func TestShotTracker_MaxBounces(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 10, Y: 0},
			{X: 10, Y: 10},
			{X: 0, Y: 10},
		},
	}

	// Start tracking
	tracker.Feed([]api.Point{{X: 5, Y: 5}}, []api.Polygon{zone})

	// Simulate 4 bounces (should only record 2)
	now := time.Now()
	for i := 0; i < 4; i++ {
		eventSub.PublishEvent(api.Event{
			Type:      api.EventTypeBounce,
			Timestamp: now.Add(time.Duration(i*10) * time.Millisecond),
			Position:  &api.Point{X: float64(i), Y: float64(i)},
		})
	}

	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)

	// Finalize
	tracker.Stop()

	// Verify only 2 bounces recorded
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	if len(metrics[0].Bounces) != 2 {
		t.Errorf("Expected 2 bounces (max), got %d", len(metrics[0].Bounces))
	}
}

func TestShotTracker_EventTimeWindowFiltering(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)

	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	// Publish event before shot starts (should be ignored)
	pastEvent := api.Event{
		Type:      api.EventTypeBounce,
		Timestamp: time.Now().Add(-1 * time.Hour),
		Position:  &api.Point{X: 1, Y: 1},
	}
	eventSub.PublishEvent(pastEvent)
	time.Sleep(50 * time.Millisecond)

	// Start tracking
	tracker.Feed([]api.Point{{X: 1, Y: 1}}, []api.Polygon{zone})

	// Publish event within shot window
	nowEvent := api.Event{
		Type:      api.EventTypeBounce,
		Timestamp: time.Now(),
		Position:  &api.Point{X: 1, Y: 1},
	}
	eventSub.PublishEvent(nowEvent)
	time.Sleep(200 * time.Millisecond)

	// Finalize
	tracker.Stop()

	// Verify only the current event was captured
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	if len(metrics[0].Bounces) != 1 {
		t.Errorf("Expected 1 bounce, got %d (past events should be filtered)", len(metrics[0].Bounces))
	}
}

func TestShotTracker_TargetsAndZonesPreserved(t *testing.T) {
	eventSub := newMockEventSub()
	writer := newMockWriter()
	log := slog.Default()

	tracker := NewShotTracker(eventSub, writer, log)
	defer tracker.Stop()

	targets := []api.Point{
		{X: 1, Y: 1},
		{X: 2, Y: 2},
	}

	zones := []api.Polygon{
		{
			Vertices: []api.Point{
				{X: 0, Y: 0},
				{X: 3, Y: 0},
				{X: 3, Y: 3},
				{X: 0, Y: 3},
			},
		},
	}

	// Start tracking with specific targets and zones
	tracker.Feed(targets, zones)
	time.Sleep(50 * time.Millisecond)

	// Finalize
	tracker.Stop()

	// Verify targets and zones are preserved
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]
	if len(metric.Targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(metric.Targets))
	}
	if len(metric.Zones) != 1 {
		t.Errorf("Expected 1 zone, got %d", len(metric.Zones))
	}

	// Verify target values
	if metric.Targets[0].X != 1 || metric.Targets[0].Y != 1 {
		t.Errorf("Target 0 mismatch: got %v", metric.Targets[0])
	}
	if metric.Targets[1].X != 2 || metric.Targets[1].Y != 2 {
		t.Errorf("Target 1 mismatch: got %v", metric.Targets[1])
	}
}

// TestShotTracker_NilEventSub verifies that the tracker handles nil eventSub gracefully.
func TestShotTracker_NilEventSub(t *testing.T) {
	writer := newMockWriter()
	log := slog.Default()

	// Create tracker with nil eventSub (simulates simulator environment)
	tracker := NewShotTracker(nil, writer, log)
	defer tracker.Stop()

	// Define a target zone
	zone := api.Polygon{
		Vertices: []api.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}
	target := api.Point{X: 1, Y: 1}

	// Start tracking a shot
	tracker.Feed([]api.Point{target}, []api.Polygon{zone})

	// Wait a bit to ensure no panic occurs
	time.Sleep(100 * time.Millisecond)

	// Start a second shot to finalize the first
	tracker.Feed([]api.Point{target}, []api.Polygon{zone})

	// Wait for metric to be written
	time.Sleep(100 * time.Millisecond)

	// Verify metric was written
	metrics := writer.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]

	// With no event subscription, outcome should be unknown
	if metric.Outcome != api.OutcomeUnknown {
		t.Errorf("Expected outcome %s, got %s", api.OutcomeUnknown, metric.Outcome)
	}

	// Should have no bounces without events
	if len(metric.Bounces) != 0 {
		t.Errorf("Expected 0 bounces, got %d", len(metric.Bounces))
	}

	// Should have correct shot count
	if metric.ShotCount != 1 {
		t.Errorf("Expected shot count 1, got %d", metric.ShotCount)
	}

	// Should have correct targets and zones
	if len(metric.Targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(metric.Targets))
	}
	if len(metric.Zones) != 1 {
		t.Errorf("Expected 1 zone, got %d", len(metric.Zones))
	}
}
