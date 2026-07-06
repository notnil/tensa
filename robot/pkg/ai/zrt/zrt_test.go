package zrt

import (
	"context"
	"flag"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
)

// inMemoryPubSub is a simple in-memory pub/sub for testing that implements both Pub and Sub.
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

var (
	modelPath = flag.String("model", "yolov8n.onnx", "Path to ONNX model")
	camID     = flag.Int("cam", 0, "ZED Camera ID")
	svoPath   = flag.String("svo", "", "Path to SVO recording")
	width     = flag.Int("width", 1280, "Resolution Width")
	height    = flag.Int("height", 720, "Resolution Height")
	runDemo   = flag.Bool("demo", false, "Run the full demo loop")
)

// TestTrackBalls tests the TrackBalls API with metrics publishing
func TestTrackBalls(t *testing.T) {
	if !*runDemo {
		t.Skip("Skipping demo test. Run with -args -demo to execute.")
	}

	fmt.Printf("Initializing ZRT with model=%s cam=%d svo=%s res=%dx%d\n", *modelPath, *camID, *svoPath, *width, *height)

	opts := []Option{
		WithCamera(*camID),
		WithResolution(*width, *height),
	}
	if *svoPath != "" {
		opts = append(opts, WithSVO(*svoPath))
	}

	zrt, err := New(*modelPath, opts...)
	if err != nil {
		t.Fatalf("Failed to init ZRT: %v", err)
	}
	defer zrt.Close()

	// Create in-memory publisher
	ballPub := newInMemoryPubSub[metrics.Metric[Ball]]()

	// Subscribe to ball metrics
	ballCh := make(chan metrics.Metric[Ball], 100)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		if err := ballPub.Subscribe(ctx, ballCh); err != nil {
			t.Logf("Subscribe error: %v", err)
		}
	}()

	// Start tracking in background
	go func() {
		if err := zrt.TrackBalls(ctx, ballPub); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("TrackBalls error: %v", err)
		}
	}()

	// Collect metrics for a few seconds
	ballCount := 0
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ball := <-ballCh:
			ballCount++
			if ballCount%10 == 0 {
				fmt.Printf("Ball detected: ID=%d, X=%.2f, Y=%.2f, Z=%.2f, Time=%v\n",
					ball.Value.ID, ball.Value.X, ball.Value.Y, ball.Value.Z, ball.Timestamp)
			}
		case <-timeout:
			fmt.Printf("Test complete. Total balls detected: %d\n", ballCount)
			return
		}
	}
}

// TestLocate tests the Locate API with location metrics publishing
func TestLocate(t *testing.T) {
	if !*runDemo {
		t.Skip("Skipping demo test. Run with -args -demo to execute.")
	}

	opts := []Option{
		WithCamera(*camID),
		WithResolution(*width, *height),
	}
	if *svoPath != "" {
		opts = append(opts, WithSVO(*svoPath))
	}

	zrt, err := New(*modelPath, opts...)
	if err != nil {
		t.Fatalf("Failed to init ZRT: %v", err)
	}
	defer zrt.Close()

	// Create in-memory publisher
	locPub := newInMemoryPubSub[metrics.Metric[location.Loc]]()

	// Subscribe to location metrics
	locCh := make(chan metrics.Metric[location.Loc], 100)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		if err := locPub.Subscribe(ctx, locCh); err != nil {
			t.Logf("Subscribe error: %v", err)
		}
	}()

	// Start localization in background
	go func() {
		if err := zrt.Locate(ctx, locPub); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Locate error: %v", err)
		}
	}()

	// Collect metrics
	locCount := 0
	for loc := range locCh {
		locCount++
		if locCount == 1 {
			fmt.Printf("Location: X=%.2f, Y=%.2f, Rotation=%.2f\n",
				loc.Value.Location.X, loc.Value.Location.Y, loc.Value.Rotation)
		}
	}

	fmt.Printf("Total location updates: %d\n", locCount)
}

// BenchmarkTrackBalls benchmarks the TrackBalls API
func BenchmarkTrackBalls(b *testing.B) {
	if *svoPath == "" {
		b.Skip("Skipping benchmark. Provide -svo flag to run benchmarks.")
	}

	opts := []Option{
		WithCamera(*camID),
		WithSVO(*svoPath),
		WithResolution(*width, *height),
	}

	zrt, err := New(*modelPath, opts...)
	if err != nil {
		b.Fatalf("Failed to init ZRT: %v", err)
	}
	defer zrt.Close()

	// Create a publisher that counts metrics
	ballPub := &countingPublisher[metrics.Metric[Ball]]{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tracking in background
	done := make(chan error, 1)
	go func() {
		done <- zrt.TrackBalls(ctx, ballPub)
	}()

	// Warm up
	time.Sleep(500 * time.Millisecond)
	ballPub.Reset()

	b.ResetTimer()
	startTime := time.Now()

	// Let it run for N iterations based on frames
	for i := 0; i < b.N; i++ {
		time.Sleep(33 * time.Millisecond) // ~30 FPS
		if atomic.LoadInt64(&ballPub.count) < int64(i) {
			// Wait for at least one ball per iteration
			time.Sleep(10 * time.Millisecond)
		}
	}

	elapsed := time.Since(startTime)
	cancel()
	<-done

	totalBalls := atomic.LoadInt64(&ballPub.count)
	b.ReportMetric(float64(totalBalls)/float64(b.N), "balls/op")

	fps := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(fps, "fps")

	avgLatency := elapsed.Seconds() * 1000 / float64(b.N)
	b.ReportMetric(avgLatency, "ms/op")
}

// countingPublisher is a simple publisher that counts published metrics
type countingPublisher[T any] struct {
	count int64
}

func (p *countingPublisher[T]) Publish(msg T) error {
	atomic.AddInt64(&p.count, 1)
	return nil
}

func (p *countingPublisher[T]) Reset() {
	atomic.StoreInt64(&p.count, 0)
}
