#include "bridge.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <limits.h>
#include <sys/stat.h>

// In a real environment, these headers must be provided
#include <onnxruntime_c_api.h>
#include <sl/c_api/zed_interface.h> 

// ------------------------------------------------------------------
// MOCK ZED C API Declarations (to compile without ZED SDK headers)
// ------------------------------------------------------------------
#ifndef SL_MAT_TYPE_U8_C4
#define SL_MAT_TYPE_U8_C4 0
#endif

#ifndef SL_MEM_GPU
#define SL_MEM_GPU 1
#endif

#ifndef SL_VIEW_LEFT
#define SL_VIEW_LEFT 0
#endif

#ifndef SL_MEASURE_DEPTH
#define SL_MEASURE_DEPTH 1
#endif

#ifndef SL_MAT_TYPE_F32_C1
#define SL_MAT_TYPE_F32_C1 3
#endif

const OrtApi* g_ort = NULL;

typedef struct {
    int camera_id;
    void* zed_mat_gpu;
    void* zed_depth_gpu;  // Depth map
    int width;
    int height;

    OrtEnv* env;
    OrtSession* session;
    OrtSessionOptions* session_options;
    OrtMemoryInfo* memory_info; // CUDA memory info
    
    char* input_name;
    char* output_name;
    
    int64_t input_shape[4];
} ZrtContext;

