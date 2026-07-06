#!/usr/bin/env python3
"""Run the ball tracking stack in online mode over a ZED SVO stream."""

from __future__ import annotations

import argparse
import json
import threading
import time
from dataclasses import dataclass, field
from queue import Queue

import cv2
import numpy as np
import pyzed.sl as sl
from tqdm import tqdm
from ultralytics import YOLO

from pipeline_common import (
    build_detection_records,
    get_camera_extrinsics,
    infer_machine_pose_from_rgb_frame,
    match_stereo_detections,
    transform_camera_to_court,
    triangulate_stereo,
)
from profiler import StageProfiler
from refiner import refine_trajectory
from tracker import BallDetection3D, MultiSensorMultiTargetTracker

DEFAULT_EXTRINSICS_PRESET = "camera_stand"
DEFAULT_IMGSZ = 1280

_SENTINEL_EOF = "EOF"
_SENTINEL_ERROR = "ERROR"


class FramePrefetcher:
    """Background thread that grabs ZED frames and converts BGRA->BGR ahead of the main loop.

    This overlaps frame acquisition and color conversion with GPU inference,
    ensuring the GPU never stalls waiting for the next frame.
    """

    def __init__(self, zed, runtime_params, *, max_frames: int | None = None, queue_size: int = 2):
        self._zed = zed
        self._runtime_params = runtime_params
        self._max_frames = max_frames
        self._queue: Queue = Queue(maxsize=queue_size)
        self._thread = threading.Thread(target=self._run, daemon=True)

    def start(self):
        self._thread.start()

    def _run(self):
        left_mat = sl.Mat()
        right_mat = sl.Mat()
        frame_count = 0

        while True:
            if self._max_frames is not None and frame_count >= self._max_frames:
                self._queue.put((_SENTINEL_EOF, None, None, None))
                return

            err = self._zed.grab(self._runtime_params)
            if err == sl.ERROR_CODE.END_OF_SVOFILE_REACHED:
                self._queue.put((_SENTINEL_EOF, None, None, None))
                return
            if err != sl.ERROR_CODE.SUCCESS:
                self._queue.put((_SENTINEL_ERROR, err, None, None))
                return

            timestamp_ns = int(
                self._zed.get_timestamp(sl.TIME_REFERENCE.IMAGE).get_nanoseconds()
            )
            self._zed.retrieve_image(left_mat, sl.VIEW.LEFT)
            self._zed.retrieve_image(right_mat, sl.VIEW.RIGHT)

            # Slice off alpha channel as a zero-copy view instead of cvtColor.
            # The underlying sl.Mat buffer is overwritten on the next grab, so
            # we must copy here anyway — .copy() on the 3-channel view is still
            # cheaper than a full BGRA→BGR cvtColor.
            left_bgr = left_mat.get_data()[:, :, :3].copy()
            right_bgr = right_mat.get_data()[:, :, :3].copy()

            self._queue.put((timestamp_ns, left_bgr, right_bgr, None))
            frame_count += 1

    def get_frame(self):
        """Return (timestamp_ns, left_bgr, right_bgr) or None on EOF.

        Raises RuntimeError on grab errors.
        """
        item = self._queue.get()
        tag = item[0]
        if tag == _SENTINEL_EOF:
            return None
        if tag == _SENTINEL_ERROR:
            raise RuntimeError(f"Grab error: {item[1]}")
        return item[0], item[1], item[2]

    def drain_batch(
        self, max_frames: int, fill_timeout: float = 0.005,
    ) -> list[tuple[int, np.ndarray, np.ndarray]]:
        """Block for at least one frame, then try to fill up to *max_frames*.

        After the first frame, each subsequent frame is given up to
        *fill_timeout* seconds to arrive before we give up and return a
        partial batch.  This keeps batches full in steady state (the
        prefetcher is typically faster than inference) without adding
        meaningful latency when frames are slow to arrive.

        Returns a list of (timestamp_ns, left_bgr, right_bgr) tuples.
        An empty list signals EOF.  Raises RuntimeError on grab errors.
        """
        first = self.get_frame()
        if first is None:
            return []

        batch = [first]
        while len(batch) < max_frames:
            try:
                item = self._queue.get(timeout=fill_timeout)
            except Exception:
                # queue.Empty — timed out waiting for the next frame
                break
            tag = item[0]
            if tag == _SENTINEL_EOF:
                # Put sentinel back so the next drain_batch also sees EOF.
                self._queue.put(item)
                break
            if tag == _SENTINEL_ERROR:
                raise RuntimeError(f"Grab error: {item[1]}")
            batch.append((item[0], item[1], item[2]))
        return batch

    def join(self):
        self._thread.join()


