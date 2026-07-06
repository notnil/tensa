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
| [`assets/hero/tensa-hero-movement.gif`](../assets/hero/tensa-hero-movement.gif) | Animated README hero showing the robot moving on court. |
| [`assets/hero/tensa-hero-movement.mp4`](../assets/hero/tensa-hero-movement.mp4) | Full MP4 version of the movement hero. |
| [`assets/ai/localization-camera-to-court.jpg`](../assets/ai/localization-camera-to-court.jpg) | Multi-camera localization inputs beside the solved court pose. |
| [`assets/ai/localization-eval-montage.jpg`](../assets/ai/localization-eval-montage.jpg) | Localization evaluation cases across lighting, courts, and camera modes. |
| [`assets/ai/trajectory-3d.png`](../assets/ai/trajectory-3d.png) | 3D ball trajectory visualization. |
| [`assets/ai/yolo-ball-detection-demo.mp4`](../assets/ai/yolo-ball-detection-demo.mp4) | Detector output on tennis-ball footage. |
| [`assets/hardware/throw-system-test.mp4`](../assets/hardware/throw-system-test.mp4) | Independent thrower consistency test. |

## Code Entrypoints

- [AI and perception](../ai/README.md)
- [Robot control](../robot/README.md)
- [Firmware](../firmware/README.md)
- [Hardware assets](../hardware/README.md)
