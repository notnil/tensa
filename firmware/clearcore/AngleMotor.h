/**
 * AngleMotor.h
 * 
 * Header file for the angle motor control functionality.
 * Controls a stepper motor that adjusts the launch angle of the ball.
 */

#ifndef ANGLE_MOTOR_H
#define ANGLE_MOTOR_H

#include "ClearCore.h"
#include <math.h>  // For PI constant

class AngleMotor {
public:
    // Constructor that takes a ClearCore connector pin (must be configured for step/dir mode)
    AngleMotor(MotorDriver &motorConnector);
    
    // Initialize the angle motor
    void init();
    
    // Home the motor to establish a reference position
    bool homeMotor(unsigned long timeoutMs = 10000);
    
    // Move to an absolute angle position in radians
    bool moveToAngle(float angleInRadians);
    
    // Get current angle in radians
    float getCurrentAngle() const;
    
    // Check if the motor is currently in motion
    bool isMoving() const;
    
    // Check for alerts, to be called in the main loop
    void checkAlerts();
    
    // Get the motor driver pointer for fault handling
    MotorDriver* getMotorDriver();
    
    // Verify if the motor has reached the commanded position
    bool verifyPosition() const;
    
    // Print detailed status information about the motor
    void printStatus() const;
    
    // Wait for motion to complete with timeout
    bool waitForMotionComplete(unsigned long timeoutMs = 5000);
    
    // NEW: Set auto-recovery mode (whether to auto-clear faults)
    void setAutoRecovery(bool enabled);
    
    // NEW: Get count of alerts detected since startup or last reset
    unsigned int getAlertCount() const;
    
    // NEW: Reset alert counter
    void resetAlertCount();
    
private:
    MotorDriver &m_motorConnector;
    
    // Homing state
    bool m_isHomed = false;  // Flag to indicate if the motor has been homed
    
    // Fault handling
    bool m_autoRecoveryEnabled = true;   // Whether to automatically try to clear faults
    unsigned int m_alertCount = 0;       // Number of alerts detected
    
    // Angle motor calibration constants
    static const int32_t MEASURED_STEPS_45_DEG = 4350;  // Empirically measured steps for 45 degrees
    
    // Using constexpr for float constants as required by C++ standard
    static constexpr float MIN_ANGLE = 0.0f;  // Minimum angle in radians (0 degrees)
    static constexpr float MAX_ANGLE_DEGREES = 45.0f;  // Maximum angle in degrees
    
    // Define these values in the cpp file to avoid constant expression issues
    static constexpr float PI_VALUE = 3.14159265358979323846f;
    static constexpr float MAX_ANGLE = MAX_ANGLE_DEGREES * PI_VALUE / 180.0f;  // Max angle in radians
    static constexpr float ANGLE_STEPS_PER_RADIAN = MEASURED_STEPS_45_DEG / MAX_ANGLE;  // Steps per radian
    
    // Motor configuration constants
    static const uint32_t MAX_VELOCITY = 300;      // Maximum speed (RPM)
    static const uint32_t MAX_ACCELERATION = 300;  // Maximum acceleration (RPM/s)
    static const uint32_t PULSES_PER_REVOLUTION = 800;  // Number of encoder pulses per full revolution
    
    // Current position tracking
    int32_t m_currentSteps = 0;
    float m_currentAngle = 0.0f;
    
    // Convert angle to motor steps
    int32_t angleToSteps(float angleInRadians) const;
    
    // Convert steps to angle
    float stepsToAngle(int32_t steps) const;
};

#endif // ANGLE_MOTOR_H