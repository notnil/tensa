"""Trajectory refinement: piecewise physics-informed fitting with bounce detection."""

import numpy as np
from dataclasses import dataclass

GRAVITY = 9.81  # m/s²
# Bounce detection tuning for RFU (Z-up) tracks from visualize_results_rerun.py.
# The detector is plane-independent: no absolute z=0 reliance.
# Method summary:
# 1) Build high-recall bounce candidates from vertical motion reversals.
# 2) Handle flat/noisy minima with windowed trend checks.
# 3) Filter candidates by local low-envelope proximity, restitution plausibility,
#    and temporal spacing between accepted bounces.
# Why this design:
# - Camera pose noise biases absolute Z, so fixed ground-plane thresholds are brittle.
# - Velocity reversal is a strong impact signal even with vertical offset.
# - Flat minima are common with frame quantization/noisy detections, so we include
#   a windowed trend trigger to avoid missing obvious bounces.
# - Restitution and spacing gates suppress false positives from jitter.
BOUNCE_LOW_ENVELOPE_WINDOW = 9  # frames
BOUNCE_MAX_HEIGHT_ABOVE_LOW = 0.32  # m
BOUNCE_MIN_DOWNWARD_SPEED = 0.8  # m/s
BOUNCE_MIN_UPWARD_SPEED = 0.8  # m/s
BOUNCE_MIN_DV = 1.4  # m/s
BOUNCE_TRIGGER_WINDOW = 2  # frames on each side for robust pre/post slope
BOUNCE_MIN_NET_DROP = 0.05  # m
BOUNCE_MIN_NET_RISE = 0.05  # m
BOUNCE_LOCAL_MIN_EPS = 0.04  # m
BOUNCE_MIN_RESTITUTION = 0.12
BOUNCE_MAX_RESTITUTION = 1.30
BOUNCE_MIN_FRAMES_BETWEEN = 3


@dataclass
class BounceEvent:
    """A detected bounce point."""

    index: int  # index into the track's position array
    timestamp_ns: int
    position: np.ndarray


@dataclass
class BounceCandidate:
    """A bounce candidate, accepted or rejected, with diagnostic metadata."""

    index: int
    timestamp_ns: int
    position: np.ndarray
    accepted: bool
    rejection_reason: str | None
    v_before: float
    v_after: float
    dv: float
    restitution: float
    height_above_low: float


@dataclass
class RefinedTrajectory:
    """Result of trajectory refinement."""

    track_id: int
    timestamps_ns: np.ndarray
    raw_positions: np.ndarray  # (N, 3)
    refined_positions: np.ndarray  # (N, 3)
    bounces: list[BounceEvent]
    bounce_candidates: list[BounceCandidate]
    x_fit: tuple  # (x0, vx) — X = court-right
    z_fit: tuple  # (z0, vz) — Z = court-up
    y_segments: list[tuple]  # [(z0, vz) per segment] on gravity axis (Z-up)
    residual: float  # RMS fit error


