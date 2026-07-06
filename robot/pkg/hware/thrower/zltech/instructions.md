# ZLAC8030L V2.1 Servo Driver

## CANopen Communication Quick Start Guide

| Version | Description | Date |
| :--- | :--- | :--- |
| V2.00 | First edition | 2023/9/11 |
| V2.01 | Add 603F Speed setting error<br>Added the address for enabling the 202F 06 Speed Offset function | 2024/03/25 |

-----

*Shenzhen ZhongLing Technology Co., Ltd. TEL: +86-0755-29799302 FAX: +86-0755-2912 4283 WEB: [www.zirobotmotor.com](https://www.google.com/search?q=https://www.zirobotmotor.com)*

-----

## CATALOG

| | |
| :--- | :--- |
| **1. Outline** | 3 |
| **2. Wiring Connection** | 3 |
| 2.1 Basic Wiring Diagram | 3 |
| 2.2 CANopen Port | 4 |
| **3. Protocol Format** | 4 |
| 3.1 Communication Setting | 4 |
| 3.2 CANopen Basic Format | 4 |
| 3.3 SDO Basic Format | 4 |
| 3.4 Heartbeat Message | 5 |
| **4. Control Mode** | 6 |
| 4.1 Profile Velocity Mode | 6 |
| 4.2 Profile Relative Position Mode | 6 |
| 4.3 Profile Absolute Position Mode | 6 |
| 4.4 Profile Torque Mode | 7 |
| 4.5 General Command | 7 |
| 4.6 Emergency Stop Command | 7 |
| **5. Function Setting** | 7 |
| 5.1 Alarm PWM Processing Method | 7 |
| 5.2 Parking Mode | 8 |
| 5.3 Speed Resolution | 8 |
| 5.4 I/O Emergency Stop Processing Method | 8 |
| 5.5 Brake Function | 9 |
| **6. PDO Mapping Steps** | 10 |
| 6.1 TPDO Mapping | 10 |
| 6.2 RPDO Mapping | 11 |
| 6.3 Mapping Description | 11 |
| **7. CANopen Status Word** | 12 |
| 7.1 Profile Velocity Mode Status Word | 12 |
| 7.2 Profile Position Mode Status Word | 12 |
| 7.3 Profile Torque Mode Status Word | 13 |
| **8. Fault Code** | 13 |
| **9. Object Dictionary** | 14 |

-----

## 1\. Outline

This manual only gives a brief introduction to the most commonly used related concepts and precautions in the use of ZLAC8030L, so that users can understand the normal use of ZLAC8030L series products in the shortest time.

**Communication Standard followed by ZLAC8030L**

  * CAN 2.0A Standard
  * CANopen Standard protocol DS 301 V4.02
  * CANopen Standard protocol DS 402 V2.01

**Services supported by ZLAC8030L**

  * Support SDO service
  * Support PDO service: each slave station can be configured with up to 4 TxPDOs and 4 RxPDOS
  * Support NMT Slave service
  * Device monitor: support heartbeat message

## 2\. Wiring Connection

### 2.1 Basic Wiring Diagram

**(Image of a ZLTECH ZLAC8030L Digital AC Servo Driver with labels pointing to its various ports and connections)**

  * **Communication Cable**
  * **I/O Port**
  * **Signal cable**
  * **Auxiliary Power Supply**
  * **Bleed Resistor**
  * **Motor Power Cable**
  * **Power Cable**

**Note:** Motor power cable sequence is U (Yellow), V (Green), W (Blue).

### 2.2 CANopen Port

**Note:** There is only one set of CAN interface, if the user needs to connect multiple drives, please connect in parallel to CANL (pin1), CANH (pin2) and SGND (pin3), this drive communication is with isolation, the user needs to connect the ground signal SGND.

| Port | Pin | Symbol | Name | Function |
| :--- | :--- | :--- | :--- | :--- |
| | 1 | CANL | CAN | CAN/RS485 is an isolated |
| | 2 | CANH | | output and is recommended |
| | | | | when used while connecting to |
| | 3 | SGND | Communication | a common ground |
| | 4 | | common | |
| | | | ground | |
| | 4 | A | RS485 | |
| | 5 | B | | |

## 3\. Protocol Format

### 3.1 Communication Setting

Baud rate: 500K, ID: 4 (default)

### 3.2 CANopen Basic Format

**Note:** ZLAC8030L will send a `700+ID` NMT message when it is powered on. Receiving this message indicates successful communication. If this message is not received, please check the wiring connection and baud rate to ensure consistency, or power on again.

Example of a received NMT message:

  * **Frame ID:** `0x00000704`
  * **Format Type:** Data Frame Standard
  * **DLC:** `0x01`
  * **Data:** `00`

### 3.3 SDO Basic Format

| COB-ID | Byte0 | Byte1:2 | Byte3 | Byte4:7 |
| :--- | :--- | :--- | :--- | :--- |
| Frame ID | SDO Command Word | Object Index | Object Sub-Index | Data |

**Example:**
| COB-ID | Byte0 | Byte1 | Byte2 | Byte3 | Byte4 | Byte5 | Byte6 | Byte7 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Master → Slave (Driver)** |
| `0x604` | 23 | FF | 60 | 00 | 64 | 00 | 00 | 00 |
| **Slave (Driver) → Master** |
| `0x584` | 60 | FF | 60 | 00 | 00 | 00 | 00 | 00 |

#### 3.3.1 COB-ID Format

  * **Send frame ID:** `0x600` + Node address
  * **Return frame ID:** `0x580` + Node address

#### 3.3.2 Command word

| Command | Function Type | Data Length |
| :--- | :--- | :--- |
| 2F | Set M-\>S Request | 1 Byte |
| 2B | Set M-\>S Request | 2 Byte |
| 23 | Set M-\>S Request | 4 Byte |
| 60 | Set Feedback S-\>M Confirm | |
| 40 | Read M-\>S Request | 0 Byte |
| 80 | Read Fault S-\>M Answer | 4 Byte |

#### 3.3.3 Index and Data Form

Example command: `23 FF 60 00 64 00 00 00`

  * **Command:** `23`
  * **INDEX:** `FF 60` (The target speed index is FF 60, so the actual value is: 60 FF)
  * **Sub-Index:** `00`
  * **DATA:** `64 00 00 00` (The left and right target speed data is in the same format as the index)
  * **Byte Order:** Little Endian (Low bit in front, high bit in back).

### 3.4 Heartbeat Message

  * **Setting instruction:**
      * **Frame ID:** `604`
      * **Data:** `2B 17 10 00 E8 03 00 00` (time is 1000ms)
  * **Heartbeat message format:**

| Heartbeat Producer → Consumer |
| :--- | :--- |
| **COB-ID** | **Byte 0** |
| `0x700`+Node-ID | Status |

  * **Status description:**

| Status | Description |
| :--- | :--- |
| `$0 \times 00$` | Boot-up |
| `$0 \times 04$` | Stop Status |
| `$0 \times 05$` | Operation Status |
| `0x7F` | Pre-operation Status |

**Note:** ZLAC8030L is producer of heartbeat message.

## 4\. Control Mode

### 4.1 Profile Velocity Mode

| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 60 60 00 03 00 00 00` | `60 60 60 00 00 00 00 00` | Set velocity mode |
| `2B 40 60 00 06 00 00 00` | `60 40 60 00 00 00 00 00` | Enable |
| `2B 40 60 00 07 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `2B 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `23 FF 60 00 64 00 00 00` | `60 FF 60 00 00 00 00 00` | Set target speed 100rpm |
| `23 FF 60 00 9C FF FF FF` | `60 FF 60 00 00 00 00 00` | Set target speed -100rpm |
| `23 FF 60 00 32 00 00 00` | `60 FF 60 00 00 00 00 00` | Set target speed 50rpm |
| `23 FF 60 00 CE FF FF FF` | `60 FF 60 00 00 00 00 00` | Set target speed -50rpm |

### 4.2 Profile Relative Position Mode

| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 60 60 00 01 00 00 00` | `60 60 60 00 00 00 00 00` | Set position mode |
| `23 81 60 00 30 00 00 00` | `60 81 60 01 00 00 00 00` | Set max speed 60RPM |
| `2B 40 60 00 06 00 00 00` | `60 40 60 00 00 00 00 00` | Enable |
| `2B 40 60 00 07 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `2B 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `23 7A 60 00 00 7D 00 00` | `60 7A 60 00 00 00 00 00` | Set target positon 32000 |
| `2B 40 60 00 4F 00 00 00` | `60 40 60 00 00 00 00 00` | Start relative motion |
| `28 40 60 00 5F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `23 7A 60 00 00 83 FF FF` | `60 7A 60 00 00 00 00 00` | Set target positon -32000 |
| `28 40 60 00 4F 00 00 00` | `60 40 60 00 00 00 00 00` | Start relative motion |
| `28 40 60 00 5F 00 00 00` | `60 40 60 00 00 00 00 00` | |

### 4.3 Profile Absolute Position Mode

| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 60 60 00 01 00 00 00` | `60 60 60 00 00 00 00 00` | Set position mode |
| `23 81 60 00 30 00 00 00` | `60 81 60 01 00 00 00 00` | Set max speed 60 RPM |
| `2B 40 60 00 06 00 00 00` | `60 40 60 00 00 00 00 00` | Enable |
| `2B 40 60 00 07 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `28 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `23 7A 60 00 00 7D 00 00` | `60 7A 60 00 00 00 00 00` | Set target positon 32000 |
| `28 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | Start absolute motion |
| `2B 40 60 00 1F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `23 7A 60 00 00 83 FF FF` | `60 7A 60 00 00 00 00 00` | Set target positon -32000 |
| `2B 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | Start absolute motion |
| `28 40 60 00 1F 00 00 00` | `60 40 60 00 00 00 00 00` | |

### 4.4 Profile Torque Mode

| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 60 60 00 04 00 00 00` | `60 60 60 00 00 00 00 00` | Set torque mode |
| `2B 40 60 00 06 00 00 00` | `60 40 60 00 00 00 00 00` | Enable |
| `2B 40 60 00 07 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `2B 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | |
| `2B 71 60 00 E8 03 00 00` | `60 71 60 00 00 00 00 00` | Set target torque 1000mA/s |
| `2B 71 60 00 18 FC FF FF` | `60 71 60 00 00 00 00 00` | Set target torque -1000mA/s |

### 4.5 General Command

| Master Station (COB-ID:0x604) | Function Description |
| :--- | :--- |
| `2B 40 60 00 00 00 00 00` | Stop |
| `2B 40 60 00 80 00 00 00` | Clear Fault |
| `40 64 60 00 00 00 00 00` | Read motor encoder |
| `40 6C 60 00 00 00 00 00` | Read motor speed (Unit: 1RPM) |
| `40 77 60 00 00 00 00 00` | Read motor current (Unit: 0.1A) |
| `40 3F 60 00 00 00 00 00` | Read fault code |
| `40 25 20 00 00 00 00 00` | Read software version |
| `40 26 20 01 00 00 00 00` | Read motor temperature (Unit:0. 12) |

### 4.6 Emergency Stop Command

| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `28 40 60 00 02 00 00 00` | `60 40 60 00 00 00 00 00` | Motor stops and keep enabled status |
| `28 40 60 00 0F 00 00 00` | `60 40 60 00 00 00 00 00` | Motor Enable (Release Emergency Stop) |

**Attention:** After sending emergency stop command, user needs to send enable command to release the emergency stop status.

## 5\. Function Setting

### 5.1 Alarm PWM Processing Method

  * **Open Command:**

      * **Frame ID:** `604`
      * **Data:** `2B 2F 20 01 01 00 00 00`

  * **Close Command:**

      * **Frame ID:** `604`
      * **Data:** `2B 2F 20 01 00 00 00 00`

  * **Save To EEPROM:**

      * **Frame ID:** `604`
      * **Data:** `2B 09 20 00 02 00 00 00`

  * **Trigger Mechanism:** When enabling this function, driver will enter an alarm and short-circuit the motor's power UVW (after the motor power cable UVW short-circuit, it will generate resistance during motor's rotation).

  * **Function:** To prevent the robot from sliding instantly after motor alarms.

