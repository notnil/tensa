# Architecture Notes

Tensa split naturally into four layers.

## 1. Perception

The later AI stack used ZED stereo cameras and a Python pipeline:

1. Read synchronized stereo frames from SVO recordings.
2. Detect tennis ball candidates in left and right images.
3. Match left/right detections with epipolar and area constraints.
4. Triangulate 3D ball points directly from disparity.
5. Transform camera-frame points into machine and court coordinates.
6. Associate detections into trajectories with a physics-aware tracker.
7. Refine trajectories and bounces with gravity-constrained fitting.

This was chosen after SDK depth-map lookups proved too noisy for a tennis ball. Direct triangulation was simpler and gave better range and depth stability.

## 2. Localization and Court Geometry

The court coordinate frame uses the net center as a natural origin. The system combines known court geometry, camera extrinsics, camera intrinsics, and machine pose to convert detections into useful robot/court coordinates.

Localization work included:

- keypoint-based court/camera pose estimation,
- labeler workflows for ground-truth machine pose,
- multi-camera consistency checks,
- robot-relative to court-space transforms for player and ball positions.

## 3. Robot Runtime

The Go runtime was responsible for hardware coordination:

- mecanum drive commands,
- thrower command dispatch,
- controller/BLE integration,
- ZED camera control through a CGO wrapper,
- navigation primitives,
- drill definitions,
- logs and telemetry streams.

The runtime code is in `robot/`.

## 4. Firmware and Low-Level Motion

The ClearCore firmware owned the throw mechanism:

- top and bottom throw wheel velocity control,
- angle motor homing and positioning,
- dispenser speed control,
- load sensor reads,
- simple TCP command protocol,
- motor fault handling.

The Go runtime talks to this firmware through the line-oriented protocol documented in `robot/pkg/hware/thrower/protocol.md`.
