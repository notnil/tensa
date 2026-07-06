package wheels

import (
	"encoding/binary"
	"fmt"
)

// Status represents the collective status of all four wheels.
// Each field (FrontLeft, FrontRight, RearRight, RearLeft) holds the WheelStatus for the respective wheel.
type Status Set[WheelStatus]

// BinaryMarshal marshals the Status struct into a binary format.
func (s *Status) BinaryMarshal() ([]byte, error) {
	var result []byte
	// Marshal each WheelStatus
	for _, ws := range []WheelStatus{s.FrontLeft, s.FrontRight, s.RearRight, s.RearLeft} {
		data, err := ws.BinaryMarshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal wheel status: %w", err)
		}
		result = append(result, data...)
	}
	return result, nil
}

// BinaryUnmarshal unmarshals the binary data into the Status struct.
func (s *Status) BinaryUnmarshal(data []byte) error {
	if len(data) < 44 { // Minimum 11 bytes per wheel * 4 wheels
		return fmt.Errorf("insufficient data for unmarshaling Status: need at least 44 bytes, got %d", len(data))
	}
	// Unmarshal each WheelStatus
	wheels := []struct {
		ptr *WheelStatus
	}{
		{&s.FrontLeft},
		{&s.FrontRight},
		{&s.RearRight},
		{&s.RearLeft},
	}
	offset := 0
	for i, wheel := range wheels {
		if offset+11 > len(data) {
			return fmt.Errorf("insufficient data for unmarshaling wheel status at index %d: need at least 11 bytes, got %d", i, len(data)-offset)
		}
		// Read error length to determine total size for this wheel status
		errLen := int(binary.LittleEndian.Uint16(data[offset+9 : offset+11]))
		totalLen := 11 + errLen
		if offset+totalLen > len(data) {
			return fmt.Errorf("insufficient data for unmarshaling wheel status error string at index %d: need %d bytes, got %d", i, totalLen, len(data)-offset)
		}
		if err := wheel.ptr.BinaryUnmarshal(data[offset : offset+totalLen]); err != nil {
			return fmt.Errorf("failed to unmarshal wheel status at index %d: %w", i, err)
		}
		offset += totalLen
	}
	return nil
}

// String returns a string representation of the Status struct.
func (s *Status) String() string {
	return fmt.Sprintf("Status{FL{en=%t sp=%.1f cur=%.1f err=%q} FR{en=%t sp=%.1f cur=%.1f err=%q} RL{en=%t sp=%.1f cur=%.1f err=%q} RR{en=%t sp=%.1f cur=%.1f err=%q}}",
		s.FrontLeft.Enabled, s.FrontLeft.Speed, s.FrontLeft.Current, s.FrontLeft.Error,
		s.FrontRight.Enabled, s.FrontRight.Speed, s.FrontRight.Current, s.FrontRight.Error,
		s.RearLeft.Enabled, s.RearLeft.Speed, s.RearLeft.Current, s.RearLeft.Error,
		s.RearRight.Enabled, s.RearRight.Speed, s.RearRight.Current, s.RearRight.Error,
	)
}
