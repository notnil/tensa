#ifndef BRIDGE_H
#define BRIDGE_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void* ZrtPipeline;

typedef struct {
    float x;
    float y;
    float w;
    float h;
    float confidence;
    int class_id;
    float depth;  // Depth at center of detection (in meters)
} Detection;

typedef struct {
    Detection* detections;
    int count;
} DetectionResult;

/**
 * Initialize the pipeline.
 * 
 * @param model_path Path to the optimized ONNX model.
 * @param cam_id ZED Camera ID (usually 0). Ignored if svo_path is set.
 * @param svo_path Path to SVO file (optional, pass NULL for live camera).
 * @param width Image width (must match model expectation if no resize).
 * @param height Image height.
 * @return ZrtPipeline handle or NULL on failure.
 */
ZrtPipeline zrt_init(const char* model_path, int cam_id, const char* svo_path, int width, int height);

/**
 * Run inference on the next frame from ZED.
 * 
 * @param pipeline The pipeline handle.
 * @param result Pointer to DetectionResult to populate.
 * @return 0 on success, non-zero on error.
 */
int zrt_run_inference(ZrtPipeline pipeline, DetectionResult* result);

/**
 * Run inference using the last captured frame (cached in GPU memory).
 * Useful for benchmarking pure inference throughput.
 */
int zrt_run_inference_cached(ZrtPipeline pipeline, DetectionResult* result);

/**
 * Free the result structure contents.
 */
void zrt_free_result(DetectionResult* result);

/**
 * Close the pipeline and free resources.
 */
void zrt_close(ZrtPipeline pipeline);

/**
 * Get depth value at specific pixel coordinates.
 * 
 * @param pipeline The pipeline handle.
 * @param x Pixel x coordinate.
 * @param y Pixel y coordinate.
 * @return Depth in meters, or -1.0 on error/invalid depth.
 */
float zrt_get_depth_at(ZrtPipeline pipeline, int x, int y);

#ifdef __cplusplus
}
#endif

#endif // BRIDGE_H
