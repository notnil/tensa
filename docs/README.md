# Documentation

This directory collects the higher-level notes for understanding Tensa as a
system. Start here when you want the reasoning behind the code rather than the
package-level API details.

## Guides

| Guide | Focus |
|-------|-------|
| [Architecture](architecture.md) | End-to-end system flow from cameras to court-space perception to hardware commands. |
| [Robot runtime](robot-runtime.md) | Go control process, state model, hardware interfaces, and test boundaries. |
| [Hardware](hardware.md) | Compute, sensors, mecanum drive, thrower, mechanical packaging, and firmware boundaries. |
| [Ball tracking](ai/ball-tracking-methodology.md) | Stereo detection, triangulation, filtering, trajectory association, and physics refinement. |
| [Localization](ai/localization-methodology.md) | Machine pose estimation from court geometry, camera calibration, and labeling/evaluation views. |

## Visual Assets

| Asset | What it shows |
|-------|---------------|
| [`assets/hero/tensa-hero-movement.webp`](../assets/hero/tensa-hero-movement.webp) | Animated README hero showing the robot moving on court. |
| [`assets/hero/tensa-hero-movement.mp4`](../assets/hero/tensa-hero-movement.mp4) | Full MP4 version of the movement hero. |
| [`assets/diagrams/robot-system-architecture.svg`](../assets/diagrams/robot-system-architecture.svg) | Robot architecture across sensing, perception, runtime, app, mobility, thrower, and telemetry subsystems. |
| [`assets/diagrams/ai-perception-pipeline.svg`](../assets/diagrams/ai-perception-pipeline.svg) | Perception flow from ZED stereo frames to localization, triangulation, physics tracking, and runtime outputs. |
| [`assets/diagrams/runtime-control-loop.svg`](../assets/diagrams/runtime-control-loop.svg) | Go runtime control loop from perception and controller streams to navigation and thrower commands. |
| [`assets/diagrams/hardware-subsystems.svg`](../assets/diagrams/hardware-subsystems.svg) | Hardware and firmware boundaries across compute, cameras, drive base, thrower, power, and packaging. |
| [`assets/ai/localization-methodology-diagram.svg`](../assets/ai/localization-methodology-diagram.svg) | Conceptual localization flow from camera observations to court pose. |
| [`assets/ai/localization-demo.mp4`](../assets/ai/localization-demo.mp4) | Multi-camera localization demo with court pose overlay. |
| [`assets/ai/localization-camera-to-court.jpg`](../assets/ai/localization-camera-to-court.jpg) | Multi-camera localization inputs beside the solved court pose. |
| [`assets/ai/localization-eval-montage.jpg`](../assets/ai/localization-eval-montage.jpg) | Localization evaluation cases across lighting, courts, and camera modes. |
| [`assets/ai/trajectory-3d.png`](../assets/ai/trajectory-3d.png) | 3D ball trajectory visualization. |
| [`assets/ai/yolo-ball-detection-demo.mp4`](../assets/ai/yolo-ball-detection-demo.mp4) | Detector output on tennis-ball footage. |
| [`assets/robot/court-movement-demo.mp4`](../assets/robot/court-movement-demo.mp4) | Indoor court movement demo showing the prototype traversing laterally. |
| [`assets/app/mobile-app-screens.png`](../assets/app/mobile-app-screens.png) | iOS app screens for Social, Skills, Drills, and Performance. |
| [`assets/hardware/throw-system-test.mp4`](../assets/hardware/throw-system-test.mp4) | Independent thrower consistency test. |

## Code Entrypoints

- [AI and perception](../ai/README.md)
- [Robot control](../robot/README.md)
- [Firmware](../firmware/README.md)
- [Hardware assets](../hardware/README.md)
