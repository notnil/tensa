import os
import sys
import argparse
import cv2
import json
import time
import numpy as np
from flask import Flask, render_template, Response, request, jsonify, send_file

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from localize_camera import CameraLocalizerBackend, GRID_LAYOUT, CAMERA_NAMES, COURT_KEYPOINTS, CAMERA_EXTRINSICS, EXTRINSICS_PRESETS

app = Flask(__name__, template_folder=os.path.join(os.path.dirname(__file__), 'templates'))
backend = None


def get_world_detections(timestamp):
    """Transform local detections to world coordinates for a given timestamp."""
    # Find frame index for this timestamp
    frame_idx = -1
    for i, folder in enumerate(backend.frame_folders):
        if folder == timestamp:
            frame_idx = i
            break
    
    if frame_idx == -1 or frame_idx not in backend.results:
        return None, None

    res = backend.results[frame_idx]
    machine_pose = res['machine_pose']
    pose = {
        "x": float(machine_pose['x']),
        "y": float(machine_pose['y']),
        "z": float(machine_pose['z']),
        "yaw": float(machine_pose.get('yaw', 0.0))
    }
    
    machine_pos = np.array([pose["x"], pose["y"], pose["z"]])
    # The labels.jsonl stores machine_yaw_deg where 0=right(+X), 90=front(+Y).
    # In localize_camera.py, the internal R matrix corresponds to yaw = machine_yaw_deg - 90.
    yaw_internal_rad = np.radians(pose["yaw"] - 90)
    
    # Rotation matrix for machine orientation (maps Machine -> Court)
    cos_y, sin_y = np.cos(yaw_internal_rad), np.sin(yaw_internal_rad)
    R_yaw = np.array([
        [cos_y, -sin_y, 0],
        [sin_y,  cos_y, 0],
        [0,      0,     1]
    ])

    det_path = os.path.join(backend.input_dir, timestamp, "ball_detections.json")
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


