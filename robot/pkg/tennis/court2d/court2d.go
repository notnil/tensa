package court2d

//go:generate stringer -type=KeyPoint,Section

import (
	"math"
)

// KeyPoint is one of an enumeration of key points of a tennis court.
type KeyPoint int8

const (
	KP1 KeyPoint = iota
	KP2
	KP3
	KP4
	KP5
	KP6
	KP7
	KP8
	KP9
	KP10
	KP11
	KP12
	KP13
	KP14
	KP15
	KP16
	KP17
	KP18
	KP19
	KP20
	KP21
)

var (
	keyPoints         = []KeyPoint{KP1, KP2, KP3, KP4, KP5, KP6, KP7, KP8, KP9, KP10, KP11, KP12, KP13, KP14, KP15, KP16, KP17, KP18, KP19, KP20, KP21}
	keyPointsToPoints = []Point{}
)

func init() {
	keyPointsToPoints = []Point{
		{X: -5.4864, Y: 11.8872}, {X: -4.1148, Y: 11.8872}, {X: 0.0000, Y: 11.8872}, {X: 4.1148, Y: 11.8872}, {X: 5.4864, Y: 11.8872},
		{X: -4.1148, Y: 6.4008}, {X: 0.0000, Y: 6.4008}, {X: 4.1148, Y: 6.4008},
		{X: -5.4864, Y: 0.0000}, {X: -4.1148, Y: 0.0000}, {X: 0.0000, Y: 0.0000}, {X: 4.1148, Y: 0.0000}, {X: 5.4864, Y: 0.0000},
		{X: -4.1148, Y: -6.4008}, {X: 0.0000, Y: -6.4008}, {X: 4.1148, Y: -6.4008},
		{X: -5.4864, Y: -11.8872}, {X: -4.1148, Y: -11.8872}, {X: 0.0000, Y: -11.8872}, {X: 4.1148, Y: -11.8872}, {X: 5.4864, Y: -11.8872},
	}
}

// KeyPoints returns all of the KeyPoints on the tennis court.
func KeyPoints() []KeyPoint {
	return append([]KeyPoint(nil), keyPoints...)
}

// Point returns the point in the court's coordinate system
func (kp KeyPoint) Point() Point {
	return keyPointsToPoints[kp]
}

// Section is a recognized region of the tennis court.  The two half courts that make
// up a tennis court are designated near and far as seen from the user.  If the court
// is shown in a top-down view, the far court is on top.
type Section int

const (
	FarAdDoublesAlley Section = iota
	FarAdServiceBox
	FarBackcourt
	FarDeuceDoublesAlley
	FarDeuceServiceBox
	NearAdDoublesAlley
	NearAdServiceBox
	NearBackcourt
	NearDeuceDoublesAlley
	NearDeuceServiceBox
	FarSinglesCourt
	FarDoublesCourt
	NearSinglesCourt
	NearDoublesCourt
	SinglesCourt
	DoublesCourt
)

var primarySections = []Section{FarAdDoublesAlley, FarAdServiceBox, FarBackcourt, FarDeuceDoublesAlley, FarDeuceServiceBox,
	NearAdServiceBox, NearAdServiceBox, NearBackcourt, NearDeuceDoublesAlley, NearDeuceServiceBox}

// PrimarySections returns all of the primary (not compound) sections of a tennis court.
func PrimarySections() []Section {
	return append([]Section(nil), primarySections...)
}

var compoundSections = []Section{FarSinglesCourt, FarDoublesCourt, NearSinglesCourt, NearDoublesCourt, SinglesCourt, DoublesCourt}

// CompoundSections returns all of the compound sections of a tennis court.
func CompoundSections() []Section {
	return append([]Section(nil), primarySections...)
}

var sections = []Section{FarAdDoublesAlley, FarAdServiceBox, FarBackcourt, FarDeuceDoublesAlley, FarDeuceServiceBox,
	NearAdServiceBox, NearAdServiceBox, NearBackcourt, NearDeuceDoublesAlley, NearDeuceServiceBox,
	FarSinglesCourt, FarDoublesCourt, NearSinglesCourt, NearDoublesCourt, SinglesCourt, DoublesCourt,
}

// Sections returns all of the sections of a tennis court.
func Sections() []Section {
	return append([]Section(nil), primarySections...)
}

// KeyPoints returns the four corner KeyPoints of a Section.
func (s Section) KeyPoints() [4]KeyPoint {
	switch s {
	case FarAdDoublesAlley:
		return [4]KeyPoint{KP4, KP5, KP13, KP12}
	case FarAdServiceBox:
		return [4]KeyPoint{KP7, KP8, KP12, KP11}
	case FarBackcourt:
		return [4]KeyPoint{KP2, KP4, KP8, KP6}
	case FarDeuceDoublesAlley:
		return [4]KeyPoint{KP1, KP2, KP10, KP9}
	case FarDeuceServiceBox:
		return [4]KeyPoint{KP6, KP7, KP11, KP10}
	case NearAdDoublesAlley:
		return [4]KeyPoint{KP9, KP10, KP18, KP17}
	case NearAdServiceBox:
		return [4]KeyPoint{KP10, KP11, KP15, KP14}
	case NearBackcourt:
		return [4]KeyPoint{KP14, KP16, KP20, KP18}
	case NearDeuceDoublesAlley:
		return [4]KeyPoint{KP12, KP13, KP21, KP20}
	case NearDeuceServiceBox:
		return [4]KeyPoint{KP11, KP12, KP16, KP15}
	case FarSinglesCourt:
		return [4]KeyPoint{KP2, KP4, KP12, KP10}
	case FarDoublesCourt:
		return [4]KeyPoint{KP1, KP5, KP13, KP9}
	case NearSinglesCourt:
		return [4]KeyPoint{KP10, KP12, KP20, KP18}
	case NearDoublesCourt:
		return [4]KeyPoint{KP9, KP13, KP21, KP17}
	case SinglesCourt:
		return [4]KeyPoint{KP2, KP4, KP20, KP18}
	case DoublesCourt:
		return [4]KeyPoint{KP1, KP5, KP21, KP17}
	default:
		panic("no points for section")
	}
}

// Rectangle returns the Rectangle of the Section
func (s Section) Rectangle() Polygon {
	xMin, xMax, yMin, yMax := math.MaxFloat64, -math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64
	for _, kp := range s.KeyPoints() {
		x, y := kp.Point().X, kp.Point().Y
		xMin = math.Min(xMin, x)
		xMax = math.Max(xMax, x)
		yMin = math.Min(yMin, y)
		yMax = math.Max(yMax, y)
	}
	cords := []Point{
		{X: xMin, Y: yMin},
		{X: xMax, Y: yMin},
		{X: xMax, Y: yMax},
		{X: xMin, Y: yMax},
		{X: xMin, Y: yMin},
	}
	return Polygon{Vertices: cords}
}
