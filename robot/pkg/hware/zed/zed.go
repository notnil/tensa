// Package zed provides an interface for controlling Stereolabs ZED stereo cameras.
// The package uses CGO to bridge to the ZED SDK C API for hardware access,
// and includes a mock implementation for testing without physical hardware.
//
// The ZED SDK must be installed at /usr/local/zed/ for the production implementation to work.
package zed

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"time"
)

// Camera defines the interface for controlling a ZED stereo camera.
// Implementations must handle initialization, frame capture, and cleanup.
type Camera interface {
	// Open initializes and opens the camera with the configured settings.
	// Returns an error if the camera cannot be opened or initialized.
	Open() error

	// Grab captures a new frame from the camera.
	// This must be called before RetrieveImage to get fresh image data.
	// Returns an error if frame capture fails.
	Grab() error

	// RetrieveImage retrieves the left camera RGB image as RGBA bytes.
	// Grab must be called first to capture a new frame.
	// Returns an error if image retrieval fails.
	RetrieveImage() (*Image, error)

	// RetrieveImageView retrieves an image from a specific view (left, right, depth, etc.).
	// Grab must be called first to capture a new frame.
	// Returns an error if image retrieval fails.
	RetrieveImageView(view ViewType) (*Image, error)

	// RetrieveMeasure retrieves depth or other measurement data.
	// Grab must be called first to capture a new frame.
	// Returns an error if measure retrieval fails.
	RetrieveMeasure(measure MeasureType) (*Measure, error)

	// EnableObjectDetection initializes and starts the object detection module.
	// Returns an error if object detection cannot be started.
	EnableObjectDetection(params ObjectDetectionParameters) error

	// RetrieveObjects retrieves detected objects from the current frame.
	// Grab must be called first to capture a new frame.
	// Returns an error if object retrieval fails.
	RetrieveObjects() (*Objects, error)

	// DisableObjectDetection disables the object detection module.
	DisableObjectDetection() error

	// EnablePositionalTracking enables positional tracking.
	// This is required for object tracking with persistent IDs and for retrieving pose data.
	EnablePositionalTracking() error

	// GetPosition retrieves the current camera pose (position and orientation).
	// EnablePositionalTracking must be called first.
	// Returns an error if positional tracking is not enabled.
	GetPosition() (*Pose, error)

	// DisablePositionalTracking disables positional tracking.
	DisablePositionalTracking() error

	// EnableRecording starts recording to an SVO file.
	// Returns an error if recording cannot be started.
	EnableRecording(params RecordingParameters) error

	// DisableRecording stops recording and closes the SVO file.
	DisableRecording() error

	// GetRecordingStatus returns the current recording status.
	GetRecordingStatus() (*RecordingStatus, error)

	// PauseRecording pauses or resumes recording.
	// If pause is true, recording is paused. If false, recording is resumed.
	PauseRecording(pause bool) error

	// EnableBodyTracking initializes and starts the body tracking module.
	// Returns an error if body tracking cannot be started.
	EnableBodyTracking(params BodyTrackingParameters) error

	// RetrieveBodies retrieves detected bodies from the current frame.
	// Grab must be called first to capture a new frame.
	// Returns an error if body retrieval fails.
	RetrieveBodies() (*Bodies, error)

	// DisableBodyTracking disables the body tracking module.
	DisableBodyTracking() error

	// GetSetting returns the current value of a camera video setting.
	// Returns an error if the setting cannot be retrieved or is not supported.
	GetSetting(setting VideoSetting) (int, error)

	// SetSetting sets the value of a camera video setting.
	// Use VideoSettingValueAuto (-1) to enable automatic mode for supported settings.
	// Returns an error if the setting cannot be set or is not supported.
	SetSetting(setting VideoSetting, value int) error

	// GetSettingRange returns the min/max range for range-type settings.
	// This is used for VideoSettingAutoExposureTimeRange, VideoSettingAutoAnalogGainRange,
	// and VideoSettingAutoDigitalGainRange.
	// Returns an error if the setting is not a range type or is not supported.
	GetSettingRange(setting VideoSetting) (min, max int, err error)

	// SetSettingRange sets the min/max range for range-type settings.
	// This is used for VideoSettingAutoExposureTimeRange, VideoSettingAutoAnalogGainRange,
	// and VideoSettingAutoDigitalGainRange.
	// Returns an error if the setting is not a range type or is not supported.
	SetSettingRange(setting VideoSetting, min, max int) error

	// IsSettingSupported returns true if the given video setting is supported by this camera.
	// Some settings are only available on specific camera models (e.g., ZED X).
	IsSettingSupported(setting VideoSetting) bool

	// GetSerialNumber returns the serial number of the opened camera.
	// Returns 0 if the camera is not opened.
	GetSerialNumber() uint

	// Close closes the camera and releases all resources.
	// Returns an error if cleanup fails.
	Close() error
}

// Image represents a captured image from the ZED camera.
// The data is stored in RGBA format with 4 bytes per pixel.
// Implements the standard Go image.Image interface.
type Image struct {
	Width  int    // Width of the image in pixels
	Height int    // Height of the image in pixels
	Data   []byte // RGBA pixel data (4 bytes per pixel: R, G, B, A)
}

