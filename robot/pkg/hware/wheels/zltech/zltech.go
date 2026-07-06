// zltech.go
//
// Package zltech implements a CAN‑open client for the ZLAC8015D dual‑hub‑servo
// driver from ZhongLing Technology (ZLTECH).  All commands, object indices and
// behaviours are taken from:
//
//   - ZLAC8015D CAN‑open Communication Quick‑Start Guide V1.00
//   - ZLAC8015D CAN‑open Communication Routine V1.07
//   - ZLAC8015D Servo Driver Manual V1.03
//
// CAN bus functionality is implemented in separate platform-specific files
// (can_linux.go, can_darwin.go, can_windows.go) for better maintainability.
package zltech

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/canbus/canopen"
)

// -------- Object dictionary (excerpt; full coverage of manuals) --------------

const (
	idxControlWord     = 0x6040 // U16
	idxStatusWord      = 0x6041 // U16
	idxModeOfOperation = 0x6060 // I8
	idxModeDisplay     = 0x6061 // I8
	idxTargetVelocity  = 0x60FF // I32 (sub‑idx 1 L, 2 R)
	idxTargetPosition  = 0x607A // I32 (sub‑idx 1 L, 2 R)
	idxTargetTorque    = 0x6071 // I32 (sub‑idx 3 combined)
	idxProfileVelocity = 0x6081 // U32
	idxProfileAcc      = 0x6083 // U32
	idxProfileDec      = 0x6084 // U32
	idxCobIDSYNC       = 0x1005 // U32
	idxFaultCode       = 0x603F // U16
	// Manufacturer-specific indices used for E-stop wiring/mode and saving params
	idxInputFuncSel        = 0x2030 // manufacturer specific
	idxIOEmergencyStopMode = 0x2026 // manufacturer specific
	idxSyncAsyncControl    = 0x200F // manufacturer specific (0: async, 1: sync)
	idxSaveToEEPROM        = 0x2010 // manufacturer specific
)

// Control‑word bits (CiA‑402).
const (
	cwShutdown       = 0x0006
	cwSwitchOn       = 0x0007
	cwEnableOp       = 0x000F
	cwDisableVoltage = 0x0000
	cwQuickStop      = 0x0002
	cwFaultReset     = 0x0080
)

// Modes of operation.
const (
	modeProfilePos = 1
	modeProfileVel = 3
	modeProfileTor = 4
)

// NMT commands.
const (
	nmtStart          = 0x01
	nmtPreOperational = 0x80
	nmtCobID          = 0x000
)

// zltech/indices.go
const (
	// Temperature
	idxMotorTemperature = 0x2032
	subTempLeft         = 0x01
	subTempRight        = 0x02
	subTempDriver       = 0x03

	// Speed
	idxVelocityActual = 0x606C
	subSpeedLeft      = 0x01
	subSpeedRight     = 0x02
	subSpeedCombo     = 0x03

	// Current / torque-actual
	idxTorqueActual = 0x6077
	subCurrentLeft  = 0x01
	subCurrentRight = 0x02
	subCurrentCombo = 0x03
)

// Side indicates which hub you want to read.
type Side int

const (
	Left Side = iota
	Right
)

// velocityTransport selects how target velocity and acceleration/deceleration
// are sent to the drive.
type velocityTransport int

const (
	velocityViaSDO velocityTransport = iota
	velocityViaRPDO
)

// -----------------------------------------------------------------------------
// Client
// -----------------------------------------------------------------------------

type Client struct {
	nodeID byte
	bus    canbus.Bus
	mux    *canbus.Mux
	// logger is the structured logger used by the client.
	logger *slog.Logger

	// SDO client via canopen
	sdo *canopen.SDOClient

	// SYNC production period
	syncPeriod time.Duration
	// syncCancel cancels the background SYNC producer goroutine.
	syncCancel context.CancelFunc
	// syncWG waits for the SYNC goroutine to exit.
	syncWG sync.WaitGroup

	// cache of last written accel/decel to avoid redundant writes
	lastAccMs uint32
	lastDecMs uint32

	// transport for velocity/accel commands (default SDO)
	velTransport velocityTransport
}

