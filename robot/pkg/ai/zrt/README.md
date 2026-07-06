# Zero-Copy Inference Pipeline (ZRT)

This package implements a high-performance inference pipeline for NVIDIA GPUs, leveraging the ZED SDK and ONNX Runtime with Zero-Copy memory management.

## Quick Start

After completing the installation steps below, run a benchmark:

```bash
cd /path/to/tensa

# Set up environment
export CGO_CFLAGS="-I$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/include -I$(pwd)/../zed-c-api/include -I/usr/local/cuda/include -I/usr/local/zed/include"
export CGO_LDFLAGS="-L$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib -L$(pwd)/../zed-c-api/build -L/usr/local/cuda/lib64 -L/usr/local/zed/lib -Wl,-rpath,$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib -Wl,-rpath,$(pwd)/../zed-c-api/build -Wl,-rpath,/usr/local/zed/lib"
export LD_LIBRARY_PATH="$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib:$(pwd)/../zed-c-api/build:/usr/local/cuda/lib64:/usr/local/zed/lib:/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH"

# Run benchmark
go test -bench=BenchmarkInferenceWithMetrics -benchtime=100x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model $(pwd)/yolov8n_optimized_tensorrt.onnx \
    -svo /path/to/recording.svo2 \
    -width 640 -height 640
```

## Architecture

1.  **ZED SDK**: Captures images directly to GPU memory (`SL_MEM_GPU`) from live camera or SVO recordings.
2.  **Zero-Copy Bridge**: Wraps the raw CUDA pointer from ZED into an ONNX Runtime Tensor without CPU copying.
3.  **ONNX Runtime (CUDA/TensorRT)**: Executes the model on GPU. Preprocessing (normalization, channel swap) is handled within the model graph via Graph Surgery.
4.  **Go**: Orchestrates the pipeline and consumes the final detection results.

## Performance

### Measured Performance (NVIDIA L4 GPU, 640x640 input, CUDA 12.6)

**With CUDA Provider + Graph Optimizations:**
- **23.85 FPS** sustained throughput
- **41.93 ms** average latency per frame
- **84 detections** per frame (YOLOv8n)
- **2.7 KB** memory per operation
- **2 allocations** per operation
- Zero CPU-GPU memory copies

**With TensorRT FP16 (Verified):**
- **~660 FPS** Pure Inference throughput (Zero-Copy, No I/O)
- **~1.5 ms** latency per frame (Compute only)
- **~33 FPS** End-to-End (limited by SVO playback speed / real-time simulation)
- **First run**: 30-90s (TensorRT engine building and caching)
- **Subsequent runs**: Instant startup with cached engines

**Optimizations Active:**
- Extended graph optimization level
- CUDA kernel auto-tuning (improves over time)
- Zero-copy GPU memory pipeline
- Optimized cuDNN convolution algorithms
- TensorRT Execution Provider (FP16, Graph Optimization)
- EfficientNMS Plugin (On-Graph Post-Processing)

## Prerequisites

**Important**: This pipeline requires **CUDA 12.6** for optimal compatibility with pre-built ONNX Runtime binaries. Ensure you download the CUDA 12 version of the ZED SDK.

### 1. Install ZED SDK

