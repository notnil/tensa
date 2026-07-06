import numpy as np
from scipy.optimize import linear_sum_assignment
from collections import defaultdict
from dataclasses import dataclass

@dataclass
class BallDetection3D:
    camera_id: str
    x: float
    y: float
    z: float
    depth: float


class TennisBallKalmanFilter:
    """
    Maintains a 6D state: [x, y, z, vx, vy, vz]
    and updates with 3D position measurements: [x, y, z].
    Position coordinates represent meters in 'court space'.
    """

    def __init__(self, fps):
        self.fps = fps
        dt = 1.0 / fps

        # Build F for constant velocity in x,y and basic velocity in z
        # We'll inject gravity separately via a B matrix
        self.F = np.eye(6)
        self.F[0, 3] = dt  # x depends on vx
        self.F[1, 4] = dt  # y depends on vy
        self.F[2, 5] = dt  # z depends on vz

        # A simple "control" vector for gravity in z
        self.B = np.array([0, 0, -0.5 * 9.8 * dt**2, 0, 0, -9.8 * dt], dtype=float)

        # Measurement matrix H picks out x, y, z
        self.H = np.array(
            [
                [1, 0, 0, 0, 0, 0],
                [0, 1, 0, 0, 0, 0],
                [0, 0, 1, 0, 0, 0],
            ],
            dtype=float,
        )

        # Process noise covariance Q
        # - Higher noise for z and vz due to effect of gravity, bounces
        # - Lower noise for vx, vy because these stay relatively constant
        self.Q = np.diag([0.01, 0.01, 0.4, 0.001, 0.001, 0.1])

        # State estimate (x), initialized to zeros
        self.x = np.zeros(6, dtype=float)

        # Covariance (P): use large initial uncertainty
        self.P = np.eye(6, dtype=float) * 1000.0

        # Coefficient of restitution for bounce
        self.alpha = 0.75

    def predict(self):
        """Predict next state and covariance."""
        self.x = self.F @ self.x + self.B  # Incorporate gravity
        self.P = self.F @ self.P @ self.F.T + self.Q

        # Bounce logic: if we predict the ball going below the ground plane
        # TODO: revisit margin of error
        if self.x[2] < 0.1:
            self.x[2] = -self.x[2]
            self.x[5] = -self.alpha * self.x[5]

    def update(self, z: np.ndarray, depth: float):
        """Update the state using measurement z = [x, y, z]."""
        # Residual
        y = z - (self.H @ self.x)
        # Scale measurement noise based on ball depth (distance from camera)
        R = (np.eye(3) * 2) + (0.2 * depth)
        # Residual covariance
        S = self.H @ self.P @ self.H.T + R
        # Kalman gain
        K = self.P @ self.H.T @ np.linalg.inv(S)
        # State update
        self.x += K @ y
        # Covariance update
        I = np.eye(6)
        self.P = (I - K @ self.H) @ self.P

    def get_state(self) -> np.ndarray:
        """Returns the current estimate: [x, y, z, vx, vy, vz]."""
        return self.x

    def set_state(self, x_init: np.ndarray):
        """Initialize the filter state vector if you have a known starting position."""
        self.x = x_init

    def copy(self):
        kf_copy = TennisBallKalmanFilter(self.fps)
        kf_copy.F = self.F.copy()
        kf_copy.B = self.B.copy()
        kf_copy.H = self.H.copy()
        kf_copy.Q = self.Q.copy()
        kf_copy.x = self.x.copy()
        kf_copy.P = self.P.copy()
        return kf_copy


