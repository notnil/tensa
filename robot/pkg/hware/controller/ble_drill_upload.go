package controller

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/paypal/gatt"
)

// Message type constants for DrillUpload protocol
const (
	DrillUploadMsgInit     = 0x01
	DrillUploadMsgChunk    = 0x02
	DrillUploadMsgFinalize = 0x03
)

// BLEDrillUploadInitMessage represents the initial upload message.
// Binary format: [Type=0x01][IDLen: uint8][DrillID: string][CompressedSize: uint32][UncompressedSize: uint32][SHA256: 32 bytes]
type BLEDrillUploadInitMessage struct {
	DrillID          string
	CompressedSize   uint32
	UncompressedSize uint32
	SHA256Hash       [32]byte
}

// UnmarshalBinary decodes BLEDrillUploadInitMessage from binary format.
func (m *BLEDrillUploadInitMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("invalid init message: too short")
	}

	if data[0] != DrillUploadMsgInit {
		return fmt.Errorf("invalid message type: expected 0x01, got 0x%02x", data[0])
	}

	idLen := data[1]
	if len(data) < 2+int(idLen)+4+4+32 {
		return fmt.Errorf("invalid init message: expected %d bytes, got %d", 2+int(idLen)+4+4+32, len(data))
	}

	m.DrillID = string(data[2 : 2+idLen])
	offset := 2 + int(idLen)
	m.CompressedSize = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	m.UncompressedSize = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	copy(m.SHA256Hash[:], data[offset:offset+32])

	return nil
}

// BLEDrillUploadChunkMessage represents a chunk of data.
// Binary format: [Type=0x02][Sequence: uint16][Data: bytes]
type BLEDrillUploadChunkMessage struct {
	Sequence uint16
	Data     []byte
}

// UnmarshalBinary decodes BLEDrillUploadChunkMessage from binary format.
func (m *BLEDrillUploadChunkMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return fmt.Errorf("invalid chunk message: too short")
	}

	if data[0] != DrillUploadMsgChunk {
		return fmt.Errorf("invalid message type: expected 0x02, got 0x%02x", data[0])
	}

	m.Sequence = binary.LittleEndian.Uint16(data[1:3])
	m.Data = make([]byte, len(data)-3)
	copy(m.Data, data[3:])

	return nil
}

// uploadState tracks the state of an ongoing drill upload.
type uploadState struct {
	drillID          string
	compressedSize   uint32
	uncompressedSize uint32
	expectedHash     [32]byte
	buffer           *bytes.Buffer
	nextSequence     uint16
	initialized      bool
}

// BLEDrillUploadHandler handles drill file uploads over BLE.
type BLEDrillUploadHandler struct {
	mu              sync.Mutex
	state           *uploadState
	drillsDirectory string
	statusNotifier  *BLEDrillStatusNotifier
	log             *slog.Logger
}

// NewBLEDrillUploadHandler creates a new drill upload handler.
func NewBLEDrillUploadHandler(drillsDirectory string, statusNotifier *BLEDrillStatusNotifier, log *slog.Logger) *BLEDrillUploadHandler {
	return &BLEDrillUploadHandler{
		drillsDirectory: drillsDirectory,
		statusNotifier:  statusNotifier,
		log:             log,
	}
}

// WriteHandler returns a handler for drill upload write operations.
func (h *BLEDrillUploadHandler) WriteHandler(hw Hardware) func(r gatt.Request, data []byte) (status byte) {
	return func(r gatt.Request, data []byte) (status byte) {
		if len(data) == 0 {
			hw.Logger().Error("BLE: empty drill upload message")
			return gatt.StatusUnexpectedError
		}

		msgType := data[0]

		switch msgType {
		case DrillUploadMsgInit:
			return h.handleInit(hw, data)
		case DrillUploadMsgChunk:
			return h.handleChunk(hw, data)
		case DrillUploadMsgFinalize:
			return h.handleFinalize(hw, data)
		default:
			hw.Logger().Error("BLE: invalid drill upload message type", "type", msgType)
			return gatt.StatusUnexpectedError
		}
	}
}

