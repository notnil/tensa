#!/usr/bin/env python3
"""
Label integrity analysis and cleanup.

Detects labels that were saved to wrong frames due to a race condition
in the web labeler. Uses iterative rolling median trajectory smoothing
to identify outliers.

Usage:
    python analyze_labels.py --prefix /path/to/tensa-recordings [--threshold 3.0] [--window 21] [--dry-run]
"""

import argparse
import json
import math
import os
import sys
from datetime import datetime

import numpy as np


def rolling_median(arr, win):
    """Compute rolling median with given window size."""
    result = np.zeros_like(arr)
    half = win // 2
    for i in range(len(arr)):
        lo = max(0, i - half)
        hi = min(len(arr), i + half + 1)
        result[i] = np.median(arr[lo:hi])
    return result


def iterative_flag(xs, ys, win=21, threshold=3.0, iterations=5):
    """
    Iteratively smooth trajectory and flag outliers.
    
    1. Compute rolling median of X(t) and Y(t)
    2. Flag points >threshold from smoothed trajectory
    3. Exclude flagged points and re-smooth
    4. Repeat until convergence or max iterations
    """
    n = len(xs)
    flagged = np.zeros(n, dtype=bool)
    residuals = np.zeros(n)
    xs_smooth = xs.copy()
    ys_smooth = ys.copy()

    for it in range(iterations):
        clean_idx = np.where(~flagged)[0]
        if len(clean_idx) < 5:
            break

        xs_smooth_clean = rolling_median(xs[clean_idx], min(win, len(clean_idx)))
        ys_smooth_clean = rolling_median(ys[clean_idx], min(win, len(clean_idx)))

        # Interpolate smooth values back to all indices
        xs_smooth = np.interp(np.arange(n), clean_idx, xs_smooth_clean)
        ys_smooth = np.interp(np.arange(n), clean_idx, ys_smooth_clean)

        residuals = np.sqrt((xs - xs_smooth) ** 2 + (ys - ys_smooth) ** 2)
        new_flagged = residuals > threshold
        if np.array_equal(new_flagged, flagged):
            break
        flagged = new_flagged

    return flagged, residuals


def main():
    parser = argparse.ArgumentParser(description="Analyze and clean localization labels")
    parser.add_argument("--prefix", required=True, help="Path to tensa-recordings directory")
    parser.add_argument("--threshold", type=float, default=3.0, help="Distance threshold in meters (default: 3.0)")
    parser.add_argument("--window", type=int, default=21, help="Rolling median window size (default: 21)")
    parser.add_argument("--iterations", type=int, default=5, help="Number of smoothing iterations (default: 5)")
    parser.add_argument("--dry-run", action="store_true", help="Print stats only, don't write files")
    args = parser.parse_args()

    labels_path = os.path.join(args.prefix, "localization_labels.jsonl")
    queue_path = os.path.join(args.prefix, "localization_queue.txt")

    if not os.path.exists(labels_path):
        print(f"Error: {labels_path} not found")
        sys.exit(1)

    # Load labels
    labels = []
    with open(labels_path) as f:
        for line in f:
            line = line.strip()
            if line:
                labels.append(json.loads(line))
    print(f"Loaded {len(labels)} labels from {labels_path}")

    # Load queue
    queue_frames = set()
    if os.path.exists(queue_path):
        with open(queue_path) as f:
            queue_frames = set(line.strip() for line in f if line.strip())
        print(f"Loaded queue with {len(queue_frames)} frames")

    # Group by dataset
    datasets = {}
    for label in labels:
        ds = label["dataset"]
        if ds not in datasets:
            datasets[ds] = []
        datasets[ds].append(label)

    # Process each dataset
    clean_labels = []
    flagged_labels = []
    requeue_ids = set()

    print(f"\n{'Dataset':<30} {'Total':>6} {'Flagged':>8} {'Rate':>8}")
    print("-" * 56)

    for ds in sorted(datasets.keys()):
        ds_labels = sorted(datasets[ds], key=lambda l: int(l["timestamp_ns"]))

        if len(ds_labels) < 3:
            # Too few labels to analyze, keep them all
            clean_labels.extend(ds_labels)
            print(f"{ds:<30} {len(ds_labels):>6} {'(skip)':>8} {'':>8}")
            continue

        xs = np.array([l["machine_pose"]["x"] for l in ds_labels])
        ys = np.array([l["machine_pose"]["y"] for l in ds_labels])

        flagged, residuals = iterative_flag(
            xs, ys, win=args.window, threshold=args.threshold, iterations=args.iterations
        )

        n_flagged = int(np.sum(flagged))
        rate = n_flagged / len(ds_labels) * 100

        for i, label in enumerate(ds_labels):
            label["_residual"] = float(residuals[i])
            if flagged[i]:
                flagged_labels.append(label)
                fid = f"{label['dataset']}/{label['timestamp_ns']}"
                requeue_ids.add(fid)
            else:
                clean_labels.append(label)

        print(f"{ds:<30} {len(ds_labels):>6} {n_flagged:>8} {rate:>7.1f}%")

    print("-" * 56)
    print(f"{'TOTAL':<30} {len(labels):>6} {len(flagged_labels):>8} {len(flagged_labels)/len(labels)*100:>7.1f}%")
    print(f"\nClean labels: {len(clean_labels)}")
    print(f"Flagged labels: {len(flagged_labels)}")
    print(f"Frames to re-queue: {len(requeue_ids)}")

    if args.dry_run:
        print("\n[DRY RUN] No files written.")
        return

    # Write clean labels
    clean_path = os.path.join(args.prefix, "localization_labels_clean.jsonl")
    with open(clean_path, "w") as f:
        for label in clean_labels:
            out = {k: v for k, v in label.items() if not k.startswith("_")}
            f.write(json.dumps(out) + "\n")
    print(f"\nWrote {len(clean_labels)} clean labels to {clean_path}")

    # Write flagged labels (with residual for inspection)
    flagged_path = os.path.join(args.prefix, "localization_labels_flagged.jsonl")
    with open(flagged_path, "w") as f:
        for label in flagged_labels:
            f.write(json.dumps(label) + "\n")
    print(f"Wrote {len(flagged_labels)} flagged labels to {flagged_path}")

    # Backup original
    backup_path = labels_path + f".backup_{datetime.now().strftime('%Y%m%dT%H%M%S')}"
    os.rename(labels_path, backup_path)
    print(f"Backed up original to {backup_path}")

    # Replace with clean labels
    os.rename(clean_path, labels_path)
    print(f"Replaced {labels_path} with clean labels")

    # Add flagged frames back to queue for re-labeling
    new_queue = requeue_ids - queue_frames
    if new_queue:
        with open(queue_path, "a") as f:
            for fid in sorted(new_queue):
                f.write(fid + "\n")
        print(f"Added {len(new_queue)} frames back to queue for re-labeling")
    else:
        print("All flagged frames already in queue")


if __name__ == "__main__":
    main()
