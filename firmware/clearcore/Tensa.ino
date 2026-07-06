#include "ClearCore.h"
#include "ButtonIO.h"
#include "DispenserMotor.h"
#include "AngleMotor.h"
#include "MotorFault.h"
#include "ThrowMotorSystem.h"
#include "Server.h"

// Motor Definitions
#define TOP_THROW_MOTOR_PIN     ConnectorM0
#define BOTTOM_THROW_MOTOR_PIN  ConnectorM1
#define ANGLE_MOTOR_PIN         ConnectorM2
#define DISPENSER_PIN           ConnectorM3

// GPIO Definitions
#define BUTTON_PIN              ConnectorIO0

// Create objects
ButtonIO button(BUTTON_PIN);
DispenserMotor dispenserMotor(DISPENSER_PIN);
AngleMotor angleMotor(ANGLE_MOTOR_PIN);
ThrowMotorSystem throwMotors(TOP_THROW_MOTOR_PIN, BOTTOM_THROW_MOTOR_PIN);

#define HANDLE_MOTOR_FAULTS (1)           // Enable/disable automatic fault handling
#define MAX_HOMING_ATTEMPTS 2             // Maximum number of homing attempts
#define MOTOR_INIT_DELAY 5000             // Motor initialization delay (ms)
#define HOMING_TIMEOUT 20000              // Timeout for homing sequence (ms)
#define MOTION_TIMEOUT 10000              // Timeout for motion completion (ms)

// MAC address of the ClearCore
byte mac[] = { 0xDE, 0xAD, 0xBE, 0xEF, 0xFE, 0xED };
IPAddress ip(192, 168, 1, 177);
#define PORT_NUM 80

// Initialize the ClearCore as a server
TensaServer server(PORT_NUM);

void setup() {
  // Initialize serial communication
  Serial.begin(115200);
  delay(5000);  // Wait for serial to initialize
  Serial.println("Server Example Starting...");

  // Set motor M0/M1 group to direct A/B mode for the throw motors
  MotorMgr.MotorModeSet(MotorManager::MOTOR_M0M1, Connector::CPM_MODE_A_DIRECT_B_DIRECT);
  // Set motor M2/M3 group to step and direction mode for the angle motor and dispenser motor
  MotorMgr.MotorModeSet(MotorManager::MOTOR_M2M3, Connector::CPM_MODE_STEP_AND_DIR);

  Serial.println("Inits");
  // Initialize components
  button.init();
  dispenserMotor.init();
  angleMotor.init();
  throwMotors.init();

  // dispenserMotor.setSpeed(DEFAULT_RPM);  // Set to 15 RPM for testing

  // Configure angle motor fault handling
  angleMotor.setAutoRecovery(true);  // Enable automatic fault recovery
  angleMotor.resetAlertCount();      // Start with a clean alert counter

  // Try homing with configured number of attempts
  bool homingSuccess = false;
  for (int attempt = 1; attempt <= MAX_HOMING_ATTEMPTS; attempt++) {
    Serial.print("Homing attempt ");
    Serial.print(attempt);
    Serial.print("/");
    Serial.println(MAX_HOMING_ATTEMPTS);

    // Home the angle motor to establish a reference position
    if (angleMotor.homeMotor(HOMING_TIMEOUT)) {
      Serial.println("Homing successful!");
      homingSuccess = true;
      break;
    }

    Serial.println("Homing attempt failed");
    delay(500);  // Wait before trying again
  }

  if (!homingSuccess) {
    Serial.println("ERROR: All homing attempts failed!");
    Serial.println("Motor position commands will be skipped.");
  }

  // Print final status
  angleMotor.printStatus();

  Serial.println("Initialization complete.");

  // Initialize the server with our callback function
  if (server.init(processCommand, mac, ip)) {
    Serial.println("Server initialized successfully");
  } else {
    Serial.println("Failed to initialize server");
  }
}

void loop() {
  // Update the server - this handles client connections and command processing
  server.update();
}

// Function to find all spaces in a string
void findAllSpaces(const String& str, int* spaceIndices, int& count, int maxSpaces) {
  count = 0;
  for (int i = 0; i < str.length() && count < maxSpaces; i++) {
    if (str.charAt(i) == ' ') {
      spaceIndices[count++] = i;
    }
  }
}

// Process THROW command
String processTHROWCommand(const String& command) {
  // Find all spaces in the command
  int spaceIndices[3];  // We need at most 3 spaces for THROW command
  int spaceCount = 0;
  findAllSpaces(command, spaceIndices, spaceCount, 3);

  // Check if we have enough spaces to separate all parameters
  if (spaceCount < 3) {
    return "ERR Missing parameters";
  }

  // Extract parameters using space indices
  String cmd = command.substring(0, spaceIndices[0]);
  int16_t topThrowSpeedRPM = command.substring(spaceIndices[0] + 1, spaceIndices[1]).toInt();
  int16_t bottomThrowSpeedRPM = command.substring(spaceIndices[1] + 1, spaceIndices[2]).toInt();
  float angle = command.substring(spaceIndices[2] + 1).toFloat();

  // Validate parameters
  if (topThrowSpeedRPM < 0 || bottomThrowSpeedRPM < 0 || angle < 0 || angle > 0.7854) {
    return "ERR Invalid parameter values";
  }

  // Check for faults in throw motors
  //throwMotors.checkAndHandleFaults();

  // Check the angle motor for alerts
  //angleMotor.checkAlerts();

  angleMotor.moveToAngle(angle);
  throwMotors.moveTopMotor(topThrowSpeedRPM);
  throwMotors.moveBottomMotor(bottomThrowSpeedRPM);

  return "OK";
}

// Process DISP command
String processDISPCommand(const String& command) {
  // Since format is always "DISP <speed>", extract everything after "DISP "
  String speedStr = command.substring(5);  // Skip "DISP "

  // Parse and validate speed
  int16_t dispenserSpeed = speedStr.toInt();
  if (dispenserSpeed < 0) return "ERR Invalid dispenser speed";

  Serial.print("Setting dispenser speed: ");
  Serial.println(dispenserSpeed);

  // Try to set the speed, check if successful
  if (!dispenserMotor.setSpeed(dispenserSpeed)) {
    if (dispenserSpeed > MAX_RPM) {
      return "ERR Speed exceeds maximum (" + String(MAX_RPM) + " RPM)";
    } else {
      return "ERR Invalid motor speed";
    }
  }

  return "OK";
}

// Process LOAD command
String processLOADCommand(const String& command) {
  bool isLoaded = button.isPressed();
  return "OK " + String(isLoaded ? "1" : "0");
}

// Command processing function (callback)
String processCommand(const String& command) {
  Serial.print("Processing command: ");
  Serial.println(command);

  // Check if command starts with valid command type
  if (command.startsWith("THROW")) {
    return processTHROWCommand(command);
  } else if (command.startsWith("DISP")) {
    return processDISPCommand(command);
  } else if (command.startsWith("LOAD")) {
    return processLOADCommand(command);
  }

  return "ERR Unknown command";
}

