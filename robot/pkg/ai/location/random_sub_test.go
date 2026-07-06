package location

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/numeric"
)

func TestRandomWalkSub(t *testing.T) {
	// Define a simple rectangular polygon for testing.
	polygon := court2d.DoublesCourt.Rectangle()

	cfg := RandomWalkConfig{
		Polygon:         polygon,
		Interval:        10 * time.Millisecond,
		Seed:            time.Now().UnixNano(),
		PositionDelta:   numeric.Range[float64]{Min: 0.1, Max: 0.5},
		RotationDelta:   numeric.Range[float64]{Min: 0.01, Max: 0.1},
		InitialPoint:    nil, // Let the constructor pick a random initial point
		InitialRotation: nil, // Let the constructor pick a random initial rotation
	}

	sub := NewRandomWalkSub(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan metrics.Metric[Loc])
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := sub.Subscribe(ctx, ch)
		if err != context.Canceled {
			t.Errorf("Subscribe exited with unexpected error: %v", err)
		}
	}()

	// Wait for a few messages to be generated
	numMessages := 0
	for i := 0; i < 5; i++ {
		select {
		case <-ch:
			numMessages++
		case <-time.After(1 * time.Second):
			t.Fatal("timed out waiting for messages")
		}
	}

	if numMessages == 0 {
		t.Error("Expected to receive at least one message, got 0")
	}

	// Cancel the context to stop the subscriber
	cancel()
	wg.Wait() // Wait for the goroutine to finish
}
