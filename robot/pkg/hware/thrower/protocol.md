# Tensa Throw System Protocol

## Overview

The Tensa Throw System uses a minimalist, space-separated text protocol for efficient communication. Commands are sent as single-line ASCII text, and responses are provided as either a success ("OK") with data or an error ("ERR <reason>").  All speeds are Ints in RPM.

## Commands

### THROW

Set Throw Speeds and Angle

| Parameter | Description                          | Valid Range        |
|-----------|--------------------------------------|--------------------|
| `<top>`   | Top motor speed in RPM          | Positive ints   |
| `<bottom>`| Bottom motor speed in RPM       | Positive ints   |
| `<angle>` | Throw angle in radians          | 0 to π/4 (0.7854)  |

```
THROW 10 10 0.5
OK
```

### DISP 

Set Dispenser Speed

| Parameter | Description                          | Valid Range        |
|-----------|--------------------------------------|--------------------|
| `<speed>` | Dispenser in RPMs                    | Positive Ints   |


```
DISP 10
OK
```
### LOAD

Check if a Ball is Loaded.  Responds with 1 if loaded, otherwise 0. 

```
LOAD
OK 1
```

## Notes

- All commands are sent as single-line ASCII text.
- Speed values must be positive.
- Errors are returned in the format: ERR <reason>.
- The protocol is optimized for fast parsing and low overhead.