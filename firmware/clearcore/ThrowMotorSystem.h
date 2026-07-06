/**
 * ThrowMotorSystem.h
 * 
 * Implements the throw motor system for the Tensa firmware.
 * Controls the top and bottom throw motors with precise timing.
 */

#ifndef THROW_MOTOR_SYSTEM_H
#define THROW_MOTOR_SYSTEM_H

#include "ClearCore.h"
#include "Arduino.h"
#include <math.h>  // for abs()
#include <stdint.h> // for int16_t

// Motor configuration constants
#define PULSES_PER_REVOLUTION 800
#define RPM_TO_CPS 60
#define MAX_VELOCITY_THROW 3300
#define MAX_ACCELERATION_THROW 1500

// Motor indices for identification
#define MOTOR_TOP_THROW 0
#define MOTOR_BOTTOM_THROW 1

// Flag for enabling automatic fault handling
#define AUTO_HANDLE_FAULTS true

// Timing constants (in microseconds and milliseconds)
#define QUAD_SIGNAL_DELAY_US 5
#define MOTOR_INIT_TIMEOUT_MS 8000
#define MOTOR_ENABLE_DELAY_MS 100
#define MOTOR_VERIFY_DELAY_MS 500
#define FAULT_RECOVERY_SHORT_DELAY_MS 10
#define FAULT_RECOVERY_LONG_DELAY_MS 100

// Velocity control constants
#define VELOCITY_RESOLUTION 1.0f

class ThrowMotorSystem {
public:
    // Constructor - changed to use references like AngleMotor
    ThrowMotorSystem(MotorDriver &topMotor, MotorDriver &bottomMotor);
    
    // Initialize the throw motor system
    void init();
    
    // Move both throw motors at specified velocities in RPM
    void moveThrowMotors(int16_t topVelocityRPM, int16_t bottomVelocityRPM);
    
    // Move individual motors at specified velocities in RPM
    void moveTopMotor(int16_t velocityInRPM);
    void moveBottomMotor(int16_t velocityInRPM);
    
    // Check and handle motor faults
    void checkAndHandleFaults();
    
    // Get current commanded velocity (in RPM)
    int16_t getTopMotorSpeed() const { return m_commandedVelocityTop; }
    
    // Get current commanded velocity for bottom motor (in RPM)
    // Returns the absolute value (positive) for consistency
    int16_t getBottomMotorSpeed() const { return abs(m_commandedVelocityBottom); }
    
private:
    // Motor references - references instead of pointers
    MotorDriver &m_topMotor;
    MotorDriver &m_bottomMotor;
    
    // Track commanded velocities for both motors (in RPM)
    int16_t m_commandedVelocityTop;
    int16_t m_commandedVelocityBottom;  // Stored as negative value internally
    
    // Velocity resolution as defined in MSP software
    float m_velocityResolution;
    
    // Generic motor movement function - updated to use references
    void moveMotor(int16_t velocityInRPM, MotorDriver &motor, int16_t& commandedVelocity, 
                  int16_t velocityMultiplier, int motorIndex);
    
    // Apply quadrature signals for a velocity change - updated to use references
    void applyQuadratureForVelocity(MotorDriver &motor, int16_t currentVelocity, int16_t targetVelocity);
    
    // Clear motor faults - updated to use references
    void clearMotorFaults(MotorDriver &motor, int motorIndex);
    
    // Print motor alerts - updated to use references
    void printAlerts(MotorDriver &motor, int motorIndex);
};

#endif // THROW_MOTOR_SYSTEM_H