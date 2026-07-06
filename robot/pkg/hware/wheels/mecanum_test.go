package wheels_test

import (
	"math"
	"testing"

	"github.com/notnil/tensa/pkg/hware/wheels"
	"github.com/notnil/tensa/pkg/util/rotation"
)

type MacanumTranslateTestCase struct {
	Name      string
	Direction float64
	Speed     float64
	Expected  wheels.Set[float64]
}

var MacanumTranslateTestCases = []MacanumTranslateTestCase{
	{
		Name:      "East",
		Direction: rotation.East,
		Speed:     1,
		Expected: wheels.Set[float64]{
			FrontLeft:  1,
			FrontRight: -1,
			RearLeft:   -1,
			RearRight:  1,
		},
	},
	{
		Name:      "North",
		Direction: rotation.North,
		Speed:     1,
		Expected: wheels.Set[float64]{
			FrontLeft:  1,
			FrontRight: 1,
			RearLeft:   1,
			RearRight:  1,
		},
	},
	{
		Name:      "West",
		Direction: rotation.West,
		Speed:     1,
		Expected: wheels.Set[float64]{
			FrontLeft:  -1,
			FrontRight: 1,
			RearLeft:   1,
			RearRight:  -1,
		},
	},
	{
		Name:      "South",
		Direction: rotation.South,
		Speed:     1,
		Expected: wheels.Set[float64]{
			FrontLeft:  -1,
			FrontRight: -1,
			RearLeft:   -1,
			RearRight:  -1,
		},
	},
	{
		Name:      "NorthEast",
		Direction: rotation.NorthEast,
		Speed:     1,
		Expected: wheels.Set[float64]{
			FrontLeft:  1,
			FrontRight: 0,
			RearLeft:   0,
			RearRight:  1,
		},
	},
}

func roundDriveConfig(cfg wheels.Set[float64]) wheels.Set[float64] {
	return wheels.Set[float64]{
		FrontLeft:  math.Round(cfg.FrontLeft),
		FrontRight: math.Round(cfg.FrontRight),
		RearLeft:   math.Round(cfg.RearLeft),
		RearRight:  math.Round(cfg.RearRight),
	}
}

func TestMacanumTranslate(t *testing.T) {
	for _, tc := range MacanumTranslateTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			cfg := wheels.MacanumTranslate(tc.Direction, tc.Speed)
			cfg = roundDriveConfig(cfg)
			if cfg != tc.Expected {
				t.Fatalf("Test case %q failed\nExpected: %+v\nGot: %+v", tc.Name, tc.Expected, cfg)
			}
		})
	}
}

type MacanumRotateTestCase struct {
	Speed    float64
	Expected wheels.Set[float64]
}

var MacanumRotateTestCases = []MacanumRotateTestCase{
	{
		Speed: -1,
		Expected: wheels.Set[float64]{
			FrontLeft:  1,
			FrontRight: -1,
			RearLeft:   1,
			RearRight:  -1,
		},
	},
	{
		Speed: 1,
		Expected: wheels.Set[float64]{
			FrontLeft:  -1,
			FrontRight: 1,
			RearLeft:   -1,
			RearRight:  1,
		},
	},
}

func TestMacanumRotate(t *testing.T) {
	for _, tc := range MacanumRotateTestCases {
		cfg := wheels.MacanumRotate(tc.Speed)
		cfg = roundDriveConfig(cfg)
		if cfg != tc.Expected {
			t.Fatalf("Expected: %+v, got: %+v", tc.Expected, cfg)
		}
	}
}