@dataclass
class TrackEmitState:
    committed_count: int = 0
    emitted_bounce_frames: set[int] = field(default_factory=set)
    ended: bool = False


class JsonlEventWriter:
    def __init__(self, output_path: str):
        self.output_path = output_path
        self.stream = open(output_path, "w", encoding="utf-8")
        self._should_close = True

    def emit(self, event_type: str, **payload):
        record = {"type": event_type, **payload}
        self.stream.write(json.dumps(record, default=self._json_default) + "\n")
        self.stream.flush()

    def close(self):
        if self._should_close:
            self.stream.close()

    @staticmethod
    def _json_default(value):
        if isinstance(value, np.integer):
            return int(value)
        if isinstance(value, np.floating):
            return float(value)
        if isinstance(value, np.ndarray):
            return value.tolist()
        raise TypeError(f"Object of type {type(value).__name__} is not JSON serializable")


class OnlineRefinementEmitter:
    def __init__(self, lookahead_frames: int):
        self.lookahead_frames = lookahead_frames
        self.track_states: dict[int, TrackEmitState] = {}

    def emit_updates_for_track(
        self,
        track_id: int,
        track,
        frame_timestamps_ns: dict[int, int],
        writer: JsonlEventWriter,
        *,
        force: bool = False,
    ):
        points_with_frame = self._extract_track_series(track, frame_timestamps_ns)
        if not points_with_frame:
            return

        state = self.track_states.setdefault(track_id, TrackEmitState())
        frames = np.asarray([frame for frame, _ts, _point in points_with_frame], dtype=np.int64)
        timestamps_ns = np.asarray([ts for _frame, ts, _point in points_with_frame], dtype=np.int64)
        positions = np.asarray([point for _frame, _ts, point in points_with_frame], dtype=float)

        # The newest observations stay provisional until enough future context
        # arrives to make bounce timing and piecewise fits line up with the
        # offline refiner's decisions.
        commit_count = len(points_with_frame) if force else max(0, len(points_with_frame) - self.lookahead_frames)
        if commit_count <= 0:
            return

        # Re-running the batch refiner over the buffered prefix keeps the online
        # output behavior aligned with the proven offline stack before we try to
        # optimize this into a more incremental form.
        refined = refine_trajectory(track_id=track_id, positions=positions, timestamps_ns=timestamps_ns)

        if commit_count > state.committed_count:
            new_frames = frames[state.committed_count:commit_count]
            new_timestamps = timestamps_ns[state.committed_count:commit_count]
            new_points = refined.refined_positions[state.committed_count:commit_count]
            writer.emit(
                "trajectory_segment_finalized",
                track_id=track_id,
                start_frame=int(new_frames[0]),
                end_frame=int(new_frames[-1]),
                source="refined",
                points=[
                    {
                        "frame_index": int(frame_index),
                        "timestamp_ns": int(timestamp_ns),
                        "x": float(point[0]),
                        "y": float(point[1]),
                        "z": float(point[2]),
                    }
                    for frame_index, timestamp_ns, point in zip(new_frames, new_timestamps, new_points)
                ],
            )
            state.committed_count = commit_count

        for bounce in refined.bounces:
            if bounce.index >= commit_count:
                continue
            bounce_frame = int(frames[bounce.index])
            if bounce_frame in state.emitted_bounce_frames:
                continue
            state.emitted_bounce_frames.add(bounce_frame)
            writer.emit(
                "bounce_finalized",
                track_id=track_id,
                frame_index=bounce_frame,
                timestamp_ns=int(timestamps_ns[bounce.index]),
                x=float(bounce.position[0]),
                y=float(bounce.position[1]),
                z=float(bounce.position[2]),
            )

    def mark_ended(self, track_id: int):
        state = self.track_states.setdefault(track_id, TrackEmitState())
        state.ended = True

    def is_ended(self, track_id: int) -> bool:
        return self.track_states.get(track_id, TrackEmitState()).ended

    @staticmethod
    def _extract_track_series(track, frame_timestamps_ns: dict[int, int]):
        points_with_frame = []
        for frame_index, camera_map in sorted(track.detections.items()):
            if not camera_map or frame_index not in frame_timestamps_ns:
                continue
            dets = list(camera_map.values())
            # We collapse same-frame multi-camera measurements into one court-space
            # sample because the refiner expects a single position per timestep.
            point = np.array(
                [
                    np.mean([d.x for d in dets]),
                    np.mean([d.y for d in dets]),
                    np.mean([d.z for d in dets]),
                ],
                dtype=float,
            )
            points_with_frame.append((int(frame_index), int(frame_timestamps_ns[frame_index]), point))
        return points_with_frame


