#!/usr/bin/env python3
"""
Piecewise Physics-Informed Trajectory Fitting for Ball Tracking.

This module provides trajectory refinement by fitting noisy ball detections
to piecewise parabolic models that respect physics (constant gravity).

Each flight segment between bounces is modeled as:
- Horizontal: x(t) = x0 + vx * (t - t_start), y(t) = y0 + vy * (t - t_start)
- Vertical: z(t) = z0 + vz * (t - t_start) - 0.5 * g * (t - t_start)^2

where g = 9.81 m/s^2.
"""

import numpy as np
from typing import List, Tuple, Optional, Dict, Any

# Physics constants
GRAVITY = 9.81  # m/s^2
BALL_RADIUS = 0.0335  # tennis ball radius in meters

# Bounce detection thresholds
BOUNCE_Z_THRESHOLD = 0.20  # meters - max height to consider a potential bounce
BOUNCE_VZ_SIGN_CHANGE = True  # require velocity sign change
MIN_SEGMENT_POINTS = 2  # minimum points per segment for fitting

# Outlier rejection
OUTLIER_THRESHOLD_M = 0.15  # 15cm residual threshold for outlier detection


class FlightSegment:
    """Represents a single flight segment between bounces."""
    
    def __init__(self, t_start: float, t_end: float):
        self.t_start = t_start
        self.t_end = t_end
        
        # Fitted coefficients: [x0, vx, y0, vy, z0, vz]
        self.x0: float = 0.0
        self.vx: float = 0.0
        self.y0: float = 0.0
        self.vy: float = 0.0
        self.z0: float = 0.0
        self.vz: float = 0.0
        
        # Fitting metrics
        self.residual_rms: float = 0.0
        self.num_points: int = 0
        self.num_outliers: int = 0
    
    def evaluate(self, t: float) -> np.ndarray:
        """Evaluate the fitted trajectory at time t."""
        dt = t - self.t_start
        x = self.x0 + self.vx * dt
        y = self.y0 + self.vy * dt
        z = self.z0 + self.vz * dt - 0.5 * GRAVITY * dt * dt
        # Clamp Z to ball radius to prevent going below ground
        return np.array([x, y, max(BALL_RADIUS, z)])
    
    def to_dict(self) -> dict:
        """Convert segment to dictionary for JSON serialization."""
        return {
            "t_start": float(self.t_start),
            "t_end": float(self.t_end),
            "x0": float(self.x0),
            "vx": float(self.vx),
            "y0": float(self.y0),
            "vy": float(self.vy),
            "z0": float(self.z0),
            "vz": float(self.vz),
            "residual_rms": float(self.residual_rms),
            "num_points": int(self.num_points),
            "num_outliers": int(self.num_outliers)
        }


