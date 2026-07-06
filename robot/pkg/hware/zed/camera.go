//go:build zed_sdk
// +build zed_sdk

package zed

/*
#cgo LDFLAGS: -L/usr/local/zed/lib -lsl_zed_c -L/usr/local/cuda/targets/aarch64-linux/lib -lcuda -lcudart
#cgo CFLAGS: -I/usr/local/zed/include -I/usr/local/cuda/targets/aarch64-linux/include

#include <sl/c_api/zed_interface.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"log/slog"
	"unsafe"
)

// ProdCamera is the production implementation of the Camera interface using the ZED SDK C API.
type ProdCamera struct {
	cameraID              C.int
	config                Config
	logger                *slog.Logger
	width                 int
	height                int
	serialNumber          uint
	objectDetectionParams ObjectDetectionParameters
	bodyTrackingParams    BodyTrackingParameters
}

var _ Camera = (*ProdCamera)(nil)

// resolutionToC converts Go Resolution enum to C API enum value
func resolutionToC(r Resolution) C.enum_SL_RESOLUTION {
	switch r {
	case ResolutionHD2K:
		return C.SL_RESOLUTION_HD2K
	case ResolutionHD1080:
		return C.SL_RESOLUTION_HD1080
	case ResolutionHD720:
		return C.SL_RESOLUTION_HD720
	case ResolutionVGA:
		return C.SL_RESOLUTION_VGA
	case ResolutionHD1200:
		return C.SL_RESOLUTION_HD1200
	default:
		return C.SL_RESOLUTION_HD720 // Default to HD720
	}
}

// depthModeToC converts Go DepthMode enum to C API enum value
func depthModeToC(d DepthMode) C.enum_SL_DEPTH_MODE {
	switch d {
	case DepthModeNone:
		return C.SL_DEPTH_MODE_NONE
	case DepthModePerformance:
		return C.SL_DEPTH_MODE_PERFORMANCE
	case DepthModeQuality:
		return C.SL_DEPTH_MODE_QUALITY
	case DepthModeUltra:
		return C.SL_DEPTH_MODE_ULTRA
	case DepthModeNeuralLight:
		return C.SL_DEPTH_MODE_NEURAL_LIGHT
	case DepthModeNeural:
		return C.SL_DEPTH_MODE_NEURAL
	case DepthModeNeuralPlus:
		return C.SL_DEPTH_MODE_NEURAL_PLUS
	default:
		return C.SL_DEPTH_MODE_NEURAL // Default to Neural for better quality
	}
}

// viewTypeToC converts Go ViewType enum to C API enum value
func viewTypeToC(v ViewType) C.enum_SL_VIEW {
	switch v {
	case ViewLeft:
		return C.SL_VIEW_LEFT
	case ViewRight:
		return C.SL_VIEW_RIGHT
	case ViewDepth:
		return C.SL_VIEW_DEPTH
	case ViewConfidence:
		return C.SL_VIEW_CONFIDENCE
	case ViewNormals:
		return C.SL_VIEW_NORMALS
	case ViewSideBySide:
		return C.SL_VIEW_SIDE_BY_SIDE
	default:
		return C.SL_VIEW_LEFT // Default to left
	}
}

// measureTypeToC converts Go MeasureType enum to C API enum value
func measureTypeToC(m MeasureType) C.enum_SL_MEASURE {
	switch m {
	case MeasureDepth:
		return C.SL_MEASURE_DEPTH
	case MeasureDisparity:
		return C.SL_MEASURE_DISPARITY
	case MeasureConfidence:
		return C.SL_MEASURE_CONFIDENCE
	case MeasureXYZ:
		return C.SL_MEASURE_XYZ
	default:
		return C.SL_MEASURE_DEPTH // Default to depth
	}
}

// objectDetectionModelToC converts Go ObjectDetectionModel to C API enum value
func objectDetectionModelToC(m ObjectDetectionModel) C.enum_SL_OBJECT_DETECTION_MODEL {
	switch m {
	case ObjectDetectionModelMultiClassBoxFast:
		return C.SL_OBJECT_DETECTION_MODEL_MULTI_CLASS_BOX_FAST
	case ObjectDetectionModelMultiClassBoxMedium:
		return C.SL_OBJECT_DETECTION_MODEL_MULTI_CLASS_BOX_MEDIUM
	case ObjectDetectionModelMultiClassBoxAccurate:
		return C.SL_OBJECT_DETECTION_MODEL_MULTI_CLASS_BOX_ACCURATE
	default:
		return C.SL_OBJECT_DETECTION_MODEL_MULTI_CLASS_BOX_FAST
	}
}

// objectClassToC converts Go ObjectClass to C API enum value
func objectClassToC(c ObjectClass) C.enum_SL_OBJECT_CLASS {
	return C.enum_SL_OBJECT_CLASS(c)
}

// objectClassFromC converts C API enum to Go ObjectClass
func objectClassFromC(c C.enum_SL_OBJECT_CLASS) ObjectClass {
	return ObjectClass(c)
}

// objectSubclassFromC converts C API enum to Go ObjectSubclass
func objectSubclassFromC(s C.enum_SL_OBJECT_SUBCLASS) ObjectSubclass {
	return ObjectSubclass(s)
}

// objectTrackingStateFromC converts C API enum to Go ObjectTrackingState
func objectTrackingStateFromC(s C.enum_SL_OBJECT_TRACKING_STATE) ObjectTrackingState {
	return ObjectTrackingState(s)
}

// objectActionStateFromC converts C API enum to Go ObjectActionState
func objectActionStateFromC(s C.enum_SL_OBJECT_ACTION_STATE) ObjectActionState {
	return ObjectActionState(s)
}

// bodyFormatToC converts Go BodyFormat to C API enum value
func bodyFormatToC(f BodyFormat) C.enum_SL_BODY_FORMAT {
	switch f {
	case BodyFormat18:
		return C.SL_BODY_FORMAT_BODY_18
	case BodyFormat34:
		return C.SL_BODY_FORMAT_BODY_34
	case BodyFormat38:
		return C.SL_BODY_FORMAT_BODY_38
	default:
		return C.SL_BODY_FORMAT_BODY_18
	}
}

// bodyTrackingModelToC converts Go BodyTrackingModel to C API enum value
func bodyTrackingModelToC(m BodyTrackingModel) C.enum_SL_BODY_TRACKING_MODEL {
	switch m {
	case BodyTrackingModelHumanBodyFast:
		return C.SL_BODY_TRACKING_MODEL_HUMAN_BODY_FAST
	case BodyTrackingModelHumanBodyMedium:
		return C.SL_BODY_TRACKING_MODEL_HUMAN_BODY_MEDIUM
	case BodyTrackingModelHumanBodyAccurate:
		return C.SL_BODY_TRACKING_MODEL_HUMAN_BODY_ACCURATE
	default:
		return C.SL_BODY_TRACKING_MODEL_HUMAN_BODY_FAST
	}
}

// videoSettingToC converts Go VideoSetting to C API enum value
func videoSettingToC(s VideoSetting) C.enum_SL_VIDEO_SETTINGS {
	switch s {
	case VideoSettingBrightness:
		return C.SL_VIDEO_SETTINGS_BRIGHTNESS
	case VideoSettingContrast:
		return C.SL_VIDEO_SETTINGS_CONTRAST
	case VideoSettingHue:
		return C.SL_VIDEO_SETTINGS_HUE
	case VideoSettingSaturation:
		return C.SL_VIDEO_SETTINGS_SATURATION
	case VideoSettingSharpness:
		return C.SL_VIDEO_SETTINGS_SHARPNESS
	case VideoSettingGamma:
		return C.SL_VIDEO_SETTINGS_GAMMA
	case VideoSettingGain:
		return C.SL_VIDEO_SETTINGS_GAIN
	case VideoSettingExposure:
		return C.SL_VIDEO_SETTINGS_EXPOSURE
	case VideoSettingAECAGC:
		return C.SL_VIDEO_SETTINGS_AEC_AGC
	case VideoSettingWhiteBalanceTemperature:
		return C.SL_VIDEO_SETTINGS_WHITEBALANCE_TEMPERATURE
	case VideoSettingWhiteBalanceAuto:
		return C.SL_VIDEO_SETTINGS_WHITEBALANCE_AUTO
	case VideoSettingLEDStatus:
		return C.SL_VIDEO_SETTINGS_LED_STATUS
	case VideoSettingExposureTime:
		return C.SL_VIDEO_SETTINGS_EXPOSURE_TIME
	case VideoSettingAnalogGain:
		return C.SL_VIDEO_SETTINGS_ANALOG_GAIN
	case VideoSettingDigitalGain:
		return C.SL_VIDEO_SETTINGS_DIGITAL_GAIN
	case VideoSettingAutoExposureTimeRange:
		return C.SL_VIDEO_SETTINGS_AUTO_EXPOSURE_TIME_RANGE
	case VideoSettingAutoAnalogGainRange:
		return C.SL_VIDEO_SETTINGS_AUTO_ANALOG_GAIN_RANGE
	case VideoSettingAutoDigitalGainRange:
		return C.SL_VIDEO_SETTINGS_AUTO_DIGITAL_GAIN_RANGE
	case VideoSettingExposureCompensation:
		return C.SL_VIDEO_SETTINGS_EXPOSURE_COMPENSATION
	case VideoSettingDenoising:
		return C.SL_VIDEO_SETTINGS_DENOISING
	case VideoSettingSceneIlluminance:
		return C.SL_VIDEO_SETTINGS_SCENE_ILLUMINANCE
	default:
		return C.SL_VIDEO_SETTINGS_BRIGHTNESS
	}
}

// NewCamera creates a new production ZED camera instance with the given configuration.
func NewCamera(cfg Config) *ProdCamera {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	logger := cfg.Logger.With("system", "zed")

	return &ProdCamera{
		cameraID: -1, // Will be set in Open
		config:   cfg,
		logger:   logger,
	}
}

// Open initializes and opens the ZED camera with the configured settings.
func (c *ProdCamera) Open() error {
	c.logger.Info("Opening ZED camera", "resolution", c.config.Resolution, "fps", c.config.FPS,
		"serial_number", c.config.SerialNumber, "camera_id", c.config.CameraID)

	// Create camera instance
	c.cameraID = C.int(c.config.CameraID)
	created := C.sl_create_camera(c.cameraID)
	if !created {
		return fmt.Errorf("failed to create camera: camera ID %d", c.cameraID)
	}

	// Create initialization parameters
	var initParams C.struct_SL_InitParameters

	// Set resolution
	initParams.resolution = resolutionToC(c.config.Resolution)

	// Set FPS
	if c.config.FPS > 0 {
		initParams.camera_fps = C.int(c.config.FPS)
	} else {
		initParams.camera_fps = 30
	}

	// Set other parameters
	initParams.input_type = C.SL_INPUT_TYPE_USB
	initParams.camera_device_id = c.cameraID
	initParams.camera_image_flip = C.SL_FLIP_MODE_AUTO
	initParams.camera_disable_self_calib = C.bool(false)
	initParams.enable_image_enhancement = C.bool(true)
	initParams.svo_real_time_mode = C.bool(true)
	initParams.depth_mode = depthModeToC(c.config.DepthMode)
	initParams.depth_stabilization = 30
	initParams.depth_maximum_distance = 40
	initParams.depth_minimum_distance = -1
	initParams.coordinate_unit = C.SL_UNIT_METER
	initParams.coordinate_system = C.SL_COORDINATE_SYSTEM_LEFT_HANDED_Y_UP
	initParams.sdk_gpu_id = -1
	initParams.sdk_verbose = 1                 // Enable verbose logging
	initParams.sensors_required = C.bool(true) // Required for object detection
	initParams.enable_right_side_measure = C.bool(false)
	initParams.open_timeout_sec = 5.0
	initParams.async_grab_camera_recovery = C.bool(false)
	initParams.grab_compute_capping_fps = 0
	initParams.enable_image_validity_check = C.bool(false)

	// Open the camera - use serial number if provided, otherwise use camera ID
	serialNum := C.uint(c.config.SerialNumber)
	errCode := C.sl_open_camera(c.cameraID, &initParams, serialNum, C.CString(""), C.CString(""), 0, C.CString(""), C.CString(""), C.CString(""))
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		C.sl_close_camera(c.cameraID)
		return fmt.Errorf("failed to open camera: error code %d", int(errCode))
	}

	// Get actual resolution dimensions from camera
	c.width = int(C.sl_get_width(c.cameraID))
	c.height = int(C.sl_get_height(c.cameraID))

	// Get and store the serial number
	c.serialNumber = uint(C.sl_get_zed_serial(c.cameraID))

	c.logger.Info("ZED camera opened successfully",
		"width", c.width,
		"height", c.height,
		"fps", c.config.FPS,
		"serial_number", c.serialNumber)

	// Apply MaxExposureTime if set (ZED X only)
	if c.config.MaxExposureTime > 0 {
		if c.IsSettingSupported(VideoSettingAutoExposureTimeRange) {
			maxUs := int(c.config.MaxExposureTime.Microseconds())
			c.logger.Info("Setting auto exposure time range", "max", c.config.MaxExposureTime, "max_us", maxUs)
			if err := c.SetSettingRange(VideoSettingAutoExposureTimeRange, 100, maxUs); err != nil {
				return fmt.Errorf("failed to set auto exposure time range: %w", err)
			}
		} else {
			c.logger.Warn("MaxExposureTime set but VideoSettingAutoExposureTimeRange is not supported by this camera")
		}
	}

	return nil
}

// EnablePositionalTracking enables the positional tracking module.
// This is required for object tracking to work with persistent IDs.
func (c *ProdCamera) EnablePositionalTracking() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Enabling positional tracking")

	// Create positional tracking parameters
	var trackingParams C.struct_SL_PositionalTrackingParameters

	// Set identity quaternion for initial rotation
	trackingParams.initial_world_rotation.w = 1.0
	trackingParams.initial_world_rotation.x = 0.0
	trackingParams.initial_world_rotation.y = 0.0
	trackingParams.initial_world_rotation.z = 0.0

	// Set zero vector for initial position
	trackingParams.initial_world_position.x = 0.0
	trackingParams.initial_world_position.y = 0.0
	trackingParams.initial_world_position.z = 0.0

	trackingParams.enable_area_memory = C.bool(true)
	trackingParams.enable_pose_smoothing = C.bool(false)
	trackingParams.set_floor_as_origin = C.bool(false)
	trackingParams.set_as_static = C.bool(false)

	// Enable positional tracking
	errCode := C.sl_enable_positional_tracking(c.cameraID, &trackingParams, nil)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to enable positional tracking: error code %d", int(errCode))
	}

	c.logger.Info("Positional tracking enabled successfully")
	return nil
}

// GetPosition retrieves the current camera pose (position and orientation).
func (c *ProdCamera) GetPosition() (*Pose, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	// Create quaternion and translation structures to receive the data
	var cRotation C.struct_SL_Quaternion
	var cTranslation C.struct_SL_Vector3

	// Get the pose in the world reference frame
	state := C.sl_get_position(c.cameraID, &cRotation, &cTranslation, C.SL_REFERENCE_FRAME_WORLD)

	// Convert C pose to Go pose
	pose := &Pose{
		Valid:     true,
		Timestamp: uint64(C.sl_get_current_timestamp(c.cameraID)),
		Position: Vector3{
			X: float32(cTranslation.x),
			Y: float32(cTranslation.y),
			Z: float32(cTranslation.z),
		},
		Orientation: Quaternion{
			X: float32(cRotation.x),
			Y: float32(cRotation.y),
			Z: float32(cRotation.z),
			W: float32(cRotation.w),
		},
		Translation: Vector3{
			X: float32(cTranslation.x),
			Y: float32(cTranslation.y),
			Z: float32(cTranslation.z),
		},
		PoseConfidence: 100.0, // C API doesn't provide confidence directly
	}

	// Calculate Euler angles from quaternion (ZYX convention)
	// Roll (X-axis rotation)
	sinr_cosp := float64(2 * (cRotation.w*cRotation.x + cRotation.y*cRotation.z))
	cosr_cosp := float64(1 - 2*(cRotation.x*cRotation.x+cRotation.y*cRotation.y))
	roll := float32(atan2(sinr_cosp, cosr_cosp))

	// Pitch (Y-axis rotation)
	sinp := float64(2 * (cRotation.w*cRotation.y - cRotation.z*cRotation.x))
	var pitch float32
	if abs32(float32(sinp)) >= 1 {
		pitch = float32(copysign(1.5707963, sinp)) // use 90 degrees if out of range
	} else {
		pitch = float32(asin(sinp))
	}

	// Yaw (Z-axis rotation)
	siny_cosp := float64(2 * (cRotation.w*cRotation.z + cRotation.x*cRotation.y))
	cosy_cosp := float64(1 - 2*(cRotation.y*cRotation.y+cRotation.z*cRotation.z))
	yaw := float32(atan2(siny_cosp, cosy_cosp))

	pose.EulerAngles = Vector3{
		X: roll,
		Y: pitch,
		Z: yaw,
	}

	// Generate rotation matrix from quaternion
	pose.Rotation = quaternionToRotationMatrix(pose.Orientation)

	// Map tracking state from C to Go
	switch state {
	case C.SL_POSITIONAL_TRACKING_STATE_SEARCHING:
		pose.TrackingState = TrackingStateSearching
	case C.SL_POSITIONAL_TRACKING_STATE_OK:
		pose.TrackingState = TrackingStateOK
	case C.SL_POSITIONAL_TRACKING_STATE_OFF:
		pose.TrackingState = TrackingStateOff
		pose.Valid = false
	case C.SL_POSITIONAL_TRACKING_STATE_FPS_TOO_LOW:
		pose.TrackingState = TrackingStateFPSTooLow
	default:
		pose.TrackingState = TrackingStateOff
		pose.Valid = false
	}

	return pose, nil
}

// Helper math functions for quaternion to Euler conversion
func atan2(y, x float64) float64 {
	// Simple atan2 implementation
	if x == 0 {
		if y > 0 {
			return 1.5707963
		}
		return -1.5707963
	}
	angle := atan(y / x)
	if x < 0 {
		if y >= 0 {
			return angle + 3.14159265
		}
		return angle - 3.14159265
	}
	return angle
}

func atan(x float64) float64 {
	// Taylor series for atan
	if x > 1 {
		return 1.5707963 - atan(1/x)
	}
	if x < -1 {
		return -1.5707963 - atan(1/x)
	}
	// For |x| <= 1, use Taylor series
	result := x
	term := x
	x2 := x * x
	for i := 1; i < 10; i++ {
		term *= -x2
		result += term / float64(2*i+1)
	}
	return result
}

func asin(x float64) float64 {
	// Use atan2 for asin: asin(x) = atan2(x, sqrt(1-x^2))
	if x*x >= 1 {
		if x > 0 {
			return 1.5707963
		}
		return -1.5707963
	}
	return atan2(x, sqrt64(1-x*x))
}

func sqrt64(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func copysign(mag, sign float64) float64 {
	if sign < 0 {
		return -mag
	}
	return mag
}

// quaternionToRotationMatrix converts a quaternion to a 3x3 rotation matrix
func quaternionToRotationMatrix(q Quaternion) Matrix3x3 {
	var m Matrix3x3

	xx := q.X * q.X
	xy := q.X * q.Y
	xz := q.X * q.Z
	xw := q.X * q.W
	yy := q.Y * q.Y
	yz := q.Y * q.Z
	yw := q.Y * q.W
	zz := q.Z * q.Z
	zw := q.Z * q.W

	m[0][0] = 1 - 2*(yy+zz)
	m[0][1] = 2 * (xy - zw)
	m[0][2] = 2 * (xz + yw)

	m[1][0] = 2 * (xy + zw)
	m[1][1] = 1 - 2*(xx+zz)
	m[1][2] = 2 * (yz - xw)

	m[2][0] = 2 * (xz - yw)
	m[2][1] = 2 * (yz + xw)
	m[2][2] = 1 - 2*(xx+yy)

	return m
}

// DisablePositionalTracking disables the positional tracking module.
func (c *ProdCamera) DisablePositionalTracking() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Disabling positional tracking")
	C.sl_disable_positional_tracking(c.cameraID, nil)
	return nil
}

// svoCompressionModeToC converts Go SVOCompressionMode to C enum
func svoCompressionModeToC(mode SVOCompressionMode) C.enum_SL_SVO_COMPRESSION_MODE {
	switch mode {
	case SVOCompressionLossless:
		return C.SL_SVO_COMPRESSION_MODE_LOSSLESS
	case SVOCompressionH264:
		return C.SL_SVO_COMPRESSION_MODE_H264
	case SVOCompressionH265:
		return C.SL_SVO_COMPRESSION_MODE_H265
	case SVOCompressionH264Lossless:
		return C.SL_SVO_COMPRESSION_MODE_H264_LOSSLESS
	case SVOCompressionH265Lossless:
		return C.SL_SVO_COMPRESSION_MODE_H265_LOSSLESS
	default:
		return C.SL_SVO_COMPRESSION_MODE_H264
	}
}

// EnableRecording starts recording to an SVO file.
func (c *ProdCamera) EnableRecording(params RecordingParameters) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	cFilename := C.CString(params.Filename)
	defer C.free(unsafe.Pointer(cFilename))

	compressionMode := svoCompressionModeToC(params.CompressionMode)
	bitrate := C.uint(params.Bitrate)
	targetFPS := C.int(params.TargetFPS)
	transcode := C.bool(params.Transcode)

	c.logger.Info("Enabling SVO recording",
		"filename", params.Filename,
		"compression", params.CompressionMode.String(),
		"bitrate", params.Bitrate,
		"target_fps", params.TargetFPS,
	)

	errCode := C.sl_enable_recording(c.cameraID, cFilename, compressionMode, bitrate, targetFPS, transcode)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to enable recording: error code %d", errCode)
	}

	c.logger.Info("SVO recording enabled successfully")
	return nil
}

// DisableRecording stops recording and closes the SVO file.
func (c *ProdCamera) DisableRecording() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Disabling SVO recording")
	C.sl_disable_recording(c.cameraID)
	return nil
}

// GetRecordingStatus returns the current recording status.
func (c *ProdCamera) GetRecordingStatus() (*RecordingStatus, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	cStatus := C.sl_get_recording_status(c.cameraID)
	if cStatus == nil {
		return nil, fmt.Errorf("failed to get recording status")
	}

	status := &RecordingStatus{
		IsRecording:             bool(cStatus.is_recording),
		IsPaused:                bool(cStatus.is_paused),
		Status:                  bool(cStatus.status),
		CurrentCompressionTime:  float64(cStatus.current_compression_time),
		CurrentCompressionRatio: float64(cStatus.current_compression_ratio),
		AverageCompressionTime:  float64(cStatus.average_compression_time),
		AverageCompressionRatio: float64(cStatus.average_compression_ratio),
	}

	return status, nil
}

// PauseRecording pauses or resumes recording.
func (c *ProdCamera) PauseRecording(pause bool) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Pausing/resuming recording", "pause", pause)
	C.sl_pause_recording(c.cameraID, C.bool(pause))
	return nil
}

// Grab captures a new frame from the camera.
func (c *ProdCamera) Grab() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	var runtimeParams C.struct_SL_RuntimeParameters
	runtimeParams.enable_depth = C.bool(true)
	runtimeParams.confidence_threshold = 95
	runtimeParams.reference_frame = C.SL_REFERENCE_FRAME_CAMERA
	runtimeParams.texture_confidence_threshold = 100
	runtimeParams.remove_saturated_areas = C.bool(true)

	errCode := C.sl_grab(c.cameraID, &runtimeParams)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to grab frame: error code %d", int(errCode))
	}

	return nil
}

// RetrieveImage retrieves the left camera RGB image as RGBA bytes.
func (c *ProdCamera) RetrieveImage() (*Image, error) {
	return c.RetrieveImageView(ViewLeft)
}

// RetrieveImageView retrieves an image from a specific view.
func (c *ProdCamera) RetrieveImageView(view ViewType) (*Image, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	// Create a Mat to receive the image
	imagePtr := C.sl_mat_create_new(C.int(c.width), C.int(c.height), C.SL_MAT_TYPE_U8_C4, C.SL_MEM_CPU)
	if imagePtr == nil {
		return nil, fmt.Errorf("failed to create mat")
	}
	defer C.sl_mat_free(imagePtr, C.SL_MEM_CPU)

	// Retrieve the image from the specified view
	errCode := C.sl_retrieve_image(c.cameraID, imagePtr, viewTypeToC(view), C.SL_MEM_CPU, C.int(c.width), C.int(c.height), nil)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return nil, fmt.Errorf("failed to retrieve image: error code %d", int(errCode))
	}

	// Get the actual data pointer from the mat
	dataPtr := C.sl_mat_get_ptr(imagePtr, C.SL_MEM_CPU)
	if dataPtr == nil {
		return nil, fmt.Errorf("failed to get mat data pointer")
	}

	// Get the step (bytes per row) - may include padding
	step := int(C.sl_mat_get_step_bytes(imagePtr, C.SL_MEM_CPU))
	width := int(C.sl_mat_get_width(imagePtr))
	height := int(C.sl_mat_get_height(imagePtr))

	// Calculate actual image size and copy data
	imageSize := width * height * 4 // 4 bytes per pixel (BGRA)
	imageData := make([]byte, imageSize)

	// Copy from C memory to Go memory, accounting for stride
	if step == width*4 {
		// No padding, direct copy
		cData := (*[1 << 30]byte)(unsafe.Pointer(dataPtr))[:imageSize:imageSize]
		copy(imageData, cData)
	} else {
		// Has padding, copy row by row
		for y := 0; y < height; y++ {
			srcRow := unsafe.Pointer(uintptr(unsafe.Pointer(dataPtr)) + uintptr(y*step))
			dstRow := imageData[y*width*4 : (y+1)*width*4]
			cRow := (*[1 << 30]byte)(srcRow)[: width*4 : width*4]
			copy(dstRow, cRow)
		}
	}

	return &Image{
		Width:  c.width,
		Height: c.height,
		Data:   imageData,
	}, nil
}

// RetrieveMeasure retrieves depth or other measurement data.
func (c *ProdCamera) RetrieveMeasure(measure MeasureType) (*Measure, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	// Determine mat type based on measure type
	var matType C.enum_SL_MAT_TYPE
	var channelsPerPixel int

	switch measure {
	case MeasureDepth, MeasureDisparity, MeasureConfidence:
		matType = C.SL_MAT_TYPE_F32_C1
		channelsPerPixel = 1
	case MeasureXYZ:
		matType = C.SL_MAT_TYPE_F32_C4
		channelsPerPixel = 4
	default:
		return nil, fmt.Errorf("unsupported measure type: %v", measure)
	}

	// Create a Mat to receive the measure
	measurePtr := C.sl_mat_create_new(C.int(c.width), C.int(c.height), matType, C.SL_MEM_CPU)
	if measurePtr == nil {
		return nil, fmt.Errorf("failed to create mat")
	}
	defer C.sl_mat_free(measurePtr, C.SL_MEM_CPU)

	// Retrieve the measure
	errCode := C.sl_retrieve_measure(c.cameraID, measurePtr, measureTypeToC(measure), C.SL_MEM_CPU, C.int(c.width), C.int(c.height), nil)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return nil, fmt.Errorf("failed to retrieve measure: error code %d", int(errCode))
	}

	// Get the actual data pointer from the mat
	dataPtr := C.sl_mat_get_ptr(measurePtr, C.SL_MEM_CPU)
	if dataPtr == nil {
		return nil, fmt.Errorf("failed to get mat data pointer")
	}

	// Get dimensions
	width := int(C.sl_mat_get_width(measurePtr))
	height := int(C.sl_mat_get_height(measurePtr))
	step := int(C.sl_mat_get_step_bytes(measurePtr, C.SL_MEM_CPU))

	// Calculate size and copy data
	dataSize := width * height * channelsPerPixel
	measureData := make([]float32, dataSize)

	// Copy from C memory to Go memory
	bytesPerRow := width * channelsPerPixel * 4 // 4 bytes per float32
	if step == bytesPerRow {
		// No padding, direct copy
		cData := (*[1 << 28]float32)(unsafe.Pointer(dataPtr))[:dataSize:dataSize]
		copy(measureData, cData)
	} else {
		// Has padding, copy row by row
		floatsPerRow := width * channelsPerPixel
		for y := 0; y < height; y++ {
			srcRow := unsafe.Pointer(uintptr(unsafe.Pointer(dataPtr)) + uintptr(y*step))
			dstRow := measureData[y*floatsPerRow : (y+1)*floatsPerRow]
			cRow := (*[1 << 28]float32)(srcRow)[:floatsPerRow:floatsPerRow]
			copy(dstRow, cRow)
		}
	}

	return &Measure{
		Width:  width,
		Height: height,
		Data:   measureData,
		Type:   measure,
	}, nil
}

// EnableObjectDetection initializes and starts the object detection module.
func (c *ProdCamera) EnableObjectDetection(params ObjectDetectionParameters) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Enabling object detection",
		"model", params.DetectionModel,
		"tracking", params.EnableTracking,
		"confidence_threshold", params.DetectionConfidenceThreshold)

	// Create object detection parameters with all default values
	var objDetParams C.struct_SL_ObjectDetectionParameters
	objDetParams.instance_module_id = C.uint(params.InstanceModuleID)
	objDetParams.enable_tracking = C.bool(params.EnableTracking)
	objDetParams.enable_segmentation = C.bool(params.EnableSegmentation)
	objDetParams.detection_model = objectDetectionModelToC(params.DetectionModel)
	objDetParams.max_range = -1.0           // Use default from init parameters
	objDetParams.prediction_timeout_s = 0.2 // Default
	objDetParams.fused_objects_group_name = nil
	objDetParams.custom_onnx_file = nil
	objDetParams.filtering_mode = C.SL_OBJECT_FILTERING_MODE_NMS_3D

	// Initialize custom ONNX resolution to zero (not using custom model)
	objDetParams.custom_onnx_dynamic_input_shape.width = 0
	objDetParams.custom_onnx_dynamic_input_shape.height = 0

	// Initialize batch parameters
	objDetParams.batch_parameters.enable = C.bool(false)
	objDetParams.batch_parameters.id_retention_time = 240.0
	objDetParams.batch_parameters.latency = 2.0

	// Enable object detection
	errCode := C.sl_enable_object_detection(c.cameraID, &objDetParams)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to enable object detection: error code %d", int(errCode))
	}

	// Store the parameters for use in RetrieveObjects
	c.objectDetectionParams = params

	c.logger.Info("Object detection enabled successfully")
	return nil
}

// RetrieveObjects retrieves detected objects from the current frame.
func (c *ProdCamera) RetrieveObjects() (*Objects, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	// Create runtime parameters with class filter
	var runtimeParams C.struct_SL_ObjectDetectionRuntimeParameters
	runtimeParams.detection_confidence_threshold = C.float(c.objectDetectionParams.DetectionConfidenceThreshold)

	// Set class filter based on stored parameters
	if len(c.objectDetectionParams.ClassFilter) > 0 {
		// Initialize all to disabled
		for i := 0; i < int(ObjectClassLast); i++ {
			runtimeParams.object_class_filter[i] = C.int(0)
			runtimeParams.object_confidence_threshold[i] = C.int(c.objectDetectionParams.DetectionConfidenceThreshold)
		}
		// Enable only specified classes
		for _, class := range c.objectDetectionParams.ClassFilter {
			runtimeParams.object_class_filter[int(class)] = C.int(1)
		}
	} else {
		// Enable all classes
		for i := 0; i < int(ObjectClassLast); i++ {
			runtimeParams.object_class_filter[i] = C.int(1)
			runtimeParams.object_confidence_threshold[i] = C.int(c.objectDetectionParams.DetectionConfidenceThreshold)
		}
	}

	// Retrieve objects
	var cObjects C.struct_SL_Objects
	errCode := C.sl_retrieve_objects(c.cameraID, &runtimeParams, &cObjects, 0)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return nil, fmt.Errorf("failed to retrieve objects: error code %d", int(errCode))
	}

	// Convert C objects to Go objects
	objects := &Objects{
		Timestamp:  uint64(cObjects.timestamp),
		IsNew:      cObjects.is_new != 0,
		IsTracked:  cObjects.is_tracked != 0,
		ObjectList: make([]ObjectData, 0),
	}

	// Convert each detected object
	numObjects := int(cObjects.nb_objects)
	if numObjects > 0 {
		// Access the object_list array (it's a fixed array, not a pointer)
		for i := 0; i < numObjects; i++ {
			cObj := cObjects.object_list[i]

			obj := ObjectData{
				ID:            int(cObj.id),
				Label:         objectClassFromC(cObj.label),
				Sublabel:      objectSubclassFromC(cObj.sublabel),
				TrackingState: objectTrackingStateFromC(cObj.tracking_state),
				ActionState:   objectActionStateFromC(cObj.action_state),
				Position: Vector3{
					X: float32(cObj.position.x),
					Y: float32(cObj.position.y),
					Z: float32(cObj.position.z),
				},
				Velocity: Vector3{
					X: float32(cObj.velocity.x),
					Y: float32(cObj.velocity.y),
					Z: float32(cObj.velocity.z),
				},
				Confidence: float32(cObj.confidence),
				Dimensions: Vector3{
					X: float32(cObj.dimensions.x),
					Y: float32(cObj.dimensions.y),
					Z: float32(cObj.dimensions.z),
				},
			}

			// Convert 2D bounding box
			bbox := cObj.bounding_box_2d
			obj.BoundingBox2D = BoundingBox2D{
				X:      int(bbox[0].x),
				Y:      int(bbox[0].y),
				Width:  int(bbox[2].x - bbox[0].x),
				Height: int(bbox[2].y - bbox[0].y),
			}

			objects.ObjectList = append(objects.ObjectList, obj)
		}
	}

	return objects, nil
}

// DisableObjectDetection disables the object detection module.
func (c *ProdCamera) DisableObjectDetection() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Disabling object detection")
	C.sl_disable_object_detection(c.cameraID, 0, C.bool(false))
	return nil
}

// EnableBodyTracking initializes and starts the body tracking module.
func (c *ProdCamera) EnableBodyTracking(params BodyTrackingParameters) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Enabling body tracking",
		"model", params.DetectionModel,
		"tracking", params.EnableTracking,
		"body_format", params.BodyFormat,
		"confidence_threshold", params.DetectionConfidenceThreshold)

	// Create body tracking parameters
	var bodyTrackingParams C.struct_SL_BodyTrackingParameters
	bodyTrackingParams.instance_module_id = C.uint(params.InstanceModuleID)
	bodyTrackingParams.enable_tracking = C.bool(params.EnableTracking)
	bodyTrackingParams.enable_segmentation = C.bool(params.EnableSegmentation)
	bodyTrackingParams.detection_model = bodyTrackingModelToC(params.DetectionModel)
	bodyTrackingParams.body_format = bodyFormatToC(params.BodyFormat)
	bodyTrackingParams.enable_body_fitting = C.bool(params.EnableBodyFitting)
	bodyTrackingParams.max_range = -1.0           // Use default from init parameters
	bodyTrackingParams.prediction_timeout_s = 0.2 // Default

	// Enable body tracking
	errCode := C.sl_enable_body_tracking(c.cameraID, &bodyTrackingParams)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to enable body tracking: error code %d", int(errCode))
	}

	// Store the parameters for use in RetrieveBodies
	c.bodyTrackingParams = params

	c.logger.Info("Body tracking enabled successfully")
	return nil
}

// RetrieveBodies retrieves detected bodies from the current frame.
func (c *ProdCamera) RetrieveBodies() (*Bodies, error) {
	if c.cameraID < 0 {
		return nil, fmt.Errorf("camera not opened")
	}

	// Create runtime parameters
	var runtimeParams C.struct_SL_BodyTrackingRuntimeParameters
	runtimeParams.detection_confidence_threshold = C.float(c.bodyTrackingParams.DetectionConfidenceThreshold)

	// Retrieve bodies
	var cBodies C.struct_SL_Bodies
	errCode := C.sl_retrieve_bodies(c.cameraID, &runtimeParams, &cBodies, 0)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return nil, fmt.Errorf("failed to retrieve bodies: error code %d", int(errCode))
	}

	// Convert C bodies to Go bodies
	bodies := &Bodies{
		Timestamp: uint64(cBodies.timestamp),
		IsNew:     cBodies.is_new != 0,
		IsTracked: cBodies.is_tracked != 0,
		BodyList:  make([]BodyData, 0),
	}

	// Convert each detected body
	numBodies := int(cBodies.nb_bodies)
	numKeypoints := c.bodyTrackingParams.BodyFormat.NumKeypoints()

	if numBodies > 0 {
		// Access the body_list array
		for i := 0; i < numBodies; i++ {
			cBody := cBodies.body_list[i]

			body := BodyData{
				ID:             int(cBody.id),
				UniqueObjectID: C.GoString((*C.char)(unsafe.Pointer(&cBody.unique_object_id[0]))),
				TrackingState:  objectTrackingStateFromC(cBody.tracking_state),
				ActionState:    objectActionStateFromC(cBody.action_state),
				Position: Vector3{
					X: float32(cBody.position.x),
					Y: float32(cBody.position.y),
					Z: float32(cBody.position.z),
				},
				Velocity: Vector3{
					X: float32(cBody.velocity.x),
					Y: float32(cBody.velocity.y),
					Z: float32(cBody.velocity.z),
				},
				Confidence: float32(cBody.confidence),
				Dimensions: Vector3{
					X: float32(cBody.dimensions.x),
					Y: float32(cBody.dimensions.y),
					Z: float32(cBody.dimensions.z),
				},
				HeadPosition: Vector3{
					X: float32(cBody.head_position.x),
					Y: float32(cBody.head_position.y),
					Z: float32(cBody.head_position.z),
				},
				GlobalRootOrientation: Quaternion{
					X: float32(cBody.global_root_orientation.x),
					Y: float32(cBody.global_root_orientation.y),
					Z: float32(cBody.global_root_orientation.z),
					W: float32(cBody.global_root_orientation.w),
				},
			}

			// Convert 2D bounding box
			bbox := cBody.bounding_box_2d
			body.BoundingBox2D = BoundingBox2D{
				X:      int(bbox[0].x),
				Y:      int(bbox[0].y),
				Width:  int(bbox[2].x - bbox[0].x),
				Height: int(bbox[2].y - bbox[0].y),
			}

			// Convert head 2D bounding box
			headBbox := cBody.head_bounding_box_2d
			body.HeadBoundingBox2D = BoundingBox2D{
				X:      int(headBbox[0].x),
				Y:      int(headBbox[0].y),
				Width:  int(headBbox[2].x - headBbox[0].x),
				Height: int(headBbox[2].y - headBbox[0].y),
			}

			// Convert keypoints 2D
			body.Keypoints2D = make([]Keypoint2D, numKeypoints)
			for j := 0; j < numKeypoints; j++ {
				body.Keypoints2D[j] = Keypoint2D{
					X: float32(cBody.keypoint_2d[j].x),
					Y: float32(cBody.keypoint_2d[j].y),
				}
			}

			// Convert keypoints 3D
			body.Keypoints3D = make([]Keypoint3D, numKeypoints)
			for j := 0; j < numKeypoints; j++ {
				body.Keypoints3D[j] = Keypoint3D{
					X: float32(cBody.keypoint[j].x),
					Y: float32(cBody.keypoint[j].y),
					Z: float32(cBody.keypoint[j].z),
				}
			}

			// Convert keypoint confidence
			body.KeypointConfidence = make([]float32, numKeypoints)
			for j := 0; j < numKeypoints; j++ {
				body.KeypointConfidence[j] = float32(cBody.keypoint_confidence[j])
			}

			// Convert local position per joint (only if body fitting is enabled)
			if c.bodyTrackingParams.EnableBodyFitting {
				body.LocalPositionPerJoint = make([]Vector3, numKeypoints)
				for j := 0; j < numKeypoints; j++ {
					body.LocalPositionPerJoint[j] = Vector3{
						X: float32(cBody.local_position_per_joint[j].x),
						Y: float32(cBody.local_position_per_joint[j].y),
						Z: float32(cBody.local_position_per_joint[j].z),
					}
				}

				// Convert local orientation per joint
				body.LocalOrientationPerJoint = make([]Quaternion, numKeypoints)
				for j := 0; j < numKeypoints; j++ {
					body.LocalOrientationPerJoint[j] = Quaternion{
						X: float32(cBody.local_orientation_per_joint[j].x),
						Y: float32(cBody.local_orientation_per_joint[j].y),
						Z: float32(cBody.local_orientation_per_joint[j].z),
						W: float32(cBody.local_orientation_per_joint[j].w),
					}
				}
			}

			bodies.BodyList = append(bodies.BodyList, body)
		}
	}

	return bodies, nil
}

// DisableBodyTracking disables the body tracking module.
func (c *ProdCamera) DisableBodyTracking() error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	c.logger.Info("Disabling body tracking")
	C.sl_disable_body_tracking(c.cameraID, 0, C.bool(false))
	return nil
}

// GetSetting returns the current value of a camera video setting.
func (c *ProdCamera) GetSetting(setting VideoSetting) (int, error) {
	if c.cameraID < 0 {
		return 0, fmt.Errorf("camera not opened")
	}

	var value C.int
	errCode := C.sl_get_camera_settings(c.cameraID, videoSettingToC(setting), &value)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return 0, fmt.Errorf("failed to get setting %s: error code %d", setting, int(errCode))
	}

	return int(value), nil
}

// SetSetting sets the value of a camera video setting.
func (c *ProdCamera) SetSetting(setting VideoSetting, value int) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	errCode := C.sl_set_camera_settings(c.cameraID, videoSettingToC(setting), C.int(value))
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to set setting %s to %d: error code %d", setting, value, int(errCode))
	}

	return nil
}

// GetSettingRange returns the min/max range for range-type settings.
func (c *ProdCamera) GetSettingRange(setting VideoSetting) (min, max int, err error) {
	if c.cameraID < 0 {
		return 0, 0, fmt.Errorf("camera not opened")
	}

	var minVal, maxVal C.int
	errCode := C.sl_get_camera_settings_min_max(c.cameraID, videoSettingToC(setting), &minVal, &maxVal)
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return 0, 0, fmt.Errorf("failed to get setting range %s: error code %d", setting, int(errCode))
	}

	return int(minVal), int(maxVal), nil
}

// SetSettingRange sets the min/max range for range-type settings.
func (c *ProdCamera) SetSettingRange(setting VideoSetting, min, max int) error {
	if c.cameraID < 0 {
		return fmt.Errorf("camera not opened")
	}

	errCode := C.sl_set_camera_settings_min_max(c.cameraID, videoSettingToC(setting), C.int(min), C.int(max))
	if errCode != C.SL_ERROR_CODE_SUCCESS {
		return fmt.Errorf("failed to set setting range %s to [%d, %d]: error code %d", setting, min, max, int(errCode))
	}

	return nil
}

// IsSettingSupported returns true if the given video setting is supported by this camera.
func (c *ProdCamera) IsSettingSupported(setting VideoSetting) bool {
	if c.cameraID < 0 {
		return false
	}

	return bool(C.sl_is_camera_setting_supported(c.cameraID, videoSettingToC(setting)))
}

// GetSerialNumber returns the serial number of the opened camera.
func (c *ProdCamera) GetSerialNumber() uint {
	return c.serialNumber
}

// Close closes the camera and releases all resources.
func (c *ProdCamera) Close() error {
	if c.cameraID < 0 {
		return nil // Already closed
	}

	c.logger.Info("Closing ZED camera")
	C.sl_close_camera(c.cameraID)
	c.cameraID = -1

	return nil
}