def run_model_batch(model, frames_bgr: list, imgsz: int, conf_thres: float, half: bool = False):
    results = model.predict(
        frames_bgr,
        imgsz=imgsz,
        conf=conf_thres,
        half=half,
        verbose=False,
        save=False,
    )
    return [r.cpu().numpy().boxes for r in results]


def emit_stale_tracks(tracker, emitter, frame_timestamps_ns, writer):
    for track_id, track in tracker.tracks.items():
        if emitter.is_ended(track_id) or track.last_seen_in_frame is None:
            continue
        if (tracker.current_frame - track.last_seen_in_frame) < tracker.frames_before_track_stale:
            continue

        emitter.emit_updates_for_track(track_id, track, frame_timestamps_ns, writer, force=True)
        writer.emit(
            "track_ended",
            track_id=track_id,
            final_frame=int(track.last_seen_in_frame),
            reason="stale",
        )
        emitter.mark_ended(track_id)


def emit_eof_tracks(tracker, emitter, frame_timestamps_ns, writer):
    for track_id, track in tracker.tracks.items():
        if emitter.is_ended(track_id) or track.last_seen_in_frame is None:
            continue
        emitter.emit_updates_for_track(track_id, track, frame_timestamps_ns, writer, force=True)
        writer.emit(
            "track_ended",
            track_id=track_id,
            final_frame=int(track.last_seen_in_frame),
            reason="eof",
        )
        emitter.mark_ended(track_id)


def collect_frame_detections_payload(
    frame_index: int,
    timestamp_ns: int,
    left_detections,
    right_detections,
    stereo_matches,
    triangulated_detections,
):
    def _projected_to_payload(item):
        if item is None:
            return None

        camera_frame_position = item["camera_position"]
        camera_projection = {
            "camera_frame_position": {
                "x": float(camera_frame_position[0]),
                "y": float(camera_frame_position[1]),
                "z": float(camera_frame_position[2]),
            },
            "camera_depth": float(item["camera_depth"]),
            "disparity": float(item["disparity"]),
        }

        court_position = item["court_position"]
        camera_projection["court_frame_position"] = {
            "x": float(court_position[0]),
            "y": float(court_position[1]),
            "z": float(court_position[2]),
        }

        return camera_projection

    return {
        "frame_index": int(frame_index),
        "timestamp_ns": int(timestamp_ns),
        "left_detections": [
            {
                "index": int(det["index"]),
                "confidence": float(det["confidence"]),
                "bbox": [float(v) for v in det["bbox"]],
            }
            for det in left_detections
        ],
        "right_detections": [
            {
                "index": int(det["index"]),
                "confidence": float(det["confidence"]),
                "bbox": [float(v) for v in det["bbox"]],
            }
            for det in right_detections
        ],
        "triangulated_detections": [
            {
                "left_index": int(item["left_detection"]["index"]),
                "right_index": int(item["right_detection"]["index"]),
                "projection_3d": _projected_to_payload(item),
            }
            for item in triangulated_detections
        ],
        "stereo_matches": [
            {"left_index": int(left_det["index"]), "right_index": int(right_det["index"])}
            for left_det, right_det in stereo_matches
        ],
    }


def collect_raw_trajectories(tracker, frame_timestamps_ns: dict[int, int]):
    trajectories = []
    for track_id, track in tracker.tracks.items():
        points = []
        for state_frame, state, _covariance in track.state_history:
            frame_ts = frame_timestamps_ns.get(state_frame)
            if frame_ts is None:
                continue
            points.append(
                {
                    "frame_index": int(state_frame),
                    "timestamp_ns": int(frame_ts),
                    "x": float(state[0]),
                    "y": float(state[1]),
                    "z": float(state[2]),
                }
            )

        trajectories.append(
            {
                "track_id": int(track_id),
                "points": points,
            }
        )

    return trajectories


