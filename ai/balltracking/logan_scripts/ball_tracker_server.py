#!/usr/bin/env python3
"""
Simple Flask server for the Ball Tracker visualizer.
Serves both the HTML visualizer and the camera images from the data folder.
"""

import os
import argparse
from flask import Flask, send_from_directory, send_file

app = Flask(__name__)

# Will be set from command line args
DATA_DIR = None
FALLBACK_DIR = None
TEMPLATES_DIR = None


@app.route('/')
def index():
    """Serve the ball tracker HTML."""
    return send_from_directory(TEMPLATES_DIR, 'ball_tracker.html')


@app.route('/tracking.json')
def tracking_json():
    """Serve the tracking.json file."""
    return send_from_directory(DATA_DIR, 'tracking.json')


@app.route('/data/<path:filepath>')
def serve_data(filepath):
    """Serve files from the data directory (images, etc.)."""
    # Special handling for triangulated images structure: cam/timestamp/left.jpg
    # Incoming request is usually: data/timestamp/cam.jpg
    parts = filepath.split('/')
    if len(parts) == 2:
        timestamp, filename = parts
        cam = filename.replace('.jpg', '').replace('_annotated', '')
        if cam in ['front', 'back', 'left', 'right']:
            # 1. Check for triangulated image structure in DATA_DIR: cam/timestamp/left.jpg
            tri_path = os.path.join(cam, timestamp, 'left.jpg')
            full_tri_path = os.path.join(DATA_DIR, tri_path)
            if os.path.exists(full_tri_path):
                return send_from_directory(DATA_DIR, tri_path)
            
            # 2. Check for original structure in FALLBACK_DIR: timestamp/cam.jpg
            if FALLBACK_DIR:
                orig_path = os.path.join(timestamp, filename)
                if os.path.exists(os.path.join(FALLBACK_DIR, orig_path)):
                    return send_from_directory(FALLBACK_DIR, orig_path)
    
    # Default behavior: try DATA_DIR then FALLBACK_DIR
    if os.path.exists(os.path.join(DATA_DIR, filepath)):
        return send_from_directory(DATA_DIR, filepath)
    if FALLBACK_DIR and os.path.exists(os.path.join(FALLBACK_DIR, filepath)):
        return send_from_directory(FALLBACK_DIR, filepath)
    
    return send_from_directory(DATA_DIR, filepath)


@app.route('/<timestamp>/<filename>')
def serve_frame_image(timestamp, filename):
    """Serve camera images from timestamp folders."""
    return send_from_directory(os.path.join(DATA_DIR, timestamp), filename)


def main():
    global DATA_DIR, FALLBACK_DIR, TEMPLATES_DIR
    
    parser = argparse.ArgumentParser(description="Ball Tracker Visualizer Server")
    parser.add_argument(
        "--data-dir", "-d",
        default="/home/logan/Documents/data/recordings_exported",
        help="Directory containing tracking.json and frame folders"
    )
    parser.add_argument(
        "--fallback-dir", "-f",
        default=None,
        help="Fallback directory to look for images if not found in data-dir"
    )
    parser.add_argument(
        "--templates-dir", "-t",
        default=os.path.join(os.path.dirname(__file__), "templates"),
        help="Directory containing ball_tracker.html"
    )
    parser.add_argument("--port", "-p", type=int, default=5002, help="Port to run on")
    parser.add_argument("--host", type=str, default="0.0.0.0", help="Host to bind to")
    args = parser.parse_args()
    
    DATA_DIR = os.path.abspath(args.data_dir)
    FALLBACK_DIR = os.path.abspath(args.fallback_dir) if args.fallback_dir else None
    TEMPLATES_DIR = os.path.abspath(args.templates_dir)
    
    print(f"Ball Tracker Server")
    print(f"  Data directory: {DATA_DIR}")
    if FALLBACK_DIR:
        print(f"  Fallback directory: {FALLBACK_DIR}")
    print(f"  Templates directory: {TEMPLATES_DIR}")
    print(f"  Server: http://localhost:{args.port}")
    print()
    
    app.run(host=args.host, port=args.port, debug=False)


if __name__ == "__main__":
    main()
