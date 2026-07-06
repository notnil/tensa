// fault.go  – diagnostics helpers for ZLAC8015D
package zltech

import (
	"fmt"
	"strings"
)

// FaultCode represents a 16-bit per-motor fault word.
// For our units, the combined 0x603F 32-bit value maps as:
// low-word = left motor, high-word = right motor.
type FaultCode uint16

const (
	FaultNone                   FaultCode = 0x0000
	FaultOverVoltage            FaultCode = 0x0001
	FaultUnderVoltage           FaultCode = 0x0002
	FaultOverCurrent            FaultCode = 0x0004
	FaultOverload               FaultCode = 0x0008
	FaultCurrentOutOfTolerance  FaultCode = 0x0010
	FaultEncoderOutOfTolerance  FaultCode = 0x0020
	FaultVelocityOutOfTolerance FaultCode = 0x0040
	FaultReferenceVoltageError  FaultCode = 0x0080
	FaultHallError              FaultCode = 0x0200
	FaultHighMotorTemperature   FaultCode = 0x0400
	FaultEncoderError           FaultCode = 0x0800
	FaultEEPROMReadWriteError   FaultCode = 0x0100
	FaultSpeedSettingError      FaultCode = 0x2000
)

// String returns a human-readable list of all active fault bits.
// The drive may set multiple bits at once; we enumerate known flags
// rather than matching a single value. Unknown bits are reported in hex.
func (f FaultCode) String() string {
	if f == FaultNone {
		return "no error"
	}
	var parts []string
	known := FaultCode(0)
	add := func(bit FaultCode, label string) {
		if f&bit != 0 {
			parts = append(parts, label)
			known |= bit
		}
	}
	add(FaultOverVoltage, "over-voltage")
	add(FaultUnderVoltage, "under-voltage")
	add(FaultOverCurrent, "over-current")
	add(FaultOverload, "over-load")
	add(FaultCurrentOutOfTolerance, "current out-of-tolerance")
	add(FaultEncoderOutOfTolerance, "encoder out-of-tolerance")
	add(FaultVelocityOutOfTolerance, "velocity out-of-tolerance")
	add(FaultReferenceVoltageError, "reference-voltage error")
	add(FaultHallError, "hall error")
	add(FaultHighMotorTemperature, "motor over-temperature")
	add(FaultEncoderError, "encoder error")
	add(FaultEEPROMReadWriteError, "EEPROM R/W error")
	add(FaultSpeedSettingError, "speed-setting error")
	if unknown := f &^ known; unknown != 0 {
		parts = append(parts, fmt.Sprintf("unknown(0x%04X)", uint16(unknown)))
	}
	return strings.Join(parts, ", ")
}

// -----------------------------------------------------------------------------
// Status-word helpers (object 0x6041)
// -----------------------------------------------------------------------------

// StatusWord is the raw 16-bit CiA-402 status register.
type StatusWord uint16

// Selected status bits per CiA-402. We keep commonly used ones explicit.
const (
	swBitReadyToSwitchOn   StatusWord = 1 << 0
	swBitSwitchedOn        StatusWord = 1 << 1
	swBitOperationEnabled  StatusWord = 1 << 2
	swBitFault             StatusWord = 1 << 3
	swBitVoltageEnabled    StatusWord = 1 << 4
	swBitQuickStopActive   StatusWord = 1 << 5
	swBitSwitchOnDisabled  StatusWord = 1 << 6
	swBitWarning           StatusWord = 1 << 7
	StatusBitTargetReached StatusWord = 1 << 10
)

// OpState enumerates the four patterns that appear in bits 0-3.
type OpState uint8

const (
	StateNotReadyToSwitchOn OpState = iota
	StateSwitchOnDisabled
	StateReadyToSwitchOn
	StateSwitchedOn
	StateOperationEnabled
	StateQuickStopActive
	StateFault
)

// State derives the CiA‑402 state using the standard statusword bit patterns.
// We use the canonical mask 0x006F and compare against documented patterns,
// with pragmatic fallbacks for vendor quirks (e.g. 0x0060 seen as switch‑on disabled).
func (sw StatusWord) State() OpState {
	// Canonical fast‑path: use masked pattern compare
	masked := uint16(sw) & 0x006F
	switch masked {
	case 0x0000:
		return StateNotReadyToSwitchOn
	case 0x0040, 0x0060: // some firmwares set both bits 5 and 6
		return StateSwitchOnDisabled
	case 0x0021:
		return StateReadyToSwitchOn
	case 0x0023:
		return StateSwitchedOn
	case 0x0027:
		return StateOperationEnabled
	case 0x0007:
		return StateQuickStopActive
	case 0x0008, 0x000F: // fault or fault reaction active
		return StateFault
	}

	// Fallbacks: infer most specific state from individual bits
	if sw&swBitFault != 0 {
		return StateFault
	}
	if sw&swBitOperationEnabled != 0 {
		return StateOperationEnabled
	}
	if sw&swBitSwitchedOn != 0 {
		return StateSwitchedOn
	}
	if sw&swBitReadyToSwitchOn != 0 {
		return StateReadyToSwitchOn
	}
	if sw&swBitSwitchOnDisabled != 0 {
		return StateSwitchOnDisabled
	}
	if sw&swBitQuickStopActive != 0 {
		return StateQuickStopActive
	}
	return StateNotReadyToSwitchOn
}

// String renders the state in plain English (handy for logs).
func (s OpState) String() string {
	switch s {
	case StateNotReadyToSwitchOn:
		return "not ready"
	case StateReadyToSwitchOn:
		return "ready to switch on"
	case StateSwitchOnDisabled:
		return "switch-on disabled"
	case StateSwitchedOn:
		return "switched on"
	case StateOperationEnabled:
		return "operation enabled"
	case StateQuickStopActive:
		return "quick-stop active"
	case StateFault:
		return "fault"
	default:
		return "unknown"
	}
}

// String renders a concise human-readable representation of the status word,
// including the decoded CiA-402 state and notable flags when set.
func (sw StatusWord) String() string {
	var flags []string
	if sw&swBitVoltageEnabled != 0 {
		flags = append(flags, "voltage-enabled")
	}
	if sw&swBitWarning != 0 {
		flags = append(flags, "warning")
	}
	if sw&StatusBitTargetReached != 0 {
		flags = append(flags, "target-reached")
	}
	if len(flags) == 0 {
		flags = append(flags, "no-flags")
	}
	return fmt.Sprintf("%s (0x%04X) [%s]", sw.State(), uint16(sw), strings.Join(flags, ", "))
}
