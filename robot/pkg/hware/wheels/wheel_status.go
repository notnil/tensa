package wheels

import (
	"encoding/binary"
	"fmt"
	"math"
)

// WheelStatus represents the detailed status of a single wheel.
// It includes information about its operational state, speed, current draw, and any errors.
type WheelStatus struct {
	Enabled bool    `json:"enabled"` // Whether the wheels are enabled
	Speed   float64 `json:"speed"`   // Speed of the wheels
	Current float64 `json:"current"` // Current of the wheels
	Error   string  `json:"error"`   // Error of the wheels
}

// DefaultWheelStatus provides a default, zeroed-out WheelStatus.
// This typically represents a wheel that is disabled, not moving, and has no errors.
var DefaultWheelStatus = WheelStatus{
	Enabled: false,
	Speed:   0,
	Current: 0,
	Error:   "",
}

// BinaryMarshal marshals the WheelStatus struct into a binary format using float32 for encoding to reduce space.
func (ws *WheelStatus) BinaryMarshal() ([]byte, error) {
	// Convert error to string
	errStr := ws.Error
	errBytes := []byte(errStr)
	errLen := len(errBytes)

	// Buffer size: 1 byte for Enabled, 4 bytes each for Speed and Current, 2 bytes for error length, and error string bytes
	buf := make([]byte, 11+errLen)
	if ws.Enabled {
		buf[0] = 1
	} else {
		buf[0] = 0
	}
	// Encode Speed as float32
	binary.LittleEndian.PutUint32(buf[1:5], math.Float32bits(float32(ws.Speed)))
	// Encode Current as float32
	binary.LittleEndian.PutUint32(buf[5:9], math.Float32bits(float32(ws.Current)))
	// Encode Error string length as uint16
	binary.LittleEndian.PutUint16(buf[9:11], uint16(errLen))
	// Copy Error string bytes
	copy(buf[11:], errBytes)
	return buf, nil
}

// BinaryUnmarshal unmarshals the binary data into the WheelStatus struct.
func (ws *WheelStatus) BinaryUnmarshal(data []byte) error {
	if len(data) < 11 {
		return fmt.Errorf("insufficient data for unmarshaling WheelStatus: need at least 11 bytes, got %d", len(data))
	}
	ws.Enabled = data[0] == 1
	// Decode Speed from float32
	ws.Speed = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[1:5])))
	// Decode Current from float32
	ws.Current = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[5:9])))
	// Decode Error string length
	errLen := int(binary.LittleEndian.Uint16(data[9:11]))
	if len(data) < 11+errLen {
		return fmt.Errorf("insufficient data for unmarshaling WheelStatus error string: need %d bytes, got %d", 11+errLen, len(data))
	}
	// Decode Error string
	if errLen > 0 {
		ws.Error = string(data[11 : 11+errLen])
	} else {
		ws.Error = ""
	}
	return nil
}
