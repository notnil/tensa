# Player Tracking Package

The `player` package provides player location tracking by subscribing to DeepStream's person tracking and robot localization NATS messages. It transforms robot-relative 3D pose detections into world coordinate player locations on the tennis court.

## Overview

This package:
- Subscribes to `PERSON_TRACKER` NATS topic for 3D person pose data
- Subscribes to `LOCALIZER` NATS topic for robot location data
- Maintains a history of recent robot locations for timestamp correlation
- Transforms robot-relative coordinates to world coordinates using the robot's position and rotation
- Outputs player locations as `pubsubx.Sub[metrics.Metric[[]location.Loc]]`

## Usage

### Stateful Provider (Recommended)

The `StatefulProvider` implements the `Provider` interface and manages subscriptions internally:

```go
package main

import (
    "log/slog"
    "os"
    "time"
    
    "github.com/notnil/tensa/pkg/ai/player"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    // Create stateful provider (manages subscriptions internally)
    provider, err := player.NewStatefulProvider(
        "nats://localhost:4222",  // NATS URL
        "PERSON_TRACKER",          // Person tracking topic
        "LOCALIZER",               // Robot localization topic
        logger,
    )
    if err != nil {
        logger.Error("failed to create provider", slog.Any("error", err))
        return
    }
    defer provider.Close()
    
    // Poll for current player locations
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for range ticker.C {
        players, err := provider.Players()
        if err != nil {
            logger.Warn("no player data", slog.Any("error", err))
            continue
        }
        
        for i, playerLoc := range players {
            logger.Info("player detected",
                slog.Int("player_id", i),
                slog.Float64("x", playerLoc.Location.X),
                slog.Float64("y", playerLoc.Location.Y),
                slog.Float64("yaw", playerLoc.Rotation),
            )
        }
    }
}
```

### Streaming Subscriber (Advanced)

For lower-level control, use the `Sub` directly:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/notnil/tensa/pkg/ai/location"
    "github.com/notnil/tensa/pkg/ai/player"
    "github.com/notnil/tensa/pkg/metrics"
)

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    // Create player subscriber
    playerSub := player.NewSub(
        "nats://localhost:4222",  // NATS URL
        "PERSON_TRACKER",          // Person tracking topic
        "LOCALIZER",               // Robot localization topic
        logger,
    )
    
    // Subscribe to player locations
    ctx := context.Background()
    playerCh := make(chan metrics.Metric[[]location.Loc])
    
    go func() {
        if err := playerSub.Subscribe(ctx, playerCh); err != nil {
            logger.Error("subscription error", slog.Any("error", err))
        }
    }()
    
    // Process player locations as they arrive
    for metric := range playerCh {
        players := metric.Value
        for i, playerLoc := range players {
            logger.Info("player detected",
                slog.Int("player_id", i),
                slog.Float64("x", playerLoc.Location.X),
                slog.Float64("y", playerLoc.Location.Y),
                slog.Time("timestamp", metric.Timestamp),
            )
        }
    }
}
```

## Coordinate System

### Input Coordinates (Robot-Relative)
The person tracker publishes 3D poses in the robot's coordinate frame:
- **X-axis**: Points to the robot's right (in millimeters)
- **Y-axis**: Points forward from the robot (in millimeters)
- **Z-axis**: Points upward (in millimeters)

### Output Coordinates (World Frame)
Player locations are transformed to the tennis court coordinate system:
- **X-axis**: Court width (meters)
- **Y-axis**: Court length (meters)
- **Origin**: Center of the court

The transformation accounts for:
1. Robot's position on the court (`Localizer.X`, `Localizer.Y`)
2. Robot's yaw rotation (`Localizer.Rotation`)
3. Unit conversion (millimeters to meters)

## Timestamp Correlation

The package maintains a rolling history of robot locations (default: 1000 entries, 5 seconds max age). When person tracking data arrives:

1. Extracts the oldest frame timestamp from the person tracker message
2. Finds the robot location with the closest timestamp
3. Uses that robot location to transform person coordinates to world frame

This ensures accurate spatial correlation even when the two data streams arrive at slightly different rates.

## Player Location Calculation

For each detected person:
1. **Centroid Calculation**: Computes the average of all 34 pose keypoints in 3D space
2. **Yaw Calculation**: Determines player facing direction using shoulder keypoints (indices 20 and 21):
   - Calculates the shoulder line vector from right shoulder to left shoulder
   - Computes the perpendicular vector pointing forward
   - Uses `atan2` to get the yaw angle
   - Requires minimum confidence of 0.5 for both shoulder keypoints
3. **Coordinate Transform**: Applies rotation and translation to convert from robot frame to world frame:
   ```
   worldX = robotX + (robotRelX * cos(θ) - robotRelY * sin(θ))
   worldY = robotY + (robotRelX * sin(θ) + robotRelY * cos(θ))
   worldYaw = robotYaw + playerRelativeYaw
   ```
   where `θ` is the robot's yaw angle

## Interface Compatibility

The package provides two main interfaces:

### Provider Interface
```go
type Provider interface {
    Players() ([]location.Loc, error)
}
```

Implemented by `StatefulProvider`, which manages subscriptions and provides synchronous access to the latest player locations. This is the recommended interface for most use cases.

### Streaming Sub
The `Sub` type implements:
```go
pubsubx.Sub[metrics.Metric[[]location.Loc]]
```

This makes it compatible with the existing pubsub infrastructure and can be used anywhere a streaming location subscriber is expected.

## Configuration

### StatefulProvider Configuration
The `StatefulProvider` has a configurable staleness threshold:
- `maxAge`: Maximum age of player data before it's considered stale (default: 5 seconds)

When data is older than `maxAge`, `Players()` will return an error instead of stale data.

### Sub Configuration
The `Sub` struct can be configured with:
- `maxHistorySize`: Maximum number of robot location entries to keep (default: 1000)
- `maxHistoryAge`: Maximum age of robot location entries (default: 5 seconds)

These defaults ensure good timestamp matching while preventing unbounded memory growth.

## Notes

- Player yaw is calculated from shoulder keypoint positions (requires confidence ≥ 0.5 for both shoulders)
- If shoulder keypoints have insufficient confidence, yaw defaults to 0
- The package requires both person tracker and localizer data streams to be active
- If no robot location is available when person data arrives, a warning is logged and the frame is skipped
- Multiple players can be detected simultaneously (one `location.Loc` per detected person)
- Yaw angles are normalized to the range [-π, π] and follow the convention where 0 points along positive X axis

## Testing

Run tests with:
```bash
go test ./pkg/ai/player/...
```

For integration testing with NATS:
```bash
NATS_URL="nats://localhost:4222" go test ./pkg/ai/player/... -v
```

