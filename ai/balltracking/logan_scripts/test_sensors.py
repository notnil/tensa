import sys
import pyzed.sl as sl

def test_sensor_read(svo_path):
    print(f"Testing SVO: {svo_path}")
    
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.depth_mode = sl.DEPTH_MODE.NONE # Disable depth for speed
    
    zed = sl.Camera()
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Error opening SVO: {err}")
        return

    # Check if sensors are present in metadata
    info = zed.get_camera_information()
    print(f"Camera Model: {info.camera_model}")
    print(f"Sensors Firmware: {info.sensors_configuration.firmware_version}")
    print(f"IMU Available: {info.sensors_configuration.is_sensor_available(sl.SENSOR_TYPE.GYROSCOPE)}")
    print(f"Magnetometer Available: {info.sensors_configuration.is_sensor_available(sl.SENSOR_TYPE.MAGNETOMETER)}")
    
    sensors_data = sl.SensorsData()
    rt_param = sl.RuntimeParameters()
    
    # Try to grab a few frames and check sensor data
    for i in range(10):
        if zed.grab(rt_param) == sl.ERROR_CODE.SUCCESS:
            # Try TIME_REFERENCE.IMAGE (synced with image)
            err_img = zed.get_sensors_data(sensors_data, sl.TIME_REFERENCE.IMAGE)
            
            # Try TIME_REFERENCE.CURRENT (latest available)
            sensors_data_curr = sl.SensorsData()
            err_curr = zed.get_sensors_data(sensors_data_curr, sl.TIME_REFERENCE.CURRENT)
            
            print(f"\nFrame {i}:")
            print(f"  get_sensors_data(IMAGE)   -> {err_img}")
            if err_img == sl.ERROR_CODE.SUCCESS:
                imu = sensors_data.get_imu_data()
                print(f"    IMU timestamp: {imu.timestamp.get_milliseconds()}")
                print(f"    Ang Vel: {imu.get_angular_velocity()}")
            
            print(f"  get_sensors_data(CURRENT) -> {err_curr}")
            if err_curr == sl.ERROR_CODE.SUCCESS:
                imu = sensors_data_curr.get_imu_data()
                print(f"    IMU timestamp: {imu.timestamp.get_milliseconds()}")

    zed.close()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python test_sensors.py <svo_file>")
    else:
        test_sensor_read(sys.argv[1])
