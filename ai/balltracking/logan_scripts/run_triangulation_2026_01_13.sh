#!/bin/bash
# Runner script for Jan 13th recording session with body_cam extrinsics
# Updated for the decoupled pipeline

DATA_DIR="data/tensa-recordings/2026-01-13T13-11-17"
OUTPUT_DIR="output_triangulated_2026_01_13"
PYTHON="/home/logan/Documents/venv/bin/python3"

# Step 1: Export synchronized frames
echo "Step 1: Exporting synchronized frames..."
$PYTHON scripts/export_svo_synced.py "$DATA_DIR" "$OUTPUT_DIR"

# Step 2: Run 3D detection
echo "Step 2: Running 3D detection (SAM3)..."
$PYTHON scripts/detect_balls.py "$OUTPUT_DIR" --detector sam3

# Step 3: Generate tracking data using body_cam preset
echo "Step 3: Generating tracking data using body_cam preset..."
$PYTHON scripts/generate_tracking.py \
    --input "$OUTPUT_DIR" \
    --labels-dir "$DATA_DIR" \
    --output "$OUTPUT_DIR/tracking.json" \
    --preset "body_cam" \
    --machine-x 0 --machine-y 11.88 --machine-yaw 270

echo "All tasks completed. Results in $OUTPUT_DIR/tracking.json"
