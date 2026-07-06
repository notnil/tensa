# Scripts Index

This document provides an overview of all scripts in this directory. For technical details on the multi-camera synchronization and localization approach, see [METHODOLOGY.md](./METHODOLOGY.md).

---

## Data Export & Synchronization

### `export_svo_synced.py`
Exports synchronized images and depth data from 4 ZED camera SVO files (front, back, left, right). Uses a streaming laggard-advancement algorithm to align frames within 1ms tolerance across all cameras.

**Usage:**
```bash
python export_svo_synced.py <input_dir> <output_dir> [--max-frames N] [--fast-depth] [--skip N]
```

**Outputs:**
- `{timestamp}/` folders containing `{cam}.jpg`, `{cam}_right.jpg`, and `{cam}_xyz.npy` for each camera
- `calibration.json` with stereo calibration parameters

### `export_sam3_annotated.py`
Similar to `export_svo_synced.py` but also runs SAM3 ball detection during export, producing both original and annotated images with detection bounding boxes.

**Usage:**
```bash
python export_sam3_annotated.py <input_dir> <output_dir> [--max-frames N] [--conf 0.35] [--fast-depth]
```

### `run_sequential_export.py`
Orchestrates sequential SVO export for each camera followed by post-synchronization. Useful when parallel export isn't feasible.

**Usage:**
```bash
python run_sequential_export.py <input_dir> <output_dir> [--sample-interval N] [--fast-depth] [--tolerance-ms N]
```

---

## Ball Detection

### `detect_balls.py`
Multi-camera ball detection and 3D triangulation using either SAM3 or YOLO detectors. Processes stereo image pairs and computes 3D ball positions via disparity-based triangulation.

**Usage:**
```bash
python detect_balls.py <input_dir> --detector [sam3|yolo] [--conf 0.35]
```

**Outputs:** `{cam}/triangulated_detections.json` for each camera

### `test_ball_detection.py`
Test script for ball detection methods (ZED built-in, SAM3, or circle detection). Processes a single SVO file and exports annotated frames and a video.

**Usage:**
```bash
python test_ball_detection.py --method [zed|sam3|sam3_custom_od|diff] [--prompt "tennis ball"] [--conf 0.4]
```

---

## Tracking & Trajectory

### `ball_tracker.py`
Core tracking utilities and classes for ball trajectory management. Includes:
- `BallTracker` class for associating detections across frames
- `Track` class for representing individual trajectories
- Physics-aware motion prediction with gravity and bounce handling
- Coordinate transformation utilities

### `generate_tracking.py`
Generates `tracking.json` from triangulated detections. Transforms camera-local 3D positions to world coordinates using machine pose and camera extrinsics.

**Usage:**
```bash
python generate_tracking.py --input <triangulated_dir> [--labels-dir <labels_dir>] [--output tracking.json] [--preset body_cam] [--refine]
```

### `trajectory_refinement.py`
Piecewise physics-informed trajectory fitting module. Fits noisy ball detections to parabolic models with:
- Automatic bounce detection
- Gravity-constrained vertical motion
- Constant-velocity horizontal motion
- Outlier rejection

---

## Machine Localization

### `localize_camera.py`
Backend for camera localization using N-point correspondence. Computes machine position and yaw from clicked court keypoints using least-squares optimization.

**Key features:**
- Camera extrinsic presets (`camera_stand`, `body_cam`, `body_cam_v2_front_5deg`)
- Court keypoint definitions (KP1-KP21)
- Stereo triangulation support
- Fixed-height constraint for robust fitting

### `localize_camera_web.py`
Flask web server providing a UI for manual machine localization. Allows clicking keypoints in camera images to compute machine pose.

**Usage:**
```bash
python localize_camera_web.py [input_dir] [--output labels.jsonl] [--port 5000]
```

**Endpoints:**
- `/` - Main labeling interface
- `/visualizer` - 3D visualization page
- `/api/compute` - Compute pose from N points
- `/api/save` - Save labeled frame

### `generate_machine_tracking.py`
Converts `labels.jsonl` to a machine tracking CSV with columns: frame, timestamp_ms, x, y, rotation_rad.

**Usage:**
```bash
python generate_machine_tracking.py <labels_jsonl> <output_csv> [preset]
```

---

## Visualization & Analysis

### `ball_tracker_server.py`
Flask server for the ball tracker visualizer. Serves the HTML UI and camera images from exported data.

**Usage:**
```bash
python ball_tracker_server.py [--data-dir <path>] [--fallback-dir <path>] [--port 5002]
```

### `visualize_detections.py`
3D ball detection visualizer web app. Transforms local detections to world coordinates and displays them overlaid on a court model.

**Usage:**
```bash
python visualize_detections.py --data <data_dir> --labels <labels.csv> [--port 5001]
```

### `analyze_ball_area.py`
Analyzes the relationship between ball distance and pixel area (bounding box dimensions). Fits the inverse relationship `Dimension = k / distance` and generates plots.

**Output:** `area_distance_relationship.png`

### `analyze_triangulated_area.py`
Same analysis as above but using triangulated (stereo) depth instead of depth-map depth. Compares the fitted constant to the original depth-based fit.

### `create_grid_image.py`
Creates a 2x2 grid visualization showing left/right images and their normalized depth maps.

**Usage:**
```bash
python create_grid_image.py <frame_prefix> [--max_depth N]
```

---

## Debugging & Utilities

### `inspect_sdk.py`
Prints ZED SDK version and inspects sensor data structures (angular velocity, orientation).

### `profile_export.py`
Profiles SVO file operations: open time, sequential grab speed, seek+grab speed for both `DEPTH_MODE.NONE` and `NEURAL_PLUS`.

### `test_sensors.py`
Tests IMU/sensor data availability and retrieval from an SVO file.

**Usage:**
```bash
python test_sensors.py <svo_file>
```

---

## Runner Scripts (Shell)

### `run_all_triangulation.sh`
Full pipeline runner: export → detect → track.

```bash
./run_all_triangulation.sh
```

### `run_triangulation_2026_01_13.sh`
Pipeline runner configured for the Jan 13, 2026 recording session with `body_cam` extrinsics and specific machine pose.

---

## Templates

The `templates/` folder contains HTML files for the web-based tools:
- `ball_tracker.html` - Ball tracker visualizer UI
- `index.html` - Localization labeling interface
- `visualizer.html` - 3D court visualization
