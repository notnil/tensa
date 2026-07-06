# Robot Control

This directory contains the Go control stack that ran on the Jetson-class robot computer.

## What Is Here

- `cmd/tensactl`: the main robot control process.
- `pkg/hware`: hardware interfaces for the thrower, mecanum drive, ZED cameras, controller, and Jetson stats.
- `pkg/ai`: runtime consumers for localization, player tracking, navigation, and drill logic.
- `pkg/tennis`: court geometry helpers used by navigation and drill targeting.
- `pkg/pubsubx` and `pkg/metrics`: lightweight stream plumbing used between perception, control, and UI layers.

The production deployment files from the original private repo were intentionally omitted or replaced. They contained machine-specific network settings, internal registry paths, and developer workstation assumptions that are not useful in a public portfolio repo.

## Build

```bash
cd robot
go test ./...
go build ./cmd/tensactl
```

ZED hardware support requires the Stereolabs SDK and the `zed_sdk` build tag:

```bash
cd robot
go build -tags=zed_sdk ./cmd/tensactl
```