### 5.2 Parking Mode

  * **Open Command:**

      * **Frame ID:** `604`
      * **Data:** `2B 2F 20 04 01 00 00 00`

  * **Close Command:**

      * **Frame ID:** `604`
      * **Data:** `2B 2F 20 04 00 00 00 00`

  * **Trigger Mechanism:** When enabling this function, the motor output current will not exceed 3A.

  * **Function:** When the robot is charging or standby, enter this function to prevent the motor from over temperature problem.

### 5.3 Speed Resolution

  * **Setting Instruction:**

      * **Frame ID:** `604`
      * **Data:** `2B 2F 20 05 0A 00 00 00` (setting range: 0-10, 10 is hexadecimal)

  * **Save To EEPROM:**

      * **Frame ID:** `604`
      * **Data:** `2B 09 20 00 02 00 00 00`

  * **Rule:**

      * Set to A, output speed unit: `$1/10=0.1~RPM$`. Eg: target speed is 100 RPM, and the actual output is 10 RPM.
      * Set to 5, output speed units: `$1/5=0.2~RPM$`. Eg: target speed is 100 RPM, and the actual output is 20 RPM.
      * Set to 1, output speed unit: `$1/1=1$ RPM`. Eg: target speed is 100 RPM, and the actual output is 100 RPM.

  * **Trigger Mechanism:** After enabling the testing function, it must be saved and restarted to be effective.

  * **Function:** User could use more precise target speed control.

