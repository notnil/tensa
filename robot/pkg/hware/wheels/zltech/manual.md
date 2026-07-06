# ZLAC8015D Servo Driver Protocol Manual

This document provides a comprehensive overview of the ZLAC8015D Servo Driver, including its specifications, wiring, and communication protocols. It consolidates information from the user manual, CANopen communication routine, and quick start guide.

***

## 1. Product Introduction

### 1.1 Overview
The **ZLAC8015D** is a high-performance digital AC servo driver designed specifically for hub servo motors. It features high integration, combining driver and controller functions, and supports both RS485 and CANopen bus communication. 

 

### 1.2 Features
* **Communication**:
    * **CANopen**: Supports CiA301 and CiA402 protocols. Baud rates range from 100 to 1000 Kbps (default 500 Kbps). Up to 127 devices can be networked. 
    * **RS485**: Supports Modbus-RTU protocol. Baud rates range from 9600 to 128000 bps (default 115200 bps). Up to 32 devices can be networked. 
* **Control Modes**: Supports position, velocity, and torque control modes. 
* **Voltage**: Input voltage range of 24V-48VDC. 
* **I/O**: 2 programmable, isolated signal input ports for functions like enable, start/stop, emergency stop, and limit. 
* **Protection**: Includes over-voltage, under-voltage, and over-current protection. 

### 1.3 Applications
The driver is suitable for a variety of applications, including:
* Automated Guided Vehicles (AGVs) 
* Delivery Robots 
* Service Robots 
* Automated Handling Machines 

***

## 2. Specifications

### 2.1 Electrical Specifications

| Driver Parameter | Min Value | Typical Value | Max Value | Unit |
| :--- | :--- | :--- | :--- | :--- |
| Input voltage | 20 VDC | 36VDC | 48VDC | V |
| Output current (peak) | 0 | 15 | 30 | A |
| Control signal input current | 7 | 10 | 16 | mA |
| Over-voltage protection | | 75 | | VDC |
| Under-voltage protection | | 16 | | VDC |
| Input signal voltage | | 5 | | VDC |
| Insulation resistance | 18 | 20 | | ΜΩ |

### 2.2 Environmental Specifications

| Environment | Parameter | Value |
| :--- | :--- | :--- |
| | Cooling Type | Natural cooling or forced cooling |
| | Application Occasion | Avoid dust, oil mist, and corrosive gases |
| **Working** | Working temperature | $0\sim50^{\circ}C$ |
| | Max. ambient humidity | 90% RH (no condensation) |
| | Storage temperature | $-10^{\sim}70^{\circ}C$ |
| | Vibration | 10~55Hz/0.15mm |

***

## 3. Installation

### 3.1 Dimensions

 
*All units are in mm.*

### 3.2 Installation Guidelines
* Install the driver on a non-flammable metal frame to ensure good heat dissipation. 
* For optimal heat dissipation, narrow-side installation is recommended. 
* Use M3 screws for mounting. 
* Ensure the installation environment is free from dust, corrosive gases, and flammable materials. 
* Avoid installing in locations with poor air circulation or where the ambient temperature exceeds 60°C. 

***

## 4. Interfaces and Wiring

 

### 4.1 Power and Motor Interface

There are separate interfaces for the left and right motors. The power supply can be connected to either interface, or both simultaneously. 

**Left Motor Power & Supply** 

| Pin | Mark | Name | Function |
| :-- | :--- | :--- | :--- |
| 1 | DC | Power Supply | Power supply 24V-48V |
| 2 | GND | Interface | Power ground |
| 3 | U | Motor Power | Connect to motor phase U |
| 4 | V | Interface | Connect to motor phase V |
| 5 | W | | Connect to motor phase W |

**Right Motor Power & Supply** 

| Pin | Mark | Name | Function |
| :-- | :--- | :--- | :--- |
| 1 | U | Motor Power | Connect to motor phase U |
| 2 | V | Interface | Connect to motor phase V |
| 3 | W | | Connect to motor phase W |
| 4 | DC | Power Supply | Power supply 24V-48V |
| 5 | GND | Interface | Power ground |

