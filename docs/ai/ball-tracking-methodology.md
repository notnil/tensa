# Ball Tracking Methodology

This document describes the approach for detecting, tracking, and refining tennis ball trajectories from multi-camera stereo footage.

## Why Direct Stereo

Tennis balls were a poor fit for raw SDK depth-map lookups: they are small,
fast, texture-light, and frequently motion blurred. The more reliable approach
was to detect the ball independently in the left and right camera frames, match
those detections using stereo constraints, and triangulate depth directly from
disparity.

That made the pipeline easier to debug. A bad 3D point could be traced back to a
specific left/right detection pair, match constraint, or calibration issue.

## 1. Ball Detection

### Stereo Triangulation

Ball detection uses a two-stage approach:

1. **2D Detection**: A visual detector (e.g., SAM3, YOLO) identifies ball candidates in both left and right camera images, producing bounding boxes and confidence scores.

2. **Stereo Matching**: Detections from left and right images are matched based on:
   - **Epipolar constraint**: Matching detections must have similar Y coordinates (within a tolerance, e.g., 5 pixels)
   - **Disparity direction**: Left image X coordinate must be greater than right image X coordinate
   - **Area similarity**: Bounding box areas must be within 30% of each other

3. **Triangulation**: For matched pairs, 3D position is computed from disparity:
   ```
   disparity = u_left - u_right
   Z = (fx * baseline) / disparity
   X = (u_left - cx) * Z / fx
   Y = (v - cy) * Z / fy
   ```

### Coordinate Transform to World Space

Detections in camera frame are transformed to court coordinates:

1. **Camera → Machine**: Apply camera extrinsics (rotation and translation)
2. **Machine → Court**: Apply machine yaw rotation and position offset

```
p_machine = R_cam @ p_camera + t_cam
p_court = R_yaw @ p_machine + machine_position
```

Court-space output is the key integration point with the robot runtime. Once the
ball is expressed in meters relative to the court, targeting and drill logic do
not need to reason about camera pixels.

## 2. Detection Filtering

### Exclusion Zones (camera_stand profile only)

For the `camera_stand` extrinsics profile, certain image regions produce unreliable detections due to the camera mounting position and are excluded:

| Camera | Exclusion Zone |
|--------|---------------|
| left   | Bottom 25% AND right half of image |
| right  | Bottom 25% of image |

These zones typically contain the camera stand structure or floor reflections. Other extrinsics profiles (e.g., `body_cam`) may not require exclusion zones.

### Court Bounds Filter

Detections are filtered based on court position:
- Only balls on the **near side** of the net are kept (Y < -0.305m from net center)
- This filters out false positives on the far court

### Confidence Threshold

Detections below the minimum confidence threshold (default: 0.5) are discarded.

## 3. Multi-Frame Tracking

### Association Algorithm

Each frame, detections are associated with existing tracks using nearest-neighbor matching:

1. **Prediction**: For each active track, predict the next position using a physics-aware motion model
2. **Matching**: Associate each detection with the nearest predicted position within the maximum association distance (5m)
3. **Track Creation**: Unassigned detections start new tracks
4. **Track Termination**: Tracks without detections for N consecutive frames (default: 5) are terminated

The tracker is intentionally physics-aware but not fully dependent on a perfect
model. It uses the model to predict and refine, while still accepting noisy
detections when they are close enough to the predicted path.

### Physics-Aware Motion Prediction

Position prediction accounts for gravity and bounces:

```
velocity_z -= gravity * dt
new_position = last_position + velocity * dt

if new_position_z < ball_radius:
    new_position_z = ball_radius
    velocity_z = -velocity_z * coefficient_of_restitution
```

### Velocity Estimation

Velocity is computed from consecutive positions with gravity compensation:

```
raw_velocity = (position - last_position) / dt
velocity_z += 0.5 * gravity * dt  # Compensate for gravity effect
velocity = 0.5 * previous_velocity + 0.5 * new_velocity  # Exponential smoothing
```

## 4. Trajectory Validation

Completed tracks are validated against multiple criteria:

| Criterion | Threshold | Purpose |
|-----------|-----------|---------|
| Minimum frames | 3 | Filter noise/spurious detections |
| Minimum Y displacement | 3.0m | Ensure ball traveled meaningfully along the court |
| Minimum average velocity | 2.0 m/s | Filter stationary/rolling balls |
| Not stationary | 3 consecutive frames < 0.1 m/s | Filter balls that stopped moving |

**Y displacement** is computed over the entire track as `max(Y) - min(Y)`, representing the total distance traveled along the court axis (baseline to baseline). This filters out trajectories that didn't travel down the court, such as brief false detections or sideways movement.

Tracks failing any criterion are discarded.

## 5. Trajectory Refinement

### Piecewise Physics-Informed Fitting

Raw detections are noisy. Trajectory refinement fits the data to a physically-plausible model:

**Horizontal motion** (constant velocity):
```
x(t) = x0 + vx * t
y(t) = y0 + vy * t
```

**Vertical motion** (gravity):
```
z(t) = z0 + vz * t - 0.5 * g * t²
```

where g = 9.81 m/s².

### Bounce Detection

Bounces are detected by identifying sharp upward velocity changes near ground level:

1. Compute vertical velocity before and after each point
2. A bounce is detected when:
   - Height is below threshold (0.20m)
   - Upward velocity change exceeds 3.0 m/s

### Fitting Procedure

1. **Global X/Y fit**: Fit a single constant-velocity model to all X and Y coordinates (prevents trajectory "twisting")

2. **Segment identification**: Split trajectory at detected bounce times

3. **Per-segment Z fit**: For each flight segment, fit Z with gravity constraint:
   ```
   z_transformed = z_observed + 0.5 * g * t²
   [z0, vz] = least_squares_fit(z_transformed)
   ```

4. **Interpolation**: Evaluate the fitted model at any timestamp to get smooth, physics-consistent positions

## 6. Physics Constants

| Constant | Value | Description |
|----------|-------|-------------|
| Gravity | 9.81 m/s² | Acceleration due to gravity |
| Ball radius | 0.0335 m | Tennis ball radius (ground clamp) |
| Coefficient of restitution | 0.75 | Energy retained after bounce |

## 7. Output Data Format

### Per-Frame Detections

| Field | Description |
|-------|-------------|
| `timestamp_ns` | Frame timestamp in nanoseconds |
| `cam` | Source camera name |
| `confidence` | Detection confidence (0-1) |
| `x`, `y`, `z` | Position in court coordinates (meters) |
| `track_id` | Assigned trajectory ID |

### Trajectories

| Field | Description |
|-------|-------------|
| `track_id` | Unique trajectory identifier |
| `positions` | List of [x, y, z] positions |
| `timestamps` | List of timestamps for each position |
| `refined_positions` | Physics-fitted positions (if refinement enabled) |
| `refinement_info` | Bounce times, segment parameters, fit quality |