def run(args):
    event_writer = JsonlEventWriter(args.event_output)
    detection_writer = JsonlEventWriter(args.detections_output)
    raw_trajectory_writer = JsonlEventWriter(args.raw_trajectory_output)
    zed = None
    progress = None
    prefetcher = None
    profiler = StageProfiler(enabled=args.profile)
    try:
        model = YOLO(
            args.weights, 
            #  specify task type explicitly because can't auto-infer from tensorrt engine
            task='detect'
        )
        extrinsics = get_camera_extrinsics(args.extrinsics_preset)
        if args.camera_name not in extrinsics:
            raise ValueError(
                f"Camera '{args.camera_name}' not found in preset '{args.extrinsics_preset}'."
            )

        zed = sl.Camera()
        init_params = sl.InitParameters()
        init_params.set_from_svo_file(args.svo)
        init_params.coordinate_units = sl.UNIT.METER
        # We do our own stereo triangulation from left/right YOLO detections,
        # so disable the SDK's internal depth engine to save GPU.
        init_params.depth_mode = sl.DEPTH_MODE.NONE
        # Even in "online" mode we keep SVO playback non-real-time so the
        # pipeline sees every frame and emits deterministic results.
        init_params.svo_real_time_mode = False
        init_params.enable_image_validity_check = 1

        status = zed.open(init_params)
        if status != sl.ERROR_CODE.SUCCESS:
            raise RuntimeError(f"Failed to open SVO: {status}")

        cam_info = zed.get_camera_information()
        cam_res = cam_info.camera_configuration.resolution
        total_frames = zed.get_svo_number_of_frames()
        calib = cam_info.camera_configuration.calibration_parameters
        stereo_calib = {
            "fx": float(calib.left_cam.fx),
            "fy": float(calib.left_cam.fy),
            "cx": float(calib.left_cam.cx),
            "cy": float(calib.left_cam.cy),
            "baseline": float(calib.get_camera_baseline()),
        }
        camera_intrinsics = {
            args.camera_name: {
                "mat": np.array(
                    [
                        [stereo_calib["fx"], 0.0, stereo_calib["cx"]],
                        [0.0, stereo_calib["fy"], stereo_calib["cy"]],
                        [0.0, 0.0, 1.0],
                    ],
                    dtype=np.float32,
                ),
                "dist": np.zeros((4,), dtype=np.float32),
                "fisheye": False,
            }
        }

        runtime_params = sl.RuntimeParameters()

        tracker = MultiSensorMultiTargetTracker(fps=cam_info.camera_configuration.fps)
        emitter = OnlineRefinementEmitter(lookahead_frames=args.bounce_lookahead_frames)
        frame_timestamps_ns: dict[int, int] = {}
        frame_detections: list[dict] = []

        machine_pose = None
        inferred_localizer = None

        print(
            f"Processing SVO {args.svo} ({cam_res.width}x{cam_res.height}, {total_frames} frames)",
        )

        progress_total = total_frames
        if args.max_frames is not None:
            progress_total = min(progress_total, args.max_frames)
        progress = tqdm(total=progress_total, unit="frame", dynamic_ncols=True)

        camera_fps = cam_info.camera_configuration.fps
        # Allow up to 500ms of prefetched frames so the GPU never stalls
        # waiting for frame data, even if individual grabs are bursty.
        prefetch_queue_size = max(2, int(camera_fps * 0.5))
        prefetcher = FramePrefetcher(
            zed, runtime_params, max_frames=args.max_frames, queue_size=prefetch_queue_size,
        )
        prefetcher.start()


        max_batch = args.inference_batch_size

        while True:
            with profiler.measure("wait_for_batch"):
                batch = prefetcher.drain_batch(max_batch)
            if not batch:
                break

            # Localize from the first frame we ever see (before any batched
            # inference) because court transforms depend on machine_pose.
            if machine_pose is None:
                with profiler.measure("initial_localization"):
                    frame_left_rgb = cv2.cvtColor(batch[0][1], cv2.COLOR_BGR2RGB)
                    machine_pose, inferred_localizer = infer_machine_pose_from_rgb_frame(
                        frame_rgb=frame_left_rgb,
                        camera_name=args.camera_name,
                        extrinsics=extrinsics,
                        camera_intrinsics=camera_intrinsics,
                    )
                print(
                    "Inferred machine pose from first frame: "
                    f"x={machine_pose['x']:.3f}, y={machine_pose['y']:.3f}, "
                    f"z={machine_pose['z']:.3f}, yaw={machine_pose['yaw']:.1f}",
                )

            # Flatten all left/right frames into one big YOLO batch:
            # [left_0, right_0, left_1, right_1, ...]
            all_images = []
            for _ts, left_bgr, right_bgr in batch:
                all_images.append(left_bgr)
                all_images.append(right_bgr)

            # Pad to the fixed TensorRT batch size (2 images per frame-pair).
            # Padding frames use a zero image at the same resolution so the
            # engine always sees its compiled batch dimension.
            real_count = len(all_images)
            fixed_batch = 2 * max_batch
            if real_count < fixed_batch:
                pad_image = np.zeros_like(all_images[0])
                all_images.extend([pad_image] * (fixed_batch - real_count))

            with profiler.measure("batched_detection"):
                all_boxes = run_model_batch(
                    model,
                    all_images,
                    imgsz=args.imgsz,
                    conf_thres=args.conf_thres,
                    half=not args.fp32,
                )
                # Discard results from padding frames.
                all_boxes = all_boxes[:real_count]

            # Post-process each frame pair sequentially (tracker is stateful
            # and must see frames in order).
            for i, (timestamp_ns, _left_bgr, _right_bgr) in enumerate(batch):
                frame_index = tracker.current_frame
                left_det_boxes = all_boxes[2 * i]
                right_det_boxes = all_boxes[2 * i + 1]

                with profiler.measure("frame_total"):
                    frame_timestamps_ns[frame_index] = timestamp_ns

                    left_detections = build_detection_records(left_det_boxes)
                    right_detections = build_detection_records(right_det_boxes)

                    with profiler.measure("stereo_match_triangulate"):
                        stereo_matches = match_stereo_detections(
                            left_detections,
                            right_detections,
                            y_tolerance=args.stereo_y_tolerance,
                            area_tolerance=args.stereo_area_tolerance,
                        )
                        left_to_right_match = {
                            left_det["index"]: right_det for left_det, right_det in stereo_matches
                        }

                        triangulated = []
                        for det in left_detections:
                            matched_right_det = left_to_right_match.get(det["index"])
                            if matched_right_det is None:
                                continue

                            cx, cy = det["center"]
                            right_cx, _right_cy = matched_right_det["center"]
                            stereo_pos, disparity = triangulate_stereo(cx, cy, right_cx, stereo_calib)
                            if stereo_pos is None:
                                continue

                            triangulated.append(
                                {
                                    "left_detection": det,
                                    "right_detection": matched_right_det,
                                    "camera_position": stereo_pos,
                                    "camera_depth": float(np.linalg.norm(stereo_pos)),
                                    "disparity": float(disparity),
                                    "court_position": transform_camera_to_court(
                                        stereo_pos,
                                        args.camera_name,
                                        machine_pose,
                                        extrinsics,
                                    ),
                                }
                            )

                        frame_detections.append(
                            collect_frame_detections_payload(
                                frame_index=frame_index,
                                timestamp_ns=timestamp_ns,
                                left_detections=left_detections,
                                right_detections=right_detections,
                                stereo_matches=stereo_matches,
                                triangulated_detections=triangulated,
                            )
                        )


                    with profiler.measure("tracker_process_frame"):
                        # map to the tracker's input data structure
                        world_detections = []
                        for item in triangulated:
                            court_position = item["court_position"]
                            world_detections.append(
                                BallDetection3D(
                                    camera_id=f"{args.camera_name}_stereo",
                                    x=float(court_position[0]),
                                    y=float(court_position[1]),
                                    z=float(court_position[2]),
                                    depth=float(item["camera_depth"]),
                                )
                            )
                        tracker.process_frame(world_detections)

                    with profiler.measure("track_emission"):
                        for track_id, track in tracker.tracks.items():
                            if emitter.is_ended(track_id):
                                continue
                            emitter.emit_updates_for_track(track_id, track, frame_timestamps_ns, event_writer)

                    with profiler.measure("stale_track_emission"):
                        emit_stale_tracks(tracker, emitter, frame_timestamps_ns, event_writer)

                    with profiler.measure("event_write_frame_processed"):
                        event_writer.emit(
                            "frame_processed",
                            frame_index=frame_index,
                            timestamp_ns=timestamp_ns,
                            left_detection_count=len(left_detections),
                            right_detection_count=len(right_detections),
                            triangulated_detection_count=len(triangulated),
                            active_track_count=sum(
                                0 if emitter.is_ended(track_id) else 1 for track_id in tracker.tracks
                            ),
                        )

                    progress.update(1)
                    progress.set_postfix(batch=f"{len(batch)}/{max_batch}", refresh=False)

        emit_eof_tracks(tracker, emitter, frame_timestamps_ns, event_writer)
        detection_writer.emit(
            "detections",
            total_frames=len(frame_detections),
            frames=frame_detections,
        )
        raw_trajectories = collect_raw_trajectories(tracker, frame_timestamps_ns)
        raw_trajectory_writer.emit(
            "trajectories_raw",
            total_tracks=len(raw_trajectories),
            trajectories=raw_trajectories,
        )

        if inferred_localizer is not None and inferred_localizer.pose is not None:
            event_writer.emit(
                "run_summary",
                total_frames_processed=len(frame_timestamps_ns),
                machine_pose=machine_pose,
                camera_pose={
                    "x": float(inferred_localizer.pose.x),
                    "y": float(inferred_localizer.pose.y),
                    "z": float(inferred_localizer.pose.altitude),
                    "yaw_rad": float(inferred_localizer.pose.yaw),
                },
            )
        else:
            event_writer.emit(
                "run_summary",
                total_frames_processed=len(frame_timestamps_ns),
                machine_pose=machine_pose,
            )
    finally:
        if progress is not None:
            progress.close()
        # Drain the prefetcher so its thread exits cleanly before we close the
        # ZED camera handle it references.
        if prefetcher is not None:
            prefetcher.join()
        if zed is not None:
            zed.close()
        event_writer.close()
        detection_writer.close()
        raw_trajectory_writer.close()
        profiler.print_summary()


