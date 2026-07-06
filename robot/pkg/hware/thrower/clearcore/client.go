// Package clearcore provides a client for communicating with the ClearCore microcontroller
// that controls the tennis ball throwing machine. It handles text-based communication
// following the protocol defined in protocol.md.
package clearcore

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Command names for the text protocol
const (
	cmdThrow = "THROW" // Command to set throw motor speeds and angle
	cmdDisp  = "DISP"  // Command to set dispenser motor speed
	cmdLoad  = "LOAD"  // Command to check if a ball is loaded

	respOK  = "OK"  // Indicates successful command execution
	respErr = "ERR" // Indicates an error occurred during command execution
)

// Client implements the protocol for communicating with the ClearCore microcontroller
// that controls the tennis ball throwing machine. It handles text-based communication
// following the protocol defined in protocol.md.
type Client struct {
	conn    net.Conn      // TCP connection to the ClearCore microcontroller
	timeout time.Duration // Timeout for read/write operations
	reader  *bufio.Reader // Buffered reader for reading responses
	mu      *sync.Mutex   // Mutex for thread-safe operations
}

// NewClient creates a new client with the specified timeout
func NewClient(conn net.Conn, timeout time.Duration) *Client {
	return &Client{
		conn:    conn,
		timeout: timeout,
		reader:  bufio.NewReader(conn),
		mu:      &sync.Mutex{},
	}
}

// SetThrow sets the throw system motors to their respective speeds and adjusts the angle motor.
// Parameters:
// - top: Speed of the top throw motor in RPM (must be positive)
// - bottom: Speed of the bottom throw motor in RPM (must be positive)
// - angle: Throw angle in radians (must be between 0 and π/4)
func (c *Client) SetThrow(top, bottom int, angle float64) error {
	// Validate parameters
	if top < 0 {
		return errors.New("clearcore.Client: top speed must be positive")
	}
	if bottom < 0 {
		return errors.New("clearcore.Client: bottom speed must be positive")
	}
	if angle < 0 || angle > math.Pi/4 {
		return errors.New("clearcore.Client: angle must be between 0 and π/4 radians")
	}

	// Create command string
	cmd := fmt.Sprintf("%s %d %d %.4f", cmdThrow, top, bottom, angle)

	// Send command and get response
	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	// Parse response (should just be "OK")
	if !strings.HasPrefix(resp, respOK) {
		return fmt.Errorf("clearcore.Client: unexpected response: %s", resp)
	}

	return nil
}

// SetDispenser sets the ball dispenser motor to the given speed.
// Parameters:
// - speed: Speed of the dispenser motor in RPM (must be positive)
func (c *Client) SetDispenser(speed int) error {
	// Validate parameters
	if speed < 0 {
		return errors.New("clearcore.Client: dispenser speed must be positive")
	}

	// Create command string
	cmd := fmt.Sprintf("%s %d", cmdDisp, speed)

	// Send command and get response
	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	// Parse response (should just be "OK")
	if !strings.HasPrefix(resp, respOK) {
		return fmt.Errorf("clearcore.Client: unexpected response: %s", resp)
	}

	return nil
}

// GetLoaded checks if a ball is loaded by querying the ClearCore microcontroller.
// Returns:
// - bool: true if a ball is loaded, false otherwise
// - error: any error that occurred during the operation
func (c *Client) GetLoaded() (bool, error) {
	// Send command
	resp, err := c.sendCommand(cmdLoad)
	if err != nil {
		return false, err
	}

	// Parse response (should be "OK 0" or "OK 1")
	parts := strings.Fields(resp)
	if len(parts) != 2 || parts[0] != respOK {
		return false, fmt.Errorf("clearcore.Client: unexpected response format: %s", resp)
	}

	// Convert the second part to a boolean
	loaded, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("clearcore.Client: invalid response value: %s", parts[1])
	}

	return loaded == 1, nil
}

// sendCommand sends a command to the ClearCore and returns the response.
// It handles the text-based protocol communication.
// Parameters:
// - cmd: The command string to send
// Returns:
// - string: The response from the controller
// - error: Any error that occurred during communication
func (c *Client) sendCommand(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Add newline to command
	cmdWithNewline := cmd + "\n"

	// Set write timeout
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", fmt.Errorf("clearcore.Client: failed to set write deadline: %w", err)
	}

	// Send command
	_, err := c.conn.Write([]byte(cmdWithNewline))
	if err != nil {
		return "", fmt.Errorf("clearcore.Client: failed to send command: %w", err)
	}

	// Set read timeout
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", fmt.Errorf("clearcore.Client: failed to set read deadline: %w", err)
	}

	// Read response line
	respLine, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("clearcore.Client: failed to read response: %w", err)
	}

	// Trim trailing newline
	respLine = strings.TrimSuffix(respLine, "\n")

	// Check for error response
	if strings.HasPrefix(respLine, respErr) {
		errMsg := respLine[4:] // Skip "ERR " prefix
		return "", fmt.Errorf("clearcore.Client: device returned error: %s", errMsg)
	}

	return respLine, nil
}
