# Tensa

[![CI](https://github.com/notnil/tensa/actions/workflows/ci.yml/badge.svg)](https://github.com/notnil/tensa/actions/workflows/ci.yml)

[![Tensa autonomous tennis robot moving on court](assets/hero/tensa-hero-movement.webp)](assets/hero/tensa-hero-movement.mp4)

Tensa is an autonomous tennis robot that moves around the court on its own using a mecanum drive base, localizes itself against tennis-court geometry, tracks balls and players with ZED stereo cameras, and drives a programmable throw system for repeatable shots from a compact mobile platform.

This repo is a curated engineering snapshot of the AI, robot-control, firmware, and hardware work.

## What It Does

Tensa combines perception, localization, motion, and ball delivery into one court-aware robot:

- Sees the court through multiple ZED stereo cameras.
- Estimates robot pose in a tennis-court coordinate frame.
- Detects tennis balls in stereo image pairs, triangulates 3D position, and tracks flight paths through bounces.
- Tracks players and court context for drill logic and targeting.
- Moves holonomically with a mecanum drive base.
- Controls a dual-wheel thrower, angle axis, dispenser, and load sensor through ClearCore firmware.

## System Overview

Tensa is organized around a simple idea: every subsystem should agree on where
things are on the tennis court. Camera detections begin as pixels, but the robot
runtime needs meters, headings, trajectories, and targets.

```text
ZED stereo cameras
        |
        v
Ball, player, and court perception
        |
        v
Court-space localization and 3D tracking
        |
        v
Robot runtime, drill logic, and targeting
        |
        v
Mecanum movement + ClearCore thrower firmware
```

The cameras provide the raw scene. The AI stack turns image observations into
court-space state. The Go runtime consumes that state to decide where the robot
should move, where it should aim, and what the throw system should do. The
firmware owns the low-level motor details for repeatable ball delivery.

The code is useful for understanding the architecture and implementation direction. Reproducing the full robot requires hardware, model weights, calibration data, and ZED recordings that are intentionally not included.

## Highlights

- Real-time stereo ball tracking from ZED cameras using a custom 2D-to-3D triangulation path instead of noisy SDK depth-map lookups.
- YOLO/SAM-assisted tennis ball detection, with TensorRT export support for Jetson-class inference.
- Multi-camera court localization and robot pose estimation from known court geometry.
- Physics-aware ball trajectory tracking with Kalman filtering, gravity, bounces, and offline/online refinement.
- Go robot-control stack for mecanum drive, thrower control, BLE/gamepad control, ZED camera capture, NATS-style pub/sub, and drill logic.
- ClearCore firmware for the throw system: top/bottom wheel motors, angle control, dispenser, load sensor, TCP command server, and fault handling.
- Hardware exploration around mecanum drive modules, compact throw assemblies, Jetson/ZED camera packaging, composite chassis, and serviceable electronics.

## Repository Map

| Path | What is there |
|------|---------------|
| `ai/` | Python ball tracking, localization, and training/evaluation code. |
| `robot/` | Go runtime for hardware control, drill logic, telemetry, camera interfaces, and court geometry helpers. |
| `firmware/` | ClearCore firmware and motor parameter/configuration snapshots for the throw system. |
| `hardware/` | Hardware notes and visual references for the mobile robot, drive base, thrower, and camera packaging. |
| `docs/` | Architecture, robot runtime, ball-tracking, localization, and hardware methodology notes. |
| `assets/` | Selected AI, localization, hardware, and app visuals. |

## Suggested Reading Path

1. Start with the [documentation index](docs/README.md) and [Architecture notes](docs/architecture.md) for the system split.
2. Read [Ball tracking methodology](docs/ai/ball-tracking-methodology.md) for stereo geometry and the physics tracker.
3. Read [Localization methodology](docs/ai/localization-methodology.md) for court-frame pose estimation.
4. Read [Robot runtime](docs/robot-runtime.md) for the Go control loop and hardware abstractions.
5. Browse [Firmware](firmware/README.md) and [Hardware notes](docs/hardware.md) for the thrower, drive base, and packaging work.

## AI System

The perception work evolved toward a four-camera ZED setup. Each camera gives a
stereo pair, so the robot can use left/right image geometry instead of treating
the scene as a single flat video feed.

The central AI tasks were:

- localize the robot against known tennis-court geometry,
- detect tennis balls in stereo image pairs,
- triangulate 3D ball positions from disparity,
- associate noisy detections into flight trajectories,
- track players and court context for drill logic,
- feed the runtime with court-space state instead of pixels.

![Conceptual localization flow from camera observations to court pose](assets/ai/localization-methodology-diagram.svg)

### Localization

Localization gives the robot its own court pose: X position, Y position, and
heading. The solver combines known court geometry, camera intrinsics, camera
extrinsics, and observed court keypoints. Once the robot has a pose estimate,
ball detections, player positions, navigation targets, and throw targets can all
use the same coordinate frame.

![Camera views projected into a court-coordinate localization estimate](assets/ai/localization-camera-to-court.jpg)

[Localization demo video](assets/ai/localization-demo.mp4)

### Ball Tracking

The most successful ball-tracking path used independent left/right 2D
detections, epipolar matching, direct stereo triangulation, and transformation
into court coordinates. The major lesson was that SDK depth maps worked well for
surfaces and people but were unreliable for tennis balls: the ball is small,
textureless, fast, and often blurred. Direct stereo geometry produced much more
stable depth and cleaner bounce locations.

![3D ball trajectory visualizer](assets/ai/trajectory-3d.png)

[YOLO ball detector demo video](assets/ai/yolo-ball-detection-demo.mp4)

More detail:

- [AI overview](ai/README.md)
- [Training code notes](ai/training/README.md)
- [Ball tracking methodology](docs/ai/ball-tracking-methodology.md)
- [Localization methodology](docs/ai/localization-methodology.md)
- [Architecture notes](docs/architecture.md)

## Hardware and Firmware

Tensa's hardware direction combined a Jetson compute stack, ZED cameras, a
ClearCore throw controller, mecanum drive modules, a compact dual-wheel throw
system, and a composite shell/chassis concept.

The hardware split mirrors the software split:

- Jetson-class compute runs perception, runtime coordination, and debug tools.
- ZED cameras provide stereo views for court localization and 3D ball tracking.
- A mecanum drive base lets the robot translate and rotate independently.
- ClearCore firmware controls the throw wheels, angle axis, dispenser, and load
  sensor.
- The mechanical package keeps cameras high enough for court coverage while
  preserving room for the hopper, thrower, batteries, and electronics.

The mecanum base matters because the robot can strafe into position without
turning its camera and thrower package away from the court. The tradeoff is that
localization errors become visible quickly, so movement and targeting depend on
stable court-frame pose estimates.

[Court movement demo video](assets/robot/court-movement-demo.mp4)

![Throw system prototype](assets/hardware/throw-system.jpg)

![Mecanum wheel assembly concept](assets/hardware/mecanum-assembly.png)

![Shell and camera layout concept](assets/hardware/shell-camera-layout.jpg)

[Throw system consistency test video](assets/hardware/throw-system-test.mp4)

More detail:

- [Firmware overview](firmware/README.md)
- [Hardware notes](docs/hardware.md)
- [Thrower protocol](robot/pkg/hware/thrower/protocol.md)

## iOS App

The companion iOS app turned the robot stack into a player-facing training
experience. It organized autonomous drills, skill progressions, weekly
challenges, leaderboards, and post-session performance review around the same
court-space data used by the robot runtime.

![Tensa iOS app screens for Social, Skills, Drills, and Performance](assets/app/mobile-app-screens.png)

The app covered four major workflows:

- Social challenges and leaderboard views for weekly competitive drills.
- Skill cards and drill libraries for browsing practice content by technique.
- Drill launch and playback flows that connected mobile UI actions to the
  robot's autonomous movement and ball-feed routines.
- Performance summaries with Tensa score, shot statistics, player movement,
  court activity, and in/out placement maps.

The iOS control path used the robot BLE interface for movement, rotation,
stopping, throw setup, load/throw commands, recording, drill start/stop, live
location updates, and player-pose notifications. That made the app both a
training product surface and a practical controller for court testing.

## Running Checks

The default checks are designed to run on a normal development machine without robot hardware:

```bash
make test-go
make test-python
```

The Go code builds without ZED hardware by using stub implementations. ZED support requires the Stereolabs SDK and the `zed_sdk` build tag.

GitHub Actions runs the same checks on `main` and pull requests.

Hardware/native checks are opt-in:

```bash
make test-go-native   # includes native packages such as ZED/ONNX runtime
make test-go-hardware # includes CAN, BLE, speaker, and robot bench tests
```

## Included and Omitted Assets

Included:

- Core Go robot-control code.
- Python ball-tracking, localization, and training/evaluation scaffolding.
- ClearCore thrower firmware and motor configuration snapshots.
- Representative AI, localization, hardware, and firmware visuals.

Not included:

- Raw datasets, SVO recordings, training runs, and cloud buckets.
- Large model checkpoints, TensorRT engines, and private calibration datasets.
- Private deployment details, machine-specific setup, and internal service wiring.

## License

MIT. See [LICENSE](LICENSE).
