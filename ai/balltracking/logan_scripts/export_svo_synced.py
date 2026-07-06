import sys
import argparse
import os
import numpy as np
import cv2
import time
import json
from concurrent.futures import ThreadPoolExecutor

try:
    import pyzed.sl as sl
except ImportError:
    print("Error: pyzed not found. Please install the ZED SDK and Python API.")
    sys.exit(1)

# Camera names in the multi-camera setup
CAMERA_NAMES = ["front", "back", "left", "right"]

# Synchronization tolerance in nanoseconds (10ms = 10,000_000 ns)
SYNC_TOLERANCE_NS = 10_000_000

def parse_args():
    """Parse command line arguments for streaming synchronized SVO export."""
    parser = argparse.ArgumentParser(
        description="Streaming export of synchronized images and depth from 4 ZED SVOs."
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
        "--fast-depth",
        action="store_true",
        help="Use PERFORMANCE depth mode instead of NEURAL_PLUS"
    )
    parser.add_argument(
        "--skip",
        type=int,
        default=1,
        help="Only export every Nth synchronized frame (default: 1, export all)"
    )
    parser.add_argument(
        "--offset",
        type=int,
        default=1,
        help="Start exporting from the Nth synchronized frame (default: 1)"
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

def export_synced_frame(cameras, timestamps, output_dir, exported_count):
    """Export images and XYZ point clouds for all cameras at the current synchronized state."""
    ref_ts = timestamps["front"]
    frame_dir = os.path.join(output_dir, str(ref_ts))
    os.makedirs(frame_dir, exist_ok=True)
    
    for cam_name in CAMERA_NAMES:
        zed = cameras[cam_name]
        
        # Prepare containers
        image_left = sl.Mat()
        image_right = sl.Mat()
        point_cloud = sl.Mat()
        
        # Retrieve
        zed.retrieve_image(image_left, sl.VIEW.LEFT)
        zed.retrieve_image(image_right, sl.VIEW.RIGHT)
        zed.retrieve_measure(point_cloud, sl.MEASURE.XYZ)
        
        # Save images
        left_img_path = os.path.join(frame_dir, f"{cam_name}.jpg")
        right_img_path = os.path.join(frame_dir, f"{cam_name}_right.jpg")
        cv2.imwrite(left_img_path, image_left.get_data(), [int(cv2.IMWRITE_JPEG_QUALITY), 95])
        cv2.imwrite(right_img_path, image_right.get_data(), [int(cv2.IMWRITE_JPEG_QUALITY), 95])
        
        # Save XYZ point cloud (x, y, z for each pixel)
        xyz_path = os.path.join(frame_dir, f"{cam_name}_xyz.npy")
        np.save(xyz_path, point_cloud.get_data())
        
    return True

def main():
    args = parse_args()
    
    # 1. Initialize cameras
    svo_files = {name: os.path.join(args.input_dir, f"{name}.svo2") for name in CAMERA_NAMES}
    for path in svo_files.values():
        if not os.path.isfile(path):
            # Try .svo extension if .svo2 is missing
            if not os.path.isfile(path.replace(".svo2", ".svo")):
                print(f"Error: Missing SVO file: {path}")
                sys.exit(1)
            svo_files[next(k for k,v in svo_files.items() if v==path)] = path.replace(".svo2", ".svo")

    depth_mode = sl.DEPTH_MODE.PERFORMANCE if args.fast_depth else sl.DEPTH_MODE.NEURAL_PLUS
    print(f"Initializing 4 cameras (Depth: {'PERFORMANCE' if args.fast_depth else 'NEURAL_PLUS'})...")
    
    cameras = {}
    current_ts = {}
    
    for name in CAMERA_NAMES:
        name, result = open_camera(name, svo_files[name], depth_mode)
        if result:
            cameras[name], current_ts[name] = result
        else:
            print(f"Error: Failed to open {name}")
            for c in cameras.values(): c.close()
            sys.exit(1)

    # 1.5 Save Calibration Data
    print("\nSaving calibration data...")
    calibration_data = {}
    for cam_name in CAMERA_NAMES:
        zed = cameras[cam_name]
        cam_info = zed.get_camera_information()
        calib = cam_info.camera_configuration.calibration_parameters
        
        calibration_data[cam_name] = {
            "fx": calib.left_cam.fx,
            "fy": calib.left_cam.fy,
            "cx": calib.left_cam.cx,
            "cy": calib.left_cam.cy,
            "baseline": calib.get_camera_baseline()
        }
    
    os.makedirs(args.output_dir, exist_ok=True)
    calib_path = os.path.join(args.output_dir, "calibration.json")
    with open(calib_path, 'w') as f:
        json.dump(calibration_data, f, indent=2)
    print(f"Calibration saved to {calib_path}")

    # 2. Initial Alignment (Fast-forward lagging cameras to roughly the same start)
    print("\nAligning starting positions...")
    max_start_ts = max(current_ts.values())
    
    for cam_name in CAMERA_NAMES:
        zed = cameras[cam_name]
        ts = current_ts[cam_name]
        # Fast forward until we are within ~33ms of the leader
        while ts < max_start_ts - 33_000_000:
            if zed.grab() != sl.ERROR_CODE.SUCCESS:
                print(f"Warning: {cam_name} reached end during initial alignment.")
                break
            ts = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
            current_ts[cam_name] = ts
            
    # 3. Streaming Sync Loop
    skip_info = f", exporting every {args.skip}th frame" if args.skip > 1 else ""
    offset_info = f" starting from offset {args.offset}" if args.offset > 1 else ""
    print(f"Starting synchronized export{skip_info}{offset_info}...")
    exported_count = 0
    sync_count = 0  # Total synchronized frames found (before skipping)
    
    try:
        while True:
            # Check for sync
            ts_values = list(current_ts.values())
            min_ts = min(ts_values)
            max_ts = max(ts_values)
            
            if (max_ts - min_ts) <= SYNC_TOLERANCE_NS:
                # SUCCESS: Found a synchronized frame
                sync_count += 1
                
                # Only export if this matches the skip and offset
                should_export = (sync_count % args.skip == args.offset % args.skip) or (args.skip == 1)
                
                if should_export:
                    ref_ts = current_ts["front"]
                    frame_dir = os.path.join(args.output_dir, str(ref_ts))
                    if not os.path.exists(frame_dir):
                        export_synced_frame(cameras, current_ts, args.output_dir, exported_count)
                    
                    exported_count += 1
                    if exported_count % 10 == 0:
                        sys.stdout.write(f"\r  Exported {exported_count} frames (scanned {sync_count} synced frames)")
                        sys.stdout.flush()
                
                if args.max_frames and exported_count >= args.max_frames:
                    break
                    
                # Advance ALL cameras
                failed_cam = None
                for cam_name in CAMERA_NAMES:
                    zed = cameras[cam_name]
                    if zed.grab() != sl.ERROR_CODE.SUCCESS:
                        failed_cam = cam_name
                        break
                    current_ts[cam_name] = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                
                if failed_cam:
                    print(f"\nEnd of SVO reached for {failed_cam}")
                    break
            else:
                # FAILURE: Advance ONLY the laggard(s)
                # Find the camera with the minimum timestamp
                laggard = min(current_ts, key=current_ts.get)
                zed = cameras[laggard]
                if zed.grab() != sl.ERROR_CODE.SUCCESS:
                    print(f"\nEnd of SVO reached for laggard {laggard}")
                    break
                current_ts[laggard] = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                
    except KeyboardInterrupt:
        print("\nExport interrupted by user.")
    
    print(f"\n\nExport complete. {exported_count} frames saved to {args.output_dir} (from {sync_count} synced frames)")
    for zed in cameras.values():
        zed.close()

if __name__ == "__main__":
    main()

