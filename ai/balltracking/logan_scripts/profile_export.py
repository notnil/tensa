import time
import os
import pyzed.sl as sl

def profile_svo(svo_path):
    print(f"Profiling {svo_path}...")
    
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.depth_mode = sl.DEPTH_MODE.NONE
    
    zed = sl.Camera()
    
    start = time.time()
    err = zed.open(init_params)
    end = time.time()
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Failed to open: {err}")
        return
    print(f"  Open (DEPTH_MODE.NONE) took: {end - start:.4f}s")
    
    nb_frames = zed.get_svo_number_of_frames()
    print(f"  Frames: {nb_frames}")
    
    # Profile grab()
    start = time.time()
    for i in range(10):
        zed.grab()
    end = time.time()
    print(f"  10 sequential grab() took: {end - start:.4f}s ({(end - start)/10:.4f}s per grab)")
    
    # Profile set_svo_position() + grab()
    start = time.time()
    for i in range(5):
        pos = (i + 1) * (nb_frames // 10)
        zed.set_svo_position(pos)
        zed.grab()
    end = time.time()
    print(f"  5 jump + grab() took: {end - start:.4f}s ({(end - start)/5:.4f}s per jump)")
    
    zed.close()
    
    # Profile NEURAL_PLUS open
    init_params.depth_mode = sl.DEPTH_MODE.NEURAL_PLUS
    start = time.time()
    err = zed.open(init_params)
    end = time.time()
    if err == sl.ERROR_CODE.SUCCESS:
        print(f"  Open (NEURAL_PLUS) took: {end - start:.4f}s")
        zed.close()
    else:
        print(f"  Failed to open (NEURAL_PLUS): {err}")

if __name__ == "__main__":
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    svo = os.path.join(project_root, "data/zed-recordings-multicam/2025-12-29T17-24-38/front.svo2")
    if os.path.exists(svo):
        profile_svo(svo)
    else:
        print(f"File not found: {svo}")





