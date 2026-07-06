package zrt

// Ball represents a tracked tennis ball with 3D coordinates.
type Ball struct {
	ID int     // Unique identifier for this ball detection
	X  float64 // X coordinate in meters (camera frame)
	Y  float64 // Y coordinate in meters (camera frame)
	Z  float64 // Z coordinate in meters (depth from camera)
}

// config holds configuration for the ZRT pipeline.
type config struct {
	camID   int
	svoPath string
	width   int
	height  int
}

// Option is a functional option for configuring the ZRT pipeline.
type Option func(*config)

// WithCamera sets the ZED camera ID (default: 0).
func WithCamera(id int) Option {
	return func(c *config) {
		c.camID = id
	}
}

// WithSVO sets the path to an SVO recording file for playback instead of live camera.
func WithSVO(path string) Option {
	return func(c *config) {
		c.svoPath = path
	}
}

// WithResolution sets the image resolution (default: 640x640).
// Must match the model input size.
func WithResolution(width, height int) Option {
	return func(c *config) {
		c.width = width
		c.height = height
	}
}

// detection represents a single object detection result with depth.
type detection struct {
	X, Y, W, H float32
	Confidence float32
	ClassID    int
	Depth      float32 // Depth in meters
}

// pixelTo3D converts 2D pixel coordinates and depth to 3D coordinates in camera frame.
// Uses simplified pinhole camera model with typical ZED camera parameters.
func pixelTo3D(px, py, depth float64, width, height int) (x, y, z float64) {
	if depth <= 0 || depth > 20.0 { // Invalid depth or too far
		return 0, 0, 0
	}

	// ZED HD720 typical horizontal FOV is ~90 degrees
	// Focal length approximation: f_x = width / (2 * tan(HFOV/2))
	// For 90 deg HFOV: f_x ≈ width / 2
	fx := float64(width) / 2.0
	fy := fx // Assume square pixels

	// Principal point (image center)
	cx := float64(width) / 2.0
	cy := float64(height) / 2.0

	// Convert pixel to normalized camera coordinates
	// X = (px - cx) * Z / fx
	// Y = (py - cy) * Z / fy
	// Z = depth
	z = depth
	x = (px - cx) * z / fx
	y = (py - cy) * z / fy

	return x, y, z
}
