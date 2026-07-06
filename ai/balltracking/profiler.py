from __future__ import annotations

import time
from contextlib import contextmanager
from dataclasses import dataclass, field

import numpy as np

PROFILED_STAGE_ORDER = (
    "grab_frame",
    "retrieve_images",
    "bgra_to_bgr",
    "initial_localization",
    "stereo_detection",
    "stereo_match_triangulate",
    "world_transform",
    "tracker_process_frame",
    "track_emission",
    "stale_track_emission",
    "event_write_frame_processed",
    "frame_total",
)


@dataclass
class StageProfileStats:
    samples_s: list[float] = field(default_factory=list)
    total_s: float = 0.0

    def add(self, duration_s: float):
        self.samples_s.append(duration_s)
        self.total_s += duration_s


class StageProfiler:
    def __init__(self, enabled: bool):
        self.enabled = enabled
        self._stats: dict[str, StageProfileStats] = {}

    @contextmanager
    def measure(self, stage_name: str):
        start = time.perf_counter()
        try:
            yield
        finally:
            if self.enabled:
                self._stats.setdefault(stage_name, StageProfileStats()).add(time.perf_counter() - start)

    def print_summary(self):
        if not self.enabled:
            return

        frame_total = self._stats.get("frame_total")
        frame_samples = len(frame_total.samples_s) if frame_total is not None else 0
        if frame_samples == 0:
            print("\nProfiling summary: no frames were processed.")
            return

        frame_total_sum = max(frame_total.total_s, 1e-9)
        print("\nProfiling summary (steady-state frame stages)")
        print(
            f"{'stage':<28} {'total_s':>9} {'mean_ms':>9} {'p50_ms':>9} {'p95_ms':>9} {'share_%':>8}"
        )
        for stage_name in PROFILED_STAGE_ORDER:
            if stage_name in {"frame_total", "initial_localization"}:
                continue
            stats = self._stats.get(stage_name)
            if stats is None or not stats.samples_s:
                continue
            samples_ms = np.asarray(stats.samples_s, dtype=float) * 1000.0
            print(
                f"{stage_name:<28} "
                f"{stats.total_s:>9.3f} "
                f"{np.mean(samples_ms):>9.3f} "
                f"{np.percentile(samples_ms, 50):>9.3f} "
                f"{np.percentile(samples_ms, 95):>9.3f} "
                f"{(stats.total_s / frame_total_sum) * 100.0:>8.2f}"
            )

        frame_samples_ms = np.asarray(frame_total.samples_s, dtype=float) * 1000.0
        print(
            f"{'frame_total':<28} "
            f"{frame_total.total_s:>9.3f} "
            f"{np.mean(frame_samples_ms):>9.3f} "
            f"{np.percentile(frame_samples_ms, 50):>9.3f} "
            f"{np.percentile(frame_samples_ms, 95):>9.3f} "
            f"{100.0:>8.2f}"
        )

        localization_stats = self._stats.get("initial_localization")
        if localization_stats is not None and localization_stats.samples_s:
            localization_ms = np.asarray(localization_stats.samples_s, dtype=float) * 1000.0
            print("\nOne-time stages")
            print(
                f"{'stage':<28} {'count':>7} {'total_s':>9} {'mean_ms':>9} {'p50_ms':>9} {'p95_ms':>9}"
            )
            print(
                f"{'initial_localization':<28} "
                f"{len(localization_stats.samples_s):>7} "
                f"{localization_stats.total_s:>9.3f} "
                f"{np.mean(localization_ms):>9.3f} "
                f"{np.percentile(localization_ms, 50):>9.3f} "
                f"{np.percentile(localization_ms, 95):>9.3f}"
            )
