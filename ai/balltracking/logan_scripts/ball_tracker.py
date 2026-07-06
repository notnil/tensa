#!/usr/bin/env python3
"""
Shared utilities for ball tracking and trajectory management.
"""

import os
import sys
import json
import numpy as np
from typing import Dict, List, Optional, Tuple

# Import camera extrinsics from localize_camera
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from localize_camera import CAMERA_EXTRINSICS
from trajectory_refinement import refine_track

# ============================================================================
# Configuration
# ============================================================================

# Exclusion zones (pixel coordinates, 1920x1080 images)
EXCLUSION_ZONES = {
    "left": {"pixel_y_min": 810, "pixel_x_min": 960},  # bottom 1/4 AND right half
    "right": {"pixel_y_min": 810},  # bottom 1/4
}

# Court-space filter: only keep balls where y < this value
COURT_Y_MAX_M = -0.305  # ~1 foot from net

# Tracking parameters
MAX_ASSOCIATION_DIST_M = 5.0  # max distance to associate detection with track
MIN_TRACK_FRAMES = 3  # discard tracks shorter than this
MAX_GAP_FRAMES = 5  # terminate track after this many frames without detection
STATIONARY_VELOCITY_THRESHOLD_MPS = 0.1  # velocity below this = stationary
STATIONARY_FRAME_COUNT = 3  # consecutive frames below threshold to mark stationary
MIN_Y_DISPLACEMENT_M = 3.0  # minimum Y displacement to keep a trajectory
MIN_DETECTION_CONFIDENCE = 0.5  # ignore detections below this confidence
MIN_TRAJECTORY_VELOCITY_MPS = 2.0  # minimum average velocity to keep trajectory (filters slow rolling balls)

# Physics constants
GRAVITY = 9.81  # m/s^2
COEFFICIENT_OF_RESTITUTION = 0.75  # energy loss on bounce (tennis ball ~0.75)
BALL_RADIUS = 0.0335  # tennis ball radius in meters

# Regression constants (Dimension = K / distance) for geometric depth refinement
# Derived from front camera data: MeanDim = 113.22 / distance
CAMERA_GEOMETRIC_CONSTANTS = {
    "front": 113.22,
}

# Assumed frame rate (will be computed from timestamps if possible)
DEFAULT_FRAME_RATE_HZ = 30.0


# ============================================================================
# Detection Filtering
# ============================================================================

def is_in_exclusion_zone(cam: str, pixel_x: float, pixel_y: float) -> bool:
    """Check if a detection is in the exclusion zone for this camera."""
    if cam not in EXCLUSION_ZONES:
        return False
    
    zone = EXCLUSION_ZONES[cam]
    
    # Check y threshold (required for all exclusion zones)
    if "pixel_y_min" in zone and pixel_y <= zone["pixel_y_min"]:
        return False
    
    # For left camera, also need x threshold
    if cam == "left":
        if "pixel_x_min" in zone and pixel_x <= zone["pixel_x_min"]:
            return False
        # In exclusion zone if y > threshold AND x > threshold
        return pixel_y > zone.get("pixel_y_min", 0) and pixel_x > zone.get("pixel_x_min", 0)
    
    # For right camera, only y threshold matters
    return pixel_y > zone.get("pixel_y_min", 0)


def transform_to_world(
    p_camera: np.ndarray,
    cam: str,
    machine_pos: np.ndarray,
    yaw_deg: float,
    extrinsics_dict: Optional[Dict] = None
) -> np.ndarray:
    """Transform a point from camera coordinates to world coordinates."""
    if extrinsics_dict is not None:
        ext = extrinsics_dict[cam]
    else:
        ext = CAMERA_EXTRINSICS[cam]
    
    # Camera to machine frame
    p_machine = ext["t"] + ext["R"] @ p_camera
    
    # Machine to world frame (rotation around Z axis)
    # The yaw in labels.jsonl is machine_yaw_deg where 0=right(+X), 90=front(+Y)
    # Internal rotation: yaw_rad = yaw_deg - 90
    yaw_rad = np.radians(yaw_deg - 90)
    cos_y, sin_y = np.cos(yaw_rad), np.sin(yaw_rad)
    R_yaw = np.array([
        [cos_y, -sin_y, 0],
        [sin_y,  cos_y, 0],
        [0,      0,     1]
    ])
    
    p_world = R_yaw @ p_machine + machine_pos
    return p_world


def is_in_court_bounds(p_world: np.ndarray) -> bool:
    """Check if a world-space point passes the court filter."""
    # Only keep balls on near court (y < -0.305m, i.e., >1ft from net)
    return p_world[1] < COURT_Y_MAX_M


# ============================================================================
# Trajectory Tracking
# ============================================================================