// Option applies configuration to Client during construction.
type Option func(*Client)

// WithLogger sets the logger used by the client.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithSyncPeriod sets the SYNC message period.
func WithSyncPeriod(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.syncPeriod = d
		}
	}
}

// WithVelocityViaRPDO configures the client to send velocity/accel via RPDOs.
func WithVelocityViaRPDO() Option {
	return func(c *Client) { c.velTransport = velocityViaRPDO }
}

// New returns a ready‑to‑use client; no network traffic is issued.
func New(bus canbus.Bus, nodeID byte, opts ...Option) *Client {
	mux := canbus.NewMux(bus)
	return NewWithMux(bus, mux, nodeID, opts...)
}

// NewWithMux allows sharing a single mux for multiple clients on the same bus.
func NewWithMux(bus canbus.Bus, mux *canbus.Mux, nodeID byte, opts ...Option) *Client {
	c := &Client{
		bus:          bus,
		mux:          mux,
		nodeID:       nodeID,
		logger:       slog.Default(),
		syncPeriod:   5 * time.Millisecond,
		velTransport: velocityViaSDO,
	}
	c.sdo = canopen.NewSDOClient(bus, canopen.NodeID(nodeID), mux,
		canopen.WithTimeout(1*time.Second),
		canopen.WithExpeditedMode(canopen.ExpeditedModeClassic),
		canopen.WithLenientUpload())
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Close stops background readers associated with this client.
// Callers remain responsible for closing the underlying bus.
func (c *Client) Close() error {
	c.logger.Info("closing client")

	// Stop SYNC producer first (non-blocking, just cancels goroutine)
	c.stopSYNC()

	if err := c.Halt(context.Background()); err != nil {
		// Log error but don't fail shutdown - if CAN is down, we can't communicate anyway
		c.logger.Warn("failed to halt motor during close", "error", err)
	}

	// No additional resources to close here; the underlying bus is owned by caller.
	return nil
}

// -----------------------------------------------------------------------------
// Public high‑level API
// -----------------------------------------------------------------------------

// Init sends the four‑step CiA‑402 transition to OPERATION ENABLE and starts
// default heart‑beat production at 1 s.
func (c *Client) Init(ctx context.Context) error {
	// Enter Pre-operational so SDOs are reliably accepted on all firmware
	if err := c.preOperational(); err != nil {
		return fmt.Errorf("init step -2 (nmt pre-op) failed: %w", err)
	}
	time.Sleep(20 * time.Millisecond)

	// Attempt to clear any pre-existing faults while in Pre-operational.
	// It's safe to call this even if there's no fault.
	if err := c.ClearFault(ctx); err != nil {
		return fmt.Errorf("init step 0 (clear fault) failed: %w", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow time for state transition

	// While still in Pre-operational, set ASYNC control per manual (0x200F = 0).
	// Some firmware only allows modifying manufacturer params in Pre-op.
	if err := c.setSyncMode(ctx, false); err != nil {
		return fmt.Errorf("failed to set async control (0x200F=0) in pre-op: %w", err)
	}

	// While in Pre-operational, configure RPDO mappings if requested.
	if c.velTransport == velocityViaRPDO {
		if err := c.configureRPDOForVelocityAndAccel(ctx); err != nil {
			return fmt.Errorf("configure RPDO failed: %w", err)
		}
	}

	// Transition to Operational for CiA-402 controlword sequence
	if err := c.startNode(); err != nil {
		return fmt.Errorf("init step -1 (nmt start) failed: %w", err)
	}
	time.Sleep(20 * time.Millisecond)

	// Step 1: Shutdown
	if err := c.writeControlWord(uint16(cwShutdown)); err != nil {
		return fmt.Errorf("init step 1 (shutdown) failed: %w", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow time for state transition

	// Step 2: Switch On
	if err := c.writeControlWord(uint16(cwSwitchOn)); err != nil {
		return fmt.Errorf("init step 2 (switch on) failed: %w", err)
	}
	time.Sleep(10 * time.Millisecond) // Allow time for state transition

	// Step 3: Enable Operation
	if err := c.Enable(ctx); err != nil {
		return err
	}
	// Ensure drive is in profile-velocity mode by default after enabling
	if err := c.setMode(ctx, modeProfileVel); err != nil {
		return fmt.Errorf("failed to set velocity mode: %w", err)
	}
	// Start producing SYNC messages in the background to keep the drive in lockstep.
	if err := c.startSYNC(ctx, c.syncPeriod); err != nil {
		return fmt.Errorf("failed to start SYNC producer: %w", err)
	}
	return nil
}

// Enable transitions the drive to OPERATION ENABLE.
func (c *Client) Enable(ctx context.Context) error {
	// Read current state to decide the minimal sequence needed.
	sw, err := c.StatusWord(ctx)
	if err != nil {
		return err
	}
	switch sw.State() {
	case StateOperationEnabled:
		return nil
	case StateSwitchedOn:
		return c.writeControlWord(uint16(cwEnableOp))
	case StateReadyToSwitchOn:
		if err := c.writeControlWord(uint16(cwSwitchOn)); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		return c.writeControlWord(uint16(cwEnableOp))
	case StateSwitchOnDisabled, StateNotReadyToSwitchOn:
		// Walk full sequence: Shutdown -> Switch On -> Enable Operation
		if err := c.writeControlWord(uint16(cwShutdown)); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		if err := c.writeControlWord(uint16(cwSwitchOn)); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		return c.writeControlWord(uint16(cwEnableOp))
	case StateQuickStopActive:
		// Clearing quick-stop by writing enable-op is sufficient.
		return c.writeControlWord(uint16(cwEnableOp))
	case StateFault:
		return c.RecoverFromFault(ctx)
	default:
		// Fallback to full sequence.
		if err := c.sdo.WriteU16(idxControlWord, 0, uint16(cwShutdown)); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		if err := c.sdo.WriteU16(idxControlWord, 0, uint16(cwSwitchOn)); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
		return c.sdo.WriteU16(idxControlWord, 0, uint16(cwEnableOp))
	}
}

// Disable disables the drive power stage (voltage disabled) so the wheels
// are not driven and can freewheel.
func (c *Client) Disable(ctx context.Context) error {
	return c.writeControlWord(uint16(cwDisableVoltage))
}

// Halt issues a quick‑stop (motor brakes, drive remains enabled).
func (c *Client) Halt(ctx context.Context) error {
	return c.writeControlWord(uint16(cwQuickStop))
}

// ClearFault resets the CiA‑402 state machine from FAULT to SWITCH ON DISABLED.
func (c *Client) ClearFault(ctx context.Context) error {
	return c.sdo.WriteU16(idxControlWord, 0, uint16(cwFaultReset))
}

// writeControlWord writes the CiA-402 controlword (0x6040) as a 16-bit value.
func (c *Client) writeControlWord(value uint16) error {
	return c.sdo.WriteU16(idxControlWord, 0, value)
}

// RecoverFromFault clears a fault and walks the CiA‑402 sequence back to
// OPERATION ENABLE so the drive can run again.
func (c *Client) RecoverFromFault(ctx context.Context) error {
	if err := c.ClearFault(ctx); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	if err := c.writeControlWord(uint16(cwShutdown)); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	if err := c.writeControlWord(uint16(cwSwitchOn)); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)

	return c.Enable(ctx)
}

// StartNode sends the CANopen NMT Start command (ID 0x000) for this node.
func (c *Client) startNode() error {
	n := canopen.NMT{Command: canopen.NMTStart, Node: c.nodeID}
	f, _ := n.MarshalCANFrame()
	c.logFrame("tx", f)
	return c.bus.Send(f)
}

// PreOperational sends the CANopen NMT Pre-operational command (ID 0x000).
func (c *Client) preOperational() error {
	n := canopen.NMT{Command: canopen.NMTEnterPreOperational, Node: c.nodeID}
	f, _ := n.MarshalCANFrame()
	c.logFrame("tx", f)
	return c.bus.Send(f)
}

// -----------------------------------------------------------------------------
// Profile‑velocity helpers
// -----------------------------------------------------------------------------

func (c *Client) SetProfileVelocity(ctx context.Context, rpmL, rpmR int32,
	accMs, decMs uint32) error {

	if c.velTransport == velocityViaRPDO {
		return c.setProfileVelocityRPDO(ctx, rpmL, rpmR, accMs, decMs)
	}
	return c.setProfileVelocitySDO(ctx, rpmL, rpmR, accMs, decMs)
}

// setProfileVelocitySDO writes acc/dec and velocities via SDOs, caching acc/dec.
func (c *Client) setProfileVelocitySDO(ctx context.Context, rpmL, rpmR int32, accMs, decMs uint32) error {
	if c.lastAccMs != accMs {
		if err := c.sdo.WriteU32(idxProfileAcc, 1, uint32(accMs)); err != nil {
			return err
		}
		if err := c.sdo.WriteU32(idxProfileAcc, 2, uint32(accMs)); err != nil {
			return err
		}
		c.lastAccMs = accMs
	}
	if c.lastDecMs != decMs {
		if err := c.sdo.WriteU32(idxProfileDec, 1, uint32(decMs)); err != nil {
			return err
		}
		if err := c.sdo.WriteU32(idxProfileDec, 2, uint32(decMs)); err != nil {
			return err
		}
		c.lastDecMs = decMs
	}
	if err := c.sdo.WriteU32(idxTargetVelocity, 1, uint32(rpmL)); err != nil {
		return err
	}
	return c.sdo.WriteU32(idxTargetVelocity, 2, uint32(rpmR))
}

// setProfileVelocityRPDO sends velocities via RPDO1, and acc/dec via RPDO2/3 when changed.
func (c *Client) setProfileVelocityRPDO(ctx context.Context, rpmL, rpmR int32, accMs, decMs uint32) error {
	// RPDO2: acceleration (left/right)
	if c.lastAccMs != accMs {
		var f canbus.Frame
		f.ID = canopen.COBID(canopen.FC_RPDO2, canopen.NodeID(c.nodeID))
		f.Len = 8
		binary.LittleEndian.PutUint32(f.Data[0:4], uint32(accMs))
		binary.LittleEndian.PutUint32(f.Data[4:8], uint32(accMs))
		c.logFrame("tx", f)
		if err := c.bus.Send(f); err != nil {
			return err
		}
		c.lastAccMs = accMs
	}

	// RPDO3: deceleration (left/right)
	if c.lastDecMs != decMs {
		var f canbus.Frame
		f.ID = canopen.COBID(canopen.FC_RPDO3, canopen.NodeID(c.nodeID))
		f.Len = 8
		binary.LittleEndian.PutUint32(f.Data[0:4], uint32(decMs))
		binary.LittleEndian.PutUint32(f.Data[4:8], uint32(decMs))
		c.logFrame("tx", f)
		if err := c.bus.Send(f); err != nil {
			return err
		}
		c.lastDecMs = decMs
	}

	// RPDO1: target velocities (left/right)
	{
		var f canbus.Frame
		f.ID = canopen.COBID(canopen.FC_RPDO1, canopen.NodeID(c.nodeID))
		f.Len = 8
		binary.LittleEndian.PutUint32(f.Data[0:4], uint32(rpmL))
		binary.LittleEndian.PutUint32(f.Data[4:8], uint32(rpmR))
		c.logFrame("tx", f)
		return c.bus.Send(f)
	}
}

// configureRPDOForVelocityAndAccel maps RPDO1/2/3 for velocity/accel/decel.
// Transmission type = 0xFF (asynchronous)
func (c *Client) configureRPDOForVelocityAndAccel(ctx context.Context) error {
	// RPDO1 (0x1400/0x1600) velocities: 0x60FF:1, 0x60FF:2
	if err := c.sdo.WriteU8(0x1600, 0, 0); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1600, 1, uint32(0x60FF0120)); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1600, 2, uint32(0x60FF0220)); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1400, 2, 0xFF); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1400, 1, uint32(0x200+uint32(c.nodeID))); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1600, 0, 2); err != nil {
		return err
	}

	// RPDO2 (0x1401/0x1601) acceleration: 0x6083:1, 0x6083:2
	if err := c.sdo.WriteU8(0x1601, 0, 0); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1601, 1, uint32(0x60830120)); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1601, 2, uint32(0x60830220)); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1401, 2, 0xFF); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1401, 1, uint32(0x300+uint32(c.nodeID))); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1601, 0, 2); err != nil {
		return err
	}

	// RPDO3 (0x1402/0x1602) deceleration: 0x6084:1, 0x6084:2
	if err := c.sdo.WriteU8(0x1602, 0, 0); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1602, 1, uint32(0x60840120)); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1602, 2, uint32(0x60840220)); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1402, 2, 0xFF); err != nil {
		return err
	}
	if err := c.sdo.WriteU32(0x1402, 1, uint32(0x400+uint32(c.nodeID))); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1602, 0, 2); err != nil {
		return err
	}

	return nil
}

