# BLE Controller Protocol Documentation

This document describes the Bluetooth Low Energy (BLE) protocol for controlling the tennis ball machine hardware.

## Overview

The BLE interface provides a GATT server that exposes characteristics for controlling and monitoring the hardware. Client applications can connect to this BLE server, read from, write to, and subscribe to notifications from these characteristics to interact with the machine.

- All multi-byte numerical values are little-endian unless otherwise noted.
- Floats are IEEE-754 `float32`.

### Safety: Command Timeout Protection

**IMPORTANT**: The robot implements automatic timeout protection for movement commands. If no `Move` or `Rotate` commands are received within the configured timeout period (default: 300ms), the robot will automatically stop for safety.

This protects against:
- Sticky BLE controllers that continue sending outdated commands
- Controller disconnects that leave the robot moving
- Communication failures or software crashes

**Best Practice**: Client applications should continuously send movement commands at a regular interval (recommended: 50-100ms) when the user is actively controlling the robot. To stop the robot, either:
1. Send a `Stop` command, or
2. Simply stop sending movement commands and let the timeout protection engage

## Services and Characteristics

The BLE interface exposes a single primary service:

- **Service UUID**: `00010000-8786-40ba-ab96-99b91ac981d8`

### Characteristics

| Characteristic | UUID | Operations | Data Format / Binary Protocol |
| :--- | :--- | :--- | :--- |
| Move | `00010001-8786-40ba-ab96-99b91ac981d8` | Write | 8 bytes: `Direction` (float32) + `Speed` (float32). Direction is radians; positive is left (CCW). |
| Rotate | `00010002-8786-40ba-ab96-99b91ac981d8` | Write | 4 bytes: `Speed` (float32, radians/sec). Positive is CCW. |
| Wheel Status | `00010003-8786-40ba-ab96-99b91ac981d8` | Read | Binary marshaled `wheels.Status` containing 4 wheels in sequence. See Wheel Status Format below. |
| Set Throw | `00010004-8786-40ba-ab96-99b91ac981d8` | Write | 12 bytes: `TopWheelSpeed` (float32) + `BottomWheelSpeed` (float32) + `Angle` (float32). |
| Throw | `00010005-8786-40ba-ab96-99b91ac981d8` | Write | No data. Commands the thrower to throw once. |
| Wheel Enable | `00010006-8786-40ba-ab96-99b91ac981d8` | Write | 1 byte: `Enable` (0 = disable, 1 = enable). |
| Stop | `00010007-8786-40ba-ab96-99b91ac981d8` | Write | No data. Stops all hardware components. |
| Load | `00010008-8786-40ba-ab96-99b91ac981d8` | Write | No data. Loads a ball into the thrower. |
| Location | `0001000b-8786-40ba-ab96-99b91ac981d8` | Notify | 20 bytes `BLELocationMessage`: `Loc` (12 bytes = X, Y, Rotation float32) + `Timestamp` (8 bytes = int64 Unix seconds). Subscribe to receive real-time location updates. |
| Record Start | `00010016-8786-40ba-ab96-99b91ac981d8` | Write | No data. Starts ZED camera recording. |
| Record Stop | `00010017-8786-40ba-ab96-99b91ac981d8` | Write | No data. Stops ZED camera recording. |
| Drill Start | `00010010-8786-40ba-ab96-99b91ac981d8` | Write | Variable-length UTF-8 string: Drill ID (slug format, e.g., "speed-drill"). Returns error if drill not found. |
| Drill Stop | `00010011-8786-40ba-ab96-99b91ac981d8` | Write | No data. Stops the running drill and stops motors. |
| Drill Status | `00010012-8786-40ba-ab96-99b91ac981d8` | Notify | 3+ bytes: `StatusCode` (uint8) + `MessageLength` (uint16 LE) + `Message` (UTF-8 string). Provides status notifications for drill operations. |
| Drill Upload | `00010013-8786-40ba-ab96-99b91ac981d8` | Write | Variable-length binary: Drill plugin upload protocol. See Drill Upload Protocol below. |
| Drill Check | `00010014-8786-40ba-ab96-99b91ac981d8` | Write + Read | Write: UTF-8 drill ID. Read: 1 byte (0x01 = exists, 0x00 = not found). Check if drill is cached before starting. |
| Player Pose | `00010015-8786-40ba-ab96-99b91ac981d8` | Notify | Variable-length `BLEPlayerPoseMessage`: Updates every 250ms with detected player poses. See Player Pose Format below. |

### Message Structures

- `BLELocationMessage` (20 bytes total)
  - 12 bytes location: `X` (float32 LE), `Y` (float32 LE), `Rotation` (float32 LE)
  - 8 bytes timestamp: Unix seconds as int64 LE

