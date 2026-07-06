import os
import sys
import json
import numpy as np
import argparse
import csv
from flask import Flask, render_template, jsonify, request, send_from_directory

# Add scripts directory to path to import from localize_camera
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from localize_camera import CAMERA_EXTRINSICS, CAMERA_NAMES

app = Flask(__name__)

class DetectionVisualizer:
    def __init__(self, data_dir, labels_file):
        self.data_dir = data_dir
        self.labels_file = labels_file
        self.labels = {}
        self.frame_folders = []
        self.load_labels()
        self.scan_data_dir()

    def load_labels(self):
        """Load machine poses from labels.csv."""
        if not os.path.exists(self.labels_file):
            print(f"Warning: Labels file {self.labels_file} not found.")
            return

        with open(self.labels_file, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                ts = row['timestamp_ns']
                self.labels[ts] = {
                    "x": float(row['machine_x']),
                    "y": float(row['machine_y']),
                    "z": float(row['machine_z']),
                    "yaw": float(row['machine_yaw'])
                }
        print(f"Loaded {len(self.labels)} labels.")

    def scan_data_dir(self):
        """Scan data directory for synchronized frame folders."""
        entries = os.listdir(self.data_dir)
        self.frame_folders = sorted([
            e for e in entries 
            if os.path.isdir(os.path.join(self.data_dir, e)) and e.isdigit()
        ], key=lambda x: int(x))
        print(f"Found {len(self.frame_folders)} frame folders.")

    def get_world_detections(self, timestamp):
        """Transform local detections to world coordinates for a given timestamp."""
        if timestamp not in self.labels:
            return None, None

        pose = self.labels[timestamp]
        machine_pos = np.array([pose["x"], pose["y"], pose["z"]])
        # The labels.csv stores machine_yaw_deg where 0=right(+X), 90=front(+Y).
        # In localize_camera.py, the internal R matrix corresponds to yaw = machine_yaw_deg - 90.
        yaw_internal_rad = np.radians(pose["yaw"] - 90)
        
        # Rotation matrix for machine orientation (maps Machine -> Court)
        cos_y, sin_y = np.cos(yaw_internal_rad), np.sin(yaw_internal_rad)
        R_yaw = np.array([
            [cos_y, -sin_y, 0],
            [sin_y,  cos_y, 0],
            [0,      0,     1]
        ])

        det_path = os.path.join(self.data_dir, timestamp, "ball_detections.json")
        if not os.path.exists(det_path):
            return [], pose

        with open(det_path, 'r') as f:
            local_detections = json.load(f)

        world_detections = []
        for det in local_detections:
            cam_name = det["cam"]
            if cam_name not in CAMERA_EXTRINSICS:
                continue

            # Exclude detections in the bottom 25% of left/right images
            if cam_name in ["left", "right"]:
                pixel_y = det.get("pixel_xy", [0, 0])[1]
                if pixel_y > 810:  # 1080 * 0.75
                    continue

            ext = CAMERA_EXTRINSICS[cam_name]
            p_zed = np.array(det["translation_xyz"])
            
            # 1. Transform to Machine Frame
            p_machine = ext["t"] + ext["R"] @ p_zed
            
            # 2. Transform to Court Frame
            p_court = R_yaw @ p_machine + machine_pos
            
            # Filter detections on the other side of the net (y=0)
            # If machine is at y < 0, exclude y > 0. If machine is at y > 0, exclude y < 0.
            if (machine_pos[1] < 0 and p_court[1] > 0) or (machine_pos[1] > 0 and p_court[1] < 0):
                continue

            # Clip Z to be at least the tennis ball radius (approx 0.0335m)
            # This ensures the bottom of the ball touches the ground (Z=0)
            ball_z = max(0.0335, float(p_court[2]))
            
            world_detections.append({
                "cam": cam_name,
                "confidence": det["confidence"],
                "position_world": {
                    "x": float(p_court[0]),
                    "y": float(p_court[1]),
                    "z": ball_z
                }
            })

        return world_detections, pose

visualizer = None

@app.route('/')
def index():
    return render_template('visualizer.html')

@app.route('/api/frames')
def get_frames():
    return jsonify({
        "frames": visualizer.frame_folders,
        "labeled_timestamps": list(visualizer.labels.keys())
    })

@app.route('/api/detections/<timestamp>')
def get_detections(timestamp):
    detections, pose = visualizer.get_world_detections(timestamp)
    if detections is None:
        return jsonify(success=False, error="No label for this timestamp")
    return jsonify(success=True, detections=detections, pose=pose)

@app.route('/data/<path:filename>')
def serve_data(filename):
    return send_from_directory(visualizer.data_dir, filename)

def main():
    global visualizer
    parser = argparse.ArgumentParser(description="3D Ball Detection Visualizer")
    parser.add_argument("--data", type=str, required=True, help="Directory containing synchronized frame folders")
    parser.add_argument("--labels", type=str, required=True, help="Path to labels.csv")
    parser.add_argument("--port", type=int, default=5001, help="Port to run Flask on")
    args = parser.parse_args()

    visualizer = DetectionVisualizer(args.data, args.labels)
    
    # Ensure templates directory exists
    template_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'templates')
    if not os.path.exists(template_dir):
        os.makedirs(template_dir)

    app.run(host='0.0.0.0', port=args.port, debug=True)

if __name__ == "__main__":
    main()