ZrtPipeline zrt_init(const char* model_path, int cam_id, const char* svo_path, int width, int height) {
    if (!g_ort) {
        g_ort = OrtGetApiBase()->GetApi(ORT_API_VERSION);
        if (!g_ort) {
            fprintf(stderr, "Error: Failed to get ORT API\n");
            return NULL;
        }
    }

    ZrtContext* ctx = (ZrtContext*)calloc(1, sizeof(ZrtContext));
    if (!ctx) return NULL;

    ctx->camera_id = cam_id;
    ctx->width = width;
    ctx->height = height;

    // 0. Init ZED Camera
    // First, create the camera instance
    if (!sl_create_camera(cam_id)) {
        fprintf(stderr, "Error: Failed to create ZED camera instance (already exists or error)\n");
        // Continue anyway - might already be created
    }

    struct SL_InitParameters init_params;
    // Initialize with defaults
    memset(&init_params, 0, sizeof(init_params));
    
    init_params.camera_fps = 0;
    init_params.resolution = SL_RESOLUTION_HD720;
    init_params.camera_device_id = cam_id;
    init_params.input_type = SL_INPUT_TYPE_USB;

    if (svo_path && strlen(svo_path) > 0) {
        init_params.input_type = SL_INPUT_TYPE_SVO;
        // init_params.svo_real_time_mode = 0; // Default is 1 (real-time)
    }

    // Now open the camera
    int err = sl_open_camera(cam_id, &init_params, 0, svo_path, "", 0, "", "", "");
    if (err != 0) {
        fprintf(stderr, "Error: Failed to open ZED camera (code %d)\n", err);
        free(ctx);
        return NULL;
    }

    // 1. Init ZED Mat (GPU)
    // We assume external setup or linkage to ZED SDK
    ctx->zed_mat_gpu = sl_mat_create_new(width, height, SL_MAT_TYPE_U8_C4, SL_MEM_GPU);
    if (!ctx->zed_mat_gpu) {
        fprintf(stderr, "Error: Failed to create ZED Mat\n");
        // Close camera if mat creation fails
        // sl_close_camera(cam_id); 
        free(ctx);
        return NULL;
    }

    // Also create depth mat
    ctx->zed_depth_gpu = sl_mat_create_new(width, height, SL_MAT_TYPE_F32_C1, SL_MEM_GPU);
    if (!ctx->zed_depth_gpu) {
        fprintf(stderr, "Error: Failed to create ZED Depth Mat\n");
        sl_mat_destroy(ctx->zed_mat_gpu);
        free(ctx);
        return NULL;
    }

    // 2. Init ORT
    if (g_ort->CreateEnv(ORT_LOGGING_LEVEL_WARNING, "zrt_pipeline", &ctx->env) != NULL) {
        fprintf(stderr, "Error: Failed to create ORT Env\n");
        free(ctx); return NULL;
    }

    if (g_ort->CreateSessionOptions(&ctx->session_options) != NULL) {
        free(ctx); return NULL;
    }

    // Enable extended graph optimizations
    g_ort->SetSessionGraphOptimizationLevel(ctx->session_options, ORT_ENABLE_EXTENDED);

    // Create TensorRT cache directory if it doesn't exist
    mkdir("./trt_engine_cache", 0755);

    // Try TensorRT Provider first (must come before CUDA) - Optional
    // Note: TensorRT provides best performance but requires compatible versions
    OrtTensorRTProviderOptionsV2* trt_opts;
    OrtStatus* trt_status = g_ort->CreateTensorRTProviderOptions(&trt_opts);
    if (trt_status == NULL && trt_opts != NULL) {
        // Set TensorRT options
        const char* trt_keys[] = {
            "device_id",
            "trt_fp16_enable",
            "trt_engine_cache_enable",
            "trt_engine_cache_path",
            "trt_max_workspace_size"
        };
        const char* trt_values[] = {
            "0",                              // device_id
            "1",                              // enable FP16
            "1",                              // enable engine caching
            "./trt_engine_cache",            // cache directory
            "2147483648"                      // 2GB workspace
        };

        OrtStatus* update_status = g_ort->UpdateTensorRTProviderOptions(trt_opts, trt_keys, trt_values, 5);
        if (update_status == NULL) {
            OrtStatus* append_status = g_ort->SessionOptionsAppendExecutionProvider_TensorRT_V2(ctx->session_options, trt_opts);
            if (append_status != NULL) {
                printf("Info: TensorRT provider not available, using CUDA with optimizations.\n");
                g_ort->ReleaseStatus(append_status);
            } else {
                printf("TensorRT provider enabled with FP16 precision.\n");
            }
        } else {
            g_ort->ReleaseStatus(update_status);
        }
        g_ort->ReleaseTensorRTProviderOptions(trt_opts);
    } else {
        if (trt_status != NULL) {
            g_ort->ReleaseStatus(trt_status);
        }
        printf("Info: TensorRT not available, using CUDA with graph optimizations.\n");
    }

    // Enable CUDA Provider (fallback if TensorRT fails)
    OrtCUDAProviderOptions cuda_opts;
    memset(&cuda_opts, 0, sizeof(cuda_opts));
    cuda_opts.device_id = 0;
    cuda_opts.cudnn_conv_algo_search = OrtCudnnConvAlgoSearchHeuristic; // Faster than Exhaustive
    cuda_opts.gpu_mem_limit = SIZE_MAX;
    cuda_opts.arena_extend_strategy = 0;
    cuda_opts.do_copy_in_default_stream = 1;
    cuda_opts.has_user_compute_stream = 0;
    cuda_opts.user_compute_stream = NULL;
    cuda_opts.default_memory_arena_cfg = NULL;
    cuda_opts.tunable_op_enable = 1;              // Enable for this specific model
    cuda_opts.tunable_op_tuning_enable = 1;
    cuda_opts.tunable_op_max_tuning_duration_ms = 0;

    if (g_ort->SessionOptionsAppendExecutionProvider_CUDA(ctx->session_options, &cuda_opts) != NULL) {
        fprintf(stderr, "Warning: Failed to append CUDA provider. Using CPU.\n");
    }

    // Create Session
    OrtStatus* status = g_ort->CreateSession(ctx->env, model_path, ctx->session_options, &ctx->session);
    if (status != NULL) {
        const char* msg = g_ort->GetErrorMessage(status);
        fprintf(stderr, "Error: Failed to create ORT Session: %s\n", msg);
        g_ort->ReleaseStatus(status);
        free(ctx); return NULL;
    }

    // Create CUDA Memory Info (Crucial for Zero-Copy input)
    // Allocator type: OrtArenaAllocator
    // Mem type: OrtMemTypeDefault
    if (g_ort->CreateMemoryInfo("Cuda", OrtArenaAllocator, 0, OrtMemTypeDefault, &ctx->memory_info) != NULL) {
        fprintf(stderr, "Error: Failed to create CUDA MemoryInfo. Ensure CUDA provider is linked.\n");
        free(ctx); return NULL;
    }

    // Get Input/Output names
    OrtAllocator* allocator;
    g_ort->GetAllocatorWithDefaultOptions(&allocator);
    g_ort->SessionGetInputName(ctx->session, 0, allocator, &ctx->input_name);
    g_ort->SessionGetOutputName(ctx->session, 0, allocator, &ctx->output_name);

    // Setup Input Shape (BGRA: 1, H, W, 4)
    ctx->input_shape[0] = 1;
    ctx->input_shape[1] = height;
    ctx->input_shape[2] = width;
    ctx->input_shape[3] = 4;

    return (ZrtPipeline)ctx;
}

