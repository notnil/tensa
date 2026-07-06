/**
 * AngleMotor.cpp
 * 
 * Implementation of the AngleMotor class which controls
 * a stepper motor that adjusts the launch angle.
 */

#include "AngleMotor.h"
#include "MotorFault.h"
#include <math.h>

// Constructor
AngleMotor::AngleMotor(MotorDriver &motorConnector)
    : m_motorConnector(motorConnector), m_currentSteps(0), m_currentAngle(0.0f), m_alertCount(0), m_autoRecoveryEnabled(false), m_isHomed(false) {
}

// Initialize the angle motor
void AngleMotor::init() {
    // Set the motor connector to the correct mode
    // Note: The calling code must set MotorMgr.MotorModeSet to STEP_AND_DIR mode
    
    // Configure HLFB mode and carrier frequency
    // This tells our code how to interpret the HLFB signal from the ClearPath motor
    m_motorConnector.HlfbMode(MotorDriver::HLFB_MODE_HAS_BIPOLAR_PWM);
    m_motorConnector.HlfbCarrier(MotorDriver::HLFB_CARRIER_482_HZ);
    
    // Set velocity and acceleration limits
    // Convert from RPM to pulses per second:
    // 1. RPM / 60 = revolutions per second
    // 2. revolutions per second * PULSES_PER_REVOLUTION = pulses per second
    uint32_t velocityLimit = (MAX_VELOCITY * PULSES_PER_REVOLUTION) / 60;
    uint32_t accelerationLimit = (MAX_ACCELERATION * PULSES_PER_REVOLUTION) / 60;
    
    m_motorConnector.VelMax(velocityLimit);
    m_motorConnector.AccelMax(accelerationLimit);
    
    // Note: The motor is not enabled here. The homeMotor() function 
    // will enable the motor, which triggers the ClearPath's built-in
    // homing sequence.
    
    // Reset the homed state
    m_isHomed = false;
}

// Home the motor to establish a reference position
bool AngleMotor::homeMotor(unsigned long timeoutMs) {
    Serial.println("Starting homing sequence...");

    // Check for alerts and clear them if needed
    if (m_motorConnector.StatusReg().bit.AlertsPresent) {
        Serial.println("Alerts present before homing, attempting to clear...");
        if (HANDLE_MOTOR_FAULTS) {
            ClearMotorFaults(&m_motorConnector, "Angle Motor");
            // Give motor time to recover after clearing faults
            delay(100);
        }
    }

    // Make sure motor starts disabled
    m_motorConnector.EnableRequest(false);
    delay(100);

    Serial.println("Enabling motor to trigger homing...");
    m_motorConnector.EnableRequest(true);

    // Wait for HLFB to assert (indicating homing is complete)
    Serial.println("Waiting for HLFB to assert...");
    unsigned long startTime = millis();
    while (m_motorConnector.HlfbState() != MotorDriver::HLFB_ASSERTED) {
        // Check for timeout
        if (millis() - startTime > timeoutMs) {
            Serial.println("Homing timed out waiting for HLFB");
            m_isHomed = false;
            return false;
        }
        
        // Check for alerts during homing
        if (m_motorConnector.StatusReg().bit.AlertsPresent) {
            Serial.println("Alert detected during homing:");
            PrintAlerts(&m_motorConnector, "Angle Motor");
            m_isHomed = false;
            return false;
        }
        
        delay(1);
    }

    Serial.println("HLFB asserted, homing successful");
    // Reset position tracking to 0
    m_motorConnector.PositionRefSet(0);
    m_isHomed = true;
    return true;
}

// Move to an absolute angle position in radians
bool AngleMotor::moveToAngle(float angleInRadians) {
    if (!m_isHomed) {
        Serial.println("Cannot move: Motor not homed");
        return false;
    }

    int position = angleInRadians * MEASURED_STEPS_45_DEG / MAX_ANGLE;
    
    // Check if the position is within the valid range
    if (position < 0 || position > MEASURED_STEPS_45_DEG) {
        Serial.print("Position out of range: ");
        Serial.println(position);
        return false;
    }

    if (m_motorConnector.StatusReg().bit.AlertsPresent) {
        Serial.println("Cannot move: Alerts present");
        if (HANDLE_MOTOR_FAULTS) {
            ClearMotorFaults(&m_motorConnector, "Angle Motor");
            // Give motor time to recover after clearing faults
            delay(100);
        }
        return false;
    }

    Serial.print("Moving to position: ");
    Serial.println(position);
    m_motorConnector.Move(position, MotorDriver::MOVE_TARGET_ABSOLUTE);
    return true;
}

// Get current angle in radians
float AngleMotor::getCurrentAngle() const {
    return m_currentAngle;
}

// Check if the motor is currently in motion
bool AngleMotor::isMoving() const {
    // Check HLFB, steps complete flag, and StatusReg for movement
    if (m_motorConnector.HlfbState() != MotorDriver::HLFB_ASSERTED) {
        // HLFB not asserted indicates motor is still moving or has a fault
        Serial.println("Debug: isMoving - HLFB not asserted");
        return true;
    }
    
    if (!m_motorConnector.StepsComplete()) {
        // Steps not complete means the motor is still moving
        Serial.println("Debug: isMoving - Steps not complete");
        return true;
    }
    
    // Motor is not moving
    return false;
}

