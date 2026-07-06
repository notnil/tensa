# ZED Camera Package

This package provides a Go interface for controlling Stereolabs ZED stereo cameras using CGO to bridge to the ZED SDK C API.

## Features

- Simple interface for controlling ZED stereo cameras
- Image capture in RGBA format for easy processing
- Depth sensing with multiple quality modes
- Object detection and tracking
- Body tracking with skeleton keypoints
- Positional tracking (6DoF pose estimation)
- SVO recording and playback
- Configurable resolution, FPS, and depth mode
- Thread-safe operations

## Requirements

### For Production Use (Hardware)

- ZED SDK installed at `/usr/local/zed/`
- CGO enabled
- ZED 2i camera (USB 3.0)

### For Development/Testing

- Build without the `zed_sdk` tag to use stub implementation
- Stub returns errors but allows code to compile without hardware

## Installation

1. Install the ZED SDK from [Stereolabs](https://www.stereolabs.com/developers/release/)
2. Ensure the SDK is installed at `/usr/local/zed/`

## Usage

### With ZED Hardware

```go
package main

import (
    "log"
    "log/slog"
    
    "github.com/notnil/tensa/pkg/hware/zed"
)

func main() {
    // Create configuration
    cfg := zed.DefaultConfig()
    cfg.Resolution = zed.ResolutionHD720
    cfg.FPS = 30
    cfg.Logger = slog.Default()

    // Create and open camera
    cam := zed.NewCamera(cfg)
    if err := cam.Open(); err != nil {
        log.Fatalf("Failed to open camera: %v", err)
    }
    defer cam.Close()

    // Capture a frame
    if err := cam.Grab(); err != nil {
        log.Fatalf("Failed to grab frame: %v", err)
    }

    // Retrieve the image
    img, err := cam.RetrieveImage()
    if err != nil {
        log.Fatalf("Failed to retrieve image: %v", err)
    }

    // Process the image
    log.Printf("Captured image: %dx%d, %d bytes", img.Width, img.Height, len(img.Data))
}
```

### Object Detection

```go
// Enable positional tracking (required for object tracking)
if err := cam.EnablePositionalTracking(); err != nil {
    log.Fatalf("Failed to enable tracking: %v", err)
}
defer cam.DisablePositionalTracking()

// Enable object detection
params := zed.DefaultObjectDetectionParameters()
params.EnableTracking = true
if err := cam.EnableObjectDetection(params); err != nil {
    log.Fatalf("Failed to enable object detection: %v", err)
}
defer cam.DisableObjectDetection()

// Retrieve detected objects
if err := cam.Grab(); err != nil {
    log.Fatalf("Failed to grab: %v", err)
}

objects, err := cam.RetrieveObjects()
if err != nil {
    log.Fatalf("Failed to retrieve objects: %v", err)
}

for _, obj := range objects.ObjectList {
    log.Printf("Detected %s at position (%.2f, %.2f, %.2f)", 
        obj.Sublabel, obj.Position.X, obj.Position.Y, obj.Position.Z)
}
```

### Body Tracking

```go
// Enable body tracking
bodyParams := zed.DefaultBodyTrackingParameters()
bodyParams.BodyFormat = zed.BodyFormat18
if err := cam.EnableBodyTracking(bodyParams); err != nil {
    log.Fatalf("Failed to enable body tracking: %v", err)
}
defer cam.DisableBodyTracking()

// Retrieve bodies
bodies, err := cam.RetrieveBodies()
if err != nil {
    log.Fatalf("Failed to retrieve bodies: %v", err)
}

for _, body := range bodies.BodyList {
    log.Printf("Body %d detected with %d keypoints", body.ID, len(body.Keypoints3D))
}
```

### Positional Tracking

```go
// Enable positional tracking
if err := cam.EnablePositionalTracking(); err != nil {
    log.Fatalf("Failed to enable tracking: %v", err)
}
defer cam.DisablePositionalTracking()

// Get camera pose
if err := cam.Grab(); err != nil {
    log.Fatalf("Failed to grab: %v", err)
}

pose, err := cam.GetPosition()
if err != nil {
    log.Fatalf("Failed to get position: %v", err)
}

log.Printf("Camera position: (%.2f, %.2f, %.2f)", 
    pose.Position.X, pose.Position.Y, pose.Position.Z)
```

## Building

### Without ZED SDK (Default)

By default, the package uses a stub implementation that returns errors. This allows the package to compile without the SDK:

```bash
go build ./pkg/hware/zed/...
```

### With ZED SDK

To build with actual ZED hardware support, use the `zed_sdk` build tag:

```bash
go build -tags=zed_sdk ./pkg/hware/zed/...
```

## Testing

Run tests with ZED hardware:

```bash
go test -tags=zed_sdk ./pkg/hware/zed/...
```

The hardware test will be skipped if the ZED SDK is not installed or if no camera is connected.

### Test Application

A comprehensive test application is available at `cmd/testing/zedtest`:

```bash
# Build with ZED SDK support
cd cmd/testing/zedtest
go build -tags=zed_sdk

# Capture frames with body tracking
./zedtest -track-bodies -frames 10 -output ./output -draw-keypoints

# Show camera pose while capturing
./zedtest -show-pose -frames 10

# Record to SVO file
./zedtest -record -svo-file recording.svo -frames 100
```

## API Reference

### Camera Interface

```go
type Camera interface {
    // Core operations
    Open() error
    Grab() error
    RetrieveImage() (*Image, error)
    RetrieveImageView(view ViewType) (*Image, error)
    RetrieveMeasure(measure MeasureType) (*Measure, error)
    Close() error
    
    // Object detection
    EnableObjectDetection(params ObjectDetectionParameters) error
    RetrieveObjects() (*Objects, error)
    DisableObjectDetection() error
    
    // Body tracking
    EnableBodyTracking(params BodyTrackingParameters) error
    RetrieveBodies() (*Bodies, error)
    DisableBodyTracking() error
    
    // Positional tracking
    EnablePositionalTracking() error
    GetPosition() (*Pose, error)
    DisablePositionalTracking() error
    
    // Recording
    EnableRecording(params RecordingParameters) error
    DisableRecording() error
    GetRecordingStatus() (*RecordingStatus, error)
    PauseRecording(pause bool) error
}
```

### Image Structure

```go
type Image struct {
    Width  int    // Width in pixels
    Height int    // Height in pixels
    Data   []byte // RGBA data (4 bytes per pixel: R, G, B, A)
}
```

The `Image` type implements the standard Go `image.Image` interface, so it can be used with any Go image processing libraries:

```go
img, err := cam.RetrieveImage()
if err != nil {
    log.Fatal(err)
}

// Use directly as image.Image
var stdImg image.Image = img

// Or convert to image.RGBA for easier manipulation
rgba := img.ToRGBA()

// Now you can use standard image libraries
err = png.Encode(file, rgba)
```

### Measure Structure

```go
type Measure struct {
    Width  int         // Width of the measure in pixels
    Height int         // Height of the measure in pixels
    Data   []float32   // Measurement data (format depends on MeasureType)
    Type   MeasureType // Type of measurement
}
```

The `Measure` type holds depth, disparity, confidence, or point cloud data. You can retrieve measurement data after calling `Grab()`:

```go
// Retrieve depth map
depthMeasure, err := cam.RetrieveMeasure(zed.MeasureDepth)
if err != nil {
    log.Fatal(err)
}

// Convert to grayscale visualization
grayImg, err := depthMeasure.ToGrayscaleImage()
if err != nil {
    log.Fatal(err)
}

// Save as PNG
file, _ := os.Create("depth.png")
defer file.Close()
png.Encode(file, grayImg)
```

Available measure types:
- `MeasureDepth` - Depth map in configured units (meters by default)
- `MeasureDisparity` - Raw disparity map from stereo vision
- `MeasureConfidence` - Depth confidence/certainty values (0-100)
- `MeasureXYZ` - 3D point cloud with X, Y, Z, W coordinates

The `ToGrayscaleImage()` method converts depth, disparity, and confidence measures into normalized grayscale images (0-255 range). It automatically scales the values to utilize the full grayscale range. Point cloud data (MeasureXYZ) is not supported for grayscale conversion and will return an error.

### Available Resolutions

- `ResolutionHD2K` - 2208x1242
- `ResolutionHD1080` - 1920x1080
- `ResolutionHD720` - 1280x720
- `ResolutionVGA` - 672x376

### Depth Modes

- `DepthModeNone` - Disable depth sensing
- `DepthModePerformance` - Prioritize speed (traditional stereo)
- `DepthModeQuality` - Balance speed and accuracy (traditional stereo)
- `DepthModeUltra` - Prioritize accuracy (traditional stereo)
- `DepthModeNeuralLight` - AI-based depth (light version)
- `DepthModeNeural` - AI-based depth (standard)
- `DepthModeNeuralPlus` - AI-based depth (most accurate)

### Body Formats

- `BodyFormat18` - 18 keypoints (COCO18 skeleton)
- `BodyFormat34` - 34 keypoints (extended skeleton)
- `BodyFormat38` - 38 keypoints (full body with hands and feet)

## Notes

- The package uses the left camera for RGB images
- Images are returned in RGBA format (4 bytes per pixel)
- All operations are thread-safe
- The ZED SDK C API is used for maximum compatibility
- Positional tracking must be enabled for object tracking and body tracking
- Build with `-tags=zed_sdk` to enable hardware support
- Without the build tag, stub implementations are used that return errors

## References

- [ZED SDK Documentation](https://www.stereolabs.com/docs)
- [ZED SDK C API Reference](https://www.stereolabs.com/docs/api/c/index.html)
- [ZED SDK GitHub](https://github.com/stereolabs/zed-sdk)