// -----------------------------------------------------------------------------
// Diagnostics
// -----------------------------------------------------------------------------

// FaultCodes reads the combined 32-bit fault code (0x603F) and splits it into
// left and right 16-bit fault words.
// Note: Despite the manual suggesting high=left and low=right, hardware tests
// show the opposite on our units. We therefore map low=left and high=right.
func (c *Client) FaultCodes(ctx context.Context) (FaultCode, FaultCode, error) {
	v, err := c.sdo.ReadU32(idxFaultCode, 0)
	if err != nil {
		return 0, 0, err
	}
	u := uint32(v)
	left := FaultCode(uint16(u & 0xFFFF))
	right := FaultCode(uint16((u >> 16) & 0xFFFF))
	return left, right, nil
}

// StatusWord reads the current status word from the drive
func (c *Client) StatusWord(ctx context.Context) (StatusWord, error) {
	// Some firmware responds with 4 bytes for 16-bit objects. Try U32 first.
	v32, err := c.sdo.ReadU32(idxStatusWord, 0)
	if err == nil {
		return StatusWord(uint16(v32 & 0xFFFF)), nil
	}
	val, err2 := c.sdo.ReadU16(idxStatusWord, 0)
	if err2 != nil {
		return 0, err
	}
	return StatusWord(val), nil
}