### 4.2 Encoder and Hall Port (J2/J6)

These ports are for the incremental encoder and Hall sensors for both the left and right motors. 

| Pin | Mark | Name | Function |
| :-- | :--- | :--- | :--- |
| 1 | A+ | Encoder | |
| 2 | A- | | |
| 3 | B+ | | |
| 4 | B- | | |
| 5 | RTC+ | Temperature Sensor | |
| 6 | RTC- | | |
| 7 | V | Hall Sensor | |
| 8 | W | | |
| 9 | U | | |
| 10 | GND | Power ground | |
| 11 | VCC | Power positive | Output to encoder and HALL |
| 12 | GND | Power ground | |

### 4.3 Control Signal Port (J3 & J4)

**J3 - Brake and Output Port** 

| Pin | Mark | Name | Function |
| :-- | :--- | :--- | :--- |
| 1 | BGND-L | Left brake power- | Left brake control |
| 2 | -BR-L | Left brake- | |
| 3 | BDC-L | Left brake power+/+ | |
| 4 | BGND-R | Right brake power- | Right brake control |
| 5 | -BR-R | Right brake- | |
| 6 | BDC-R | Right brake power+/+ | |
| 7 | OUTPUT1 | Internal pull up 5V output | Configurable via CAN/485 |
| 8 | OUTPUT2 | | Configurable via CAN/485 |

**J4 - Encoder Output and Input Port** 

| Pin | Mark | Name | Function |
| :-- | :--- | :--- | :--- |
| 1 | AOUT-L | Left motor encoder A | Left motor encoder output signal |
| 2 | BOUT-L | Left motor encoder B | |
| 3 | AOUT-R | Right motor encoder A | Right motor encoder output signal |
| 4 | BOUT-R | Right motor encoder B | |
| 5 | +5V | Encoder +5V power | External power output (<100mA) |
| 6 | GND | Encoder +5V power supply- | |
| 7 | INPUT1 | Input signal, internally limited 5V | Configurable via CAN/485 |
| 8 | INPUT2 | Input signal, internally limited 5V | Configurable via CAN/485 |

### 4.4 Communication Port (J5)
This port provides access to both CANopen and RS485 communication. 

| Pin | Mark | Name |
| :-- | :--- | :--- |
| 1 | CANH | CAN |
| 2 | A | RS485 |
| 3 | CANL | CAN |
| 4 | B | RS485 |
| 5 | CANH | CAN |
| 6 | A | RS485 |
| 7 | CANL | CAN |
| 8 | B | RS485 |

**Note**: Pins (1, 3) and (5, 7) are two sets of CANopen ports that can be used simultaneously. 

### 4.5 I/O Wiring
The two programmable inputs (INPUT1, INPUT2) are opto-isolated. A driving current of at least 10mA is required for reliable conduction. The default input voltage level is 5V; for higher voltages, an external current-limiting resistor is needed (e.g., 1KΩ for 12V, 2KΩ for 24V). 

 

### 4.6 Brake Circuit Wiring
The driver includes two brake control circuits. An external DC power supply (20V-24V) is required for the electromagnetic brake. 

 

### 4.7 Regen (Regenerative) Circuit
For applications involving high speeds (>100 RPM) or frequent emergency stops, an external regenerative resistor is recommended to dissipate the back electromotive force (EMF). A recommended starting point is a **5Ω, 100W** resistor. 

 

***

## 5. Communication Protocols

### 5.1 CANopen Protocol (CiA 301 / 402)

#### **NMT (Network Management)**
NMT services manage the state of the CANopen nodes (slaves). The master sends commands to transition slaves between states. 
* **COB-ID**: `0x000` 
* **Data Format**: `[Command Byte | Node-ID]` 

**NMT Commands:** 

| Command | Description |
| :--- | :--- |
| `0x01` | Start Node (Enter Operating State) |
| `0x02` | Stop Node (Enter Stop State) |
| `0x80` | Enter Pre-operational State |
| `0x81` | Reset Node (Application) |
| `0x82` | Reset Node (Communication) |

