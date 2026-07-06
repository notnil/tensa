package drillutil

import (
	"math"
	"testing"
)

func TestInverseSubjective(t *testing.T) {
	tests := []struct {
		name  string
		speed SubjectiveSpeed
		spin  SubjectiveSpin
		angle float64
	}{
		{
			name:  "Zero speed and spin",
			speed: Speed0,
			spin:  SpinZero,
			angle: 0.5,
		},
		{
			name:  "Medium speed with topspin",
			speed: Speed5,
			spin:  SpinThree,
			angle: 0.3,
		},
		{
			name:  "High speed with backspin",
			speed: Speed8,
			spin:  SpinNegThree,
			angle: 0.4,
		},
		{
			name:  "Max speed with max topspin",
			speed: Speed10,
			spin:  SpinFive,
			angle: 0.2,
		},
		{
			name:  "Max speed with max backspin",
			speed: Speed10,
			spin:  SpinNegFive,
			angle: 0.6,
		},
		{
			name:  "Low speed with negative spin",
			speed: Speed2,
			spin:  SpinNegTwo,
			angle: 0.35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert subjective to raw motor speeds
			settings := Subjective(tt.speed, tt.spin, tt.angle)

			// Convert back using inverse
			gotSpeed, gotSpin := InverseSubjective(settings.Top, settings.Bottom)

			// Check that we get back the original values (with small floating point tolerance)
			tolerance := 0.0001
			if math.Abs(float64(gotSpeed-tt.speed)) > tolerance {
				t.Errorf("InverseSubjective() speed = %v, want %v", gotSpeed, tt.speed)
			}
			if math.Abs(float64(gotSpin-tt.spin)) > tolerance {
				t.Errorf("InverseSubjective() spin = %v, want %v", gotSpin, tt.spin)
			}
		})
	}
}

func TestSubjectiveInverseRoundTrip(t *testing.T) {
	// Test that Subjective and InverseSubjective are true inverses
	for speed := 0.0; speed <= 10.0; speed += 0.5 {
		for spin := -5.0; spin <= 5.0; spin += 0.5 {
			settings := Subjective(SubjectiveSpeed(speed), SubjectiveSpin(spin), 0.3)
			gotSpeed, gotSpin := InverseSubjective(settings.Top, settings.Bottom)

			tolerance := 0.0001
			if math.Abs(float64(gotSpeed)-speed) > tolerance {
				t.Errorf("Round trip failed: speed %v -> %v", speed, gotSpeed)
			}
			if math.Abs(float64(gotSpin)-spin) > tolerance {
				t.Errorf("Round trip failed: spin %v -> %v", spin, gotSpin)
			}
		}
	}
}

func TestInverseSubjectiveWithArbitraryValues(t *testing.T) {
	// Test with arbitrary motor speeds
	tests := []struct {
		name      string
		top       float64
		bottom    float64
		wantSpeed float64
		wantSpin  float64
		tolerance float64
	}{
		{
			name:      "Equal speeds (no spin)",
			top:       200.0,
			bottom:    200.0,
			wantSpeed: 4.0, // (200 + 200 - 200) / 50 = 4.0
			wantSpin:  0.0,
			tolerance: 0.0001,
		},
		{
			name:      "Top faster (topspin)",
			top:       250.0,
			bottom:    200.0,
			wantSpeed: 5.0, // (250 + 200 - 200) / 50 = 5.0
			wantSpin:  50.0 / 25.7,
			tolerance: 0.01,
		},
		{
			name:      "Bottom faster (backspin)",
			top:       200.0,
			bottom:    250.0,
			wantSpeed: 5.0, // (200 + 250 - 200) / 50 = 5.0
			wantSpin:  -50.0 / 25.7,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSpeed, gotSpin := InverseSubjective(tt.top, tt.bottom)

			if math.Abs(float64(gotSpeed)-tt.wantSpeed) > tt.tolerance {
				t.Errorf("InverseSubjective(%v, %v) speed = %v, want %v",
					tt.top, tt.bottom, gotSpeed, tt.wantSpeed)
			}
			if math.Abs(float64(gotSpin)-tt.wantSpin) > tt.tolerance {
				t.Errorf("InverseSubjective(%v, %v) spin = %v, want %v",
					tt.top, tt.bottom, gotSpin, tt.wantSpin)
			}
		})
	}
}