int zrt_run_inference(ZrtPipeline pipeline, DetectionResult* result) {
    if (!pipeline || !result) return -1;
    ZrtContext* ctx = (ZrtContext*)pipeline;

    // 1. Capture Image (ZED)
    // With SVO, sl_grab might return EOF if reached end of file?
    // We need to provide RuntimeParameters (can't be NULL)
    struct SL_RuntimeParameters runtime_params;
    memset(&runtime_params, 0, sizeof(runtime_params));
    runtime_params.enable_depth = 1; // Enable depth for ball tracking
    runtime_params.confidence_threshold = 100;
    runtime_params.texture_confidence_threshold = 100;
    runtime_params.reference_frame = SL_REFERENCE_FRAME_CAMERA;
    runtime_params.remove_saturated_areas = 1;
    runtime_params.enable_fill_mode = 0;
    
    int grab_err = sl_grab(ctx->camera_id, &runtime_params);
    if (grab_err != 0) {
        // EOF or Error. Return special code for EOF? 
        // Typically 0 is success.
        return -2;
    }

    // 2. Retrieve Image to GPU memory
    // NOTE: We pass NULL as custream to use default stream.
    if (sl_retrieve_image(ctx->camera_id, ctx->zed_mat_gpu, SL_VIEW_LEFT, SL_MEM_GPU, 0, 0, NULL) != 0) {
         fprintf(stderr, "Error: ZED retrieve failed\n");
         return -2;
    }

    // 2b. Retrieve Depth to GPU memory
    if (sl_retrieve_measure(ctx->camera_id, ctx->zed_depth_gpu, SL_MEASURE_DEPTH, SL_MEM_GPU, 0, 0, NULL) != 0) {
         fprintf(stderr, "Error: ZED depth retrieve failed\n");
         return -2;
    }
    
    // 3. Get GPU Pointer from ZED Mat
    // The ZED C API returns int* which is actually the device pointer.
    void* zed_gpu_ptr = (void*)sl_mat_get_ptr(ctx->zed_mat_gpu, SL_MEM_GPU);
    if (!zed_gpu_ptr) {
        fprintf(stderr, "Error: Failed to get ZED GPU pointer\n");
        return -3;
    }

    // 4. Create Input Tensor (Zero Copy)
    // We wrap the ZED CUDA pointer directly.
    OrtValue* input_tensor = NULL;
    size_t input_data_size = ctx->width * ctx->height * 4 * sizeof(uint8_t);
    
    OrtStatus* status = g_ort->CreateTensorWithDataAsOrtValue(
        ctx->memory_info,
        zed_gpu_ptr,
        input_data_size,
        ctx->input_shape,
        4,
        ONNX_TENSOR_ELEMENT_DATA_TYPE_UINT8,
        &input_tensor
    );

    if (status != NULL) {
        fprintf(stderr, "Error: Failed to wrap ZED pointer in ORT Tensor\n");
        return -4;
    }

    // 5. Run Inference
    const char* input_names[] = { ctx->input_name };
    const char* output_names[] = { ctx->output_name };
    OrtValue* output_tensor = NULL;

    status = g_ort->Run(
        ctx->session,
        NULL,
        input_names,
        (const OrtValue* const*)&input_tensor,
        1,
        output_names,
        1,
        &output_tensor
    );

    g_ort->ReleaseValue(input_tensor); // Does not free the underlying ZED pointer

    if (status != NULL) {
        const char* msg = g_ort->GetErrorMessage(status);
        fprintf(stderr, "Error: Inference run failed: %s\n", msg);
        g_ort->ReleaseStatus(status);
        return -5;
    }

    // 6. Process Results
    // We only count detections if format is [N, 6]
    // Otherwise (raw benchmark) we ignore details but complete execution
    struct OrtTensorTypeAndShapeInfo* info;
    g_ort->GetTensorTypeAndShape(output_tensor, &info);
    
    // Assume CPU accessible for now (or Unified Memory on Jetson)
    float* out_data;
    g_ort->GetTensorMutableData(output_tensor, (void**)&out_data);

    size_t dim_count;
    g_ort->GetDimensionsCount(info, &dim_count);
    int64_t* dims = (int64_t*)malloc(sizeof(int64_t) * dim_count);
    g_ort->GetDimensions(info, dims, dim_count);

    int num_boxes = 0;
    // Check if output is [1, 84, 8400] (Raw YOLO) -> Skip parsing
    if (dim_count == 3 && dims[1] == 84 && dims[2] == 8400) {
        // Raw output, just return empty detection list but success status
        num_boxes = 0;
    } else {
        // NMS output [1, N, 6]
        if (dim_count >= 2) num_boxes = (int)dims[dim_count - 2]; 
        if (dim_count == 2) num_boxes = (int)dims[0];
        if (dim_count == 3) num_boxes = (int)dims[1];
    }
    
    result->count = num_boxes;
    result->detections = (Detection*)calloc(num_boxes, sizeof(Detection));
    
    for(int i=0; i<num_boxes; i++) {
        int b = i * 6; // Stride 6
        result->detections[i].x = out_data[b+0];
        result->detections[i].y = out_data[b+1];
        result->detections[i].w = out_data[b+2];
        result->detections[i].h = out_data[b+3];
        result->detections[i].confidence = out_data[b+4];
        result->detections[i].class_id = (int)out_data[b+5];
        
        // Get depth at center of detection
        int center_x = (int)(result->detections[i].x + result->detections[i].w / 2.0f);
        int center_y = (int)(result->detections[i].y + result->detections[i].h / 2.0f);
        result->detections[i].depth = zrt_get_depth_at((ZrtPipeline)ctx, center_x, center_y);
    }

    free(dims);
    g_ort->ReleaseTensorTypeAndShapeInfo(info);
    g_ort->ReleaseValue(output_tensor);

    return 0;
}

