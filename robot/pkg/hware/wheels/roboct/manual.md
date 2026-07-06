RoboCT Servo Drive Communication Protocol (Version 2.1)

Hangzhou RoboCT Technology Development Co., Ltd.
(Applicable for DC/AC servo drives)

1.1 CANopen Main Documentation
	•	CiA Draft Standard 301 (Version 4.02): Application Layer and Communication Profile
	•	CiA Draft Standard 402 (Version 1.2): CANopen Device Profile

These are the main CANopen specification documents relevant to this drive.

1.2 Terms and Abbreviations
	•	CAN: Controller Area Network (standard fieldbus for automation).
	•	CiA: CAN in Automation (international users’ and manufacturers’ association).
	•	COB: Communication Object. A basic message unit on a CAN network. Data transmitted in a COB goes to all nodes on the network; it is part of the CAN message frame.
	•	LMT: Layer Management. A CAN application layer service element used to configure parameters for each layer in a CAN node.
	•	NMT: Network Management. A CAN application layer service element responsible for initial network configuration, node control, and error handling.
	•	OD: Object Dictionary. A drive’s internal database of all communication objects (parameters). Changing a parameter in the OD means issuing an instruction to the drive (via the panel or CAN).
	•	PDO: Process Data Object. Used for time-critical data (e.g. control commands, status words, sensor values).
	•	RO: Read-only access (for an OD entry).
	•	RW: Read/write access.
	•	WO: Write-only access.
	•	SDO: Service Data Object. Used for non-time-critical data transfers.

1.3 CAN Summary

The CAN bus is a serial communication protocol featuring high speed and strong noise immunity, suitable for noisy industrial environments. However, CAN itself only defines the Physical (layer 1) and Data Link (layer 2) layers of the ISO/OSI model. Application-layer protocols must be defined separately. Over time, several CAN application-layer protocols have been developed; the most popular are CANopen, DeviceNet, and SAE J1939.

1.4 CANopen Summary

CANopen is a high-level communication protocol built on CAN. It includes a set of communication protocols (like NMT, SDO, PDO, SYNC, etc.) and device profiles for different types of devices. CANopen uses periodic and event-driven communication, which reduces bus load and ensures short reaction times. Even at lower baud rates, high performance can be achieved, reducing electromagnetic interference and cabling costs.

A CANopen device (like the RoboCT drive) contains an ISO 11898-compliant CAN transceiver and controller. The RoboCT smart drive integrates CANopen so users can communicate with it via a PC, PLC, HMI, or other controllers. The drive implements the CiA DS-301 (communication profile) and CiA DS-402 (device profile for drives).

Note: CANopen communication requires correct configuration of SDO, PDO, SYNC, Emergency, NMT, and Heartbeat objects, especially their COB-IDs (identifiers). Misconfigured IDs can result in loss of communication. (In the descriptions below, “RX” means the device receives data (another device transmits), and “TX” means the device transmits to others.)

2.1 CAN Hardware Interface

The drive’s CAN interface is available on ports CN5 and CN6 (communication connectors). For example, on the AC servo drive, the pin definitions are as follows (others are similar; see product manual for differences):

Pin (CN5/CN6)	Definition
1	485A
2	485B
3	SGND (Signal Ground)
4	RS232-RXD
5	RS232-TXD
6	+5V (Sensor supply)
7	CANH (CAN High)
8	CANL (CAN Low)

Figure 2.1 – Communication interface connectors CN5 and CN6 (diagram on page 9). The CAN bus wiring should include a 120 Ω termination resistor at each end of the bus (the drive does not have an internal terminator, so the user must add them externally). Figure 2.2 – CAN bus connection circuit (diagram on page 9).

2.2 CAN Baud Rate and Node Configuration

The CAN communication baud rate and node ID can be configured via the PC-based monitoring software or the drive’s front panel. After changing these, the drive must be restarted for the new settings to take effect. In the drive’s parameter table, parameter Pr.299 selects the CAN protocol: Pr.299 = 0 for the proprietary MOTEC protocol, Pr.299 = 1 for the CANopen protocol.

Supported Baud Rates: The drive’s parameter Pr.13 sets the CAN baud rate. The value of Pr.13 corresponds to baud rates as shown:

Pr.13 Value	CAN Baud Rate
0	50 kbit/s
1	125 kbit/s
2	250 kbit/s
3	500 kbit/s
4	750 kbit/s
5	1000 kbit/s

Node ID: The drive’s node ID is set by parameter Pr.11 (changes take effect after power cycle). The current Node ID can also be read from object 0x100B.
	•	Object 0x1000 – Device Type: Identifies the device profile type.
Index: 0x1000
Name: Device Type (Manufacturer device type)
Object Code: VAR
Data Type: UINT32
Access: RO (read-only)
PDO Mapping: NO
Default Value: 402 (indicates CiA DS-402 device profile)
	•	Object 0x100B – Node ID: Reflects the drive’s current CANopen node ID.
Index: 0x100B
Name: Node ID
Object Code: VAR
Data Type: UINT8
Access: RO
PDO Mapping: NO
Value Range: 0…255
Default Value: 1

2.3 Manufacturer Information

Objects 0x1008, 0x1009, and 0x100A provide manufacturer-specific identification:

Index	Name	Description
0x1008	Manufacturer device name	Device name/model
0x1009	Manufacturer hardware version	Hardware version information
0x100A	Manufacturer software version	Software version information

	•	Object 0x1008 – Manufacturer Device Name:
Index: 0x1008
Name: Manufacturer device name
Object Code: VAR
Data Type: VISIBLE_STRING (or text)
Access: RO
PDO Mapping: NO
Default Value: (varies by model)
Description: Indicates the product series. For example:
411XX-BLD (servo drive model),
412XX-SLD,
413XX-MLD,
414XX-HLD,
420XX-MBD (hollow cup drive),
421XX-QBLD,
422XX-XBLD,
423XX-DBD,
424XX-MBLD,
425XX-EBLD,
5XXXX- (AC servo drive series). (“XX” are model-specific numbers.)
	•	Object 0x1009 – Manufacturer Hardware Version:
Index: 0x1009
Name: Manufacturer hardware version
Object Code: VAR
Data Type: UINT16
Access: RO
PDO Mapping: NO
Default Value: (not fixed)
Description: The hardware version number. For example, a value 30201 corresponds to version 3.02.01. If object 0x1009 contains 10203, that represents version 1.02.03.
	•	Object 0x100A – Manufacturer Software Version:
Index: 0x100A
Name: Manufacturer software version
Object Code: VAR
Data Type: UINT16
Access: RO
PDO Mapping: NO
Default Value: (not fixed)
Description: The software version number. For example, if 0x100A contains 10203, that represents software version 1.02.03.

3.1 SDO (Service Data Object) Configuration

CANopen communication uses several types of objects. SDOs are used for configuration and less time-critical data. PDOs handle real-time data, SYNC coordinates network actions, Emergency signals error events, NMT manages network state, and Heartbeat monitors node health. Each of these has associated object dictionary entries which must be configured correctly (especially COB-IDs).

Note: In the descriptions below, “RX” (receive) is from the perspective of this drive (meaning another device transmits to the drive’s COB-ID), and “TX” (transmit) means the drive sends out data.

3.1.1 SDO COB-IDs

The object 0x1200 defines the SDO communication parameters for the drive’s SDO server. This object is an array containing the COB-IDs used for SDO communication:
	•	Object 0x1200 – Server SDO Parameters: (Array with 3 entries)
	•	Sub-Index 0: Number of entries
	•	Access: RO
	•	Data Type: UINT8
	•	Default: 2 (two entries in this object)
	•	Sub-Index 1: COB-ID Client->Server (SDO Rx ID)
	•	Access: RO
	•	Data Type: UINT32
	•	Default: 0x600 + NodeID
	•	Sub-Index 2: COB-ID Server->Client (SDO Tx ID)
	•	Access: RO
	•	Data Type: UINT32
	•	Default: 0x580 + NodeID

3.1.2 SDO Abort Codes

If an SDO transfer is aborted, the drive returns an abort code to the SDO client. The SDO abort message is 8 bytes:
	•	Byte 0: SDO Command Specifier (contains 0x80 indicating an abort)
	•	Bytes 1–2: Object Index that caused the abort
	•	Byte 3: Sub-Index that caused the abort
	•	Bytes 4–7: Abort Code (32-bit value describing the error)

Common SDO Abort Codes:
	•	0x05030000: Toggle bit not alternated
	•	0x05040001: Client/Server command specifier not valid or unknown
	•	0x05030005: Out of memory
	•	0x06010000: Unsupported access to an object
	•	0x06010001: Attempt to read a write-only object
	•	0x06010002: Attempt to write a read-only object
	•	0x06020000: Object does not exist in the OD
	•	0x06040041: Object cannot be mapped to PDO
	•	0x06040042: Number and length of mapped objects exceed PDO length
	•	0x06040043: General parameter incompatibility
	•	0x06040047: General internal incompatibility in device
	•	0x06060000: Object access failed due to hardware error
	•	0x06060010: Data type does not match, length of service parameter does not match
	•	0x06060012: Data type does not match, length of service parameter too high
	•	0x06060013: Data type does not match, length of service parameter too low
	•	0x06090011: Sub-index does not exist
	•	0x06090030: Value range exceeded (on write)
	•	0x06090031: Value too large (on write)
	•	0x06090032: Value too small (on write)
	•	0x06090036: Maximum value less than minimum value
	•	0x08000000: General error
	•	0x08000020: Data cannot be transferred or stored to the application
	•	0x08000021: Data cannot be transferred because of local control
	•	0x08000022: Data cannot be transferred because of present device state
	•	0x08000023: Object dictionary not present or dynamic creation failed (e.g. OD generated from file and file is corrupted)

3.2 PDO (Process Data Object) Configuration

The drive supports up to 3 Receive PDOs (RPDOs) and 3 Transmit PDOs (TPDOs). PDOs allow real-time data (such as control commands or sensor feedback) to be transmitted with minimal overhead. Each PDO must be configured with a COB-ID and a mapping of OD entries to include.

