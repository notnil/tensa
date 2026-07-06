# Person Detection Drill

## Overview

This drill demonstrates a simple person detection loop that checks for players every second and plays an audio notification when a person is detected.

## Features

- Checks for detected players every 1 second using the PlayerProvider
- Plays "Person detected" audio when one or more players are detected
- Logs detection events for debugging
- Handles context cancellation gracefully

## Purpose

This is a basic example drill that showcases:
- How to use a time ticker for periodic checks
- How to query the PlayerProvider interface
- How to play embedded audio files
- Proper context handling for cancellation

## Usage

The drill will run indefinitely until cancelled, checking for players every second and providing audio feedback when people are detected.

## Building

```bash
cd pkg/ai/drillsx/examples
make person-detection
```

## Audio Files

- `person_detected.mp3` - Generated via TTS (Text-to-Speech) from the `//tts:filename=` comment in main.go

