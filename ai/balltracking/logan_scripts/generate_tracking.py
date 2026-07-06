#!/usr/bin/env python3
"""
Generate tracking.json from triangulated detections.
"""

import os
import sys
import json
import argparse
import numpy as np
from typing import Dict, List, Optional, Tuple

# Add current dir to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from localize_camera import get_camera_extrinsics
from ball_tracker import (
    Track, BallTracker, transform_to_world, is_in_exclusion_zone, 
    is_in_court_bounds, load_labels, get_all_frame_timestamps,
    MIN_DETECTION_CONFIDENCE
)

# 16ms tolerance for matching camera detections to labels
SYNC_TOLERANCE_NS = 16_000_000 

def load_triangulated_detections(base_dir: str, cameras: List[str]) -> Dict[str, Dict[str, List[dict]]]:
    """Load triangulated detections from each camera's JSON file."""
    all_cam_detections = {}
    for cam in cameras:
        det_path = os.path.join(base_dir, cam, "triangulated_detections.json")
        if not os.path.exists(det_path):
            print(f"Warning: {det_path} not found.")
            continue
        
        with open(det_path, 'r') as f:
            data = json.load(f)
            # data is list of {"timestamp": ..., "detections": [...]}
            # convert to dict for faster lookup
            cam_data = {str(item["timestamp"]): item["detections"] for item in data}
            all_cam_detections[cam] = cam_data
            print(f"Loaded {len(cam_data)} frames of detections for {cam}")
            
    return all_cam_detections

def get_closest_detections(ts_str: str, cam_data: Dict[str, List[dict]], tolerance_ns: int) -> List[dict]:
    """Find detections in cam_data with timestamp closest to ts_str within tolerance."""
    ts = int(ts_str)
    best_ts = None
    min_diff = tolerance_ns + 1
    
    # Simple linear search (can be optimized if needed)
    for cam_ts_str in cam_data.keys():
        diff = abs(int(cam_ts_str) - ts)
        if diff < min_diff:
            min_diff = diff
            best_ts = cam_ts_str
            
    if best_ts:
        return cam_data[best_ts]
    return []

