import sys
import argparse
import os
import json
import numpy as np
import cv2
import time

# Camera names in the multi-camera setup
CAMERA_NAMES = ["front", "back", "left", "right"]

# Mapping for 2x2 grid: Top-Left, Top-Right, Bottom-Left, Bottom-Right
GRID_LAYOUT = [
    ["front", "back"],
    ["left", "right"]
]

# Camera extrinsics: position of LEFT lens relative to machine center (in meters)
# Machine coordinate system: +X=right, +Y=front, +Z=up
# Rotations map from ZED IMAGE frame (X=right, Y=down, Z=forward) to Machine frame

# Pitch angle for back, left, right cameras (40 degrees down)
alpha = np.radians(40)
c = np.cos(alpha)
s = np.sin(alpha)

# Rotation matrix for 40 deg pitch down (around camera X axis)
# Standard: forward=Z, down=Y
# Pitched: forward = [0, -s, c], down = [0, c, s] in unpitched camera frame
R_pitch = np.array([
    [1, 0, 0],
    [0, c, s],
    [0, -s, c]
])

# Pitch for front camera (25 degrees down)
alpha_front = np.radians(25)
c_f = np.cos(alpha_front)
s_f = np.sin(alpha_front)
R_pitch_front = np.array([
    [1, 0, 0],
    [0, c_f, s_f],
    [0, -s_f, c_f]
])

# Pitch for body_cam_v2 front camera (5 degrees down)
alpha_front_v2 = np.radians(5)
c_fv2 = np.cos(alpha_front_v2)
s_fv2 = np.sin(alpha_front_v2)
R_pitch_front_v2 = np.array([
    [1, 0, 0],
    [0, c_fv2, s_fv2],
    [0, -s_fv2, c_fv2]
])

