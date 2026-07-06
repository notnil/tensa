/**
 * MotorFault.cpp
 * 
 * Implementation of motor fault handling functionality.
 */

#include "MotorFault.h"

// Helper function to log messages using Serial
void LogMessage(const char* message) {
    Serial.println(message);
}

// Check if a motor has any alerts present
bool HasAlerts(MotorDriver* motor) {
    return motor->StatusReg().bit.AlertsPresent;
}

void PrintAlerts(MotorDriver* motor, const char* motorName) {
    Serial.print("Alerts present for motor ");
    Serial.print(motorName);
    Serial.println(":");
    
    if (motor->AlertReg().bit.MotionCanceledInAlert) {
        Serial.println("    MotionCanceledInAlert");
    }
    if (motor->AlertReg().bit.MotionCanceledPositiveLimit) {
        Serial.println("    MotionCanceledPositiveLimit");
    }
    if (motor->AlertReg().bit.MotionCanceledNegativeLimit) {
        Serial.println("    MotionCanceledNegativeLimit");
    }
    if (motor->AlertReg().bit.MotionCanceledSensorEStop) {
        Serial.println("    MotionCanceledSensorEStop");
    }
    if (motor->AlertReg().bit.MotionCanceledMotorDisabled) {
        Serial.println("    MotionCanceledMotorDisabled");
    }
    if (motor->AlertReg().bit.MotorFaulted) {
        Serial.println("    MotorFaulted");
    }
}

bool ClearMotorFaults(MotorDriver* motor, const char* motorName) {
    // If no alerts, nothing to clear
    if (!HasAlerts(motor)) {
        return true;
    }
    
    // Handling fault: clearing faults by cycling enable signal to motor.
    Serial.print("Clearing faults for ");
    Serial.println(motorName);
    
    // Print alerts to help with debugging
    PrintAlerts(motor, motorName);
    
    // Disable the motor first
    motor->EnableRequest(false);
    delay(100);  // Give more time for disable to take effect
    
    // Clear alerts while motor is disabled
    motor->ClearAlerts();
    delay(50);
    
    // Re-enable the motor
    motor->EnableRequest(true);
    delay(200);  // Give more time for enable to take effect
    
    // Check if alerts are still present
    if (HasAlerts(motor)) {
        Serial.println("Initial fault clearing failed, trying again");
        
        // Try another cycle of disable/clear/enable
        motor->EnableRequest(false);
        delay(200);
        motor->ClearAlerts();
        delay(50);
        motor->EnableRequest(true);
        delay(200);
    }
    
    // Final check
    if (HasAlerts(motor)) {
        Serial.println("WARNING: Could not clear all faults!");
        return false;
    } else {
        Serial.println("Faults cleared successfully.");
        return true;
    }
}

bool CheckAndHandleFaults(MotorDriver* motor, const char* motorName) {
    Serial.print("Checking motor: ");
    Serial.println(motorName);

    if (motor->StatusReg().bit.MotorInFault) {
        Serial.print(motorName);
        Serial.println(" fault detected!");

        if (HANDLE_MOTOR_FAULTS) {
            return ClearMotorFaults(motor, motorName);
        } else {
            Serial.println("Enable automatic fault handling by setting HANDLE_MOTOR_FAULTS to 1.");
        }

        Serial.println("Motion may not have completed as expected. Proceed with caution.");
        return true;
    } else {
        if (HasAlerts(motor)) {
            Serial.print(motorName);
            Serial.println(" has alerts present.");
            PrintAlerts(motor, motorName);
            
            if (HANDLE_MOTOR_FAULTS) {
                return ClearMotorFaults(motor, motorName);
            }
            
            return true;
        } else {
            Serial.print(motorName);
            Serial.println(" Ready");
            return false;
        }
    }
}

