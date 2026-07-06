package roboct

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/canbus/canopen"
)

// Object Dictionary Indices
const (
	idxControlWord    = 0x6040
	idxStatusWord     = 0x6041
	idxModeOfOp       = 0x6060
	idxModeDisplay    = 0x6061
	idxTargetVelocity = 0x60FF
	idxProfileAccel   = 0x6082 // T-curve acceleration
	idxProfileDecel   = 0x6083 // T-curve deceleration
	idxVelocityActual = 0x606C
	idxCurrentActual  = 0x6078
	idxDCBusVoltage   = 0x6079
	idxIPMTemp        = 0x6091
	idxFaultCode      = 0x200B
)

// CiA 402 Control Word Bits
const (
	cwShutdown       = 0x0006
	cwSwitchOn       = 0x0007
	cwEnableOp       = 0x000F
	cwDisableVoltage = 0x0000
	cwQuickStop      = 0x0002
	cwFaultReset     = 0x0080
)

// Modes of Operation
const (
	modeProfileVelocity = 3
)

// Client represents a single RoboCT drive node.
type Client struct {
	nodeID byte
	bus    canbus.Bus
	sdo    *canopen.SDOClient
	logger *slog.Logger

	// Cache last written accel/decel to avoid redundant writes
	lastAcc uint32
	lastDec uint32
}

// Option applies configuration to Client.
type Option func(*Client)

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// New creates a new Client for a RoboCT drive.
func New(bus canbus.Bus, nodeID byte, opts ...Option) *Client {
	mux := canbus.NewMux(bus)
	return NewWithMux(bus, mux, nodeID, opts...)
}

