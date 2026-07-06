import sys
import pyzed.sl as sl
import os

def check_timestamps(input_dir, num_frames=10):
    cameras = ["front", "back", "left", "right"]
    data = {cam: [] for cam in cameras}
    
    for cam in cameras:
        path = os.path.join(input_dir, f"{cam}.svo2")
        print(f"Opening {cam}...")
        init_params = sl.InitParameters()
        init_params.set_from_svo_file(path)
        init_params.svo_real_time_mode = False
        zed = sl.Camera()
        err = zed.open(init_params)
        if err != sl.ERROR_CODE.SUCCESS:
            print(f"Error opening {cam}: {err}")
            continue
        
        print(f"Reading {num_frames} frames from {cam}...")
        for i in range(num_frames):
            if zed.grab() == sl.ERROR_CODE.SUCCESS:
                ts = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
                data[cam].append(ts)
            else:
                break
        zed.close()
    
    # Print comparison
    print("\nTimestamp Comparison (first 10 frames):")
    for i in range(num_frames):
        row = f"Frame {i}: "
        timestamps = []
        for cam in cameras:
            if i < len(data[cam]):
                timestamps.append(data[cam][i])
                row += f"{cam}: {data[cam][i]}  "
            else:
                row += f"{cam}: N/A  "
        
        if len(timestamps) == 4:
            diff = max(timestamps) - min(timestamps)
            row += f" | MAX DIFF: {diff/1e6:.3f} ms"
        print(row)

if __name__ == "__main__":
    input_dir = "/home/logan/Documents/data/tensa-recordings/2026-01-17T08-46-52"
    check_timestamps(input_dir)
