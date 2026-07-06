//go:build !zed_sdk
// +build !zed_sdk

package zed

import (
	"fmt"
	"log/slog"
)

// ProdCamera is a stub implementation when CGO is disabled or during testing.
type ProdCamera struct {
	config Config
	logger *slog.Logger
}

// NewCamera creates a new stub camera instance that returns an error on Open.
func NewCamera(cfg Config) *ProdCamera {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	logger := cfg.Logger.With("system", "zed-stub")

	return &ProdCamera{
		config: cfg,
		logger: logger,
	}
}

// Open returns an error indicating that CGO is required.
func (c *ProdCamera) Open() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// Grab returns an error indicating that CGO is required.
func (c *ProdCamera) Grab() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// RetrieveImage returns an error indicating that CGO is required.
func (c *ProdCamera) RetrieveImage() (*Image, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// RetrieveImageView returns an error indicating that CGO is required.
func (c *ProdCamera) RetrieveImageView(view ViewType) (*Image, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// RetrieveMeasure returns an error indicating that CGO is required.
func (c *ProdCamera) RetrieveMeasure(measure MeasureType) (*Measure, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// EnableObjectDetection returns an error indicating that CGO is required.
func (c *ProdCamera) EnableObjectDetection(params ObjectDetectionParameters) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// RetrieveObjects returns an error indicating that CGO is required.
func (c *ProdCamera) RetrieveObjects() (*Objects, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// DisableObjectDetection returns an error indicating that CGO is required.
func (c *ProdCamera) DisableObjectDetection() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// EnablePositionalTracking returns an error indicating that CGO is required.
func (c *ProdCamera) EnablePositionalTracking() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// GetPosition returns an error indicating that CGO is required.
func (c *ProdCamera) GetPosition() (*Pose, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// DisablePositionalTracking returns an error indicating that CGO is required.
func (c *ProdCamera) DisablePositionalTracking() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// EnableRecording returns an error indicating that CGO is required.
func (c *ProdCamera) EnableRecording(params RecordingParameters) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// DisableRecording returns an error indicating that CGO is required.
func (c *ProdCamera) DisableRecording() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// GetRecordingStatus returns an error indicating that CGO is required.
func (c *ProdCamera) GetRecordingStatus() (*RecordingStatus, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// PauseRecording returns an error indicating that CGO is required.
func (c *ProdCamera) PauseRecording(pause bool) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// EnableBodyTracking returns an error indicating that CGO is required.
func (c *ProdCamera) EnableBodyTracking(params BodyTrackingParameters) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// RetrieveBodies returns an error indicating that CGO is required.
func (c *ProdCamera) RetrieveBodies() (*Bodies, error) {
	return nil, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// DisableBodyTracking returns an error indicating that CGO is required.
func (c *ProdCamera) DisableBodyTracking() error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// GetSetting returns an error indicating that CGO is required.
func (c *ProdCamera) GetSetting(setting VideoSetting) (int, error) {
	return 0, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// SetSetting returns an error indicating that CGO is required.
func (c *ProdCamera) SetSetting(setting VideoSetting, value int) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// GetSettingRange returns an error indicating that CGO is required.
func (c *ProdCamera) GetSettingRange(setting VideoSetting) (min, max int, err error) {
	return 0, 0, fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// SetSettingRange returns an error indicating that CGO is required.
func (c *ProdCamera) SetSettingRange(setting VideoSetting, min, max int) error {
	return fmt.Errorf("ZED camera requires CGO and the ZED SDK to be installed")
}

// IsSettingSupported returns false for the stub implementation.
func (c *ProdCamera) IsSettingSupported(setting VideoSetting) bool {
	return false
}

// GetSerialNumber returns 0 for the stub implementation.
func (c *ProdCamera) GetSerialNumber() uint {
	return 0
}

// Close does nothing for the stub implementation.
func (c *ProdCamera) Close() error {
	return nil
}