#### **Heartbeat Message**
The driver acts as a Heartbeat Producer to indicate its status and presence on the network. The interval is configured in object `0x1017`. 
* **COB-ID**: `0x700` + Node-ID 
* **Data**: `[Status Byte]` 

**Heartbeat Status Codes:** 

| Status | Description |
| :--- | :--- |
| `0x00` | Boot-up |
| `0x04` | Stop Status |
| `0x05` | Operation Status |
| `0x7F` | Pre-operation Status |

#### **SDO (Service Data Object)**
SDOs are used to access (read/write) entries in the device's Object Dictionary. It uses a confirmed client/server model. 
* **Send COB-ID (Master -> Driver)**: `0x600` + Node-ID 
* **Return COB-ID (Driver -> Master)**: `0x580` + Node-ID 

**SDO Message Format:** 

| Byte 0 | Bytes 1-2 | Byte 3 | Bytes 4-7 |
| :--- | :--- | :--- | :--- |
| Command | Index (LSB first) | Sub-Index | Data (LSB first) |

**Common SDO Command Words:** 

| Command | Function | Data Length |
| :--- | :--- | :--- |
| `0x2F` | Write 1 Byte | 1 Byte |
| `0x2B` | Write 2 Bytes | 2 Bytes |
| `0x23` | Write 4 Bytes | 4 Bytes |
| `0x40` | Read | 0 Bytes |
| `0x60` | Write Response | 0-4 Bytes |
| `0x4F`/`4B`/`43` | Read Response | 1/2/4 Bytes |
| `0x80` | Error Response | 4 Bytes |

#### **PDO (Process Data Object)**
PDOs are used for real-time data transfer without the overhead of a confirmation message. The driver supports 4 TPDOs (Transmit) and 4 RPDOs (Receive). PDOs must be mapped to objects in the Object Dictionary before use. 

**PDO Mapping Steps (Example: Map speed `0x606C` to TPDO0)**

1.  **Clear existing mapping**: Write `0` to object `0x1A00`, sub-index `0`. 
    * `ID: 0x601, Data: 2F 00 1A 00 00 00 00 00`
2.  **Map the new object**: Write the mapping entry (Index, Sub-Index, Length) to `0x1A00`, sub-index `1`. The mapping content for `0x606C`, sub-index `03` (32-bit value) is `0x606C0320`.
    * `ID: 0x601, Data: 23 00 1A 01 20 03 6C 60` 
3.  **Set Transmission Type**: Write the trigger type to `0x1800`, sub-index `2`. (e.g., `0xFE` for event trigger). 
    * `ID: 0x601, Data: 2F 00 18 02 FE 00 00 00`
4.  **Enable the mapping**: Write the number of mapped objects (in this case, 1) to `0x1A00`, sub-index `0`. 
    * `ID: 0x601, Data: 2F 00 1A 00 01 00 00 00`
5.  **Start PDO Communication**: Send the NMT "Start Node" command. 
    * `ID: 0x000, Data: 01 01` (for Node-ID 1)

#### **CiA402 State Machine**
The driver's behavior is governed by the CiA402 state machine. Transitions are controlled via bits in the **Controlword (0x6040)**, and the current state is reflected in the **Statusword (0x6041)**.

 

**Basic Enable Sequence (Initialization):** 
To reach the `OPERATION ENABLE` state where the motor can be controlled, send the following sequence of commands to the Controlword (`0x6040`):
1.  **Shutdown**: `0x06` -> transitions to `READY TO SWITCH ON`
2.  **Switch On**: `0x07` -> transitions to `SWITCHED ON`
3.  **Enable Operation**: `0x0F` -> transitions to `OPERATION ENABLE`

### 5.2 RS485 Protocol (Modbus-RTU)
The driver supports Modbus-RTU communication via the RS485 interface on port J5. 
* **Default Baud Rate**: 115200 bps 
* **Protocol**: Modbus-RTU 

***

## 6. Control Modes

The operating mode is selected by writing to the **Modes of Operation object (0x6060)**. 

