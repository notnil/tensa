/**
 * Server.cpp
 * 
 * Implementation of the Ethernet server for the Tensa firmware.
 * This implementation wraps the Arduino Ethernet libraries.
 */

#include "Server.h"

// Constructor
TensaServer::TensaServer(uint16_t port)
  : m_port(port), m_server(port), m_commandCallback(nullptr), m_initialized(false) {
  // Nothing else to initialize here
}

/**
 * Initialize the server with a callback function for command processing
 */
bool TensaServer::init(CommandCallback callback, byte mac[6], const IPAddress& ip) {
  // Check if the callback function is valid
  if (callback == nullptr) {
    Serial.println("Error: No callback function provided");
    return false;
  }

  // Store the callback function
  m_commandCallback = callback;

  // Make sure the physical link is active before continuing
  while (Ethernet.linkStatus() == LinkOFF) {
    Serial.println("The Ethernet cable is unplugged...");
    delay(1000);
  }

  // Configure with the provided IP address
  Ethernet.begin(mac, ip);

  Serial.print("Server IP address: ");
  Serial.println(Ethernet.localIP());

  // Start the server
  m_server.begin();

  Serial.print("Server listening on port ");
  Serial.println(m_port);

  m_initialized = true;
  return true;
}

/**
 * Update method to be called in the main loop
 */
void TensaServer::update() {
  // Check if the server is initialized
  if (!m_initialized || m_commandCallback == nullptr) {
    Serial.println("Server not initialized or callback missing");
    return;
  }

  //Serial.println("Checking for client connections...");

  // Look for clients with data available
  m_client = m_server.available();

  if (m_client) {
    IPAddress clientIP = m_client.remoteIP();

    // Check if m_client has a valid IP address
    if (clientIP != IPAddress(0, 0, 0, 0)) {
      Serial.print("Client connected from IP: ");
      Serial.println(clientIP);
    } else {
      Serial.println("Ignoring invalid client connection (0.0.0.0).");
      m_client.stop();  // Ignore invalid clients
    }

    while (m_client.connected()) {
      while (m_client.available()) {
        // Read a line from the client
        String command = readLine(m_client);
        
        // Process the command if it's not empty
        if (command.length() > 0) {
          // Call the command callback function to process the command
          String response = m_commandCallback(command);
          m_client.println(response);
    
        }
      }
    }
  }
}

/**
 * Helper method to read a line from the client
 */
String TensaServer::readLine(EthernetClient& client) {
  String currentLine = "";

  Serial.println("Reading line from m_client...");
  // Read until newline or timeout
  while (m_client.available()) {
    char c = m_client.read();

    if (c == '\n') {
      // End of line found
      currentLine.trim();  // Remove any trailing whitespace
      Serial.print("Line read complete: ");
      Serial.println(currentLine);
      return currentLine;
    } else if (c != '\r') {
      // Add character to the current line (skip carriage returns)
      currentLine += c;
    }
  }

  // Return whatever we've read so far
  currentLine.trim();
  Serial.print("Partial line read: ");
  Serial.println(currentLine);
  return currentLine;
}
