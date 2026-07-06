package court2d

import "math"

// The `direction` is measured in radians with 0 being directly right (positive X direction),
// pi/2 (90°) being forward (positive Y direction), etc. Valid values are between 0 and 2*pi.
const (
	Right = 0
	Up    = math.Pi / 2
	Left  = math.Pi
	Down  = (3 * math.Pi) / 2
)
