/**
 * ButtonIO.cpp
 * 
 * Implementation file for button input functionality.
 */

#include "ButtonIO.h"

ButtonIO::ButtonIO(Connector &buttonPin) : m_buttonPin(buttonPin) {
    // Constructor just stores the reference to the pin
}

void ButtonIO::init() {
    // Configure the button pin as input
    m_buttonPin.Mode(Connector::INPUT_DIGITAL);
}

bool ButtonIO::isPressed() {
    // Read the current raw button state
    return m_buttonPin.State();
}