3.2.1 RPDO (Receive PDO) Parameters

Objects 0x1400–0x1402 define the communication parameters for RPDOs 1, 2, 3 respectively. Each is an array with the COB-ID and the Transmission Type for that PDO:
	•	Object 0x1400/0x1401/0x1402 – Receive PDO (1–3) Parameters: (Array, 3 sub-indices)
	•	Sub-Index 0: Number of entries (RO, UINT8, default 2)
	•	Sub-Index 1: COB-ID used by RPDO (RX COB-ID) – RW (UINT32)
	•	Allowed ranges:
• For 0x1400: 0x201–0x27F (default 0x200 + NodeID)
• For 0x1401: 0x301–0x37F (default 0x300 + NodeID)
• For 0x1402: 0x401–0x47F (default 0x400 + NodeID)
	•	Sub-Index 2: Transmission Type – RW (UINT8)
	•	Allowed range: 0…255 (default 254 for RPDOs, which is asynchronous/event-driven)

Transmission Type values (for PDOs):
	•	0: Synchronous – device sends PDO upon receiving a SYNC if the PDO’s transmission trigger condition is met at that moment (manufacturer-specific synchronous).
	•	1–240: Synchronous – device sends PDO every Nth SYNC (if N=1, every SYNC; N=2, every 2nd SYNC; etc.).
	•	254 or 255: Asynchronous – device sends PDO when an event occurs. If an Event Timer is set, the PDO is sent periodically every configured interval.

Note: Transmission Type 254 is typically “event-driven” (like 255) in this context, and 255 is event-driven with an event timer.

Objects 0x1600–0x1602 define the mapping for RPDOs 1–3. These specify which OD entries are transmitted in each RPDO and in what order:
	•	Object 0x1600/0x1601/0x1602 – Receive PDO (1–3) Mapping: (Array, up to 4 mapped objects)
	•	Sub-Index 0: Number of mapped objects (RW, UINT8, allowed 0…4, default 0)
	•	Sub-Index 1–4: Mapped object entries (RW, UINT32, default 0)
	•	Each mapping entry is 32 bits: 0xOOOSSLL, where:
OOOO = OD Index, SS = Sub-index, LL = Length in bits.
	•	Example: to map Controlword (0x6040, sub-index 0) which is 16-bit, the entry would be 0x60400010 (index 0x6040, sub-index 0x00, length 0x10 in hex = 16 decimal).
	•	Each RPDO can carry up to 64 bits of data in total (4 entries of 16 bits each, for instance).

3.2.2 TPDO (Transmit PDO) Parameters

Objects 0x1800–0x1802 define the communication parameters for TPDOs 1, 2, 3. Each is an array with COB-ID, Transmission Type, Inhibit Time, Event Timer, etc.:
	•	Object 0x1800/0x1801/0x1802 – Transmit PDO (1–3) Parameters: (Array, 5 sub-indices)
	•	Sub-Index 0: Number of entries (RO, UINT8, default 5)
	•	Sub-Index 1: COB-ID used by TPDO (TX COB-ID) – RW (UINT32)
	•	Allowed ranges:
• 0x1800: 0x181–0x1FF (default 0x180 + NodeID)
• 0x1801: 0x281–0x2FF (default 0x280 + NodeID)
• 0x1802: 0x381–0x3FF (default 0x380 + NodeID)
	•	Sub-Index 2: Transmission Type – RW (UINT8, allowed 0…255, default 255 for asynchronous TPDOs)
	•	(Transmission Type meanings are the same as described above for RPDOs.)
	•	Sub-Index 3: Inhibit Time – RW (UINT16, units 100 µs, default 0)
	•	A nonzero Inhibit Time introduces a minimum delay (Inhibit Time * 100 µs) after a TPDO is sent, before another of the same TPDO can be sent. This helps avoid flooding the bus if the data changes very rapidly.
	•	Sub-Index 4: Event Timer – RW (UINT16, units ms, default 0)
	•	If Transmission Type = 254 or 255 (event-driven), and Event Timer is set >0, the TPDO will be transmitted periodically at this interval in addition to event-based sending. (If set to 0, only event-triggered sending is used.)

Transmission Type values for TPDOs: (same as RPDO, see above)
	•	0: Synchronous (manufacturer-specific behavior; typically not used for TPDOs in standard DS402 profile).
	•	1–240: Synchronous (send on nth SYNC).
	•	254: Asynchronous (device sends on internal event; optional Event Timer).
	•	255: Asynchronous (device sends on internal event; optional Event Timer).

Objects 0x1A00–0x1A02 define the mapping for TPDOs 1–3, analogous to the RPDO mapping objects:
	•	Object 0x1A00/0x1A01/0x1A02 – Transmit PDO (1–3) Mapping: (Array, up to 4 mapped objects)
	•	Sub-Index 0: Number of mapped objects (RW, UINT8, allowed 0…4, default 0)
	•	Sub-Index 1–4: Mapped object entries (RW, UINT32, default 0)
	•	Format of each entry is the same 0xOOOSSLL as described above for RPDO mapping.
	•	Example: to map Statusword (0x6041, sub-index 0, 16-bit) to a TPDO, set Sub-Index0 = 1, Sub-Index1 = 0x60410010.
	•	Each TPDO can carry up to 64 bits of data total.

3.3 SYNC Object

Synchronization (SYNC) is a broadcast message (COB-ID 0x80 by default) that can be used to coordinate PDO transfers on the network. The drive acts as a SYNC consumer (not generator) in typical setups. SYNC has the highest priority to ensure timely network synchronization and carries no data bytes (just the COB-ID).
	•	Object 0x1005 – SYNC COB-ID:
Index: 0x1005
Name: Sync message COB-ID
Object Code: VAR
Data Type: UINT32
Access: RO
PDO Mapping: NO
Default Value: 0x80 (the COB-ID used for the SYNC message)

(This object is fixed to 0x80 in this implementation, meaning the drive expects SYNC at COB-ID 0x80.)

3.4 Emergency Object

When the drive encounters a fault, it sends out an Emergency Message (EMCY) with a predefined COB-ID and 8-byte data. The EMCY COB-ID is 0x80 + NodeID for this drive (this is automatically configured).
	•	Object 0x1014 – Emergency COB-ID:
Index: 0x1014
Name: Emergency COB-ID
Object Code: VAR
Data Type: UINT32
Access: RO
PDO Mapping: NO
Default Value: 0x80 + NodeID (the CAN-ID used for Emergency messages from this node)

When a new error occurs on the drive, it sends one Emergency message. It will not send another for the same error until a new error event happens (to avoid flooding). The Emergency message data layout is:
	•	Bytes 0–1: Error Code (low 16 bits)
	•	Bytes 2–3: Error Code (high 16 bits)
	•	Byte 4: Error Register (8-bit, corresponds to object 0x1001)
	•	Bytes 5–7: Manufacturer-specific error field (unused in this drive, set to 0)

The Error Code corresponds to the detailed fault code (also accessible via object 0x200B). The Error Register (object 0x1001) provides a summary of error categories.
	•	Object 0x1001 – Error Register:
Index: 0x1001
Name: Error register
Object Code: VAR
Data Type: UINT8
Access: RO
PDO Mapping: NO
Default Value: 0
Description: This register’s bits indicate error categories as follows:
Bit 0: Generic error
Bit 1: Current error
Bit 2: Voltage error
Bit 3: Temperature error
Bit 4: Communication error
Bit 5: Device profile specific error
Bit 6: Reserved
Bit 7: Manufacturer specific error
(If a bit is 1, an error of that category is present.)

3.5 NMT Service and Heartbeat

NMT (Network Management) is used by the CANopen master to control the state of slave nodes. NMT commands are broadcast with COB-ID 0 (zero) and consist of 2 bytes: a command specifier and a node ID (the node ID 0 means “apply to all nodes”). Common NMT commands include:

Command Byte	Description (Action)	Target State
0x01	Start Remote Node (enable PDOs/SDOs)	Operational
0x02	Stop Remote Node (disable comm except heartbeat/NMT)	Stopped
0x80	Enter Pre-Operational (enable SDOs only)	Pre-Operational
0x81	Reset Node (application reset: resets variables to init values)	Reset Application
0x82	Reset Communication (reinitialize CANopen stack)	Reset Communication

Heartbeat: Each node can produce a heartbeat message to indicate it is alive. The heartbeat is a single byte published at a regular interval (configured per node) on COB-ID 0x700 + NodeID. The master monitors these to detect node failures. The heartbeat byte values are:
	•	0: Boot-up (node just started and not yet in operational state)
	•	4: Stopped
	•	5: Operational
	•	127: Pre-Operational

The drive’s Producer Heartbeat interval is set via object 0x1017. If set to 0, the drive will not produce heartbeat messages.
	•	Object 0x1017 – Producer Heartbeat Time:
Index: 0x1017
Name: Producer heartbeat time
Object Code: VAR
Data Type: UINT16 (in milliseconds)
Access: RW
PDO Mapping: NO
Value Range: 0…65535 ms
Default Value: 0 (heartbeat disabled by default)

(Setting a nonzero value causes the drive to transmit a heartbeat every that many milliseconds.)

Node Guarding (Life Guarding): An older method (now largely replaced by heartbeat) where the master polls each slave with a Remote Frame (RTR) on COB-ID 0x700 + NodeID. The slave responds with a message containing a toggle bit and the node’s state. The toggle bit alternates 0/1 on each reply to ensure the response is fresh. The slave’s response byte: bit7 = toggle, bits6–0 = state value (with same codes as heartbeat). If the master doesn’t receive a response within a timeout, it considers the node lost.

Node guarding parameters in the OD:
	•	Object 0x100C – Guard Time:
Index: 0x100C
Name: Guard Time
Object Code: VAR
Data Type: UINT16 (ms)
Access: RW
PDO Mapping: NO
Default Value: 0
Description: The period at which the master sends remote frame requests for node guarding. (If 0, node guarding is disabled.)
	•	Object 0x100D – Life Time Factor:
