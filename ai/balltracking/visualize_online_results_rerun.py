#!/usr/bin/env python3
#
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "numpy>=1.24",
#   "rerun-sdk>=0.29.2",
# ]
# ///
#
"""Visualize online_runner.py JSONL results in Rerun, synchronized with MP4 frames."""

import argparse
import json
from collections import defaultdict
from pathlib import Path

import numpy as np
import rerun as rr
import rerun.blueprint as rrb

import court_dims

MAX_ASSET_VIDEO_BYTES = 2_147_000_000  # Practical limit from 32-bit C int/Arrow blob path.


def log_tennis_court():
    """Render tennis court lines and net using court_dims geometry."""
    sections_rub = []
    for section in court_dims.Section.primary_sections():
        section_court = [(*kp.coordinates, 0.0) for kp in section.key_points]
        section_court.append(section_court[0])
        sections_rub.append([tuple(pt) for pt in section_court])

    rr.log(
        "world/court/lines",
        rr.LineStrips3D(sections_rub, colors=[(255, 255, 255)]),
        static=True,
    )

    net_court = np.array(
        [
            [-court_dims.NET_X, 0.0, 0.0],
            [-court_dims.NET_X, 0.0, court_dims.NET_HEIGHT],
            [court_dims.NET_X, 0.0, court_dims.NET_HEIGHT],
            [court_dims.NET_X, 0.0, 0.0],
        ],
        dtype=float,
    )
    rr.log(
        "world/court/net",
        rr.LineStrips3D([[tuple(pt) for pt in net_court]], colors=[(0, 255, 0)]),
        static=True,
    )


def load_results(jsonl_path):
    """Load online_runner event output indexed by frame."""
    track_points_by_track = defaultdict(dict)
    bounces_by_frame = defaultdict(list)
    frame_processed = {}
    run_summary = None

    with open(jsonl_path, "r", encoding="utf-8") as stream:
        for line_number, line in enumerate(stream, start=1):
            line = line.strip()
            if not line:
                continue

            try:
                event = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"Invalid JSON on line {line_number} of {jsonl_path}: {exc}") from exc

            event_type = event.get("type")
            if event_type == "trajectory_segment_finalized":
                track_id = int(event["track_id"])
                for point in event.get("points", []):
                    frame_index = int(point["frame_index"])
                    track_points_by_track[track_id][frame_index] = (
                        float(point["x"]),
                        float(point["y"]),
                        float(point["z"]),
                    )
            elif event_type == "bounce_finalized":
                frame_index = int(event["frame_index"])
                bounces_by_frame[frame_index].append(
                    (
                        int(event["track_id"]),
                        (float(event["x"]), float(event["y"]), float(event["z"])),
                    )
                )
            elif event_type == "frame_processed":
                frame_index = int(event["frame_index"])
                frame_processed[frame_index] = {
                    "timestamp_ns": int(event["timestamp_ns"]),
                    "active_track_count": int(event.get("active_track_count", 0)),
                }
            elif event_type == "run_summary":
                run_summary = event

    track_series = {}
    for track_id, points_by_frame in track_points_by_track.items():
        series = sorted(points_by_frame.items(), key=lambda item: item[0])
        track_series[track_id] = series

    return track_series, bounces_by_frame, frame_processed, run_summary


