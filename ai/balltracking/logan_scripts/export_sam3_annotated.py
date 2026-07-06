import sys
import argparse
import os
import numpy as np
import cv2
import time
import json
import torch
from PIL import Image
from concurrent.futures import ThreadPoolExecutor

# Add SAM3 to path
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
sam3_path = os.path.join(project_root, 'sam3')
if os.path.exists(sam3_path):
    sys.path.insert(0, sam3_path)

try:
    import pyzed.sl as sl
except ImportError:
    print("Error: pyzed not found. Please install the ZED SDK and Python API.")
    sys.exit(1)

try:
    from sam3 import build_sam3_image_model
    from sam3.model.sam3_image_processor import Sam3Processor
    HAS_SAM3 = True
except ImportError:
    HAS_SAM3 = False
    print("Warning: SAM3 not found. Detection will be skipped.")

# Camera names in the multi-camera setup
CAMERA_NAMES = ["front", "back", "left", "right"]

# Synchronization tolerance in nanoseconds (1ms = 1,000,000 ns)
SYNC_TOLERANCE_NS = 1_000_000

def parse_args():
    """Parse command line arguments for synchronized SAM3 ball detection export."""
    parser = argparse.ArgumentParser(
        description="Synchronized export of 4 ZED SVOs with SAM3 ball detection (Original + Annotated images)."
    )
    parser.add_argument(
        "input_dir", 
        type=str, 
        help="Directory containing 4 SVO files (front.svo2, back.svo2, left.svo2, right.svo2)"
    )
    parser.add_argument(
        "output_dir", 
        type=str, 
        help="Path to the output directory"
    )
    parser.add_argument(
        "--max-frames",
        type=int,
        default=None,
        help="Maximum number of synchronized frames to export"
    )
    parser.add_argument(
        "--conf",
        type=float,
        default=0.35,
        help="Confidence threshold for SAM3 detections"
    )
    parser.add_argument(
        "--fast-depth",
        action="store_true",
        help="Use PERFORMANCE depth mode instead of NEURAL_PLUS"
    )
    return parser.parse_args()

def open_camera(cam_name, svo_path, depth_mode):
    """Worker function to open a single camera in streaming mode."""
    print(f"  Opening {cam_name}...")
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.coordinate_units = sl.UNIT.METER
    init_params.depth_mode = depth_mode
    
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Error opening {cam_name}: {err}")
        return cam_name, None
    
    # Perform initial grab to get the first timestamp
    if zed.grab() != sl.ERROR_CODE.SUCCESS:
        print(f"Error: Initial grab failed for {cam_name}")
        zed.close()
        return cam_name, None
        
    ts = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
    return cam_name, (zed, ts)

def get_robust_3d(mask, xyz_data):
    """
    Calculate robust 3D translation using the median of valid pixels in the mask.
    xyz_data is a numpy array of shape (H, W, 4) or (H, W, 3).
    """
    # mask is boolean (H, W)
    pixels_3d = xyz_data[mask]
    
    # Filter out NaNs and Infs
    valid_mask = np.all(np.isfinite(pixels_3d[:, :3]), axis=1)
    valid_pixels = pixels_3d[valid_mask, :3]
    
    if len(valid_pixels) == 0:
        return None
    
    # Use median for robustness against outliers
    median_xyz = np.nanmedian(valid_pixels, axis=0)
    return median_xyz.tolist()

def process_detections(cam_name, image_np, xyz_np, sam3_processor, sam3_state, prompt):
    """Run SAM3 detection on a single camera image and return detection data + annotated image."""
    if not HAS_SAM3 or sam3_processor is None:
        return [], image_np.copy()

    # SAM3 expects RGB PIL Image
    image_rgb = cv2.cvtColor(image_np, cv2.COLOR_BGRA2RGB)
    pil_img = Image.fromarray(image_rgb)
    
    # Set image and prompt
    sam3_state = sam3_processor.set_image(pil_img, state=sam3_state)
    sam3_processor.reset_all_prompts(sam3_state)
    sam3_state = sam3_processor.set_text_prompt(prompt, state=sam3_state)
    
    boxes = sam3_state.get("boxes", [])
    masks = sam3_state.get("masks", [])
    scores = sam3_state.get("scores", [])
    
    detections = []
    annotated_img = image_np.copy()
    
    # Convert to CPU if they are tensors
    if torch.is_tensor(boxes): boxes = boxes.cpu().numpy()
    if torch.is_tensor(masks): masks = masks.cpu().numpy()
    if torch.is_tensor(scores): scores = scores.cpu().numpy()
    
    for i in range(len(boxes)):
        mask = masks[i]
        if mask.ndim == 3: # Squeeze if [1, H, W]
            mask = mask.squeeze(0)
            
        xyz_trans = get_robust_3d(mask, xyz_np)
        
        box = boxes[i].tolist() # [x1, y1, x2, y2]
        score = float(scores[i])
        
        detections.append({
            "cam": cam_name,
            "label": "tennis_ball",
            "confidence": score,
            "bbox_xyxy": box,
            "translation_xyz": xyz_trans
        })
        
        # Draw on annotated image
        x1, y1, x2, y2 = map(int, box)
        cv2.rectangle(annotated_img, (x1, y1), (x2, y2), (0, 255, 0), 2)
        cv2.putText(annotated_img, f"ball {score:.2f}", (x1, y1-10), 
                    cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 2)
        
    return detections, annotated_img