Index: 0x100D
Name: Life Time Factor
Object Code: VAR
Data Type: UINT8
Access: RW
PDO Mapping: NO
Default Value: 0
Description: The multiple of the Guard Time that the master will wait for a response. If no response in GuardTime * LifeTimeFactor, the node is considered offline. (If set to 0, life guarding is disabled.)

4.1 Save and Restore Parameters

The drive supports storing its parameters to non-volatile memory and restoring defaults via standardized objects:
	•	Object 0x1010 – Store Parameters (Save to Flash):
This object is an array (often called Store EDS in CiA). Writing a specific sub-index triggers saving.
	•	Sub-Index 0: Number of entries (UINT8, RO, default 1)
	•	Sub-Index 1: Save Command (UINT16, RW, default 0) – Write 1 to this sub-index to save all parameters to Flash. After execution, the value automatically resets to 0.
	•	Object 0x1011 – Restore Default Parameters:
Similar structure to 0x1010. Writing 1 to sub-index 1 restores default settings from Flash and resets the sub-index to 0 after execution.
	•	Sub-Index 0: Number of entries (UINT8, RO, default 1)
	•	Sub-Index 1: Load Default Command (UINT16, RW, default 0) – Write 1 to restore defaults.

5.1 Drive State Machine

The drive’s control follows the CiA 402 state machine, with states like Switch On Disabled, Ready to Switch On, Operation Enabled, Quick Stop Active, Fault, etc. The Controlword (object 0x6040) is used by the controller to command state transitions, and the Statusword (object 0x6041) is used by the drive to report its current state.

The state machine can be conceptualized in three major groups: “Power Disabled” (drive is not enabled), “Power Enabled” (drive is enabled and can operate), and “Fault” (drive has encountered a fault). Transitions between states are controlled by specific bit patterns in the Controlword set by the master.

Figure 5.1 – Drive status and state transition diagram (page 28). – This diagram shows the CiA 402 state machine. The states are: Not Ready to Switch On, Switch On Disabled, Ready to Switch On, Switched On, Operation Enabled, Quick Stop Active, Fault Reaction Active, and Fault. The Controlword bits 0,1,2,3, and 7 are primarily used to command transitions between these states, and the Statusword bits reflect the current state.

5.1.1 State Descriptions
	•	Not Ready to Switch On: Initial state after drive is powered on. Drive is initializing; drive functions are disabled. Only communication is possible. (Statusword indicates “Not ready to switch on.”)
	•	Switch On Disabled: Drive initialization complete, parameters established. Drive enable is still off (power stage disabled). (Status: “Switch on disabled.”)
	•	Ready to Switch On: High-voltage stage may be on, but drive output is still disabled. (Status: “Ready to switch on.”)
	•	Switched On: Servo enable is on (power stage active), but drive is not yet enabled for operation (won’t move). (Status: “Switched on.”)
	•	Operation Enabled: Servo is fully enabled; drive will execute motions and functions. No faults present. (Status: “Operation enabled.”)
	•	Quick Stop Active: Drive is performing a quick stop (rapid deceleration). How the motor stops depends on the Quick Stop option code. (Status: “Quick stop.”)
	•	Fault Reaction Active: A fault has occurred and the drive is executing its fault reaction (e.g., ramping down). Servo is disabled (drive output cut off or limiting). (Status: “Fault reaction active.”)
	•	Fault: The drive is in a fault state (after fault reaction is done). Servo remains disabled. A reset is required to exit this state. (Status: “Fault.”)

Controlword (0x6040): This 16-bit command word from the controller has various bits that control state transitions and modes:
	•	Bits 0–3,7 are used for state control:
	•	Bit0: Switch On (1 = request to switch on the drive)
	•	Bit1: Enable Voltage (1 = apply drive power)
	•	Bit2: Quick Stop (0 = execute quick stop, 1 = normal operation)
	•	Bit3: Enable Operation (1 = enable operation once other conditions met)
	•	Bit7: Fault Reset (1 = reset fault, rising edge)
	•	Bits 4,5,6,8 are mode-specific or manufacturer-specific:
	•	Bit4: Halt (for some modes, e.g., profile modes to command a halt)
	•	Bit5: (Not used in standard DS402; reserved or manufacturer-specific)
	•	Bit6: Mode specific (e.g., absolute/relative selection in position mode)
	•	Bit8: Mode specific (varies by manufacturer, not used in standard profile here)
	•	Bits 9–10: Reserved (always 0)
	•	Bits 11–15: Manufacturer specific

State Transition Commands (Controlword bits 0,1,2,3,7): Different combinations of these bits command different state transitions. The table below summarizes some typical commands (X = don’t care):

Controlword bits: 7   3 2 1 0    Action (State Transition)         Transition No.
                   |   | | | |
Shutdown           0   X 1 1 0   -> Disable drive (Switch On Disabled)   2, 6, 8  
Switch On          0   0 1 1 1   -> Enable drive (Switched On)          3*  
Enable Operation   0   1 1 1 1   -> Enable operation (Operation Enabled) 4, 16  
Disable Voltage    0   X X 0 X   -> Disable voltage (turn off power)    7, 9, 10, 12  
Quick Stop         0   X 0 1 X   -> Quick stop (transition to Quick Stop Active) 7, 10, 11  
Disable Operation  0   0 1 1 1   -> Disable operation (go to “Switched On”)      5  
Fault Reset        1   X X X X   -> Fault reset (rising edge triggers)   15  

(Transitions indicated with * or ** correspond to combined actions. * indicates that enabling will be performed; ** indicates same action as *.)

Mode-specific Control Bits (4,5,6,8): The function of bits 4,5,6,8 depends on the current operation mode of the drive:

Bit	In Profile Position Mode	In Profile Velocity Mode	In Torque (Current) Mode
4	New set-point – triggers a new target position (when set from 0→1)	Not used (n/a)	Not used (n/a)
5	Not used	Not used	Not used
6	Abs/Rel – 0 = relative, 1 = absolute positioning mode	Not used	Not used
8	Halt – 1 = execute halt (stop motion)	Halt – same function (stop)	Halt – same function (stop)

(Bits 9 and 10 are reserved for future use and generally set to 0. Bits 11–15 are vendor-specific.)
	•	Object 0x6040 – Controlword: (16-bit, RW, PDO-mappable) contains these control bits. Default value after reset is 0 (drive in Not Ready state). Changing specific bits as above causes state changes or mode actions.

Statusword (0x6041): This 16-bit status word from the drive reflects the drive’s current state and other status flags. It includes bits defined by CiA 402:
	•	Bit 0: Ready to switch on
	•	Bit 1: Switched on
	•	Bit 2: Operation enabled
	•	Bit 3: Fault
	•	Bit 4: Voltage enabled (internal power stage on)
	•	Bit 5: Quick stop (active)
	•	Bit 6: Switch on disabled
	•	Bit 7: Warning
	•	Bit 8: Manufacturer specific
	•	Bit 9: Remote (controlled via NMT master, not used in this drive)
	•	Bit 10: Target reached (in profiling modes)
	•	Bit 11: Internal limit active (e.g., limit switch or following error triggered)
	•	Bit 12,13: Operation mode-specific bits
	•	Bit 14,15: Manufacturer specific bits

The meaning of some combinations of status bits 0–3,5,6 (the state bits) is summarized below (x = don’t care):

Statusword (bits 0-3,5,6 represented as b5 b3 b2 b1 b0):
x0xx 0000 – Not Ready to Switch On  
x1xx 0000 – Switch On Disabled  
x01x 0001 – Ready to Switch On  
x01x 0011 – Switched On  
x01x 0111 – Operation Enabled  
x00x 0111 – Quick Stop Active  
x0xx 1111 – Fault Reaction Active  
x0xx 1000 – Fault

Other status bits behaviors:
	•	Bit 4 (Voltage enabled): 1 when the drive’s DC bus is powered (internal high voltage is on).
	•	Bit 5 (Quick stop): 1 when Quick Stop is active. It resets to 0 when Quick Stop action is completed.
	•	Bit 7 (Warning): 1 if any warning is present. Also set if an illegal controlword is received (command error). Cleared when no warnings.
	•	Bit 8: Reserved (0 in this implementation).
	•	Bit 9 (Remote): Not supported (would indicate if node is under remote NMT control).
	•	Bit 10 (Target reached): Indicates if the target is reached in profile modes.
	•	In Profile Position mode: set to 1 when actual position is within the Position Window (0x6067) of the target for the duration of Position Window Time (0x6068). Resets to 0 when a new target is set or if actual position leaves the window.
	•	In Profile Velocity mode: set to 1 when actual speed is within Velocity Window (0x606D) of the target speed for the duration of Velocity Window Time (0x606E). Resets when target speed changes or actual speed leaves the window.
	•	In Profile Torque mode: set to 1 when actual torque/current is within a threshold of target (similar concept).
	•	Bit 11 (Internal limit active): 1 if a limit is triggered (could be position limit, software limit, or following error).
	•	Bits 12, 13: Mode-specific, meaning depends on operation mode (e.g., in Homing mode, bit12 = Homing attained, bit13 = Homing error; in other modes these might indicate other conditions like capture events).
	•	Bits 14, 15: Manufacturer specific.
	•	Object 0x6041 – Statusword: (16-bit, RO, PDO-mappable) contains these bits. Default value is 0 after reset.

5.2 Quick Stop, Halt, and Fault Reaction Options

CiA 402 defines configurable behaviors for Quick Stop, Halt, and Fault Reaction. The following objects configure those behaviors:

Index	Name	Description
0x6094	Quick stop option code	Action for Quick Stop command
0x6095	Halt option code	Action for Halt command
0x6096	Fault reaction option code	Action when fault occurs

	•	Object 0x6094 – Quick Stop Option Code:
Index: 0x6094
Name: Quick stop option code (Stop option)
Object Code: VAR
Data Type: INT16
Access: RW (optional)
PDO Mapping: NO
Value Range: 0…2 (and manufacturer-specific negative values)
Default Value: 1
Defined Values:
	•	-32768…-1: Manufacturer-specific options (if any)
	•	0: Disable drive immediately (instant stop, no ramp)
	•	1: Decelerate motor to stop using the preset deceleration ramp
	•	2: Quick stop by releasing motor (coast to stop)
(Note: Value 2, “motor release,” is only applicable for actual emergency stop commands or an external emergency stop switch. Internal quick-stop events will not use option 2; they will behave as option 1 in that case.)
	•	Object 0x6095 – Halt Option Code:
Index: 0x6095
Name: Halt option code (Stop motion option)
Object Code: VAR
Data Type: INT16
Access: RW (optional)
PDO Mapping: NO
Value Range: 0…1 (and manufacturer-specific negative values)
Default Value: 1
Defined Values:
	•	-32768…0: Manufacturer-specific (if any; not used here)
	•	1: Decelerate to stop at the specified Halt deceleration (object 0x6069)
(The drive currently only supports value 1, meaning a Halt command will cause a deceleration to stop using the configured Halt deceleration. Value 0 or others are not applicable in this implementation.)
	•	Object 0x6096 – Fault Reaction Option Code:
Index: 0x6096
Name: Fault reaction option code
Object Code: VAR
Data Type: INT16
Access: RW (optional)
PDO Mapping: NO
Value Range: 0…1 (and manufacturer-specific negative values)
Default Value: 1
Defined Values:
	•	-32768…0: Manufacturer-specific (if any)
	•	1: Disable drive on fault (turn off drive output, motor freewheels)
(The drive uses 1 = immediate disable on fault as the only supported fault reaction. Essentially, when a fault occurs, the drive’s power stage is disabled, removing torque from the motor. Other values are reserved.)

6.1 Operation Modes Overview

The drive supports multiple operation modes (per CiA 402 profile) for motion control. The primary modes implemented are:
	•	Profile Position Mode (a.k.a. Position mode, includes support for multi-segment continuous positioning)
	•	Profile Velocity Mode (a.k.a. Velocity mode)
	•	Profile Torque Mode (a.k.a. Current mode, since torque is proportional to current)

The default mode after drive startup is Profile Position (mode value = 1). The user can switch the mode via object 0x6060 as needed (e.g., set 3 for velocity mode, 4 for torque mode).
	•	Object 0x6060 – Modes of Operation (Operation mode setting):
Index: 0x6060
Name: Operation mode (setpoint)
Object Code: VAR
Data Type: INT8 (or INT16, depending on implementation)
Access: RW (mandatory)
PDO Mapping: YES
Value Range: 1, 3, 4 (supported modes)
Default Value: 1 (Profile Position)
Defined Values:
	•	1: Profile Position Mode (also called “Contour position mode”)
	•	2: Reserved (not used)
	•	3: Profile Velocity Mode (“Contour speed mode”)
	•	4: Profile Torque (Current) Mode (“Contour current mode”)
	•	5…127: Reserved for other modes (e.g., homing = 6, interpolation = 7, etc., which are not supported here)
	•	Object 0x6061 – Modes of Operation Display (Current mode indicator):
Index: 0x6061
Name: Operation mode display
Object Code: VAR
Data Type: INT8 (or INT16)
Access: RO (mandatory)
PDO Mapping: YES
Value Range: 1, 3, 4
Default Value: 1 (reflecting the default mode)
Description: Indicates the active operation mode of the drive (using same codes as 0x6060).

6.2 Profile Position Mode (Position Control)

In Profile Position Mode, the drive moves to specified target positions using defined motion profiles. The drive supports both S-curve and T-curve trajectory planning algorithms for position moves, selectable via an object (0x6086). S-curve profiles produce smooth accelerations (jerk-limited), while T-curve profiles are trapezoidal (constant acceleration).

The user can update target positions on the fly, but the behavior differs between S-curve and T-curve modes:
	•	In S-curve mode: Both start and end speeds of a point-to-point move are 0. If a new target is issued before the current move finishes, the drive will immediately abort the current move and start the new one, which can cause a sudden stop and mechanical shock. Also, in S-curve mode, the trajectory parameters (max speed, accel, decel) cannot be changed on the fly during a motion; they are fixed for that move once it starts. Changing them mid-move can lead to trajectory errors.
	•	In T-curve mode: Moves can start and end at arbitrary speeds (including 0). The drive supports a continuous motion mode in T-curve, allowing the target position to be updated continuously without stopping. In T-curve mode, not only can the target position be changed during motion, but the trajectory parameters (max speed, accel, decel) can also be adjusted in real-time, giving a very flexible motion profile.

(The drive allows single-segment and multi-segment position moves. In single-segment mode, one move is executed at a time. In multi-segment mode (continuous trajectory), the controller can queue a new target before the current move ends, and the drive will transition smoothly to the new target in T-curve mode.)

Position Mode Parameters (Object Dictionary entries):

Key objects related to position control mode include:

Index	Name	Description
0x6062	Position demand value	Target position set-value (internal)
0x6063	Position actual value	Actual position value (feedback)
0x6065	Following error window	Position error allowable window
0x6066	Following error time out	Time limit for position error
0x6067	Position window	Target position attained window
0x6068	Position window time	Target position dwell time
0x607A	Target position	Profile target position command
0x607D	Software position limits	Software limit positions (min/max)
0x607F	Max profile velocity	Maximum motion speed (for profiling)
0x6080	S-curve max acceleration	Max acceleration in S-curve profile
0x6081	S-curve max jerk	Max jerk in S-curve profile
0x6082	T-curve acceleration	Acceleration in T-curve profile
0x6083	T-curve deceleration	Deceleration in T-curve profile
0x6084	T-curve max reverse velocity	Max reverse speed for direction change (T-curve)
0x6085	Quick stop deceleration	Decel rate for Quick Stop
0x6086	Motion profile type	Trajectory type (S-curve vs T-curve)
0x608F	Encoder resolution	Encoder counts per revolution
0x6064	Position error value	Instantaneous position error
0x6069	Halt deceleration	Deceleration for Halt command

Trajectory Planning: The drive’s implementation offers two trajectory planning algorithms for position moves: S-curve and T-curve. These are selected by object 0x6086.
	•	S-curve planning yields smooth acceleration (no sudden jerk), but as noted, parameters cannot be altered mid-motion and issuing a new move command will abort the current move.
	•	T-curve planning (trapezoidal velocity) allows dynamic updates and continuous compound moves.

Figures 6.1 and 6.2 (page 47) illustrate velocity profiles for single point-to-point moves in S-curve and T-curve modes, respectively. Both start and end at zero velocity, but the S-curve profile has a smooth acceleration curve while the T-curve has linear acceleration. If a new move is commanded mid-way in S-curve mode, the current motion stops abruptly and the new trajectory starts (causing an abrupt velocity change). In T-curve mode, a new command can be integrated seamlessly.

Position Mode Object Details:
	•	0x6062 – Position Demand Value: (INT32, RO, Pulse counts)
Real-time internal position demand (the setpoint after trajectory generation). This is updated according to the trajectory planner.
Value Range: -2,147,483,648…2,147,483,647 (counts)
Default: 0
	•	0x6063 – Position Actual Value: (INT32, RO, Pulse counts)
The actual measured position of the motor (feedback).
Value Range: -2,147,483,648…2,147,483,647
Default: 0
	•	0x6065 – Following Error Window: (UINT16, RW, Pulse)
The maximum allowable position error (difference between target and actual) during operation. If the position error exceeds this window, it may be considered a following error fault.
Value Range: 0…65535 pulses
Default: 0 (if 0, following error check may be disabled)
	•	0x6066 – Following Error Time Out: (UINT16, RW, ms)
The time window for the following error. A following error fault is generated if the position error exceeds the window (0x6065) for longer than this time.
Value Range: 0…65535 ms
Default: 0 (if 0, the drive may trigger immediately when error exceeds window)
	•	0x6067 – Position Window: (UINT16, RW, Pulse)
The “attained position” window. When the difference between target and actual position falls within this window, the drive considers the target reached (position attained). Essentially, [Target - Window, Target + Window] defines the acceptable range for considering the move complete.
Default: 1000 pulses
	•	0x6068 – Position Window Time: (UINT16, RW, ms)
The time that the actual position must remain inside the Position Window before the drive signals that the target is reached.
Default: 1000 ms (1 second)
	•	0x607A – Target Position: (INT32, RW, Pulse counts)
The commanded target position for profile position moves. If using absolute positioning (see object 0x2006), this is an absolute position. If using relative, this is an offset from the current position.
Default: 0
Note: The interpretation (abs/rel) is determined by Controlword bit6 or by object 0x2006 if using the dedicated channel.
	•	0x607D – Software Position Limits: (ARRAY, 2× INT32, RW)
Software-enforced travel limits to prevent the motor from moving beyond a safe range. Once set and active, the motor will be limited to within these positions.
	•	Sub-Index 0: Number of entries (2)
	•	Sub-Index 1: Min Position Limit (INT32, RW)
	•	Sub-Index 2: Max Position Limit (INT32, RW)
Default: Not defined (– –) by default, meaning no software limit until configured.
	•	0x607F – Max Profile Velocity: (UINT16, RW, RPM)
The maximum velocity the trajectory planner will use (for both S and T profiles).
Value Range: 0…5000 RPM
Default: 3000 RPM
	•	0x6080 – Max Acceleration (S-curve): (UINT16, RW, RPS^2)
Maximum acceleration value for S-curve trajectory planning.
Value Range: 1…1000 (in some internal units corresponding to rev/s^2)
Default: 200
	•	0x6081 – Max Jerk (S-curve): (UINT16, RW, RPS^3)
Maximum jerk (rate of change of acceleration) for S-curve planning.
Value Range: 1…1000
Default: 200
	•	0x6082 – Acceleration (T-curve): (UINT16, RW, RPS^2)
Acceleration rate for T-curve trajectory planning.
Value Range: 1…1000
Default: 100
	•	0x6083 – Deceleration (T-curve): (UINT16, RW, RPS^2)
