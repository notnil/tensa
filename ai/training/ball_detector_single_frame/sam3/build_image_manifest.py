#!/usr/bin/env python3
"""Build a CSV listing all image files in ./data."""

from pathlib import Path
import re

import pandas as pd


DATA_DIR = Path(__file__).resolve().parent / "data"
OUTPUT_CSV = Path(__file__).resolve().parent / "image_manifest.csv"
IMAGE_EXTS = {".jpg", ".jpeg", ".png"}
FILENAME_RE = re.compile(r"^(?P<clip>.+)_(?P<frame>\d+)$")


def main() -> None:
    if not DATA_DIR.is_dir():
        raise SystemExit(f"Data directory not found: {DATA_DIR}")

    records = []
    for file_path in DATA_DIR.iterdir():
        if not file_path.is_file() or file_path.suffix.lower() not in IMAGE_EXTS:
            continue

        match = FILENAME_RE.match(file_path.stem)
        if not match:
            print(f"Skipping file with unexpected name: {file_path.name}")
            continue

        clip_name = match.group("clip")
        frame_idx = int(match.group("frame"))
        records.append(
            {
                "path": f"./data/{file_path.name}",
                "clip_name": clip_name,
                "frame_idx": frame_idx,
            }
        )

    if not records:
        raise SystemExit("No image files matched the expected pattern.")

    df = (
        pd.DataFrame(records)
        .sort_values(["clip_name", "frame_idx"])
        .reset_index(drop=True)
    )
    # Validate no missing or empty values slipped in.
    if df.isna().any().any():
        raise ValueError("Found missing (NaN) values in the manifest.")
    if df[["path", "clip_name"]].eq("").any().any():
        raise ValueError("Found empty strings in required manifest fields.")
    unique_clips = df["clip_name"].nunique()
    df.to_csv(OUTPUT_CSV, index=False)
    print(f"Wrote {len(df)} rows to {OUTPUT_CSV}")
    print(f"Unique clips: {unique_clips}")


if __name__ == "__main__":
    main()