def load_detections(jsonl_path):
    """Load non-online per-frame detection payloads from detections.jsonl."""
    detections_by_frame = {}
    totals = {"total_frames": 0}

    with open(jsonl_path, "r", encoding="utf-8") as stream:
        for line_number, line in enumerate(stream, start=1):
            line = line.strip()
            if not line:
                continue

            try:
                event = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"Invalid JSON on line {line_number} of {jsonl_path}: {exc}") from exc

            if event.get("type") != "detections":
                continue

            totals["total_frames"] = int(event.get("total_frames", totals["total_frames"]))
            for frame_entry in event.get("frames", []):
                frame_index = int(frame_entry["frame_index"])
                frame_payload = {
                    "timestamp_ns": int(frame_entry.get("timestamp_ns", 0)),
                    "left_detections": [
                        {
                            "index": int(det["index"]),
                            "confidence": float(det["confidence"]),
                            "bbox": [float(v) for v in det["bbox"]],
                        }
                        for det in frame_entry.get("left_detections", [])
                    ],
                    "right_detections": [
                        {
                            "index": int(det["index"]),
                            "confidence": float(det["confidence"]),
                            "bbox": [float(v) for v in det["bbox"]],
                        }
                        for det in frame_entry.get("right_detections", [])
                    ],
                    "triangulated_detections": [
                        {
                            "left_index": int(det.get("left_index", -1)),
                            "right_index": int(det.get("right_index", -1)),
                            "projection_3d": {
                                "camera_frame_position": {
                                    "x": float(det["projection_3d"]["camera_frame_position"]["x"]),
                                    "y": float(det["projection_3d"]["camera_frame_position"]["y"]),
                                    "z": float(det["projection_3d"]["camera_frame_position"]["z"]),
                                },
                                "camera_depth": float(det["projection_3d"]["camera_depth"]),
                                "disparity": float(det["projection_3d"]["disparity"]),
                                "court_frame_position": {
                                    "x": float(det["projection_3d"]["court_frame_position"]["x"]),
                                    "y": float(det["projection_3d"]["court_frame_position"]["y"]),
                                    "z": float(det["projection_3d"]["court_frame_position"]["z"]),
                                },
                            },
                        }
                        for det in frame_entry.get("triangulated_detections", [])
                        if det.get("projection_3d")
                        and det["projection_3d"].get("court_frame_position") is not None
                    ],
                }
                detections_by_frame[frame_index] = frame_payload

    return detections_by_frame, totals


def load_raw_trajectories(jsonl_path):
    """Load raw track trajectories from trajectories_raw.jsonl."""
    raw_trajectories = defaultdict(list)
    totals = {"total_tracks": 0}

    with open(jsonl_path, "r", encoding="utf-8") as stream:
        for line_number, line in enumerate(stream, start=1):
            line = line.strip()
            if not line:
                continue

            try:
                event = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"Invalid JSON on line {line_number} of {jsonl_path}: {exc}") from exc

            if event.get("type") != "trajectories_raw":
                continue

            totals["total_tracks"] = int(event.get("total_tracks", totals["total_tracks"]))
            for trajectory in event.get("trajectories", []):
                track_id = int(trajectory["track_id"])
                for point in trajectory.get("points", []):
                    raw_trajectories[track_id].append(
                        (
                            int(point["frame_index"]),
                            (
                                float(point["x"]),
                                float(point["y"]),
                                float(point["z"]),
                            ),
                        )
                    )

    track_series = {}
    for track_id, points in raw_trajectories.items():
        track_series[track_id] = sorted(points, key=lambda item: item[0])

    return track_series, totals


def init_rerun(run_name):
    rr.init(run_name, spawn=True)
    rr.log("/", rr.ViewCoordinates.RFU, static=True)

    bp = rrb.Blueprint(
        rrb.Horizontal(
            rrb.Spatial2DView(origin="video"),
            rrb.Spatial3DView(origin="world", background=(14, 18, 22)),
            column_shares=[1, 1],
        )
    )
    rr.send_blueprint(bp)

    rr.log(
        "world/court_axes",
        rr.Arrows3D(
            origins=[(0.0, 0.0, 0.0)] * 3,
            vectors=[(1.0, 0.0, 0.0), (0.0, 1.0, 0.0), (0.0, 0.0, 1.0)],
            colors=[(0, 220, 0), (0, 160, 255), (255, 140, 0)],
            labels=["+X court-right", "+Y court-forward", "+Z court-up"],
        ),
        static=True,
    )
    rr.log(
        "world/court_origin",
        rr.Points3D(positions=[(0.0, 0.0, 0.0)], colors=[(255, 255, 255)], radii=0.04),
        static=True,
    )
    log_tennis_court()


