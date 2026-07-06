import json
import os
import numpy as np
import matplotlib.pyplot as plt

# Path to the triangulated data
FRONT_DETECTIONS = "/home/logan/Documents/output_triangulated/front/triangulated_detections.json"
OUTPUT_DIR = "/home/logan/Documents/output_triangulated"

def main():
    if not os.path.exists(FRONT_DETECTIONS):
        print(f"Error: Detections file not found at {FRONT_DETECTIONS}")
        return

    print("Loading triangulated detections...")
    with open(FRONT_DETECTIONS, 'r') as f:
        data = json.load(f)

    print(f"Total frames: {len(data)}")

    # 1. Extract distance and mean dimension data
    samples = []
    
    print("Extracting data points...")
    for frame in data:
        for det in frame.get("detections", []):
            if det.get("confidence", 0) <= 0.5:
                continue
                
            xyz = np.array(det["triangulated_xyz"])
            dist = np.linalg.norm(xyz)
            
            # Use the same distance range as the previous analysis
            if 1.0 <= dist <= 10.0:
                # Get bbox from left camera (triangulation center is based on left/right)
                bbox = det.get("left_bbox")
                if not bbox:
                    continue
                    
                w = bbox[2] - bbox[0]
                h = bbox[3] - bbox[1]
                
                # Aspect ratio filter (balls should be roughly square)
                if min(w, h) > 0:
                    aspect_ratio = max(w, h) / min(w, h)
                    if aspect_ratio < 1.5:
                        mean_dim = (w + h) / 2.0
                        samples.append((dist, mean_dim))

    if not samples:
        print("No valid data points found. Wait for triangulation to finish?")
        return

    dists, mean_dims = zip(*samples)
    dists, mean_dims = np.array(dists), np.array(mean_dims)
    
    print(f"Total data points collected: {len(samples)}")

    # 2. Analyze Relationship
    # Relationship: Dimension = k / distance
    inv_dist = 1.0 / dists
    k = np.sum(mean_dims * inv_dist) / np.sum(inv_dist**2)
    y_pred = k / dists
    r2 = 1 - np.sum((mean_dims - y_pred)**2) / np.sum((mean_dims - np.mean(mean_dims))**2)

    print("\n--- RESULTS ---")
    print(f"Relationship: Mean Dimension = {k:.2f} / distance")
    print(f"R-squared: {r2:.4f}")
    
    # 3. Visualization
    plt.figure(figsize=(10, 6))
    plt.scatter(dists, mean_dims, alpha=0.3, s=10, label='Triangulated Data Points', color='blue')
    
    d_range = np.linspace(min(dists), 10, 100)
    
    # New Fit (Triangulated)
    plt.plot(d_range, k / d_range, 'r', label=f'Triangulated Fit: Dim = {k:.2f}/d', linewidth=2)
    
    # Old Fit (Depth Map)
    k_old = 113.22
    plt.plot(d_range, k_old / d_range, 'g--', label=f'Original Fit: Dim = {k_old:.2f}/d', linewidth=2)
    
    plt.xlabel('Distance (m)')
    plt.ylabel('Mean Dimension (pixels)')
    plt.title('Ball Mean Dimension vs Distance (Triangulation - Front Camera)')
    plt.legend()
    plt.grid(True, alpha=0.3)
    
    save_path = os.path.join(OUTPUT_DIR, "triangulated_area_distance_relationship.png")
    plt.savefig(save_path)
    print(f"Plot saved to: {save_path}")

if __name__ == "__main__":
    main()
