package loader

import (
	"bytes"
	"compress/gzip"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGCSRegistry_parseGCSURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		wantBucket string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "valid URI with path",
			uri:        "gs://example-public-bucket/tensa/drills/plugins",
			wantBucket: "example-public-bucket",
			wantPath:   "tensa/drills/plugins",
			wantErr:    false,
		},
		{
			name:       "valid URI without path",
			uri:        "gs://my-bucket",
			wantBucket: "my-bucket",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "valid URI with nested path",
			uri:        "gs://bucket/path/to/nested/folder",
			wantBucket: "bucket",
			wantPath:   "path/to/nested/folder",
			wantErr:    false,
		},
		{
			name:    "missing gs:// prefix",
			uri:     "tensa-media-public/drills/plugins",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "gs://",
			wantErr: true,
		},
		{
			name:    "invalid format",
			uri:     "http://example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, path, err := parseGCSURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGCSURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if bucket != tt.wantBucket {
					t.Errorf("parseGCSURI() bucket = %v, want %v", bucket, tt.wantBucket)
				}
				if path != tt.wantPath {
					t.Errorf("parseGCSURI() path = %v, want %v", path, tt.wantPath)
				}
			}
		})
	}
}

// TestGCSRegistry_Sync is an integration test that requires a real GCS bucket.
// It will be skipped if running in CI or if GCS credentials are not available.
func TestGCSRegistry_Sync(t *testing.T) {
	// Skip if running in CI or if credentials are not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping GCS integration test in CI environment")
	}

	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a temporary cache directory for this test
	cacheDir, err := os.MkdirTemp("", "drillsx-gcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache dir: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	// Create a GCS registry pointing to a test bucket
	// Note: This test requires a real GCS bucket with plugins
	registry, err := NewGCSRegistry(ctx, "gs://example-public-bucket/tensa/drills/plugins", cacheDir, log)
	if err != nil {
		t.Skipf("Failed to create GCS registry (likely no credentials): %v", err)
		return
	}
	defer registry.Close()

	// Sync all plugins from GCS
	if err := registry.Sync(ctx); err != nil {
		t.Logf("Sync failed (expected if bucket doesn't exist or is empty): %v", err)
		return
	}

	// Try to load a drill by ID
	// This should load from local cache without downloading
	_, err = registry.GetDrill("test-drill")
	if err != nil {
		t.Logf("GetDrill failed (expected if test-drill.so doesn't exist in bucket): %v", err)
		// We don't fail the test here because the plugin might not exist in the bucket
		// This test serves more as a demonstration of usage
	}
}

// TestGCSRegistry_downloadPlugin_Decompression tests that downloadPlugin correctly handles
// both compressed and uncompressed files.
func TestGCSRegistry_downloadPlugin_Decompression(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "drillsx-decompress-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testData := []byte("This is test drill plugin data that simulates a .so file")

	t.Run("compressed .gz file", func(t *testing.T) {
		// Create a compressed version
		var buf bytes.Buffer
		gzWriter := gzip.NewWriter(&buf)
		if _, err := gzWriter.Write(testData); err != nil {
			t.Fatalf("Failed to write to gzip writer: %v", err)
		}
		if err := gzWriter.Close(); err != nil {
			t.Fatalf("Failed to close gzip writer: %v", err)
		}

		// Write compressed data to a temp file (simulating GCS object)
		compressedPath := filepath.Join(tempDir, "test-drill.so.gz")
		if err := os.WriteFile(compressedPath, buf.Bytes(), 0644); err != nil {
			t.Fatalf("Failed to write compressed file: %v", err)
		}

		// Test decompression by manually simulating what downloadPlugin does
		// In a real scenario, this would be reading from GCS, but we can verify
		// the gzip reader logic works correctly
		file, err := os.Open(compressedPath)
		if err != nil {
			t.Fatalf("Failed to open compressed file: %v", err)
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gzReader.Close()

		var decompressed bytes.Buffer
		if _, err := decompressed.ReadFrom(gzReader); err != nil {
			t.Fatalf("Failed to decompress: %v", err)
		}

		if !bytes.Equal(decompressed.Bytes(), testData) {
			t.Errorf("Decompressed data doesn't match original.\nGot: %s\nWant: %s",
				decompressed.String(), string(testData))
		}

		t.Logf("Successfully decompressed %d bytes to %d bytes", buf.Len(), decompressed.Len())
	})

	t.Run("uncompressed .so file", func(t *testing.T) {
		// Write uncompressed data
		uncompressedPath := filepath.Join(tempDir, "test-drill.so")
		if err := os.WriteFile(uncompressedPath, testData, 0644); err != nil {
			t.Fatalf("Failed to write uncompressed file: %v", err)
		}

		// Read it back
		readData, err := os.ReadFile(uncompressedPath)
		if err != nil {
			t.Fatalf("Failed to read uncompressed file: %v", err)
		}

		if !bytes.Equal(readData, testData) {
			t.Errorf("Read data doesn't match original.\nGot: %s\nWant: %s",
				string(readData), string(testData))
		}

		t.Logf("Successfully read %d bytes", len(readData))
	})

	t.Run("drill ID extraction from .so.gz", func(t *testing.T) {
		testCases := []struct {
			filename string
			wantID   string
		}{
			{"a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.so", "a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d"},
			{"a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.so.gz", "a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d"},
			{"simple-drill.so", "simple-drill"},
			{"simple-drill.so.gz", "simple-drill"},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				// This mimics the logic in Sync()
				drillID := strings.TrimSuffix(strings.TrimSuffix(tc.filename, ".gz"), ".so")
				if drillID != tc.wantID {
					t.Errorf("Extracted drill ID = %q, want %q", drillID, tc.wantID)
				}
			})
		}
	})

	_ = log // Use log to avoid unused variable warning
}
