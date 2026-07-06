import pyzed.sl as sl
import sys

print(f"ZED SDK Version: {sl.Camera().get_sdk_version()}")
# Check if CUDA is available
# sl.Camera().get_device_list() is not directly available but we can check sl.ERROR_CODE
cameras = sl.Camera.get_device_list()
print(f"Detected cameras: {len(cameras)}")
for cam in cameras:
    print(f"  - {cam.serial_number}")
