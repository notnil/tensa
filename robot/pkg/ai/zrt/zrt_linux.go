//go:build linux

package zrt

/*
#cgo CFLAGS: -I${SRCDIR}/clib
#cgo LDFLAGS: -lonnxruntime -lsl_zed_c
#include "bridge.h"
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"unsafe"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
	"github.com/notnil/tensa/pkg/tennis/court2d"
)

// ZRT manages the Zero-Copy inference pipeline for ball tracking and localization.
type ZRT struct {
	ptr        C.ZrtPipeline
	mu         sync.Mutex
	nextBallID int
	width      int
	height     int
}

// New initializes the ZED + ORT pipeline with the given model and options.
// modelPath: Path to the optimized ONNX model.
func New(modelPath string, opts ...Option) (*ZRT, error) {
	cfg := &config{
		camID:  0,
		width:  640,
		height: 640,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cPath))

	var cSvoPath *C.char
	if cfg.svoPath != "" {
		cSvoPath = C.CString(cfg.svoPath)
		defer C.free(unsafe.Pointer(cSvoPath))
	}

	ptr := C.zrt_init(cPath, C.int(cfg.camID), cSvoPath, C.int(cfg.width), C.int(cfg.height))
	if ptr == nil {
		return nil, fmt.Errorf("failed to initialize ZRT pipeline: check logs for details")
	}

	return &ZRT{
		ptr:    ptr,
		width:  cfg.width,
		height: cfg.height,
	}, nil
}

// TrackBalls continuously tracks tennis balls and publishes Ball metrics.
// Runs until the context is cancelled.
func (z *ZRT) TrackBalls(ctx context.Context, pub pubsubx.Pub[metrics.Metric[Ball]]) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Run inference
			detections, err := z.runInference()
			if err != nil {
				slog.Error("inference failed", "error", err)
				continue
			}

			// Convert detections to balls with 3D coordinates
			for _, det := range detections {
				z.mu.Lock()
				ballID := z.nextBallID
				z.nextBallID++
				z.mu.Unlock()

				// Convert 2D detection + depth to 3D coordinates
				x, y, depth := float64(det.X), float64(det.Y), float64(det.Depth)
				ballX, ballY, ballZ := pixelTo3D(x, y, depth, z.width, z.height)

				ball := Ball{
					ID: ballID,
					X:  ballX,
					Y:  ballY,
					Z:  ballZ,
				}

				// Publish the ball metric
				metric := metrics.NewMetric(ball)
				if err := pub.Publish(metric); err != nil {
					slog.Error("failed to publish ball metric", "error", err)
				}
			}
		}
	}
}

// Locate continuously determines the robot's location and publishes location metrics.
// Runs until the context is cancelled.
// This is a placeholder implementation - future enhancements could use visual odometry,
// AprilTag detection, or integration with other sensors.
func (z *ZRT) Locate(ctx context.Context, pub pubsubx.Pub[metrics.Metric[location.Loc]]) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// TODO: Implement actual localization using visual odometry or markers
			// For now, publish a placeholder location at origin
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

// Close frees the pipeline resources.
func (z *ZRT) Close() {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.ptr != nil {
		C.zrt_close(z.ptr)
		z.ptr = nil
	}
}

// runInference captures a frame from ZED and runs inference.
// Returns a slice of detections with depth information.
func (z *ZRT) runInference() ([]detection, error) {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.ptr == nil {
		return nil, fmt.Errorf("pipeline is closed")
	}

	var result C.DetectionResult

	// The bridge handles ZED retrieve -> GPU Pointer -> ORT Tensor -> Run
	ret := C.zrt_run_inference(z.ptr, &result)
	if ret != 0 {
		return nil, fmt.Errorf("inference failed with code %d", ret)
	}
	defer C.zrt_free_result(&result)

	count := int(result.count)
	if count == 0 {
		return []detection{}, nil
	}

	dets := make([]detection, count)

	// Access the C array safely
	// 1<<30 is a standard trick to cast pointer to large array
	cDets := (*[1 << 30]C.Detection)(unsafe.Pointer(result.detections))[:count:count]

	for i := 0; i < count; i++ {
		d := cDets[i]
		dets[i] = detection{
			X:          float32(d.x),
			Y:          float32(d.y),
			W:          float32(d.w),
			H:          float32(d.h),
			Confidence: float32(d.confidence),
			ClassID:    int(d.class_id),
			Depth:      float32(d.depth),
		}
	}

	return dets, nil
}
