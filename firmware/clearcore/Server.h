/**
 * Server.h
 * 
 * Ethernet server implementation for the Tensa firmware.
 * Handles client connections and command processing through a callback mechanism.
 * This implementation wraps the Arduino Ethernet libraries.
 */

#ifndef TENSA_SERVER_H
#define TENSA_SERVER_H

#include <Arduino.h>
#include <SPI.h>
#include <Ethernet.h>

// Callback function type definition
// Takes a string command as input and returns a string response
typedef String (*CommandCallback)(const String&);

class TensaServer {
public:
    // Constructor with default port 80
    TensaServer(uint16_t port = 80);
    
    /**
     * Initialize the server with a callback function for command processing
     * 
     * @param callback Function that will process received commands and return responses
     * @param mac MAC address of the Ethernet shield
     * @param ip IP address to assign to the server
     * @return True if initialization was successful, false otherwise
     */
    bool init(CommandCallback callback, byte mac[6], const IPAddress& ip);
    
    /**
     * Update method to be called in the main loop to check for client connections
     * and handle incoming commands
     */
    void update();
    
private:
    // Port number for the server
    uint16_t m_port;
    
    // Ethernet server instance
    EthernetServer m_server;
    
    // Client instance for handling connections
    EthernetClient m_client;
    
    // Callback function for processing commands
    CommandCallback m_commandCallback;
    
    // Flag to track initialization status
    bool m_initialized;
    
    // Helper method to read a line from the client
    String readLine(EthernetClient& client);
};

#endif // TENSA_SERVER_H 