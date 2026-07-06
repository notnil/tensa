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

From the repository root, run the portable public checks:

```bash
make test-go
```

That target skips the native zero-copy ZED/ONNX runtime package and hardware
integration tests that require a robot, CAN bus, BLE adapter, speakers, or GPU
camera SDKs.

For a full local hardware bench run on a configured robot:

```bash
cd robot
go test -tags=hardware ./...
go build ./cmd/tensactl
```

ZED hardware support requires the Stereolabs SDK and the `zed_sdk` build tag:

```bash
cd robot
go build -tags=zed_sdk ./cmd/tensactl
```

The experimental zero-copy `pkg/ai/zrt` runtime also requires ONNX Runtime,
CUDA, and the ZED C API libraries. It is documented in
`pkg/ai/zrt/README.md` and intentionally excluded from the default public CI
target.
