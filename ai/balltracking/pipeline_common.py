"""Shared helpers for stereo detection, court transforms, and pose inference."""

import json
from pathlib import Path

import cv2
import numpy as np
import torch

from localization.localizer import LocalizerModel


def build_detection_records(det_boxes):
    """Normalize detector output for matching and downstream processing."""
    detections = []
    for i, det in enumerate(det_boxes):
        xywh = det.xywh[0]
        conf = float(det.conf[0])
        cx, cy, w, h = map(float, xywh)
        bbox_xyxy = [cx - w / 2, cy - h / 2, cx + w / 2, cy + h / 2]
        area = float(max(0.0, w * h))
        detections.append(
            {
                "index": i,
                "center": (cx, cy),
                "bbox": bbox_xyxy,
                "area": area,
                "confidence": conf,
            }
        )
    return detections


def match_stereo_detections(left_detections, right_detections, y_tolerance=5.0, area_tolerance=0.3):
    """Match left/right detections using rectified stereo constraints."""
    matches = []
    used_right = set()

    for left_det in left_detections:
        left_cx, left_cy = left_det["center"]
        left_area = left_det["area"]

        best_match = None
        best_score = float("inf")

        for i, right_det in enumerate(right_detections):
            if i in used_right:
                continue

            right_cx, right_cy = right_det["center"]
            right_area = right_det["area"]

            y_diff = abs(left_cy - right_cy)
            if y_diff > y_tolerance:
                continue

            if left_cx <= right_cx:
                continue

            area_ratio = (
                min(left_area, right_area) / max(left_area, right_area)
                if max(left_area, right_area) > 0
                else 0.0
            )
            if area_ratio < (1 - area_tolerance):
                continue

            score = y_diff + (1 - area_ratio) * 10.0
            if score < best_score:
                best_score = score
                best_match = (i, right_det)

        if best_match is not None:
            used_right.add(best_match[0])
            matches.append((left_det, best_match[1]))

    return matches


def triangulate_stereo(u_l, v_l, u_r, calib):
    """Triangulate 3D position from rectified stereo correspondence."""
    disparity = u_l - u_r
    if disparity <= 0:
        return None, None

    z = (calib["fx"] * calib["baseline"]) / disparity
    x = (u_l - calib["cx"]) * z / calib["fx"]
    y = (v_l - calib["cy"]) * z / calib["fy"]
    return np.array([x, y, z], dtype=float), float(disparity)


def write_calibration_json(path, camera_name, left_cam_calib, baseline, width, height, svo_path):
    """Persist the factory intrinsics used by downstream localization code."""
    fx = float(left_cam_calib.fx)
    fy = float(left_cam_calib.fy)
    cx = float(left_cam_calib.cx)
    cy = float(left_cam_calib.cy)
    payload = {
        "source_svo": str(svo_path),
        "camera_name": camera_name,
        "cameras": {
            camera_name: {
                "mat": [
                    [fx, 0.0, cx],
                    [0.0, fy, cy],
                    [0.0, 0.0, 1.0],
                ],
                "dist": [0.0, 0.0, 0.0, 0.0],
                "fisheye": False,
                "resolution": {"width": int(width), "height": int(height)},
                "baseline": float(baseline),
            }
        },
    }
    with open(path, "w") as f:
        json.dump(payload, f, indent=2)


alpha = np.radians(40)
c = np.cos(alpha)
s = np.sin(alpha)
R_pitch = np.array(
    [
        [1, 0, 0],
        [0, c, s],
        [0, -s, c],
    ]
)

alpha_front = np.radians(25)
c_f = np.cos(alpha_front)
s_f = np.sin(alpha_front)
R_pitch_front = np.array(
    [
        [1, 0, 0],
        [0, c_f, s_f],
        [0, -s_f, c_f],
    ]
)

alpha_front_v2 = np.radians(5)
c_fv2 = np.cos(alpha_front_v2)
s_fv2 = np.sin(alpha_front_v2)
R_pitch_front_v2 = np.array(
    [
        [1, 0, 0],
        [0, c_fv2, s_fv2],
        [0, -s_fv2, c_fv2],
    ]
)

