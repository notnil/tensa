package controller

import (
	"context"

	"github.com/paypal/gatt"
)

// BLERecordStartHandler handles write requests to start ZED recording.
type BLERecordStartHandler struct{}

func (BLERecordStartHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		hw.Logger().Info("BLE: Start recording command received")
		// Using Background context here as the recording needs to persist
		// beyond this individual request lifecycle.
		if err := hw.StartRecording(context.Background()); err != nil {
			hw.Logger().Error("BLE: failed to start ZED recording", "error", err)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}

// BLERecordStopHandler handles write requests to stop ZED recording.
type BLERecordStopHandler struct{}

func (BLERecordStopHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		hw.Logger().Info("BLE: Stop recording command received")
		if err := hw.StopRecording(); err != nil {
			hw.Logger().Error("BLE: failed to stop ZED recording", "error", err)
			return gatt.StatusUnexpectedError
		}
		return gatt.StatusSuccess
	}
}