class Track:
    """Represents a single ball trajectory."""
    
    def __init__(self, track_id: int, frame_idx: int, position: np.ndarray, 
                 cam: str, confidence: float, timestamp_ns: str):
        self.track_id = track_id
        self.positions: List[np.ndarray] = [position]
        self.frame_indices: List[int] = [frame_idx]
        self.timestamps: List[str] = [timestamp_ns]
        self.cameras: List[str] = [cam]
        self.confidences: List[float] = [confidence]
        self.velocity: Optional[np.ndarray] = None
        self.frames_since_update = 0
        self.stationary_count = 0
        self.is_stationary = False
    
    @property
    def last_position(self) -> np.ndarray:
        return self.positions[-1]
    
    @property
    def last_frame_idx(self) -> int:
        return self.frame_indices[-1]
    
    def predict_position(self, dt: float) -> np.ndarray:
        """Predict next position using physics-aware model with gravity and bounce."""
        if self.velocity is None:
            return self.last_position
        
        pos = self.last_position.copy()
        vel = self.velocity.copy()
        
        # Apply gravity to vertical velocity (z component)
        vel[2] -= GRAVITY * dt
        
        # Predict new position
        new_pos = pos + vel * dt
        
        # Check for bounce (ball hits ground)
        if new_pos[2] < BALL_RADIUS:
            # Time to hit ground
            # z + vz*t - 0.5*g*t^2 = BALL_RADIUS
            # Simplified: assume bounce happens, reflect z velocity
            new_pos[2] = BALL_RADIUS
            vel[2] = -vel[2] * COEFFICIENT_OF_RESTITUTION
            
            # Continue with remaining time after bounce (simplified)
            # For now just set to ball radius height
        
        return new_pos
    
    def update(self, frame_idx: int, position: np.ndarray, cam: str, 
               confidence: float, timestamp_ns: str, dt: float):
        """Update track with new detection."""
        # Compute velocity (accounting for gravity)
        if dt > 0:
            # Raw velocity from position change
            raw_velocity = (position - self.last_position) / dt
            
            # Adjust z-velocity to account for gravity effect during dt
            # The observed velocity includes gravity's effect, so we estimate
            # the "launch" velocity by adding back half the gravity effect
            new_velocity = raw_velocity.copy()
            new_velocity[2] += 0.5 * GRAVITY * dt  # Compensate for gravity
            
            if self.velocity is not None:
                # Smooth velocity with exponential moving average
                self.velocity = 0.5 * self.velocity + 0.5 * new_velocity
            else:
                self.velocity = new_velocity
            
            # Check for stationary
            speed = np.linalg.norm(self.velocity)
            if speed < STATIONARY_VELOCITY_THRESHOLD_MPS:
                self.stationary_count += 1
                if self.stationary_count >= STATIONARY_FRAME_COUNT:
                    self.is_stationary = True
            else:
                self.stationary_count = 0
                self.is_stationary = False
        
        self.positions.append(position)
        self.frame_indices.append(frame_idx)
        self.timestamps.append(timestamp_ns)
        self.cameras.append(cam)
        self.confidences.append(confidence)
        self.frames_since_update = 0
    
    def mark_missed(self):
        """Mark that this track had no detection this frame."""
        self.frames_since_update += 1
    
    def is_terminated(self) -> bool:
        """Check if track should be terminated."""
        return self.frames_since_update >= MAX_GAP_FRAMES
    
    def get_y_displacement(self) -> float:
        """Calculate the Y displacement (max - min) of this track."""
        if len(self.positions) < 2:
            return 0.0
        y_values = [p[1] for p in self.positions]
        return max(y_values) - min(y_values)
    
    def get_average_velocity(self) -> float:
        """Calculate average velocity (total distance / total time)."""
        if len(self.positions) < 2 or len(self.timestamps) < 2:
            return 0.0
        
        # Total distance traveled
        total_dist = 0.0
        for i in range(1, len(self.positions)):
            dist = np.linalg.norm(self.positions[i] - self.positions[i-1])
            total_dist += dist
        
        # Total time
        start_ts = int(self.timestamps[0])
        end_ts = int(self.timestamps[-1])
        total_time = (end_ts - start_ts) / 1e9  # nanoseconds to seconds
        
        if total_time <= 0:
            return 0.0
        
        return total_dist / total_time
    
    def is_valid(self) -> bool:
        """Check if track meets all criteria to keep."""
        if len(self.positions) < MIN_TRACK_FRAMES:
            return False
        if self.is_stationary:
            return False
        if self.get_y_displacement() < MIN_Y_DISPLACEMENT_M:
            return False
        if self.get_average_velocity() < MIN_TRAJECTORY_VELOCITY_MPS:
            return False
        return True
    
    def to_dict(self, refine: bool = False, all_timestamps: Optional[List[str]] = None) -> dict:
        """Convert track to dictionary for JSON output.
        
        Args:
            refine: If True, include refined positions from physics-based fitting.
            all_timestamps: Full list of timestamps in recording for interpolation.
        """
        result = {
            "track_id": self.track_id,
            "start_frame": self.frame_indices[0],
            "end_frame": self.frame_indices[-1],
            "num_detections": len(self.positions),
            "positions": [[float(p[0]), float(p[1]), float(p[2])] for p in self.positions],
            "frame_indices": self.frame_indices,
            "timestamps": self.timestamps,
            "cameras": self.cameras,
            "confidences": self.confidences
        }
        
        if refine and all_timestamps:
            # Get the range of frame indices
            start_idx = self.frame_indices[0]
            end_idx = self.frame_indices[-1]
            
            # Collect interpolation timestamps for EVERY frame in the range
            interp_ts = all_timestamps[start_idx : end_idx + 1]
            
            # Apply physics-based trajectory refinement with interpolation
            refinement = refine_track(
                self.timestamps,
                [[float(p[0]), float(p[1]), float(p[2])] for p in self.positions],
                interpolation_timestamps_ns=interp_ts
            )
            result["refined_positions"] = refinement["refined_positions"]
            result["refinement_info"] = refinement["fit_info"]
            result["refinement_success"] = refinement["success"]
        
        return result