### 6.1 Profile Velocity Mode (`0x6060` = 3)
The motor accelerates to a target velocity and maintains it.
* **Primary Objects**:
    * `0x60FF`: Target Velocity (Left/Right/Sync)
    * `0x6083`: Profile Acceleration
    * `0x6084`: Profile Deceleration
* **Example Command (Synchronous control, 100 RPM for both motors)**: 
    * `ID: 0x601, Data: 23 FF 60 03 64 00 64 00`

### 6.2 Profile Position Mode (`0x6060` = 1)
The motor moves to a target position based on a defined motion profile. Can be relative or absolute.
* **Primary Objects**:
    * `0x607A`: Target Position
    * `0x6081`: Profile Velocity (Max speed during move)
    * `0x6083`: Profile Acceleration
    * `0x6084`: Profile Deceleration
* **Start Command**:
    * **Relative Motion**: Toggle Controlword (`0x6040`) from `0x4F` to `0x5F`. 
    * **Absolute Motion**: Toggle Controlword (`0x6040`) from `0x0F` to `0x1F`. 

### 6.3 Profile Torque Mode (`0x6060` = 4)
The motor applies a specific target torque.
* **Primary Objects**:
    * `0x6071`: Target Torque
    * `0x6087`: Torque Slope (rate of torque change)
* **Example Command (Synchronous control, 1000 mA for both motors)**: 
    * `ID: 0x601, Data: 23 71 60 03 E8 03 E8 03`

***

## 7. Faults and Status

### 7.1 LED Indicators
* **Green LED**: Power indicator. Solid on when powered. 
* **Red LED**: Fault indicator. Flashes a number of times corresponding to the fault code. 

### 7.2 Fault Codes
Faults are read from the **Error Code object (0x603F)** or the **Error Register (0x1001)**. Faults are cleared by writing `0x80` to the **Controlword (0x6040)**. 

| Red Flashes | Fault Code (0x603F) | Description | Common Causes |
| :--- | :--- | :--- | :--- |
| 1 | `0x0001` | Over-Voltage | Power supply too high; excessive back EMF. |
| 2 | `0x0002` | Under-Voltage | Power supply too low. |
| 3 | `0x0004` | Over-Current | Instantaneous current too high; loose motor power cable. |
| 4 | `0x0008` | Over-Load | Motor stall; incorrect parameters. |
| 6 | `0x0020` | Position/Encoder Out-of-Tolerance | Motor stall; encoder issue. |
| 9 | `0x0100` | Parameter Reading/EEPROM Error | Firmware update without factory reset; EEPROM damage. |
| 10 | `0x0200` | HALL Fault | Loose motor cable; incorrect Hall signal. |
| 11 | `0x0400` | High Motor Temperature | Motor current too high; damaged thermistor. |
| 12 | `0x0800` | Encoder Error | Loose or disconnected encoder cable. |
| 13 | `0x2000` | Speed Setting Error | Given speed exceeds rated speed. |

*Note: For faults related to a specific motor (e.g., Over-current), the fault code appears in the low 16 bits of `0x603F` for the right motor and the high 16 bits for the left motor.* 

***

## 8. Object Dictionary

This section lists the main objects available for configuration and monitoring via CANopen SDO.

### CiA301 Communication Parameters (1000h)

| Index | Sub | Name | Description | Type | Attr. |
| :--- | :-: | :--- | :--- | :--- | :--- |
| `1000h` | 0 | Device Type | Indicates CiA402 device profile. | U32 | RO |
| `1001h` | 0 | Error Register | Current driver error status. | U8 | RO |
| `1005h` | 0 | COB-ID SYNC | COB-ID for the SYNC message. | U32 | RW |
| `1017h` | 0 | Producer Heartbeat Time | Heartbeat interval in ms. | U16 | RW |
| `1200h` | 1 | COB-ID Client to Server | SDO receive ID (`600h` + NodeID). | U32 | RO |
| `1200h` | 2 | COB-ID Server to Client | SDO transmit ID (`580h` + NodeID). | U32 | RO |
| `1400h`-`1403h` | | RPDO Communication Parameter | Configures RPDO 0-3 (COB-ID, Type, etc.). | | RW/S |
| `1600h`-`1603h` | | RPDO Mapping Parameter | Maps objects to RPDO 0-3. | | RW/S |
| `1800h`-`1803h` | | TPDO Communication Parameter | Configures TPDO 0-3 (COB-ID, Type, etc.). | | RW/S |
| `1A00h`-`1A03h` | | TPDO Mapping Parameter | Maps objects to TPDO 0-3. | | RW/S |

