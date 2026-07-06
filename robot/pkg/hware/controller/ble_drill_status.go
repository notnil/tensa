package controller

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/paypal/gatt"
)

// DrillStatusCode represents the status of a drill operation.
type DrillStatusCode uint8

const (
	DrillStatusSuccess       DrillStatusCode = 0
	DrillStatusErrorNotFound DrillStatusCode = 1
	DrillStatusErrorLoad     DrillStatusCode = 2
	DrillStatusErrorRunning  DrillStatusCode = 3
	DrillStatusUploadSuccess DrillStatusCode = 4
	DrillStatusUploadError   DrillStatusCode = 5
)

// BLEDrillStatusMessage represents a drill status notification.
// Binary format: [StatusCode: uint8][MessageLength: uint16][Message: UTF-8 string]
type BLEDrillStatusMessage struct {
	StatusCode DrillStatusCode
	Message    string
}

// MarshalBinary encodes BLEDrillStatusMessage into binary format.
func (m BLEDrillStatusMessage) MarshalBinary() ([]byte, error) {
	messageBytes := []byte(m.Message)
	messageLen := len(messageBytes)
	if messageLen > 0xFFFF {
		return nil, fmt.Errorf("message too long: %d bytes (max 65535)", messageLen)
	}

	// Allocate buffer: 1 byte status + 2 bytes length + message
	buf := make([]byte, 3+messageLen)
	buf[0] = uint8(m.StatusCode)
	binary.LittleEndian.PutUint16(buf[1:3], uint16(messageLen))
	copy(buf[3:], messageBytes)

	return buf, nil
}

// UnmarshalBinary decodes BLEDrillStatusMessage from binary format.
func (m *BLEDrillStatusMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid data length: expected at least 3 bytes, got %d", len(data))
	}

	m.StatusCode = DrillStatusCode(data[0])
	messageLen := binary.LittleEndian.Uint16(data[1:3])

	if len(data) < 3+int(messageLen) {
		return fmt.Errorf("invalid data length: expected %d bytes, got %d", 3+messageLen, len(data))
	}

	m.Message = string(data[3 : 3+messageLen])
	return nil
}

// BLEDrillStatusNotifier manages drill status notifications to BLE clients.
type BLEDrillStatusNotifier struct {
	mu        sync.RWMutex
	notifiers map[string]gatt.Notifier // keyed by central ID
	log       *slog.Logger
}

// NewBLEDrillStatusNotifier creates a new drill status notifier.
func NewBLEDrillStatusNotifier(log *slog.Logger) *BLEDrillStatusNotifier {
	return &BLEDrillStatusNotifier{
		notifiers: make(map[string]gatt.Notifier),
		log:       log,
	}
}

// NotifyHandler returns a handler for drill status notification subscriptions.
func (n *BLEDrillStatusNotifier) NotifyHandler(hw Hardware) func(r gatt.Request, notifier gatt.Notifier) {
	return func(r gatt.Request, notifier gatt.Notifier) {
		centralID := r.Central.ID()
		n.log.Info("BLE: drill status notifications started", "central", centralID)

		// Register this notifier
		n.mu.Lock()
		n.notifiers[centralID] = notifier
		n.mu.Unlock()

		// Wait until client unsubscribes (Done() returns false when active)
		// Use ticker to periodically check if the notifier is still active
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for !notifier.Done() {
			<-ticker.C
		}

		// Unregister this notifier
		n.mu.Lock()
		delete(n.notifiers, centralID)
		n.mu.Unlock()

		n.log.Info("BLE: drill status notifications stopped", "central", centralID)
	}
}

// SendStatus sends a drill status notification to all subscribed clients.
func (n *BLEDrillStatusNotifier) SendStatus(code DrillStatusCode, message string) {
	msg := BLEDrillStatusMessage{
		StatusCode: code,
		Message:    message,
	}

	data, err := msg.MarshalBinary()
	if err != nil {
		n.log.Error("BLE: failed to marshal drill status", "error", err)
		return
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.notifiers) == 0 {
		n.log.Debug("BLE: no clients subscribed to drill status")
		return
	}

	for centralID, notifier := range n.notifiers {
		if _, err := notifier.Write(data); err != nil {
			n.log.Error("BLE: failed to send drill status notification",
				"central", centralID,
				"error", err)
		} else {
			n.log.Debug("BLE: sent drill status notification",
				"central", centralID,
				"code", code,
				"message", message)
		}
	}
}

// SendError is a convenience method to send an error status.
func (n *BLEDrillStatusNotifier) SendError(code DrillStatusCode, message string) {
	n.SendStatus(code, message)
}

// SendSuccess is a convenience method to send a success status.
func (n *BLEDrillStatusNotifier) SendSuccess(message string) {
	n.SendStatus(DrillStatusSuccess, message)
}
