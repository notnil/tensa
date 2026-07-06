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

# INCREASED TOLERANCE: 33ms instead of 1ms
SYNC_TOLERANCE_NS = 33_000_000

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("input_dir", type=str)
    parser.add_argument("output_dir", type=str)
    parser.add_argument("--max-frames", type=int, default=10) # Small test
    return parser.parse_args()

def open_camera(cam_name, svo_path, depth_mode):
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.coordinate_units = sl.UNIT.METER
    init_params.depth_mode = depth_mode
    
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        return cam_name, None
    
    if zed.grab() != sl.ERROR_CODE.SUCCESS:
        zed.close()
        return cam_name, None
        
    ts = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
    return cam_name, (zed, ts)

def main():
    args = parse_args()
    svo_files = {name: os.path.join(args.input_dir, f"{name}.svo2") for name in CAMERA_NAMES}
    depth_mode = sl.DEPTH_MODE.NONE # FAST FOR TEST
    
    cameras = {}
    current_ts = {}
    
    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = [executor.submit(open_camera, name, svo_files[name], depth_mode) for name in CAMERA_NAMES]
        for future in futures:
            name, result = future.result()
            if result:
                cameras[name], current_ts[name] = result
            else:
                print(f"Failed to open {name}")
                sys.exit(1)

    print("\nStarting synchronized export TEST...")
    exported_count = 0
    sync_count = 0
    
    try:
        while exported_count < args.max_frames:
            ts_values = list(current_ts.values())
            min_ts = min(ts_values)
            max_ts = max(ts_values)
            diff = max_ts - min_ts
            
            if diff <= SYNC_TOLERANCE_NS:
                sync_count += 1
                ref_ts = current_ts["front"]
                print(f"SYNC FOUND: Frame {exported_count}, TS: {ref_ts}, Diff: {diff/1e6:.2f}ms")
                exported_count += 1
                    
                # Advance ALL
                for cam_name in CAMERA_NAMES:
                    zed = cameras[cam_name]
                    if zed.grab() != sl.ERROR_CODE.SUCCESS:
                        break
                    current_ts[cam_name] = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
            else:
                # Advance only laggard
                laggard = min(current_ts, key=current_ts.get)
                zed = cameras[laggard]
                if zed.grab() != sl.ERROR_CODE.SUCCESS:
                    break
                current_ts[laggard] = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                
    except KeyboardInterrupt:
        pass
    
    print(f"\nTest complete. Found {exported_count} frames.")
    for zed in cameras.values():
        zed.close()

if __name__ == "__main__":
    main()
