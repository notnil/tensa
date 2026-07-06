package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/paypal/gatt"
)

// BLE Location Notification System
//
// This implements a configurable BLE location notification system using two characteristics:
// 1. LocationConfigUUID (write) - Configure what type of location data to send
// 2. LocationUUID (notify) - Receive location notifications
//
// Usage pattern:
// 1. Client writes LocatorType (1 byte) to LocationConfigUUID to configure the type
// 2. Client subscribes to LocationUUID to start receiving notifications
// 3. Server sends BLELocationMessage data on LocationUUID based on configured type

type BLELocationMessage struct {
	Location  location.Loc
	Timestamp time.Time
}

// MarshalBinary encodes BLELocationMessage into binary format.
// Format: [Localization (12 bytes), Unix Timestamp (8 bytes)]
func (c BLELocationMessage) MarshalBinary() ([]byte, error) {
	locBytes, err := c.Location.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal location: %w", err)
	}

	// Marshal timestamp as Unix seconds (int64)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(c.Timestamp.Unix()))

	// Concatenate location and timestamp bytes
	buf := make([]byte, 0, len(locBytes)+len(timeBytes))
	buf = append(buf, locBytes...)
	buf = append(buf, timeBytes...)

	return buf, nil
}

// UnmarshalBinary decodes BLELocationMessage from binary format.
// Expected format: [Localization (12 bytes), Unix Timestamp (8 bytes)]
func (c *BLELocationMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 20 { // 12 bytes for Localization + 8 bytes for timestamp
		return fmt.Errorf("invalid data length for BLELocationMessage, expected at least 20 bytes, got %d", len(data))
	}

	// Unmarshal Localization
	err := c.Location.UnmarshalBinary(data[0:12])
	if err != nil {
		return fmt.Errorf("failed to unmarshal location: %w", err)
	}

	// Unmarshal timestamp (Unix seconds)
	unixTime := int64(binary.LittleEndian.Uint64(data[12:20]))
	c.Timestamp = time.Unix(unixTime, 0)

	return nil
}

type BLELocationNotifier struct{}

func (n BLELocationNotifier) NotifyHandler(hw Hardware) func(r gatt.Request, notifier gatt.Notifier) {
	return func(r gatt.Request, notifier gatt.Notifier) {
		locatorSub := hw.Locator()
		locCh := make(chan metrics.Metric[location.Loc])
		errCh := make(chan error)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			err := locatorSub.Subscribe(ctx, locCh)
			errCh <- err
		}()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var latestLoc *metrics.Metric[location.Loc]

		for !notifier.Done() {
			select {
			case <-ctx.Done():
				return
			case <-errCh:
				return
			case loc := <-locCh:
				// Store the latest location
				latestLoc = &loc
			case <-ticker.C:
				// Send the latest location at most every 100ms
				if latestLoc != nil {
					data, err := BLELocationMessage{
						Location:  latestLoc.Value,
						Timestamp: latestLoc.Timestamp,
					}.MarshalBinary()
					if err != nil {
						hw.Logger().Error("BLE: failed to marshal location", "error", err)
						continue
					}
					_, err = notifier.Write(data)
					if err != nil {
						hw.Logger().Error("BLE: failed to write location", "error", err)
						continue
					}
					hw.Logger().Debug("BLE: sent location", "x", latestLoc.Value.Location.X, "y", latestLoc.Value.Location.Y, "theta", latestLoc.Value.Rotation)
					latestLoc = nil
				}
			}
		}
	}
}
