/**
 * ButtonIO.h
 * 
 * Header file for button input functionality.
 */

#ifndef BUTTON_IO_H
#define BUTTON_IO_H

#include "ClearCore.h"

class ButtonIO {
public:
    // Constructor that takes a ClearCore connector pin
    ButtonIO(Connector &buttonPin);
    
    // Initialize the button
    void init();
    
    // Read the current raw button state (true = pressed)
    bool isPressed();
    
private:
    Connector &m_buttonPin;
};

#endif // BUTTON_IO_H