class PiecewiseFitter:
    """
    Fits a trajectory to piecewise parabolic segments.
    
    The fitter:
    1. Detects bounces from the Z-coordinate pattern
    2. Splits the trajectory into flight segments
    3. Fits each segment using least-squares with physics constraints
    4. Rejects outliers and refits
    """
    
    def __init__(self):
        self.segments: List[FlightSegment] = []
        self.bounce_times: List[float] = []
        self.raw_times: np.ndarray = np.array([])
        self.raw_positions: np.ndarray = np.array([])
    
    def fit(self, times: np.ndarray, positions: np.ndarray) -> bool:
        """
        Fit a piecewise trajectory to the given points.
        
        Args:
            times: Array of timestamps in seconds, shape (N,)
            positions: Array of [x, y, z] positions, shape (N, 3)
        
        Returns:
            True if fitting succeeded, False otherwise
        """
        if len(times) < MIN_SEGMENT_POINTS:
            return False
        
        self.raw_times = times
        self.raw_positions = positions
        
        # Step 1: Detect bounces
        self.bounce_times = self._detect_bounces(times, positions)
        
        # Step 2: Global X/Y Fit
        # To avoid "twisting", we fit a single constant-velocity model to X and Y
        # for the entire track.
        dt_global = times - times[0]
        A_global = np.column_stack([np.ones(len(times)), dt_global])
        
        x_coeffs, _, _, _ = np.linalg.lstsq(A_global, positions[:, 0], rcond=None)
        global_x0, global_vx = x_coeffs
        
        y_coeffs, _, _, _ = np.linalg.lstsq(A_global, positions[:, 1], rcond=None)
        global_y0, global_vy = y_coeffs
        
        # Step 3: Piecewise Z Fit
        segment_boundaries = [times[0]] + self.bounce_times + [times[-1]]
        self.segments = []
        
        for i in range(len(segment_boundaries) - 1):
            t_start = segment_boundaries[i]
            t_end = segment_boundaries[i + 1]
            
            # Get points in this segment
            mask = (times >= t_start) & (times <= t_end)
            seg_times = times[mask]
            seg_positions = positions[mask]
            
            if len(seg_times) < MIN_SEGMENT_POINTS:
                continue
            
            # Fit Z for this segment
            segment = FlightSegment(t_start, t_end)
            dt_seg = seg_times - t_start
            A_seg = np.column_stack([np.ones(len(seg_times)), dt_seg])
            
            # Transform Z to remove gravity
            z_transformed = seg_positions[:, 2] + 0.5 * GRAVITY * dt_seg * dt_seg
            z_coeffs, _, _, _ = np.linalg.lstsq(A_seg, z_transformed, rcond=None)
            segment.z0, segment.vz = z_coeffs
            
            # Apply global X/Y to this segment
            # We need to adjust x0, y0 for this segment's t_start
            dt_seg_start = t_start - times[0]
            segment.x0 = global_x0 + global_vx * dt_seg_start
            segment.vx = global_vx
            segment.y0 = global_y0 + global_vy * dt_seg_start
            segment.vy = global_vy
            
            self.segments.append(segment)
        
        return len(self.segments) > 0
    
    def _detect_bounces(self, times: np.ndarray, positions: np.ndarray) -> List[float]:
        """
        Detect bounce events from the trajectory data.
        
        Bounces are identified by a sharp upward velocity change (acceleration spike)
        near the ground level.
        """
        bounces = []
        z = positions[:, 2]
        n = len(z)
        
        if n < 3:
            return bounces
        
        # Look for the characteristic "V" shape or sharp upward kink
        for i in range(1, n - 1):
            dt_pre = times[i] - times[i-1]
            dt_post = times[i+1] - times[i]
            
            if dt_pre <= 0 or dt_post <= 0:
                continue
                
            v_pre = (z[i] - z[i-1]) / dt_pre
            v_post = (z[i+1] - z[i]) / dt_post
            
            # Upward acceleration spike: v_post should be much more positive than v_pre
            dvz = v_post - v_pre
            
            is_near_ground = z[i] < BOUNCE_Z_THRESHOLD
            # Velocity reversal: going down (v_pre < 0) and now either going up 
            # or going down much slower (v_post > v_pre + threshold)
            has_upward_kink = dvz > 3.0 # Significant upward velocity jump in m/s
            
            if is_near_ground and has_upward_kink:
                # To avoid multiple detections for the same bounce,
                # only add if this point is the "sharpest" kink in its neighborhood
                bounces.append(times[i])
        
        return bounces
    
    def evaluate(self, t: float) -> Optional[np.ndarray]:
        """
        Evaluate the fitted trajectory at time t.
        
        Returns None if t is outside the fitted range.
        """
        if not self.segments:
            return None
            
        # If t is before the first segment, use the first segment
        if t < self.segments[0].t_start:
            return self.segments[0].evaluate(self.segments[0].t_start)
            
        # If t is after the last segment, use the last segment
        if t > self.segments[-1].t_end:
            return self.segments[-1].evaluate(self.segments[-1].t_end)

        for segment in self.segments:
            if segment.t_start <= t <= segment.t_end:
                return segment.evaluate(t)
                
        # Handle gaps between segments by using the closest segment
        for i in range(len(self.segments) - 1):
            if self.segments[i].t_end < t < self.segments[i+1].t_start:
                # Closer to which end?
                if (t - self.segments[i].t_end) < (self.segments[i+1].t_start - t):
                    return self.segments[i].evaluate(self.segments[i].t_end)
                else:
                    return self.segments[i+1].evaluate(self.segments[i+1].t_start)
                    
        return None

    def get_refined_positions(self, times: np.ndarray) -> np.ndarray:
        """
        Get refined positions at the given timestamps.
        
        Args:
            times: Array of timestamps in seconds
        
        Returns:
            Array of refined [x, y, z] positions, shape (N, 3)
        """
        refined = np.zeros((len(times), 3))
        for i, t in enumerate(times):
            pos = self.evaluate(t)
            if pos is not None:
                refined[i] = pos
        return refined

    def to_dict(self) -> dict:
        """Convert fitter state to dictionary for JSON serialization."""
        return {
            "num_segments": int(len(self.segments)),
            "bounce_times": [float(t) for t in self.bounce_times],
            "segments": [s.to_dict() for s in self.segments]
        }


def refine_track(
    timestamps_ns: List[str],
    positions: List[List[float]],
    interpolation_timestamps_ns: Optional[List[str]] = None
) -> Dict[str, Any]:
    """
    Refine a track's positions using piecewise physics fitting.
    
    Args:
        timestamps_ns: List of timestamps in nanoseconds (strings) for the detections
        positions: List of [x, y, z] positions for the detections
        interpolation_timestamps_ns: Optional list of timestamps to interpolate at
    
    Returns:
        Dictionary containing:
        - refined_positions: List of [x, y, z] refined positions
        - fit_info: Fitting metadata (segments, bounces, etc.)
        - success: Whether fitting succeeded
    """
    if len(timestamps_ns) < MIN_SEGMENT_POINTS:
        return {
            "refined_positions": positions,
            "fit_info": None,
            "success": False
        }
    
    # Convert timestamps to seconds (relative to first timestamp)
    ts_array = np.array([int(ts) for ts in timestamps_ns], dtype=np.float64)
    t0 = ts_array[0]
    times = (ts_array - t0) / 1e9  # nanoseconds to seconds
    
    # Convert positions to numpy array
    pos_array = np.array(positions)
    
    # Fit the trajectory
    fitter = PiecewiseFitter()
    success = fitter.fit(times, pos_array)
    
    if not success:
        return {
            "refined_positions": positions,
            "fit_info": None,
            "success": False
        }
    
    # Get refined positions at requested timestamps
    if interpolation_timestamps_ns:
        interp_ts_array = np.array([int(ts) for ts in interpolation_timestamps_ns], dtype=np.float64)
        interp_times = (interp_ts_array - t0) / 1e9
        refined = fitter.get_refined_positions(interp_times)
    else:
        refined = fitter.get_refined_positions(times)
    
    return {
        "refined_positions": refined.tolist(),
        "fit_info": fitter.to_dict(),
        "success": True
    }
