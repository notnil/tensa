package zed

import (
	"context"
	"image"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestProdCamera tests the production ZED camera implementation.
// This test requires actual ZED hardware and the ZED SDK to be installed.
// The test will be skipped if the hardware is not available.
func TestProdCamera(t *testing.T) {
	// Skip if ZED SDK is not available
	if _, err := os.Stat("/usr/local/zed/lib"); os.IsNotExist(err) {
		t.Skip("ZED SDK not found at /usr/local/zed/lib, skipping hardware test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := DefaultConfig()
	cfg.Logger = logger
	cfg.Resolution = ResolutionHD720
	cfg.FPS = 30

	cam := NewCamera(cfg)

	// Test Open
	err := cam.Open()
	if err != nil {
		t.Skipf("Failed to open ZED camera (hardware may not be connected): %v", err)
	}
	defer cam.Close()

	// Test Grab
	err = cam.Grab()
	if err != nil {
		t.Fatalf("Failed to grab frame: %v", err)
	}

	// Test RetrieveImage
	img, err := cam.RetrieveImage()
	if err != nil {
		t.Fatalf("Failed to retrieve image: %v", err)
	}

	// Validate image dimensions
	expectedWidth, expectedHeight := cfg.Resolution.Dimensions()
	if img.Width != expectedWidth {
		t.Errorf("Expected width %d, got %d", expectedWidth, img.Width)
	}
	if img.Height != expectedHeight {
		t.Errorf("Expected height %d, got %d", expectedHeight, img.Height)
	}

	// Validate image data size
	expectedSize := expectedWidth * expectedHeight * 4 // 4 bytes per pixel
	if len(img.Data) != expectedSize {
		t.Errorf("Expected image data size %d, got %d", expectedSize, len(img.Data))
	}

	// Test that image data is not all zeros (actual image data)
	allZeros := true
	for _, b := range img.Data {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Image data is all zeros, expected actual image data")
	}

	// Test multiple frames
	t.Log("Capturing 10 frames to test continuous operation...")
	for i := 0; i < 10; i++ {
		err = cam.Grab()
		if err != nil {
			t.Fatalf("Failed to grab frame %d: %v", i+1, err)
		}

		img, err = cam.RetrieveImage()
		if err != nil {
			t.Fatalf("Failed to retrieve image %d: %v", i+1, err)
		}

		if len(img.Data) != expectedSize {
			t.Errorf("Frame %d: Expected image data size %d, got %d", i+1, expectedSize, len(img.Data))
		}
	}

	t.Logf("Successfully captured 10 frames from ZED camera")
}

// TestResolutionDimensions tests that resolution dimensions are correct.
func TestResolutionDimensions(t *testing.T) {
	tests := []struct {
		resolution     Resolution
		expectedWidth  int
		expectedHeight int
	}{
		{ResolutionHD2K, 2208, 1242},
		{ResolutionHD1080, 1920, 1080},
		{ResolutionHD720, 1280, 720},
		{ResolutionVGA, 672, 376},
	}

	for _, tt := range tests {
		t.Run(tt.resolution.String(), func(t *testing.T) {
			width, height := tt.resolution.Dimensions()
			if width != tt.expectedWidth {
				t.Errorf("Expected width %d, got %d", tt.expectedWidth, width)
			}
			if height != tt.expectedHeight {
				t.Errorf("Expected height %d, got %d", tt.expectedHeight, height)
			}
		})
	}
}

// TestImageMethods tests the Image helper methods.
func TestImageMethods(t *testing.T) {
	img := &Image{
		Width:  1280,
		Height: 720,
		Data:   make([]byte, 1280*720*4),
	}

	// Test BytesPerPixel
	if img.BytesPerPixel() != 4 {
		t.Errorf("Expected BytesPerPixel 4, got %d", img.BytesPerPixel())
	}

	// Test Size
	expectedSize := 1280 * 720 * 4
	if img.Size() != expectedSize {
		t.Errorf("Expected Size %d, got %d", expectedSize, img.Size())
	}
}

// TestImageInterface tests that Image implements the standard image.Image interface.
func TestImageInterface(t *testing.T) {
	// Create a test image with some data
	img := &Image{
		Width:  100,
		Height: 100,
		Data:   make([]byte, 100*100*4),
	}

	// Set a test pixel (50, 50) to red
	offset := (50*100 + 50) * 4
	img.Data[offset+0] = 255 // R
	img.Data[offset+1] = 0   // G
	img.Data[offset+2] = 0   // B
	img.Data[offset+3] = 255 // A

	// Verify it implements image.Image
	var _ image.Image = img

	// Test Bounds
	bounds := img.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("Expected bounds 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Test ColorModel
	if img.ColorModel() == nil {
		t.Error("ColorModel returned nil")
	}

	// Test At
	c := img.At(50, 50)
	r, g, b, a := c.RGBA()
	// RGBA() returns 16-bit values, so we need to convert
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 || a>>8 != 255 {
		t.Errorf("Expected red pixel (255,0,0,255), got (%d,%d,%d,%d)", r>>8, g>>8, b>>8, a>>8)
	}

	// Test At with out of bounds
	c = img.At(-1, -1)
	r, g, b, a = c.RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0 {
		t.Errorf("Expected zero color for out of bounds, got (%d,%d,%d,%d)", r, g, b, a)
	}

	// Test ToRGBA conversion
	rgba := img.ToRGBA()
	if rgba.Bounds().Dx() != 100 || rgba.Bounds().Dy() != 100 {
		t.Errorf("Expected RGBA bounds 100x100, got %dx%d", rgba.Bounds().Dx(), rgba.Bounds().Dy())
	}

	// Verify the pixel data was copied correctly
	rgbaColor := rgba.RGBAAt(50, 50)
	if rgbaColor.R != 255 || rgbaColor.G != 0 || rgbaColor.B != 0 || rgbaColor.A != 255 {
		t.Errorf("Expected red pixel in RGBA (255,0,0,255), got (%d,%d,%d,%d)",
			rgbaColor.R, rgbaColor.G, rgbaColor.B, rgbaColor.A)
	}
}

// TestDefaultConfig tests that the default configuration is valid.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Resolution != ResolutionHD1200 {
		t.Errorf("Expected default resolution HD1200, got %v", cfg.Resolution)
	}
	if cfg.FPS != 30 {
		t.Errorf("Expected default FPS 30, got %d", cfg.FPS)
	}
	if cfg.DepthMode != DepthModeNeural {
		t.Errorf("Expected default depth mode NEURAL, got %v", cfg.DepthMode)
	}
	if cfg.CameraID != 0 {
		t.Errorf("Expected default camera ID 0, got %d", cfg.CameraID)
	}
	if cfg.Logger == nil {
		t.Error("Expected default logger to be non-nil")
	}
}

// TestMeasureToGrayscaleImage tests converting measure data to grayscale images.
func TestMeasureToGrayscaleImage(t *testing.T) {
	t.Run("Depth measure", func(t *testing.T) {
		// Create a test depth measure with gradient values
		width, height := 10, 10
		data := make([]float32, width*height)
		for i := range data {
			data[i] = float32(i) // Values from 0 to 99
		}

		measure := &Measure{
			Width:  width,
			Height: height,
			Data:   data,
			Type:   MeasureDepth,
		}

		img, err := measure.ToGrayscaleImage()
		if err != nil {
			t.Fatalf("ToGrayscaleImage failed: %v", err)
		}

		// Verify image dimensions
		bounds := img.Bounds()
		if bounds.Dx() != width || bounds.Dy() != height {
			t.Errorf("Expected dimensions %dx%d, got %dx%d", width, height, bounds.Dx(), bounds.Dy())
		}

		// Verify values are normalized (first pixel should be darkest, last should be brightest)
		firstPixel := img.GrayAt(0, 0)
		lastPixel := img.GrayAt(width-1, height-1)
		if firstPixel.Y >= lastPixel.Y {
			t.Errorf("Expected gradient: first pixel (%d) should be darker than last pixel (%d)",
				firstPixel.Y, lastPixel.Y)
		}
	})

	t.Run("Disparity measure", func(t *testing.T) {
		measure := &Measure{
			Width:  5,
			Height: 5,
			Data:   make([]float32, 25),
			Type:   MeasureDisparity,
		}
		// Set some test values
		for i := range measure.Data {
			measure.Data[i] = float32(i * 10)
		}

		img, err := measure.ToGrayscaleImage()
		if err != nil {
			t.Fatalf("ToGrayscaleImage failed for disparity: %v", err)
		}

		if img.Bounds().Dx() != 5 || img.Bounds().Dy() != 5 {
			t.Errorf("Expected dimensions 5x5, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
		}
	})

	t.Run("Confidence measure", func(t *testing.T) {
		measure := &Measure{
			Width:  3,
			Height: 3,
			Data:   []float32{0.1, 0.5, 0.9, 0.2, 0.6, 1.0, 0.3, 0.7, 0.8},
			Type:   MeasureConfidence,
		}

		img, err := measure.ToGrayscaleImage()
		if err != nil {
			t.Fatalf("ToGrayscaleImage failed for confidence: %v", err)
		}

		if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 3 {
			t.Errorf("Expected dimensions 3x3, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
		}
	})

	t.Run("XYZ point cloud - should error", func(t *testing.T) {
		measure := &Measure{
			Width:  10,
			Height: 10,
			Data:   make([]float32, 10*10*4), // 4 channels for XYZ+W
			Type:   MeasureXYZ,
		}

		_, err := measure.ToGrayscaleImage()
		if err == nil {
			t.Error("Expected error for point cloud data, got nil")
		}
	})

	t.Run("All same values", func(t *testing.T) {
		// All pixels have the same depth value
		measure := &Measure{
			Width:  5,
			Height: 5,
			Data:   make([]float32, 25),
			Type:   MeasureDepth,
		}
		for i := range measure.Data {
			measure.Data[i] = 42.0
		}

		img, err := measure.ToGrayscaleImage()
		if err != nil {
			t.Fatalf("ToGrayscaleImage failed for uniform values: %v", err)
		}

		// All pixels should be middle gray (128)
		for y := 0; y < 5; y++ {
			for x := 0; x < 5; x++ {
				pixel := img.GrayAt(x, y)
				if pixel.Y != 128 {
					t.Errorf("Expected uniform gray 128, got %d at (%d,%d)", pixel.Y, x, y)
				}
			}
		}
	})
}

// TestVideoSettingString tests the String() method for VideoSetting enum.
func TestVideoSettingString(t *testing.T) {
	tests := []struct {
		setting  VideoSetting
		expected string
	}{
		{VideoSettingBrightness, "BRIGHTNESS"},
		{VideoSettingContrast, "CONTRAST"},
		{VideoSettingHue, "HUE"},
		{VideoSettingSaturation, "SATURATION"},
		{VideoSettingSharpness, "SHARPNESS"},
		{VideoSettingGamma, "GAMMA"},
		{VideoSettingGain, "GAIN"},
		{VideoSettingExposure, "EXPOSURE"},
		{VideoSettingAECAGC, "AEC_AGC"},
		{VideoSettingWhiteBalanceTemperature, "WHITEBALANCE_TEMPERATURE"},
		{VideoSettingWhiteBalanceAuto, "WHITEBALANCE_AUTO"},
		{VideoSettingLEDStatus, "LED_STATUS"},
		{VideoSettingExposureTime, "EXPOSURE_TIME"},
		{VideoSettingAnalogGain, "ANALOG_GAIN"},
		{VideoSettingDigitalGain, "DIGITAL_GAIN"},
		{VideoSettingAutoExposureTimeRange, "AUTO_EXPOSURE_TIME_RANGE"},
		{VideoSettingAutoAnalogGainRange, "AUTO_ANALOG_GAIN_RANGE"},
		{VideoSettingAutoDigitalGainRange, "AUTO_DIGITAL_GAIN_RANGE"},
		{VideoSettingExposureCompensation, "EXPOSURE_COMPENSATION"},
		{VideoSettingDenoising, "DENOISING"},
		{VideoSettingSceneIlluminance, "SCENE_ILLUMINANCE"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.setting.String(); got != tt.expected {
				t.Errorf("VideoSetting.String() = %q, want %q", got, tt.expected)
			}
		})
	}

	// Test unknown value
	t.Run("Unknown", func(t *testing.T) {
		unknown := VideoSetting(999)
		got := unknown.String()
		if got != "Unknown(999)" {
			t.Errorf("Expected Unknown(999), got %q", got)
		}
	})
}

// TestVideoSettingValueAuto tests the VideoSettingValueAuto constant.
func TestVideoSettingValueAuto(t *testing.T) {
	if VideoSettingValueAuto != -1 {
		t.Errorf("Expected VideoSettingValueAuto to be -1, got %d", VideoSettingValueAuto)
	}
}

// TestArrayRecording tests recording from all 4 ZED X cameras simultaneously.
// This test requires actual ZED hardware with 4 cameras connected.
// Recordings are saved to testdata/recordings/ for inspection.
func TestArrayRecording(t *testing.T) {
	// Skip if ZED SDK is not available
	if _, err := os.Stat("/usr/local/zed/lib"); os.IsNotExist(err) {
		t.Skip("ZED SDK not found at /usr/local/zed/lib, skipping hardware test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create output directory for test recordings in testdata
	testOutputDir := "testdata/recordings"
	os.MkdirAll(testOutputDir, 0755)

	// Map camera positions to serial numbers for stable identification across reboots
	cameraSerials := map[CameraPosition]uint{
		CameraPositionBack:  47684341,
		CameraPositionRight: 47714292,
		CameraPositionFront: 46768884,
		CameraPositionLeft:  47698110,
	}

	// Create 4 cameras (one for each position) using serial numbers
	cameras := make(map[CameraPosition]Camera)
	positions := AllCameraPositions()

	for i, pos := range positions {
		cfg := Config{
			Resolution:   ResolutionHD1080, // ZED X compatible
			FPS:          30,
			DepthMode:    DepthModeNone, // No depth for recording test
			CameraID:     i,
			SerialNumber: cameraSerials[pos],
			Logger:       logger,
		}
		cameras[pos] = NewCamera(cfg)
	}

	// Create the array
	arrayCfg := ArrayConfig{
		OutputDir: testOutputDir,
		Cameras:   cameras,
		RecordingParams: RecordingParameters{
			CompressionMode: SVOCompressionH265,
			Bitrate:         0, // Default
			TargetFPS:       0, // Use camera FPS
			Transcode:       false,
		},
		Logger: logger,
	}

	arr, err := NewArray(arrayCfg)
	if err != nil {
		t.Fatalf("Failed to create array: %v", err)
	}

	// Start the array (open all cameras)
	ctx := context.Background()
	err = arr.Start(ctx)
	if err != nil {
		t.Skipf("Failed to start camera array (hardware may not be connected): %v", err)
	}

	t.Log("All cameras opened successfully")

	// Record for 10 seconds
	recordingDuration := 10 * time.Second
	t.Logf("Recording for %v...", recordingDuration)

	recordCtx, cancel := context.WithTimeout(context.Background(), recordingDuration)
	defer cancel()

	err = arr.Record(recordCtx)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Recording failed: %v", err)
	}

	t.Log("Recording completed")

	// Verify recordings were created
	entries, err := os.ReadDir(testOutputDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No recording directories created")
	}

	// Check the latest recording directory
	recordingDir := filepath.Join(testOutputDir, entries[len(entries)-1].Name())
	t.Logf("Checking recording directory: %s", recordingDir)

	expectedFiles := []string{"back.svo2", "right.svo2", "front.svo2", "left.svo2"}
	for _, filename := range expectedFiles {
		filepath := filepath.Join(recordingDir, filename)
		info, err := os.Stat(filepath)
		if err != nil {
			t.Errorf("Recording file not found: %s", filepath)
			continue
		}
		t.Logf("  %s: %d bytes", filename, info.Size())
		if info.Size() == 0 {
			t.Errorf("Recording file is empty: %s", filename)
		}
	}
}

// TestArrayImage tests the Image() method that captures a 2x2 grid from all cameras.
// This test requires actual ZED hardware with 4 cameras connected.
// The resulting image is written to testdata/zed_array_image.png for visual inspection.
func TestArrayImage(t *testing.T) {
	// Skip if ZED SDK is not available
	if _, err := os.Stat("/usr/local/zed/lib"); os.IsNotExist(err) {
		t.Skip("ZED SDK not found at /usr/local/zed/lib, skipping hardware test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Map camera positions to serial numbers for stable identification across reboots
	cameraSerials := map[CameraPosition]uint{
		CameraPositionBack:  47684341,
		CameraPositionRight: 47714292,
		CameraPositionFront: 46768884,
		CameraPositionLeft:  47698110,
	}

	// Create 4 cameras (one for each position) using serial numbers
	cameras := make(map[CameraPosition]Camera)
	positions := AllCameraPositions()

	for i, pos := range positions {
		cfg := Config{
			Resolution:   ResolutionHD1080,
			FPS:          30,
			DepthMode:    DepthModeNone,
			CameraID:     i, // Still need unique camera IDs for the SDK internal tracking
			SerialNumber: cameraSerials[pos],
			Logger:       logger,
		}
		cameras[pos] = NewCamera(cfg)
	}

	// Create the array
	arrayCfg := ArrayConfig{
		Cameras: cameras,
		Logger:  logger,
	}

	arr, err := NewArray(arrayCfg)
	if err != nil {
		t.Fatalf("Failed to create array: %v", err)
	}

	// Start the array (open all cameras)
	ctx := context.Background()
	err = arr.Start(ctx)
	if err != nil {
		t.Skipf("Failed to start camera array (hardware may not be connected): %v", err)
	}

	// Close cameras when done
	defer func() {
		for _, pos := range positions {
			cameras[pos].Close()
		}
	}()

	t.Log("All cameras opened successfully")

	// Capture the 2x2 grid image
	t.Log("Capturing 2x2 grid image...")
	gridImg, err := arr.Image()
	if err != nil {
		t.Fatalf("Failed to capture grid image: %v", err)
	}

	t.Logf("Grid image dimensions: %dx%d", gridImg.Width, gridImg.Height)

	// Verify the dimensions are 2x the single camera resolution
	expectedWidth := 1920 * 2  // HD1080 width * 2
	expectedHeight := 1080 * 2 // HD1080 height * 2
	if gridImg.Width != expectedWidth {
		t.Errorf("Expected width %d, got %d", expectedWidth, gridImg.Width)
	}
	if gridImg.Height != expectedHeight {
		t.Errorf("Expected height %d, got %d", expectedHeight, gridImg.Height)
	}

	// Verify image data is not all zeros
	allZeros := true
	for _, b := range gridImg.Data {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Image data is all zeros, expected actual image data")
	}

	// Write the image to testdata directory for visual inspection
	testdataDir := "testdata"
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	outputPath := filepath.Join(testdataDir, "zed_array_image.png")
	t.Logf("Writing image to %s...", outputPath)

	// Convert to standard image.RGBA and encode as PNG
	rgba := gridImg.ToRGBA()
	f, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, rgba); err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}

	t.Logf("Image written to %s - layout: top-left=back, top-right=right, bottom-left=front, bottom-right=left", outputPath)
}

// TestCameraSettings tests the camera video settings functionality.
// This test requires actual ZED hardware and the ZED SDK to be installed.
// The test will be skipped if the hardware is not available.
func TestCameraSettings(t *testing.T) {
	// Skip if ZED SDK is not available
	if _, err := os.Stat("/usr/local/zed/lib"); os.IsNotExist(err) {
		t.Skip("ZED SDK not found at /usr/local/zed/lib, skipping hardware test")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := DefaultConfig()
	cfg.Logger = logger
	cfg.Resolution = ResolutionHD1080 // ZED X doesn't support HD720
	cfg.FPS = 30

	cam := NewCamera(cfg)

	err := cam.Open()
	if err != nil {
		t.Skipf("Failed to open ZED camera (hardware may not be connected): %v", err)
	}
	defer cam.Close()

	// Test IsSettingSupported for common settings
	t.Run("IsSettingSupported", func(t *testing.T) {
		// These settings should be supported on all ZED cameras
		commonSettings := []VideoSetting{
			VideoSettingBrightness,
			VideoSettingContrast,
			VideoSettingSaturation,
			VideoSettingSharpness,
			VideoSettingGamma,
			VideoSettingGain,
			VideoSettingExposure,
		}

		for _, setting := range commonSettings {
			supported := cam.IsSettingSupported(setting)
			t.Logf("Setting %s supported: %v", setting, supported)
		}
	})

	// Test GetSetting for readable settings
	t.Run("GetSetting", func(t *testing.T) {
		settingsToRead := []VideoSetting{
			VideoSettingBrightness,
			VideoSettingContrast,
			VideoSettingSaturation,
			VideoSettingSharpness,
			VideoSettingGamma,
			VideoSettingGain,
			VideoSettingExposure,
			VideoSettingAECAGC,
			VideoSettingWhiteBalanceTemperature,
			VideoSettingWhiteBalanceAuto,
			VideoSettingLEDStatus,
		}

		for _, setting := range settingsToRead {
			if !cam.IsSettingSupported(setting) {
				t.Logf("Skipping unsupported setting: %s", setting)
				continue
			}

			value, err := cam.GetSetting(setting)
			if err != nil {
				t.Logf("GetSetting(%s) returned error (may be expected for some cameras): %v", setting, err)
				continue
			}
			t.Logf("Setting %s = %d", setting, value)
		}
	})

	// Test SetSetting and GetSetting roundtrip
	t.Run("SetSetting roundtrip", func(t *testing.T) {
		// Test setting brightness if supported
		if !cam.IsSettingSupported(VideoSettingBrightness) {
			t.Skip("Brightness not supported on this camera")
		}

		// Get original value
		original, err := cam.GetSetting(VideoSettingBrightness)
		if err != nil {
			t.Fatalf("Failed to get original brightness: %v", err)
		}
		t.Logf("Original brightness: %d", original)

		// Set a new value (brightness is 0-8)
		newValue := 4
		if original == 4 {
			newValue = 5
		}

		err = cam.SetSetting(VideoSettingBrightness, newValue)
		if err != nil {
			t.Fatalf("Failed to set brightness to %d: %v", newValue, err)
		}

		// Read back the value
		readBack, err := cam.GetSetting(VideoSettingBrightness)
		if err != nil {
			t.Fatalf("Failed to read back brightness: %v", err)
		}

		if readBack != newValue {
			t.Errorf("Brightness roundtrip failed: set %d, got %d", newValue, readBack)
		} else {
			t.Logf("Brightness roundtrip successful: set %d, read %d", newValue, readBack)
		}

		// Restore original value
		err = cam.SetSetting(VideoSettingBrightness, original)
		if err != nil {
			t.Logf("Warning: failed to restore original brightness: %v", err)
		}
	})

	// Test auto exposure toggle
	t.Run("Auto exposure toggle", func(t *testing.T) {
		if !cam.IsSettingSupported(VideoSettingAECAGC) {
			t.Skip("AEC/AGC not supported on this camera")
		}

		// Get current auto mode
		autoMode, err := cam.GetSetting(VideoSettingAECAGC)
		if err != nil {
			t.Fatalf("Failed to get AEC/AGC mode: %v", err)
		}
		t.Logf("Current AEC/AGC mode: %d (0=manual, 1=auto)", autoMode)

		// Toggle the mode
		newMode := 1 - autoMode // Toggle between 0 and 1
		err = cam.SetSetting(VideoSettingAECAGC, newMode)
		if err != nil {
			t.Logf("Failed to toggle AEC/AGC mode (may be expected): %v", err)
		} else {
			t.Logf("Toggled AEC/AGC to: %d", newMode)

			// Restore original mode
			cam.SetSetting(VideoSettingAECAGC, autoMode)
		}
	})

	// Test ZED X-specific settings (may not be available on all cameras)
	t.Run("ZED X settings", func(t *testing.T) {
		zedXSettings := []VideoSetting{
			VideoSettingExposureTime,
			VideoSettingAnalogGain,
			VideoSettingDigitalGain,
			VideoSettingExposureCompensation,
			VideoSettingDenoising,
			VideoSettingSceneIlluminance,
		}

		for _, setting := range zedXSettings {
			if cam.IsSettingSupported(setting) {
				value, err := cam.GetSetting(setting)
				if err != nil {
					t.Logf("ZED X setting %s error: %v", setting, err)
				} else {
					t.Logf("ZED X setting %s = %d", setting, value)
				}
			} else {
				t.Logf("ZED X setting %s not supported (expected for non-ZED X cameras)", setting)
			}
		}
	})

	// Test range settings (ZED X only)
	t.Run("Range settings", func(t *testing.T) {
		rangeSettings := []VideoSetting{
			VideoSettingAutoExposureTimeRange,
			VideoSettingAutoAnalogGainRange,
			VideoSettingAutoDigitalGainRange,
		}

		for _, setting := range rangeSettings {
			if cam.IsSettingSupported(setting) {
				min, max, err := cam.GetSettingRange(setting)
				if err != nil {
					t.Logf("Range setting %s error: %v", setting, err)
				} else {
					t.Logf("Range setting %s = [%d, %d]", setting, min, max)
				}
			} else {
				t.Logf("Range setting %s not supported (expected for non-ZED X cameras)", setting)
			}
		}
	})
}