def log_robot_pose(machine_pose):
    yaw_deg = float(machine_pose["yaw"])
    yaw_rad = np.radians(yaw_deg - 90.0)
    cos_y, sin_y = np.cos(yaw_rad), np.sin(yaw_rad)
    rotation = np.array(
        [
            [cos_y, -sin_y, 0.0],
            [sin_y, cos_y, 0.0],
            [0.0, 0.0, 1.0],
        ],
        dtype=float,
    )

    machine_pos = np.array(
        [float(machine_pose["x"]), float(machine_pose["y"]), float(machine_pose.get("z", 0.0))],
        dtype=float,
    )
    origin = tuple(machine_pos)
    front_vec = rotation @ np.array([0.0, 1.0, 0.0], dtype=float)
    right_vec = rotation @ np.array([1.0, 0.0, 0.0], dtype=float)
    up_vec = np.array([0.0, 0.0, 1.0], dtype=float)

    rr.log(
        "world/robot/base",
        rr.Points3D(positions=[origin], colors=[(255, 220, 70)], radii=[0.08]),
        static=True,
    )
    rr.log(
        "world/robot/pose_axes",
        rr.Arrows3D(
            origins=[origin, origin, origin],
            vectors=[
                tuple(front_vec * 0.9),
                tuple(right_vec * 0.6),
                tuple(up_vec * 0.5),
            ],
            colors=[(255, 80, 80), (80, 220, 120), (80, 160, 255)],
            labels=["robot_front(+Y)", "robot_right(+X)", "robot_up(+Z)"],
        ),
        static=True,
    )


def log_video_asset(mp4_path):
    """Log an encoded video once and create frame-indexed references to its frames."""
    video_size = mp4_path.stat().st_size
    if video_size > MAX_ASSET_VIDEO_BYTES:
        size_gb = video_size / (1024**3)
        limit_gb = MAX_ASSET_VIDEO_BYTES / (1024**3)
        proxy_name = f"{mp4_path.stem}_rerun_proxy.mp4"
        raise RuntimeError(
            "MP4 is too large for Rerun AssetVideo in the current Python SDK path "
            f"({size_gb:.2f} GiB > ~{limit_gb:.2f} GiB).\n"
            "Create a smaller proxy MP4 and pass that to --mp4, e.g.:\n"
            f"  ffmpeg -i {mp4_path} -map 0:v:0 -vsync 0 -an -c:v libx264 -preset veryfast -crf 28 {proxy_name}"
        )

    video_asset = rr.AssetVideo(path=str(mp4_path))
    rr.log("video", video_asset, static=True)

    frame_timestamps_ns = video_asset.read_frame_timestamps_nanos()
    frame_indices = list(range(len(frame_timestamps_ns)))

    rr.send_columns(
        "video",
        indexes=[rr.TimeColumn("frame_index", sequence=frame_indices)],
        columns=rr.VideoFrameReference.columns_nanos(frame_timestamps_ns),
    )
    return frame_timestamps_ns


def build_track_updates_by_frame(track_series):
    updates_by_frame = defaultdict(list)
    for track_id, series in track_series.items():
        progressive_points = []
        for frame_index, point in series:
            progressive_points.append(point)
            updates_by_frame[frame_index].append((track_id, progressive_points.copy()))
    return updates_by_frame


def bbox_to_polyline(bbox):
    x1, y1, x2, y2 = map(float, bbox)
    return [(x1, y1), (x2, y1), (x2, y2), (x1, y2), (x1, y1)]


