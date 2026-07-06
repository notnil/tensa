/**
 * ThrowMotorSystem.cpp
 * 
 * Implementation of the throw motor system for the Tensa firmware.
 */

#include "ThrowMotorSystem.h"

// Constructor - updated to use references
ThrowMotorSystem::ThrowMotorSystem(MotorDriver &topMotor, MotorDriver &bottomMotor)
    : m_topMotor(topMotor)
    , m_bottomMotor(bottomMotor)
    , m_commandedVelocityTop(0)
    , m_commandedVelocityBottom(0)
    , m_velocityResolution(VELOCITY_RESOLUTION)
{
}

// Initialize the motors
void ThrowMotorSystem::init()
{
    Serial.println("Initializing throw motors...");
    
    // Note: Motor mode is now set in the main setup() function
    // to ensure consistent initialization across all motors
    
    // Configure the motors with longer timeout
    m_topMotor.HlfbMode(MotorDriver::HLFB_MODE_HAS_BIPOLAR_PWM);
    m_topMotor.HlfbCarrier(MotorDriver::HLFB_CARRIER_482_HZ);
    m_bottomMotor.HlfbMode(MotorDriver::HLFB_MODE_HAS_BIPOLAR_PWM);
    m_bottomMotor.HlfbCarrier(MotorDriver::HLFB_CARRIER_482_HZ);
    
    // Set velocity and acceleration limits
    uint32_t velocityLimit = (MAX_VELOCITY_THROW * PULSES_PER_REVOLUTION) / RPM_TO_CPS;
    uint32_t accelerationLimit = (MAX_ACCELERATION_THROW * PULSES_PER_REVOLUTION) / RPM_TO_CPS;
    
    m_topMotor.VelMax(velocityLimit);
    m_topMotor.AccelMax(accelerationLimit);
    m_bottomMotor.VelMax(velocityLimit);
    m_bottomMotor.AccelMax(accelerationLimit);
    
    // Initial motor state
    m_topMotor.MotorInAState(false);
    m_topMotor.MotorInBState(false);
    m_bottomMotor.MotorInAState(false);
    m_bottomMotor.MotorInBState(false);
    
    // Disable motors first to ensure a clean start
    m_topMotor.EnableRequest(false);
    m_bottomMotor.EnableRequest(false);
    delay(MOTOR_ENABLE_DELAY_MS);  // Wait for disable to take effect
    
    // Now enable the motors
    Serial.println("Enabling throw motors...");
    m_topMotor.EnableRequest(true);
    m_bottomMotor.EnableRequest(true);
    
    // Wait for HLFB to assert with more robust checking
    unsigned long startTime = millis();
    const unsigned long timeoutMs = MOTOR_INIT_TIMEOUT_MS;
    bool topReady = false;
    bool bottomReady = false;
    
    Serial.println("Waiting for throw motors to initialize...");
    while ((!topReady || !bottomReady) && (millis() - startTime < timeoutMs)) {
        // Check top motor
        if (!topReady && m_topMotor.HlfbState() == MotorDriver::HLFB_ASSERTED) {
            topReady = true;
            Serial.println("Top throw motor ready");
        }
        
        // Check bottom motor
        if (!bottomReady && m_bottomMotor.HlfbState() == MotorDriver::HLFB_ASSERTED) {
            bottomReady = true;
            Serial.println("Bottom throw motor ready");
        }
        
        delay(50);  // Small delay to avoid overwhelming the system
    }
    
    // Check if both motors initialized successfully
    if (!topReady) {
        Serial.println("WARNING: Top throw motor failed to initialize within timeout");
    }
    if (!bottomReady) {
        Serial.println("WARNING: Bottom throw motor failed to initialize within timeout");
    }
    
    // Check for and clear any existing faults
    checkAndHandleFaults();
    
    // Start with motors stopped (speed 0)
    moveTopMotor(0);
    moveBottomMotor(0);
    
    // Double-check motor status after initialization
    Serial.println("Verifying throw motor status...");
    delay(MOTOR_VERIFY_DELAY_MS);  // Give motors time to stabilize
    
    // Print final status of both motors
    if (m_topMotor.StatusReg().bit.Enabled) {
        Serial.println("Top throw motor is enabled");
    } else {
        Serial.println("WARNING: Top throw motor is not enabled");
    }
    
    if (m_bottomMotor.StatusReg().bit.Enabled) {
        Serial.println("Bottom throw motor is enabled");
    } else {
        Serial.println("WARNING: Bottom throw motor is not enabled");
    }
    
    Serial.println("Throw motors initialized");
}

// Move both throw motors at specified velocities
void ThrowMotorSystem::moveThrowMotors(int16_t topVelocityRPM, int16_t bottomVelocityRPM)
{
    // Use the generic moveMotor function to move both motors
    moveMotor(topVelocityRPM, m_topMotor, m_commandedVelocityTop, 1, MOTOR_TOP_THROW);
    moveMotor(bottomVelocityRPM, m_bottomMotor, m_commandedVelocityBottom, -1, MOTOR_BOTTOM_THROW);
}

