package thrower

import (
	"fmt"
	"math"
)

// Settings contains the configuration parameters for the throw system motors.
// All values are in radians/radians per second and must be within specified ranges.
type Settings struct {
	Top    float64 `json:"top"`    // Top is the speed of the top throw motor in radians per second (must be positive)
	Bottom float64 `json:"bottom"` // Bottom is the speed of the bottom throw motor in radians per second (must be positive)
	Angle  float64 `json:"angle"`  // Angle is the throw angle in radians (must be between 0 and π/4)
}

// IsMoving returns true if the throw system is moving.
func (s *Settings) IsMoving() bool {
	return s.Top != 0 || s.Bottom != 0
}

// Stop is the default settings for the throw system.
var Stop = Settings{
	Top:    0,
	Bottom: 0,
	Angle:  0,
}

// MarshalBinary encodes the Settings struct into a 12-byte slice (3 float32 values, little-endian).
func (s *Settings) MarshalBinary() ([]byte, error) {
	data := make([]byte, 12)
	topBits := math.Float32bits(float32(s.Top))
	bottomBits := math.Float32bits(float32(s.Bottom))
	angleBits := math.Float32bits(float32(s.Angle))

	data[0] = byte(topBits)
	data[1] = byte(topBits >> 8)
	data[2] = byte(topBits >> 16)
	data[3] = byte(topBits >> 24)

	data[4] = byte(bottomBits)
	data[5] = byte(bottomBits >> 8)
	data[6] = byte(bottomBits >> 16)
	data[7] = byte(bottomBits >> 24)

	data[8] = byte(angleBits)
	data[9] = byte(angleBits >> 8)
	data[10] = byte(angleBits >> 16)
	data[11] = byte(angleBits >> 24)

	return data, nil
}

// UnmarshalBinary decodes a 12-byte slice (3 float32 values, little-endian) into the Settings struct.
func (s *Settings) UnmarshalBinary(data []byte) error {
	if len(data) != 12 {
		return fmt.Errorf("thrower: invalid Settings binary length: expected 12, got %d", len(data))
	}

	topBits := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	bottomBits := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	angleBits := uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16 | uint32(data[11])<<24

	s.Top = float64(math.Float32frombits(topBits))
	s.Bottom = float64(math.Float32frombits(bottomBits))
	s.Angle = float64(math.Float32frombits(angleBits))

	return nil
}

// Info contains the current state of the thrower.
type Info struct {
	Loaded         bool     `json:"loaded"`
	DispenserSpeed float64  `json:"dispenser_speed"`
	ThrowSettings  Settings `json:"throw_settings"`
}

// MarshalBinary encodes the Info struct into a binary format.
// The format is:
//
//	[0]    : Loaded (1 byte, 0 = false, 1 = true)
//	[1-4]  : DispenserSpeed (float32, little-endian)
//	[5-16] : ThrowSettings (12 bytes, see Settings.MarshalBinary)
func (i *Info) MarshalBinary() ([]byte, error) {
	data := make([]byte, 1+4+12) // 1 byte for Loaded, 4 for DispenserSpeed, 12 for ThrowSettings

	// Encode Loaded
	if i.Loaded {
		data[0] = 1
	} else {
		data[0] = 0
	}

	// Encode DispenserSpeed as float32, little-endian
	dsBits := math.Float32bits(float32(i.DispenserSpeed))
	data[1] = byte(dsBits)
	data[2] = byte(dsBits >> 8)
	data[3] = byte(dsBits >> 16)
	data[4] = byte(dsBits >> 24)

	// Encode ThrowSettings
	settingsBytes, err := i.ThrowSettings.MarshalBinary()
	if err != nil {
		return nil, err
	}
	copy(data[5:], settingsBytes)

	return data, nil
}

// UnmarshalBinary decodes a binary slice into the Info struct.
// Expects the same format as MarshalBinary.
func (i *Info) UnmarshalBinary(data []byte) error {
	if len(data) != 17 {
		return fmt.Errorf("thrower: invalid Info binary length: expected 17, got %d", len(data))
	}

	// Decode Loaded
	i.Loaded = data[0] != 0

	// Decode DispenserSpeed
	dsBits := uint32(data[1]) | uint32(data[2])<<8 | uint32(data[3])<<16 | uint32(data[4])<<24
	i.DispenserSpeed = float64(math.Float32frombits(dsBits))

	// Decode ThrowSettings
	return i.ThrowSettings.UnmarshalBinary(data[5:17])
}