def log_detections_for_frame(frame_index, detections_by_frame):
    frame_payload = detections_by_frame.get(frame_index)
    if frame_payload is None:
        rr.log(
            "video/detections/left",
            rr.LineStrips2D(strips=[]),
        )
        rr.log(
            "video/detections/right",
            rr.LineStrips2D(strips=[]),
        )
        rr.log(
            "world/detection_projections/court",
            rr.Points3D(positions=[], colors=[], radii=[]),
        )
        return

    left_detections = frame_payload.get("left_detections", [])
    rr.log(
        "video/detections/left",
        rr.LineStrips2D(
            strips=[bbox_to_polyline(det["bbox"]) for det in left_detections],
            colors=[(255, 0, 255)] * len(left_detections),
        ),
    )
    right_detections = frame_payload.get("right_detections", [])
    rr.log(
        "video/detections/right",
        rr.LineStrips2D(
            strips=[bbox_to_polyline(det["bbox"]) for det in right_detections],
            colors=[(0, 255, 255)] * len(right_detections),
        ),
    )

    triangulated = frame_payload.get("triangulated_detections", [])
    rr.log(
        "world/detection_projections/court",
        rr.Points3D(
            positions=[
                (
                    float(det["projection_3d"]["court_frame_position"]["x"]),
                    float(det["projection_3d"]["court_frame_position"]["y"]),
                    float(det["projection_3d"]["court_frame_position"]["z"]),
                )
                for det in triangulated
            ],
            colors=[(255, 200, 20)] * len(triangulated),
            radii=[0.02] * len(triangulated),
        ),
    )


def track_color(track_id):
    rng = np.random.default_rng(track_id + 1337)
    return tuple(int(v) for v in rng.integers(60, 256, size=3))


def log_track_updates_for_frame(frame_index, track_updates_by_frame, track_root):
    for track_id, strip_points in track_updates_by_frame.get(frame_index, []):
        color = track_color(track_id)
        if len(strip_points) >= 2:
            rr.log(
                f"{track_root}/track_{track_id}",
                rr.LineStrips3D(
                    strips=[strip_points],
                    colors=[color],
                    radii=[0.012],
                ),
            )
        rr.log(
            f"{track_root}/track_{track_id}/points/{frame_index}",
            rr.Points3D(
                positions=[strip_points[-1]],
                colors=[color],
                radii=[0.02],
            ),
        )


def log_bounces_for_frame(frame_index, bounces_by_frame, bounce_root):
    for track_id, (x, y, _z) in bounces_by_frame.get(frame_index, []):
        color = track_color(track_id)
        rr.log(
            f"{bounce_root}/track_{track_id}/frame_{frame_index}",
            rr.LineStrips3D(
                strips=[[(x, y, 0.0), (x, y, 1.0)]],
                colors=[color],
                radii=[0.06],
            ),
        )