// Verify if the motor has reached the commanded position
bool AngleMotor::verifyPosition() const {
    // First check if motor is still moving
    if (isMoving()) {
        Serial.println("Debug: verifyPosition - Motor is still moving");
        return false;
    }
    
    // Check for alerts
    if (m_motorConnector.StatusReg().bit.AlertsPresent) {
        Serial.println("Debug: verifyPosition - Alerts present, position not verified");
        return false;
    }
    
    // If motor is not moving, has HLFB asserted, and no alerts, position is verified
    Serial.println("Debug: verifyPosition - Position verified successfully");
    return true;
}

// Check for alerts during operation
void AngleMotor::checkAlerts() {
    // Check for faults during operation
    if (HasAlerts(&m_motorConnector)) {
        m_alertCount++;  // Increment alert counter
        Serial.print("Alert detected during operation (count: ");
        Serial.print(m_alertCount);
        Serial.println(")");
        
        // Print details about the alert
        PrintAlerts(&m_motorConnector, "Angle Motor");
        
        // If auto-recovery is enabled, try to clear the faults
        if (m_autoRecoveryEnabled) {
            Serial.println("Auto-recovery: Attempting to clear faults...");
            if (ClearMotorFaults(&m_motorConnector, "Angle Motor")) {
                Serial.println("Auto-recovery: Faults cleared successfully");
            } else {
                Serial.println("Auto-recovery: Failed to clear faults");
            }
        }
    }
}

// Print detailed status information about the motor
void AngleMotor::printStatus() const {
    Serial.print("Angle Motor Status - Position: ");
    Serial.print(getCurrentAngle() * 180.0 / PI);
    Serial.print(" degrees, Moving: ");
    Serial.print(isMoving() ? "Yes" : "No");
    Serial.print(", HLFB: ");
    Serial.print(m_motorConnector.HlfbState() == MotorDriver::HLFB_ASSERTED ? "Asserted" : "Not Asserted");
    Serial.print(", Homed: ");
    Serial.println(m_isHomed ? "Yes" : "No");
}

// Wait for motion to complete with timeout
bool AngleMotor::waitForMotionComplete(unsigned long timeoutMs) {
    Serial.println("Waiting for motion to complete...");
    unsigned long startTime = millis();
    
    // Monitor HLFB and other status during motion
    Serial.print("Initial HLFB state: ");
    Serial.println(m_motorConnector.HlfbState() == MotorDriver::HLFB_ASSERTED ? "Asserted" : "Not Asserted");
    
    while (isMoving()) {
        // Print status periodically
        if ((millis() - startTime) % 1000 < 10) {  // Roughly every second
            Serial.print("HLFB state: ");
            Serial.println(m_motorConnector.HlfbState() == MotorDriver::HLFB_ASSERTED ? "Asserted" : "Not Asserted");
            Serial.print("StepsComplete: ");
            Serial.println(m_motorConnector.StepsComplete() ? "Yes" : "No");
        }
        
        // Check for timeout
        if (millis() - startTime > timeoutMs) {
            Serial.println("Motion timeout - target position not reached");
            Serial.print("Final HLFB state: ");
            Serial.println(m_motorConnector.HlfbState() == MotorDriver::HLFB_ASSERTED ? "Asserted" : "Not Asserted");
            return false;
        }
        
        // Check for alerts during motion
        if (HasAlerts(&m_motorConnector)) {
            Serial.println("Alert detected during motion");
            PrintAlerts(&m_motorConnector, "Angle Motor");
            return false;
        }
        
        delay(10);  // Small delay to avoid excessive polling
    }
    
    Serial.println("Motion completed successfully");
    Serial.print("Final HLFB state: ");
    Serial.println(m_motorConnector.HlfbState() == MotorDriver::HLFB_ASSERTED ? "Asserted" : "Not Asserted");
    return true;
}

// Get the motor driver pointer for fault handling
MotorDriver* AngleMotor::getMotorDriver() {
    return &m_motorConnector;
}

// Set auto-recovery mode (whether to auto-clear faults)
void AngleMotor::setAutoRecovery(bool enabled) {
    m_autoRecoveryEnabled = enabled;
    Serial.print("Auto-recovery ");
    Serial.println(enabled ? "enabled" : "disabled");
}

// Get count of alerts detected since startup or last reset
unsigned int AngleMotor::getAlertCount() const {
    return m_alertCount;
}

// Reset alert counter
void AngleMotor::resetAlertCount() {
    m_alertCount = 0;
    Serial.println("Alert counter reset");
}

// Convert angle to motor steps
int32_t AngleMotor::angleToSteps(float angleInRadians) const {
    return (int32_t)(angleInRadians * ANGLE_STEPS_PER_RADIAN);
}

// Convert steps to angle
float AngleMotor::stepsToAngle(int32_t steps) const {
    return (float)steps / ANGLE_STEPS_PER_RADIAN;
}