- `BLEPlayerPoseMessage` (variable length)
  - 1 byte: `Count` (uint8, number of detected players, 0-255)
  - For each player (12 bytes each):
    - 4 bytes: `X` (float32 LE, meters)
    - 4 bytes: `Y` (float32 LE, meters)
    - 4 bytes: `Rotation` (float32 LE, radians, player facing direction)
  - 8 bytes: `Timestamp` (int64 LE, Unix seconds)
  - **Minimum size**: 9 bytes (0 players + timestamp)
  - **Example with 2 players**: 33 bytes (1 + 12 + 12 + 8)

### Wheel Status Format

The `Wheel Status` characteristic returns a variable-length binary structure containing status for all 4 wheels in sequence: **FrontLeft**, **FrontRight**, **RearRight**, **RearLeft**.

Each wheel's status contains:
- 1 byte: `Enabled` (0 = disabled, 1 = enabled)
- 4 bytes: `Speed` (float32 LE, rad/s)
- 4 bytes: `Current` (float32 LE, amperes)
- 2 bytes: `ErrorLength` (uint16 LE, length of error string)
- N bytes: `Error` (UTF-8 string, variable length)

**Minimum size**: 44 bytes (4 wheels × 11 bytes each, with empty error strings)

**Example Structure**:
```
[FrontLeft: 11+ bytes][FrontRight: 11+ bytes][RearRight: 11+ bytes][RearLeft: 11+ bytes]
```

Each wheel block:
```
[Enabled:1][Speed:4][Current:4][ErrorLength:2][Error:N]
```

To decode:
1. Read each wheel sequentially (order matters)
2. For each wheel, read the first 11 bytes for enabled, speed, current, and error length
3. Read the next N bytes (where N = ErrorLength) for the error string
4. Move to the next wheel and repeat

### Endianness Notes

- All fields are little-endian, including `uint32`, `int64`, and `float32` fields.

## Drill Operations

Drills are pre-programmed tennis practice routines that coordinate navigation, throwing, and timing automatically. Drills are identified by slug strings (human-readable identifiers like "speed-drill") and are loaded dynamically from a drill registry. Navigation and ball feeding are handled internally by the drill system - clients simply start and stop drills.

### Drill Start Characteristic

**UUID**: `00010010-8786-40ba-ab96-99b91ac981d8`  
**Type**: Write  
**Description**: Starts a drill by its slug string identifier.

#### Protocol

The data sent to start a drill is a **UTF-8 encoded string** containing the drill's slug.

**Format**: `[DrillID: UTF-8 string]`

Where `DrillID` is the drill's slug (e.g., `"speed-drill"`).

#### Example Drills

Available drills are defined in the drill registry (typically loaded from `drills.json`). Examples include:

| Drill Slug | Name | Description |
|------------|------|-------------|
| `two-handed-backhand-grip` | Two-Handed Backhand Grip | Build a stable, connected two-hander |
| `speed-drill` | Speed Drill | Practice adjusting to different ball speeds |
| `criss-cross` | Criss-Cross | Master court coverage with alternating patterns |
| `target-practice` | Target Practice | Systematic accuracy training across all court keypoints |

#### Example Usage

To start the "Speed Drill":

1. Encode the slug as UTF-8 bytes: `speed-drill`
2. Write these bytes to the `Drill Start` characteristic


### Drill Stop Characteristic

**UUID**: `00010011-8786-40ba-ab96-99b91ac981d8`  
**Type**: Write  
**Description**: Stops any currently running drill and stops all motors.

Send any data (or empty payload) to this characteristic to stop the current drill. The data content is ignored.

### Drill Error Handling

Common error scenarios:

- **Empty drill ID**: Drill start data cannot be empty
- **Unknown drill ID**: Drill slug not found in registry
- **Hardware errors**: Hardware subsystem failures during drill execution (navigation, throwing, etc.)
- **Drill characteristics unavailable**: BLE controller was not initialized with drill support

Drill characteristics return standard BLE status codes:
- `0x00` (Success): Operation completed successfully
- `0x0E` (Unexpected Error): Operation failed (e.g., drill not found, drill already running)

## Client Integration Guide

Below is a minimal flow to build a client in another app (e.g., Web Bluetooth, iOS, Android, desktop). UUIDs must be lowercase for Web Bluetooth.

