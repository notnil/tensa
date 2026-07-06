package wheels

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// DefaultCommandTimeout is the default time after which DefensiveWheels will automatically
	// stop the robot if no movement commands are received. This provides safety against
	// controller disconnects, sticky controllers, software crashes, or communication failures.
	DefaultCommandTimeout = 300 * time.Millisecond

	// DefaultMonitorInterval is how often the defensive monitor checks for command timeout.
	DefaultMonitorInterval = 50 * time.Millisecond
)

// DefensiveWheels wraps any Wheels implementation with automatic timeout protection.
// If no Move or Rotate commands are received within the configured timeout period,
// the robot is automatically stopped for safety.
//
// This addresses safety issues with:
//   - Sticky BLE controllers that continue sending commands
//   - Controller disconnects that leave the robot moving
//   - Software crashes that stop sending commands
//   - Communication failures that interrupt the control loop
//
// The wrapper is generic and works with any Wheels implementation (zltech, mock, etc.).
type DefensiveWheels struct {
	wheels          Wheels        // Underlying wheels implementation
	commandTimeout  time.Duration // Timeout after which to stop if no commands received
	monitorInterval time.Duration // How often to check for timeout
	logger          *slog.Logger  // Logger for debugging timeout events

	// Thread-safe state management
	mu              sync.RWMutex // Protects lastCommandTime and hasTimedOut
	lastCommandTime time.Time    // Timestamp of the last Move/Rotate command received
	hasTimedOut     bool         // Tracks if we've already stopped due to timeout

	// Goroutine lifecycle management
	ctx    context.Context    // Context for monitor loop cancellation
	cancel context.CancelFunc // Function to cancel monitor loop
}

// DefensiveConfig holds configuration for DefensiveWheels initialization.
type DefensiveConfig struct {
	CommandTimeout  time.Duration // How long to wait before auto-stopping if no commands received (0 = use DefaultCommandTimeout)
	MonitorInterval time.Duration // How often to check for timeout (0 = use DefaultMonitorInterval)
}

// NewDefensive creates a new DefensiveWheels instance that wraps the provided Wheels
// with automatic command timeout protection.
//
// The implementation immediately starts a background goroutine (monitorLoop) that:
//  1. Monitors the time since the last movement command
//  2. Automatically stops the robot if no commands are received within the timeout period
//  3. Provides protection against controller disconnects and sticky controllers
//
// Parameters:
//   - wheels: The underlying Wheels implementation to wrap
//   - timeout: Command timeout duration (0 = use DefaultCommandTimeout)
//   - logger: Logger for debugging timeout events
//
// The returned DefensiveWheels implements io.Closer and should be closed when no longer needed.
func NewDefensive(wheels Wheels, timeout time.Duration, logger *slog.Logger) *DefensiveWheels {
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	dw := &DefensiveWheels{
		wheels:          wheels,
		commandTimeout:  timeout,
		monitorInterval: DefaultMonitorInterval,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		lastCommandTime: time.Now(), // Initialize to prevent immediate timeout
		hasTimedOut:     false,
	}

	// Start the monitoring goroutine immediately
	go dw.monitorLoop()

	return dw
}

// Move commands the robot to translate in a specific direction at a given linear speed.
//
// This method updates the last command timestamp and passes the command through to the
// underlying wheels implementation. The timestamp update resets the timeout counter,
// preventing automatic stops as long as commands continue to arrive.
//
// Parameters:
//   - dir: Direction of movement in radians
//   - speed: Linear speed in radians per second
func (dw *DefensiveWheels) Move(dir, speed float64) error {
	// Update last command time and reset timeout state
	dw.mu.Lock()
	dw.lastCommandTime = time.Now()
	dw.hasTimedOut = false
	dw.mu.Unlock()

	// Pass through to underlying wheels
	return dw.wheels.Move(dir, speed)
}

