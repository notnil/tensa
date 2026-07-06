// Package api - mirrored types and interfaces to eliminate external dependencies.
// These types are redeclared here so plugins don't need to import the concrete packages.
package api

import (
	"context"
	"io"
	"math"
	"math/rand"
)

// Point represents a 2D point with X and Y coordinates on the tennis court.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Distance calculates the Euclidean distance between two points.
func (p Point) Distance(p2 Point) float64 {
	return math.Sqrt((p.X-p2.X)*(p.X-p2.X) + (p.Y-p2.Y)*(p.Y-p2.Y))
}

// Halfway returns the midpoint between two points.
func (p Point) Halfway(p2 Point) Point {
	return Point{
		X: (p.X + p2.X) / 2,
		Y: (p.Y + p2.Y) / 2,
	}
}

// Angle returns the angle from this point to another point in radians.
// The angle is normalized to the range [-π, π].
func (p Point) Angle(p2 Point) float64 {
	angle := math.Atan2(p2.Y-p.Y, p2.X-p.X)
	// Normalize to [-π, π]
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

// Polygon represents a shape defined by a series of vertices.
type Polygon struct {
	Vertices []Point `json:"vertices"`
}

// IsValid checks if the polygon has at least three vertices and is convex.
func (p Polygon) IsValid() bool {
	n := len(p.Vertices)
	if n < 3 {
		return false
	}
	var initialSign float64
	foundNonZero := false
	for i := 0; i < n; i++ {
		a := p.Vertices[i]
		b := p.Vertices[(i+1)%n]
		c := p.Vertices[(i+2)%n]
		cross := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
		if cross != 0 {
			if !foundNonZero {
				initialSign = cross
				foundNonZero = true
			} else if cross*initialSign < 0 {
				return false
			}
		}
	}
	return foundNonZero
}

// RandomPoint generates a random point inside the polygon.
func (p Polygon) RandomPoint(r *rand.Rand) Point {
	bboxMin, bboxMax := p.boundingBox()
	for {
		randPoint := Point{
			X: bboxMin.X + r.Float64()*(bboxMax.X-bboxMin.X),
			Y: bboxMin.Y + r.Float64()*(bboxMax.Y-bboxMin.Y),
		}
		if p.containsPoint(randPoint) {
			return randPoint
		}
	}
}

func (p Polygon) boundingBox() (Point, Point) {
	var minX, maxX, minY, maxY float64
	for i, v := range p.Vertices {
		if i == 0 || v.X < minX {
			minX = v.X
		}
		if i == 0 || v.X > maxX {
			maxX = v.X
		}
		if i == 0 || v.Y < minY {
			minY = v.Y
		}
		if i == 0 || v.Y > maxY {
			maxY = v.Y
		}
	}
	return Point{X: minX, Y: minY}, Point{X: maxX, Y: maxY}
}

func (p Polygon) containsPoint(point Point) bool {
	var inside bool
	j := len(p.Vertices) - 1
	for i := 0; i < len(p.Vertices); i++ {
		if (p.Vertices[i].Y > point.Y) != (p.Vertices[j].Y > point.Y) &&
			point.X < (p.Vertices[j].X-p.Vertices[i].X)*(point.Y-p.Vertices[i].Y)/(p.Vertices[j].Y-p.Vertices[i].Y)+p.Vertices[i].X {
			inside = !inside
		}
		j = i
	}
	return inside
}

// Location represents a location with position and rotation.
type Location struct {
	Point    Point   `json:"point"`
	Rotation float64 `json:"rotation"`
}

// Range represents a numeric range with min and max values.
type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Settings contains the configuration parameters for the thrower.
type Settings struct {
	Top    float64 `json:"top"`    // Top motor speed in radians per second
	Bottom float64 `json:"bottom"` // Bottom motor speed in radians per second
	Angle  float64 `json:"angle"`  // Throw angle in radians
}

// Navigator defines the interface for navigation capabilities.
// Concrete implementations from pkg/ai/navigation automatically satisfy this interface.
type Navigator interface {
	Navigate(ctx context.Context, dest Location) error
}

// Mover defines the interface for low-level movement control.
// Concrete implementations from pkg/hware automatically satisfy this interface.
// This is used for direct movement commands without high-level navigation.
type Mover interface {
	// Move commands the mover to travel in a specified direction (in radians relative to its frame)
	// at the specified speed.
	Move(dir, speed float64) error
	// Rotate commands the mover to rotate at the specified speed.
	Rotate(speed float64) error
	// Stop commands the mover to stop moving.
	Stop() error
}

// Thrower defines the interface for controlling the ball throwing mechanism.
// Concrete implementations from pkg/hware/thrower automatically satisfy this interface.
type Thrower interface {
	Set(s Settings) error
	Throw(ctx context.Context) error
	Load(ctx context.Context) error
}

// AudioPlayer defines the interface for audio playback.
// Concrete implementations from pkg/audio automatically satisfy this interface.
type AudioPlayer interface {
	Play(r io.Reader) error
}

// Sub is a generic subscription interface for receiving messages.
// Concrete implementations from pkg/pubsubx automatically satisfy this interface.
type Sub[T any] interface {
	Subscribe(ctx context.Context, ch chan<- T) error
}