// Generic motor movement function - updated to use references
void ThrowMotorSystem::moveMotor(int16_t velocityInRPM, MotorDriver &motor, int16_t &commandedVelocity, 
                                int16_t velocityMultiplier, int motorIndex)
{  
    // Check for negative inputs, redundant commands, and exceeding valid range
    if (velocityInRPM < 0 || velocityInRPM > static_cast<int16_t>(MAX_VELOCITY_THROW))
    {
        return;
    }
    
    // Apply the multiplier to the velocity (positive for top motor, negative for bottom motor)
    int16_t adjustedVelocityInRPM = velocityInRPM * velocityMultiplier;
    
    // Check for redundant commands
    if (adjustedVelocityInRPM == commandedVelocity)
    {
        return;
    }
    
    // Check if a motor fault is currently preventing motion
    if (motor.StatusReg().bit.MotorInFault)
    {
        if (AUTO_HANDLE_FAULTS)
        {
            clearMotorFaults(motor, motorIndex);
        }
        commandedVelocity = 0;  // Reset commanded velocity to 0
        return;
    }
    
    // Directly apply the velocity change using quadrature signals
    applyQuadratureForVelocity(motor, commandedVelocity, adjustedVelocityInRPM);
    
    // Keep track of the new commanded velocity
    commandedVelocity = adjustedVelocityInRPM;
}

// Helper method to apply quadrature pulses for a velocity change - updated to use references
void ThrowMotorSystem::applyQuadratureForVelocity(MotorDriver &motor, int16_t currentVelocity, int16_t targetVelocity)
{
    // Determine which order the quadrature must be sent by determining if the
    // new velocity is greater or less than the previously commanded velocity
    // If greater, Input A begins the quadrature. If less, Input B begins the
    // quadrature.
    int32_t currentVelocityRounded = static_cast<int32_t>(round(static_cast<float>(currentVelocity) / m_velocityResolution));
    int32_t targetVelocityRounded = static_cast<int32_t>(round(static_cast<float>(targetVelocity) / m_velocityResolution));
    int32_t velocityDifference = labs(targetVelocityRounded - currentVelocityRounded);
    
    // Ensure we're sending a consistent and accurate quadrature signal
    bool increasing = (targetVelocity > currentVelocity);
    
    for (int32_t i = 0; i < velocityDifference; i++)
    {
        if (increasing)
        {
            // Toggle Input A to begin the quadrature signal.
            motor.MotorInAState(true);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInBState(true);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInAState(false);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInBState(false);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
        }
        else
        {
            motor.MotorInBState(true);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInAState(true);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInBState(false);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
            motor.MotorInAState(false);
            delayMicroseconds(QUAD_SIGNAL_DELAY_US);
        }
    }
}

// Move the top motor at a specific velocity
void ThrowMotorSystem::moveTopMotor(int16_t velocityInRPM)
{
    // Call the generic moveMotor function with the appropriate parameters for the top motor
    // Use multiplier of 1 for positive direction
    moveMotor(velocityInRPM, m_topMotor, m_commandedVelocityTop, 1, MOTOR_TOP_THROW);
}

// Move the bottom motor at a specific velocity
void ThrowMotorSystem::moveBottomMotor(int16_t velocityInRPM)
{
    // Call the generic moveMotor function with the appropriate parameters for the bottom motor
    // Use multiplier of -1 to invert direction
    moveMotor(velocityInRPM, m_bottomMotor, m_commandedVelocityBottom, -1, MOTOR_BOTTOM_THROW);
}

// Check and handle faults for both motors
void ThrowMotorSystem::checkAndHandleFaults()
{
    // Check top motor
    if (m_topMotor.StatusReg().bit.MotorInFault)
    {
        Serial.print("Top Throw Motor fault detected! ");
        Serial.print("Status Register: 0x");
        Serial.println(m_topMotor.StatusReg().reg, HEX);

        if (AUTO_HANDLE_FAULTS)
        {
            clearMotorFaults(m_topMotor, MOTOR_TOP_THROW);
        }
        else
        {
            Serial.println("Fault handling is disabled.");
            printAlerts(m_topMotor, MOTOR_TOP_THROW);
        }
    }
    
    // Check bottom motor
    if (m_bottomMotor.StatusReg().bit.MotorInFault)
    {
        Serial.print("Bottom Throw Motor fault detected! ");
        Serial.print("Status Register: 0x");
        Serial.println(m_bottomMotor.StatusReg().reg, HEX);

        if (AUTO_HANDLE_FAULTS)
        {
            clearMotorFaults(m_bottomMotor, MOTOR_BOTTOM_THROW);
        }
        else
        {
            Serial.println("Fault handling is disabled.");
            printAlerts(m_bottomMotor, MOTOR_BOTTOM_THROW);
        }
    }
}