Download and install ZED SDK 5.1+ with **CUDA 12 support** from [stereolabs.com](https://www.stereolabs.com/developers/):
- For Ubuntu 24: [ZED SDK 5.1 CUDA 12 - Ubuntu 24](https://download.stereolabs.com/zedsdk/5.1/cu12/ubuntu24)
- For Ubuntu 22: [ZED SDK 5.1 CUDA 12 - Ubuntu 22](https://download.stereolabs.com/zedsdk/5.1/cu12/ubuntu22)

**Install silently:**
```bash
chmod +x ZED_SDK_*.run
sudo ./ZED_SDK_*.run -- silent skip_tools skip_cuda
```

### 2. Install ZED C API

The standard ZED SDK is C++. You must build and install the C wrapper.

```bash
# Navigate to the parent directory (Documents folder, same level as tensa)
cd /path/to/Documents

# Clone the C API wrapper
git clone https://github.com/stereolabs/zed-c-api
cd zed-c-api

# Build and install (if CMake is available)
mkdir build
cd build
cmake ..
make
sudo make install

# OR build manually without CMake
g++ -shared -fPIC -o libsl_zed_c.so \
    src/zed_interface.cpp \
    src/ZEDController.cpp \
    src/ZEDFusionController.cpp \
    -Iinclude \
    -I/usr/local/zed/include \
    -I/usr/local/cuda/include \
    -L/usr/local/zed/lib \
    -L/usr/local/cuda/lib64 \
    -lsl_zed -lcudart

# Install to system or local lib directory
sudo cp libsl_zed_c.so /usr/local/lib/
# OR keep in project: cp libsl_zed_c.so /path/to/project/libs/
```

This creates `libsl_zed_c.so` and provides the C API headers.

### 3. Install TensorRT Libraries (Optional but Recommended)

TensorRT provides optimal inference performance with FP16 acceleration. The pipeline will work with CUDA-only if TensorRT is not available.

**For x86_64 Linux (CUDA 12):**
```bash
# Install TensorRT 10 for CUDA 12
sudo apt-get install -y libnvinfer10 libnvinfer-dev libnvinfer-plugin10
```

**For Jetson:**
```bash
# TensorRT comes pre-installed with JetPack
# Verify installation:
dpkg -l | grep tensorrt
```

**Note**: TensorRT is enabled by default in `bridge.c`. The first run will take 30-90s to build the engine. Subsequent runs use the cache.
1. `bridge.c` enables TensorRT provider with FP16 and engine caching.
2. Cached engines are stored in `./trt_engine_cache`.

### 4. Install ONNX Runtime with GPU Support

**Option A: Pre-built Binary (x86_64 Linux, CUDA 12 - Recommended)**

```bash
# Navigate to the parent directory (Documents folder, same level as tensa)
cd /path/to/Documents

# Download ONNX Runtime 1.19.2 GPU package (built for CUDA 12)
curl -L -o onnxruntime-linux-x64-gpu-1.19.2.tgz \
    https://github.com/microsoft/onnxruntime/releases/download/v1.19.2/onnxruntime-linux-x64-gpu-1.19.2.tgz
tar -xzf onnxruntime-linux-x64-gpu-1.19.2.tgz

# Verify libraries
ls -lh onnxruntime-linux-x64-gpu-1.19.2/lib/
```

**Note**: Extract the ONNX Runtime folder to the parent directory (same level as the `tensa` repository). The environment setup uses relative paths (`../`) to access these libraries from the tensa directory.

**Option B: Build from Source (Jetson/Custom, CUDA 12)**

For Jetson JetPack 6.2 + TensorRT or custom CUDA builds:

```bash
git clone --recursive --branch v1.19.2 https://github.com/microsoft/onnxruntime
cd onnxruntime

./build.sh --config Release --update --build --parallel \
    --use_cuda --cuda_home /usr/local/cuda \
    --use_tensorrt --tensorrt_home /usr \
    --cudnn_home /usr \
    --build_shared_lib --skip_tests
```

**CUDA 12.6 Setup:**

Ensure CUDA 12.6 is set as the default:
```bash
sudo update-alternatives --set cuda /usr/local/cuda-12.6
```

**cuDNN Installation:**

```bash
# Install cuDNN 9 for CUDA 12
sudo apt-get install -y libcudnn9-cuda-12 libcudnn9-dev-cuda-12
```

**Verify Installation:**

```bash
nvcc --version  # Should show CUDA 12.6
nvidia-smi      # Should show CUDA Version: 12.x
```

### 5. Prepare the YOLO Model

The ONNX model must be modified to accept raw ZED input (BGRA uint8) to avoid CPU preprocessing.

**Step 1: Export YOLOv8 to ONNX**

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install ultralytics onnx onnx-graphsurgeon numpy

# Export model with compatible opset
yolo export model=yolov8n.pt format=onnx opset=17
```

**Step 2: Optimize for Zero-Copy Pipeline**

```bash
python3 pkg/ai/zrt/optimize_model.py \
    --input yolov8n.onnx \
    --output yolov8n_optimized_tensorrt.onnx
```

This script:
- Casts BGRA uint8 to FP32 (for TensorRT compatibility)
- Swaps channels BGR→RGB and drops alpha
- Normalizes by dividing by 255.0
- Transposes from HWC to CHW format
- Appends EfficientNMS_TRT plugin for on-graph post-processing
- Validates FP16 compatibility
- Forces ONNX IR version 9 for compatibility with ONNX Runtime 1.17.1+ and IR version 10 for ONNX Runtime 1.19.2

**Important**: The model input size must match the resolution you pass to the test (default: 640x640).

### 6. Environment Setup

Before running tests or benchmarks, set up the required environment variables:

```bash
# Navigate to project root
cd /path/to/tensa

# Set compiler flags for CGO
export CGO_CFLAGS="-I$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/include -I$(pwd)/../zed-c-api/include -I/usr/local/cuda/include -I/usr/local/zed/include"

# Set linker flags with rpath for runtime library loading
export CGO_LDFLAGS="-L$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib -L$(pwd)/../zed-c-api/build -L/usr/local/cuda/lib64 -L/usr/local/zed/lib -Wl,-rpath,$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib -Wl,-rpath,$(pwd)/../zed-c-api/build -Wl,-rpath,/usr/local/zed/lib"

# Set library path for runtime
export LD_LIBRARY_PATH="$(pwd)/../onnxruntime-linux-x64-gpu-1.19.2/lib:$(pwd)/../zed-c-api/build:/usr/local/cuda/lib64:/usr/local/zed/lib:/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH"
```

**Note**: Adjust paths if you installed ONNX Runtime in a different location. The above assumes:
- ONNX Runtime 1.19.2 extracted in the parent directory (Documents folder)
- ZED C API built in `../zed-c-api/build/` (relative to tensa directory)
- CUDA 12.6 at `/usr/local/cuda`
- ZED SDK at `/usr/local/zed`

### 7. Run the Demo

**With Live Camera:**

```bash
go test -v ./pkg/ai/zrt/... -args -demo \
    -model $(pwd)/yolov8n_optimized_tensorrt.onnx \
    -width 640 -height 640
```

**With SVO Recording:**

```bash
go test -v ./pkg/ai/zrt/... -args -demo \
    -model $(pwd)/yolov8n_optimized_tensorrt.onnx \
    -svo /path/to/recording.svo2 \
    -width 640 -height 640
```

**Example with actual paths:**

```bash
go test -v ./pkg/ai/zrt/... -args -demo \
    -model /home/logan/Documents/tensa/yolov8n_optimized_tensorrt.onnx \
    -svo /home/logan/Documents/zed-recordings/HD1080_SN39440864_12-27-57.svo2 \
    -width 640 -height 640
```

**Flags:**
- `-demo`: Enable the demo loop (required, otherwise test is skipped)
- `-model`: Path to optimized ONNX model
- `-svo`: Path to SVO recording file (optional, omit for live camera)
- `-cam`: ZED camera ID (default: 0, ignored if `-svo` is set)
- `-width`, `-height`: Resolution (must match model input size, typically 640x640)

### 8. Running Benchmarks

The package includes Go benchmarks to measure inference performance:

**Quick Benchmark (100 iterations):**

```bash
go test -bench=BenchmarkInferenceWithMetrics -benchtime=100x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model yolov8n_optimized_tensorrt.onnx \
    -svo /path/to/recording.svo2 \
    -width 640 -height 640
```

**Extended Benchmark (500 iterations for stable measurements):**

```bash
go test -bench=BenchmarkInference -benchtime=500x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model yolov8n_optimized_tensorrt.onnx \
    -svo /path/to/recording.svo2 \
    -width 640 -height 640
```

**Run All Benchmarks:**

```bash
cd pkg/ai/zrt
./run_benchmarks.sh
```

**Save Benchmark Results:**

```bash
./run_benchmarks.sh > benchmark_$(date +%Y%m%d_%H%M%S).txt
```

**Compare Before/After Optimizations:**

```bash
# Save baseline
go test -bench=. -benchmem ./pkg/ai/zrt/... -args ... > baseline.txt

# Make optimization changes...

# Run new benchmark
go test -bench=. -benchmem ./pkg/ai/zrt/... -args ... > optimized.txt

# Compare (requires: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat baseline.txt optimized.txt
```

**Benchmark Metrics:**
- `ns/op`: Nanoseconds per inference operation (latency)
- `fps`: Frames per second (throughput)
- `ms/op`: Milliseconds per operation (latency)
- `detections/op`: Average number of detections per frame
- `B/op`: Bytes allocated per operation
- `allocs/op`: Number of allocations per operation

## Implementation Details

- `bridge.c`: C-side implementation linking ZED C API and ONNX Runtime C API with TensorRT/CUDA optimizations.
- `bridge.h`: C API header definitions.
- `zrt.go`: Go wrapper using CGO.
- `zrt_test.go`: Test suite and performance benchmarks.
- `optimize_model.py`: Python graph surgery script for model preprocessing.
- `run_benchmarks.sh`: Automated benchmark runner script.

## Known Limitations

- **Fixed Resolution**: The optimized model has fixed input dimensions. The ZED capture resolution must match the model input size.
- **IR Version**: ONNX Runtime 1.19.2 supports up to IR version 10. The optimization script forces this version.

## Troubleshooting

### Compilation Errors

- **Missing Headers**: Ensure ZED SDK and ONNX Runtime headers are available:
  ```bash
  export CGO_CFLAGS="-I/path/to/onnxruntime/include -I/path/to/zed-c-api/include -I/usr/local/cuda/include"
  ```

- **Linker Errors**: Set library paths correctly:
  ```bash
  export CGO_LDFLAGS="-L/path/to/libs -Wl,-rpath,/path/to/libs"
  ```

### Runtime Errors

- **SIGSEGV in `sl_grab()`**: Ensure you call `sl_create_camera()` before `sl_open_camera()` and pass a valid `SL_RuntimeParameters` struct (not NULL).

- **"Unsupported model IR version"**: Re-export the model with the optimization script which forces IR version 10.

- **"Invalid dimensions for input"**: The model expects 640x640 by default. Match your `-width` and `-height` flags to the model input size.

- **CUDA Provider Failed**: Ensure CUDA libraries are in `LD_LIBRARY_PATH`:
  ```bash
  export LD_LIBRARY_PATH="/usr/local/cuda/lib64:/usr/local/zed/lib:$LD_LIBRARY_PATH"
  ```
  
  If using PyTorch's CUDA libraries:
  ```bash
  NV_LIBS="$VENV/lib/python3.12/site-packages/nvidia"
  export LD_LIBRARY_PATH="$NV_LIBS/cudnn/lib:$NV_LIBS/cuda_runtime/lib:$NV_LIBS/cublas/lib:$NV_LIBS/cufft/lib:$LD_LIBRARY_PATH"
  ```

- **"Failed to open ZED camera"**: Check that:
  - ZED SDK is properly installed
  - Camera is connected (for live mode)
  - SVO file exists and is readable (for playback mode)
  - You have permissions to access the camera device

- **TensorRT Engine Building**: First run will take 30-60 seconds to build optimized engines. These are cached in `./trt_engine_cache/` for subsequent runs.

- **FP16 Numerical Issues**: If you see unexpected inference results, try disabling FP16 by setting `trt_fp16_enable` to `"0"` in bridge.c.