EXTRINSICS_PRESETS = {
    "camera_stand": {
        "expected_z": 1.2954,
        "front": {
            "t": np.array([-0.060, 0.1139, 0.0359]),
            "R": np.array([
                [1, 0, 0],  # X_m = X_img
                [0, 0, 1],  # Y_m = Z_img
                [0, -1, 0]  # Z_m = -Y_img
            ])
        },
        "right": {
            "t": np.array([0.1157, 0.060, 0.0252]),
            "R": np.array([
                [0, 0, 1],
                [-1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "back": {
            "t": np.array([0.060, -0.1157, 0.0252]),
            "R": np.array([
                [-1, 0, 0],
                [0, 0, -1],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "left": {
            "t": np.array([-0.1157, -0.060, 0.0252]),
            "R": np.array([
                [0, 0, -1],
                [1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
    },
    "body_cam": {
        "expected_z": 1.12395,  # 44.25 inches
        "front": {
            "t": np.array([-0.060, 0.273, 1.124]),
            "R": np.array([
                [1, 0, 0],
                [0, 0, 1],
                [0, -1, 0]
            ]) @ R_pitch_front
        },
        "right": {
            "t": np.array([0.248, 0.060, 1.124]),
            "R": np.array([
                [0, 0, 1],
                [-1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "back": {
            "t": np.array([0.060, -0.273, 1.124]),
            "R": np.array([
                [-1, 0, 0],
                [0, 0, -1],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "left": {
            "t": np.array([-0.248, -0.060, 1.124]),
            "R": np.array([
                [0, 0, -1],
                [1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
    },
    "body_cam_v2_front_5deg": {
        "expected_z": 1.12395,  # 44.25 inches (same as body_cam)
        "front": {
            "t": np.array([-0.060, 0.273, 1.124]),
            "R": np.array([
                [1, 0, 0],
                [0, 0, 1],
                [0, -1, 0]
            ]) @ R_pitch_front_v2
        },
        "right": {
            "t": np.array([0.248, 0.060, 1.124]),
            "R": np.array([
                [0, 0, 1],
                [-1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "back": {
            "t": np.array([0.060, -0.273, 1.124]),
            "R": np.array([
                [-1, 0, 0],
                [0, 0, -1],
                [0, -1, 0]
            ]) @ R_pitch
        },
        "left": {
            "t": np.array([-0.248, -0.060, 1.124]),
            "R": np.array([
                [0, 0, -1],
                [1, 0, 0],
                [0, -1, 0]
            ]) @ R_pitch
        },
    },
}

# Default extrinsics for backward compatibility
CAMERA_EXTRINSICS = EXTRINSICS_PRESETS["camera_stand"]

def get_camera_extrinsics(preset_name="camera_stand"):
    """Get extrinsics for a specific preset."""
    if preset_name not in EXTRINSICS_PRESETS:
        print(f"Warning: Extrinsic preset '{preset_name}' not found. Using 'camera_stand'.")
        return EXTRINSICS_PRESETS["camera_stand"]
    return EXTRINSICS_PRESETS[preset_name]

# Court keypoints from court2d (X, Y in meters, Z=0 for all ground points)
# Origin is at net center (KP11)
COURT_KEYPOINTS = {
    "KP1":  {"x": -5.4864, "y": 11.8872, "z": 0},
    "KP2":  {"x": -4.1148, "y": 11.8872, "z": 0},
    "KP3":  {"x": 0.0000,  "y": 11.8872, "z": 0},
    "KP4":  {"x": 4.1148,  "y": 11.8872, "z": 0},
    "KP5":  {"x": 5.4864,  "y": 11.8872, "z": 0},
    "KP6":  {"x": -4.1148, "y": 6.4008,  "z": 0},
    "KP7":  {"x": 0.0000,  "y": 6.4008,  "z": 0},
    "KP8":  {"x": 4.1148,  "y": 6.4008,  "z": 0},
    "KP9":  {"x": -5.4864, "y": 0.0000,  "z": 0},
    "KP10": {"x": -4.1148, "y": 0.0000,  "z": 0},
    "KP11": {"x": 0.0000,  "y": 0.0000,  "z": 0},  # Net center (origin)
    "KP12": {"x": 4.1148,  "y": 0.0000,  "z": 0},
    "KP13": {"x": 5.4864,  "y": 0.0000,  "z": 0},
    "KP14": {"x": -4.1148, "y": -6.4008, "z": 0},
    "KP15": {"x": 0.0000,  "y": -6.4008, "z": 0},
    "KP16": {"x": 4.1148,  "y": -6.4008, "z": 0},
    "KP17": {"x": -5.4864, "y": -11.8872, "z": 0},
    "KP18": {"x": -4.1148, "y": -11.8872, "z": 0},
    "KP19": {"x": 0.0000,  "y": -11.8872, "z": 0},
    "KP20": {"x": 4.1148,  "y": -11.8872, "z": 0},
    "KP21": {"x": 5.4864,  "y": -11.8872, "z": 0},
}


def compute_machine_pose_n_points(points_data, extrinsics=None, fixed_z=None):
    """
    Compute machine position in court coordinates using N points (N >= 2) from any cameras.
    Uses least-squares optimization for yaw and translation.
    
    Args:
        points_data: List of dicts, each with {"cam": str, "kp": str, "vec": [x, y, z]}
        extrinsics: Dict of camera extrinsics. If None, uses global CAMERA_EXTRINSICS.
        fixed_z: Optional fixed Z height for the machine center.
    
    Returns:
        dict with machine_position, machine_yaw_deg, point_heights, verification_error (RMSE)
    """
    if len(points_data) < 2:
        return None
    
    if extrinsics is None:
        extrinsics = CAMERA_EXTRINSICS
        
    # Convert all points to machine frame vectors and court positions
    u_list = []  # Vectors in machine frame
    p_list = []  # Court positions
    
    for pt in points_data:
        ext = extrinsics[pt["cam"]]
        vec_cam = np.array(pt["vec"])
        kp_court = COURT_KEYPOINTS[pt["kp"]]
        
        # Vector in machine frame (from machine center to keypoint)
        u = ext["t"] + ext["R"] @ vec_cam
        u_list.append(u)
        
        # Court position
        p = np.array([kp_court["x"], kp_court["y"], kp_court["z"]])
        p_list.append(p)
    
    u_arr = np.array(u_list)  # (N, 3)
    p_arr = np.array(p_list)  # (N, 3)
    N = len(u_arr)
    
    # 1. Center points for optimal rotation estimation (XY plane only)
    u_mean_xy = np.mean(u_arr[:, :2], axis=0)
    p_mean_xy = np.mean(p_arr[:, :2], axis=0)
    
    u_centered = u_arr[:, :2] - u_mean_xy
    p_centered = p_arr[:, :2] - p_mean_xy
    
    # 2. Solve for optimal yaw using closed-form solution
    # psi = atan2(sum(p'_y * u'_x - p'_x * u'_y), sum(p'_x * u'_x + p'_y * u'_y))
    numerator = np.sum(p_centered[:, 1] * u_centered[:, 0] - p_centered[:, 0] * u_centered[:, 1])
    denominator = np.sum(p_centered[:, 0] * u_centered[:, 0] + p_centered[:, 1] * u_centered[:, 1])
    
    if abs(denominator) < 1e-9 and abs(numerator) < 1e-9:
        yaw = 0.0
    else:
        yaw = np.arctan2(numerator, denominator)
    
    cos_y, sin_y = np.cos(yaw), np.sin(yaw)
    R = np.array([
        [cos_y, -sin_y, 0],
        [sin_y,  cos_y, 0],
        [0,      0,     1]
    ])
    
    # 3. Compute optimal translation
    # t = mean(p) - R @ mean(u) for XY
    u_mean_3d = np.mean(u_arr, axis=0)
    p_mean_3d = np.mean(p_arr, axis=0)
    
    t_optimal = p_mean_3d - R @ u_mean_3d
    machine_x, machine_y = t_optimal[0], t_optimal[1]
    
    # 4. Compute height
    point_heights = []
    for i in range(N):
        z_i = p_arr[i, 2] - u_arr[i, 2]
        point_heights.append(float(z_i))
    
    if fixed_z is not None:
        machine_z = float(fixed_z)
    else:
        machine_z = np.mean(point_heights)
    
    machine_pos = np.array([machine_x, machine_y, machine_z])
    
    # 5. Compute RMSE verification error
    errors = []
    for i in range(N):
        p_computed = R @ u_arr[i] + machine_pos
        error_i = np.linalg.norm(p_computed - p_arr[i])
        errors.append(error_i)
    rmse = np.sqrt(np.mean(np.array(errors) ** 2))
    
    # Machine orientation (angle of its front vector +Y_m in court space)
    # 0 deg = +X (right), 90 deg = +Y (far baseline)
    front_c = R @ np.array([0, 1, 0])
    machine_yaw_deg = np.degrees(np.arctan2(front_c[1], front_c[0])) % 360
    
    return {
        "machine_position": {"x": float(machine_x), "y": float(machine_y), "z": float(machine_z)},
        "machine_yaw_deg": float(machine_yaw_deg),
        "point_heights": point_heights,
        "yaw_deg": float(np.degrees(yaw)),
        "verification_error": float(rmse),
        "fixed_z": fixed_z is not None
    }


class CameraLocalizerBackend:
    """Backend for camera localization using pre-exported synchronized frames.
    
    Reads from a folder structure:
        input_dir/
            {timestamp_ns}/
                front.jpg, front_xyz.npy
                back.jpg, back_xyz.npy
                left.jpg, left_xyz.npy
                right.jpg, right_xyz.npy
    
    Saves results to a JSONL file where each line is a JSON object for one frame.
    """
    
    def __init__(self, input_dir, output_file="labels.jsonl"):
        self.input_dir = input_dir
        self.output_file = output_file
        self.current_frame_idx = 0
        self.total_frames = 0
        
        self.preset_name = "camera_stand"
        self.extrinsics = EXTRINSICS_PRESETS[self.preset_name]
        
        # List of timestamp folder names (sorted)
        self.frame_folders = []
        
        # Cache for current frame data
        self.current_images = {}
        self.current_xyz = {}
        self.current_timestamp = None
        
        # Get image resolution from first frame (set during initialization)
        self.resolution = None
        
        # Results: frame_idx -> {points, machine_pose, timestamp_ns, verification_error}
        self.results = {}
        self.load_results()
        
        # Load calibration data
        self.calibration = {}
        self.load_calibration()

    def load_calibration(self):
        """Load stereo calibration data from the input directory."""
        calib_path = os.path.join(self.input_dir, "calibration.json")
        if os.path.exists(calib_path):
            try:
                with open(calib_path, 'r') as f:
                    self.calibration = json.load(f)
                print(f"Loaded calibration for {len(self.calibration)} cameras.")
            except Exception as e:
                print(f"Error loading calibration: {e}")
        else:
            print(f"Warning: No calibration.json found in {self.input_dir}")

    def load_results(self):
        """Load results from JSONL file."""
        if os.path.exists(self.output_file):
            try:
                with open(self.output_file, 'r') as f:
                    for line in f:
                        line = line.strip()
                        if not line:
                            continue
                        record = json.loads(line)
                        frame_idx = record['frame_idx']
                        self.results[frame_idx] = {
                            "points": record['points'],
                            "machine_pose": record['machine_pose'],
                            "verification_error": record.get('verification_error', 0.0),
                            "timestamp_ns": record.get('timestamp_ns', ''),
                            "preset_name": record.get('preset_name', 'camera_stand')
                        }
                print(f"Loaded {len(self.results)} labels from {self.output_file}")
            except Exception as e:
                print(f"Could not load existing results: {e}, starting fresh.")

    def save_all_to_jsonl(self):
        """Save all results to JSONL file."""
        with open(self.output_file, 'w') as f:
            for frame_idx in sorted(self.results.keys()):
                res = self.results[frame_idx]
                record = {
                    "frame_idx": frame_idx,
                    "timestamp_ns": res['timestamp_ns'],
                    "points": res['points'],
                    "machine_pose": res['machine_pose'],
                    "verification_error": res['verification_error'],
                    "preset_name": res.get('preset_name', 'camera_stand')
                }
                f.write(json.dumps(record) + '\n')
        print(f"Saved {len(self.results)} labels to {self.output_file}")

    def initialize(self):
        """Scan input directory for timestamp folders and initialize."""
        print(f"Scanning {self.input_dir} for synchronized frames...")
        
        entries = os.listdir(self.input_dir)
        self.frame_folders = sorted([
            e for e in entries 
            if os.path.isdir(os.path.join(self.input_dir, e)) and e.isdigit()
        ], key=lambda x: int(x))
        
        if not self.frame_folders:
            print("Error: No timestamp folders found in input directory.")
            return False
        
        self.total_frames = len(self.frame_folders)
        print(f"Found {self.total_frames} synchronized frames.")
        
        # Verify first frame has all required files
        first_folder = os.path.join(self.input_dir, self.frame_folders[0])
        for cam_name in CAMERA_NAMES:
            img_path = os.path.join(first_folder, f"{cam_name}.jpg")
            xyz_path = os.path.join(first_folder, f"{cam_name}_xyz.npy")
            if not os.path.exists(img_path):
                print(f"Error: Missing {img_path}")
                return False
            if not os.path.exists(xyz_path):
                print(f"Error: Missing {xyz_path}")
                return False
        
        # Get resolution from first image
        first_img = cv2.imread(os.path.join(first_folder, "front.jpg"))
        if first_img is None:
            print("Error: Could not read first image.")
            return False
        
        h, w = first_img.shape[:2]
        self.resolution = {"width": w, "height": h}
        print(f"Image resolution: {w}x{h}")
        
        # Load first frame
        self.seek_and_grab(0)
        
        print("Initialization complete.")
        return True
    
    def initialize_cameras(self):
        return self.initialize()

    def close_all(self):
        pass

    def seek_and_grab(self, frame_idx):
        """Load a specific frame by index."""
        frame_idx = max(0, min(frame_idx, self.total_frames - 1))
        self.current_frame_idx = frame_idx
        
        folder_name = self.frame_folders[frame_idx]
        folder_path = os.path.join(self.input_dir, folder_name)
        self.current_timestamp = folder_name
        
        self.current_images = {}
        self.current_xyz = {}
        
        for cam_name in CAMERA_NAMES:
            img_path = os.path.join(folder_path, f"{cam_name}.jpg")
            xyz_path = os.path.join(folder_path, f"{cam_name}_xyz.npy")
            
            img = cv2.imread(img_path)
            if img is not None:
                self.current_images[cam_name] = img
            
            if os.path.exists(xyz_path):
                self.current_xyz[cam_name] = np.load(xyz_path)
        
        return True

    def get_images(self):
        timestamps = {name: int(self.current_timestamp) for name in CAMERA_NAMES}
        return self.current_images, timestamps
    
    @property
    def metadata(self):
        class Resolution:
            def __init__(self, w, h):
                self.width = w
                self.height = h
        
        if self.resolution:
            res = Resolution(self.resolution["width"], self.resolution["height"])
        else:
            res = Resolution(1920, 1080)
        
        return {name: {"resolution": res} for name in CAMERA_NAMES}

    def get_point_3d(self, cam_name, x, y):
        """Get the 3D vector from camera to clicked point (in IMAGE coords)."""
        if cam_name not in self.current_xyz:
            return None
        
        xyz_data = self.current_xyz[cam_name]
        
        h, w = xyz_data.shape[:2]
        if x < 0 or x >= w or y < 0 or y >= h:
            return None
        
        point3d = xyz_data[y, x]
        img_x, img_y, img_z = point3d[0], point3d[1], point3d[2]
        
        if np.isfinite(img_x) and np.isfinite(img_y) and np.isfinite(img_z):
            return (float(img_x), float(img_y), float(img_z))
        
        return None

    def triangulate_point(self, cam_name, left_x, left_y, right_x):
        """
        Compute high-accuracy 3D position from stereo correspondence.
        
        Args:
            cam_name: Camera name
            left_x, left_y: Pixel coordinates in LEFT image
            right_x: Pixel X-coordinate in RIGHT image
            
        Returns:
            [X, Y, Z] in meters relative to left camera, or None
        """
        if cam_name not in self.calibration:
            return None
        
        cal = self.calibration[cam_name]
        fx = cal["fx"]
        fy = cal["fy"]
        cx = cal["cx"]
        cy = cal["cy"]
        baseline = cal["baseline"]
        
        disparity = left_x - right_x
        if disparity <= 0:
            return None
        
        # Stereo triangulation formulas
        Z = (fx * baseline) / disparity
        X = (left_x - cx) * Z / fx
        Y = (left_y - cy) * Z / fy
        
        return (float(X), float(Y), float(Z))

    def set_preset(self, preset_name):
        if preset_name in EXTRINSICS_PRESETS:
            self.preset_name = preset_name
            self.extrinsics = EXTRINSICS_PRESETS[preset_name]
            return True
        return False

    def compute_localization(self, points_data):
        """
        Compute localization from N points (N >= 2).
        
        Args:
            points_data: List of {"cam": str, "kp": str, "vec": [x, y, z]}
        
        Returns:
            dict with status, machine_position, machine_yaw_deg, point_heights, verification_error
        """
        if len(points_data) < 2:
            return {"status": "error", "message": "Need at least 2 points"}
        
        target_z = self.extrinsics.get("expected_z")
        result = compute_machine_pose_n_points(points_data, extrinsics=self.extrinsics, fixed_z=target_z)
        
        if result is None:
            return {"status": "error", "message": "Failed to compute pose"}
            
        return {
            "status": "complete",
            "num_points": len(points_data),
            "preset_name": self.preset_name,
            "machine_position": result["machine_position"],
            "machine_yaw_deg": result["machine_yaw_deg"],
            "point_heights": result["point_heights"],
            "verification_error": result["verification_error"],
            "fixed_z": True,
            "target_z": target_z
        }

    def save_result(self, frame_idx, points, machine_pose, verification_error=0.0):
        """Save a computed localization result.
        
        Args:
            frame_idx: Frame index
            points: List of {"cam": str, "kp": str, "vec": [x, y, z]}
            machine_pose: {"x": float, "y": float, "z": float, "yaw": float, "preset": str}
            verification_error: RMSE of point errors
        """
        self.results[frame_idx] = {
            "points": points,
            "machine_pose": machine_pose,
            "verification_error": verification_error,
            "timestamp_ns": self.current_timestamp,
            "preset_name": self.preset_name
        }
        self.save_all_to_jsonl()
        return True

    def remove_result(self, frame_idx):
        if frame_idx in self.results:
            del self.results[frame_idx]
            self.save_all_to_jsonl()
            return True
        return False

    def get_all_results(self):
        return [
            {"frame_idx": f, **self.results[f]}
            for f in sorted(self.results.keys())
        ]


def main():
    parser = argparse.ArgumentParser(description="Camera Localizer Backend")
    parser.add_argument("input_dir", help="Directory containing synchronized frame folders")
    args = parser.parse_args()
    
    backend = CameraLocalizerBackend(args.input_dir)
    if backend.initialize():
        print(f"Successfully loaded {backend.total_frames} frames.")
    else:
        print("Failed to initialize.")
        sys.exit(1)


if __name__ == "__main__":
    main()
