# Documentation

This directory collects the higher-level notes for understanding Tensa as a
system. Start here when you want the reasoning behind the code rather than the
package-level API details.

## Guides

| Guide | Focus |
|-------|-------|
| [Architecture](architecture.md) | End-to-end system flow from cameras to court-space perception to hardware commands. |
| [Hardware](hardware.md) | Compute, sensors, mecanum drive, thrower, mechanical packaging, and firmware boundaries. |
| [Ball tracking](ai/ball-tracking-methodology.md) | Stereo detection, triangulation, filtering, trajectory association, and physics refinement. |
| [Localization](ai/localization-methodology.md) | Machine pose estimation from court geometry, camera calibration, and labeling/evaluation views. |

## Code Entrypoints

- [AI and perception](../ai/README.md)
- [Robot control](../robot/README.md)
- [Firmware](../firmware/README.md)
- [Hardware assets](../hardware/README.md)
