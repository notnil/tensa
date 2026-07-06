package court2d_test

import (
	"math"
	"testing"

	"github.com/notnil/tensa/pkg/tennis/court2d"
)

func TestPointDistance(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2   court2d.Point
		expected float64
	}{
		{"Same point", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 0}, 0},
		{"Horizontal", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 3, Y: 0}, 3},
		{"Vertical", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 4}, 4},
		{"Diagonal", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 3, Y: 4}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p1.Distance(tt.p2); got != tt.expected {
				t.Errorf("Distance() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPointHalfway(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2   court2d.Point
		expected court2d.Point
	}{
		{"Same point", court2d.Point{X: 1, Y: 1}, court2d.Point{X: 1, Y: 1}, court2d.Point{X: 1, Y: 1}},
		{"Horizontal", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 4, Y: 0}, court2d.Point{X: 2, Y: 0}},
		{"Vertical", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 6}, court2d.Point{X: 0, Y: 3}},
		{"Diagonal", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 2, Y: 2}, court2d.Point{X: 1, Y: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p1.Halfway(tt.p2); got != tt.expected {
				t.Errorf("Halfway() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPointAngle(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2   court2d.Point
		expected float64
	}{
		{"Same point", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 0}, 0},
		{"Right", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 1, Y: 0}, 0},
		{"Up", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 1}, math.Pi / 2},
		{"Left", court2d.Point{X: 0, Y: 0}, court2d.Point{X: -1, Y: 0}, math.Pi},
		{"Down", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: -1}, 3 * math.Pi / 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p1.Angle(tt.p2); math.Abs(got-tt.expected) > 1e-6 {
				t.Errorf("Angle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPointDestination(t *testing.T) {
	tests := []struct {
		name     string
		p        court2d.Point
		angle    float64
		distance float64
		expected court2d.Point
	}{
		{
			name:     "Right",
			p:        court2d.Point{X: 0, Y: 0},
			angle:    0,
			distance: 1,
			expected: court2d.Point{X: 1, Y: 0},
		},
		{
			name:     "Up",
			p:        court2d.Point{X: 0, Y: 0},
			angle:    math.Pi / 2.0,
			distance: 1,
			expected: court2d.Point{X: 0, Y: 1},
		},
		{
			name:     "Left",
			p:        court2d.Point{X: 0, Y: 0},
			angle:    math.Pi,
			distance: 1,
			expected: court2d.Point{X: -1, Y: 0},
		},
		{
			name:     "Down",
			p:        court2d.Point{X: 0, Y: 0},
			angle:    3 * math.Pi / 2,
			distance: 1,
			expected: court2d.Point{X: 0, Y: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.Destination(tt.angle, tt.distance)
			if math.Abs(got.X-tt.expected.X) > 1e-6 || math.Abs(got.Y-tt.expected.Y) > 1e-6 {
				t.Errorf("Destination(%s, %f, %f) = %v, want %v", tt.name, tt.angle, tt.distance, got, tt.expected)
			}
		})
	}
}

func TestPointMirror(t *testing.T) {
	tests := []struct {
		name     string
		point    court2d.Point
		expected court2d.Point
	}{
		{"Origin", court2d.Point{X: 0, Y: 0}, court2d.Point{X: 0, Y: 0}},
		{"Positive X", court2d.Point{X: 3, Y: 2}, court2d.Point{X: -3, Y: 2}},
		{"Negative X", court2d.Point{X: -5, Y: 7}, court2d.Point{X: 5, Y: 7}},
		{"Positive Y", court2d.Point{X: 4, Y: 6}, court2d.Point{X: -4, Y: 6}},
		{"Negative Y", court2d.Point{X: 2, Y: -3}, court2d.Point{X: -2, Y: -3}},
		{"Both negative", court2d.Point{X: -1, Y: -1}, court2d.Point{X: 1, Y: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.point.Mirror(); got != tt.expected {
				t.Errorf("Mirror() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolygonContainsPoint(t *testing.T) {
	square := court2d.Polygon{
		Vertices: []court2d.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	tests := []struct {
		name     string
		polygon  court2d.Polygon
		point    court2d.Point
		expected bool
	}{
		{"Inside", square, court2d.Point{X: 1, Y: 1}, true},
		{"On edge", square, court2d.Point{X: 1, Y: 0}, true},
		{"Outside", square, court2d.Point{X: 3, Y: 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.polygon.ContainsPoint(tt.point); got != tt.expected {
				t.Errorf("ContainsPoint() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolygonClockwiseContainsPoint(t *testing.T) {
	square := court2d.Polygon{
		Vertices: []court2d.Point{
			{X: 0, Y: 0},
			{X: 0, Y: 2},
			{X: 2, Y: 2},
			{X: 2, Y: 0},
		},
	}

	tests := []struct {
		name     string
		polygon  court2d.Polygon
		point    court2d.Point
		expected bool
	}{
		{"Inside", square, court2d.Point{X: 1, Y: 1}, true},
		{"On edge", square, court2d.Point{X: 1, Y: 0}, true},
		{"Outside", square, court2d.Point{X: 3, Y: 3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.polygon.ContainsPoint(tt.point); got != tt.expected {
				t.Errorf("ContainsPoint() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolygonBoundingBox(t *testing.T) {
	polygon := court2d.Polygon{
		Vertices: []court2d.Point{
			{X: 0, Y: 0},
			{X: 2, Y: 0},
			{X: 2, Y: 2},
			{X: 0, Y: 2},
		},
	}

	expectedMin := court2d.Point{X: 0, Y: 0}
	expectedMax := court2d.Point{X: 2, Y: 2}

	gotMin, gotMax := polygon.BoundingBox()

	if gotMin != expectedMin || gotMax != expectedMax {
		t.Errorf("BoundingBox() = (%v, %v), want (%v, %v)", gotMin, gotMax, expectedMin, expectedMax)
	}
}

func TestPolygonIsValid(t *testing.T) {
	tests := []struct {
		name     string
		polygon  court2d.Polygon
		expected bool
	}{
		{
			name: "Valid square",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 0, Y: 2},
					{X: 2, Y: 2},
					{X: 2, Y: 0},
				},
			},
			expected: true,
		},
		{
			name: "Valid triangle",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 0, Y: 1},
				},
			},
			expected: true,
		},
		{
			name: "Invalid - fewer than 3 vertices",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 1, Y: 1},
				},
			},
			expected: false,
		},
		{
			name: "Invalid - collinear points",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 1, Y: 1},
					{X: 2, Y: 2},
				},
			},
			expected: false,
		},
		{
			name: "Invalid - concave polygon",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 2, Y: 0},
					{X: 1, Y: 1},
					{X: 2, Y: 2},
					{X: 0, Y: 2},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.polygon.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPolygonMirror(t *testing.T) {
	tests := []struct {
		name     string
		polygon  court2d.Polygon
		expected court2d.Polygon
	}{
		{
			name: "Square",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 1, Y: 1},
					{X: 3, Y: 1},
					{X: 3, Y: 3},
					{X: 1, Y: 3},
				},
			},
			expected: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: -1, Y: 1},
					{X: -3, Y: 1},
					{X: -3, Y: 3},
					{X: -1, Y: 3},
				},
			},
		},
		{
			name: "Triangle",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: 2, Y: 0},
					{X: 1, Y: 2},
				},
			},
			expected: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 0, Y: 0},
					{X: -2, Y: 0},
					{X: -1, Y: 2},
				},
			},
		},
		{
			name: "Mixed coordinates",
			polygon: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: -2, Y: -1},
					{X: 3, Y: -1},
					{X: 3, Y: 2},
					{X: -2, Y: 2},
				},
			},
			expected: court2d.Polygon{
				Vertices: []court2d.Point{
					{X: 2, Y: -1},
					{X: -3, Y: -1},
					{X: -3, Y: 2},
					{X: 2, Y: 2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.polygon.Mirror()
			if len(got.Vertices) != len(tt.expected.Vertices) {
				t.Errorf("Mirror() returned polygon with %d vertices, want %d", len(got.Vertices), len(tt.expected.Vertices))
				return
			}
			for i, vertex := range got.Vertices {
				if vertex != tt.expected.Vertices[i] {
					t.Errorf("Mirror() vertex[%d] = %v, want %v", i, vertex, tt.expected.Vertices[i])
				}
			}
		})
	}
}
