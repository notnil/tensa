package court3d

import (
	"fmt"
	"math"

	"github.com/notnil/tensa/pkg/tennis/court2d"
)

// Point represents a 3D point with X, Y, and Z coordinates.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// String returns a string representation of the Point.
func (p Point) String() string {
	return fmt.Sprintf("(%.4f, %.4f, %.4f)", p.X, p.Y, p.Z)
}

// Distance calculates the Euclidean distance between two 3D points.
func (p Point) Distance(p2 Point) float64 {
	return math.Sqrt((p.X-p2.X)*(p.X-p2.X) + (p.Y-p2.Y)*(p.Y-p2.Y) + (p.Z-p2.Z)*(p.Z-p2.Z))
}

const (
	// NetHeightAtPost is the height of the net at the posts in meters (3.5 feet).
	NetHeightAtPost = 1.07
	// NetHeightAtCenter is the height of the net at the center in meters (3 feet).
	NetHeightAtCenter = 0.914
	// NetPostOffset is the distance of the net post outside the doubles sideline in meters (3 feet).
	NetPostOffset = 0.914
	// doublesHalfWidth is the standard half-width derived from court2d values (5.4864m)
	doublesHalfWidth = 5.4864
)

// KeyPointPosition converts a court2d.KeyPoint to a court3d.Point by retrieving the 2D coordinates and setting Z=0.
func KeyPointPosition(kp court2d.KeyPoint) Point {
	p2d := kp.Point()
	return Point{
		X: p2d.X,
		Y: p2d.Y,
		Z: 0,
	}
}

// LeftNetPostBase returns the base position of the left net post.
func LeftNetPostBase() Point {
	x := doublesHalfWidth + NetPostOffset
	return Point{X: -x, Y: 0, Z: 0}
}

// RightNetPostBase returns the base position of the right net post.
func RightNetPostBase() Point {
	x := doublesHalfWidth + NetPostOffset
	return Point{X: x, Y: 0, Z: 0}
}

// LeftNetPostTop returns the top position of the left net post.
func LeftNetPostTop() Point {
	base := LeftNetPostBase()
	return Point{X: base.X, Y: base.Y, Z: NetHeightAtPost}
}

// RightNetPostTop returns the top position of the right net post.
func RightNetPostTop() Point {
	base := RightNetPostBase()
	return Point{X: base.X, Y: base.Y, Z: NetHeightAtPost}
}

// NetCenterTop returns the point at the top center of the net (the strap).
func NetCenterTop() Point {
	return Point{X: 0, Y: 0, Z: NetHeightAtCenter}
}
