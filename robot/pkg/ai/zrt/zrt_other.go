//go:build !linux

package zrt

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
	"github.com/notnil/tensa/pkg/tennis/court2d"
)

// ZRT manages the Zero-Copy inference pipeline for ball tracking and localization.
// This is a mock implementation for non-Linux platforms.
type ZRT struct {
	mu         sync.Mutex
	nextBallID int
	width      int
	height     int
	closed     bool
}

// New initializes a mock ZRT pipeline.
// modelPath: Path to the optimized ONNX model (ignored in mock).
func New(modelPath string, opts ...Option) (*ZRT, error) {
	cfg := &config{
		camID:  0,
		width:  640,
		height: 640,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	slog.Info("zrt: using mock implementation (non-Linux platform)",
		"model", modelPath,
		"width", cfg.width,
		"height", cfg.height,
	)

	return &ZRT{
		width:  cfg.width,
		height: cfg.height,
	}, nil
}

// TrackBalls continuously publishes simulated Ball metrics.
// Runs until the context is cancelled.
func (z *ZRT) TrackBalls(ctx context.Context, pub pubsubx.Pub[metrics.Metric[Ball]]) error {
	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			z.mu.Lock()
			if z.closed {
				z.mu.Unlock()
				return nil
			}

			// Simulate occasional ball detections (~10% of frames)
			if rand.Float32() < 0.1 {
				ballID := z.nextBallID
				z.nextBallID++
				z.mu.Unlock()

				// Generate random ball position within reasonable bounds
				ball := Ball{
					ID: ballID,
					X:  (rand.Float64() - 0.5) * 4.0, // -2 to 2 meters
					Y:  rand.Float64() * 2.0,         // 0 to 2 meters height
					Z:  2.0 + rand.Float64()*8.0,     // 2 to 10 meters depth
				}

				metric := metrics.NewMetric(ball)
				if err := pub.Publish(metric); err != nil {
					slog.Error("failed to publish ball metric", "error", err)
				}
			} else {
				z.mu.Unlock()
			}
		}
	}
}

// Locate continuously publishes simulated location metrics.
// Runs until the context is cancelled.
func (z *ZRT) Locate(ctx context.Context, pub pubsubx.Pub[metrics.Metric[location.Loc]]) error {
	ticker := time.NewTicker(100 * time.Millisecond) // 10 Hz
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			z.mu.Lock()
			if z.closed {
				z.mu.Unlock()
				return nil
			}
			z.mu.Unlock()

			// Publish a simulated location at origin
			loc := location.Loc{
				Location: court2d.Point{X: 0, Y: 0},
				Rotation: 0,
			}

			metric := metrics.NewMetric(loc)
			if err := pub.Publish(metric); err != nil {
				slog.Error("failed to publish location metric", "error", err)
			}
		}
	}
}

// Close marks the mock pipeline as closed.
func (z *ZRT) Close() {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.closed = true
}
