#!/bin/bash
set -e

# ZRT Inference Benchmarking Script
# This script runs comprehensive benchmarks on the ZRT inference pipeline

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TENSA_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
MODEL="${TENSA_DIR}/yolov8n_optimized_tensorrt.onnx"
SVO="/home/logan/Documents/zed-recordings/HD1080_SN39440864_12-27-57.svo2"
WIDTH=640
HEIGHT=640

# Setup environment
export CGO_CFLAGS="-I${TENSA_DIR}/onnxruntime-linux-x64-gpu-1.19.2/include -I${TENSA_DIR}/zed-c-api/include -I/usr/local/cuda/include -I/usr/local/zed/include"
export CGO_LDFLAGS="-L${TENSA_DIR}/onnxruntime-linux-x64-gpu-1.19.2/lib -L${TENSA_DIR}/zed-c-api/build -L/usr/local/cuda/lib64 -L/usr/local/zed/lib -Wl,-rpath,${TENSA_DIR}/onnxruntime-linux-x64-gpu-1.19.2/lib -Wl,-rpath,${TENSA_DIR}/zed-c-api/build -Wl,-rpath,/usr/local/zed/lib"
export LD_LIBRARY_PATH="${TENSA_DIR}/onnxruntime-linux-x64-gpu-1.19.2/lib:${TENSA_DIR}/zed-c-api/build:/usr/local/cuda/lib64:/usr/local/zed/lib:/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH"

cd "$TENSA_DIR"

echo "========================================"
echo "ZRT Inference Pipeline Benchmarks"
echo "========================================"
echo "Model: $MODEL"
echo "SVO: $SVO"
echo "Resolution: ${WIDTH}x${HEIGHT}"
echo ""

# Run basic inference benchmark
echo "Running basic inference benchmark..."
go test -bench=BenchmarkInference$ -benchtime=100x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model "$MODEL" -svo "$SVO" -width $WIDTH -height $HEIGHT

echo ""
echo "----------------------------------------"
echo ""

# Run benchmark with detailed metrics
echo "Running inference benchmark with metrics..."
go test -bench=BenchmarkInferenceWithMetrics$ -benchtime=100x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model "$MODEL" -svo "$SVO" -width $WIDTH -height $HEIGHT

echo ""
echo "----------------------------------------"
echo ""

# Run longer benchmark for stable measurements
echo "Running extended benchmark (500 iterations)..."
go test -bench=BenchmarkInference$ -benchtime=500x -benchmem \
    ./pkg/ai/zrt/... \
    -args -model "$MODEL" -svo "$SVO" -width $WIDTH -height $HEIGHT

echo ""
echo "========================================"
echo "Benchmarks Complete!"
echo "========================================"
echo ""
echo "To save results to a file:"
echo "  ./run_benchmarks.sh > benchmark_results_\$(date +%Y%m%d_%H%M%S).txt"
echo ""
echo "To compare with baseline:"
echo "  go test -bench=. -benchmem ./pkg/ai/zrt/... > new.txt"
echo "  benchstat baseline.txt new.txt"


