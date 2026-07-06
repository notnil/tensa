import sys
import argparse
import os
import json
from concurrent.futures import ThreadPoolExecutor

try:
    import pyzed.sl as sl
except ImportError:
    print("Error: pyzed not found.")
    sys.exit(1)

CAMERA_NAMES = ["front", "back", "left", "right"]

def open_camera(cam_name, svo_path):
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.depth_mode = sl.DEPTH_MODE.NONE
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        return cam_name, None
    if zed.grab() != sl.ERROR_CODE.SUCCESS:
        zed.close()
        return cam_name, None
    return cam_name, (zed, zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds())

def main():
    input_dir = "/home/logan/Documents/data/tensa-recordings/2026-01-17T08-46-52"
    svo_files = {name: os.path.join(input_dir, f"{name}.svo2") for name in CAMERA_NAMES}
    
    cameras = {}
    current_ts = {}
    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = [executor.submit(open_camera, name, svo_files[name]) for name in CAMERA_NAMES]
        for future in futures:
            name, result = future.result()
            if result:
                cameras[name], current_ts[name] = result

    print("Analyzing drift for 100 loops...")
    for i in range(100):
        ts_values = list(current_ts.values())
        min_ts = min(ts_values)
        max_ts = max(ts_values)
        diff = max_ts - min_ts
        
        print(f"Step {i:03}: Diff {diff/1e6:6.3f}ms | " + " ".join([f"{n}: {current_ts[n] % 1_000_000_000 / 1e6:6.2f}" for n in CAMERA_NAMES]))
        
        # Logic from export script: if sync, advance all. Else advance laggard.
        if diff <= 1_000_000: # 1ms
            for name in CAMERA_NAMES:
                if cameras[name].grab() == sl.ERROR_CODE.SUCCESS:
                    current_ts[name] = cameras[name].get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
        else:
            laggard = min(current_ts, key=current_ts.get)
            if cameras[laggard].grab() == sl.ERROR_CODE.SUCCESS:
                current_ts[laggard] = cameras[laggard].get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
            else:
                break

    for zed, _ in cameras.values(): zed.close()

if __name__ == "__main__":
    main()
