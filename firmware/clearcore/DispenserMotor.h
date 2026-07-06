/**
 * DispenserMotor.h
 * 
 * Header file for stepper motor control using ClearCore's M3 connector
 * for pulse generation to a standard stepper driver (DM860I).
 * 
 * DM860I Specs:
 * - Pulse Input Frequency: 0~200kHz
 * - Pulse Width: 2.5μS minimum
 * - Configured for 12800 pulses/rev
 * 
 * Implementation:
 * - Uses M3 for step pulse generation
 * - Configures pulse width to 5μs (MotorManager::CLOCK_RATE_LOW)
 * - Frequency is calculated as (RPM × Steps per Revolution) / 60
 * - At 15 RPM with 12800 steps/rev: (15 * 12800) / 60 = 3200 Hz
 * - Direction is set in software (false = forward)
 */

#ifndef DISPENSER_MOTOR_H
#define DISPENSER_MOTOR_H

#include "ClearCore.h"

// Motor and Driver Configuration
#define STEPS_PER_REV 12800    // Steps per revolution of the stepper motor
#define DEFAULT_RPM 15         // Default motor speed in RPM
#define MAX_RPM 30            // Maximum motor speed in RPM
#define DEFAULT_ACCEL 30000    // Default acceleration (steps/sec²)
#define SECONDS_PER_MINUTE 60  // Conversion factor from minutes to seconds
#define STOPPED_RPM 0          // RPM value for stopped motor

class DispenserMotor {
public:
    // Constructor that takes a ClearCore MotorDriver reference
    DispenserMotor(MotorDriver& motorDriver);
    
    // Initialize the motor
    // - Sets pulse width to 5μs using MotorManager::CLOCK_RATE_LOW
    // - Sets direction to forward
    // - Configures acceleration
    void init();
    
    // Set the motor speed in RPM (integer)
    // Returns true if successful, false if speed exceeds MAX_RPM
    bool setSpeed(uint8_t rpm);
    
    // Get the current speed in RPM
    uint8_t getSpeed() const;
    
private:
    MotorDriver& m_motorDriver;
    
    // Current settings
    uint32_t m_rpm = STOPPED_RPM;  // Current motor speed (0 RPM - stopped)
};

#endif // DISPENSER_MOTOR_H