def export_synced_frame(cameras, timestamps, output_dir, sam3_processor, sam3_state):
    """Export images, point clouds, and detections for all cameras."""
    ref_ts = timestamps["front"]
    frame_dir = os.path.join(output_dir, str(ref_ts))
    os.makedirs(frame_dir, exist_ok=True)
    
    all_detections = []
    
    # Prompt as a union of the two requested strings
    combined_prompt = "small tennis ball . tennis ball"
    
    for cam_name in CAMERA_NAMES:
        zed = cameras[cam_name]
        
        # Prepare containers
        image_mat = sl.Mat()
        xyz_mat = sl.Mat()
        
        # Retrieve
        zed.retrieve_image(image_mat, sl.VIEW.LEFT)
        zed.retrieve_measure(xyz_mat, sl.MEASURE.XYZ)
        
        image_np = image_mat.get_data()
        xyz_np = xyz_mat.get_data()
        
        # Process Detections
        cam_detections, annotated_img = process_detections(cam_name, image_np, xyz_np, sam3_processor, sam3_state, combined_prompt)
        all_detections.extend(cam_detections)
        
        # Save ORIGINAL image
        img_path = os.path.join(frame_dir, f"{cam_name}.jpg")
        cv2.imwrite(img_path, image_np, [int(cv2.IMWRITE_JPEG_QUALITY), 95])

        # Save ANNOTATED image
        annotated_path = os.path.join(frame_dir, f"{cam_name}_annotated.jpg")
        cv2.imwrite(annotated_path, annotated_img, [int(cv2.IMWRITE_JPEG_QUALITY), 95])
        
        # Save XYZ point cloud (x, y, z for each pixel)
        xyz_path = os.path.join(frame_dir, f"{cam_name}_xyz.npy")
        np.save(xyz_path, xyz_np)
        
    # Save all detections for this frame
    det_path = os.path.join(frame_dir, "ball_detections.json")
    with open(det_path, 'w') as f:
        json.dump(all_detections, f, indent=4)
        
    return True

def main():
    args = parse_args()
    
    # 1. Initialize SAM3
    sam3_processor = None
    sam3_state = {}
    if HAS_SAM3:
        print("Initializing SAM3...")
        if torch.cuda.is_available():
            torch.backends.cuda.matmul.allow_tf32 = True
            torch.backends.cudnn.allow_tf32 = True
            
        # Build Model
        sam3_root = os.path.join(project_root, "sam3")
        bpe_path = os.path.join(sam3_root, "assets", "bpe_simple_vocab_16e6.txt.gz")
        if not os.path.exists(bpe_path):
             bpe_path = None 

        model = build_sam3_image_model(bpe_path=bpe_path)
        sam3_processor = Sam3Processor(model, confidence_threshold=args.conf)
        print("SAM3 Initialized.")

    # 2. Initialize cameras
    svo_files = {name: os.path.join(args.input_dir, f"{name}.svo2") for name in CAMERA_NAMES}
    for path in svo_files.values():
        if not os.path.isfile(path):
            alt_path = path.replace(".svo2", ".svo")
            if not os.path.isfile(alt_path):
                print(f"Error: Missing SVO file: {path}")
                sys.exit(1)
            svo_files[next(k for k,v in svo_files.items() if v==path)] = alt_path

    depth_mode = sl.DEPTH_MODE.PERFORMANCE if args.fast_depth else sl.DEPTH_MODE.NEURAL_PLUS
    print(f"Initializing 4 cameras (Depth: {'PERFORMANCE' if args.fast_depth else 'NEURAL_PLUS'})...")
    
    cameras = {}
    current_ts = {}
    
    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = [executor.submit(open_camera, name, svo_files[name], depth_mode) for name in CAMERA_NAMES]
        for future in futures:
            name, result = future.result()
            if result:
                cameras[name], current_ts[name] = result
            else:
                for c, _ in cameras.values(): c.close()
                sys.exit(1)

    # 3. Initial Alignment
    print("\nAligning starting positions...")
    max_start_ts = max(current_ts.values())
    
    for cam_name in CAMERA_NAMES:
        zed = cameras[cam_name]
        ts = current_ts[cam_name]
        while ts < max_start_ts - 33_000_000:
            if zed.grab() != sl.ERROR_CODE.SUCCESS:
                break
            ts = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
            current_ts[cam_name] = ts
            
    # 4. Streaming Sync Loop
    print("Starting synchronized detection export...")
    exported_count = 0
    
    try:
        while True:
            ts_values = list(current_ts.values())
            min_ts = min(ts_values)
            max_ts = max(ts_values)
            
            if (max_ts - min_ts) <= SYNC_TOLERANCE_NS:
                # SUCCESS: Export synchronized frame with detections
                export_synced_frame(cameras, current_ts, args.output_dir, sam3_processor, sam3_state)
                exported_count += 1
                sys.stdout.write(f"\r  Exported {exported_count} frames")
                sys.stdout.flush()
                
                if args.max_frames and exported_count >= args.max_frames:
                    break
                    
                # Advance ALL
                failed_cam = None
                for cam_name in CAMERA_NAMES:
                    if cameras[cam_name].grab() != sl.ERROR_CODE.SUCCESS:
                        failed_cam = cam_name
                        break
                    current_ts[cam_name] = cameras[cam_name].get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                
                if failed_cam:
                    print(f"\nEnd of SVO reached for {failed_cam}")
                    break
            else:
                # FAILURE: Advance ONLY laggard
                laggard = min(current_ts, key=current_ts.get)
                if cameras[laggard].grab() != sl.ERROR_CODE.SUCCESS:
                    break
                current_ts[laggard] = cameras[laggard].get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                
    except KeyboardInterrupt:
        print("\nInterrupted.")
    
    print(f"\n\nDone. {exported_count} frames saved to {args.output_dir}")
    for zed in cameras.values():
        zed.close()

if __name__ == "__main__":
    main()
