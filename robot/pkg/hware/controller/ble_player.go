package controller

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/paypal/gatt"
)

// BLEPlayerPoseMessage contains player pose data for multiple players.
type BLEPlayerPoseMessage struct {
	Players   []location.Loc
	Timestamp time.Time
}

// MarshalBinary encodes BLEPlayerPoseMessage into binary format.
// Format: [Count (1 byte), Player1 Loc (12 bytes), Player2 Loc (12 bytes), ..., Unix Timestamp (8 bytes)]
func (p BLEPlayerPoseMessage) MarshalBinary() ([]byte, error) {
	count := len(p.Players)
	if count > 255 {
		return nil, fmt.Errorf("too many players: %d (max 255)", count)
	}

	// Calculate buffer size: 1 byte count + (12 bytes per player) + 8 bytes timestamp
	bufSize := 1 + (count * 12) + 8
	buf := make([]byte, bufSize)

	// Write player count
	buf[0] = byte(count)

	// Write each player location
	offset := 1
	for i, player := range p.Players {
		locBytes, err := player.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal player %d: %w", i, err)
		}
		copy(buf[offset:offset+12], locBytes)
		offset += 12
	}

	// Write timestamp (Unix seconds)
	binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(p.Timestamp.Unix()))

	return buf, nil
}

// UnmarshalBinary decodes BLEPlayerPoseMessage from binary format.
// Expected format: [Count (1 byte), Player1 Loc (12 bytes), Player2 Loc (12 bytes), ..., Unix Timestamp (8 bytes)]
func (p *BLEPlayerPoseMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 9 { // Minimum: 1 byte count + 8 bytes timestamp
		return fmt.Errorf("invalid data length for BLEPlayerPoseMessage, expected at least 9 bytes, got %d", len(data))
	}

	// Read player count
	count := int(data[0])

	// Validate expected length
	expectedLen := 1 + (count * 12) + 8
	if len(data) < expectedLen {
		return fmt.Errorf("invalid data length: expected %d bytes for %d players, got %d", expectedLen, count, len(data))
	}

	// Read each player location
	p.Players = make([]location.Loc, count)
	offset := 1
	for i := 0; i < count; i++ {
		var loc location.Loc
		if err := loc.UnmarshalBinary(data[offset : offset+12]); err != nil {
			return fmt.Errorf("failed to unmarshal player %d: %w", i, err)
		}
		p.Players[i] = loc
		offset += 12
	}

	// Read timestamp
	unixTime := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	p.Timestamp = time.Unix(unixTime, 0)

	return nil
}

// BLEPlayerPoseNotifier implements a BLE notifier that sends player pose data
// at regular intervals (250ms).
type BLEPlayerPoseNotifier struct{}

func (n BLEPlayerPoseNotifier) NotifyHandler(hw Hardware) func(r gatt.Request, notifier gatt.Notifier) {
	return func(r gatt.Request, notifier gatt.Notifier) {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		for !notifier.Done() {
			<-ticker.C

			// Get current player poses
			players, err := hw.PlayerProvider().Players()
			if err != nil {
				// Don't log every error as it's expected when no players are detected
				// Just send an empty player list
				players = []location.Loc{}
			}

			// Marshal and send the data
			data, err := BLEPlayerPoseMessage{
				Players:   players,
				Timestamp: time.Now(),
			}.MarshalBinary()
			if err != nil {
				hw.Logger().Error("BLE: failed to marshal player pose", "error", err)
				continue
			}

			_, err = notifier.Write(data)
			if err != nil {
				hw.Logger().Error("BLE: failed to write player pose", "error", err)
				continue
			}
			hw.Logger().Debug("BLE: sent player detection", "player_count", len(players))
		}
	}
}
