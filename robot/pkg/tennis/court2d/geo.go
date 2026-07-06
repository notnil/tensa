// Package court2d provides geometric primitives and operations for 2D tennis court calculations.
package court2d

import (
	"math"
	"math/rand"

	"github.com/notnil/tensa/pkg/util/rotation"
)

// Point represents a 2D point with X and Y coordinates.
// (0, 0) is the center of the court.
// X increases to the right and is parallel to the net.
// Y increases upwards and is perpendicular to the net.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Pt returns a point with the given coordinates.
func Pt(x, y float64) Point {
	return Point{X: x, Y: y}
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

// Angle returns the angle from the origin point to the target point in radians.
func (p Point) Angle(p2 Point) float64 {
	angle := math.Atan2(p2.Y-p.Y, p2.X-p.X)
	return rotation.Normalize(angle)
}

// Destination calculates a new point given an angle and distance from the origin point.
func (p Point) Destination(angle float64, distance float64) Point {
	return Point{
		X: p.X + math.Cos(angle)*distance,
		Y: p.Y + math.Sin(angle)*distance,
	}
}

// Mirror returns a new point with the X coordinate negated (mirrored across the Y-axis).
func (p Point) Mirror() Point {
	return Point{
		X: -p.X,
		Y: p.Y,
	}
}

// Polygon represents a shape defined by a series of vertices.
// The first and last vertices are not the same.
// The vertices can be in clockwise or counterclockwise order.
// The polygon must be convex.
type Polygon struct {
	Vertices []Point `json:"vertices"`
}

// IsValid checks if the polygon has at least three vertices and is convex.
// It returns false if there are fewer than three vertices, if the points are collinear,
// or if the vertices do not consistently turn in the same direction.
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
		// Compute the cross product of (b - a) and (c - b)
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
	// If all computed cross products are zero then the points are collinear (degenerate)
	return foundNonZero
}

// ContainsPoint checks if a given point is inside the polygon using the ray-casting algorithm.
func (p Polygon) ContainsPoint(point Point) bool {
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

// BoundingBox returns the minimal bounding box that contains the polygon.
func (p Polygon) BoundingBox() (Point, Point) {
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

// RandomPoint generates a random point inside the polygon.
func (p Polygon) RandomPoint(r *rand.Rand) Point {
	bboxMin, bboxMax := p.BoundingBox()
	for {
		randPoint := Point{
			X: bboxMin.X + r.Float64()*(bboxMax.X-bboxMin.X),
			Y: bboxMin.Y + r.Float64()*(bboxMax.Y-bboxMin.Y),
		}
		if p.ContainsPoint(randPoint) {
			return randPoint
		}
	}
}

// Mirror returns a new polygon with all vertices mirrored across the Y-axis.
// Each vertex has its X coordinate negated while Y coordinates remain unchanged.
func (p Polygon) Mirror() Polygon {
	mirrored := make([]Point, len(p.Vertices))
	for i, vertex := range p.Vertices {
		mirrored[i] = vertex.Mirror()
	}
	return Polygon{Vertices: mirrored}
}

// Intersect finds the intersection polygon of two convex polygons.
func (p1 Polygon) Intersect(p2 Polygon) Polygon {
	outputList := p1.Vertices

	for j := 0; j < len(p2.Vertices); j++ {
		inputList := outputList
		outputList = []Point{}

		s := p2.Vertices[(j+len(p2.Vertices)-1)%len(p2.Vertices)]
		e := p2.Vertices[j]

		for i := 0; i < len(inputList); i++ {
			currentPoint := inputList[i]
			prevPoint := inputList[(i+len(inputList)-1)%len(inputList)]

			if isLeft(s, e, currentPoint) {
				if !isLeft(s, e, prevPoint) {
					outputList = append(outputList, intersection(s, e, prevPoint, currentPoint))
				}
				outputList = append(outputList, currentPoint)
			} else if isLeft(s, e, prevPoint) {
				outputList = append(outputList, intersection(s, e, prevPoint, currentPoint))
			}
		}
	}

	return Polygon{Vertices: outputList}
}

// isLeft checks if a point is to the left of an edge.
func isLeft(a, b, c Point) bool {
	return (b.X-a.X)*(c.Y-a.Y)-(b.Y-a.Y)*(c.X-a.X) > 0
}

// intersection computes the intersection point of two lines (not line segments).
func intersection(a1, a2, b1, b2 Point) Point {
	// Line AB represented as a1x + b1y = c1
	a1Coeff := a2.Y - a1.Y
	b1Coeff := a1.X - a2.X
	c1 := a1Coeff*a1.X + b1Coeff*a1.Y

	// Line CD represented as a2x + b2y = c2
	a2Coeff := b2.Y - b1.Y
	b2Coeff := b1.X - b2.X
	c2 := a2Coeff*b1.X + b2Coeff*b1.Y

	determinant := a1Coeff*b2Coeff - a2Coeff*b1Coeff
	if determinant == 0 {
		// Lines are parallel, return an arbitrary point
		return Point{X: -1, Y: -1} // Using -1, -1 to indicate no intersection
	} else {
		x := (b2Coeff*c1 - b1Coeff*c2) / determinant
		y := (a1Coeff*c2 - a2Coeff*c1) / determinant
		return Point{X: x, Y: y}
	}
}
