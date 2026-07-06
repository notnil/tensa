package zrt_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/ai/zrt"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
)

// inMemoryPubSub is a simple in-memory pub/sub for examples that implements both Pub and Sub.
type inMemoryPubSub[T any] struct {
	ps  *pubsubx.InMemoryPub[T]
	sub *pubsubx.InMemorySub[T]
}

func newInMemoryPubSub[T any]() *inMemoryPubSub[T] {
	ps := pubsubx.NewInMemoryPubSub(100)
	topic := "test"
	return &inMemoryPubSub[T]{
		ps:  pubsubx.NewInMemoryPublisher[T](ps, topic),
		sub: pubsubx.NewInMemorySubscriber[T](ps, topic),
	}
}

func (p *inMemoryPubSub[T]) Publish(msg T) error {
	return p.ps.Publish(msg)
}

func (p *inMemoryPubSub[T]) Subscribe(ctx context.Context, ch chan<- T) error {
	return p.sub.Subscribe(ctx, ch)
}

// Example demonstrates the new ZRT API for ball tracking
func Example_trackBalls() {
	// Initialize ZRT with functional options
	z, err := zrt.New(
		"/path/to/model.onnx",
		zrt.WithCamera(0),
		zrt.WithResolution(640, 640),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer z.Close()

	// Create a publisher for ball metrics
	ballPub := newInMemoryPubSub[metrics.Metric[zrt.Ball]]()

	// Subscribe to ball metrics
	ballCh := make(chan metrics.Metric[zrt.Ball], 100)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := ballPub.Subscribe(ctx, ballCh); err != nil {
			log.Printf("Subscribe error: %v", err)
		}
	}()

	// Start ball tracking
	go func() {
		if err := z.TrackBalls(ctx, ballPub); err != nil {
			log.Printf("TrackBalls error: %v", err)
		}
	}()

	// Process ball metrics
	for ball := range ballCh {
		fmt.Printf("Ball detected: ID=%d, Position=(%.2f, %.2f, %.2f)\n",
			ball.Value.ID, ball.Value.X, ball.Value.Y, ball.Value.Z)
	}
}

// Example demonstrates the new ZRT API for localization
func Example_locate() {
	// Initialize ZRT
	z, err := zrt.New(
		"/path/to/model.onnx",
		zrt.WithSVO("/path/to/recording.svo2"),
		zrt.WithResolution(640, 640),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer z.Close()

	// Create a publisher for location metrics
	locPub := newInMemoryPubSub[metrics.Metric[location.Loc]]()

	// Subscribe to location metrics
	locCh := make(chan metrics.Metric[location.Loc], 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := locPub.Subscribe(ctx, locCh); err != nil {
			log.Printf("Subscribe error: %v", err)
		}
	}()

	// Start localization
	go func() {
		if err := z.Locate(ctx, locPub); err != nil {
			log.Printf("Locate error: %v", err)
		}
	}()

	// Process location metrics
	for loc := range locCh {
		fmt.Printf("Location: X=%.2f, Y=%.2f, Rotation=%.2f rad\n",
			loc.Value.Location.X, loc.Value.Location.Y, loc.Value.Rotation)
	}
}

// Example demonstrates running both tracking and localization concurrently
func Example_combined() {
	// Initialize ZRT
	z, err := zrt.New(
		"/path/to/model.onnx",
		zrt.WithCamera(0),
		zrt.WithResolution(640, 640),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer z.Close()

	// Create publishers
	ballPub := newInMemoryPubSub[metrics.Metric[zrt.Ball]]()
	locPub := newInMemoryPubSub[metrics.Metric[location.Loc]]()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start both tracking and localization
	go func() {
		if err := z.TrackBalls(ctx, ballPub); err != nil {
			log.Printf("TrackBalls error: %v", err)
		}
	}()

	go func() {
		if err := z.Locate(ctx, locPub); err != nil {
			log.Printf("Locate error: %v", err)
		}
	}()

	// Wait for context to complete
	<-ctx.Done()
	fmt.Println("Tracking and localization complete")
}
