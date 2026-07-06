import sys
import pyzed.sl as sl
import os

def check_svo_metadata(path):
    print(f"Opening {path}...")
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(path)
    init_params.svo_real_time_mode = False
    init_params.depth_mode = sl.DEPTH_MODE.NONE # FAST
    
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Error opening {path}: {err}")
        return
    
    count = zed.get_svo_number_of_frames()
    info = zed.get_camera_information()
    fps = info.camera_configuration.fps
    
    # Get first timestamp
    zed.set_svo_position(0)
    zed.grab()
    ts_start = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
    
    # Get last timestamp
    zed.set_svo_position(count - 1)
    zed.grab()
    ts_end = zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
    
    print(f"File: {os.path.basename(path)}")
    print(f"Total Frames: {count}")
    print(f"FPS: {fps}")
    print(f"Start TS: {ts_start}")
    print(f"End TS:   {ts_end}")
    print(f"Duration: {(ts_end - ts_start)/1e9:.2f} seconds")
    
    zed.close()

if __name__ == "__main__":
    input_dir = "/home/logan/Documents/data/tensa-recordings/2026-01-17T08-46-52"
    for cam in ["front", "back", "left", "right"]:
        path = os.path.join(input_dir, f"{cam}.svo2")
        if os.path.exists(path):
            check_svo_metadata(path)
        else:
            print(f"Not found: {path}")