def main():
    parser = argparse.ArgumentParser(description="Run online ball tracking over a ZED SVO input.")
    parser.add_argument("--svo", type=str, required=True, help="Path to the input SVO file.")
    parser.add_argument("--weights", type=str, default="yolo_ball_detector.pt", help="YOLO weights path.")
    parser.add_argument(
        "--camera-name",
        type=str,
        default="front",
        choices=["front", "back", "left", "right"],
        help="Camera label used for pose inference and court transforms.",
    )
    parser.add_argument(
        "--event-output",
        type=str,
        default="results.jsonl",
        help="Path for JSONL event output.",
    )
    parser.add_argument(
        "--detections-output",
        type=str,
        default="detections.jsonl",
        help="Path for JSONL per-frame 2D detections output.",
    )
    parser.add_argument(
        "--raw-trajectory-output",
        type=str,
        default="trajectories_raw.jsonl",
        help="Path for JSONL raw trajectory output.",
    )
    parser.add_argument(
        "--extrinsics-preset",
        type=str,
        default=DEFAULT_EXTRINSICS_PRESET,
        help="Named camera extrinsics preset.",
    )
    parser.add_argument(
        "--bounce-lookahead-frames",
        type=int,
        default=10,
        help="Frames of lookahead before bounce/trajectory outputs are considered final.",
    )
    parser.add_argument("--imgsz", type=int, default=DEFAULT_IMGSZ, help="YOLO inference image size.")
    parser.add_argument("--conf-thres", type=float, default=0.3, help="YOLO confidence threshold.")
    parser.add_argument(
        "--stereo-y-tolerance",
        type=float,
        default=5.0,
        help="Maximum pixel row mismatch for stereo pairing.",
    )
    parser.add_argument(
        "--stereo-area-tolerance",
        type=float,
        default=0.3,
        help="Maximum relative area mismatch tolerance for stereo pairing.",
    )
    parser.add_argument(
        "--max-frames",
        type=int,
        default=None,
        help="Optional frame cap for debugging shorter runs.",
    )
    parser.add_argument(
        "--fp32",
        action="store_true",
        help="Use full FP32 precision inference instead of the default FP16 half-precision.",
    )
    parser.add_argument(
        "--inference-batch-size",
        type=int,
        default=4,
        help="Max stereo frame-pairs to batch into a single YOLO call (2N images).",
    )
    parser.add_argument(
        "--profile",
        action="store_true",
        help="Print an end-of-run wall-clock stage timing summary.",
    )
    args = parser.parse_args()

    if args.bounce_lookahead_frames < 1:
        raise ValueError("--bounce-lookahead-frames must be >= 1")

    run(args)


if __name__ == "__main__":
    main()
