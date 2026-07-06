import json
import os
import numpy as np
import matplotlib.pyplot as plt
from collections import defaultdict

# Path to the exported data
DATA_ROOT = "/home/logan/Documents/data/recordings_exported"
TRACKING_FILE = os.path.join(DATA_ROOT, "tracking.json")

def calculate_track_displacement(frames):
    """Calculate the total displacement of each track."""
    track_positions = defaultdict(list)
    for frame in frames:
        for det in frame.get("detections", []):
            track_id = det.get("track_id")
            if track_id is not None:
                track_positions[track_id].append(np.array([det["x"], det["y"], det["z"]]))
    
    track_displacement = {}
    for track_id, positions in track_positions.items():
        if len(positions) < 2:
            track_displacement[track_id] = 0.0
            continue
        
        dist = 0.0
        for i in range(len(positions) - 1):
            dist += np.linalg.norm(positions[i+1] - positions[i])
        track_displacement[track_id] = dist
        
    return track_displacement

def main():
    if not os.path.exists(TRACKING_FILE):
        print(f"Error: Tracking file not found at {TRACKING_FILE}")
        return

    print("Loading tracking data...")
    with open(TRACKING_FILE, 'r') as f:
        data = json.load(f)

    frames = data.get("frames", [])
    print(f"Total frames: {len(frames)}")

    # 1. Identify moving tracks
    print("Identifying moving tracks...")
    track_displacement = calculate_track_displacement(frames)
    MOVING_THRESHOLD_M = 1.0
    moving_tracks = {tid for tid, disp in track_displacement.items() if disp > MOVING_THRESHOLD_M}
    print(f"Found {len(moving_tracks)} moving tracks.")

    # 2. Extract distance and mean dimension data
    samples = []
    
    print("Extracting data points...")
    for frame_idx, frame in enumerate(frames):
        timestamp = frame["timestamp_ns"]
        det_file = os.path.join(DATA_ROOT, str(timestamp), "ball_detections.json")
        if not os.path.exists(det_file):
            continue

        detailed_detections = None
        for d in frame.get("detections", []):
            tid = d.get("track_id")
            if tid not in moving_tracks:
                continue
            
            if d.get("cam") != "front" or d.get("confidence", 0) <= 0.5:
                continue
                
            if detailed_detections is None:
                with open(det_file, 'r') as f:
                    detailed_detections = json.load(f)
            
            for dd in detailed_detections:
                if dd.get("cam") == "front" and abs(dd["confidence"] - d["confidence"]) < 1e-6:
                    xyz = np.array(dd["translation_xyz"])
                    dist = np.linalg.norm(xyz)
                    
                    if 1.0 <= dist <= 10.0:
                        bbox = dd["bbox_xyxy"]
                        w = bbox[2] - bbox[0]
                        h = bbox[3] - bbox[1]
                        
                        # Aspect ratio filter (balls should be roughly square)
                        aspect_ratio = max(w, h) / min(w, h) if min(w, h) > 0 else 999
                        
                        if aspect_ratio < 1.5:
                            mean_dim = (w + h) / 2.0
                            samples.append((dist, mean_dim, timestamp, tid))
                    break

    if not samples:
        print("No valid data points found.")
        return

    dists, mean_dims, timestamps, tids = zip(*samples)
    dists, mean_dims = np.array(dists), np.array(mean_dims)
    
    print(f"Total data points collected: {len(samples)}")

    # 3. Analyze Relationship
    # Relationship: Dimension = k / distance
    inv_dist = 1.0 / dists
    k = np.sum(mean_dims * inv_dist) / np.sum(inv_dist**2)
    y_pred = k / dists
    r2 = 1 - np.sum((mean_dims - y_pred)**2) / np.sum((mean_dims - np.mean(mean_dims))**2)

    print("\n--- RESULTS ---")
    print(f"Relationship: Mean Dimension = {k:.2f} / distance")
    print(f"R-squared: {r2:.4f}")
    
    # 4. Visualization
    plt.figure(figsize=(10, 6))
    plt.scatter(dists, mean_dims, alpha=0.3, s=10, label='Data Points', color='blue')
    
    d_range = np.linspace(min(dists), 10, 100)
    plt.plot(d_range, k / d_range, 'r', label=f'Fit: Dim = {k:.2f}/d', linewidth=2)
    
    plt.xlabel('Distance (m)')
    plt.ylabel('Mean Dimension (pixels)')
    plt.title('Ball Mean Dimension vs Distance (Front Camera)')
    plt.legend()
    plt.grid(True, alpha=0.3)
    
    save_path = os.path.join(DATA_ROOT, "area_distance_relationship.png")
    plt.savefig(save_path)
    print(f"Plot saved to: {save_path}")

if __name__ == "__main__":
    main()