### 5.4 I/O Emergency Stop Processing Method

#### 5.4.1 Wiring Diagram J3, J6

**(Image showing the wiring between J3 and J6 connectors. J3 Pin 5 (GND) is connected to J6 Pin 3 (XCOM). J3 Pin 6 (5V) is connected to J6 Pin 7. A switch is connected between J6 Pin 1 (X0) and J6 Pin 7.)**

  * **I/O emergency stop processing method (CAN address; 202Fh 03)**
      * **0:** Lock shaft (Motor stops with holding force)
      * **1:** Release shaft (Turning off PWM output signal, motor is under free running status)
  * **Method a:** Set value of object `605Ah` to 5: When pressing the emergency stop button, the motor will stop according to the deceleration time and turn cut off the PWM control signal, to cut off the current supply to the motor.
  * **Method b:** Set value of object `605Ah` to 6: When pressing the emergency stop button, the motor will stop according to the emergency stop deceleration time and then turn off the PWM control signal, to cut off the current supply to the motor.
  * **Method c:** Set value of object `605Ah` to 7: When pressing the emergency stop button, the PWM control signal will be immediately turned off, and the motor will continue to run under inertia and gradually stop.

#### 5.4.2 CANopen Command Setting

  * **Enable input interface INPUT1 emergency stop function:** Frame ID: `604` Data: `2B 30 20 02 09 00 00 00`

  * **Enable input interface INPUT2 emergency stop function:** Frame ID: `604` Data: `2B 30 20 03 09 00 00 00`

  * **Save To EEPROM:** Frame ID: `604` Data: `2B 09 20 00 02 00 00 00 00`

  * **Turn on I/O emergency stop and release the shaft function command:** frame ID: `604` Data: `2B 2F 20 03 01 00 00 00`

  * **Turn off the I/O emergency stop and release the shaft function instruction:** frame ID: `604` Data: `2B 2F 20 03 00 00 00 00 00`

  * **Save To EEPROM:** frame ID: `604` Data: `2B 09 20 00 02 00 00 00 00`

  * **Trigger mechanism:** When this function is turned on, the motor will be in a disabled state after the driver triggers the external emergency stop (it is not turned on, but the motor will be in an enabled state after the external emergency stop is triggered).

  * **Function:** When the robot is in an abnormal state, it will trigger an external emergency stop.

