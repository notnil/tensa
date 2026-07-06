package court2d_test

import (
	"testing"

	"github.com/notnil/tensa/pkg/tennis/court2d"
)

func TestSectionContains(t *testing.T) {
	var tests = []struct {
		pt   court2d.Point
		secs []court2d.Section
	}{
		{
			court2d.Point{X: -1.0, Y: 1.0},
			[]court2d.Section{
				court2d.FarDeuceServiceBox, court2d.FarSinglesCourt, court2d.FarDoublesCourt, court2d.SinglesCourt, court2d.DoublesCourt,
			},
		},
		{
			court2d.Point{X: 1.0, Y: 1.0},
			[]court2d.Section{
				court2d.FarAdServiceBox, court2d.FarSinglesCourt, court2d.FarDoublesCourt, court2d.SinglesCourt, court2d.DoublesCourt,
			},
		},
	}
	for _, tt := range tests {
		for _, sec := range tt.secs {
			b := sec.Rectangle().ContainsPoint(tt.pt)
			if !b {
				t.Errorf("point %v should be in section %s", tt.pt, sec)
			}
		}
	}
}
