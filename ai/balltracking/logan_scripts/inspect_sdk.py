import pyzed.sl as sl

def inspect_sdk():
    print(f"SDK Version: {sl.Camera.get_sdk_version()}")
    
    sensors_data = sl.SensorsData()
    imu_data = sensors_data.get_imu_data()
    
    # Inspect angular velocity
    av = imu_data.get_angular_velocity()
    print(f"get_angular_velocity return type: {type(av)}")
    print(f"get_angular_velocity content: {av}")

    # Inspect orientation
    orient = imu_data.get_pose().get_orientation()
    print(f"get_orientation return type: {type(orient)}")
    
if __name__ == "__main__":
    inspect_sdk()