def detect_bounces(
    positions: np.ndarray,
    timestamps_ns: np.ndarray,
) -> tuple[list[int], list[BounceCandidate]]:
    """Detect bounce indices from raw positions.

    The detector is intentionally plane-independent:
    - Candidate trigger: vertical velocity reversal + minimum impulse.
    - Acceptance filters: local low-envelope proximity, restitution plausibility,
      and minimum spacing from previous accepted bounce.
    """

    n = len(positions)
    if n < 3:
        return [], []

    # Smooth height slightly to reduce single-frame jitter spikes before
    # computing velocities and local envelopes.
    # Why: one-frame outliers can create fake sign flips in v_up.
    z_raw = positions[:, 2]
    if len(z_raw) >= 3:
        z = np.convolve(z_raw, np.array([0.25, 0.5, 0.25], dtype=float), mode="same")
        z[0] = z_raw[0]
        z[-1] = z_raw[-1]
    else:
        z = z_raw

    dts = np.diff(timestamps_ns) * 1e-9
    # Court space in visualize_results_rerun.py is RFU: Z is up.
    v_up = np.diff(z) / np.maximum(dts, 1e-9)

    # Per-track local low envelope for "near-ground" gating without depending
    # on a globally accurate ground plane.
    # Why: this follows each track's own baseline when global Z drifts.
    half_window = max(1, BOUNCE_LOW_ENVELOPE_WINDOW // 2)
    low_env = np.zeros_like(z, dtype=float)
    for i in range(len(z)):
        start = max(0, i - half_window)
        end = min(len(z), i + half_window + 1)
        low_env[i] = float(np.min(z[start:end]))

    bounce_indices: list[int] = []
    candidates: list[BounceCandidate] = []
    last_bounce_idx = -10_000

    for i in range(1, len(v_up)):
        v_before = float(v_up[i - 1])
        v_after = float(v_up[i])
        dv = float(v_after - v_before)
        height_above_low = float(z[i] - low_env[i])

        # Windowed trend around the candidate index to survive flat minima where
        # single-frame sign flips are weak/ambiguous.
        # Why: if several consecutive points are nearly flat at the minimum,
        # v_before/v_after alone can be too small to pass strict sign checks.
        pre_idx = max(0, i - BOUNCE_TRIGGER_WINDOW)
        post_idx = min(n - 1, i + BOUNCE_TRIGGER_WINDOW)
        dt_pre = max((timestamps_ns[i] - timestamps_ns[pre_idx]) * 1e-9, 1e-9)
        dt_post = max((timestamps_ns[post_idx] - timestamps_ns[i]) * 1e-9, 1e-9)
        v_before_win = float((z[i] - z[pre_idx]) / dt_pre)
        v_after_win = float((z[post_idx] - z[i]) / dt_post)
        dv_win = float(v_after_win - v_before_win)

        drop = float(z[pre_idx] - z[i])
        rise = float(z[post_idx] - z[i])
        window_min = float(np.min(z[pre_idx : post_idx + 1]))
        is_local_min = z[i] <= (window_min + BOUNCE_LOCAL_MIN_EPS)

        # Stage 1 (candidate generation): high recall by combining:
        # - classic impact signature (downward -> upward + impulse),
        # - flat-minimum signature from windowed descent/ascent + local min.
        # Why: missing a true bounce at this stage cannot be recovered later.
        down_ok = min(v_before, v_before_win) <= -BOUNCE_MIN_DOWNWARD_SPEED
        up_ok = max(v_after, v_after_win) >= BOUNCE_MIN_UPWARD_SPEED
        impulse_ok = max(dv, dv_win) >= BOUNCE_MIN_DV
        classic_trigger = down_ok and up_ok and impulse_ok
        flat_min_trigger = (
            is_local_min
            and v_before_win <= -0.45
            and v_after_win >= 0.45
            and drop >= BOUNCE_MIN_NET_DROP
            and rise >= BOUNCE_MIN_NET_RISE
        )
        is_candidate = classic_trigger or flat_min_trigger
        if not is_candidate:
            continue

        # Stage 2 (candidate filtering): remove physically implausible or
        # duplicate candidates while keeping true bounces.
        # Why:
        # - height_above_low rejects "mid-air reversals" from noisy tracks,
        # - restitution bounds reject non-impact kinematic artifacts,
        # - min spacing prevents one impact being double-counted across frames.
        v_down = max(abs(v_before), abs(v_before_win))
        v_upward = max(abs(v_after), abs(v_after_win))
        restitution = float(v_upward / max(v_down, 1e-6))

        rejection_reason = None
        if height_above_low > BOUNCE_MAX_HEIGHT_ABOVE_LOW:
            rejection_reason = "too_high_above_low_envelope"
        elif restitution < BOUNCE_MIN_RESTITUTION:
            rejection_reason = "restitution_too_low"
        elif restitution > BOUNCE_MAX_RESTITUTION:
            rejection_reason = "restitution_too_high"
        elif (i - last_bounce_idx) < BOUNCE_MIN_FRAMES_BETWEEN:
            rejection_reason = "too_close_to_previous_bounce"

        accepted = rejection_reason is None
        if accepted:
            bounce_indices.append(i)
            last_bounce_idx = i

        candidates.append(
            BounceCandidate(
                index=i,
                timestamp_ns=int(timestamps_ns[i]),
                position=positions[i].copy(),
                accepted=accepted,
                rejection_reason=rejection_reason,
                v_before=v_before,
                v_after=v_after,
                dv=dv,
                restitution=restitution,
                height_above_low=height_above_low,
            )
        )

    return bounce_indices, candidates


def _fit_linear(t: np.ndarray, y: np.ndarray) -> tuple[float, float]:
    """Fit y = y0 + v*t via least squares. Returns (y0, v)."""

    if len(t) < 2:
        return (y[0] if len(y) > 0 else 0.0, 0.0)
    A = np.column_stack([np.ones_like(t), t])
    result, *_ = np.linalg.lstsq(A, y, rcond=None)
    return (result[0], result[1])


def refine_trajectory(
    track_id: int,
    positions: np.ndarray,
    timestamps_ns: np.ndarray,
) -> RefinedTrajectory:
    """Apply piecewise physics-informed fitting per methodology §5.

    Coordinate system used by visualize_results_rerun.py: RFU (X=right, Y=forward, Z=up).
    Gravity acts on Z. Constant-velocity fit on X and Y.

    1. Global constant-velocity fit for X and Y
    2. Detect bounces (on Z axis)
    3. Per-segment gravity-constrained Z fit
    4. Evaluate fitted model at all timestamps
    """

    n = len(positions)
    t = (timestamps_ns - timestamps_ns[0]) * 1e-9  # seconds from start

    # Step 1: Global linear fits on non-gravity axes.
    x0, vx = _fit_linear(t, positions[:, 0])
    y0, vy = _fit_linear(t, positions[:, 1])
    z0, vz = _fit_linear(t, positions[:, 2])

    # Step 2: Bounce detection on Z (up axis).
    bounce_indices, bounce_candidates = detect_bounces(positions, timestamps_ns)
    bounces = [
        BounceEvent(index=i, timestamp_ns=timestamps_ns[i], position=positions[i].copy())
        for i in bounce_indices
    ]

    # Step 3: Per-segment fit on gravity axis (Z) with gravity constraint.
    # Split into segments at bounce points
    segment_boundaries = [0] + bounce_indices + [n]
    y_segments = []
    refined_z = np.zeros(n)

    for seg_idx in range(len(segment_boundaries) - 1):
        start = segment_boundaries[seg_idx]
        end = segment_boundaries[seg_idx + 1]
        seg_t = t[start:end]
        seg_z = positions[start:end, 2]

        if len(seg_t) < 2:
            # Not enough points for a fit; use raw
            refined_z[start:end] = seg_z
            y_segments.append((seg_z[0] if len(seg_z) > 0 else 0.0, 0.0))
            continue

        # Transform: z_transformed = z_observed + 0.5 * g * t².
        # Then fit z_transformed = z0 + vz * t (linear).
        t_local = seg_t - seg_t[0]
        z_transformed = seg_z + 0.5 * GRAVITY * t_local**2
        z0_seg, vz_seg = _fit_linear(t_local, z_transformed)
        y_segments.append((z0_seg, vz_seg))

        # Evaluate: z_fitted = z0 + vz*t - 0.5*g*t².
        refined_z[start:end] = z0_seg + vz_seg * t_local - 0.5 * GRAVITY * t_local**2

    # Step 4: Assemble refined positions [X, Y, Z].
    refined_positions = np.column_stack([
        x0 + vx * t,
        y0 + vy * t,
        refined_z,
    ])

    # Compute residual
    residual = np.sqrt(np.mean(np.sum((positions - refined_positions) ** 2, axis=1)))

    return RefinedTrajectory(
        track_id=track_id,
        timestamps_ns=timestamps_ns,
        raw_positions=positions,
        refined_positions=refined_positions,
        bounces=bounces,
        bounce_candidates=bounce_candidates,
        x_fit=(x0, vx),
        z_fit=(z0, vz),
        y_segments=y_segments,
        residual=residual,
    )
