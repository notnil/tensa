/**
 * DispenserMotor.cpp
 * 
 * Implementation file for stepper motor control using ClearCore's M3 connector
 * for pulse generation to a standard stepper driver (DM860I).
 * Only handles step pulse generation, motor is assumed to be always enabled.
 */

#include "DispenserMotor.h"
#include "Arduino.h"
#include "ClearCore.h"

DispenserMotor::DispenserMotor(MotorDriver& motorDriver) : m_motorDriver(motorDriver) {
    // Constructor stores the MotorDriver reference
}

void DispenserMotor::init() {
    // Configure motor for pulse width
    MotorMgr.MotorInputClocking(MotorManager::CLOCK_RATE_LOW); // Set to 5μs pulse width
    
    // Configure motor settings for pulse generation only
    m_motorDriver.AccelMax(DEFAULT_ACCEL);  // Set reasonable acceleration
    
    // Important: Set the direction in software
    m_motorDriver.MotorInAState(false);  // Direction signal (false = forward)
    
    // Enable the motor driver
    m_motorDriver.EnableRequest(true);
    
    Serial.println("DispenserMotor initialization:");
    Serial.print("Motor mode: ");
    Serial.println(m_motorDriver.Mode());
    Serial.print("Motor enabled: ");
    Serial.println(m_motorDriver.EnableRequest());
    Serial.print("Steps per rev: ");
    Serial.println(STEPS_PER_REV);
    Serial.print("Max acceleration: ");
    Serial.println(DEFAULT_ACCEL);
    
    // Initialize with stopped motor (0 RPM)
    setSpeed(STOPPED_RPM);
}

bool DispenserMotor::setSpeed(uint8_t rpm) {
    // Check if speed exceeds maximum or is below minimum (except 0 for stopped)
    if (rpm < STOPPED_RPM || rpm > MAX_RPM) {
        Serial.print("ERROR: Invalid speed: ");
        Serial.println(rpm);
        return false;
    }
    
    m_rpm = rpm;
    
    if (m_rpm > STOPPED_RPM) {
        // Calculate steps per second directly from RPM
        uint32_t stepsPerSec = (m_rpm * STEPS_PER_REV) / SECONDS_PER_MINUTE;
        
        Serial.print("Setting steps per second to: ");
        Serial.print(stepsPerSec);
        Serial.print(" for ");
        Serial.print(m_rpm);
        Serial.println(" RPM");
        
        // Set maximum velocity based on calculated steps per second
        m_motorDriver.VelMax(stepsPerSec);
        
        // Move continuously at the calculated steps per second
        bool moveResult = m_motorDriver.MoveVelocity(stepsPerSec);
        Serial.print("Move command result: ");
        Serial.println(moveResult ? "SUCCESS" : "FAILED");
        
        // Print motor state for debugging
        Serial.print("Motor enabled: ");
        Serial.println(m_motorDriver.EnableRequest());
    } else {
        // Stop motion
        Serial.println("Stopping motor");
        m_motorDriver.MoveStopAbrupt();
    }
    
    return true;
}

uint8_t DispenserMotor::getSpeed() const {
    return m_rpm;
}