def process_triangulated_tracking(
    input_base: str, 
    labels_dir: str, 
    output_path: str, 
    refine: bool = False,
    preset_name: str = "camera_stand",
    manual_machine_pose: Optional[dict] = None,
    frames_dir: Optional[str] = None
):
    """Process triangulated detections and generate tracking.json."""
    
    # 0. Load extrinsics
    extrinsics = get_camera_extrinsics(preset_name)
    print(f"Using extrinsic preset: {preset_name}")

    # 1. Load labels if available
    labels_path = os.path.join(labels_dir, "labels.jsonl")
    labels_dict = load_labels(labels_path)
    if labels_dict:
        print(f"Found {len(labels_dict)} labeled frames in labels.jsonl")
    
    # 2. Get all frame timestamps
    cameras = ["front", "back", "left", "right"]
    all_cam_detections = load_triangulated_detections(input_base, cameras)
    
    # Determine which timestamps to use
    if frames_dir:
        # Use synchronized frame timestamps from frames_dir
        all_timestamps = get_all_frame_timestamps(frames_dir)
        print(f"Using {len(all_timestamps)} synchronized timestamps from {frames_dir}")
    elif labels_dict:
        all_timestamps = sorted(labels_dict.keys(), key=lambda x: int(x))
        print(f"Using {len(all_timestamps)} timestamps from labels.jsonl")
    else:
        # Collect all timestamps from all loaded camera detections
        ts_set = set()
        for cam in all_cam_detections:
            ts_set.update(all_cam_detections[cam].keys())
        all_timestamps = sorted(list(ts_set), key=lambda x: int(x))
        print(f"No labels.jsonl found. Using {len(all_timestamps)} timestamps from detection files.")
    
    if not all_timestamps:
        print("No timestamps found.")
        return

    # Use first label's machine pose or manual pose or fallback to zero
    default_machine_pose = manual_machine_pose
    if not default_machine_pose and labels_dict:
        first_label = next(iter(labels_dict.values()))
        default_machine_pose = first_label.get("machine_pose")
    
    if not default_machine_pose:
        default_machine_pose = {"x": 0, "y": 0, "z": 0, "yaw": 0}
        print("Warning: No machine pose found, using zero pose.")
    else:
        print(f"Using default machine pose: x={default_machine_pose['x']:.2f}, y={default_machine_pose['y']:.2f}, yaw={default_machine_pose['yaw']:.1f}")

    # 4. Initialize tracker
    tracker = BallTracker(all_timestamps)
    
    # 5. Process each frame
    all_frame_data = []
    stats = {"total_detections": 0, "kept": 0}
    
    for i, ts in enumerate(all_timestamps):
        label = labels_dict.get(ts, {})
        frame_idx = label.get("frame_idx", i)
        machine_pose = label.get("machine_pose", default_machine_pose)
        machine_pos = np.array([machine_pose["x"], machine_pose["y"], machine_pose["z"]])
        machine_yaw = machine_pose["yaw"]
        
        filtered_detections = []
        
        # Aggregate detections from all cameras for this timestamp
        for cam in cameras:
            if cam not in all_cam_detections:
                continue
                
            cam_detections = get_closest_detections(ts, all_cam_detections[cam], SYNC_TOLERANCE_NS)
            stats["total_detections"] += len(cam_detections)
            
            for det in cam_detections:
                # Filter by confidence
                if det["confidence"] < MIN_DETECTION_CONFIDENCE:
                    continue
                
                # Check exclusion zone (using left_center as proxy for pixel pos)
                pixel_x, pixel_y = det["left_center"]
                if is_in_exclusion_zone(cam, pixel_x, pixel_y):
                    continue
                
                # Transform to world
                p_camera = np.array(det["triangulated_xyz"])
                p_world = transform_to_world(p_camera, cam, machine_pos, machine_yaw, extrinsics_dict=extrinsics)
                
                # Court filter
                if not is_in_court_bounds(p_world):
                    continue
                    
                stats["kept"] += 1
                filtered_detections.append({
                    "cam": cam,
                    "confidence": det["confidence"],
                    "x": float(p_world[0]),
                    "y": float(p_world[1]),
                    "z": float(p_world[2]),
                    # Pixel coordinates for image annotations
                    "pixel_x": float(pixel_x),
                    "pixel_y": float(pixel_y),
                    "bbox": det.get("left_bbox")
                })
        
        # Track
        tracked_detections = tracker.process_frame(frame_idx, ts, filtered_detections)
        
        all_frame_data.append({
            "timestamp_ns": ts,
            "frame_idx": frame_idx,
            "machine": machine_pose,
            "detections": tracked_detections
        })
        
        if (i + 1) % 100 == 0:
            print(f"Processed {i + 1}/{len(all_timestamps)} frames...")

    # 5. Finalize and Save
    trajectories = tracker.finalize()
    
    output = {
        "metadata": {
            "num_frames": len(all_frame_data),
            "timestamps": all_timestamps,
            "stats": stats,
            "extrinsic_preset": preset_name
        },
        "frames": all_frame_data,
        "trajectories": [t.to_dict(refine=refine, all_timestamps=all_timestamps) for t in trajectories]
    }
    
    print(f"Writing tracking data to {output_path}...")
    with open(output_path, 'w') as f:
        json.dump(output, f, indent=2)
    
    print(f"Done! Trajectories found: {len(trajectories)}")

def main():
    parser = argparse.ArgumentParser(description="Generate tracking.json from triangulated detections.")
    parser.add_argument("--input", "-i", required=True, help="Directory containing camera folders with triangulated_detections.json")
    parser.add_argument("--labels-dir", "-l", help="Directory containing labels.jsonl")
    parser.add_argument("--output", "-o", help="Output path for tracking.json")
    parser.add_argument("--refine", "-r", action="store_true", help="Apply physics-based trajectory refinement")
    parser.add_argument("--preset", default="camera_stand", help="Extrinsic preset to use (camera_stand, body_cam)")
    parser.add_argument("--machine-x", type=float, help="Manual machine X position")
    parser.add_argument("--machine-y", type=float, help="Manual machine Y position")
    parser.add_argument("--machine-z", type=float, default=0.0, help="Manual machine Z position")
    parser.add_argument("--machine-yaw", type=float, help="Manual machine yaw (0=right, 90=front)")
    parser.add_argument("--frames-dir", type=str, default=None, help="Directory containing synchronized frame folders")
    args = parser.parse_args()
    
    labels_dir = args.labels_dir or args.input
    output_path = args.output or os.path.join(args.input, "tracking.json")
    
    manual_pose = None
    if args.machine_x is not None and args.machine_y is not None and args.machine_yaw is not None:
        manual_pose = {
            "x": args.machine_x,
            "y": args.machine_y,
            "z": args.machine_z,
            "yaw": args.machine_yaw
        }
    
    process_triangulated_tracking(
        args.input, 
        labels_dir, 
        output_path, 
        args.refine,
        preset_name=args.preset,
        manual_machine_pose=manual_pose,
        frames_dir=args.frames_dir
    )

if __name__ == "__main__":
    main()