Deceleration rate for T-curve trajectory planning.
Value Range: 1…1000
Default: 100
	•	0x6084 – Max Reverse Velocity (T-curve): (UINT16, RW, RPM)
In T-curve mode, if the motor needs to reverse direction during continuous motion, this defines the maximum velocity during the reversal.
Value Range: 1…1000 RPM
Default: 100 RPM
Description: A higher value makes the motor reverse direction more quickly (shorter pause at reversal point, but more jerk/vibration), whereas a smaller value makes direction changes smoother but slower.
	•	0x6085 – Quick Stop Deceleration: (UINT16, RW, RPS^2)
Deceleration rate used for Quick Stop (if Quick Stop option 0x6094 is set to decelerate).
Value Range: 1…1000
Default: 200
	•	0x6086 – Motion Profile Type: (UINT16, RW)
Selects the trajectory curve type for position moves.
Value Range: 0–3
Default: 1
Defined Values:
	•	0: T-curve trajectory planning
	•	1: S-curve trajectory planning
	•	2: PVT (Position-Velocity-Time) curve planning (if supported)
	•	3: PT (Position-Time) curve planning (if supported)
(The drive supports 0 and 1; values 2 and 3 are placeholders with no effect unless future firmware adds those modes.)
	•	0x608F – Encoder Resolution: (UINT32, RO, counts)
The number of encoder counts per revolution of the motor. This is a fixed value based on the hardware.
Units: pulses per rev
Default: 10000 (example default, corresponds to a 2500 PPR encoder with quadrature = 10000 counts)
(This value is hardware-specific and read-only.)
	•	0x6064 – Position Error Value: (INT16, RO, pulses)
The instantaneous position error (Target position - Actual position) updated in real-time.
Value Range: -32768…32767 pulses
Default: 0
	•	0x6069 – Halt Deceleration: (UINT16, RW, RPS^2)
The deceleration rate used when a Halt command is issued (Controlword bit8 or via object 0x2002/0x2003 as appropriate).
Value Range: 1…1000
Default: 100

Position Move Examples
	•	Single-Segment Move: The controller sets up the motion parameters (target position via 0x607A, max velocity 0x607F, accel 0x6082, decel 0x6083, etc.), then sets the Controlword New Set-Point bit (or writes 1 to object 0x2001 via the dedicated channel) to start the move. The drive accepts the new set-point (Statusword bit12 “Set-point acknowledged” becomes 1) and then that bit is cleared by the controller. The motor executes the move. (Figures 6.1 and 6.2 illustrate the velocity profile for S and T curves.)
	•	Multi-Segment (Continuous) Moves: In T-curve mode, after the first segment starts, the controller can prepare a second target. Once the first move is underway, the controller updates 0x607A with a new target and again toggles the new set-point bit (or issues another start command). The drive acknowledges and smoothly transitions to the second segment without stopping (Statusword bit12 toggles accordingly). This process can continue for additional segments. Figure 6.3 (page 48) depicts a velocity curve for multiple continuous position segments. In absolute mode, each new target is an absolute position; in relative mode, each new target is relative to the current position at the time of issuing. If a new relative move is commanded while a previous move was incomplete, the remaining distance of the old move is not added to the new move (the drive essentially resets the baseline if using absolute coordinates, or simply treats the new relative command independently). Absolute mode always moves to the specified coordinate, whereas relative mode moves by a specified offset from the current position.

Figure 6.3 – Velocity profile for continuous multi-segment T-curve motion (page 48). This shows how sequential position updates result in a continuous motion. The drive seamlessly changes trajectory at points where new targets are introduced.

6.3 Profile Velocity Mode (Speed Control)

In Profile Velocity Mode, the drive controls motor speed to a target velocity with defined acceleration and deceleration ramps (the drive uses T-curve acceleration/deceleration in velocity mode). The velocity mode does not incorporate S-curve jerk control; it is essentially a first-order system (trapezoidal speed profile).

Velocity Mode Parameters:

Index	Name	Description
0x606B	Velocity demand value	Speed set-point (internal)
0x606C	Velocity actual value	Actual motor speed
0x606D	Velocity window	Speed attained window
0x606E	Velocity window time	Speed attained hold time
0x606F	Velocity threshold	Zero speed threshold
0x60FF	Target velocity	Commanded target speed
0x6092	Max velocity error	Allowed speed error
0x6093	Max velocity error time	Time allowed for speed error
0x606A	Velocity error value	Instantaneous speed error

The velocity profile is determined by target speed and the acceleration/deceleration values set by the user (object 0x6083 and 0x6082 are reused for decel/accel in velocity mode). When the target velocity is changed, the drive’s internal trajectory generator will ramp the current speed to the new target following the T-curve.
	•	0x606B – Velocity Demand Value: (INT16, RO, RPM)
The internal velocity set-point after profiling. When 0x60FF (Target velocity) is updated, the drive ramps the actual speed and this demand value changes accordingly following the accel/decel profile.
Range: -32768…32767 RPM
Default: 0
	•	0x606C – Velocity Actual Value: (INT16, RO, RPM)
The measured actual motor speed.
Range: -32768…32767 RPM
Default: 0
	•	0x606D – Velocity Window: (UINT16, RW, RPM)
A window around the target speed used to determine if the target speed has been “attained.” If target speed is V_set and window = V_win, then the velocity window is [V_set - V_win, V_set + V_win]. When actual speed falls within this window, the drive considers the speed attained (for Statusword bit10, for example).
Default: 10 RPM
	•	0x606E – Velocity Window Time: (UINT16, RW, ms)
The time the speed must remain in the velocity window before it is considered stable/reached.
Range: 0…65535 ms
Default: 0 ms (immediate — no hold time required by default)
	•	0x606F – Velocity Threshold (Zero Speed Threshold): (UINT16, RW, RPM)
A threshold to consider the motor at “zero speed.” If actual speed magnitude is below this threshold, the drive may treat the motor as stopped (used for certain internal logic or status bits).
Range: 0…100 RPM
Default: Not specified (0 or a small value by default)
	•	0x60FF – Target Velocity: (INT16, RW, RPM)
The commanded target velocity set by the user. Writing to this object updates the speed set-point (with the drive then accelerating or decelerating toward it).
Range: -32768…32767 RPM
Default: 0 RPM
	•	0x6092 – Max Velocity Error: (UINT16, RW, RPM)
The maximum allowed speed error. In velocity mode, if the difference between target and actual speed exceeds this value for longer than the allowed time (0x6093), the drive will trip a velocity error fault and disable (motor is released). If this value is 0, the drive ignores velocity error checking (no trip on speed error).
Units: RPM
Default: Not specified (likely 0 = disabled by default)
	•	0x6093 – Max Velocity Error Time: (UINT16, RW, ms)
The time duration the speed error is allowed to exceed the max error before faulting.
Range: 0…65535 ms
Default: Not specified
(If 0x6092 is nonzero and actual speed error stays above 0x6092 for longer than this time, fault occurs. Typically 0 means fault instantly if error is too high.)
	•	0x606A – Velocity Error Value: (INT16, RO, RPM)
The instantaneous difference between the velocity demand and the actual velocity (speed error). This updates in real time and can be monitored.
Range: -32768…32767 RPM
Default: 0

Velocity Mode Operation: The drive will accelerate or decelerate towards the target velocity (0x60FF) using the configured accel/decel (0x6083/0x6082). When approaching the target, if the actual speed enters the Velocity Window (0x606D) for at least the Velocity Window Time (0x606E), the Statusword bit10 “Target reached” will be set. If the target is changed, or actual speed deviates beyond the window, that bit resets.

Figure 6.11 (page 63) – Velocity curve when searching for the index pulse (Z pulse) in homing process (the text seems to reference an S-curve for homing; in velocity mode context, not directly applicable except that velocity profile is trapezoidal with immediate stop when event occurs).

6.4 Profile Torque (Current) Mode

In Profile Torque (Current) mode, the drive controls motor current (torque) to a target value. This mode is often used for torque control or current regulation. The drive also implements an I²t protection (thermal protection) for motor and power stage in this mode, as well as an optional speed limiting.

Key objects for current mode and protection:

Index	Name	Description
0x6070	Current protection time	I²t time constant (ms)
0x6071	Target current value	Current set-point (mA)
0x6072	Current protection mode	I²t protection mode (limit vs release)
0x6073	Max current	Peak current limit (mA)
0x6074	Current demand value	Actual current demand (filtered setpoint)
0x6075	Rated current	Motor rated continuous current (mA)
0x6078	Current actual value	Actual current (mA)
0x6087	Current filter (set-value filter)	Filter coefficient for current demand
0x6088	Current control type	Current mode control type settings
0x6076	Current error	Instantaneous current error
0x6077	Max velocity limit enable (in current mode)	Limits speed in torque mode

Current Mode Operation: The controller sets a target current (torque). The drive will ramp the actual current up or down based on a filter to reach the target. Meanwhile, it monitors I²t (thermal) accumulation. If the motor current stays too high for too long (exceeding I²t limits), the drive can either cut output or limit current depending on configuration. The drive also can limit motor speed in torque mode to prevent runaway when load is low (if configured to do so).
	•	0x6070 – Current Protection Time (I²t time): (UINT16, RW, ms)
The time constant for the I²t (thermal) protection algorithm, in milliseconds. This value influences how long the drive can sustain an overcurrent before triggering protection.
Default: 1000 ms (example default)
	•	0x6072 – Current Protection Mode: (UINT16, RW)
Determines action when I²t limit is reached:
0: Release mode – When I²t limit reached, the drive disables the motor (motor current is cut off, effectively freewheeling) to protect from overheating.
1: Limit mode – When I²t limit reached, the drive does not disable the motor but instead limits the motor current to the continuous current value (0x6075) to prevent further heating. Once the I²t level falls below threshold, current can rise again (hysteresis behavior).
Default: 0 (Release mode recommended during setup and tuning)
Note: Internal events (like hitting a limit switch or other “internal emergency stops”) will behave as if in mode 1 (limit) even if this is set to 2 in Quick Stop option, as explained earlier.
(Refer to the full drive manual for detailed I²t protection behavior. In general, use Release mode (0) when initially configuring the system to avoid damage.)
	•	0x6071 – Target Current Value: (INT16, RW, mA)
