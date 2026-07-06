# Architecture Notes

Tensa split naturally into four layers: perception, localization, robot runtime,
and low-level motion/firmware. The important design choice was to keep court
understanding in a stable coordinate frame so perception, movement, and throw
targeting could share the same geometry.

```text
Camera frames -> Court-space perception -> Drill/runtime decisions -> Hardware commands
```

## Data Flow

1. ZED cameras capture synchronized stereo views.
2. Perception code detects balls, players, and court features.
3. Stereo geometry converts detections into 3D machine-frame points.
4. Localization transforms machine-frame observations into court coordinates.
5. The Go runtime consumes court-space state and emits movement or thrower commands.
6. Firmware and motor drivers execute low-level motion with safety and fault handling.

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

This layer is what made the rest of the robot court-aware. Once detections were
in court coordinates, movement, drill definitions, and throw targeting could use
the same units and reference frame.

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

## Public Build Boundary

The default public build uses stubs or portable tests for hardware-dependent
packages. Hardware-specific paths are still preserved where they explain the
actual robot design, but CAN, BLE, speaker, ZED SDK, CUDA, and ONNX Runtime
checks are opt-in through the Makefile targets documented in the top-level
README.