// BytesPerPixel returns the number of bytes per pixel (4 for RGBA).
func (img *Image) BytesPerPixel() int {
	return 4
}

// Size returns the total size of the image data in bytes.
func (img *Image) Size() int {
	return img.Width * img.Height * img.BytesPerPixel()
}

// ColorModel returns the Image's color model (RGBA).
// Implements image.Image interface.
func (img *Image) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the domain for which At can return non-zero color.
// Implements image.Image interface.
func (img *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, img.Width, img.Height)
}

// At returns the color of the pixel at (x, y).
// Implements image.Image interface.
func (img *Image) At(x, y int) color.Color {
	if x < 0 || x >= img.Width || y < 0 || y >= img.Height {
		return color.RGBA{}
	}

	offset := (y*img.Width + x) * 4
	return color.RGBA{
		R: img.Data[offset+0],
		G: img.Data[offset+1],
		B: img.Data[offset+2],
		A: img.Data[offset+3],
	}
}

// ToRGBA converts the ZED Image to a standard library image.RGBA.
// This creates a new image with a copy of the data.
func (img *Image) ToRGBA() *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	copy(rgba.Pix, img.Data)
	return rgba
}

// Measure represents depth or other measurement data from the ZED camera.
type Measure struct {
	Width  int       // Width of the measure in pixels
	Height int       // Height of the measure in pixels
	Data   []float32 // Measurement data (format depends on MeasureType)
	Type   MeasureType
}

// Size returns the total size of the measure data in bytes.
func (m *Measure) Size() int {
	return len(m.Data) * 4 // 4 bytes per float32
}

// ToGrayscaleImage converts the measure data to a grayscale image.
// Supported for MeasureDepth, MeasureDisparity, and MeasureConfidence.
// Returns an error for MeasureXYZ (point cloud data).
func (m *Measure) ToGrayscaleImage() (*image.Gray, error) {
	// Check if this is a point cloud
	if m.Type == MeasureXYZ {
		return nil, fmt.Errorf("grayscale visualization not supported for point cloud data (MeasureXYZ)")
	}

	// Find min and max values for normalization (excluding NaN and Inf)
	minVal := float32(math.Inf(1))
	maxVal := float32(math.Inf(-1))

	for _, val := range m.Data {
		if !math.IsNaN(float64(val)) && !math.IsInf(float64(val), 0) {
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	// Handle case where all values are invalid
	if math.IsInf(float64(minVal), 0) || math.IsInf(float64(maxVal), 0) {
		return nil, fmt.Errorf("no valid values found in measure data")
	}

	// Create grayscale image
	img := image.NewGray(image.Rect(0, 0, m.Width, m.Height))

	// Normalize and populate image
	valueRange := maxVal - minVal
	if valueRange == 0 {
		// All values are the same, use middle gray
		for i := range img.Pix {
			img.Pix[i] = 128
		}
	} else {
		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				idx := y*m.Width + x
				val := m.Data[idx]

				var grayVal uint8
				if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
					// Invalid values render as black
					grayVal = 0
				} else {
					// Normalize to 0-255 range
					normalized := (val - minVal) / valueRange
					grayVal = uint8(normalized * 255)
				}

				img.SetGray(x, y, color.Gray{Y: grayVal})
			}
		}
	}

	return img, nil
}

// ViewType represents different camera views available.
type ViewType int

const (
	// ViewLeft represents the left camera BGRA image
	ViewLeft ViewType = iota
	// ViewRight represents the right camera BGRA image
	ViewRight
	// ViewDepth represents a color rendering of depth
	ViewDepth
	// ViewConfidence represents a color rendering of depth confidence
	ViewConfidence
	// ViewNormals represents a color rendering of surface normals
	ViewNormals
	// ViewSideBySide represents left and right images side by side
	ViewSideBySide
)

// String returns the string representation of the view type.
func (v ViewType) String() string {
	switch v {
	case ViewLeft:
		return "LEFT"
	case ViewRight:
		return "RIGHT"
	case ViewDepth:
		return "DEPTH"
	case ViewConfidence:
		return "CONFIDENCE"
	case ViewNormals:
		return "NORMALS"
	case ViewSideBySide:
		return "SIDE_BY_SIDE"
	default:
		return fmt.Sprintf("Unknown(%d)", v)
	}
}

// MeasureType represents different measurement data types.
type MeasureType int

const (
	// MeasureDepth represents depth map in configured units
	MeasureDepth MeasureType = iota
	// MeasureDisparity represents disparity map
	MeasureDisparity
	// MeasureConfidence represents depth confidence/certainty
	MeasureConfidence
	// MeasureXYZ represents 3D point cloud (X, Y, Z)
	MeasureXYZ
)