The desired current set-point in milliamps. Positive/negative correspond to torque in opposite directions.
Range: -1000…1000 mA (example range corresponding to ±10 A if each unit = 10 mA; actual scale might differ, but as given -10000 to +10000 mV for ±10V analog, here -1000 to +1000 might correspond to ± rated current)
Default: 0
	•	0x6073 – Max Current: (UINT16, RW, mA)
The peak allowable current (peak current limit) the drive will output. Typically corresponds to the maximum transient current (often some multiple of rated current).
Range: 1…8000 mA
Default: 5000 mA (example default, meaning 5 A peak)
(This might correspond to e.g. 2.5× a 2 A rated motor)
	•	0x6074 – Current Demand Value: (INT16, RO, mA)
The current set-point actually being used after filtering. This is the “commanded current” after applying the ramp/filter (0x6087). It’s read-only and shows how the drive is ramping toward the target.
Range: -32768…32767 mA
Default: 0
	•	0x6075 – Rated Current: (UINT16, RW, mA)
The continuous (rated) current of the motor. This is used as the reference for I²t calculations and for current limiting in protection mode 1.
Range: 0…65535 mA
Default: 1500 mA (example: a motor rated 1.5 A continuous)
	•	0x6078 – Current Actual Value: (INT16, RO, mA)
The measured actual motor current (torque-producing current).
Range: -32768…32767 mA
Default: 0
	•	0x6087 – Current Filter (Set-value Filter Coefficient): (UINT16, RW)
A smoothing filter coefficient for the current demand change. Range 0–999 (unitless). This only applies in certain current control modes (depending on 0x6088). Specifically, if 0x6088 is set to mode 2 (no speed limit, filtered current changes), this filter is used.
	•	A larger value means a heavier filter (slower change, smoother current ramp).
	•	A smaller value means a more responsive (faster) change to target current.
Range: 0…999
Default: 880 (a fairly high filter by default for smooth current changes)
Note: This filter has no effect in position or velocity modes, and also not if current control is set to mode 1 (no filter).
	•	0x6088 – Current Control Mode (Profile Type): (UINT16, RW)
Defines how the current mode operates regarding speed limiting and filtering:
0: Max-speed limit mode – In torque mode, respect a maximum speed (the motor will not exceed a certain speed under no load). The max speed is given by object 0x6077, and current changes may still be filtered per bit settings.
1: No speed limit, no input filtering – The current demand jumps immediately to the new target (step change). Essentially open-loop torque mode with instantaneous response.
2: No speed limit, with filtering – The current demand changes slowly according to the filter coefficient (0x6087). This provides a smooth torque ramp with no speed capping.
(These settings only take effect in current control mode; in position/speed modes this object has no effect.)
Range: 0–2
Default: 0 (Speed limit mode)
	•	0x6076 – Current Error: (INT16, RO, mA)
The instantaneous error in current (target current - actual current).
Range: -32768…32767 mA
Default: 0
	•	0x6077 – Max Velocity Limit Enable (for current mode): (UINT16, RW, RPM)
The maximum speed allowed in current mode. If current control mode (0x6088) is set to 0 (speed limit mode), the motor’s speed will be limited by this value while in torque mode. This prevents runaway speeds under no load.
Range: 0…5000 RPM
Default: 3500 RPM

Operation in Current Mode: The drive ramps the current to the target (0x6071) using either an immediate jump or a filtered ramp depending on 0x6088. While doing so, it monitors the motor speed if in mode 0 to ensure it doesn’t exceed 0x6077. If I²t accumulation reaches the threshold (based on 0x6070, 0x6073, 0x6075), the drive will either disable output (mode 0) or clamp the current (mode 1) to protect against overheating.

6.5 Homing (Find Reference/Home Position)

The drive supports a homing sequence to find a reference position (origin). Unlike some systems, the RoboCT drive does not have a dedicated “Homing mode” in the CiA 402 sense; instead, homing can be executed via special control commands in any mode when the motor is not moving (velocity = 0). The homing procedure can be triggered either by CANopen commands or by digital I/O signals.

The homing process is divided into up to three phases:
	1.	Find the Home Switch: Move the motor until a home limit switch (mechanical sensor) is actuated.
	2.	Find the Index Pulse (Z-phase of encoder): After the home switch is found, optionally move until the encoder’s index pulse is detected.
	3.	Post-Homing Offset Move (Detach from switch): Move the motor a small distance away from the switch to a defined zero reference position.

By configuring the homing parameters, the user can combine these steps in different ways (perform only step 1, or steps 1+3, or all 1+2+3, etc., depending on needs).

Important notes for homing:
	•	The speeds for the home switch search and the index pulse search are signed 16-bit values, where the magnitude is the speed and the sign indicates direction (positive = one direction, negative = the other).
	•	The number of index pulses to search for in phase 2 is configurable. If this number is set to 0, the drive will skip the index search phase entirely.
	•	The offset distance (phase 3) is a signed value as well, indicating how far and in which direction to move after finding the home reference to set the final zero position.
	•	The dwell (stable) time between phases can be configured to allow the mechanism to settle.

Homing Parameters (Object Dictionary):

Index	Name	Description
0x6098	Homing seek Z index count	Number of index pulses to search (phase 2)
0x6099	Homing seek Z index speed	Speed for searching index pulse
0x6097	Homing seek home switch speed	Speed for searching home switch (phase 1)
0x609A	Homing acceleration	Acceleration during homing moves
0x607C	Home offset (homing offset distance)	Offset from mechanical home to set as zero
0x607E	Homing stable time	Settle time between homing phases

	•	0x607C – Home Offset: (INT32, RW, pulses)
An offset distance added after homing. If the “home” sensor is not exactly at the desired mechanical zero, this offset adjusts the final zero position. After completing the homing sequence, the drive will consider the zero reference as (home switch position + this offset).
Default: 0 (no offset)
Diagram in Figure 6.4 (page 59) illustrates the home offset concept.
	•	0x6098 – Homing Seek Z-Index Count: (UINT16, RW)
The number of index pulses to find in phase 2. For example, if set to 1, the drive will locate the first index pulse after the home switch; if set to 2, it will skip one index and find the second, etc. If set to 0, the index search phase is skipped entirely.
Default: 0 (skip index search)
	•	0x6099 – Homing Seek Z-Index Velocity: (INT16, RW, RPM)
The speed at which to search for the encoder’s Z index pulse (phase 2). This is a signed value; sign determines direction of search.
Default: 0 (which typically means no movement unless configured)
(The magnitude should be set to a suitable slow speed for accurate index capture.)
	•	0x6097 – Homing Seek Home Switch Velocity: (INT16, RW, RPM)
The speed for searching the home (limit) switch (phase 1). Also signed; sign indicates search direction.
Default: 0
(This object is defined as an ARRAY in the OD, but here only one sub-index (speed) is used; it might be structured for future expansion.)
	•	0x609A – Homing Acceleration: (UINT16, RW, in “RPS^2” units)
The acceleration (and deceleration) used during homing movements.
Range: 1…1000 (internal unit corresponding to rev/s^2)
Default: Not explicitly given (assume a safe default, e.g., 50 or 100)
	•	0x607E – Homing Stable Time: (UINT16, RW, ms)
The time to wait (dwell) after each phase of homing to ensure the mechanism is stable (no bouncing of switches, etc.) before proceeding to the next phase.
Default: 1000 ms (1 second)

6.5.3 Homing Procedure

Phase 1 – Find Home Switch: When homing is initiated (e.g., via object 0x2009 or dedicated command), the drive moves at the configured speed (0x6097) in the specified direction until the home limit switch is activated. Depending on the starting position and desired search direction, there are a few scenarios: the motor may approach the switch from one side or the other, or it might already be on the switch.
	•	If the starting position is before the switch in the search direction, the motor will travel forward to engage the switch.
	•	If the motor starts past the switch (on the far side), and the search direction is forward, it might not find it going forward; typically one would set direction such that it will hit the switch. (Reverse search would be used in that case.)
	•	If the motor is already on the switch at start, that scenario is also handled (the drive might need to move off and re-engage or just consider it found immediately, depending on implementation).

(Figure 6.5, 6.6, 6.7 etc., not explicitly shown in text, presumably illustrate the forward and reverse search scenarios: e.g., start to the left of switch vs right of switch vs already at switch.)

For forward search (positive direction) three cases can be considered:
	1.	Start point is left of the home switch (before reaching it) – the drive moves forward until it hits the switch.
	2.	Start point is right of the home switch (already beyond it) – this usually would require either a different search direction or a strategy to handle.
	3.	Start point is exactly at the active switch – the drive may consider it immediately or move off then re-engage depending on logic.

For reverse search (negative direction), similarly:
	•	The behavior mirrors forward search but in the opposite direction. (The text mentions the details are analogous and not repeated.)

Figures 6.9 and 6.10 (pages 62-63) likely show these scenarios graphically:
	•	Figure 6.9 – Start point is on the left side of the home switch.
	•	Figure 6.10 – Start point is already on the home switch.

Once the home switch is detected, the drive stops. The method of stopping: It may either stop immediately or decelerate, depending on the Quick Stop option (0x6094). Typically for homing, a fast stop is desired. If 0x6094 = 0, it stops instantly; if =1, it decelerates to a stop.

(Figure 6.11, page 63) shows a velocity profile of the homing approach: starting with an S-curve acceleration to the search speed, then an immediate stop when the switch triggers.

Phase 2 – Find Index Pulse: After the home switch is found, if 0x6098 (Z-pulse count) is not zero, the drive proceeds to find the encoder’s index pulse. The drive will move in the direction specified by the sign of 0x6099 at the speed magnitude of 0x6099. It counts index pulses (rising edge of the encoder index) and stops when the specified count is reached.
	•	If 0x6098 = N (N > 0), the drive will look for the N-th index pulse after the home switch. (For N=1, the first index; N=2, the second, etc.)
	•	If 0x6098 = 0, this phase is skipped entirely.