int zrt_run_inference_cached(ZrtPipeline pipeline, DetectionResult* result) {
    if (!pipeline || !result) return -1;
    ZrtContext* ctx = (ZrtContext*)pipeline;
    
    // Skip sl_grab and sl_retrieve_image, assuming GPU memory already has data
    // from previous zrt_run_inference call.
    
    // 3. Get GPU Pointer from ZED Mat (Reuse)
    void* zed_gpu_ptr = (void*)sl_mat_get_ptr(ctx->zed_mat_gpu, SL_MEM_GPU);
    if (!zed_gpu_ptr) {
        fprintf(stderr, "Error: Failed to get ZED GPU pointer\n");
        return -3;
    }

    // 4. Create Input Tensor (Zero Copy)
    OrtValue* input_tensor = NULL;
    size_t input_data_size = ctx->width * ctx->height * 4 * sizeof(uint8_t);
    
    OrtStatus* status = g_ort->CreateTensorWithDataAsOrtValue(
        ctx->memory_info,
        zed_gpu_ptr,
        input_data_size,
        ctx->input_shape,
        4,
        ONNX_TENSOR_ELEMENT_DATA_TYPE_UINT8,
        &input_tensor
    );

    if (status != NULL) {
        fprintf(stderr, "Error: Failed to wrap ZED pointer in ORT Tensor\n");
        return -4;
    }

    // 5. Run Inference
    const char* input_names[] = { ctx->input_name };
    const char* output_names[] = { ctx->output_name };
    OrtValue* output_tensor = NULL;

    status = g_ort->Run(
        ctx->session,
        NULL,
        input_names,
        (const OrtValue* const*)&input_tensor,
        1,
        output_names,
        1,
        &output_tensor
    );

    g_ort->ReleaseValue(input_tensor);

    if (status != NULL) {
        const char* msg = g_ort->GetErrorMessage(status);
        fprintf(stderr, "Error: Inference run failed: %s\n", msg);
        g_ort->ReleaseStatus(status);
        return -5;
    }

    // 6. Process Results
    struct OrtTensorTypeAndShapeInfo* info;
    g_ort->GetTensorTypeAndShape(output_tensor, &info);
    
    float* out_data;
    g_ort->GetTensorMutableData(output_tensor, (void**)&out_data);

    size_t dim_count;
    g_ort->GetDimensionsCount(info, &dim_count);
    int64_t* dims = (int64_t*)malloc(sizeof(int64_t) * dim_count);
    g_ort->GetDimensions(info, dims, dim_count);

    int num_boxes = 0;
    // Raw output [1, 84, 8400] check
    if (dim_count == 3 && dims[1] == 84 && dims[2] == 8400) {
        num_boxes = 0;
    } else {
        if (dim_count >= 2) {
            num_boxes = (int)dims[dim_count - 2]; 
        }
        if (dim_count == 2) num_boxes = (int)dims[0];
        if (dim_count == 3) num_boxes = (int)dims[1];
    }
    
    result->count = num_boxes;
    result->detections = (Detection*)calloc(num_boxes, sizeof(Detection));
    
    for(int i=0; i<num_boxes; i++) {
        int b = i * 6; 
        result->detections[i].x = out_data[b+0];
        result->detections[i].y = out_data[b+1];
        result->detections[i].w = out_data[b+2];
        result->detections[i].h = out_data[b+3];
        result->detections[i].confidence = out_data[b+4];
        result->detections[i].class_id = (int)out_data[b+5];
        
        // Get depth at center of detection (uses cached depth map)
        int center_x = (int)(result->detections[i].x + result->detections[i].w / 2.0f);
        int center_y = (int)(result->detections[i].y + result->detections[i].h / 2.0f);
        result->detections[i].depth = zrt_get_depth_at((ZrtPipeline)ctx, center_x, center_y);
    }

    free(dims);
    g_ort->ReleaseTensorTypeAndShapeInfo(info);
    g_ort->ReleaseValue(output_tensor);

    return 0;
}