1. Discover and connect to the device's primary service `00010000-8786-40ba-ab96-99b91ac981d8`.
2. Get required characteristics by UUID. Common flows:
   - **Movement**: write to `Move` and `Rotate`. Emergency stop: write to `Stop`.
   - **Throwing**: write `Set Throw`, then `Load` (optional), then `Throw`.
   - **Location stream**: subscribe (notify) to `Location` to receive real-time position updates.
   - **Recording**: write to `Record Start` to start ZED camera recording, and `Record Stop` to stop.
   - **Player tracking**: subscribe (notify) to `Player Pose` to receive player positions every 250ms.
   - **Drills**: write UTF-8 drill slug string to `Drill Start`. Stop with `Drill Stop`. Navigation is handled automatically by drills.
3. Payload encoding tips:
   - Use IEEE-754 float32 and little-endian byte order for all numeric fields.
   - For timestamps in `Location`, encode as Unix seconds int64 little-endian.
   - For drills, encode the drill slug as a UTF-8 string.

### Example Encodings (pseudo-code)

- Encode float32 little-endian: `putUint32LE(float32ToBits(value))`
- Encode int64 little-endian: `putUint64LE(uint64(unixSeconds))`
- Encode drill slug: `encodeUTF8(drillSlug)` (for Drill Start)

### Drill Status Codes

The Drill Status characteristic notifies clients of drill operation status:

- 0: `Success` - Drill operation completed successfully
- 1: `ErrorNotFound` - Requested drill plugin not found on device
- 2: `ErrorLoad` - Failed to load drill plugin
- 3: `ErrorRunning` - Drill is already running
- 4: `UploadSuccess` - Drill upload completed successfully
- 5: `UploadError` - Drill upload failed

### Drill Upload Protocol

The Drill Upload characteristic uses a chunked binary protocol to transfer gzip-compressed drill plugins (.so files) to the robot. The protocol has three message types:

#### Init Message (0x01)
Initializes an upload session:
```
[Type: 0x01][IDLen: uint8][DrillID: UTF-8 string][CompressedSize: uint32 LE][UncompressedSize: uint32 LE][SHA256: 32 bytes]
```
- `Type`: 0x01
- `IDLen`: Length of drill ID string (1-255)
- `DrillID`: Drill slug (e.g., "speed-drill")
- `CompressedSize`: Size of gzipped data in bytes
- `UncompressedSize`: Size of original .so file
- `SHA256`: SHA256 hash of compressed data (32 bytes)

#### Chunk Message (0x02)
Sends a chunk of compressed data:
```
[Type: 0x02][Sequence: uint16 LE][Data: bytes]
```
- `Type`: 0x02
- `Sequence`: Chunk sequence number (0-based)
- `Data`: 244 bytes of compressed data (optimized for BLE 5.1 DLE with 251-byte packets)

#### Finalize Message (0x03)
Completes the upload and triggers decompression/validation:
```
[Type: 0x03]
```
- `Type`: 0x03 (single byte)

#### Upload Flow

**Recommended flow for starting a drill:**

1. Client writes drill ID to `DrillCheck` characteristic
2. Client reads from `DrillCheck` characteristic to check if drill exists (0x01 = yes, 0x00 = no)
3. If drill not present:
   a. Client gzip-compresses the .so file
   b. Client calculates SHA256 of compressed data
   c. Client sends Init message to `DrillUpload`
   d. Client splits compressed data into 244-byte chunks
   e. Client sends Chunk messages sequentially (with sequence numbers)
   f. Client sends Finalize message
   g. Robot decompresses, validates checksum, and writes file
   h. Robot sends status notification via `DrillStatus` (UploadSuccess or UploadError)
4. Client writes drill ID to `DrillStart` to begin drill
5. Drill executes (or returns error if still not found)

**Performance**: With BLE 5.1 + DLE (251-byte packets) + 2M PHY, expect ~200-400 kB/s throughput. Typical 750KB .so file compresses to ~250KB, transferring in 1-2 seconds.

### iOS/Swift Integration Tips

For iOS development with CoreBluetooth:

1. **Service Discovery**: Use `CBUUID(string: "00010000-8786-40ba-ab96-99b91ac981d8")` for the service.
2. **Float Encoding**: Use `Float32` and convert to `Data` with little-endian byte order:
   ```swift
   var value: Float32 = 1.5
   let data = withUnsafeBytes(of: &value) { Data($0) }
   ```
3. **String Encoding** (for Drill IDs):
   ```swift
   let drillID = "a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d"
   let data = drillID.data(using: .utf8)!
   ```
4. **Notifications**: Enable notifications on the Location characteristic to receive position updates.
5. **Write Without Response**: Most write characteristics support write-without-response for faster operation.

### Error Semantics

- Handlers return error statuses if payload length is incorrect or commands fail.
- `Stop` and `Drill Stop` also stop motors to ensure safety.

## References

- Go handlers: see files in `pkg/hware/controller/` matching `ble_*.go`.
- Drill system implementation: `pkg/ai/drillsx/`.