class BallTracker:
    """Manages ball tracking across frames."""
    
    def __init__(self, all_timestamps: List[str]):
        self.active_tracks: List[Track] = []
        self.completed_tracks: List[Track] = []
        self.next_track_id = 1
        self.last_timestamp_ns: Optional[int] = None
        self.all_timestamps = all_timestamps # Full list of all timestamps in recording
    
    def process_frame(
        self,
        frame_idx: int,
        timestamp_ns: str,
        detections: List[dict]
    ) -> List[dict]:
        """
        Process detections for a single frame.
        
        Args:
            frame_idx: Frame index
            timestamp_ns: Timestamp in nanoseconds (string)
            detections: List of filtered world-space detections with keys:
                        cam, confidence, x, y, z
        
        Returns:
            List of detections with track_id assigned
        """
        # Compute dt from timestamps
        current_ts = int(timestamp_ns)
        if self.last_timestamp_ns is not None:
            dt = (current_ts - self.last_timestamp_ns) / 1e9
        else:
            dt = 1.0 / DEFAULT_FRAME_RATE_HZ
        self.last_timestamp_ns = current_ts
        
        # Predict positions for active tracks
        predictions = []
        for track in self.active_tracks:
            pred_pos = track.predict_position(dt)
            predictions.append((track, pred_pos))
        
        # Match detections to tracks using nearest neighbor
        assigned_detections = []
        unassigned_detections = []
        matched_tracks = set()
        
        for det in detections:
            det_pos = np.array([det["x"], det["y"], det["z"]])
            
            # Find nearest track
            best_track = None
            best_dist = MAX_ASSOCIATION_DIST_M
            
            for track, pred_pos in predictions:
                if track.track_id in matched_tracks:
                    continue
                dist = np.linalg.norm(det_pos - pred_pos)
                if dist < best_dist:
                    best_dist = dist
                    best_track = track
            
            if best_track is not None:
                # Update track
                best_track.update(
                    frame_idx, det_pos, det["cam"], det["confidence"],
                    timestamp_ns, dt
                )
                matched_tracks.add(best_track.track_id)
                assigned_detections.append({
                    **det,
                    "track_id": best_track.track_id
                })
            else:
                unassigned_detections.append(det)
        
        # Create new tracks for unassigned detections
        for det in unassigned_detections:
            det_pos = np.array([det["x"], det["y"], det["z"]])
            new_track = Track(
                self.next_track_id, frame_idx, det_pos,
                det["cam"], det["confidence"], timestamp_ns
            )
            self.active_tracks.append(new_track)
            assigned_detections.append({
                **det,
                "track_id": self.next_track_id
            })
            self.next_track_id += 1
        
        # Mark unmatched tracks and terminate old ones
        for track in self.active_tracks:
            if track.track_id not in matched_tracks:
                track.mark_missed()
        
        # Move terminated tracks to completed
        still_active = []
        for track in self.active_tracks:
            if track.is_terminated():
                if track.is_valid():
                    self.completed_tracks.append(track)
            else:
                still_active.append(track)
        self.active_tracks = still_active
        
        return assigned_detections
    
    def finalize(self) -> List[Track]:
        """Finalize tracking and return all valid tracks."""
        # Move remaining active tracks to completed
        for track in self.active_tracks:
            if track.is_valid():
                self.completed_tracks.append(track)
        self.active_tracks = []
        
        return self.completed_tracks


# ============================================================================
# Utility Functions
# ============================================================================

def load_labels(labels_path: str) -> Dict[str, dict]:
    """Load labels.jsonl and return dict mapping timestamp_ns -> label data."""
    labels = {}
    if not os.path.exists(labels_path):
        return labels
    with open(labels_path, 'r') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            record = json.loads(line)
            ts = record.get("timestamp_ns", "")
            if ts:
                labels[str(ts)] = record
    return labels


def load_ball_detections(det_path: str) -> List[dict]:
    """Load ball_detections.json for a frame."""
    if not os.path.exists(det_path):
        return []
    with open(det_path, 'r') as f:
        return json.load(f)

def get_all_frame_timestamps(input_dir: str) -> List[str]:
    """Get all timestamp folders in the input directory."""
    timestamps = []
    if not os.path.exists(input_dir):
        return []
    for entry in os.listdir(input_dir):
        entry_path = os.path.join(input_dir, entry)
        if os.path.isdir(entry_path) and entry.isdigit():
            timestamps.append(entry)
    return sorted(timestamps, key=lambda x: int(x))
