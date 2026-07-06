import argparse
import numpy as np
import matplotlib.pyplot as plt
from PIL import Image
import os

def normalize_depth(depth_map, max_dist=None):
    """Normalize depth map to 0-255 grayscale."""
    # Mask for valid finite values (not NaN and not Inf)
    valid_mask = np.isfinite(depth_map)
    
    if not np.any(valid_mask):
        return np.zeros_like(depth_map, dtype=np.uint8)
        
    # Calculate statistics on valid pixels only
    min_val = np.min(depth_map[valid_mask])
    
    if max_dist is not None:
        max_val = max_dist
    else:
        max_val = np.max(depth_map[valid_mask])
    
    # Prepare output array
    depth_clipped = depth_map.copy()
    
    # Replace NaN with max_val (make them blend with far background)
    # Replace Inf with max_val (so they appear bright/far)
    depth_clipped[np.isnan(depth_map)] = max_val
    depth_clipped[np.isinf(depth_map)] = max_val
    
    # Clamp just in case there were other oddities or if we set a manual max_dist
    depth_clipped = np.clip(depth_clipped, min_val, max_val)

    if max_val == min_val:
        return np.zeros_like(depth_clipped, dtype=np.uint8)
        
    normalized = (depth_clipped - min_val) / (max_val - min_val)
    return (normalized * 255).astype(np.uint8)

def create_grid(frame_prefix, max_depth=None):
    # Construct file paths
    left_image_path = f"{frame_prefix}_left_image.jpg"
    right_image_path = f"{frame_prefix}_right_image.jpg"
    left_depth_path = f"{frame_prefix}_left_depth.npy"
    right_depth_path = f"{frame_prefix}_right_depth.npy"

    # Check if files exist
    files = [left_image_path, right_image_path, left_depth_path, right_depth_path]
    for f in files:
        if not os.path.exists(f):
            print(f"Error: File not found: {f}")
            return

    try:
        # Load images
        left_img = Image.open(left_image_path)
        right_img = Image.open(right_image_path)
        
        # Load depth maps
        left_depth = np.load(left_depth_path)
        right_depth = np.load(right_depth_path)
        
        # Normalize depth maps
        left_depth_norm = normalize_depth(left_depth, max_depth)
        right_depth_norm = normalize_depth(right_depth, max_depth)

        # Create figure
        fig, axes = plt.subplots(2, 2, figsize=(12, 8))
        
        # Top Left: Left Image
        axes[0, 0].imshow(left_img)
        axes[0, 0].set_title("Left Image")
        axes[0, 0].axis('off')

        # Top Right: Right Image
        axes[0, 1].imshow(right_img)
        axes[0, 1].set_title("Right Image")
        axes[0, 1].axis('off')

        # Use gray colormap for better visibility
        # Bottom Left: Left Depth
        axes[1, 0].imshow(left_depth_norm, cmap='gray')
        axes[1, 0].set_title("Left Depth (Normalized)")
        axes[1, 0].axis('off')

        # Bottom Right: Right Depth
        axes[1, 1].imshow(right_depth_norm, cmap='gray')
        axes[1, 1].set_title("Right Depth (Normalized)")
        axes[1, 1].axis('off')

        # Tight layout
        plt.tight_layout()
        
        # Save output
        basename = os.path.basename(frame_prefix)
        output_filename = f"grid_{basename}.png"
        plt.savefig(output_filename)
        print(f"Successfully saved grid image to {output_filename}")
        
    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create a 2x2 grid image from stereo frame data.")
    parser.add_argument("frame_prefix", help="Path prefix for the frame (e.g., path/to/frame_002200)")
    parser.add_argument("--max_depth", type=float, help="Maximum depth value in meters to clip visualization", default=None)
    args = parser.parse_args()
    
    create_grid(args.frame_prefix, args.max_depth)