// -----------------------------------------------------------------------------
// E-stop and parameter utilities (manufacturer specific)
// -----------------------------------------------------------------------------

// SetEmergencyStopMode configures I/O emergency stop behavior per 0x2026/3
// (0 = lock shaft, 1 = release shaft).
func (c *Client) SetEmergencyStopMode(ctx context.Context, release bool) error {
	var val byte = 0
	if release {
		val = 1
	}
	return c.sdo.WriteU8(idxIOEmergencyStopMode, 3, val)
}

// Heartbeat-related code removed.

// -----------------------------------------------------------------------------
// SYNC producer (optional master keep-alive for synchronous operation)
// -----------------------------------------------------------------------------

// SetSyncMode sets synchronous (true) or asynchronous (false) control (0x200F).
func (c *Client) setSyncMode(ctx context.Context, syncEnabled bool) error {
	if syncEnabled {
		return c.sdo.WriteU16(idxSyncAsyncControl, 0, 1)
	}
	return c.sdo.WriteU16(idxSyncAsyncControl, 0, 0)
}

// ReadCobIDSync reads COB-ID for SYNC (0x1005). Default is 0x80.
func (c *Client) readCobIDSync(ctx context.Context) (uint32, error) {
	v, err := c.sdo.ReadU32(idxCobIDSYNC, 0)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// StartSYNC starts sending SYNC frames at the given period until ctx is canceled.
// This is useful if the controller operates in synchronous mode and expects
// periodic SYNC; many devices will halt motion if SYNCs stop.
func (c *Client) startSYNC(ctx context.Context, period time.Duration) error {
	// Stop any existing SYNC producer first to avoid duplicates.
	c.stopSYNC()

	// Read COB-ID for SYNC; default to 0x80 on error or zero.
	cobID, err := c.readCobIDSync(ctx)
	if err != nil || cobID == 0 {
		cobID = 0x80
	}

	// Create internal cancellable context independent of the caller's context lifetime.
	syncCtx, cancel := context.WithCancel(context.Background())
	c.syncCancel = cancel

	t := time.NewTicker(period)
	c.syncWG.Add(1)
	go func() {
		defer t.Stop()
		defer c.syncWG.Done()
		for {
			select {
			case <-syncCtx.Done():
				return
			case <-t.C:
				f := canbus.Frame{ID: cobID, Len: 0}
				_ = c.bus.Send(f)
			}
		}
	}()
	return nil
}

// stopSYNC cancels and joins the background SYNC producer if running.
func (c *Client) stopSYNC() {
	if c.syncCancel != nil {
		c.syncCancel()
		c.syncCancel = nil
		c.syncWG.Wait()
	}
}

// zltech/client_readings.go
// Temperature returns motor or driver temperature in °C.
func (c *Client) Temperature(ctx context.Context, side Side) (float64, error) {
	sub := subTempDriver
	switch side {
	case Left:
		sub = subTempLeft
	case Right:
		sub = subTempRight
	}
	raw, err := c.sdo.ReadU16(idxMotorTemperature, uint8(sub))
	if err != nil {
		return 0, err
	}
	return float64(int16(raw)) / 10.0, nil // signed-extend if needed
}

// ControllerTemperature returns the driver/controller internal temperature in °C.
// This uses the manufacturer-specific sub-index for the controller sensor.
func (c *Client) ControllerTemperature(ctx context.Context) (float64, error) {
	raw, err := c.sdo.ReadU16(idxMotorTemperature, uint8(subTempDriver))
	if err != nil {
		return 0, err
	}
	return float64(int16(raw)) / 10.0, nil
}

// Current returns instantaneous phase current in amperes.
func (c *Client) Current(ctx context.Context, side Side) (float64, error) {
	sub := subCurrentLeft
	if side == Right {
		sub = subCurrentRight
	}
	raw, err := c.sdo.ReadU16(idxTorqueActual, uint8(sub))
	if err != nil {
		return 0, err
	}
	return float64(int16(raw)) / 10.0, nil
}

// Speed returns shaft speed in r/min.
func (c *Client) Speed(ctx context.Context, side Side) (float64, error) {
	sub := subSpeedLeft
	if side == Right {
		sub = subSpeedRight
	}
	raw, err := c.sdo.ReadU32(idxVelocityActual, uint8(sub))
	if err != nil {
		return 0, err
	}
	return float64(int32(raw)) / 10.0, nil
}

// -----------------------------------------------------------------------------
// internals
// -----------------------------------------------------------------------------

func (c *Client) setMode(ctx context.Context, mode byte) error {
	if err := c.sdo.WriteU8(idxModeOfOperation, 0, mode); err != nil {
		return fmt.Errorf("failed to write mode of operation: %w", err)
	}

	// Add a small delay to allow the drive to process the mode change
	time.Sleep(10 * time.Millisecond)
	return nil
}

// waitUntilReadyForControl polls the StatusWord until the device reaches a
// state where CiA-402 controlword writes are accepted, or until timeout elapses.
func (c *Client) waitUntilReadyForControl(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		sw, err := c.StatusWord(ctx)
		if err == nil {
			s := sw.State()
			// If in fault, try to clear and continue.
			if s == StateFault {
				_ = c.ClearFault(ctx)
				time.Sleep(20 * time.Millisecond)
				continue
			}
			// Accept states that allow controlword writes on this device.
			if s == StateSwitchOnDisabled || s == StateSwitchedOn || s == StateReadyToSwitchOn || s == StateNotReadyToSwitchOn {
				return nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("device not ready for controlword writes within %v", timeout)
}

// logFrame prints a CAN frame with id and data bytes in hex using slog.
func (c *Client) logFrame(direction string, f canbus.Frame) {
	l := c.logger
	if l == nil {
		l = slog.Default()
	}
	// Format bytes: "AA BB CC ..."
	var b strings.Builder
	for i := 0; i < int(f.Len) && i < len(f.Data); i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%02X", f.Data[i])
	}
	l.Debug("can",
		"dir", direction,
		"id", fmt.Sprintf("0x%03X", f.ID),
		"len", f.Len,
		"data", b.String(),
	)
}