// String returns the string representation of the measure type.
func (m MeasureType) String() string {
	switch m {
	case MeasureDepth:
		return "DEPTH"
	case MeasureDisparity:
		return "DISPARITY"
	case MeasureConfidence:
		return "CONFIDENCE"
	case MeasureXYZ:
		return "XYZ"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// Resolution represents the camera resolution settings.
type Resolution int

const (
	// ResolutionHD2K represents 2208x1242 resolution (2K)
	ResolutionHD2K Resolution = iota
	// ResolutionHD1080 represents 1920x1080 resolution (Full HD)
	ResolutionHD1080
	// ResolutionHD720 represents 1280x720 resolution (HD)
	ResolutionHD720
	// ResolutionVGA represents 672x376 resolution (VGA)
	ResolutionVGA
	// ResolutionHD1200 represents 1920x1200 resolution (ZED X Native)
	ResolutionHD1200
)

// String returns the string representation of the resolution.
func (r Resolution) String() string {
	switch r {
	case ResolutionHD2K:
		return "HD2K"
	case ResolutionHD1080:
		return "HD1080"
	case ResolutionHD720:
		return "HD720"
	case ResolutionVGA:
		return "VGA"
	case ResolutionHD1200:
		return "HD1200"
	default:
		return fmt.Sprintf("Unknown(%d)", r)
	}
}

// Dimensions returns the width and height for the resolution.
func (r Resolution) Dimensions() (width, height int) {
	switch r {
	case ResolutionHD2K:
		return 2208, 1242
	case ResolutionHD1080:
		return 1920, 1080
	case ResolutionHD720:
		return 1280, 720
	case ResolutionVGA:
		return 672, 376
	case ResolutionHD1200:
		return 1920, 1200
	default:
		return 0, 0
	}
}

// DepthMode represents the depth sensing mode.
type DepthMode int

const (
	// DepthModeNone disables depth sensing
	DepthModeNone DepthMode = iota
	// DepthModePerformance prioritizes speed over accuracy (traditional stereo)
	DepthModePerformance
	// DepthModeQuality balances speed and accuracy (traditional stereo)
	DepthModeQuality
	// DepthModeUltra prioritizes accuracy over speed (traditional stereo)
	DepthModeUltra
	// DepthModeNeuralLight uses AI-based depth estimation (light version)
	DepthModeNeuralLight
	// DepthModeNeural uses AI-based depth estimation (standard)
	DepthModeNeural
	// DepthModeNeuralPlus uses AI-based depth estimation (most accurate)
	DepthModeNeuralPlus
)

// String returns the string representation of the depth mode.
func (d DepthMode) String() string {
	switch d {
	case DepthModeNone:
		return "NONE"
	case DepthModePerformance:
		return "PERFORMANCE"
	case DepthModeQuality:
		return "QUALITY"
	case DepthModeUltra:
		return "ULTRA"
	case DepthModeNeuralLight:
		return "NEURAL_LIGHT"
	case DepthModeNeural:
		return "NEURAL"
	case DepthModeNeuralPlus:
		return "NEURAL_PLUS"
	default:
		return fmt.Sprintf("Unknown(%d)", d)
	}
}

// ObjectClass represents the class/category of detected objects.
type ObjectClass int

const (
	// ObjectClassPerson represents people detection
	ObjectClassPerson ObjectClass = iota
	// ObjectClassVehicle represents vehicle detection (cars, trucks, buses, motorcycles, etc.)
	ObjectClassVehicle
	// ObjectClassBag represents bag detection (backpack, handbag, suitcase, etc.)
	ObjectClassBag
	// ObjectClassAnimal represents animal detection (cow, sheep, horse, dog, cat, bird, etc.)
	ObjectClassAnimal
	// ObjectClassElectronics represents electronic device detection (cellphone, laptop, etc.)
	ObjectClassElectronics
	// ObjectClassFruitVegetable represents fruit and vegetable detection (banana, apple, orange, carrot, etc.)
	ObjectClassFruitVegetable
	// ObjectClassSport represents sport-related object detection (sport ball, etc.)
	ObjectClassSport
	// ObjectClassLast is used for array sizing
	ObjectClassLast
)

// String returns the string representation of the object class.
func (c ObjectClass) String() string {
	switch c {
	case ObjectClassPerson:
		return "PERSON"
	case ObjectClassVehicle:
		return "VEHICLE"
	case ObjectClassBag:
		return "BAG"
	case ObjectClassAnimal:
		return "ANIMAL"
	case ObjectClassElectronics:
		return "ELECTRONICS"
	case ObjectClassFruitVegetable:
		return "FRUIT_VEGETABLE"
	case ObjectClassSport:
		return "SPORT"
	default:
		return fmt.Sprintf("Unknown(%d)", c)
	}
}

// ObjectSubclass represents the subclass/subcategory of detected objects.
type ObjectSubclass int

const (
	// ObjectSubclassPerson represents a person
	ObjectSubclassPerson ObjectSubclass = iota
	// ObjectSubclassBicycle represents a bicycle
	ObjectSubclassBicycle
	// ObjectSubclassCar represents a car
	ObjectSubclassCar
	// ObjectSubclassMotorbike represents a motorbike
	ObjectSubclassMotorbike
	// ObjectSubclassBus represents a bus
	ObjectSubclassBus
	// ObjectSubclassTruck represents a truck
	ObjectSubclassTruck
	// ObjectSubclassBoat represents a boat
	ObjectSubclassBoat
	// ObjectSubclassBackpack represents a backpack
	ObjectSubclassBackpack
	// ObjectSubclassHandbag represents a handbag
	ObjectSubclassHandbag
	// ObjectSubclassSuitcase represents a suitcase
	ObjectSubclassSuitcase
	// ObjectSubclassBird represents a bird
	ObjectSubclassBird
	// ObjectSubclassCat represents a cat
	ObjectSubclassCat
	// ObjectSubclassDog represents a dog
	ObjectSubclassDog
	// ObjectSubclassHorse represents a horse
	ObjectSubclassHorse
	// ObjectSubclassSheep represents a sheep
	ObjectSubclassSheep
	// ObjectSubclassCow represents a cow
	ObjectSubclassCow
	// ObjectSubclassCellphone represents a cellphone
	ObjectSubclassCellphone
	// ObjectSubclassLaptop represents a laptop
	ObjectSubclassLaptop
	// ObjectSubclassBanana represents a banana
	ObjectSubclassBanana
	// ObjectSubclassApple represents an apple
	ObjectSubclassApple
	// ObjectSubclassOrange represents an orange
	ObjectSubclassOrange
	// ObjectSubclassCarrot represents a carrot
	ObjectSubclassCarrot
	// ObjectSubclassPersonHead represents a person's head
	ObjectSubclassPersonHead
	// ObjectSubclassSportsBall represents a sports ball (including tennis balls)
	ObjectSubclassSportsBall
	// ObjectSubclassMachinery represents machinery
	ObjectSubclassMachinery
	// ObjectSubclassLast is used for iteration
	ObjectSubclassLast
)

// String returns the string representation of the object subclass.
func (s ObjectSubclass) String() string {
	switch s {
	case ObjectSubclassPerson:
		return "PERSON"
	case ObjectSubclassBicycle:
		return "BICYCLE"
	case ObjectSubclassCar:
		return "CAR"
	case ObjectSubclassMotorbike:
		return "MOTORBIKE"
	case ObjectSubclassBus:
		return "BUS"
	case ObjectSubclassTruck:
		return "TRUCK"
	case ObjectSubclassBoat:
		return "BOAT"
	case ObjectSubclassBackpack:
		return "BACKPACK"
	case ObjectSubclassHandbag:
		return "HANDBAG"
	case ObjectSubclassSuitcase:
		return "SUITCASE"
	case ObjectSubclassBird:
		return "BIRD"
	case ObjectSubclassCat:
		return "CAT"
	case ObjectSubclassDog:
		return "DOG"
	case ObjectSubclassHorse:
		return "HORSE"
	case ObjectSubclassSheep:
		return "SHEEP"
	case ObjectSubclassCow:
		return "COW"
	case ObjectSubclassCellphone:
		return "CELLPHONE"
	case ObjectSubclassLaptop:
		return "LAPTOP"
	case ObjectSubclassBanana:
		return "BANANA"
	case ObjectSubclassApple:
		return "APPLE"
	case ObjectSubclassOrange:
		return "ORANGE"
	case ObjectSubclassCarrot:
		return "CARROT"
	case ObjectSubclassPersonHead:
		return "PERSON_HEAD"
	case ObjectSubclassSportsBall:
		return "SPORTS_BALL"
	case ObjectSubclassMachinery:
		return "MACHINERY"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// ObjectDetectionModel represents the object detection model to use.
type ObjectDetectionModel int

const (
	// ObjectDetectionModelMultiClassBoxFast uses fast bounding box detection
	ObjectDetectionModelMultiClassBoxFast ObjectDetectionModel = iota
	// ObjectDetectionModelMultiClassBoxMedium balances speed and accuracy
	ObjectDetectionModelMultiClassBoxMedium
	// ObjectDetectionModelMultiClassBoxAccurate prioritizes accuracy
	ObjectDetectionModelMultiClassBoxAccurate
)

// String returns the string representation of the detection model.
func (m ObjectDetectionModel) String() string {
	switch m {
	case ObjectDetectionModelMultiClassBoxFast:
		return "MULTI_CLASS_BOX_FAST"
	case ObjectDetectionModelMultiClassBoxMedium:
		return "MULTI_CLASS_BOX_MEDIUM"
	case ObjectDetectionModelMultiClassBoxAccurate:
		return "MULTI_CLASS_BOX_ACCURATE"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// ObjectTrackingState represents the tracking state of an object.
type ObjectTrackingState int

const (
	// ObjectTrackingStateOff indicates tracking is not initialized
	ObjectTrackingStateOff ObjectTrackingState = iota
	// ObjectTrackingStateOK indicates the object is tracked
	ObjectTrackingStateOK
	// ObjectTrackingStateSearching indicates the object is being searched for
	ObjectTrackingStateSearching
	// ObjectTrackingStateTerminate indicates tracking is about to be terminated
	ObjectTrackingStateTerminate
)

// String returns the string representation of the tracking state.
func (s ObjectTrackingState) String() string {
	switch s {
	case ObjectTrackingStateOff:
		return "OFF"
	case ObjectTrackingStateOK:
		return "OK"
	case ObjectTrackingStateSearching:
		return "SEARCHING"
	case ObjectTrackingStateTerminate:
		return "TERMINATE"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// ObjectActionState represents the action state of an object.
type ObjectActionState int

const (
	// ObjectActionStateIdle indicates the object is stationary
	ObjectActionStateIdle ObjectActionState = iota
	// ObjectActionStateMoving indicates the object is moving
	ObjectActionStateMoving
)

// String returns the string representation of the action state.
func (s ObjectActionState) String() string {
	switch s {
	case ObjectActionStateIdle:
		return "IDLE"
	case ObjectActionStateMoving:
		return "MOVING"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// ObjectDetectionParameters contains parameters for initializing object detection.
type ObjectDetectionParameters struct {
	// DetectionModel specifies which model to use for detection
	DetectionModel ObjectDetectionModel

	// EnableTracking enables object tracking across frames
	EnableTracking bool

	// EnableSegmentation enables object segmentation masks (requires more GPU)
	EnableSegmentation bool

	// DetectionConfidenceThreshold sets minimum confidence (0-100) for detections
	DetectionConfidenceThreshold int

	// ClassFilter specifies which object classes to detect (empty = all classes)
	ClassFilter []ObjectClass

	// InstanceModuleID allows multiple detection instances
	InstanceModuleID int
}

// DefaultObjectDetectionParameters returns parameters with sensible defaults.
func DefaultObjectDetectionParameters() ObjectDetectionParameters {
	return ObjectDetectionParameters{
		DetectionModel:               ObjectDetectionModelMultiClassBoxFast,
		EnableTracking:               true,
		EnableSegmentation:           false,
		DetectionConfidenceThreshold: 50,
		ClassFilter:                  []ObjectClass{}, // All classes
		InstanceModuleID:             0,
	}
}

// ObjectDetectionRuntimeParameters contains runtime parameters for object detection.
type ObjectDetectionRuntimeParameters struct {
	// DetectionConfidenceThreshold sets minimum confidence (0-100) for detections
	DetectionConfidenceThreshold int

	// ClassFilter specifies which object classes to detect at runtime
	ClassFilter []ObjectClass
}

// BoundingBox2D represents a 2D bounding box in image coordinates.
type BoundingBox2D struct {
	X      int // Top-left X coordinate
	Y      int // Top-left Y coordinate
	Width  int // Width of the box
	Height int // Height of the box
}

// Vector3 represents a 3D vector or position.
type Vector3 struct {
	X float32
	Y float32
	Z float32
}

// ObjectData contains all data for a detected object.
type ObjectData struct {
	// ID is the unique identifier for tracking
	ID int

	// Label is the object class
	Label ObjectClass

	// Sublabel is the object subclass
	Sublabel ObjectSubclass

	// TrackingState indicates the tracking status
	TrackingState ObjectTrackingState

	// ActionState indicates if object is moving
	ActionState ObjectActionState

	// Position is the 3D position in camera coordinates (meters)
	Position Vector3

	// Velocity is the 3D velocity (meters/second)
	Velocity Vector3

	// BoundingBox2D is the 2D bounding box in image coordinates
	BoundingBox2D BoundingBox2D

	// Confidence is the detection confidence (0-100)
	Confidence float32

	// Dimensions represents the 3D bounding box dimensions (width, height, depth in meters)
	Dimensions Vector3
}

// Objects contains all detected objects from a frame.
type Objects struct {
	// Timestamp of the detection
	Timestamp uint64

	// ObjectList contains all detected objects
	ObjectList []ObjectData

	// IsNew indicates if this is new detection data
	IsNew bool

	// IsTracked indicates if tracking is enabled
	IsTracked bool
}

// SVOCompressionMode represents compression modes for SVO recording.
type SVOCompressionMode int

const (
	// SVOCompressionLossless uses PNG/ZSTD lossless compression (42% of raw size, CPU-based)
	SVOCompressionLossless SVOCompressionMode = iota
	// SVOCompressionH264 uses H264 compression (1% of raw size, GPU-based)
	SVOCompressionH264
	// SVOCompressionH265 uses H265 compression (1% of raw size, GPU-based)
	SVOCompressionH265
	// SVOCompressionH264Lossless uses H264 lossless compression (25% of raw size, GPU-based)
	SVOCompressionH264Lossless
	// SVOCompressionH265Lossless uses H265 lossless compression (25% of raw size, GPU-based)
	SVOCompressionH265Lossless
)

// String returns the string representation of the compression mode.
func (m SVOCompressionMode) String() string {
	switch m {
	case SVOCompressionLossless:
		return "LOSSLESS"
	case SVOCompressionH264:
		return "H264"
	case SVOCompressionH265:
		return "H265"
	case SVOCompressionH264Lossless:
		return "H264_LOSSLESS"
	case SVOCompressionH265Lossless:
		return "H265_LOSSLESS"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// RecordingParameters contains parameters for SVO recording.
type RecordingParameters struct {
	// Filename is the path to the SVO file to create
	Filename string

	// CompressionMode specifies the compression algorithm to use
	CompressionMode SVOCompressionMode

	// Bitrate is the target bitrate in kbps (0 for default)
	// Only used for H264/H265 lossy compression
	Bitrate uint

	// TargetFPS is the target frame rate for recording (0 for camera FPS)
	TargetFPS int

	// Transcode enables re-encoding for streaming inputs
	// Set to false to avoid decoding/re-encoding when converting streams to SVO
	Transcode bool
}

// RecordingStatus contains the current status of SVO recording.
type RecordingStatus struct {
	// IsRecording indicates if recording is currently enabled
	IsRecording bool

	// IsPaused indicates if recording is currently paused
	IsPaused bool

	// Status indicates if the current frame was successfully written
	Status bool

	// CurrentCompressionTime is the compression time for the current frame in milliseconds
	CurrentCompressionTime float64

	// CurrentCompressionRatio is the compression ratio (% of raw size) for the current frame
	CurrentCompressionRatio float64

	// AverageCompressionTime is the average compression time since recording started
	AverageCompressionTime float64

	// AverageCompressionRatio is the average compression ratio since recording started
	AverageCompressionRatio float64
}

// BodyFormat represents the body skeleton format.
type BodyFormat int

const (
	// BodyFormat18 represents 18 keypoints (COCO18 skeleton)
	BodyFormat18 BodyFormat = iota
	// BodyFormat34 represents 34 keypoints
	BodyFormat34
	// BodyFormat38 represents 38 keypoints (most detailed)
	BodyFormat38
)

// String returns the string representation of the body format.
func (b BodyFormat) String() string {
	switch b {
	case BodyFormat18:
		return "BODY_18"
	case BodyFormat34:
		return "BODY_34"
	case BodyFormat38:
		return "BODY_38"
	default:
		return fmt.Sprintf("Unknown(%d)", b)
	}
}

// NumKeypoints returns the number of keypoints for this body format.
func (b BodyFormat) NumKeypoints() int {
	switch b {
	case BodyFormat18:
		return 18
	case BodyFormat34:
		return 34
	case BodyFormat38:
		return 38
	default:
		return 0
	}
}

// BodyTrackingModel represents the body tracking model to use.
type BodyTrackingModel int

const (
	// BodyTrackingModelHumanBodyFast uses fast body detection
	BodyTrackingModelHumanBodyFast BodyTrackingModel = iota
	// BodyTrackingModelHumanBodyMedium balances speed and accuracy
	BodyTrackingModelHumanBodyMedium
	// BodyTrackingModelHumanBodyAccurate prioritizes accuracy
	BodyTrackingModelHumanBodyAccurate
)

// String returns the string representation of the body tracking model.
func (m BodyTrackingModel) String() string {
	switch m {
	case BodyTrackingModelHumanBodyFast:
		return "HUMAN_BODY_FAST"
	case BodyTrackingModelHumanBodyMedium:
		return "HUMAN_BODY_MEDIUM"
	case BodyTrackingModelHumanBodyAccurate:
		return "HUMAN_BODY_ACCURATE"
	default:
		return fmt.Sprintf("Unknown(%d)", m)
	}
}

// Keypoint2D represents a 2D keypoint in image coordinates.
type Keypoint2D struct {
	X float32 // X coordinate in pixels
	Y float32 // Y coordinate in pixels
}

// Keypoint3D represents a 3D keypoint in camera coordinates.
type Keypoint3D struct {
	X float32 // X coordinate in meters
	Y float32 // Y coordinate in meters
	Z float32 // Z coordinate in meters
}

// Quaternion represents a rotation as a quaternion.
type Quaternion struct {
	X float32
	Y float32
	Z float32
	W float32
}

// Matrix3x3 represents a 3x3 rotation matrix.
type Matrix3x3 [3][3]float32

// Matrix4x4 represents a 4x4 transformation matrix.
type Matrix4x4 [4][4]float32

// TrackingState represents the tracking state of positional tracking.
type TrackingState int

const (
	// TrackingStateSearching indicates the tracking is searching for a match
	TrackingStateSearching TrackingState = iota
	// TrackingStateOK indicates tracking is working normally
	TrackingStateOK
	// TrackingStateOff indicates tracking is not enabled
	TrackingStateOff
	// TrackingStateFPSTooLow indicates FPS is too low for tracking
	TrackingStateFPSTooLow
)

// String returns the string representation of the tracking state.
func (s TrackingState) String() string {
	switch s {
	case TrackingStateSearching:
		return "SEARCHING"
	case TrackingStateOK:
		return "OK"
	case TrackingStateOff:
		return "OFF"
	case TrackingStateFPSTooLow:
		return "FPS_TOO_LOW"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// ReferenceFrame represents the reference frame for pose data.
type ReferenceFrame int

const (
	// ReferenceFrameWorld uses the world frame (fixed in space at tracking start)
	ReferenceFrameWorld ReferenceFrame = iota
	// ReferenceFrameCamera uses the camera frame
	ReferenceFrameCamera
)

// String returns the string representation of the reference frame.
func (r ReferenceFrame) String() string {
	switch r {
	case ReferenceFrameWorld:
		return "WORLD"
	case ReferenceFrameCamera:
		return "CAMERA"
	default:
		return fmt.Sprintf("Unknown(%d)", r)
	}
}

// Pose represents the camera's position and orientation in 3D space.
// This provides 6DoF (6 degrees of freedom) tracking information.
type Pose struct {
	// Valid indicates if the pose is valid
	Valid bool

	// Timestamp of the pose in nanoseconds
	Timestamp uint64

	// Position is the 3D position of the camera in meters [X, Y, Z]
	Position Vector3

	// Orientation is the camera orientation as a quaternion [X, Y, Z, W]
	Orientation Quaternion

	// Rotation is the 3x3 rotation matrix
	Rotation Matrix3x3

	// Translation is the 3D translation vector (same as Position)
	Translation Vector3

	// EulerAngles contains the Euler angles in radians [Roll, Pitch, Yaw]
	EulerAngles Vector3

	// PoseConfidence indicates the confidence in the pose (0-100)
	// Higher values indicate more reliable tracking
	PoseConfidence float32

	// TrackingState indicates the current tracking state
	TrackingState TrackingState
}

// BodyTrackingParameters contains parameters for initializing body tracking.
type BodyTrackingParameters struct {
	// DetectionModel specifies which model to use for detection
	DetectionModel BodyTrackingModel

	// EnableTracking enables body tracking across frames
	EnableTracking bool

	// EnableSegmentation enables body segmentation masks (requires more GPU)
	EnableSegmentation bool

	// BodyFormat specifies the skeleton format (18, 34, or 38 keypoints)
	BodyFormat BodyFormat

	// DetectionConfidenceThreshold sets minimum confidence (0-100) for detections
	DetectionConfidenceThreshold int

	// InstanceModuleID allows multiple tracking instances
	InstanceModuleID int

	// EnableBodyFitting enables 3D body fitting with joint rotations
	EnableBodyFitting bool
}

// DefaultBodyTrackingParameters returns parameters with sensible defaults.
func DefaultBodyTrackingParameters() BodyTrackingParameters {
	return BodyTrackingParameters{
		DetectionModel:               BodyTrackingModelHumanBodyFast,
		EnableTracking:               true,
		EnableSegmentation:           false,
		BodyFormat:                   BodyFormat18,
		DetectionConfidenceThreshold: 50,
		InstanceModuleID:             0,
		EnableBodyFitting:            false,
	}
}

// BodyTrackingRuntimeParameters contains runtime parameters for body tracking.
type BodyTrackingRuntimeParameters struct {
	// DetectionConfidenceThreshold sets minimum confidence (0-100) for detections
	DetectionConfidenceThreshold int
}

// BodyData contains all data for a detected body.
type BodyData struct {
	// ID is the unique identifier for tracking
	ID int

	// UniqueObjectID is a unique string ID for the body
	UniqueObjectID string

	// TrackingState indicates the tracking status
	TrackingState ObjectTrackingState

	// ActionState indicates if body is moving
	ActionState ObjectActionState

	// Position is the 3D position of the body center (meters)
	Position Vector3

	// Velocity is the 3D velocity (meters/second)
	Velocity Vector3

	// BoundingBox2D is the 2D bounding box in image coordinates
	BoundingBox2D BoundingBox2D

	// Confidence is the detection confidence (0-100)
	Confidence float32

	// Dimensions represents the 3D bounding box dimensions (width, height, depth in meters)
	Dimensions Vector3

	// Keypoints2D contains 2D keypoint positions in image coordinates
	Keypoints2D []Keypoint2D

	// Keypoints3D contains 3D keypoint positions in camera coordinates
	Keypoints3D []Keypoint3D

	// KeypointConfidence contains per-keypoint detection confidence (0-100)
	KeypointConfidence []float32

	// HeadBoundingBox2D is the 2D bounding box for the head
	HeadBoundingBox2D BoundingBox2D

	// HeadPosition is the 3D position of the head center
	HeadPosition Vector3

	// LocalPositionPerJoint contains local position of each keypoint relative to parent
	LocalPositionPerJoint []Vector3

	// LocalOrientationPerJoint contains local rotation of each joint as quaternion
	LocalOrientationPerJoint []Quaternion

	// GlobalRootOrientation is the global orientation of the body root
	GlobalRootOrientation Quaternion
}

// Bodies contains all detected bodies from a frame.
type Bodies struct {
	// Timestamp of the detection
	Timestamp uint64

	// BodyList contains all detected bodies
	BodyList []BodyData

	// IsNew indicates if this is new detection data
	IsNew bool

	// IsTracked indicates if tracking is enabled
	IsTracked bool
}

// VideoSetting represents camera video settings that can be adjusted.
// These correspond to the ZED SDK VIDEO_SETTINGS enum.
type VideoSetting int

// VideoSettingValueAuto is the value to pass to SetSetting to enable automatic mode
// for settings that support it (e.g., Gain, Exposure).
const VideoSettingValueAuto = -1

const (
	// VideoSettingBrightness controls brightness (0-8).
	// Not available for ZED X/X Mini cameras.
	VideoSettingBrightness VideoSetting = iota
	// VideoSettingContrast controls contrast (0-8).
	// Not available for ZED X/X Mini cameras.
	VideoSettingContrast
	// VideoSettingHue controls hue (0-11).
	// Not available for ZED X/X Mini cameras.
	VideoSettingHue
	// VideoSettingSaturation controls saturation (0-8).
	VideoSettingSaturation
	// VideoSettingSharpness controls digital sharpening (0-8).
	VideoSettingSharpness
	// VideoSettingGamma controls ISP gamma (1-9).
	VideoSettingGamma
	// VideoSettingGain controls gain (0-100, or -1 for auto).
	// If Exposure is set to auto (-1), Gain will also be automatic.
	VideoSettingGain
	// VideoSettingExposure controls exposure (0-100, or -1 for auto).
	// The value is mapped linearly to actual exposure time based on FPS.
	VideoSettingExposure
	// VideoSettingAECAGC controls automatic exposure and gain (0=manual, 1=auto).
	// Setting Gain or Exposure values will automatically set this to 0.
	VideoSettingAECAGC
	// VideoSettingWhiteBalanceTemperature controls color temperature (2800-6500).
	// Setting this will automatically disable auto white balance.
	VideoSettingWhiteBalanceTemperature
	// VideoSettingWhiteBalanceAuto controls automatic white balance (0=manual, 1=auto).
	VideoSettingWhiteBalanceAuto
	// VideoSettingLEDStatus controls the front LED (0=off, 1=on).
	// Requires camera firmware 1523 at least.
	VideoSettingLEDStatus
	// VideoSettingExposureTime controls real exposure time in microseconds.
	// Only available for ZED X/X Mini cameras. Replaces VideoSettingExposure.
	VideoSettingExposureTime
	// VideoSettingAnalogGain controls real analog gain in mDB (1000-16000 default range).
	// Only available for ZED X/X Mini cameras. Replaces VideoSettingGain.
	VideoSettingAnalogGain
	// VideoSettingDigitalGain controls real digital gain as a factor (1-256 default range).
	// Only available for ZED X/X Mini cameras. Replaces VideoSettingGain.
	VideoSettingDigitalGain
	// VideoSettingAutoExposureTimeRange is used with GetSettingRange/SetSettingRange
	// to control the auto exposure time range in microseconds.
	// Only available for ZED X/X Mini cameras.
	VideoSettingAutoExposureTimeRange
	// VideoSettingAutoAnalogGainRange is used with GetSettingRange/SetSettingRange
	// to control the auto analog gain range in mDB.
	// Only available for ZED X/X Mini cameras.
	VideoSettingAutoAnalogGainRange
	// VideoSettingAutoDigitalGainRange is used with GetSettingRange/SetSettingRange
	// to control the auto digital gain range.
	// Only available for ZED X/X Mini cameras.
	VideoSettingAutoDigitalGainRange
	// VideoSettingExposureCompensation controls exposure compensation (0-100, mapped to -2.0 to +2.0 f-stops).
	// Default is 50 (no compensation). Only available for ZED X/X Mini cameras.
	VideoSettingExposureCompensation
	// VideoSettingDenoising controls denoising level (0-100, default 50).
	// Only available for ZED X/X Mini cameras.
	VideoSettingDenoising
	// VideoSettingSceneIlluminance returns the scene illuminance level (read-only).
	// Value provided in 0.1x Lux. Only available for ZED X/X Mini cameras.
	VideoSettingSceneIlluminance
)

// String returns the string representation of the video setting.
func (v VideoSetting) String() string {
	switch v {
	case VideoSettingBrightness:
		return "BRIGHTNESS"
	case VideoSettingContrast:
		return "CONTRAST"
	case VideoSettingHue:
		return "HUE"
	case VideoSettingSaturation:
		return "SATURATION"
	case VideoSettingSharpness:
		return "SHARPNESS"
	case VideoSettingGamma:
		return "GAMMA"
	case VideoSettingGain:
		return "GAIN"
	case VideoSettingExposure:
		return "EXPOSURE"
	case VideoSettingAECAGC:
		return "AEC_AGC"
	case VideoSettingWhiteBalanceTemperature:
		return "WHITEBALANCE_TEMPERATURE"
	case VideoSettingWhiteBalanceAuto:
		return "WHITEBALANCE_AUTO"
	case VideoSettingLEDStatus:
		return "LED_STATUS"
	case VideoSettingExposureTime:
		return "EXPOSURE_TIME"
	case VideoSettingAnalogGain:
		return "ANALOG_GAIN"
	case VideoSettingDigitalGain:
		return "DIGITAL_GAIN"
	case VideoSettingAutoExposureTimeRange:
		return "AUTO_EXPOSURE_TIME_RANGE"
	case VideoSettingAutoAnalogGainRange:
		return "AUTO_ANALOG_GAIN_RANGE"
	case VideoSettingAutoDigitalGainRange:
		return "AUTO_DIGITAL_GAIN_RANGE"
	case VideoSettingExposureCompensation:
		return "EXPOSURE_COMPENSATION"
	case VideoSettingDenoising:
		return "DENOISING"
	case VideoSettingSceneIlluminance:
		return "SCENE_ILLUMINANCE"
	default:
		return fmt.Sprintf("Unknown(%d)", v)
	}
}

// Config contains configuration parameters for opening a ZED camera.
type Config struct {
	// Resolution sets the camera resolution
	Resolution Resolution

	// FPS sets the target frame rate (0 for automatic)
	FPS int

	// DepthMode configures the depth sensing mode
	DepthMode DepthMode

	// CameraID is the camera device ID (0 for first camera).
	// Used only if SerialNumber is 0.
	CameraID int

	// SerialNumber is the unique serial number of the camera.
	// If non-zero, the camera will be opened by serial number instead of CameraID.
	// This provides stable identification across reboots.
	SerialNumber uint

	// MaxExposureTime is the maximum exposure time.
	// Only applicable for ZED X/X Mini cameras.
	MaxExposureTime time.Duration

	// Logger for camera operations (optional)
	Logger *slog.Logger
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		Resolution: ResolutionHD1200,
		FPS:        30,
		DepthMode:  DepthModeNeural,
		CameraID:   0,
		Logger:     slog.Default(),
	}
}
