#!/usr/bin/env python3
"""Export a ZED SVO file to MP4."""

import argparse
import sys

import cv2
import pyzed.sl as sl
from tqdm import tqdm


def run(args):
    zed = sl.Camera()
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(args.svo)
    init_params.svo_real_time_mode = False

    # disable the SDK's internal depth engine to save GPU.
    init_params.depth_mode = sl.DEPTH_MODE.NONE

    status = zed.open(init_params)
    if status != sl.ERROR_CODE.SUCCESS:
        print(f"Failed to open SVO: {status}")
        sys.exit(1)

    cam_info = zed.get_camera_information()
    cam_res = cam_info.camera_configuration.resolution
    fps = cam_info.camera_configuration.fps
    width, height = cam_res.width, cam_res.height
    total_frames = zed.get_svo_number_of_frames()
    print(f"Camera: {width}x{height} @ {fps}fps, {total_frames} frames")

    fourcc = cv2.VideoWriter_fourcc(*"mp4v")
    writer = cv2.VideoWriter(args.output, fourcc, fps, (width, height))

    image_mat = sl.Mat()
    runtime_params = sl.RuntimeParameters()

    for _ in tqdm(range(total_frames), desc="Exporting frames"):
        err = zed.grab(runtime_params)
        if err == sl.ERROR_CODE.END_OF_SVOFILE_REACHED:
            break
        if err != sl.ERROR_CODE.SUCCESS:
            tqdm.write(f"Grab error: {err}")
            break

        # online_runner.py runs detections on rectified LEFT/RIGHT frames, so
        # the exported MP4 must use the same rectified left view for overlays
        # to line up in visualize_online_results_rerun.py.
        zed.retrieve_image(image_mat, sl.VIEW.LEFT)
        frame_bgra = image_mat.get_data()
        frame_bgr = cv2.cvtColor(frame_bgra, cv2.COLOR_BGRA2BGR)
        writer.write(frame_bgr)

    writer.release()
    zed.close()
    print(f"Wrote to {args.output}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Export ZED SVO to MP4")
    parser.add_argument("--svo", type=str, required=True, help="Path to SVO file")
    parser.add_argument("--output", type=str, default="output.mp4", help="Output MP4 path")
    args = parser.parse_args()

    run(args)
