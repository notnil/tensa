package zltech

import "fmt"

// State represents the CiA402 state machine state
type State int

const (
	StateNotReadyToSwitchOn State = iota
	StateSwitchOnDisabled
	StateReadyToSwitchOn
	StateSwitchedOn
	StateOperationEnabled
	StateQuickStopActive
	StateFaultReactionActive
	StateFault
)

func (s State) String() string {
	switch s {
	case StateNotReadyToSwitchOn:
		return "NotReadyToSwitchOn"
	case StateSwitchOnDisabled:
		return "SwitchOnDisabled"
	case StateReadyToSwitchOn:
		return "ReadyToSwitchOn"
	case StateSwitchedOn:
		return "SwitchedOn"
	case StateOperationEnabled:
		return "OperationEnabled"
	case StateQuickStopActive:
		return "QuickStopActive"
	case StateFaultReactionActive:
		return "FaultReactionActive"
	case StateFault:
		return "Fault"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// StatusWord is the CiA402 status word (0x6041)
type StatusWord uint16

// State extracts the CiA402 state from the status word
func (sw StatusWord) State() State {
	// CiA402 state machine bit patterns
	// See DS402 specification for bit definitions
	w := uint16(sw)

	// Check specific bit patterns to determine state
	if w&0x004F == 0x0000 {
		return StateNotReadyToSwitchOn
	}
	if w&0x004F == 0x0040 {
		return StateSwitchOnDisabled
	}
	if w&0x006F == 0x0021 {
		return StateReadyToSwitchOn
	}
	if w&0x006F == 0x0023 {
		return StateSwitchedOn
	}
	if w&0x006F == 0x0027 {
		return StateOperationEnabled
	}
	if w&0x006F == 0x0007 {
		return StateQuickStopActive
	}
	if w&0x004F == 0x000F {
		return StateFaultReactionActive
	}
	if w&0x004F == 0x0008 {
		return StateFault
	}

	return StateNotReadyToSwitchOn
}

// Ready returns true if the drive is ready to accept motion commands
func (sw StatusWord) Ready() bool {
	return sw.State() == StateOperationEnabled
}

// Fault returns true if the drive is in a fault state
func (sw StatusWord) Fault() bool {
	state := sw.State()
	return state == StateFault || state == StateFaultReactionActive
}

// FaultCode represents a ZLAC8015D fault/error code
type FaultCode uint16

const (
	FaultNone              FaultCode = 0x0000
	FaultOverVoltage       FaultCode = 0x0001
	FaultUnderVoltage      FaultCode = 0x0002
	FaultOverCurrent       FaultCode = 0x0004
	FaultOverLoad          FaultCode = 0x0008
	FaultPositionError     FaultCode = 0x0020
	FaultParameterError    FaultCode = 0x0100
	FaultHallFault         FaultCode = 0x0200
	FaultHighTemperature   FaultCode = 0x0400
	FaultEncoderError      FaultCode = 0x0800
	FaultSpeedSettingError FaultCode = 0x2000
)

func (fc FaultCode) String() string {
	if fc == FaultNone {
		return "NoFault"
	}

	var faults []string
	if fc&FaultOverVoltage != 0 {
		faults = append(faults, "OverVoltage")
	}
	if fc&FaultUnderVoltage != 0 {
		faults = append(faults, "UnderVoltage")
	}
	if fc&FaultOverCurrent != 0 {
		faults = append(faults, "OverCurrent")
	}
	if fc&FaultOverLoad != 0 {
		faults = append(faults, "OverLoad")
	}
	if fc&FaultPositionError != 0 {
		faults = append(faults, "PositionError")
	}
	if fc&FaultParameterError != 0 {
		faults = append(faults, "ParameterError")
	}
	if fc&FaultHallFault != 0 {
		faults = append(faults, "HallFault")
	}
	if fc&FaultHighTemperature != 0 {
		faults = append(faults, "HighTemperature")
	}
	if fc&FaultEncoderError != 0 {
		faults = append(faults, "EncoderError")
	}
	if fc&FaultSpeedSettingError != 0 {
		faults = append(faults, "SpeedSettingError")
	}

	if len(faults) == 0 {
		return fmt.Sprintf("UnknownFault(0x%04X)", uint16(fc))
	}

	result := faults[0]
	for i := 1; i < len(faults); i++ {
		result += "|" + faults[i]
	}
	return result
}

// HasFault returns true if any fault is present
func (fc FaultCode) HasFault() bool {
	return fc != FaultNone
}

// Status contains current status and diagnostic information for a wheel
type Status struct {
	State       State      // CiA402 state machine state
	ActualRPM   float64    // Actual motor speed in RPM
	Temperature float64    // Motor temperature in °C
	Current     float64    // Motor current in Amperes
	FaultCode   FaultCode  // Current fault code
	StatusWord  StatusWord // Raw status word
}

// String returns a human-readable status summary
func (s *Status) String() string {
	return fmt.Sprintf("State=%s RPM=%.1f Temp=%.1f°C Current=%.2fA Fault=%s",
		s.State, s.ActualRPM, s.Temperature, s.Current, s.FaultCode)
}

// Healthy returns true if the wheel is in a good operating state
func (s *Status) Healthy() bool {
	return !s.FaultCode.HasFault() && !s.StatusWord.Fault()
}