float zrt_get_depth_at(ZrtPipeline pipeline, int x, int y) {
    if (!pipeline) return -1.0f;
    ZrtContext* ctx = (ZrtContext*)pipeline;
    
    // Bounds check
    if (x < 0 || x >= ctx->width || y < 0 || y >= ctx->height) {
        return -1.0f;
    }
    
    // Get pointer to depth data (on GPU, but can access via unified memory or copy)
    // For simplicity, we use sl_mat_get_value which handles CPU access
    float depth_value = -1.0f;
    int err = sl_mat_get_value_float(ctx->zed_depth_gpu, x, y, &depth_value, SL_MEM_GPU);
    
    if (err != 0 || depth_value <= 0.0f || !isfinite(depth_value)) {
        return -1.0f;  // Invalid depth
    }
    
    return depth_value;
}

void zrt_free_result(DetectionResult* result) {
    if (result && result->detections) {
        free(result->detections);
        result->detections = NULL;
        result->count = 0;
    }
}

void zrt_close(ZrtPipeline pipeline) {
    if (!pipeline) return;
    ZrtContext* ctx = (ZrtContext*)pipeline;
    
    sl_close_camera(ctx->camera_id);

    if (ctx->zed_mat_gpu) sl_mat_destroy(ctx->zed_mat_gpu);
    if (ctx->zed_depth_gpu) sl_mat_destroy(ctx->zed_depth_gpu);

    if (ctx->input_name) free(ctx->input_name); // Allocator free?
    if (ctx->output_name) free(ctx->output_name);
    
    if (ctx->session) g_ort->ReleaseSession(ctx->session);
    if (ctx->session_options) g_ort->ReleaseSessionOptions(ctx->session_options);
    if (ctx->env) g_ort->ReleaseEnv(ctx->env);
    if (ctx->memory_info) g_ort->ReleaseMemoryInfo(ctx->memory_info);
    
    free(ctx);
}