(Figure 6.12, page 63) illustrates a forward index search: Starting from the stop position at the home switch, the drive moves forward until it detects the Nth index pulse, then stops.
(Figure 6.13, page 64) would similarly illustrate a reverse index search if applicable.

The drive stops immediately when the desired index pulse is found (since accurate indexing is needed). Figure 6.14 (page 64) shows the speed profile for the index search: an S-curve ramp to speed, then an immediate stop on finding the pulse.

Phase 3 – Offset Move (Detachment): After finding the index (or after the switch if index phase was skipped), the drive may perform a small move to move away from the switch or index to a final reference position. This move is typically the home offset distance (0x607C). It can be forward or reverse (sign of 0x607C indicates direction). For example, one might want the final “zero” position to be a fixed distance from the switch trigger point.

The offset move is done at a controlled speed (likely the same as one of the search speeds or a default small speed) and uses the acceleration 0x609A. After this move, the final position is considered the home (position 0).

(Figure 6.15, page 64) presumably shows an example of forward vs reverse offset moves (detachment).

Homing complete: Once all configured phases are done, the drive sets its internal position counter to 0 (or to the home offset if that was not zero). Statusword bit homing-attained (if implemented in bits 12/13) is set. The drive is now ready for normal operation using the found reference.

7. RoboCT Dedicated CANopen Commands (Manufacturer Channel)

In addition to the standard CANopen objects, the RoboCT drive provides a set of manufacturer-specific control registers (0x2000–0x200B) that allow simplified control and monitoring. These act as a “dedicated channel” for quick operations via SDO, instead of manipulating multiple standard objects.

The following table summarizes these special command objects:

Index	Function	Description (Action)
0x2000	Motor enable/disable	Enable or disable motor drive
0x2001	Start motion	Start profile motion (position move)
0x2002	Stop motion	Decelerate to stop (profile halt)
0x2003	Quick stop	Quick stop (per quick stop option)
0x2004	Reset position (Encoder zero)	Set current position as zero
0x2005	Clear error	Acknowledge and clear faults
0x2006	Absolute/Relative motion	Set positioning mode (abs vs rel)
0x2007	Start seek limit	Begin homing: find limit switch
0x2008	Start seek Z index	Continue homing: find index pulse
0x2009	Start seek home	Full homing sequence start
0x200A	CAN mode select	Select protocol (MOTEC vs CANopen)
0x200B	Error code	Current fault code (32-bit bitfield)

Object 0x2000 – Motor Enable/Disable:
Writing to this object controls the basic power stage of the drive (similar to turning the servo on or off):
	•	Index: 0x2000
	•	Name: Motor enabling/release
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW (PDO mappable)
	•	Value Range: 0 or 1
	•	Default: 0
	•	Meaning: Write 1 to enable the motor (equivalent to powering the drive and enabling it, i.e., move from Switch On Disabled to Operation Enabled through the state machine). Write 0 to disable motor output (servo off).
(Effectively, 1 corresponds to setting the controlword bits for enabling the motor; 0 corresponds to disabling.)

Object 0x2001 – Start Motion:
Triggers the execution of a motion profile that has been pre-set (target position, etc.). This is equivalent to toggling the “new set-point” bit in the Controlword for position mode.
	•	Index: 0x2001
	•	Name: Start motion
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW (PDO mappable)
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 causes the drive to begin motion toward the target defined in object 0x607A (Target Position) using the current profile parameters (velocity, accel, etc.). After the motion starts, this object’s value automatically returns to 0.
Note: Ensure that 0x607A is set, and also object 0x2006 (absolute vs relative mode) is set appropriately before triggering 0x2001. The drive uses 0x2006 to interpret the target position as absolute or relative. The drive must be enabled (0x2000 = 1) for this to have effect.

Object 0x2002 – Stop Motion:
Commands the drive to decelerate to a stop (controlled stop) using the configured halt deceleration (0x6069).
	•	Index: 0x2002
	•	Name: Stop motion
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW (PDO mappable)
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 causes the drive to ramp down velocity to zero using the deceleration set in 0x6069 (“halt decel”). After the command is issued, the object resets to 0. Essentially this is like issuing a Halt command in CiA402. The motor will come to a gentle stop.

Object 0x2003 – Quick Stop:
Commands an emergency quick stop according to the configured Quick Stop option (0x6094).
	•	Index: 0x2003
	•	Name: Quick stop
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW (PDO mappable)
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 immediately triggers a quick stop. The action taken depends on object 0x6094:
	•	If 0x6094 = 0, the motor torque is cut immediately (no decel ramp).
	•	If 0x6094 = 1, the motor decelerates to zero speed using the Quick Stop deceleration (0x6085).
	•	If 0x6094 = 2, the motor is disabled (freewheel). (This option is typically for an emergency stop button scenario and does not apply to internally commanded stops.)
After issuing, the object resets to 0.
Note: The text indicates that if quick stop option is 2 (motor release), this is only honored for actual emergency stop commands or external triggers, not for “internal” stops like limits or faults, which will behave like option 1.

Object 0x2004 – Reset Position (Encoder is cleared):
Sets the current encoder position to zero (i.e., resets the home position).
	•	Index: 0x2004
	•	Name: Encoder is cleared (Reset position)
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO (not typically needed in PDO)
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 causes the drive to reset its position counter to 0 (i.e., the current position becomes the new zero reference). This is useful for establishing a reference point. The command resets to 0 immediately after execution.

Object 0x2005 – Clear Error (Fault Reset):
Clears the drive’s fault status and allows a faulted drive to be re-enabled.
	•	Index: 0x2005
	•	Name: Fault clearing
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 will reset the drive’s fault condition (same as toggling Controlword bit7 “Fault Reset”). Any active fault is acknowledged and cleared, and the drive can transition from Fault state to Switch On Disabled (if the fault cause is gone). The value resets to 0 after execution.

Object 0x2006 – Absolute/Relative Movement Mode:
Specifies whether the next position move (triggered by 0x2001) is interpreted as absolute or relative.
	•	Index: 0x2006
	•	Name: Absolute/relative movement mode
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Units: (interpreted as boolean)
	•	Value Range: 0 or 1
	•	Default: 0 (Relative)
	•	Description: 0 = Relative positioning, 1 = Absolute positioning.
After setting this mode, when a start motion command is given, the drive will treat 0x607A as either an offset (if 0) or an absolute coordinate (if 1).

Object 0x2007 – Start Seek Limit:
Begins the first stage of homing: searching for a limit (home) switch.
	•	Index: 0x2007
	•	Name: Start looking for a limit
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 makes the drive start moving to find the home limit switch. The speed and direction are determined by object 0x6097 (homing switch speed). Once the switch is found, the drive stops and this object resets to 0.

Object 0x2008 – Start Seek Z-Index:
Continues the homing sequence by searching for the encoder’s index pulse.
	•	Index: 0x2008
	•	Name: Start finding Z pulse
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 triggers the drive to begin the index pulse search (phase 2 of homing). The drive will move as per 0x6099 (index search speed/direction) and look for the specified count of Z pulses (0x6098). After completion, it stops and resets this to 0. The move is automatically stopped after the index is found.
Note: The drive’s action depends on the prior phase being completed. This would typically be issued after 0x2007 has completed (i.e., home switch found).

Object 0x2009 – Start Seek Home:
Initiates the full homing sequence (all phases) in one command.
	•	Index: 0x2009
	•	Name: Start to find the origin
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Value Range: 0 or 1
	•	Default: 0
	•	Behavior: Writing 1 causes the drive to execute the entire homing routine (find home switch, then index pulse if configured, then apply offset). The drive will automatically go through phases 1, 2, 3 as configured by 0x6098-0x609A, 0x607C, 0x607E. After homing is finished, the value resets to 0.
This provides a one-step homing command for convenience.

Object 0x200A – CAN Mode Select:
Selects the CAN communication mode (proprietary MOTEC protocol vs standard CANopen).
	•	Index: 0x200A
	•	Name: MOTEC channel selection (CAN mode select)
	•	Object Code: VAR
	•	Data Type: UINT16
	•	Access: RW
	•	PDO Mapping: NO
	•	Value Range: 0 or 1
	•	Default Value: 0
	•	Description: 0 = MOTEC protocol, 1 = CANopen protocol.
This setting determines which CAN protocol the drive uses. Changes require a restart to take effect (similar to parameter Pr.299). Essentially, it is a software way to switch the drive between proprietary and CANopen modes.

Object 0x200B – Error Code:
This provides the current 32-bit error code bitfield of the drive (same information as the Emergency error code and Error Register, but in one 32-bit value).
	•	Index: 0x200B
	•	Name: Fault code
	•	Object Code: VAR
	•	Data Type: UINT32
	•	Access: RO
	•	PDO Mapping: NO
	•	Units: –
	•	Value Range: 0x00000000…0xFFFFFFFF
	•	Default Value: 0 (no fault)
	•	Description: Each bit of this 32-bit value corresponds to a specific fault or warning. A bit is 1 if that fault is present, otherwise 0. Multiple bits can be 1 if multiple issues occur. (This is basically the same bit assignment as the table of error codes provided earlier.)

Below is the list of fault bits, their meanings, and recommended actions (this replicates the earlier error table):

Drive Fault/Alarm Codes (0x200B bits):
	1.	0x00000001 – System failure
Cause: Program runtime failure.
Remedy: Restart the drive.
	2.	0x00000002 – Drive start failure
Cause: Initialization failed during power-on self-test.
Remedy: Restart the drive.
	3.	0x00000004 – Parameter error
Cause: CRC check error reading parameters from flash (parameters corrupted).
Remedy: Re-import or reconfigure the parameter file, then restart.
	4.	0x00000008 – Undervoltage alarm (listed as “Overpressure alarm” in text)
