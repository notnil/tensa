package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/paypal/gatt"
)

// BLELoadHandler handles BLE commands to load a ball into the thrower.
type BLELoadHandler struct{}

// WriteHandler returns a handler function that processes a load command for the thrower.
// The handler expects an empty payload (or any payload, ignored) and calls hw.Thrower().Load().
func (BLELoadHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if err := hw.Thrower().Load(context.Background()); err != nil {
			hw.Logger().Error("BLE: failed to command load", "error", err)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

// BLEThrowHandler handles write requests to command the thrower to throw a ball.
// Expects no data.
type BLEThrowHandler struct{}

func (BLEThrowHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if err := hw.Thrower().Throw(context.Background()); err != nil {
			hw.Logger().Error("BLE: failed to command throw", "error", err)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

type BLESetThrowCommand struct {
	Top    float32
	Bottom float32
	Angle  float32
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface for BLESetThrowCommand.
// It parses a binary representation into a BLESetThrowCommand struct.
func (cmd *BLESetThrowCommand) UnmarshalBinary(data []byte) error {
	// Check if data has the expected length (12 bytes - 3 float32 values)
	if len(data) != 12 {
		return fmt.Errorf("invalid data length for BLESetThrowCommand: expected 12 bytes, got %d", len(data))
	}

	// Extract Top (4 bytes, little-endian)
	topBits := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	cmd.Top = math.Float32frombits(topBits)

	// Extract Bottom (4 bytes, little-endian)
	bottomBits := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	cmd.Bottom = math.Float32frombits(bottomBits)

	// Extract Angle (4 bytes, little-endian)
	angleBits := uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16 | uint32(data[11])<<24
	cmd.Angle = math.Float32frombits(angleBits)

	return nil
}

// BLESetThrowHandler handles write requests for setting the thrower parameters.
type BLESetThrowHandler struct{}

// WriteHandler returns a handler function that processes incoming throw configuration commands.
// It unmarshals the binary data into a BLESetThrowCommand and applies the configuration.
func (BLESetThrowHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		// Expects 3 float32 values: top wheel speed, bottom wheel speed, angle
		if len(data) < 12 { // 3 * 4 bytes
			hw.Logger().Error("BLE: insufficient data for SetThrow command", "expected_bytes", 12, "received_bytes", len(data))
			return gatt.StatusUnexpectedError
		}

		topSpeed := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[0:4])))
		bottomSpeed := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4:8])))
		angle := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[8:12])))

		settings := thrower.Settings{
			Top:    topSpeed,
			Bottom: bottomSpeed,
			Angle:  angle,
		}

		if err := hw.Thrower().Set(settings); err != nil {
			hw.Logger().Error("BLE: failed to set throw parameters", "error", err, "settings", settings)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

type BLEThrowerInfoHandler struct{}

func (BLEThrowerInfoHandler) ReadHandler(hw Hardware) func(r gatt.Request) ([]byte, byte) {
	return func(r gatt.Request) ([]byte, byte) {
		info, err := hw.Thrower().Info()
		if err != nil {
			hw.Logger().Error("BLE: failed to get thrower info", "error", err)
			return nil, gatt.StatusUnexpectedError
		}
		data, err := info.MarshalBinary()
		if err != nil {
			hw.Logger().Error("BLE: failed to marshal thrower info", "error", err)
			return nil, gatt.StatusUnexpectedError
		}
		return data, gatt.StatusSuccess
	}
}
