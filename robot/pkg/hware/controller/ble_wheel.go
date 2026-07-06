package controller

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/paypal/gatt"
)

// BlEMoveHandler handles write requests for moving the robot.
// Expects 2 float32 values: direction and speed.
type BlEMoveHandler struct{}

func (BlEMoveHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		// Expects 2 float32 values: direction and speed
		if len(data) < 8 { // 2 * 4 bytes
			hw.Logger().Error("BLE: insufficient data for Move command", "expected_bytes", 8, "received_bytes", len(data))
			return gatt.StatusUnexpectedError
		}

		direction := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[0:4])))
		speed := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4:8])))

		if err := hw.Wheels().Move(direction, speed); err != nil {
			hw.Logger().Error("BLE: failed to command Move", "error", err, "direction", direction, "speed", speed)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

type BLEMoveCommand struct {
	Direction float32
	Speed     float32
}

func (cmd *BLEMoveCommand) UnmarshalBinary(data []byte) error {
	// Check if data has the expected length
	if len(data) < 8 {
		return fmt.Errorf("insufficient data for BLEMoveCommand: expected 8 bytes, got %d", len(data))
	}

	// Extract Direction (first 4 bytes, little-endian)
	dirBits := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	cmd.Direction = math.Float32frombits(dirBits)

	// Extract Speed (next 4 bytes, little-endian)
	speedBits := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	cmd.Speed = math.Float32frombits(speedBits)

	return nil
}

// BLERotateHandler handles write requests for rotating the robot.
// Expects 1 float32 value: speed.
type BLERotateHandler struct{}

func (BLERotateHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		// Expects 1 float32 value: speed
		if len(data) < 4 { // 1 * 4 bytes
			hw.Logger().Error("BLE: insufficient data for Rotate command", "expected_bytes", 4, "received_bytes", len(data))
			return gatt.StatusUnexpectedError
		}

		speed := float64(math.Float32frombits(binary.LittleEndian.Uint32(data[0:4])))

		if err := hw.Wheels().Rotate(speed); err != nil {
			hw.Logger().Error("BLE: failed to command Rotate", "error", err, "speed", speed)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

type BLERotateCommand struct {
	Speed float32
}

func (cmd *BLERotateCommand) UnmarshalBinary(data []byte) error {
	// Check if data has the expected length
	if len(data) < 4 {
		return fmt.Errorf("insufficient data for BLERotateCommand: expected 4 bytes, got %d", len(data))
	}

	// Extract Speed (4 bytes, little-endian)
	speedBits := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	cmd.Speed = math.Float32frombits(speedBits)

	return nil
}

type BLEWheelStatusHandler struct{}

func (BLEWheelStatusHandler) ReadHandler(hw Hardware) func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
	return func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		state, err := hw.Wheels().Status()
		if err != nil {
			hw.Logger().Error("BLE: failed to get wheel status", "error", err)
			rsp.SetStatus(gatt.StatusUnexpectedError)
			return
		}
		data, err := state.BinaryMarshal()
		if err != nil {
			hw.Logger().Error("BLE: failed to marshal wheel state", "error", err)
			rsp.SetStatus(gatt.StatusUnexpectedError)
			return
		}
		_, err = rsp.Write(data)
		if err != nil {
			hw.Logger().Error("BLE: failed to write wheel state", "error", err)
			rsp.SetStatus(gatt.StatusUnexpectedError)
			return
		}
		rsp.SetStatus(gatt.StatusSuccess)
	}
}

// BLEWheelEnableHandler handles write requests for enabling/disabling the wheels.
// Expects 1 byte: 1 to enable, 0 to disable.
type BLEWheelEnableHandler struct{}

func (BLEWheelEnableHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if len(data) < 1 {
			hw.Logger().Error("BLE: no data received for WheelEnable command")
			return gatt.StatusUnexpectedError
		}
		enable := data[0] == 1
		var err error
		if enable {
			err = hw.Wheels().Enable()
		} else {
			err = hw.Wheels().Disable()
		}
		if err != nil {
			hw.Logger().Error("BLE: failed to set wheel enable state", "error", err, "enable_command", enable)
			return gatt.StatusUnexpectedError
		}
		hw.Logger().Info("BLE: wheel enable state changed", "enabled", enable)
		return gatt.StatusSuccess
	}
}