### Manufacturer-Specific Parameters (2000h)

| Index | Sub | Name | Description | Default |
| :--- | :-: | :--- | :--- | :--- |
| `2001h` | 0 | RS485 Node Number | Sets the Modbus node address (1-127). | 1 |
| `2002h` | 0 | RS485 Baudrate | Sets the RS485 baud rate. | 2 (115200) |
| `2008h` | 0 | Motor Max Speed | Sets the absolute maximum speed limit (r/min). | 1000 |
| `200Ah` | 0 | CAN Node Number | Sets the CANopen node address (1-127). | 1 |
| `200Bh` | 0 | CAN Baudrate | Sets the CANopen baud rate. | 1 (500k) |
| `200Fh` | 0 | Sync/Async Control Flag | 0: Async, 1: Sync control for vel/torque modes. | 0 |
| `2010h` | 0 | Save to EEPROM | 1: Saves all RW/S parameters to EEPROM. | 0 |
| `2026h` | 1 | Alarm PWM Processing | 1: Short-circuits motor UVW on alarm. | 0 |
| `2026h` | 3 | I/O Emergency Stop Mode | 0: Lock shaft, 1: Release shaft. | 0 |
| `2026h` | 4 | Parking Mode | 1: Enables low-current parking mode. | 0 |
| `2026h` | 5 | Given Speed Resolution | Sets the resolution of the target speed value (1-10). | 1 |
| `2030h` | 2 | Input Port X0 Func. Sel. | 9: Sets INPUT1 as an emergency stop input. | 9 |
| `2030h` | 7 | Output Port B0 Func. Sel. | Controls left motor brake (0: Open, 1: Close). | 0 |
| `2030h` | 8 | Output Port B1 Func. Sel. | Controls right motor brake (0: Open, 1: Close). | 0 |

### CiA402 Motion Control Parameters (6000h)

| Index | Sub | Name | Description | Type | Attr. |
| :--- | :-: | :--- | :--- | :--- | :--- |
| `603Fh` | 0 | Error Code | Reports the current fault code. | U16 | RO |
| `6040h` | 0 | Controlword | Used to control the CiA402 state machine. | U16 | RW |
| `6041h` | 0 | Statusword | Reflects the current CiA402 state. | U16 | RO |
| `605Ah` | 0 | Quick Stop Option Code | Defines behavior on quick stop command. | I16 | RW |
| `6060h` | 0 | Modes of Operation | Sets the active control mode (1, 3, 4). | I8 | RW |
| `6061h` | 0 | Modes of Operation Display | Displays the current control mode. | I8 | RO |
| `6064h` | | Position Actual Value | Feedback of the current motor position (counts). | I32 | RO |
| `606Ch` | | Velocity Actual Value | Feedback of the current motor speed (Unit: 0.1 r/min). | I32 | RO |
| `6071h` | | Target Torque | Sets the target torque in Torque Mode (mA). | I16 | RW |
| `6077h` | | Torque Actual Value | Feedback of the current motor torque (Unit: 0.1A). | I16 | RO |
| `607Ah` | | Target Position | Sets the target position in Position Mode (counts). | I32 | RW |
| `6081h` | | Profile Velocity | Max speed for a position mode move (r/min). | U32 | RW |
| `6083h` | | Profile Acceleration | Acceleration for profile modes (ms). | U32 | RW |
| `6084h` | | Profile Deceleration | Deceleration for profile modes (ms). | U32 | RW |
| `60FFh` | | Target Velocity | Sets the target velocity in Velocity Mode (r/min). | I32 | RW |
