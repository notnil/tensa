// Package drillutil provides utility functions for drill plugins, including random
// sampling and geometry helpers. These are optional conveniences that plugins can
// choose to use or not - they are not part of the core API boundary.
package drillutil

import (
	"fmt"
	"math/rand"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

// Uniform returns a random float64 in the range [min, max].
// It validates that min <= max.
func Uniform(r *rand.Rand, min, max float64) (float64, error) {
	if min > max {
		return 0, fmt.Errorf("invalid range: min (%v) > max (%v)", min, max)
	}
	return min + r.Float64()*(max-min), nil
}

// Fixed returns the provided value unchanged.
// This is useful for maintaining consistent API when switching between fixed and random values.
func Fixed(value float64) float64 {
	return value
}

// SampleBox returns a random point within a rectangular box defined by min and max points.
// It validates that min.X <= max.X and min.Y <= max.Y.
func SampleBox(r *rand.Rand, min, max api.Point) (api.Point, error) {
	if min.X > max.X || min.Y > max.Y {
		return api.Point{}, fmt.Errorf("invalid box: min (%v) exceeds max (%v)", min, max)
	}
	return api.Point{
		X: min.X + r.Float64()*(max.X-min.X),
		Y: min.Y + r.Float64()*(max.Y-min.Y),
	}, nil
}

// SamplePolygon returns a random point within the given polygon.
// It validates that the polygon is structurally valid (≥ 3 points).
func SamplePolygon(r *rand.Rand, polygon api.Polygon) (api.Point, error) {
	if len(polygon.Vertices) < 3 {
		return api.Point{}, fmt.Errorf("polygon must have at least 3 vertices, got %d", len(polygon.Vertices))
	}
	if !polygon.IsValid() {
		return api.Point{}, fmt.Errorf("polygon is not valid (collinear or non-convex)")
	}
	return polygon.RandomPoint(r), nil
}

// SampleMultiPolygon returns a random point from one of the provided polygons.
// The polygon is selected randomly with equal probability.
// It validates that the list is non-empty and each polygon is valid.
func SampleMultiPolygon(r *rand.Rand, polygons []api.Polygon) (api.Point, error) {
	if len(polygons) == 0 {
		return api.Point{}, fmt.Errorf("multi-polygon list must be non-empty")
	}
	// Pick a random polygon
	idx := r.Intn(len(polygons))
	return SamplePolygon(r, polygons[idx])
}

// MakeLoc creates a Location with the given point and rotation.
// This is a convenience function for building location values for navigation and feeder configs.
func MakeLoc(point api.Point, rotation float64) api.Location {
	return api.Location{
		Point:    point,
		Rotation: rotation,
	}
}

// ValidateRange validates that a numeric range satisfies min <= max.
func ValidateRange(r api.Range) error {
	if r.Min > r.Max {
		return fmt.Errorf("invalid range: min (%v) > max (%v)", r.Min, r.Max)
	}
	return nil
}

// SampleRange returns a random value within the given range [min, max].
func SampleRange(rng *rand.Rand, r api.Range) (float64, error) {
	if err := ValidateRange(r); err != nil {
		return 0, err
	}
	return Uniform(rng, r.Min, r.Max)
}
