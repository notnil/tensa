package thrower_test

import (
	"context"
	"math"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/hware/thrower"
	"github.com/notnil/tensa/pkg/hware/thrower/clearcore"
	"github.com/notnil/tensa/pkg/util/jsonx"
)

func TestClearCoreThrower_Set(t *testing.T) {
	// Create a pipe for testing
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	// Create the client
	client := clearcore.NewClient(clientConn, time.Second)

	// Create the thrower with configuration
	cfg := clearcore.Config{
		DispenserSpeed:      10.0,
		ThrowDuration:       jsonx.Duration(100 * time.Millisecond),
		LoadTimeoutDuration: jsonx.Duration(500 * time.Millisecond),
		LoadPollInterval:    jsonx.Duration(50 * time.Millisecond),
		MaxThrowSpeed:       100.0,
		LoadBeforeThrow:     false,
	}
	thr := clearcore.New(client, cfg)

	// Set up the server side to handle the request
	go func() {
		// Read the request
		buf := make([]byte, 1024)
		n, err := server.Read(buf)
		if err != nil {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Verify it's a THROW command in the text protocol
		cmd := string(buf[:n])
		if !strings.HasPrefix(cmd, "THROW ") {
			t.Errorf("Expected THROW command, got: %s", cmd)
			return
		}

		// Send a success response
		if _, err := server.Write([]byte("OK\n")); err != nil {
			t.Errorf("Failed to send response: %v", err)
		}
	}()

	// Test the Set method with valid parameters
	settings := thrower.Settings{
		Top:    50.0,
		Bottom: 60.0,
		Angle:  0.5,
	}

	err := thr.Set(settings)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestClearCoreThrower_SetInvalidParams(t *testing.T) {
	// Create a pipe for testing
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	// Create the client
	client := clearcore.NewClient(clientConn, time.Second)

	// Create the thrower with configuration
	cfg := clearcore.Config{
		DispenserSpeed:      10.0,
		ThrowDuration:       jsonx.Duration(100 * time.Millisecond),
		LoadTimeoutDuration: jsonx.Duration(500 * time.Millisecond),
		LoadPollInterval:    jsonx.Duration(50 * time.Millisecond),
		MaxThrowSpeed:       100.0,
	}
	thr := clearcore.New(client, cfg)

	tests := []struct {
		name     string
		settings thrower.Settings
		wantErr  string
	}{
		{
			name: "AngleTooSmall",
			settings: thrower.Settings{
				Top:    50.0,
				Bottom: 60.0,
				Angle:  -0.1,
			},
			wantErr: "invalid angle",
		},
		{
			name: "AngleTooLarge",
			settings: thrower.Settings{
				Top:    50.0,
				Bottom: 60.0,
				Angle:  math.Pi,
			},
			wantErr: "invalid angle",
		},
		{
			name: "TopSpeedTooHigh",
			settings: thrower.Settings{
				Top:    150.0,
				Bottom: 60.0,
				Angle:  0.5,
			},
			wantErr: "top speed too high",
		},
		{
			name: "BottomSpeedTooHigh",
			settings: thrower.Settings{
				Top:    50.0,
				Bottom: 150.0,
				Angle:  0.5,
			},
			wantErr: "bottom speed too high",
		},
		{
			name: "TopSpeedNegative",
			settings: thrower.Settings{
				Top:    -10.0,
				Bottom: 60.0,
				Angle:  0.5,
			},
			wantErr: "top speed too low",
		},
		{
			name: "BottomSpeedNegative",
			settings: thrower.Settings{
				Top:    50.0,
				Bottom: -10.0,
				Angle:  0.5,
			},
			wantErr: "bottom speed too low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := thr.Set(tt.settings)
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing '%s', got '%v'", tt.wantErr, err)
			}
		})
	}
}

func TestClearCoreThrowerLoad(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	client := clearcore.NewClient(clientConn, time.Second)

	cfg := clearcore.Config{
		DispenserSpeed:      10.0,
		ThrowDuration:       jsonx.Duration(100 * time.Millisecond),
		LoadTimeoutDuration: jsonx.Duration(200 * time.Millisecond),
		LoadPollInterval:    jsonx.Duration(10 * time.Millisecond),
		MaxThrowSpeed:       100.0,
	}

	thr := clearcore.New(client, cfg)

	// Start a goroutine to handle server responses
	go func() {
		startTime := time.Now()
		for {
			// Read request
			buf := make([]byte, 1024)
			n, err := server.Read(buf)
			if err != nil {
				return // Exit if connection closed
			}

			// Check if it's a LOAD command
			cmd := string(buf[:n])
			if strings.TrimSpace(cmd) == "LOAD" {
				// Respond with loaded=true only after 50ms has passed
				elapsed := time.Since(startTime)
				loaded := elapsed >= 50*time.Millisecond

				// Create text protocol response
				var response string
				if loaded {
					response = "OK 1\n"
				} else {
					response = "OK 0\n"
				}

				if _, err := server.Write([]byte(response)); err != nil {
					t.Errorf("Failed to send response: %v", err)
					return
				}
			} else {
				// For any other command, just respond with OK
				if _, err := server.Write([]byte("OK\n")); err != nil {
					t.Errorf("Failed to send response: %v", err)
					return
				}
			}
		}
	}()

	// Test the Load method
	startTime := time.Now()
	err := thr.Load(context.Background())
	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that it took at least 50ms to load
	if elapsed < 40*time.Millisecond {
		t.Errorf("Load completed too quickly. Expected at least 40ms, got %v", elapsed)
	}
}
