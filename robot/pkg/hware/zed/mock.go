package zed

import (
	"context"
	"log/slog"
)

// MockCamera returns a no-op implementation of the Camera interface for testing purposes.
// It implements all Camera methods but performs no actual operations, always returning nil errors.
// This is useful for testing code that depends on a Camera without requiring actual hardware.
func MockCamera(logger *slog.Logger) Camera {
	return &mockCamera{
		logger: logger,
	}
}

// mockCamera is a no-op implementation of the Camera interface.
// It's used for testing scenarios where actual hardware interaction is not required.
type mockCamera struct {
	logger *slog.Logger
}

var _ Camera = (*mockCamera)(nil)

// Open is a no-op implementation that simulates opening a camera.
func (c *mockCamera) Open() error {
	c.logger.Info("Mock Camera: Open called")
	return nil
}

// Grab is a no-op implementation that simulates grabbing a frame.
func (c *mockCamera) Grab() error {
	return nil
}

// RetrieveImage returns a dummy image for testing.
func (c *mockCamera) RetrieveImage() (*Image, error) {
	return &Image{
		Width:  640,
		Height: 480,
		Data:   make([]byte, 640*480*4),
	}, nil
}

// RetrieveImageView returns a dummy image for the specified view.
func (c *mockCamera) RetrieveImageView(view ViewType) (*Image, error) {
	c.logger.Debug("Mock Camera: RetrieveImageView called", "view", view)
	return c.RetrieveImage()
}

// RetrieveMeasure returns a dummy measure for testing.
func (c *mockCamera) RetrieveMeasure(measure MeasureType) (*Measure, error) {
	return &Measure{
		Width:  640,
		Height: 480,
		Data:   make([]float32, 640*480),
		Type:   measure,
	}, nil
}

// EnableObjectDetection is a no-op implementation.
func (c *mockCamera) EnableObjectDetection(params ObjectDetectionParameters) error {
	c.logger.Info("Mock Camera: EnableObjectDetection called", "params", params)
	return nil
}

// RetrieveObjects returns an empty objects list for testing.
func (c *mockCamera) RetrieveObjects() (*Objects, error) {
	return &Objects{
		ObjectList: []ObjectData{},
		IsNew:      true,
		IsTracked:  false,
	}, nil
}

// DisableObjectDetection is a no-op implementation.
func (c *mockCamera) DisableObjectDetection() error {
	c.logger.Info("Mock Camera: DisableObjectDetection called")
	return nil
}

// EnablePositionalTracking is a no-op implementation.
func (c *mockCamera) EnablePositionalTracking() error {
	c.logger.Info("Mock Camera: EnablePositionalTracking called")
	return nil
}

// GetPosition returns a dummy pose for testing.
func (c *mockCamera) GetPosition() (*Pose, error) {
	return &Pose{}, nil
}

// DisablePositionalTracking is a no-op implementation.
func (c *mockCamera) DisablePositionalTracking() error {
	c.logger.Info("Mock Camera: DisablePositionalTracking called")
	return nil
}

// EnableRecording is a no-op implementation.
func (c *mockCamera) EnableRecording(params RecordingParameters) error {
	c.logger.Info("Mock Camera: EnableRecording called", "filename", params.Filename)
	return nil
}

// DisableRecording is a no-op implementation.
func (c *mockCamera) DisableRecording() error {
	c.logger.Info("Mock Camera: DisableRecording called")
	return nil
}

// GetRecordingStatus returns a dummy recording status.
func (c *mockCamera) GetRecordingStatus() (*RecordingStatus, error) {
	return &RecordingStatus{
		IsRecording: false,
	}, nil
}

// PauseRecording is a no-op implementation.
func (c *mockCamera) PauseRecording(pause bool) error {
	c.logger.Info("Mock Camera: PauseRecording called", "pause", pause)
	return nil
}

// EnableBodyTracking is a no-op implementation.
func (c *mockCamera) EnableBodyTracking(params BodyTrackingParameters) error {
	c.logger.Info("Mock Camera: EnableBodyTracking called")
	return nil
}

// RetrieveBodies returns an empty bodies list for testing.
func (c *mockCamera) RetrieveBodies() (*Bodies, error) {
	return &Bodies{
		BodyList:  []BodyData{},
		IsNew:     true,
		IsTracked: false,
	}, nil
}

// DisableBodyTracking is a no-op implementation.
func (c *mockCamera) DisableBodyTracking() error {
	c.logger.Info("Mock Camera: DisableBodyTracking called")
	return nil
}

// GetSetting returns a dummy value for any setting.
func (c *mockCamera) GetSetting(setting VideoSetting) (int, error) {
	return 0, nil
}

// SetSetting is a no-op implementation.
func (c *mockCamera) SetSetting(setting VideoSetting, value int) error {
	return nil
}

// GetSettingRange returns dummy range values.
func (c *mockCamera) GetSettingRange(setting VideoSetting) (min, max int, err error) {
	return 0, 100, nil
}

// SetSettingRange is a no-op implementation.
func (c *mockCamera) SetSettingRange(setting VideoSetting, min, max int) error {
	return nil
}

// IsSettingSupported always returns true for mock.
func (c *mockCamera) IsSettingSupported(setting VideoSetting) bool {
	return true
}

// GetSerialNumber returns 0 for mock.
func (c *mockCamera) GetSerialNumber() uint {
	return 0
}

// Close is a no-op implementation.
func (c *mockCamera) Close() error {
	c.logger.Info("Mock Camera: Close called")
	return nil
}

// MockArray returns a no-op implementation of the Array interface for testing purposes.
// It implements all Array methods but performs no actual operations, always returning nil errors.
func MockArray(logger *slog.Logger) Array {
	return &mockArray{
		logger: logger,
	}
}

// mockArray is a no-op implementation of the Array interface.
type mockArray struct {
	logger *slog.Logger
}

var _ Array = (*mockArray)(nil)

// Start is a no-op implementation that simulates starting the camera array.
func (a *mockArray) Start(ctx context.Context) error {
	a.logger.Info("Mock Array: Start called")
	return nil
}

// Record is a no-op implementation that blocks until context is cancelled.
func (a *mockArray) Record(ctx context.Context) error {
	a.logger.Info("Mock Array: Record called")
	<-ctx.Done()
	a.logger.Info("Mock Array: Record stopped")
	return ctx.Err()
}

// Image returns a dummy 2x2 grid image for testing.
func (a *mockArray) Image() (*Image, error) {
	// Return a 1280x960 image (2x2 grid of 640x480 cameras)
	return &Image{
		Width:  1280,
		Height: 960,
		Data:   make([]byte, 1280*960*4),
	}, nil
}
