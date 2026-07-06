package drillutil

import "github.com/notnil/tensa/pkg/ai/drillsx/api"

type SubjectiveSpeed float64

const (
	Speed0  = SubjectiveSpeed(0.0)
	Speed1  = SubjectiveSpeed(1.0)
	Speed2  = SubjectiveSpeed(2.0)
	Speed3  = SubjectiveSpeed(3.0)
	Speed4  = SubjectiveSpeed(4.0)
	Speed5  = SubjectiveSpeed(5.0)
	Speed6  = SubjectiveSpeed(6.0)
	Speed7  = SubjectiveSpeed(7.0)
	Speed8  = SubjectiveSpeed(8.0)
	Speed9  = SubjectiveSpeed(9.0)
	Speed10 = SubjectiveSpeed(10.0)
)

type SubjectiveSpin float64

const (
	SpinNegFive  = SubjectiveSpin(-5.0)
	SpinNegFour  = SubjectiveSpin(-4.0)
	SpinNegThree = SubjectiveSpin(-3.0)
	SpinNegTwo   = SubjectiveSpin(-2.0)
	SpinNegOne   = SubjectiveSpin(-1.0)
	SpinZero     = SubjectiveSpin(0.0)
	SpinOne      = SubjectiveSpin(1.0)
	SpinTwo      = SubjectiveSpin(2.0)
	SpinThree    = SubjectiveSpin(3.0)
	SpinFour     = SubjectiveSpin(4.0)
	SpinFive     = SubjectiveSpin(5.0)
)

// Subjective calculates the top and bottom speeds for a given speed and spin.
// Based on reverse engineered playmate subjective speed and spin values in assets/playmate-tachometer.csv
func Subjective(speed SubjectiveSpeed, spin SubjectiveSpin, angle float64) api.Settings {
	if speed > Speed10 || speed < Speed0 {
		panic("speed out of range")
	}
	if spin > SpinFive || spin < SpinNegFive {
		panic("spin out of range")
	}
	top := 100 + (25 * float64(speed)) + (12.85 * float64(spin))
	bottom := 100 + (25 * float64(speed)) - (12.85 * float64(spin))
	return api.Settings{
		Top:    float64(top),
		Bottom: float64(bottom),
		Angle:  angle,
	}
}

// InverseSubjective converts raw motor speeds back to subjective speed and spin values.
// Given the equations:
//
//	top = 100 + 25*speed + 12.85*spin
//	bottom = 100 + 25*speed - 12.85*spin
//
// We can solve for:
//
//	speed = (top + bottom - 200) / 50
//	spin = (top - bottom) / 25.7
func InverseSubjective(top, bottom float64) (SubjectiveSpeed, SubjectiveSpin) {
	speed := (top + bottom - 200.0) / 50.0
	spin := (top - bottom) / 25.7
	return SubjectiveSpeed(speed), SubjectiveSpin(spin)
}
