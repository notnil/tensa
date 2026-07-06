# Firmware

The firmware slice focuses on the ClearCore-based throw system controller.

The ClearCore owns timing-sensitive thrower behavior so the Go runtime can issue
simple commands without managing every motor detail directly.

## Contents

- `clearcore/`: Arduino/ClearCore firmware for top and bottom throw motors, angle control, dispenser control, load sensing, fault handling, and the TCP command server.
- `motor-configs/`: captured motor driver configuration files used during tuning.
- `param-files/`: integrated servo parameter snapshots.

## Responsibilities

- Hold top and bottom wheel velocity targets.
- Home and position the throw angle axis.
- Drive the dispenser motor.
- Report whether a ball is loaded.
- Surface motor faults and recovery state.
- Serve a small TCP command protocol for the robot runtime.

## Protocol

The ClearCore listens for line-oriented ASCII commands:

```text
THROW <top_rpm> <bottom_rpm> <angle_radians>
DISP <rpm>
LOAD
```

`LOAD` returns `OK 1` when a ball is present and `OK 0` otherwise. See `robot/pkg/hware/thrower/protocol.md` for the matching Go-side client protocol.

Network addresses, Wi-Fi credentials, bench scripts, generated vendor trees, and archived experiments are not part of this public tree.
