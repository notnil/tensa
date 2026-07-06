#!/bin/bash
# Runner script for full triangulation on all cameras using the new decoupled pipeline

PYTHON="/home/logan/Documents/venv/bin/python3"
REC_DIR="data/recordings"
OUTPUT_BASE="output_triangulated"

# Step 1: Export synchronized frames and calibration
echo "Step 1: Exporting synchronized frames..."
$PYTHON scripts/export_svo_synced.py "$REC_DIR" "$OUTPUT_BASE"

# Step 2: Run 3D detection with chosen detector (default: sam3)
echo "Step 2: Running 3D detection..."
$PYTHON scripts/detect_balls.py "$OUTPUT_BASE" --detector sam3

# Step 3: Generate tracking
echo "Step 3: Generating tracking..."
$PYTHON scripts/generate_tracking.py --input "$OUTPUT_BASE" --output "$OUTPUT_BASE/tracking.json"

echo "All tasks completed. Results in $OUTPUT_BASE/tracking.json"
