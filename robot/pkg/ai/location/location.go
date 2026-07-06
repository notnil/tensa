package location

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/notnil/tensa/pkg/tennis/court2d"
)

type Loc struct {
	Location court2d.Point
	Rotation float64
}

func (l Loc) IsZero() bool {
	return l.Location == (court2d.Point{}) && l.Rotation == 0
}

// MarshalBinary encodes the Localization into a 16-byte binary format.
// The format is: [X (4 bytes), Y (4 bytes), Rotation (4 bytes)] in little-endian float32.
func (l Loc) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf[0:4], math.Float32bits(float32(l.Location.X)))
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(float32(l.Location.Y)))
	binary.LittleEndian.PutUint32(buf[8:12], math.Float32bits(float32(l.Rotation)))
	return buf, nil
}

// UnmarshalBinary decodes the Localization from a 16-byte binary format.
// The expected format is: [X (4 bytes), Y (4 bytes), Rotation (4 bytes)] in little-endian float32.
func (l *Loc) UnmarshalBinary(data []byte) error {
	if len(data) < 12 {
		return fmt.Errorf("invalid data length for Localization, expected 12 bytes, got %d", len(data))
	}
	l.Location.X = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[0:4])))
	l.Location.Y = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4:8])))
	l.Rotation = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[8:12])))
	return nil
}