# Keep the extrinsics presets in one module so the offline visualizer and the
# new online runner cannot silently drift onto different court transforms.
EXTRINSICS_PRESETS = {
    "camera_stand": {
        "expected_z": 1.2954,
        "front": {
            "t": np.array([-0.060, 0.1139, 0.0359], dtype=float),
            "R": np.array(
                [
                    [1, 0, 0],
                    [0, 0, 1],
                    [0, -1, 0],
                ],
                dtype=float,
            ),
        },
        "right": {
            "t": np.array([0.1157, 0.060, 0.0252], dtype=float),
            "R": np.array(
                [
                    [0, 0, 1],
                    [-1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "back": {
            "t": np.array([0.060, -0.1157, 0.0252], dtype=float),
            "R": np.array(
                [
                    [-1, 0, 0],
                    [0, 0, -1],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "left": {
            "t": np.array([-0.1157, -0.060, 0.0252], dtype=float),
            "R": np.array(
                [
                    [0, 0, -1],
                    [1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
    },
    "body_cam": {
        "expected_z": 1.12395,
        "front": {
            "t": np.array([-0.060, 0.273, 1.124], dtype=float),
            "R": np.array(
                [
                    [1, 0, 0],
                    [0, 0, 1],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch_front,
        },
        "right": {
            "t": np.array([0.248, 0.060, 1.124], dtype=float),
            "R": np.array(
                [
                    [0, 0, 1],
                    [-1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "back": {
            "t": np.array([0.060, -0.273, 1.124], dtype=float),
            "R": np.array(
                [
                    [-1, 0, 0],
                    [0, 0, -1],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "left": {
            "t": np.array([-0.248, -0.060, 1.124], dtype=float),
            "R": np.array(
                [
                    [0, 0, -1],
                    [1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
    },
    "body_cam_v2_front_5deg": {
        "expected_z": 1.12395,
        "front": {
            "t": np.array([-0.060, 0.273, 1.124], dtype=float),
            "R": np.array(
                [
                    [1, 0, 0],
                    [0, 0, 1],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch_front_v2,
        },
        "right": {
            "t": np.array([0.248, 0.060, 1.124], dtype=float),
            "R": np.array(
                [
                    [0, 0, 1],
                    [-1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "back": {
            "t": np.array([0.060, -0.273, 1.124], dtype=float),
            "R": np.array(
                [
                    [-1, 0, 0],
                    [0, 0, -1],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
        "left": {
            "t": np.array([-0.248, -0.060, 1.124], dtype=float),
            "R": np.array(
                [
                    [0, 0, -1],
                    [1, 0, 0],
                    [0, -1, 0],
                ],
                dtype=float,
            )
            @ R_pitch,
        },
    },
}


def parse_float(value):
    if value is None:
        return None
    value = str(value).strip()
    if value == "":
        return None
    try:
        return float(value)
    except ValueError:
        return None


def get_camera_extrinsics(preset_name):
    if preset_name not in EXTRINSICS_PRESETS:
        raise ValueError(
            f"Unknown extrinsics preset '{preset_name}'. "
            f"Available: {', '.join(EXTRINSICS_PRESETS.keys())}"
        )
    return EXTRINSICS_PRESETS[preset_name]


def load_intrinsics_for_camera(calibration_json_path, camera_name):
    if calibration_json_path is None:
        return None
    path = Path(calibration_json_path)
    if not path.exists():
        raise FileNotFoundError(f"Calibration JSON not found: {path}")
    with open(path, "r") as f:
        payload = json.load(f)

    cameras = payload.get("cameras", {})
    cam = cameras.get(camera_name)
    if cam is None:
        if "mat" in payload and "camera_name" in payload and payload["camera_name"] == camera_name:
            cam = payload
    if cam is None:
        available = ", ".join(sorted(cameras.keys())) if cameras else "none"
        raise ValueError(
            f"Camera '{camera_name}' missing in calibration JSON '{path}'. Available: {available}"
        )

    mat = np.asarray(cam["mat"], dtype=np.float32)
    dist = np.asarray(cam.get("dist", [0.0, 0.0, 0.0, 0.0]), dtype=np.float32)
    fisheye = bool(cam.get("fisheye", False))
    return {camera_name: {"mat": mat, "dist": dist, "fisheye": fisheye}}


def infer_machine_pose_from_rgb_frame(frame_rgb, camera_name, extrinsics, camera_intrinsics=None):
    """Infer machine pose from a single RGB frame for the specified camera."""
    h, w = frame_rgb.shape[:2]

    if camera_intrinsics is None:
        focal = 0.95 * max(w, h)
        camera_intrinsics = {
            camera_name: {
                "mat": np.array(
                    [[focal, 0.0, w / 2.0], [0.0, focal, h / 2.0], [0.0, 0.0, 1.0]],
                    dtype=np.float32,
                ),
                "dist": np.zeros((4,), dtype=np.float32),
                "fisheye": False,
            }
        }

    device = "cuda" if torch.cuda.is_available() else "cpu"
    localizer = LocalizerModel(device=device, camera_intrinsics=camera_intrinsics, batch_size=1)
    loc_result = localizer.predict([frame_rgb], [camera_name])[0]
    if loc_result.pose is None:
        raise RuntimeError(
            "Failed to infer camera pose from frame (insufficient or invalid court keypoints)."
        )

    ext = extrinsics[camera_name]
    yaw_rad = float(loc_result.pose.yaw)
    theta_world = (np.pi / 2.0) + yaw_rad
    cam_forward_world = np.array([np.cos(theta_world), np.sin(theta_world)], dtype=float)
    cam_forward_machine = (ext["R"] @ np.array([0.0, 0.0, 1.0], dtype=float))[:2]

    if np.linalg.norm(cam_forward_machine) < 1e-6:
        raise RuntimeError(f"Invalid extrinsics for camera '{camera_name}': forward vector collapsed.")

    theta_world = np.arctan2(cam_forward_world[1], cam_forward_world[0])
    theta_machine = np.arctan2(cam_forward_machine[1], cam_forward_machine[0])
    yaw_machine_rad = theta_world - theta_machine
    yaw_machine_deg = (np.degrees(yaw_machine_rad) + 90.0) % 360.0

    yaw_rot_rad = np.radians(yaw_machine_deg - 90.0)
    c, s = np.cos(yaw_rot_rad), np.sin(yaw_rot_rad)
    R_yaw = np.array(
        [
            [c, -s, 0.0],
            [s, c, 0.0],
            [0.0, 0.0, 1.0],
        ],
        dtype=float,
    )

    cam_position = np.array([loc_result.pose.x, loc_result.pose.y, loc_result.pose.altitude], dtype=float)
    machine_position = cam_position - (R_yaw @ np.asarray(ext["t"], dtype=float))

    machine_pose = {
        "x": float(machine_position[0]),
        "y": float(machine_position[1]),
        "z": float(machine_position[2]),
        "yaw": float(yaw_machine_deg),
    }
    return machine_pose, loc_result


def infer_machine_pose_from_first_frame(input_mp4_path, camera_name, extrinsics, camera_intrinsics=None):
    cap = cv2.VideoCapture(str(input_mp4_path))
    ok, frame_bgr = cap.read()
    cap.release()
    if not ok or frame_bgr is None:
        raise RuntimeError(f"Could not read first frame from input mp4: {input_mp4_path}")

    frame_rgb = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2RGB)
    return infer_machine_pose_from_rgb_frame(
        frame_rgb=frame_rgb,
        camera_name=camera_name,
        extrinsics=extrinsics,
        camera_intrinsics=camera_intrinsics,
    )


def transform_camera_to_court(point_xyz_camera, camera_name, machine_pose, extrinsics):
    """Camera frame (X right, Y down, Z forward) -> court frame (X right, Y forward, Z up)."""
    ext = extrinsics[camera_name]
    p_camera = np.asarray(point_xyz_camera, dtype=float)
    p_machine = ext["t"] + ext["R"] @ p_camera

    yaw_deg = machine_pose["yaw"]
    yaw_rad = np.radians(yaw_deg - 90.0)
    cos_y, sin_y = np.cos(yaw_rad), np.sin(yaw_rad)
    R_yaw = np.array(
        [
            [cos_y, -sin_y, 0.0],
            [sin_y, cos_y, 0.0],
            [0.0, 0.0, 1.0],
        ],
        dtype=float,
    )
    machine_pos = np.array([machine_pose["x"], machine_pose["y"], machine_pose["z"]], dtype=float)
    return R_yaw @ p_machine + machine_pos
