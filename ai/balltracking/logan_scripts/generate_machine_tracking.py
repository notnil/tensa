import os
import json
import numpy as np
import sys
import csv

# Add current dir to path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from localize_camera import compute_machine_pose_n_points, get_camera_extrinsics

def process_labels(labels_path, output_csv, preset_name="body_cam"):
    if not os.path.exists(labels_path):
        print(f"Error: {labels_path} not found.")
        return

    extrinsics = get_camera_extrinsics(preset_name)
    target_z = extrinsics.get("expected_z", 1.124)
    
    print(f"Processing {labels_path} using preset {preset_name} (Z={target_z})...")
    
    results = []
    with open(labels_path, 'r') as f:
        for line in f:
            if not line.strip(): continue
            data = json.loads(line)
            frame_idx = data["frame_idx"]
            timestamp_ns = int(data["timestamp_ns"])
            points = data["points"]
            
            if len(points) < 2:
                print(f"Frame {frame_idx}: Not enough points ({len(points)})")
                continue
                
            # Re-compute localization with fixed Z
            loc = compute_machine_pose_n_points(points, extrinsics=extrinsics, fixed_z=target_z)
            
            if loc:
                results.append({
                    "frame": frame_idx,
                    "timestamp_ms": (timestamp_ns - int(results[0]["timestamp_ns"] if results else timestamp_ns)) / 1e6,
                    "timestamp_ns": timestamp_ns, # temporary for calculation
                    "x": loc["machine_position"]["x"],
                    "y": loc["machine_position"]["y"],
                    "rotation_rad": np.radians(loc["machine_yaw_deg"])
                })

    # Fix timestamps relative to first frame
    if results:
        base_ns = results[0]["timestamp_ns"]
        for r in results:
            r["timestamp_ms"] = (r["timestamp_ns"] - base_ns) / 1e6

    # Write to CSV
    with open(output_csv, 'w', newline='') as f:
        writer = csv.DictWriter(f, fieldnames=["frame", "timestamp_ms", "x", "y", "rotation_rad"])
        writer.writeheader()
        for r in results:
            writer.writerow({
                "frame": r["frame"],
                "timestamp_ms": f"{r['timestamp_ms']:.2f}",
                "x": f"{r['x']:.6f}",
                "y": f"{r['y']:.6f}",
                "rotation_rad": f"{r['rotation_rad']:.6f}"
            })
            
    print(f"Saved {len(results)} points to {output_csv}")

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python generate_machine_tracking.py <labels_jsonl> <output_csv> [preset]")
        sys.exit(1)
    
    preset = sys.argv[3] if len(sys.argv) > 3 else "body_cam"
    process_labels(sys.argv[1], sys.argv[2], preset)