def run(args):
    jsonl_path = Path(args.jsonl)
    mp4_path = Path(args.mp4)
    detections_jsonl = Path(args.detections_jsonl)
    raw_trajectories_jsonl = Path(args.raw_trajectories_jsonl)
    if not jsonl_path.exists():
        raise FileNotFoundError(f"JSONL not found: {jsonl_path}")
    if not mp4_path.exists():
        raise FileNotFoundError(f"MP4 not found: {mp4_path}")
    if not detections_jsonl.exists():
        raise FileNotFoundError(f"Detections JSONL not found: {detections_jsonl}")
    if not raw_trajectories_jsonl.exists():
        raise FileNotFoundError(f"Raw trajectories JSONL not found: {raw_trajectories_jsonl}")

    track_series, bounces_by_frame, frame_processed, run_summary = load_results(jsonl_path)
    detections_by_frame, detection_totals = load_detections(detections_jsonl)
    raw_track_series, raw_totals = load_raw_trajectories(raw_trajectories_jsonl)
    raw_track_updates_by_frame = build_track_updates_by_frame(raw_track_series)

    init_rerun(args.run_name)

    if run_summary is not None and run_summary.get("machine_pose") is not None:
        log_robot_pose(run_summary["machine_pose"])

    frame_timestamps_ns = log_video_asset(mp4_path)
    frame_count = len(frame_timestamps_ns)

    max_event_frame = -1
    if frame_processed:
        max_event_frame = max(max_event_frame, max(frame_processed))
    if bounces_by_frame:
        max_event_frame = max(max_event_frame, max(bounces_by_frame))
    for series in track_series.values():
        if series:
            max_event_frame = max(max_event_frame, series[-1][0])
    for series in raw_track_series.values():
        if series:
            max_event_frame = max(max_event_frame, series[-1][0])
    if detections_by_frame:
        max_event_frame = max(max_event_frame, max(detections_by_frame))
    if max_event_frame >= frame_count:
        raise ValueError(
            f"Event stream references frame {max_event_frame}, but MP4 only has {frame_count} frames."
        )

    track_updates_by_frame = build_track_updates_by_frame(track_series)

    for frame_index in range(frame_count):
        rr.set_time("frame_index", sequence=frame_index)
        log_track_updates_for_frame(
            frame_index,
            track_updates_by_frame,
            track_root="world/tracks",
        )
        log_track_updates_for_frame(
            frame_index,
            raw_track_updates_by_frame,
            track_root="world/raw_tracks",
        )
        log_bounces_for_frame(
            frame_index,
            bounces_by_frame,
            bounce_root="world/bounces",
        )
        log_detections_for_frame(frame_index, detections_by_frame)

    track_count = len(track_series)
    raw_track_count = len(raw_track_series)
    bounce_count = sum(len(items) for items in bounces_by_frame.values())
    print(
        f"Logged video asset with {frame_count} frames, {track_count} refined tracks, "
        f"{raw_track_count} raw tracks and {bounce_count} bounces from {jsonl_path}."
    )
    if detection_totals["total_frames"]:
        print(
            f"Logged {detection_totals['total_frames']} detection frames from {detections_jsonl}."
        )
    if raw_totals["total_tracks"]:
        print(
            f"Logged {raw_totals['total_tracks']} raw trajectories from {raw_trajectories_jsonl}."
        )
    if run_summary is not None:
        total_frames_processed = run_summary.get("total_frames_processed")
        if total_frames_processed is not None:
            print(f"Run summary frames processed: {total_frames_processed}")
        machine_pose = run_summary.get("machine_pose")
        if machine_pose is not None:
            print(
                f"Machine pose: ({float(machine_pose['x']):.3f}, {float(machine_pose['y']):.3f}, "
                f"{float(machine_pose.get('z', 0.0)):.3f}, yaw={float(machine_pose['yaw']):.1f}deg)"
            )
        camera_pose = run_summary.get("camera_pose")
        if camera_pose is not None:
            print(
                f"Camera pose ({args.camera_name}): x={float(camera_pose['x']):.3f}, "
                f"y={float(camera_pose['y']):.3f}, z={float(camera_pose['z']):.3f}, "
                f"yaw={np.degrees(float(camera_pose['yaw_rad'])):.1f}deg"
            )


def main():
    parser = argparse.ArgumentParser(
        description="Rerun visualization for online_runner.py JSONL results synchronized with MP4 frames."
    )
    parser.add_argument(
        "--jsonl",
        type=str,
        default="results.jsonl",
        help="Path to JSONL generated by online_runner.py.",
    )
    parser.add_argument(
        "--detections-jsonl",
        type=str,
        default="detections.jsonl",
        help="Path to per-frame detections JSONL generated by online_runner.py.",
    )
    parser.add_argument(
        "--raw-trajectories-jsonl",
        type=str,
        default="trajectories_raw.jsonl",
        help="Path to raw track trajectories JSONL generated by online_runner.py.",
    )
    parser.add_argument(
        "--mp4",
        type=str,
        required=True,
        help="Path to MP4 logged in Rerun.",
    )
    parser.add_argument(
        "--run-name",
        type=str,
        default="balltracking_online_results",
        help="Rerun run name.",
    )
    parser.add_argument(
        "--camera-name",
        type=str,
        default="front",
        choices=["front", "back", "left", "right"],
        help="Camera label used only for printed run-summary pose metadata.",
    )
    args = parser.parse_args()
    run(args)


if __name__ == "__main__":
    main()
