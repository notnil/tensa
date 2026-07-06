package clearcore

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()

	timeout := 2 * time.Second
	cc := NewClient(clientConn, timeout)

	if cc.conn != clientConn {
		t.Errorf("Expected connection %v, got %v", clientConn, cc.conn)
	}
	if cc.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, cc.timeout)
	}
	if cc.reader == nil {
		t.Errorf("Expected reader to be initialized, got nil")
	}
}

func TestSetThrowInvalidParams(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	tests := []struct {
		name    string
		top     int
		bottom  int
		angle   float64
		wantErr string
	}{
		{"NegativeTop", -1, 10, 0.1, "top speed must be positive"},
		{"NegativeBottom", 10, -1, 0.1, "bottom speed must be positive"},
		{"AngleTooSmall", 10, 10, -0.1, "angle must be between 0 and π/4 radians"},
		{"AngleTooLarge", 10, 10, 0.7854, "angle must be between 0 and π/4 radians"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cc.SetThrow(tt.top, tt.bottom, tt.angle)
			if err == nil || err.Error() != "clearcore.Client: "+tt.wantErr {
				t.Errorf("Expected error '%s', got '%v'", tt.wantErr, err)
			}
		})
	}
}

func TestSetThrowValid(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	go func() {
		// Read and verify request
		buf := make([]byte, 1024)
		n, err := server.Read(buf)
		if err != nil {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Verify the command format (THROW top bottom angle)
		cmd := string(buf[:n])
		if cmd != "THROW 10 10 0.5000\n" {
			t.Errorf("Invalid command format: %q", cmd)
			return
		}

		// Send response with OK
		if _, err := server.Write([]byte("OK\n")); err != nil {
			t.Errorf("Failed to send response: %v", err)
		}
	}()

	err := cc.SetThrow(10, 10, 0.5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSetDispenserInvalidSpeed(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	err := cc.SetDispenser(-1)
	if err == nil || err.Error() != "clearcore.Client: dispenser speed must be positive" {
		t.Errorf("Expected speed validation error, got %v", err)
	}
}

func TestSetDispenserValid(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	go func() {
		// Read and verify request
		buf := make([]byte, 1024)
		n, err := server.Read(buf)
		if err != nil {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Verify the command format (DISP speed)
		cmd := string(buf[:n])
		if cmd != "DISP 10\n" {
			t.Errorf("Invalid command format: %q", cmd)
			return
		}

		// Send response with OK
		if _, err := server.Write([]byte("OK\n")); err != nil {
			t.Errorf("Failed to send response: %v", err)
		}
	}()

	err := cc.SetDispenser(10)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetLoadedResponses(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
		wantErr  bool
		errMsg   string
	}{
		{"LoadedTrue", "OK 1\n", true, false, ""},
		{"LoadedFalse", "OK 0\n", false, false, ""},
		{"ErrorResponse", "ERR sensor failure\n", false, true, "device returned error: sensor failure"},
		{"InvalidFormat", "OK\n", false, true, "unexpected response format"},
		{"InvalidValue", "OK x\n", false, true, "invalid response value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, clientConn := net.Pipe()
			defer server.Close()
			cc := NewClient(clientConn, time.Second)

			go func() {
				// Read the request first
				buf := make([]byte, 1024)
				_, err := server.Read(buf)
				if err != nil {
					t.Errorf("Failed to read request: %v", err)
					return
				}

				// Send the test response
				if _, err := server.Write([]byte(tt.response)); err != nil {
					t.Errorf("Failed to send response: %v", err)
				}
			}()

			got, err := cc.GetLoaded()
			if tt.wantErr {
				if err == nil || (tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg)) {
					t.Errorf("Expected error with '%s', got %v", tt.errMsg, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("Expected %t, got %t (err: %v)", tt.want, got, err)
			}
		})
	}
}

func TestSendCommandErrorResponse(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	go func() {
		// Read the request first
		buf := make([]byte, 1024)
		_, err := server.Read(buf)
		if err != nil {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Send an error response
		_, err = server.Write([]byte("ERR something went wrong\n"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	}()

	err := cc.SetThrow(10, 10, 0.1)
	if err == nil || !strings.Contains(err.Error(), "device returned error: something went wrong") {
		t.Errorf("Expected device error, got %v", err)
	}
}

func TestSendCommandUnexpectedResponse(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, time.Second)

	go func() {
		// Read the request first
		buf := make([]byte, 1024)
		_, err := server.Read(buf)
		if err != nil {
			t.Errorf("Failed to read request: %v", err)
			return
		}

		// Send unexpected response
		_, err = server.Write([]byte("UNEXPECTED\n"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	}()

	err := cc.SetThrow(10, 10, 0.1)
	if err == nil || !strings.Contains(err.Error(), "unexpected response") {
		t.Errorf("Expected unexpected response error, got %v", err)
	}
}

func TestCommandTimeout(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	cc := NewClient(clientConn, 10*time.Millisecond) // Very short timeout

	go func() {
		// Read request to prevent write deadlock
		buf := make([]byte, 1024)
		server.Read(buf) // Ignore errors
		// Don't write response to force timeout
	}()

	_, err := cc.GetLoaded()
	if err == nil {
		t.Error("Expected timeout error, got nil")
	} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got %v", err)
	}
}
