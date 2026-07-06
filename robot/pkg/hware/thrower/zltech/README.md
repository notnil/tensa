# ZLTECH Wheel Control Client

A simplified CANopen client for controlling a single ZLAC8015D wheel motor in thrower applications.

## Features

- **Initialize**: Configure and enable the wheel motor
- **Spin**: Set wheel to spin at a specific RPM (positive or negative)
- **Stop**: Bring wheel to a controlled stop
- **Enable/Disable**: Engage or freewheel the motor
- **Status**: Read diagnostic information (RPM, temperature, current, faults)

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/notnil/tensa/pkg/hware/thrower/zltech"
    "github.com/notnil/canbus"
)

func main() {
    // Open CAN bus
    bus, err := canbus.DialSocketCAN("can0")
    if err != nil {
        log.Fatal(err)
    }
    defer bus.Close()

    // Create client for single motor on node 1
    client := zltech.New(bus, zltech.Config{
        NodeID:            1,
        Side:              zltech.Left,
        SingleMotor:       true,                 // Use sub-index 0 for single motor setup
        KeepAliveInterval: 500 * time.Millisecond, // Prevent communication timeout
    })
    defer client.Close()

    ctx := context.Background()

    // Initialize the wheel
    if err := client.Initialize(ctx); err != nil {
        log.Fatal(err)
    }

    // Spin at 100 RPM with 1 second acceleration/deceleration
    if err := client.Spin(ctx, 100, 1000, 1000); err != nil {
        log.Fatal(err)
    }

    // Wait for motor to reach speed
    time.Sleep(2 * time.Second)

    // Get status
    status, err := client.Status(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Status: %s", status)

    // Stop the wheel
    if err := client.Stop(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

The `Config` struct allows you to configure:

- **NodeID**: CANopen node ID (1-127, default: 1)
- **Side**: Which motor to control (`Left` or `Right`)
- **SingleMotor**: Set to `true` for single motor operation (uses sub-index 0), `false` for dual motor operation (uses sub-index 1/2)
- **Logger**: Optional `*slog.Logger` for debugging
- **KeepAliveInterval**: Duration for periodic velocity re-transmission (0 = disabled, recommend 500ms to prevent communication offline timeout)
- **Mux**: Optional shared `*canbus.Mux`. Required when controlling multiple wheels on the same bus (see [Multiple Wheels](#multiple-wheels))

### Single Motor vs Dual Motor Mode

The ZLAC8015D can operate in two modes:

- **Single Motor Mode** (`SingleMotor: true`): Uses CANopen sub-index 0 for all operations. Use this when only one motor is connected or when the controller is configured for single motor operation.
- **Dual Motor Mode** (`SingleMotor: false`): Uses CANopen sub-index 1 for Left, sub-index 2 for Right. Use this when controlling dual motors independently.

### Keep-Alive Feature

The ZLAC8015D has a built-in communication offline timeout (register `0x2000`, default 1000ms) that automatically stops the motor if no commands are received within the timeout period. This is a safety feature to prevent runaway motors.

To prevent this timeout during sustained operations, enable the keep-alive feature by setting `KeepAliveInterval` in the config:

```go
client := zltech.New(bus, zltech.Config{
    NodeID:            1,
    KeepAliveInterval: 500 * time.Millisecond, // Re-send commands every 500ms
})
```

When enabled:
- The client automatically re-sends the last velocity command at the specified interval
- During `Enable()`, zero velocity is sent to keep communication alive
- During `Disable()`, zero velocity continues to be sent (maintains channel without spinning motor)
- The goroutine is automatically stopped when `Close()` is called

**Recommendation**: Set `KeepAliveInterval` to 500ms or less, which is half the default timeout period, providing a safety margin.

### Multiple Wheels

When controlling multiple wheels on the same CAN bus (e.g., two separate ZLAC8015D controllers with different node IDs), all clients **must** share the same `canbus.Mux` instance. The Mux is responsible for multiplexing CAN frames, and having multiple Mux instances on the same bus will cause communication failures.

```go
// Open CAN bus
bus, err := canbus.DialSocketCAN("can0")
if err != nil {
    log.Fatal(err)
}
defer bus.Close()

// Create a shared Mux for all clients on this bus
mux := canbus.NewMux(bus)

// Create first wheel client (node ID 4)
wheel1 := zltech.New(bus, zltech.Config{
    NodeID:      4,
    Side:        zltech.Left,
    SingleMotor: true,
    Mux:         mux, // Share the Mux
})
defer wheel1.Close()

// Create second wheel client (node ID 5)
wheel2 := zltech.New(bus, zltech.Config{
    NodeID:      5,
    Side:        zltech.Left,
    SingleMotor: true,
    Mux:         mux, // Share the same Mux
})
defer wheel2.Close()

ctx := context.Background()

// Initialize both wheels
if err := wheel1.Initialize(ctx); err != nil {
    log.Fatal(err)
}
if err := wheel2.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// Both wheels can now be controlled independently
wheel1.Spin(ctx, 200, 1000, 1000)
wheel2.Spin(ctx, 200, 1000, 1000)
```

## CAN Bus Setup

The ZLAC8015D requires a configured CAN interface. On Linux:

```bash
# Bring up CAN interface at 500k baud
sudo ip link set can0 type can bitrate 500000
sudo ip link set up can0

# Verify
ip link show can0
```

## Testing

Run the integration tests with real hardware:

```bash
# Run all tests
go test -v ./pkg/hware/thrower/zltech

# Run specific test
go test -v -run TestWheelIntegration ./pkg/hware/thrower/zltech

# Specify CAN interface
CAN_INTERFACE=can1 go test -v ./pkg/hware/thrower/zltech
```

## API Reference

### Initialize

```go
func (c *Client) Initialize(ctx context.Context) error
```

Configures the wheel and transitions it through the CiA402 state machine to operational state.

### Spin

```go
func (c *Client) Spin(ctx context.Context, rpm int32, accMs, decMs uint32) error
```

Sets the wheel to spin at the given RPM with specified acceleration and deceleration times in milliseconds. Positive values spin forward, negative values spin backward.

### Stop

```go
func (c *Client) Stop(ctx context.Context) error
```

Brings the wheel to a controlled stop using the previously configured deceleration time.

### Enable/Disable

```go
func (c *Client) Enable(ctx context.Context) error
func (c *Client) Disable(ctx context.Context) error
```

Enable engages the motor (can produce torque). Disable puts the motor in freewheel mode (no torque).

### Status

```go
func (c *Client) Status(ctx context.Context) (*Status, error)
```

Returns current status and diagnostic information:

- **State**: CiA402 state machine state
- **ActualRPM**: Current motor speed
- **Temperature**: Motor temperature in °C
- **Current**: Motor current in Amperes
- **FaultCode**: Current fault code
- **StatusWord**: Raw CiA402 status word

## Protocol Details

Based on ZLAC8015D documentation:
- CANopen CiA301 and CiA402 protocols
- Default baud rate: 500 kbps
- Object Dictionary access via SDO
- Profile velocity mode (mode 3)

## License

See main repository LICENSE file.
