package controller

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"testing"
)

func TestBLEDrillUploadInitMessage_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		wantDrillID string
		wantCompSz  uint32
		wantUncomSz uint32
	}{
		{
			name: "valid init message",
			data: func() []byte {
				drillID := "test-drill-id"
				buf := make([]byte, 2+len(drillID)+4+4+32)
				buf[0] = 0x01 // Type
				buf[1] = byte(len(drillID))
				copy(buf[2:], drillID)
				// CompressedSize = 1000
				buf[2+len(drillID)] = 0xE8
				buf[2+len(drillID)+1] = 0x03
				buf[2+len(drillID)+2] = 0x00
				buf[2+len(drillID)+3] = 0x00
				// UncompressedSize = 2000
				buf[2+len(drillID)+4] = 0xD0
				buf[2+len(drillID)+5] = 0x07
				buf[2+len(drillID)+6] = 0x00
				buf[2+len(drillID)+7] = 0x00
				// SHA256 (32 bytes of 0xAA)
				for i := 0; i < 32; i++ {
					buf[2+len(drillID)+8+i] = 0xAA
				}
				return buf
			}(),
			wantErr:     false,
			wantDrillID: "test-drill-id",
			wantCompSz:  1000,
			wantUncomSz: 2000,
		},
		{
			name:    "too short",
			data:    []byte{0x01},
			wantErr: true,
		},
		{
			name:    "wrong type",
			data:    make([]byte, 50),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg BLEDrillUploadInitMessage
			err := msg.UnmarshalBinary(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if msg.DrillID != tt.wantDrillID {
					t.Errorf("DrillID = %v, want %v", msg.DrillID, tt.wantDrillID)
				}
				if msg.CompressedSize != tt.wantCompSz {
					t.Errorf("CompressedSize = %v, want %v", msg.CompressedSize, tt.wantCompSz)
				}
				if msg.UncompressedSize != tt.wantUncomSz {
					t.Errorf("UncompressedSize = %v, want %v", msg.UncompressedSize, tt.wantUncomSz)
				}
			}
		})
	}
}

func TestBLEDrillUploadChunkMessage_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		wantSeq     uint16
		wantDataLen int
	}{
		{
			name:        "valid chunk",
			data:        []byte{0x02, 0x05, 0x00, 0xAA, 0xBB, 0xCC},
			wantErr:     false,
			wantSeq:     5,
			wantDataLen: 3,
		},
		{
			name:    "too short",
			data:    []byte{0x02, 0x05},
			wantErr: true,
		},
		{
			name:    "wrong type",
			data:    []byte{0x01, 0x05, 0x00, 0xAA},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg BLEDrillUploadChunkMessage
			err := msg.UnmarshalBinary(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if msg.Sequence != tt.wantSeq {
					t.Errorf("Sequence = %v, want %v", msg.Sequence, tt.wantSeq)
				}
				if len(msg.Data) != tt.wantDataLen {
					t.Errorf("Data length = %v, want %v", len(msg.Data), tt.wantDataLen)
				}
			}
		})
	}
}

func TestGzipDecompression(t *testing.T) {
	// Test that we can compress and decompress data correctly
	original := []byte("This is test data that will be compressed and decompressed")

	// Compress
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(original); err != nil {
		t.Fatalf("Failed to write to gzip: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	compressed := buf.Bytes()
	t.Logf("Original: %d bytes, Compressed: %d bytes", len(original), len(compressed))

	// Decompress
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(gzipReader); err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if !bytes.Equal(original, decompressed.Bytes()) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestSHA256Checksum(t *testing.T) {
	// Test that we can calculate SHA256 correctly
	data := []byte("test data for checksum")
	hash := sha256.Sum256(data)

	// Verify hash is 32 bytes
	if len(hash) != 32 {
		t.Errorf("SHA256 hash length = %d, want 32", len(hash))
	}

	// Verify hash is deterministic
	hash2 := sha256.Sum256(data)
	if hash != hash2 {
		t.Errorf("SHA256 hash not deterministic")
	}

	// Verify different data produces different hash
	differentData := []byte("different data")
	differentHash := sha256.Sum256(differentData)
	if hash == differentHash {
		t.Errorf("Different data produced same hash")
	}
}

func TestBLEDrillStatusMessage_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		statusCode DrillStatusCode
		message    string
	}{
		{
			name:       "success message",
			statusCode: DrillStatusSuccess,
			message:    "Drill completed successfully",
		},
		{
			name:       "error not found",
			statusCode: DrillStatusErrorNotFound,
			message:    "Drill abc-123 not found",
		},
		{
			name:       "empty message",
			statusCode: DrillStatusSuccess,
			message:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := BLEDrillStatusMessage{
				StatusCode: tt.statusCode,
				Message:    tt.message,
			}

			// Marshal
			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary() error = %v", err)
			}

			// Unmarshal
			var decoded BLEDrillStatusMessage
			if err := decoded.UnmarshalBinary(data); err != nil {
				t.Fatalf("UnmarshalBinary() error = %v", err)
			}

			// Compare
			if decoded.StatusCode != original.StatusCode {
				t.Errorf("StatusCode = %v, want %v", decoded.StatusCode, original.StatusCode)
			}
			if decoded.Message != original.Message {
				t.Errorf("Message = %v, want %v", decoded.Message, original.Message)
			}
		})
	}
}