// handleInit processes an initialization message.
func (h *BLEDrillUploadHandler) handleInit(hw Hardware, data []byte) byte {
	var msg BLEDrillUploadInitMessage
	if err := msg.UnmarshalBinary(data); err != nil {
		hw.Logger().Error("BLE: failed to parse drill upload init", "error", err)
		return gatt.StatusUnexpectedError
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Reset any previous upload state
	if h.state != nil && h.state.initialized {
		hw.Logger().Info("BLE: aborting previous drill upload", "previousDrill", h.state.drillID)
	}

	// Initialize new upload state
	h.state = &uploadState{
		drillID:          msg.DrillID,
		compressedSize:   msg.CompressedSize,
		uncompressedSize: msg.UncompressedSize,
		expectedHash:     msg.SHA256Hash,
		buffer:           bytes.NewBuffer(make([]byte, 0, msg.CompressedSize)),
		nextSequence:     0,
		initialized:      true,
	}

	hw.Logger().Info("BLE: drill upload initialized",
		"drillID", msg.DrillID,
		"compressedSize", msg.CompressedSize,
		"uncompressedSize", msg.UncompressedSize,
		"hash", hex.EncodeToString(msg.SHA256Hash[:8]))

	return gatt.StatusSuccess
}

// handleChunk processes a data chunk message.
func (h *BLEDrillUploadHandler) handleChunk(hw Hardware, data []byte) byte {
	var msg BLEDrillUploadChunkMessage
	if err := msg.UnmarshalBinary(data); err != nil {
		hw.Logger().Error("BLE: failed to parse drill upload chunk", "error", err)
		return gatt.StatusUnexpectedError
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.state == nil || !h.state.initialized {
		hw.Logger().Error("BLE: chunk received before init")
		return gatt.StatusUnexpectedError
	}

	// Check sequence number
	if msg.Sequence != h.state.nextSequence {
		hw.Logger().Warn("BLE: unexpected chunk sequence",
			"expected", h.state.nextSequence,
			"got", msg.Sequence)
		// Still process it - we don't enforce strict ordering, just track it
	}

	// Append data to buffer
	if _, err := h.state.buffer.Write(msg.Data); err != nil {
		hw.Logger().Error("BLE: failed to write chunk to buffer", "error", err)
		h.state = nil // Reset state on error
		return gatt.StatusUnexpectedError
	}

	h.state.nextSequence = msg.Sequence + 1

	if hw.Logger().Enabled(nil, slog.LevelDebug) {
		hw.Logger().Debug("BLE: drill upload chunk received",
			"drillID", h.state.drillID,
			"sequence", msg.Sequence,
			"chunkSize", len(msg.Data),
			"totalReceived", h.state.buffer.Len(),
			"expected", h.state.compressedSize)
	}

	return gatt.StatusSuccess
}

// handleFinalize processes a finalize message and completes the upload.
func (h *BLEDrillUploadHandler) handleFinalize(hw Hardware, data []byte) byte {
	if len(data) < 1 || data[0] != DrillUploadMsgFinalize {
		hw.Logger().Error("BLE: invalid finalize message")
		return gatt.StatusUnexpectedError
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.state == nil || !h.state.initialized {
		hw.Logger().Error("BLE: finalize received before init")
		return gatt.StatusUnexpectedError
	}

	drillID := h.state.drillID
	compressedData := h.state.buffer.Bytes()

	hw.Logger().Info("BLE: drill upload finalize received",
		"drillID", drillID,
		"receivedSize", len(compressedData),
		"expectedSize", h.state.compressedSize)

	// Verify size
	if uint32(len(compressedData)) != h.state.compressedSize {
		errMsg := fmt.Sprintf("size mismatch: expected %d bytes, got %d",
			h.state.compressedSize, len(compressedData))
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}

	// Verify checksum
	actualHash := sha256.Sum256(compressedData)
	if actualHash != h.state.expectedHash {
		errMsg := fmt.Sprintf("checksum mismatch: expected %s, got %s",
			hex.EncodeToString(h.state.expectedHash[:8]),
			hex.EncodeToString(actualHash[:8]))
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}

	// Decompress
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		errMsg := fmt.Sprintf("failed to create gzip reader: %v", err)
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}
	defer gzipReader.Close()

	uncompressedData, err := io.ReadAll(gzipReader)
	if err != nil {
		errMsg := fmt.Sprintf("failed to decompress: %v", err)
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}

	if uint32(len(uncompressedData)) != h.state.uncompressedSize {
		errMsg := fmt.Sprintf("uncompressed size mismatch: expected %d bytes, got %d",
			h.state.uncompressedSize, len(uncompressedData))
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}

	// Write to file
	if err := h.writeDrillFile(drillID, uncompressedData); err != nil {
		errMsg := fmt.Sprintf("failed to write file: %v", err)
		hw.Logger().Error("BLE: drill upload failed", "error", errMsg)
		h.state = nil
		h.statusNotifier.SendError(DrillStatusUploadError, errMsg)
		return gatt.StatusUnexpectedError
	}

	hw.Logger().Info("BLE: drill upload completed successfully",
		"drillID", drillID,
		"fileSize", len(uncompressedData))

	h.state = nil
	h.statusNotifier.SendSuccess(fmt.Sprintf("Drill %s uploaded successfully", drillID))
	return gatt.StatusSuccess
}

// writeDrillFile writes the drill .so file to disk.
func (h *BLEDrillUploadHandler) writeDrillFile(drillID string, data []byte) error {
	// Ensure drills directory exists
	if err := os.MkdirAll(h.drillsDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create drills directory: %w", err)
	}

	// Write to temporary file first
	tempPath := filepath.Join(h.drillsDirectory, fmt.Sprintf(".%s.so.tmp", drillID))
	finalPath := filepath.Join(h.drillsDirectory, fmt.Sprintf("%s.so", drillID))

	if err := os.WriteFile(tempPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	h.log.Info("drill file written", "path", finalPath, "size", len(data))
	return nil
}
