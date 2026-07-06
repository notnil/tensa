# Firmware

The firmware slice focuses on the ClearCore-based throw system controller.

## Contents

- `clearcore/`: Arduino/ClearCore firmware for top and bottom throw motors, angle control, dispenser control, load sensing, fault handling, and the TCP command server.
- `motor-configs/`: captured motor driver configuration files used during tuning.
- `param-files/`: integrated servo parameter snapshots.

## Protocol

The ClearCore listens for line-oriented ASCII commands:

```text
THROW <top_rpm> <bottom_rpm> <angle_radians>
DISP <rpm>
LOAD
```

`LOAD` returns `OK 1` when a ball is present and `OK 0` otherwise. See `robot/pkg/hware/thrower/protocol.md` for the matching Go-side client protocol.

Network addresses, Wi-Fi credentials, bench scripts, generated STM32 vendor trees, and archived experiments were left out of the public repo.