Cause: Bus voltage is too low.
Remedy: Stabilize or increase power supply voltage; check if load is too heavy; reduce motor acceleration.
	5.	0x00000010 – Overvoltage alarm (“Overpressure alarm”)
Cause: Bus voltage is too high.
Remedy: Stabilize or reduce power supply voltage; check brake resistor or regenerative braking; reduce motor deceleration.
	6.	0x00000020 – I²t warning (“I2T re… to the police”)
Cause: Motor overload (thermal) – sustained high current over time (I²t threshold exceeded, primary stage).
Remedy: Check motor wiring and whether motor load is too large; allow motor to cool or reduce current demand.
	7.	0x00000040 – Overcurrent (Peak current exceeded)
Cause: Motor current exceeded the peak current limit.
Remedy: Check if motor load is excessive or jammed; reduce load.
	8.	0x00000080 – Position error overrun
Cause: In position mode, the position tracking error exceeded the allowed limit (0x6065) for too long (0x6066).
Remedy: Reduce the load or inertia; check if acceleration/deceleration is too aggressive for the system.
	9.	0x00000100 – Encoder failure
Cause: Abnormal feedback from encoder (e.g., missing signal).
Remedy: Check encoder connections and the encoder itself for damage.
	10.	0x00000200 – Speed error overrun
Cause: In velocity mode, the speed tracking error exceeded the allowed limit (0x6092/0x6093).
Remedy: Reduce load or acceleration; ensure the motor can reach commanded speed within limits.
	11.	0x00000400 – Power module over-temperature warning
Cause: Drive’s power module (IPM) temperature too high – level 1 warning.
Remedy: Check mechanical load; improve drive cooling (heat sinks, airflow).
	12.	0x00000800 – Power module over-temperature alarm
Cause: Power module temperature too high – level 2 alarm (higher threshold).
Remedy: Reduce load; improve drive cooling. (If repeated, consider a larger drive or better cooling.)
	13.	0x00001000 – STO activation
Cause: Safe Torque Off (STO) circuit activated (drive disable via safety interlock).
Remedy: Check the STO inputs/modules and wiring. Ensure STO is not triggered unintentionally.
	14.	0x00002000 – FLASH failure
Cause: Internal Flash read/write error (e.g., parameter storage failure).
Remedy: Restart the drive. If persistent, service may be required (flash memory issue).
	15.	0x00004000 – Current offset fault (“Current deviation value fault”)
Cause: The current sensor zero offset self-test failed (detected abnormal bias during startup self-test).
Remedy: This may indicate a calibration issue. Restart the drive. If fault persists, the current sensing hardware might need calibration or repair.
	16.	0x00008000 – Motor not enabled
Cause: A motion was commanded but the motor was not enabled (an action required the motor to be enabled).
Remedy: Enable the motor (servo on) before commanding motion.
	17.	0x00010000 – IPM fault alarm (IPM hitch…)
Cause: The Intelligent Power Module (drive’s inverter stage) signaled a fault (over-current, internal error, etc.).
Remedy: Restart the drive. Check STO or other protections; ensure drive is within spec (voltage, current). If recurring, hardware might be at fault.
	18.	0x00020000 – Overspeed alarm (Speed overlimit alarm)
Cause: Motor speed exceeded the defined maximum (possibly 0x6077 in torque mode or internal limit).
Remedy: Reduce commanded speed or change 0x6077 to a higher limit if safe; verify mechanical load isn’t driving motor faster (e.g., backdriving).
	19.	0x00040000 – Phase loss alarm (Lack of phase alarm)
Cause: One phase of the 3-phase supply is missing (phase loss).
Remedy: Check the input power phases and connections.
	20.	0x00080000 – Motor over-temperature alarm (Level 2)
Cause: Motor temperature too high (secondary alarm threshold reached).
Remedy: Reduce load; improve motor cooling; or reduce acceleration/deceleration to lower thermal stress.
	21.	0x00100000 – I²t warning
Cause: Motor is in continuous overload (entered I²t limiting state, indicating thermal overload condition).
Remedy: Reduce load or reduce aggressive motion so the continuous current stays under rated; ensure acceleration isn’t too high causing continuous overcurrent.
	22.	0x00200000 – Forward limit warning
Cause: Forward limit switch is triggered.
Remedy: If it’s a true limit reached, the system should stop or reverse. If it’s a false trigger, check the switch and settings.
	23.	0x00400000 – Reverse limit warning (Negative direction limit)
Cause: Reverse (negative direction) limit switch triggered.
Remedy: If intentional, take appropriate action (stop or reverse motion). If false, check switch.
	24.	0x00800000 – Discharge resistance overheating
Cause: The brake resistor (regeneration resistor) is active too long/overheating.
Remedy: Check motor duty cycle and decelerations; possibly use a larger or additional brake resistor; allow cooling time.
	25.	0x01000000 – Motor over-temperature warning (Level 1)
Cause: Motor temperature high, primary warning level.
Remedy: Reduce load or duty cycle; improve motor cooling; observe if it reaches alarm (level 2).
	26.	0x02000000 – Encoder initialization failure
Cause: Encoder failed to initialize or synchronize at startup.
Remedy: Restart the drive. If issue persists, check encoder wiring and integrity.
	27.	0x04000000 – (reserved) continue to have (no specific fault defined, reserved bit)
	28.	0x08000000 – (reserved) continue to have
	29.	0x10000000 – (reserved) continue to have
	30.	0x20000000 – (reserved) continue to have
	31.	0x40000000 – (reserved) continue to have
	32.	0x80000000 – (reserved) continue to have

(Bits 27–32 are reserved or not used; labeled “continue to have” in the text indicating placeholders.)

8. Additional Drive Data Objects

This section describes miscellaneous drive parameters available via the object dictionary:

Index	Name	Description
0x6079	DC bus voltage	Drive’s DC bus voltage (input voltage)
0x6089	Digital inputs status	Status of digital input channels
0x608A	Digital outputs status	Status of digital output channels
0x608B	Analog input 1	Value of analog input 1
0x608C	Analog input 2	Value of analog input 2
0x608D	Analog output 1	Value set on analog output 1
0x608E	Analog output 2	Value set on analog output 2
0x6090	Drive status flag	Collection of various status bits
0x6091	IPM temperature	Power module temperature (°C)

	•	0x6079 – Drive DC Bus Voltage: (UINT16, RO, Volts)
The current DC bus voltage of the drive’s power supply. This can be read in real-time to monitor supply voltage.
Range: 0…1000 V (scaled in actual volts)
Default: 0 (when no power)
(For example, a reading of 325 would indicate ~325 V DC bus, etc.)
	•	0x6089 – Digital Input Status: (UINT16, RO)
The state of the drive’s digital input channels. Bits 0–11 correspond to digital inputs 1–12.
Bit = 0: Corresponding input optocoupler is OFF (no signal).
Bit = 1: Corresponding input is ON (activated).
Range: 0…2047 (as 12 bits)
Default: 0 (no inputs active)
	•	0x608A – Digital Output Status: (UINT16, RW)
The state of the drive’s digital output channels. Bits 0–5 correspond to outputs 1–6.
Bit = 0: Output is OFF.
Bit = 1: Output is ON.
Range: 0…63 (6 bits)
Default: 0
The outputs can be written via SDO to control relays, indicators, etc., if configured. Writing this object changes the outputs accordingly.
	•	0x608B – Analog Input 1 Value: (INT16, RO, mV)
The raw value of analog input channel 1. Input range is -10 V to +10 V, and the value is given in millivolts.
Range: -10000…10000 (representing -10.000 V to +10.000 V)
Default: 0
(For example, -5000 would mean -5.000 V on analog input 1.)
	•	0x608C – Analog Input 2 Value: (INT16, RO, mV)
The raw value of analog input channel 2, in millivolts (-10V to +10V range).
Range: -10000…10000 mV
Default: 0
	•	0x608D – Analog Output 1 Value: (INT16, RW, mV)
The value (voltage) of analog output channel 1. The output range is 0 to +5 V (0 to 5000 mV). Writing to this object sets the output voltage.
Range: 0…5000 mV
Default: 0 mV (output at 0 V)
(For example, writing 2500 sets AO1 to 2.5 V.)
	•	0x608E – Analog Output 2 Value: (INT16, RW, mV)
The value of analog output channel 2, 0 to +5 V range.
Range: 0…5000 mV
Default: 0 mV
(Same behavior as AO1 for a second channel.)
	•	0x6090 – Drive Status Flag: (UINT16, RO)
A bitfield of various drive status flags (beyond the standard Statusword). Each bit indicates a particular condition:
	•	Bit0: isDone (motion complete flag for internal use)
	•	Bit1: isNearlyDone (near target flag)
	•	Bit2: isPositiveLimit (positive limit switch triggered)
	•	Bit3: isNegativeLimit (negative limit switch triggered)
	•	Bit4: isIndexPulse (index pulse found flag, e.g., in homing)
	•	Bit5: homing complete (search operation completed)
	•	Bit6: motor enabled (drive output stage active)
	•	Bit7: alarm present (or “alarm release” – possibly indicates whether alarm is cleared (0) or not (1))
(Bits 8-15 may be reserved or unused.)
Each bit is 1 if the condition is true/triggered, 0 if not.
Default: 0x0000 (all flags off)
	•	0x6091 – IPM Temperature: (UINT16, RO, °C)
The temperature of the drive’s Intelligent Power Module (power stage), in degrees Celsius. The value is scaled such that 1 = 1°C.
Range: 0…1000 (0 to 1000°C, though actual operating range is much lower)
Default: 0
(Use this to monitor the drive’s internal temperature. E.g., reading 75 means 75°C.)

9. Contact Information

For further assistance or information, please contact Hangzhou RoboCT Technology Development Co., Ltd.:
	•	Website: http://www.roboct.com￼
	•	Address: 7F & 8F, Building No.2, Lilda Science and Technology Park, No. 1500 Wenyi West Road, Cangqian Street, Yuhang District, Hangzhou, Zhejiang, China
	•	Service Hotline: 0571-28076520 ext. 2
	•	Email: ai@roboct.com