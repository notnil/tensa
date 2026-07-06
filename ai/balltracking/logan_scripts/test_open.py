import pyzed.sl as sl
import sys
init = sl.InitParameters()
init.set_from_svo_file("/home/logan/Documents/data/tensa-recordings/2026-01-17T08-46-52/front.svo2")
init.svo_real_time_mode = False
zed = sl.Camera()
if zed.open(init) == sl.ERROR_CODE.SUCCESS:
    print("OPEN SUCCESS")
    zed.close()
else:
    print("OPEN FAILED")
