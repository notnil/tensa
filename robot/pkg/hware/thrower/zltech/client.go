// Package zltech provides a CANopen client for controlling a single ZLAC8015D wheel motor.
// This is a simplified implementation focused on single-wheel control for thrower applications.
//
// Based on:
//   - ZLAC8015D CANopen Communication Quick-Start Guide V1.00
//   - ZLAC8015D CANopen Communication Routine V1.07
//   - ZLAC8015D Servo Driver Manual V1.03
package zltech

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/canbus/canopen"
)

// CANopen Object Dictionary indices
const (
	idxControlWord      = 0x6040 // U16 - CiA402 control word
	idxStatusWord       = 0x6041 // U16 - CiA402 status word
	idxModeOfOperation  = 0x6060 // I8 - Operating mode
	idxModeDisplay      = 0x6061 // I8 - Current operating mode
	idxTargetVelocity   = 0x60FF // I32 - Target velocity in RPM
	idxVelocityActual   = 0x606C // I32 - Actual velocity in 0.1 RPM
	idxProfileAcc       = 0x6083 // U32 - Acceleration time in ms
	idxProfileDec       = 0x6084 // U32 - Deceleration time in ms
	idxFaultCode        = 0x603F // U16 - Error code
	idxMotorTemperature = 0x2032 // U16 - Motor temperature (0.1°C)
	idxTorqueActual     = 0x6077 // I16 - Actual torque/current (0.1A)
	idxSyncAsyncControl = 0x200F // U16 - Sync/Async mode (0=async, 1=sync)
)

// CiA402 control word commands
const (
	cwShutdown       = 0x0006 // Shutdown command
	cwSwitchOn       = 0x0007 // Switch on command
	cwEnableOp       = 0x000F // Enable operation command
	cwDisableVoltage = 0x0000 // Disable voltage (freewheel)
	cwQuickStop      = 0x0002 // Quick stop command
	cwFaultReset     = 0x0080 // Fault reset command
)

// Operating modes
const (
	modeProfileVel = 3 // Profile velocity mode
)

// Side indicates which motor of the dual-motor controller to use
type Side int

const (
	Left Side = iota
	Right
)

func (s Side) String() string {
	if s == Left {
		return "left"
	}
	return "right"
}

// Client provides control for a single ZLAC8015D wheel motor
type Client struct {
	nodeID      byte               // CANopen node ID (1-127)
	side        Side               // Which motor (left or right)
	singleMotor bool               // If true, use sub-index 0 (single motor mode)
	bus         canbus.Bus         // CAN bus interface
	mux         *canbus.Mux        // CAN bus multiplexer
	sdo         *canopen.SDOClient // SDO client for object dictionary access
	logger      *slog.Logger       // Structured logger

	// Cached acceleration/deceleration values
	lastAccMs uint32
	lastDecMs uint32

	// Keep-alive mechanism
	keepAliveInterval time.Duration
	keepAliveCancel   context.CancelFunc
	keepAliveWG       sync.WaitGroup

	// Cached last command for keep-alive (protected by mutex)
	mu      sync.RWMutex
	lastRPM int32
	enabled bool // Track enabled/disabled state
}

// Config contains configuration for the wheel client
type Config struct {
	NodeID            byte          // CANopen node ID (default: 1)
	Side              Side          // Which motor to control (Left or Right)
	SingleMotor       bool          // If true, use sub-index 0 (single motor mode). If false, use 1/2 (dual motor mode)
	Logger            *slog.Logger  // Optional logger
	KeepAliveInterval time.Duration // 0 = disabled, >0 = enabled (recommend 500ms to prevent communication timeout)
	Mux               *canbus.Mux   // Optional shared Mux. If nil, a new Mux will be created. When controlling multiple wheels on the same bus, all clients MUST share the same Mux.
}

// New creates a new single-wheel ZLAC8015D client
func New(bus canbus.Bus, cfg Config) *Client {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.NodeID == 0 {
		cfg.NodeID = 1
	}

	// Use provided Mux or create a new one
	// When controlling multiple wheels on the same bus, all clients must share the same Mux
	mux := cfg.Mux
	if mux == nil {
		mux = canbus.NewMux(bus)
	}

	sdo := canopen.NewSDOClient(
		bus,
		canopen.NodeID(cfg.NodeID),
		mux,
		canopen.WithTimeout(1*time.Second),
		canopen.WithExpeditedMode(canopen.ExpeditedModeClassic),
		canopen.WithLenientUpload(),
	)

	return &Client{
		nodeID:            cfg.NodeID,
		side:              cfg.Side,
		singleMotor:       cfg.SingleMotor,
		bus:               bus,
		mux:               mux,
		sdo:               sdo,
		logger:            cfg.Logger,
		keepAliveInterval: cfg.KeepAliveInterval,
	}
}

