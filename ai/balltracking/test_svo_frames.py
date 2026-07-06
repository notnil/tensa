#!/usr/bin/env python3
"""Test script to extract and inspect raw frames from SVO file."""

import sys
import argparse
import cv2
import numpy as np
import pyzed.sl as sl


def main(svo_path):
    print(f"Opening SVO: {svo_path}")

    # Initialize ZED
    zed = sl.Camera()
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(svo_path)
    init_params.svo_real_time_mode = False
    init_params.enable_image_validity_check = 1 # check for frame corruption

    status = zed.open(init_params)
    if status != sl.ERROR_CODE.SUCCESS:
        print(f"Failed to open SVO: {status}")
        sys.exit(1)

    # Get camera info
    cam_info = zed.get_camera_information()
    print(f"Resolution: {cam_info.camera_configuration.resolution.width}x{cam_info.camera_configuration.resolution.height}")
    print(f"FPS: {cam_info.camera_configuration.fps}")
    print(f"Total frames: {zed.get_svo_number_of_frames()}")

    # Test different view types
    views_to_test = [
        ("LEFT", sl.VIEW.LEFT),
        ("LEFT_UNRECTIFIED", sl.VIEW.LEFT_UNRECTIFIED),
        ("RIGHT", sl.VIEW.RIGHT),
    ]

    runtime_params = sl.RuntimeParameters()

    # Grab first frame
    err = zed.grab(runtime_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Failed to grab frame: {err}")
        zed.close()
        sys.exit(1)

    print("\nTesting different VIEW types:")
    print("-" * 80)

    for view_name, view_type in views_to_test:
        image_mat = sl.Mat()
        zed.retrieve_image(image_mat, view_type)
        data = image_mat.get_data()

        print(f"\n{view_name}:")
        print(f"  Shape: {data.shape}")
        print(f"  Dtype: {data.dtype}")
        print(f"  Min/Max: {data.min()}/{data.max()}")

        if len(data.shape) == 3 and data.shape[2] >= 3:
            means = data.mean(axis=(0, 1))
            print(f"  Mean per channel: {means}")

            # Check if only green channel has data
            if data.shape[2] == 4:  # BGRA (ZED SDK channel order)
                b, g, r, a = means
                if r < 5 and b < 5 and g > 50:
                    print(f"  ⚠️  WARNING: Only green channel has significant data!")
            elif data.shape[2] == 3:  # RGB or BGR
                c0, c1, c2 = means
                if c0 < 5 and c2 < 5 and c1 > 50:
                    print(f"  ⚠️  WARNING: Only middle channel has significant data!")

        # Save frame
        filename = f"test_{view_name.lower()}_frame0.png"
        cv2.imwrite(filename, data)
        print(f"  Saved: {filename}")

    print("\n" + "=" * 80)
    print("DIAGNOSIS:")
    print("=" * 80)
    print("If all views show only green channel data, the SVO file itself is corrupted.")
    print("This likely means the video was recorded or encoded incorrectly.")
    print("\nPossible causes:")
    print("  1. Corrupted SVO file during recording")
    print("  2. Wrong codec or compression settings")
    print("  3. Hardware/camera malfunction during recording")
    print("  4. File transfer corruption")

    zed.close()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Test SVO file frame extraction")
    parser.add_argument("--svo", type=str, required=True, help="Path to SVO file")
    args = parser.parse_args()
    main(args.svo)