// Rotate commands the robot to spin in place at a specified rotational speed.
//
// Like Move(), this method updates the last command timestamp and passes through to the
// underlying wheels implementation, resetting the timeout counter.
//
// Parameters:
//   - speed: Angular speed in radians per second
func (dw *DefensiveWheels) Rotate(speed float64) error {
	// Update last command time and reset timeout state
	dw.mu.Lock()
	dw.lastCommandTime = time.Now()
	dw.hasTimedOut = false
	dw.mu.Unlock()

	// Pass through to underlying wheels
	return dw.wheels.Rotate(speed)
}

// Stop halts all robot movement immediately and resets the timeout state.
//
// CRITICAL SAFETY FEATURE: Stop commands are passed through immediately and also
// reset the timeout tracking, preventing spurious timeout messages after an explicit stop.
func (dw *DefensiveWheels) Stop() error {
	// Reset timeout state since this is an explicit stop
	dw.mu.Lock()
	dw.hasTimedOut = false
	// Note: We don't update lastCommandTime here to maintain the timeout behavior
	// for subsequent movement commands
	dw.mu.Unlock()

	// Pass through stop command immediately
	return dw.wheels.Stop()
}

// Status retrieves the current operational status of all four wheels.
// This is a pass-through to the underlying wheels implementation.
func (dw *DefensiveWheels) Status() (Status, error) {
	return dw.wheels.Status()
}

// Disable deactivates the motors.
// This is a pass-through to the underlying wheels implementation.
func (dw *DefensiveWheels) Disable() error {
	return dw.wheels.Disable()
}

// Enable activates the motors.
// This is a pass-through to the underlying wheels implementation.
func (dw *DefensiveWheels) Enable() error {
	return dw.wheels.Enable()
}

// Close terminates the DefensiveWheels instance and cleans up resources.
//
// This method:
//  1. Cancels the monitor loop goroutine
//  2. Closes the underlying wheels implementation (if it implements io.Closer)
//  3. Releases any held resources
//
// Should be called when the DefensiveWheels instance is no longer needed.
func (dw *DefensiveWheels) Close() error {
	// Stop the monitor goroutine gracefully
	dw.cancel()

	// Close the underlying wheels implementation if it supports Close
	type closer interface {
		Close() error
	}
	if c, ok := dw.wheels.(closer); ok {
		return c.Close()
	}
	return nil
}

// monitorLoop runs in a separate goroutine and monitors for command timeout.
//
// This loop serves a critical safety purpose:
//   - Monitors the time since the last movement command
//   - Automatically stops the robot if the timeout period is exceeded
//   - Provides protection against controller disconnects, sticky controllers, and communication failures
//   - Only logs and stops once per timeout event to avoid spam
//
// The loop checks at regular intervals (DefaultMonitorInterval) and terminates gracefully
// when the context is cancelled (during Close()).
func (dw *DefensiveWheels) monitorLoop() {
	ticker := time.NewTicker(dw.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dw.ctx.Done():
			// Context cancelled, exit gracefully
			return
		case <-ticker.C:
			// Time to check for timeout
			dw.mu.RLock()
			lastCmdTime := dw.lastCommandTime
			hasTimedOut := dw.hasTimedOut
			dw.mu.RUnlock()

			timeSinceLastCmd := time.Since(lastCmdTime)

			// Check if we've exceeded the command timeout
			if timeSinceLastCmd > dw.commandTimeout && !hasTimedOut {
				// Timeout exceeded - stop the robot for safety
				dw.logger.Warn("DefensiveWheels: Command timeout exceeded, stopping robot for safety",
					"timeout", dw.commandTimeout,
					"time_since_last_command", timeSinceLastCmd)

				// Mark that we've timed out to prevent repeated stops/logs
				dw.mu.Lock()
				dw.hasTimedOut = true
				dw.mu.Unlock()

				if err := dw.wheels.Rotate(0); err != nil {
					dw.logger.Error("DefensiveWheels: Failed to stop robot on timeout", "error", err)
				}
			}
		}
	}
}