def generate_grid_image():
    """Generate a 2x2 grid image from the current frame."""
    if not backend or not backend.current_images:
        return None
    
    images = backend.current_images
    res = backend.resolution
    w, h = res["width"], res["height"]
    
    scale = 1.0
    tw, th = int(w * scale), int(h * scale)
    
    row_imgs = []
    for row_names in GRID_LAYOUT:
        row_frames = []
        for name in row_names:
            if name in images:
                img = images[name].copy()
                resized = cv2.resize(img, (tw, th))
                cv2.putText(resized, name.upper(), (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (0, 255, 0), 2)
                ts_str = f"TS: {backend.current_timestamp}"
                cv2.putText(resized, ts_str, (10, th - 10), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (255, 255, 255), 1)
                row_frames.append(resized)
            else:
                row_frames.append(np.zeros((th, tw, 3), dtype=np.uint8))
        row_imgs.append(np.concatenate(row_frames, axis=1))
    
    grid = np.concatenate(row_imgs, axis=0)
    return grid


@app.route('/')
def index():
    return render_template('index.html', 
                           total_frames=backend.total_frames if backend else 0,
                           keypoints=list(COURT_KEYPOINTS.keys()),
                           presets=list(EXTRINSICS_PRESETS.keys()))


@app.route('/api/preset', methods=['POST'])
def set_preset():
    data = request.json
    preset = data.get('preset')
    if backend.set_preset(preset):
        return jsonify(success=True)
    return jsonify(success=False, error="Invalid preset")


@app.route('/api/preset')
def get_preset():
    return jsonify(success=True, preset=backend.preset_name)


@app.route('/visualizer')
def visualizer():
    return render_template('visualizer.html')


@app.route('/api/frames')
def get_frames():
    return jsonify({
        "frames": backend.frame_folders,
        "labeled_timestamps": [backend.frame_folders[idx] for idx in backend.results.keys()]
    })


@app.route('/api/detections/<timestamp>')
def api_get_detections(timestamp):
    detections, pose = get_world_detections(timestamp)
    if detections is None:
        return jsonify(success=False, error="No label for this timestamp")
    return jsonify(success=True, detections=detections, pose=pose)


@app.route('/video_feed')
def video_feed():
    grid = generate_grid_image()
    if grid is None:
        return Response(status=404)
    
    ret, buffer = cv2.imencode('.jpg', grid, [int(cv2.IMWRITE_JPEG_QUALITY), 85])
    return Response(buffer.tobytes(), mimetype='image/jpeg')


@app.route('/stereo_feed/<cam_name>')
def stereo_feed(cam_name):
    """Generate a side-by-side image for a specific camera (Left and Right)."""
    if not backend or cam_name not in CAMERA_NAMES:
        return Response(status=404)
    
    timestamp = backend.current_timestamp
    folder_path = os.path.join(backend.input_dir, timestamp)
    
    left_path = os.path.join(folder_path, f"{cam_name}.jpg")
    right_path = os.path.join(folder_path, f"{cam_name}_right.jpg")
    
    if not os.path.exists(left_path) or not os.path.exists(right_path):
        # Fallback to only left if right is missing
        left_img = cv2.imread(left_path)
        if left_img is None: return Response(status=404)
        h, w = left_img.shape[:2]
        stereo = np.concatenate([left_img, np.zeros_like(left_img)], axis=1)
        cv2.putText(stereo, "RIGHT IMAGE MISSING", (w + 50, h//2), cv2.FONT_HERSHEY_SIMPLEX, 1, (0,0,255), 2)
    else:
        left_img = cv2.imread(left_path)
        right_img = cv2.imread(right_path)
        stereo = np.concatenate([left_img, right_img], axis=1)
    
    ret, buffer = cv2.imencode('.jpg', stereo, [int(cv2.IMWRITE_JPEG_QUALITY), 85])
    return Response(buffer.tobytes(), mimetype='image/jpeg')


@app.route('/api/seek/<int:frame_idx>')
def seek(frame_idx):
    if backend:
        backend.seek_and_grab(frame_idx)
        return jsonify(success=True, timestamp=backend.current_timestamp)
    return jsonify(success=False, error="Backend not initialized")


@app.route('/api/get_vec', methods=['POST'])
def get_vec():
    """Get the 3D vector for a click without saving state."""
    data = request.json
    nx, ny = data['nx'], data['ny']
    
    col = 0 if nx < 0.5 else 1
    row = 0 if ny < 0.5 else 1
    cam_name = GRID_LAYOUT[row][col]
    
    local_nx = (nx - (col * 0.5)) * 2.0
    local_ny = (ny - (row * 0.5)) * 2.0
    
    res = backend.resolution
    orig_x = int(local_nx * res["width"])
    orig_y = int(local_ny * res["height"])
    
    vec = backend.get_point_3d(cam_name, orig_x, orig_y)
    if vec is None:
        return jsonify(success=False, error="Invalid depth at clicked point")
    
    return jsonify(success=True, camera=cam_name, vec=vec)


@app.route('/api/get_stereo_vec', methods=['POST'])
def get_stereo_vec():
    """Get the 3D vector for a stereo click (Left + Right)."""
    data = request.json
    cam_name = data['cam']
    left_nx, left_ny = data['left_nx'], data['left_ny']
    right_nx = data['right_nx']
    
    res = backend.resolution
    lx = int(left_nx * res["width"])
    ly = int(left_ny * res["height"])
    rx = int(right_nx * res["width"])
    
    vec = backend.triangulate_point(cam_name, lx, ly, rx)
    if vec is None:
        return jsonify(success=False, error="Invalid disparity or missing calibration")
    
    return jsonify(success=True, camera=cam_name, vec=vec)


@app.route('/api/calibration')
def get_calibration_status():
    """Check if calibration data is available."""
    has_calib = len(backend.calibration) > 0
    return jsonify(success=True, has_calibration=has_calib, cameras=list(backend.calibration.keys()))


@app.route('/api/compute', methods=['POST'])
def compute():
    """Compute pose from N points sent by frontend."""
    data = request.json
    points = data['points']
    
    if len(points) < 2:
        return jsonify(success=False, error="Need at least 2 points")
    
    result = backend.compute_localization(points)
    return jsonify(success=True, result=result)


@app.route('/api/save', methods=['POST'])
def save():
    data = request.json
    frame_idx = data['frame_idx']
    points = data['points']
    machine_pos = data['machine_position']
    machine_yaw = data.get('machine_yaw_deg', 0.0)
    verification_error = data.get('verification_error', 0.0)
    
    machine_pose = {
        "x": machine_pos["x"],
        "y": machine_pos["y"],
        "z": machine_pos["z"],
        "yaw": machine_yaw
    }
    
    if backend.save_result(frame_idx, points, machine_pose, verification_error):
        return jsonify(success=True)
    return jsonify(success=False, error="Failed to save")


@app.route('/api/remove/<int:frame_idx>', methods=['POST'])
def remove(frame_idx):
    if backend.remove_result(frame_idx):
        return jsonify(success=True)
    return jsonify(success=False, error="Failed to remove or not found")


@app.route('/api/labels')
def get_labels():
    return jsonify(labels=backend.get_all_results())


@app.route('/api/export')
def export_jsonl():
    return send_file(
        os.path.abspath(backend.output_file),
        as_attachment=True,
        download_name='labels.jsonl',
        mimetype='application/jsonl'
    )


DEFAULT_INPUT_DIR = "/home/logan/Documents/data/recordings_exported"


def main():
    global backend
    parser = argparse.ArgumentParser(description="Web-based Camera Localizer (N-Point)")
    parser.add_argument("input_dir", type=str, nargs='?', default=DEFAULT_INPUT_DIR,
                        help="Directory containing synchronized frame folders")
    parser.add_argument("--output", type=str, help="Output JSONL file path (defaults to input_dir/labels.jsonl)")
    parser.add_argument("--port", type=int, default=5000, help="Port to run Flask on")
    parser.add_argument("--host", type=str, default="0.0.0.0", help="Host to run Flask on")
    args = parser.parse_args()
    
    output_file = args.output if args.output else os.path.join(args.input_dir, "labels.jsonl")
    backend = CameraLocalizerBackend(args.input_dir, output_file=output_file)
    if not backend.initialize():
        print("Failed to initialize backend.")
        sys.exit(1)
        
    print(f"\nStarting web server at http://{args.host}:{args.port}")
    app.run(host=args.host, port=args.port, debug=False, threaded=True)


if __name__ == "__main__":
    main()
