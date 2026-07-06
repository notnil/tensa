# Hardware Notes

The final hardware direction was a compact, mobile, camera-rich tennis robot rather than a fixed-location ball machine.

## Compute and Sensors

- Jetson-class compute for robot control and GPU perception.
- Four ZED camera views for court coverage and stereo ball tracking.
- IMU support for yaw/pose estimation and motion checks.
- TCP-connected ClearCore controller for the throw system.

## Motion Platform

The robot explored a mecanum drive base so it could translate and rotate independently on court. That made accurate localization and low-latency pose updates important: filtered rotation data could lag during target-practice style motion, so the runtime kept room for raw/low-latency rotation paths.

## Throw System

The throw assembly used two high-speed throw motors and an angle axis. Firmware controls:

- top and bottom wheel RPM,
- throw angle,
- dispenser RPM,
- loaded-ball sensing,
- motor fault recovery.

The test video in `assets/hardware/throw-system-test.mp4` shows an independent throw-system bench test repeatedly hitting the same wall target at a fixed speed.

## Mechanical Packaging

The hardware design explored:

- composite shell/chassis concepts,
- a low center of gravity battery and wheel "skateboard",
- serviceable electronics mounted to an internal plate,
- transparent or protected camera bands around the upper shell,
- camera angles tuned around ball tracking, player visibility, and hopper volume.

These notes are preserved as project context rather than production-ready manufacturing documentation.