// NewWithMux creates a new Client for a RoboCT drive sharing a CAN mux.
func NewWithMux(bus canbus.Bus, mux *canbus.Mux, nodeID byte, opts ...Option) *Client {
	c := &Client{
		nodeID: nodeID,
		bus:    bus,
		logger: slog.Default(),
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

// Init prepares the drive for operation.
func (c *Client) Init(ctx context.Context) error {
	// 1. NMT Reset Node (Optional, but good practice to ensure clean state)
	// For now, just Start Node to ensure we are in Operational or Pre-Operational.
	if err := c.startNode(); err != nil {
		return fmt.Errorf("nmt start failed: %w", err)
	}
	time.Sleep(50 * time.Millisecond)

	// 2. Clear any existing faults
	if err := c.ClearFault(ctx); err != nil {
		c.logger.Warn("failed to clear fault during init", "err", err)
	}

	// 3. Set Mode to Profile Velocity (INT16 on this device)
	if err := c.sdo.WriteU16(idxModeOfOp, 0, modeProfileVelocity); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	// 4. Configure RPDOs for Velocity Control
	if err := c.configureRPDOs(ctx); err != nil {
		return fmt.Errorf("configure RPDOs failed: %w", err)
	}

	// 5. Transition to Enable Operation
	return c.Enable(ctx)
}

// Enable transitions the drive to Operation Enabled state.
func (c *Client) Enable(ctx context.Context) error {
	// Sequence: Shutdown -> SwitchOn -> EnableOp
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

// Disable disables the drive output.
func (c *Client) Disable(ctx context.Context) error {
	return c.writeControlWord(cwDisableVoltage)
}

// Halt performs a Quick Stop.
func (c *Client) Halt(ctx context.Context) error {
	return c.writeControlWord(cwQuickStop)
}

// SetVelocity sets the target velocity in RPM.
func (c *Client) SetVelocity(ctx context.Context, rpm int16) error {
	// RPDO1: Target Velocity (0x60FF) - Always update
	var f canbus.Frame
	// RPDO1 COB-ID = 0x200 + NodeID
	f.ID = uint32(0x200) + uint32(c.nodeID)
	f.Len = 2                                               // 1 x INT16 = 2 bytes
	binary.LittleEndian.PutUint16(f.Data[0:2], uint16(rpm)) // Cast int16 to uint16 for bytes

	c.logFrame("tx-rpdo1", f)
	if err := c.bus.Send(f); err != nil {
		return fmt.Errorf("send rpdo1 failed: %w", err)
	}

	return nil
}

// SetAcceleration sets the profile acceleration and deceleration.
// acc and dec are in RPS^2 units (1..1000).
func (c *Client) SetAcceleration(ctx context.Context, acc, dec uint16) error {
	// RPDO2: Accel (0x6082) and Decel (0x6083) - Optional update if changed
	if uint32(acc) != c.lastAcc || uint32(dec) != c.lastDec {
		var f canbus.Frame
		// RPDO2 COB-ID = 0x300 + NodeID
		f.ID = uint32(0x300) + uint32(c.nodeID)
		f.Len = 4 // 2 x UINT16 = 4 bytes
		binary.LittleEndian.PutUint16(f.Data[0:2], acc)
		binary.LittleEndian.PutUint16(f.Data[2:4], dec)

		c.logFrame("tx-rpdo2", f)
		if err := c.bus.Send(f); err != nil {
			return fmt.Errorf("send rpdo2 failed: %w", err)
		}

		c.lastAcc = uint32(acc)
		c.lastDec = uint32(dec)
	}
	return nil
}

// configureRPDOs maps RPDO1 for Velocity and RPDO2 for Accel/Decel.
func (c *Client) configureRPDOs(ctx context.Context) error {
	// RPDO1: Target Velocity (0x60FF) - 16-bit
	// Mapping 0x1600
	if err := c.sdo.WriteU8(0x1600, 0, 0); err != nil { // Disable mapping
		return err
	}
	// Map 0x60FF sub 0 len 16 (0x10) -> 0x60FF0010
	if err := c.sdo.WriteU32(0x1600, 1, 0x60FF0010); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1600, 0, 1); err != nil { // Enable 1 entry
		return err
	}

	// RPDO1 Params 0x1400
	// Transmission Type 254 (Async/Event)
	if err := c.sdo.WriteU8(0x1400, 2, 254); err != nil {
		return err
	}
	// COB-ID: Ensure valid bit (31) is 0. Default is usually 0x200+ID.
	// Just to be safe, we can write it, but usually default is fine.
	// Let's write it to be sure and enable it.
	rpdo1CobID := uint32(0x200) + uint32(c.nodeID)
	if err := c.sdo.WriteU32(0x1400, 1, rpdo1CobID); err != nil {
		return err
	}

	// RPDO2: Accel (0x6082) and Decel (0x6083) - both 16-bit
	// Mapping 0x1601
	if err := c.sdo.WriteU8(0x1601, 0, 0); err != nil { // Disable mapping
		return err
	}
	// Map 0x6082 sub 0 len 16 -> 0x60820010
	if err := c.sdo.WriteU32(0x1601, 1, 0x60820010); err != nil {
		return err
	}
	// Map 0x6083 sub 0 len 16 -> 0x60830010
	if err := c.sdo.WriteU32(0x1601, 2, 0x60830010); err != nil {
		return err
	}
	if err := c.sdo.WriteU8(0x1601, 0, 2); err != nil { // Enable 2 entries
		return err
	}

	// RPDO2 Params 0x1401
	// Transmission Type 254
	if err := c.sdo.WriteU8(0x1401, 2, 254); err != nil {
		return err
	}
	rpdo2CobID := uint32(0x300) + uint32(c.nodeID)
	if err := c.sdo.WriteU32(0x1401, 1, rpdo2CobID); err != nil {
		return err
	}

	return nil
}

// ClearFault tries to clear faults.
func (c *Client) ClearFault(ctx context.Context) error {
	return c.writeControlWord(cwFaultReset)
}

// StatusData holds common feedback values.
type StatusData struct {
	RPM         int16
	Current     int16   // mA
	Voltage     float64 // V
	Temperature int16   // Celsius
	FaultCode   uint32
	StatusWord  uint16
}

// Status reads the current drive status.
func (c *Client) Status(ctx context.Context) (StatusData, error) {
	var s StatusData

	// Read StatusWord
	sw, err := c.sdo.ReadU16(idxStatusWord, 0)
	if err != nil {
		return s, fmt.Errorf("read status word: %w", err)
	}
	s.StatusWord = sw

	// Read Actual Velocity
	// Manual 0x606C is INT16
	vel, err := c.sdo.ReadU16(idxVelocityActual, 0)
	if err != nil {
		return s, fmt.Errorf("read velocity: %w", err)
	}
	s.RPM = int16(vel)

	// Read Actual Current
	// Manual 0x6078 is INT16 (mA)
	curr, err := c.sdo.ReadU16(idxCurrentActual, 0)
	if err != nil {
		return s, fmt.Errorf("read current: %w", err)
	}
	s.Current = int16(curr)

	// Read Voltage
	// Manual 0x6079 is UINT16 (Volts)
	v, err := c.sdo.ReadU16(idxDCBusVoltage, 0)
	if err != nil {
		// Not critical
		c.logger.Debug("read voltage failed", "err", err)
	} else {
		s.Voltage = float64(v)
	}

	// Read Temp
	// Manual 0x6091 is UINT16
	temp, err := c.sdo.ReadU16(idxIPMTemp, 0)
	if err != nil {
		c.logger.Debug("read temp failed", "err", err)
	} else {
		s.Temperature = int16(temp)
	}

	// Read Fault Code (0x200B)
	fc, err := c.sdo.ReadU32(idxFaultCode, 0)
	if err != nil {
		c.logger.Debug("read fault code failed", "err", err)
	} else {
		s.FaultCode = uint32(fc)
	}

	return s, nil
}

func (c *Client) writeControlWord(val uint16) error {
	return c.sdo.WriteU16(idxControlWord, 0, val)
}

func (c *Client) startNode() error {
	n := canopen.NMT{Command: canopen.NMTStart, Node: c.nodeID}
	f, _ := n.MarshalCANFrame()
	c.logFrame("tx", f)
	return c.bus.Send(f)
}

func (c *Client) logFrame(direction string, f canbus.Frame) {
	var b strings.Builder
	for i := 0; i < int(f.Len) && i < len(f.Data); i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%02X", f.Data[i])
	}
	c.logger.Debug("can",
		"dir", direction,
		"id", fmt.Sprintf("0x%03X", f.ID),
		"len", f.Len,
		"data", b.String(),
	)
}
