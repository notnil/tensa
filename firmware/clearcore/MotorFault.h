/**
 * MotorFault.h
 * 
 * Header file for motor fault handling functionality.
 * Provides functions to check and clear faults in ClearCore motor drivers.
 */

#ifndef MOTOR_FAULT_H
#define MOTOR_FAULT_H

#include "ClearCore.h"
#include "Arduino.h"  // For String type and Serial

// Enable or disable automatic fault handling
#define HANDLE_MOTOR_FAULTS (1)

/**
 * Check if a motor has any alerts present
 * 
 * @param motor Pointer to the motor driver
 * @return True if the motor has alerts, false otherwise
 */
bool HasAlerts(MotorDriver* motor);

/**
 * Print motor alerts (using Serial)
 * 
 * @param motor Pointer to the motor driver
 * @param motorName Name of the motor for display
 */
void PrintAlerts(MotorDriver* motor, const char* motorName);

/**
 * Clear motor faults (using Serial)
 * 
 * @param motor Pointer to the motor driver
 * @param motorName Name of the motor for display
 * @return True if faults were cleared successfully, false otherwise
 */
bool ClearMotorFaults(MotorDriver* motor, const char* motorName);

/**
 * Check for and handle motor faults (using Serial)
 * 
 * @param motor Pointer to the motor driver
 * @param motorName Name of the motor for display
 * @return True if the motor is in fault state, false otherwise
 */
bool CheckAndHandleFaults(MotorDriver* motor, const char* motorName);

/**
 * Helper function to log messages to Serial
 * 
 * @param message The message to log
 */
void LogMessage(const char* message);

#endif // MOTOR_FAULT_H