// Clear motor faults - updated to use references
void ThrowMotorSystem::clearMotorFaults(MotorDriver &motor, int motorIndex)
{
    // Print out clear message
    Serial.print("Clearing Throw Motor ");
    Serial.print(motorIndex);
    Serial.println(" fault");
    
    // Print details of the fault
    printAlerts(motor, motorIndex);
    
    // Print the StatusReg values before attempting to clear
    Serial.print("Motor Status Register before clearing: 0x");
    Serial.println(motor.StatusReg().reg, HEX);
    Serial.print("Enabled: ");
    Serial.print(motor.StatusReg().bit.Enabled);
    Serial.print(", MotorInFault: ");
    Serial.print(motor.StatusReg().bit.MotorInFault);
    Serial.print(", AlertsPresent: ");
    Serial.println(motor.StatusReg().bit.AlertsPresent);
    
    // Print HLFB state before clearing
    Serial.print("HLFB State before clearing: ");
    switch (motor.HlfbState()) {
        case MotorDriver::HLFB_DEASSERTED:
            Serial.println("DEASSERTED");
            break;
        case MotorDriver::HLFB_ASSERTED:
            Serial.println("ASSERTED");
            break;
        case MotorDriver::HLFB_HAS_MEASUREMENT:
            Serial.println("HAS_MEASUREMENT");
            break;
        default:
            Serial.println("UNKNOWN");
            break;
    }
    
    // Perform the fault recovery sequence
    motor.EnableRequest(false);
    delay(FAULT_RECOVERY_SHORT_DELAY_MS);
    
    // Check status after disabling
    Serial.print("Status after disable: Enabled=");
    Serial.print(motor.StatusReg().bit.Enabled);
    Serial.print(", MotorInFault=");
    Serial.println(motor.StatusReg().bit.MotorInFault);
    
    // Now clear alerts
    Serial.println("Calling ClearAlerts()...");
    motor.ClearAlerts();
    
    // Re-enable the motor
    Serial.println("Re-enabling motor...");
    motor.EnableRequest(true);
    delay(FAULT_RECOVERY_LONG_DELAY_MS);
    
    // Check status after re-enabling
    Serial.print("Status after re-enable: Enabled=");
    Serial.print(motor.StatusReg().bit.Enabled);
    Serial.print(", MotorInFault=");
    Serial.println(motor.StatusReg().bit.MotorInFault);
    
    // Verify the fault was cleared
    if (motor.StatusReg().bit.MotorInFault) {
        Serial.println("Motor fault could not be cleared!");
        Serial.print("Alert Register after clearing attempt: 0x");
        Serial.println(motor.AlertReg().reg, HEX);
    } else {
        Serial.println("Motor fault cleared successfully");
    }
}

// Print motor alerts - updated to use references
void ThrowMotorSystem::printAlerts(MotorDriver &motor, int motorIndex)
{
    // Print motor identifier
    Serial.print("Throw Motor ");
    Serial.print(motorIndex);
    Serial.println(" Alerts: ");
    
    // Print alert status using the correct field names that exist in the AlertRegMotor structure
    if (motor.AlertReg().bit.MotorFaulted) {
        Serial.println("    MotorFaulted");
    }
    if (motor.AlertReg().bit.MotionCanceledInAlert) {
        Serial.println("    MotionCanceledInAlert");
    }
    if (motor.AlertReg().bit.MotionCanceledPositiveLimit) {
        Serial.println("    MotionCanceledPositiveLimit");
    }
    if (motor.AlertReg().bit.MotionCanceledNegativeLimit) {
        Serial.println("    MotionCanceledNegativeLimit");
    }
    if (motor.AlertReg().bit.MotionCanceledSensorEStop) {
        Serial.println("    MotionCanceledSensorEStop");
    }
    if (motor.AlertReg().bit.MotionCanceledMotorDisabled) {
        Serial.println("    MotionCanceledMotorDisabled");
    }
    
    // Add a summary of the alert register value
    Serial.print("Alert Register Value: 0x");
    Serial.println(motor.AlertReg().reg, HEX);
    
    // Print the bit-by-bit representation
    Serial.print("Alert Register Bits: ");
    for (int i = 7; i >= 0; i--) {
        Serial.print((motor.AlertReg().reg >> i) & 1);
    }
    Serial.println();
    
    // Print the ReadyState of the motor
    Serial.print("Motor Ready State: ");
    switch (motor.StatusReg().bit.ReadyState) {
        case MotorDriver::MOTOR_DISABLED:
            Serial.println("DISABLED");
            break;
        case MotorDriver::MOTOR_ENABLING:
            Serial.println("ENABLING");
            break;
        case MotorDriver::MOTOR_FAULTED:
            Serial.println("FAULTED");
            break;
        case MotorDriver::MOTOR_READY:
            Serial.println("READY");
            break;
        case MotorDriver::MOTOR_MOVING:
            Serial.println("MOVING");
            break;
        default:
            Serial.println("UNKNOWN");
            break;
    }
}

