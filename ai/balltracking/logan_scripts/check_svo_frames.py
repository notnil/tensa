import sys
import pyzed.sl as sl
import os

def check_svo(path):
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(path)
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Error opening {path}: {err}")
        return
    
    count = zed.get_svo_number_of_frames()
    print(f"{os.path.basename(path)}: {count} frames")
    zed.close()

if __name__ == "__main__":
    input_dir = "/home/logan/Documents/data/tensa-recordings/2026-01-17T08-46-52"
    for name in ["front", "back", "left", "right"]:
        path = os.path.join(input_dir, f"{name}.svo2")
        if os.path.exists(path):
            check_svo(path)
        else:
            print(f"File not found: {path}")