class BallTrack:
    """
    A single track representing one flying tennis ball, maintained by a 3D Kalman filter.
    """

    def __init__(self, fps):
        # The frame that the track was last seen in.
        self.last_seen_in_frame: int | None = None

        # frame -> {camera_id -> BallDetection3D}
        self.detections: dict[int, dict[str, BallDetection3D]] = {}

        # A dedicated Kalman filter for this track. Will be initiated with first detections.
        self.kf = None

        # Frames per second for time component of process model.
        self.fps = fps

        self.state_history = []

    def _init_kalman(self, detections: list[BallDetection3D]):
        """Initialize the Kalman filter with the means of the first detections."""
        self.kf = TennisBallKalmanFilter(self.fps)
        x_init = np.array(
            [
                np.mean([d.x for d in detections]),
                np.mean([d.y for d in detections]),
                np.mean([d.z for d in detections]),
                0,
                0,
                0,
            ],
            dtype=float,
        )
        self.kf.set_state(x_init)

    def _update_kalman_filter(self, detections: list[BallDetection3D]):
        """
        Predict, then correct with the new measurement [x, y, z].
        Adjust the measurement noise or skip that, depending on your needs.
        """
        if self.kf is None:
            self._init_kalman(detections)

        # Predict forward one step
        self.kf.predict()

        # Update filter with measurements
        for detection in detections:
            z = np.array([detection.x, detection.y, detection.z], dtype=float)
            self.kf.update(z, detection.depth)

    def step(self, frame_id: int, detections: list[BallDetection3D]):
        """Process a timestep. May contain zero or more detections. Should be called on every frame."""
        if self.last_seen_in_frame is not None and self.last_seen_in_frame > frame_id:
            raise ValueError(f"frame {frame_id} is out-of-order")

        if detections:
            self.last_seen_in_frame = frame_id
            if frame_id not in self.detections:
                self.detections[frame_id] = {}
            for detection in detections:
                self.detections[frame_id][detection.camera_id] = detection

        # Update the Kalman filter with the new detections.
        self._update_kalman_filter(detections)

        # Store the state in history.
        self.state_history.append((frame_id, self.kf.x.copy(), self.kf.P.copy()))

    def get_last_detections(self) -> tuple[dict[str, BallDetection3D], int]:
        """Return ({camera_id->detection}, frame_id). Raises if none exist."""
        if self.last_seen_in_frame is None:
            raise Exception("No detections in track yet")
        det_map = self.detections.get(self.last_seen_in_frame, None)
        if det_map is None:
            raise Exception(f"No detections in last frame = {self.last_seen_in_frame}")
        return (det_map, self.last_seen_in_frame)

    def get_detections_in_frame(self, frame_id: int) -> list[BallDetection3D]:
        det_map = self.detections.get(frame_id, {})
        return list(det_map.values())

    def is_visible(self, frame_id: int) -> bool:
        return frame_id in self.detections

    def is_incoming(self) -> bool:
        """
        Determine if the ball is flying towards the near court by checking
        if the average velocity in the y-direction is positive.
        """
        avg_vy = np.mean([vy for _, (_, _, _, _, vy, _), _ in self.state_history])
        return avg_vy > 0

    def predict_state(self, current_frame: int) -> np.ndarray | None:
        """
        Returns the predicted 3D position for 'current_frame'
        by temporarily stepping the filter forward if needed.
        """
        if self.last_seen_in_frame is None or self.kf is None:
            return None

        frame_gap = current_frame - self.last_seen_in_frame
        if frame_gap < 0:
            # Shouldn't normally happen if frames are in order
            raise Exception("Internal error: frames not in order")

        # Make a copy for a "virtual" predict
        # TODO: slower than it needs to be because of copying buffers
        kf_copy = self.kf.copy()
        for _ in range(frame_gap):
            kf_copy.predict()
        pred_state = kf_copy.get_state()  # [x, y, z, vx, vy, vz]
        return pred_state