// Initialize configures the wheel and transitions it to operational state
// This performs the CiA402 state machine initialization sequence
func (c *Client) Initialize(ctx context.Context) error {
	c.logger.Info("initializing zltech wheel", "node_id", c.nodeID, "side", c.side)

	// Enter pre-operational state for configuration
	if err := c.preOperational(); err != nil {
		return fmt.Errorf("enter pre-operational: %w", err)
	}
	time.Sleep(20 * time.Millisecond)

	// Clear any existing faults
	if err := c.clearFault(); err != nil {
		return fmt.Errorf("clear fault: %w", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Set asynchronous control mode (0x200F = 0)
	if err := c.sdo.WriteU16(idxSyncAsyncControl, 0, 0); err != nil {
		return fmt.Errorf("set async mode: %w", err)
	}

	// Enter operational state
	if err := c.startNode(); err != nil {
		return fmt.Errorf("start node: %w", err)
	}
	time.Sleep(20 * time.Millisecond)

	// CiA402 state machine sequence
	// Step 1: Shutdown -> Ready to Switch On
	if err := c.writeControlWord(cwShutdown); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Step 2: Switch On -> Switched On
	if err := c.writeControlWord(cwSwitchOn); err != nil {
		return fmt.Errorf("switch on: %w", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Step 3: Enable Operation -> Operation Enabled
	if err := c.writeControlWord(cwEnableOp); err != nil {
		return fmt.Errorf("enable operation: %w", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Set to profile velocity mode
	if err := c.sdo.WriteU8(idxModeOfOperation, 0, modeProfileVel); err != nil {
		return fmt.Errorf("set velocity mode: %w", err)
	}

	// Mark as enabled and start keep-alive if configured
	c.mu.Lock()
	c.enabled = true
	c.lastRPM = 0 // Start with zero velocity
	c.mu.Unlock()

	if c.keepAliveInterval > 0 {
		c.startKeepAlive()
	}

	c.logger.Info("zltech wheel initialized successfully")
	return nil
}

// Spin sets the wheel to spin at the given RPM
// Positive values spin forward, negative values spin backward
// accMs and decMs specify acceleration and deceleration times in milliseconds
func (c *Client) Spin(ctx context.Context, rpm int32, accMs, decMs uint32) error {
	// Update acceleration if changed
	if c.lastAccMs != accMs {
		if err := c.sdo.WriteU32(idxProfileAcc, c.subIndex(), accMs); err != nil {
			return fmt.Errorf("set acceleration: %w", err)
		}
		c.lastAccMs = accMs
	}

	// Update deceleration if changed
	if c.lastDecMs != decMs {
		if err := c.sdo.WriteU32(idxProfileDec, c.subIndex(), decMs); err != nil {
			return fmt.Errorf("set deceleration: %w", err)
		}
		c.lastDecMs = decMs
	}

	// Set target velocity
	if err := c.sdo.WriteU32(idxTargetVelocity, c.subIndex(), uint32(rpm)); err != nil {
		return fmt.Errorf("set velocity: %w", err)
	}

	// Update cached command for keep-alive
	c.mu.Lock()
	c.lastRPM = rpm
	c.mu.Unlock()

	c.logger.Debug("wheel spinning", "rpm", rpm, "acc_ms", accMs, "dec_ms", decMs)
	return nil
}

// Stop brings the wheel to a stop using the configured deceleration
func (c *Client) Stop(ctx context.Context) error {
	// Update cached command first so keep-alive sends 0
	c.mu.Lock()
	c.lastRPM = 0
	c.mu.Unlock()

	return c.Spin(ctx, 0, c.lastAccMs, c.lastDecMs)
}

// Enable enables the motor driver (motor is engaged and can produce torque)
// After enabling, the target velocity is set to zero for safety
func (c *Client) Enable(ctx context.Context) error {
	c.logger.Debug("enabling wheel")

	// Read current status to determine the right sequence
	sw, err := c.readStatusWord()
	if err != nil {
		return fmt.Errorf("read status: %w", err)
	}

	state := sw.State()
	c.logger.Debug("current state", "state", state)

	// Already enabled - still set velocity to zero
	if state == StateOperationEnabled {
		return c.setVelocityZero()
	}

	// If in fault, recover first
	if state == StateFault {
		if err := c.recoverFromFault(); err != nil {
			return err
		}
		return c.setVelocityZero()
	}

	// Walk through state machine to reach Operation Enabled
	var enableErr error
	switch state {
	case StateSwitchedOn:
		enableErr = c.writeControlWord(cwEnableOp)

	case StateReadyToSwitchOn:
		if err := c.writeControlWord(cwSwitchOn); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		enableErr = c.writeControlWord(cwEnableOp)

	default:
		// Full sequence from any other state
		if err := c.writeControlWord(cwShutdown); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		if err := c.writeControlWord(cwSwitchOn); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		enableErr = c.writeControlWord(cwEnableOp)
	}

	if enableErr != nil {
		return enableErr
	}

	// Set velocity to zero for safety
	c.mu.Lock()
	c.enabled = true
	c.lastRPM = 0
	c.mu.Unlock()

	return c.setVelocityZero()
}

// Disable disables the motor driver (motor freewheels, no torque)
func (c *Client) Disable(ctx context.Context) error {
	c.logger.Debug("disabling wheel (freewheel)")

	// Update state so keep-alive sends 0 while disabled
	c.mu.Lock()
	c.enabled = false
	c.mu.Unlock()

	return c.writeControlWord(cwDisableVoltage)
}

// Status returns current status and diagnostic information for the wheel
func (c *Client) Status(ctx context.Context) (*Status, error) {
	// Read status word
	sw, err := c.readStatusWord()
	if err != nil {
		return nil, fmt.Errorf("read status word: %w", err)
	}

	// Read actual velocity (in 0.1 RPM units)
	rawVel, err := c.sdo.ReadU32(idxVelocityActual, c.subIndex())
	if err != nil {
		return nil, fmt.Errorf("read velocity: %w", err)
	}
	actualRPM := float64(int32(rawVel)) / 10.0

	// Read motor temperature (in 0.1°C units)
	// rawTemp, err := c.sdo.ReadU16(idxMotorTemperature, c.tempSubIndex())
	// if err != nil {
	// 	return nil, fmt.Errorf("read temperature: %w", err)
	// }
	// This currently fails, so we'll use a placeholder
	tempC := 0.0
	// tempC := float64(int16(rawTemp)) / 10.0

	// Read actual current/torque (in 0.1A units)
	rawCurrent, err := c.sdo.ReadU16(idxTorqueActual, c.subIndex())
	if err != nil {
		return nil, fmt.Errorf("read current: %w", err)
	}
	currentA := float64(int16(rawCurrent)) / 10.0

	// Read fault code
	faultRaw, err := c.sdo.ReadU32(idxFaultCode, 0)
	if err != nil {
		return nil, fmt.Errorf("read fault code: %w", err)
	}

	// Extract fault code for this motor
	var faultCode uint16
	if c.side == Left {
		faultCode = uint16(faultRaw & 0xFFFF) // Low 16 bits
	} else {
		faultCode = uint16((faultRaw >> 16) & 0xFFFF) // High 16 bits
	}

	return &Status{
		State:       sw.State(),
		ActualRPM:   actualRPM,
		Temperature: tempC,
		Current:     currentA,
		FaultCode:   FaultCode(faultCode),
		StatusWord:  sw,
	}, nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.logger.Info("closing zltech wheel client")

	// Stop keep-alive goroutine
	c.stopKeepAlive()

	// Stop the wheel before closing
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = c.Stop(ctx)
	return nil
}

// Helper methods

func (c *Client) subIndex() uint8 {
	if c.singleMotor {
		return 0
	}
	if c.side == Left {
		return 1
	}
	return 2
}

func (c *Client) tempSubIndex() uint8 {
	if c.singleMotor {
		return 0
	}
	if c.side == Left {
		return 1 // subTempLeft
	}
	return 2 // subTempRight
}

func (c *Client) writeControlWord(value uint16) error {
	return c.sdo.WriteU16(idxControlWord, 0, value)
}

func (c *Client) readStatusWord() (StatusWord, error) {
	// Try 32-bit read first (some firmware responds with 4 bytes)
	val32, err := c.sdo.ReadU32(idxStatusWord, 0)
	if err == nil {
		return StatusWord(uint16(val32 & 0xFFFF)), nil
	}

	// Fall back to 16-bit read
	val16, err := c.sdo.ReadU16(idxStatusWord, 0)
	if err != nil {
		return 0, err
	}
	return StatusWord(val16), nil
}

func (c *Client) clearFault() error {
	return c.writeControlWord(cwFaultReset)
}

func (c *Client) recoverFromFault() error {
	if err := c.clearFault(); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	if err := c.writeControlWord(cwShutdown); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	if err := c.writeControlWord(cwSwitchOn); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	return c.writeControlWord(cwEnableOp)
}

func (c *Client) startNode() error {
	nmt := canopen.NMT{Command: canopen.NMTStart, Node: c.nodeID}
	frame, _ := nmt.MarshalCANFrame()
	c.logger.Debug("tx nmt start", "id", fmt.Sprintf("0x%03X", frame.ID))
	return c.bus.Send(frame)
}

func (c *Client) preOperational() error {
	nmt := canopen.NMT{Command: canopen.NMTEnterPreOperational, Node: c.nodeID}
	frame, _ := nmt.MarshalCANFrame()
	c.logger.Debug("tx nmt pre-operational", "id", fmt.Sprintf("0x%03X", frame.ID))
	return c.bus.Send(frame)
}

func (c *Client) setVelocityZero() error {
	if err := c.sdo.WriteU32(idxTargetVelocity, c.subIndex(), 0); err != nil {
		return fmt.Errorf("set velocity to zero: %w", err)
	}
	c.logger.Debug("velocity set to zero")
	return nil
}

// Keep-alive methods

// startKeepAlive starts a background goroutine that periodically re-sends
// the last velocity command to prevent communication offline timeout.
func (c *Client) startKeepAlive() {
	// Stop any existing keep-alive first
	c.stopKeepAlive()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	c.keepAliveCancel = cancel

	c.keepAliveWG.Add(1)
	go func() {
		defer c.keepAliveWG.Done()

		ticker := time.NewTicker(c.keepAliveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				c.logger.Debug("keep-alive goroutine stopping")
				return
			case <-ticker.C:
				c.sendKeepAliveCommand()
			}
		}
	}()

	c.logger.Info("keep-alive started", "interval", c.keepAliveInterval)
}

// stopKeepAlive stops the keep-alive goroutine and waits for it to complete.
func (c *Client) stopKeepAlive() {
	if c.keepAliveCancel != nil {
		c.keepAliveCancel()
		c.keepAliveWG.Wait()
		c.keepAliveCancel = nil
		c.logger.Debug("keep-alive stopped")
	}
}

// sendKeepAliveCommand sends the current velocity command to keep communication alive.
// If disabled, it sends zero velocity to maintain the channel without spinning the motor.
func (c *Client) sendKeepAliveCommand() {
	c.mu.RLock()
	enabled := c.enabled
	rpm := c.lastRPM
	accMs := c.lastAccMs
	decMs := c.lastDecMs
	c.mu.RUnlock()

	// If disabled, send zero velocity to keep communication alive
	if !enabled {
		rpm = 0
	}

	// Use default acceleration/deceleration if not set
	if accMs == 0 {
		accMs = 100
	}
	if decMs == 0 {
		decMs = 100
	}

	// Send the velocity command
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := c.sendVelocityCommand(ctx, rpm, accMs, decMs); err != nil {
		c.logger.Warn("keep-alive command failed", "error", err, "rpm", rpm)
	} else {
		c.logger.Debug("keep-alive command sent", "rpm", rpm, "enabled", enabled)
	}
}

// sendVelocityCommand is a helper that sends velocity commands without updating cached state.
// This is used by the keep-alive mechanism to avoid interfering with the cached lastRPM.
func (c *Client) sendVelocityCommand(ctx context.Context, rpm int32, accMs, decMs uint32) error {
	// Update acceleration if needed (this is shared state, so still update cache)
	if c.lastAccMs != accMs {
		if err := c.sdo.WriteU32(idxProfileAcc, c.subIndex(), accMs); err != nil {
			return fmt.Errorf("set acceleration: %w", err)
		}
		c.lastAccMs = accMs
	}

	// Update deceleration if needed
	if c.lastDecMs != decMs {
		if err := c.sdo.WriteU32(idxProfileDec, c.subIndex(), decMs); err != nil {
			return fmt.Errorf("set deceleration: %w", err)
		}
		c.lastDecMs = decMs
	}

	// Set target velocity
	if err := c.sdo.WriteU32(idxTargetVelocity, c.subIndex(), uint32(rpm)); err != nil {
		return fmt.Errorf("set velocity: %w", err)
	}

	return nil
}