### 5.5 Brake Function

#### 5.5.1 Wiring Diagram

**(Image of a circuit diagram showing the driver connected to an external DC brake. Connections are VDC/Brake+, Brake-, and GND.)**

**Note:** 20V-24V DC, brake doesn't have positive or negative poles, and could be wired freely.

#### 5.5.2 Brake Command Setting

  * **Release brake command:** Frame ID: `604` Data: `28 30 20 00 00 00 00`
  * **Close brake command:** Frame ID: `604` Data: `2B 30 20 0Е 01 00 00 00`
  * **Function:** If user's motor is equipped with an external electromagnetic brake, this command can be used to relese and close the brake.

## 6\. PDO Mapping Steps

### 6.1 TPDO Mapping

Configure `0x606C` as TPDO0, for transmission methods, use event trigger (254) or timer trigger (255) respectively.

**Event Trigger (254)**
| Master Station (COB-ID:0x604) | Slave Station(COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 00 1A 00 00 00 00 00` | `60 00 1A 00 00 00 00 00` | Clear TPDOO mapping |
| `23 00 1A 01 20 00 6C 60` | `60 00 1A 01 00 00 00 00` | Map 0x606C to 0x1A00 01 |
| `2F 00 18 02 FE 00 00 00` | `60 00 18 02 00 00 00 00` | Set TPDOO transmission method to event trigger |
| `2F 00 1A 00 01 00 00 00` | `60 00 1A 00 00 00 00 00` | Enable 1 TPDOO mapping |
| `2B 09 20 00 02 00 00 00` | `60 09 20 00 00 00 00 00` | Save parameters to EEPROM |

**Timer Trigger (255)**
| Master Station (COB-ID:0x604) | Slave Station (COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 00 1A 00 00 00 00 00` | `60 00 1A 00 00 00 00 00` | Clear TPDOO mapping |
| `23 00 1A 01 20 00 6C 60` | `60 00 1A 01 00 00 00 00` | Map 0x606C to 0x1A00 01 |
| `2F 00 18 02 FF 00 00 00` | `60 00 18 02 00 00 00 00` | Set TPDOO transmission method to timer trigger |
| `2B 00 18 05 E8 03 00 00` | `60 00 18 05 00 00 00 00` | Set inhibit time 500ms (unit: 0.5ms) |
| `2F 00 1A 00 01 00 00 00` | `60 00 1A 00 00 00 00 00` | Enable 1 TPDOO mapping |
| `2B 09 20 00 02 00 00 00` | `60 09 20 00 00 00 00 00` | Save parameters to EEPROM |

After the mapping is completed, send the NMT start command.

  * **NMT enable command format:**

      * **COB-ID:** `000`
      * **Data:** `$01+ID$` (`00` represents enabling PDO of all addresses)
      * **Enabling address 4:** Frame ID: `000`, Data: `01 04`
      * **Enabling all addresses:** Frame ID: `000`, Data: `01 00`

  * **TPDO upload format:**

| Slave Station (COB-ID:0x184) | Function Description |
| :--- | :--- |
| `01 02 03 04` | The data uploaded to 606C is 01 02 03 04 |

  * **NMT close command format:**
      * **COB-ID:** `000`
      * **Data:** `$80+ID$` (`00` represents closing PDO of all addresses)
      * **Closing address 4:** Frame ID: `000`, Data: `80 04`
      * **Closing all addresses:** Frame ID: `000`, Data: `80 00`
        **Note:** After closing, TPDO will stop uploading.

### 6.2 RPDO Mapping

Configure `0x60FF` as TPDO1, transmission method is event trigger (254).

| Mater Station(COB-ID:0x604) | Slave Station(COB-ID:0x584) | Function Description |
| :--- | :--- | :--- |
| `2F 01 16 00 00 00 00 00` | `60 01 16 00 00 00 00 00` | Clear RPDO1 mapping |
| `23 01 16 01 20 00 FF 60` | `60 01 16 01 00 00 00 00` | Map 0x60FF to 0x1601 01 |
| `2F 01 16 00 01 00 00 00` | `60 01 16 00 00 00 00 00` | Enable RPDO1 mapping |
| `28 09 20 00 02 00 00 00` | `60 09 20 00 00 00 00 00` | Save parameters to EEPROM |

After the mapping is completed, send the NMT start command.

  * **NMT enable command format:**

      * **COB-ID:** `000`
      * **Data:** `01+ID` (`00` represents enabling PDO of all addresses)
      * **Enabling address 4:** Frame ID: `000`, Data: `01 04`
      * **Enabling all addresses:** Frame ID: `000`, Data: `01 00`

  * **RPDO upload format:**

| Slave Station (COB-ID:0x304) | Function Description |
| :--- | :--- |
| `01 02 03 04` | Write 01 02 03 04 to 60FF |

  * **NMT close command format:**
      * **COB-ID:** `000`
      * **Data:** `80+ID` (`00` represents closing PDO of all addresses)
      * **Closing address 4:** Frame ID: `000`, Data: `80 04`
      * **Closing all addresses:** Frame ID: `000`, Data: `80 00`
        **Note:** After closing, sending RPDO will be invalid.

### 6.3 Mapping Description

The meaning of "20" in the mapping instruction `23 00 1A 01 20 00 60 60`:
**Note:** `20` represents the number of digits of the mapped index data type (converting hexadecimal "20" to decimal means "32").

Example: `606Ch` Actual speed feedback is a 32-bit value (I32).

## 7\. CANopen Status Word

### 7.1 Profile Velocity Mode Status Word (6041h)

| Bit | Definition | Function Description |
| :--- | :--- | :--- |
| Bit0\~Bit3 | `6040=0`: XXXX XXXX Xxxx 0000<br>`6040=6`: XXXX XXXX Xxxx 0001<br>`6040=7`: XXXX XXXX Xxxx 0011<br>`6040=F`: XXXX XXXX Xxxx 0111 | |
| Bit5 | Command Emergency Stop | 0: Driver is in emergency stop status;<br>1: Driver is not in emergency stop state; |
| Bit10 | | 0: Speed is not in place;<br>1: Speed is in place; |
| Bit12 | | 0: Speed is not 0 RPM;<br>1: The speed is 0 RPM; |
| Bit14 | | 0: The motor is stopping;<br>1: The motor is running; |
| Bit15 | | 0: Not in external emergency stop state;<br>1: In external emergency stop state; |

### 7.2 Profile Position Mode Status Word (6041h)

| Bit | Definition | Function Description |
| :--- | :--- | :--- |
| Bit0\~Bit3 | `6040=0`: XXXX XXXX Xxxx 0000<br>`6040=6`: XXXX XXXX XXxx 0001<br>`6040=7`: XXXX XXXX Xxxx 0011<br>`6040=F`: XXXX XXXX XXxxx 0111 | |
| Bit5 | Command Emergency Stop | 0: Driver is in emergency stop status;<br>1: Driver is not in emergency stop state; |
| Bit10 | | 0: Target position is not reached;<br>1: Target location is reached; |
| Bit12 | | 0: The target location is not valid;<br>1: The target location is valid; |
| Bit13 | (It's judged based on the threshold of driver deviation) | 0: The motor is not running in place;<br>1: The motor is running in place; |
| Bit14 | | 0: The motor is stopping;<br>1: The motor is running; |
| Bit15 | | 0: Not in external emergency stop state;<br>1: In external emergency stop state; |

### 7.3 Profile Torque Mode Status Word (6041h)

| Bit | Definition | Function Description |
| :--- | :--- | :--- |
| Bit0\~Bit3 | `6040=0`: XXXX XXXX Xxxx 0000<br>`6040=6`: XXXX XXXX Xxxx 0001<br>`6040=7`: XXXX XXXX xxxx 0011<br>`6040=F`: XXXX XXXX XXxxx 0111 | |
| Bit5 | Command Emergency Stop | 0: Driver is in emergency stop status;<br>1: Driver is not in emergency stop state; |
| Bit10 | | 0: The target torque is not reached;<br>1: Target torque is reached; |
| Bit14 | | 0: The motor is stopping;<br>1: The motor is running; |
| Bit15 | | 0: Not in external emergency stop state;<br>1: In external emergency stop state; |

## 8\. Fault Code

| Index | Fault code | Description | Troubleshooting |
| :--- | :--- | :--- | :--- |
| | `0x0000h` | No error | Driver is normal. |
| | `0x0001h` | Over-voltage | 1. Power supply voltage is too high<br>2. Excessive back electromotive force (it is recommended to add a bleeder circuit) |
| | `0x0002h` | Under-voltage | 1. Power supply voltage is too low<br>2. Check if the wiring connector is correct<br>3. Check if the motor parameters are correct |
| | `0x0004h` | Over-current | 1. Instantaneous current is too high<br>2. Motor power cable is loose |
| | `0x0008h` | Overload | 1. Check if the motor cable is loose<br>2. Check if the wiring and motor parameters are correct<br>3. Motor is stall<br>4. Motor or driver's problem |
| `603Fh` | `0x0020h` | Encoder value is out of tolerance | 1. Motor is stall<br>2. Encoder's problem |
| | `0x0080h` | Reference voltage error | Reference voltage circuit issue |
| | `0x0100h` | EEPROM read and write error | 1. Firmware is upgraded (needs to make factory settings)<br>2. EEPROM circuit is damaged |
| | `0x0200h` | Hall error | 1. Check if the motor cable is loose<br>2. Motor's problem<br>3. Driver's problem |
| | `0x0400h` | motor temperature is too high. | 1. The motor current is too high (it is recommended to monitor motor's actual current and temperature, and reduce the current in real-time control)<br>2. Motor's thermistor is damaged<br>3. Driver's circuit is damaged |
| | `0x1000h` | Encoder error | 1. Check if the motor encoder cable is loose<br>2. Check if the motor encoder cable is disconnected |
| | `0x2000h` | Motor speed setting error | The given speed exceeds the set rated speed |

## 9\. Object Dictionary

**Note:**

  * U16 means unsigned 16 bits
  * I16 means signed 16 bits
  * U32 means unsigned 32 bits
  * I32 means signed 32 bits

| Index | Sub-Index | Name | Description | Type | Attribute | PDO Mapping | Default |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **CiA301 Basic Communication Parameter Group** |
| 1000h | 00 | Equipment type | This device supports CIA301, CIA402 protocol | U32 | RO | NO | 0X00040192 |
| 1001h | 00 | Error register | Driver current error status | U8 | RO | NO | 0 |
| 1005h | 00 | Synchronous message COB identifier | Synchronous message COB identifier | U32 | RW | NO | 0x80 |
| 1009h | 00 | Hardware version | Hardware version | U16 | RO | NO | |
| 100Ah | 00 | Software version | Software version | U16 | RO | NO | |
| 1014h | 00 | COB-ID emmergency | COB-ID emmergency | U32 | RW | NO | 0x80 |
| 1017h | 00 | Producer heartbeat interval | Producer heartbeat interval, unit: ms | U16 | RW/S | NO | 0 |
| 1018h | 00 | Manufacturer Information | Sub-index | U8 | RO | NO | 5 |
| | 01 | | Vendor ID | U32 | RO | NO | 0x0100 |
| | 02 | | Product Code | U32 | RO | NO | 0x0001 |
| 1200h | 00 | SDO Server Parameters | Number of sub-indexes | U8 | RO | NO | 2 |
| | 01 | | COB-ID (Slave station receives) | COB-ID (Slave station receives) | U32 | RO | NO | 600h+Node-ID |
| | 02 | | COB-ID (Slave station sends) | COB-ID (Slave station sends) | U32 | RO | NO | 580h+Node-ID |
| 1400h | 00 | RPDO Communication Parameter | Number of sub-indexes | U8 | RO | NO | 5 |
| | 01 | RPDOO-COB-ID | Identifier COB-ID | U32 | RW/S | NO | 200+Node-ID |
| | 02 | Transmission type | Transmission type | U8 | RW/S | NO | FFh |
| | 03 | Prohibition time | Prohibition time | U16 | RW/S | NO | 0 |
| | 04 | Maintain | Maintain | U8 | RW | NO | 0 |
| | 05 | Event timer | Event timer | U16 | RW/S | NO | 0 |
| 1401h | 00 | RPDO Communication Parameter | Number of sub-indexes | U8 | RO | NO | 5 |
| | 01 | RPDO1-COB-ID | Identifier COB-ID | U32 | RW/S | NO | 300+Node-ID |
| | 02 | Transmission type | Transmission type | U8 | RW/S | NO | FFh |
| | 03 | Prohibition time | Prohibition time | U16 | RW/S | NO | 0 |
| | 04 | Maintain | Maintain | U8 | RW | NO | 0 |
| | 05 | Event timer | Event timer | U16 | RW/S | NO | 0 |
| ... | ... | ... | ... | ... | ... | ... | ... |
| **Factory Custom Parameter Group** |
| 2000h | 00 | Driver and host communication offline time | Communication offline time setting. Unit: ms Range: 0-32767 | U16 | RW/S | YES | 1000 |
| 2003h | 00 | Input signal status | 2 input signal level status; BitO-Bit1: XO-X1 input level status | U16 | RO | YES | 0 |
| 2004h | 00 | Output signal status | 2 output signal level status; Bito-Bit1: YO-Y1 Bit2-Bit3: BO-B1 output status | U16 | RO | YES | 0 |
| 2005h | 00 | Clear feedback position in profile position mode | Used to clear feedback position. 0: invalid; 1: clear the feedback position; Not saved. | U16 | RW | YES | 0 |
| 2006h | 00 | In absolute position mode, clear the current position | Used to clear the current position in absolute position mode. 0: invalid; 1: clear the current position; | U16 | RW | YES | 0 |
| ... | ... | ... | ... | ... | ... | ... | ... |
| **CIA 402 Parameter Group** |
| 603Fh | 00 | Driver last fault code | Factory-defined drive error conditions. (See list above) | U16 | RO | YES | 0 |
| 6040h | 00 | Control word | Control word | U16 | RW | YES | 0 |
| 6041h | 00 | Status word | Status word | U16 | RO | YES | 0 |
| 605Ah | 00 | Quick stop code | Driver processing method after quick stop command. 5: stop normally, maintain quick stop state; 6: decelerate suddenly to stop, maintain quick stop state; 7: emergency stop, maintain quick stop state; | I16 | RW | NO | 5 |
| 605Bh | 00 | Close operation code | Driver processing method after close command. 0: invalid; 1: stop normally, turn to ready to switch on state; | I16 | RW | NO | 1 |
| 605Ch | 00 | Disable operation code | Driver processing method after disable operation command. 0: Invalid; 1: stop normally, switch to switched on state; | I16 | RW | NO | 1 |
| 605Dh | 00 | Halt control register | Driver processing method after the control word Halt command. 0: stop normally, maintaining Operation Enabled state; 2: decelerate suddenly stop, maintain Operation Enabled state; 3: emergency stop, maintain Operation Enabled state; | I16 | RW | NO | 1 |
| 6060h | 00 | Operating mode | 0: undefined; 1: profile position mode; 3: profile velocity mode; 6: profile torque mode; | I8 | RW | YES | 0 |
| 6061h | 00 | Operating mode status | 0: undefined; 1: profile position mode; 3: profile velocity mode; 6: profile torque mode; | I8 | RO | YES | 0 |
| 6064h | 00 | Actual position feedback | Actual position feedback, unit: counts; | I32 | RO | YES | 0 |
| 606Ch | 00 | Actual speed feedback | Current motor speed, Unit: 1r/min | I32 | RO | YES | 0 |
| 6071h | 00 | Target torque | Unit: mA; Range: -30000\~30000; | I16 | RW | YES | 0 |
| 6074h | 00 | Real-time target torque | Unit: mA; Range:-300\~300; | I16 | RO | YES | 0 |
| 6077h | 00 | Real-time torque feedback | Unit: 0.1A; Range: -30000\~30000; | I16 | RO | YES | 0 |
| 607Ah | 00 | Target position | Range of total pulses operated in position mode: Relatively: -0x7FFFFFFF\~0x7FFFFFFF; Absolute: -0x3FFFFFFF\~0x3FFFFFFF: | I32 | RW | YES | 0 |
| 6081h | 00 | Max speed | Speed in profile position mode; Range: 1-1000r/min; | U32 | RW | YES | `$120r/min$` |
| 6082h | 00 | Start/stop speed in profile position mode | Start/stop speed in profile position mode; Range: 1-1000r/min; | U32 | RW | YES | `$1r/min$` |
| 6083h | 00 | S-shaped acceleration time | acceleration time; Range: 0-32767ms; | U32 | RW | YES | 500ms |
| 6084h | 00 | S-shaped Deceleration time | Range: 0-32767ms; | U32 | RW | YES | 500ms |
| 6085h | 00 | deceleration time | Deceleration time; Range: 0-32767ms; | U32 | RW | YES | 10ms |
| 6087h | 00 | Torque slope | Current 1000/second; Unit: mA/s; | U32 | RW | YES | 300ms |
| 60FFh | 00 | Target speed | Target speed in profile velocity mode; Range: -1000\~1000r/min; | I32 | RW | YES | 0 |