class MultiSensorMultiTargetTracker:
    """
    Manages multiple BallTrack objects, each governed by a 3D Kalman filter.
    """

    def __init__(self, fps: int):
        self.max_assignment_cost = 6
        self.frames_before_track_stale = int(fps / 4)
        self.fps = fps

        # Initiate state
        self.refresh()

    def refresh(self):
        self.current_frame = 0
        self.track_id_source = 0
        self.tracks: dict[int, BallTrack] = {}

    def register_new_track(self, det: BallDetection3D):
        bt = BallTrack(self.fps)
        bt.step(self.current_frame, [det])
        self.tracks[self.track_id_source] = bt
        self.track_id_source += 1

    def group_detections_by_camera(
        self, detections: list[BallDetection3D]
    ) -> dict[str, list[BallDetection3D]]:
        by_camera = defaultdict(list)
        for d in detections:
            by_camera[d.camera_id].append(d)
        return by_camera

    def assign_detections_to_tracks(self, detections: list[BallDetection3D]):
        # Filter out stale tracks
        active_tracks: list[BallTrack] = [
            t
            for t in self.tracks.values()
            if (self.current_frame - t.last_seen_in_frame)
            < self.frames_before_track_stale
        ]

        # Group by camera
        dets_by_camera = self.group_detections_by_camera(detections)

        matched_dets_per_track: dict[int, list[BallDetection3D]] = defaultdict(list)
        for camera_detections in dets_by_camera.values():
            if not active_tracks:
                # If no tracks exist yet or if all tracks are stale, each detection spawns a new track for each detection of this camera.
                # Detections from other cameras may get assigned to these new tracks.
                for d in camera_detections:
                    # TODO: this is naive; initial detections of the same ball may come from different cameras and will be close together in space.
                    self.register_new_track(d)
                continue

            cost = np.zeros(
                (len(active_tracks), len(camera_detections)), dtype=np.float32
            )

            for i, track in enumerate(active_tracks):
                predicted_state = track.predict_state(self.current_frame)
                predicted_pos = predicted_state[:3]  # We only need position
                if predicted_pos is None:
                    # Fallback if we can’t predict
                    det_map, last_frame = track.get_last_detections()
                    ref_det = next(iter(det_map.values()))
                    predicted_pos = np.array([ref_det.x, ref_det.y, ref_det.z])

                for j, d in enumerate(camera_detections):
                    distance = np.linalg.norm(np.array([d.x, d.y, d.z]) - predicted_pos)
                    cost[i][j] = distance

            # Hungarian assignment to match detections to tracks
            row_ind, col_ind = linear_sum_assignment(cost)
            matched_detection_indices = set()
            for track_idx, det_idx in zip(row_ind, col_ind):
                if cost[track_idx][det_idx] <= self.max_assignment_cost:
                    matched_detection_indices.add(det_idx)
                    matched_dets_per_track[track_idx].append(camera_detections[det_idx])

            # Create new track for unassigned detections
            # TODO: revisit track initialization
            unmatched = set(range(len(camera_detections))) - matched_detection_indices
            for det_idx in unmatched:
                self.register_new_track(camera_detections[det_idx])

        # Assign the matched detections to their tracks.
        # Making sure _all active_ tracks are incremented with one step, regardless of whether any detections are assigned.
        for i, track in enumerate(active_tracks):
            # If no detections were matched to this track, we update with an empty list.
            dets_for_track = matched_dets_per_track.get(i, [])
            track.step(self.current_frame, dets_for_track)

    def filter_detections(self, detections: list[BallDetection3D]):
        # Filter out any obvious outliers.
        # TODO: this filter is relatively liberal, especially on the z-axis. We can make it more strict once we have better sensors.
        filtered = []
        for d in detections:
            if (
                d.depth < 11
                and d.x < 10
                and d.x > -10
                and d.y < 15
                and d.y > -15
                and d.z < 5
                and d.z > -1
            ):
                filtered.append(d)
        return filtered

    def process_frame(self, detections: list[BallDetection3D]):
        """Updates the tracker for the current frame."""
        # Zero tolerance for floating-point errors like division by zero or np.mean() calls on empty arrays. Should be debugged immediately.
        with np.errstate(all="raise"):
            filtered = self.filter_detections(detections)
            if filtered:
                self.assign_detections_to_tracks(filtered)
            self.current_frame += 1

    def to_df(self):
        if not self.tracks:
            return None
        import pandas as pd

        rows = []
        for track_id, ball_track in self.tracks.items():
            for frame_id, camera_map in ball_track.detections.items():
                for camera_id, det in camera_map.items():
                    rows.append(
                        dict(
                            track_id=track_id,
                            frame=frame_id,
                            camera_id=camera_id,
                            x=det.x,
                            y=det.y,
                            z=det.z,
                        )
                    )
        df = pd.DataFrame(rows).sort_values(by=["track_id", "frame", "camera_id"])
        return df
