#!/usr/bin/env python3
"""Convert labels.csv into Ultralytics YOLO directory structure with symlinked images."""

from __future__ import annotations

import csv
import json
import os
import struct
from pathlib import Path
from typing import Iterable, Tuple

from tqdm import tqdm

DATASET_NAME = "ball_detector_single_frame"
CSV_FILENAME = "labels.csv"
OUTPUT_DIRNAME = "yolo_dataset"
DATA_CONFIG_FILENAME = "yolo_dataset.yaml"
CLASS_NAMES = ["tennis ball"]

PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
SOF_MARKERS = {
    0xC0,
    0xC1,
    0xC2,
    0xC3,
    0xC5,
    0xC6,
    0xC7,
    0xC9,
    0xCA,
    0xCB,
    0xCD,
    0xCE,
    0xCF,
}


def _jpeg_size(file_obj) -> Tuple[int, int]:
    file_obj.seek(2)
    while True:
        marker_start = file_obj.read(1)
        if not marker_start:
            break
        if marker_start != b"\xFF":
            continue

        marker = file_obj.read(1)
        if not marker:
            break
        while marker == b"\xFF":
            marker = file_obj.read(1)
            if not marker:
                break
        if not marker:
            break

        marker_val = marker[0]
        if marker_val == 0xD9:
            break
        if marker_val == 0x01 or 0xD0 <= marker_val <= 0xD7:
            continue

        size_bytes = file_obj.read(2)
        if len(size_bytes) != 2:
            break
        segment_size = struct.unpack(">H", size_bytes)[0]

        if marker_val in SOF_MARKERS:
            if segment_size < 7:
                break
            file_obj.read(1)  # precision
            height, width = struct.unpack(">HH", file_obj.read(4))
            return int(width), int(height)

        if segment_size < 2:
            break
        file_obj.seek(segment_size - 2, 1)

    raise ValueError("Could not determine JPEG dimensions.")


def get_image_size(path: Path) -> Tuple[int, int]:
    with path.open("rb") as img_file:
        header = img_file.read(24)
        img_file.seek(0)

        if header.startswith(PNG_SIGNATURE):
            img_file.seek(16)
            width, height = struct.unpack(">II", img_file.read(8))
            return int(width), int(height)

        if header[:2] == b"\xFF\xD8":
            return _jpeg_size(img_file)

    raise ValueError(f"Unsupported image format for {path}")


def normalize_split(split: str) -> str:
    mapping = {"training": "train", "validation": "val", "testing": "test"}
    cleaned = (split or "").strip().lower()
    return mapping.get(cleaned, cleaned)


def has_ball(row: dict) -> bool:
    value = str(row.get("ball_present", "")).strip().lower()
    return value not in {"0", "false", "no", ""}


def to_box(row: dict, width: int, height: int) -> list[float]:
    xmin = float(row["bbox_x_min"])
    ymin = float(row["bbox_y_min"])
    xmax = float(row["bbox_x_max"])
    ymax = float(row["bbox_y_max"])

    box_width = xmax - xmin
    box_height = ymax - ymin
    x_center = xmin + box_width / 2
    y_center = ymin + box_height / 2

    return [
        0,
        x_center / width,
        y_center / height,
        box_width / width,
        box_height / height,
    ]


def load_rows(csv_path: Path) -> Iterable[dict]:
    with csv_path.open(newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            yield row


def ensure_symlink(src: Path, dest: Path) -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    if dest.exists() or dest.is_symlink():
        if dest.is_symlink() and dest.resolve() == src.resolve():
            return
        raise FileExistsError(f"Destination already exists and is not the expected symlink: {dest}")
    os.symlink(src, dest)


def write_label_file(path: Path, boxes: list[list[float]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w") as handle:
        for box in boxes:
            handle.write(" ".join(f"{v:.6f}" if i else str(int(v)) for i, v in enumerate(box)) + "\n")


def write_data_yaml(dataset_root: Path, splits_present: set[str]) -> None:
    data = {
        "path": dataset_root.as_posix(),
        "names": CLASS_NAMES,
        "train": "images/train" if "train" in splits_present else "",
        "val": "images/val" if "val" in splits_present else "",
        "test": "images/test" if "test" in splits_present else "",
    }
    yaml_lines = [
        f"path: {data['path']}",
        f"names: {json.dumps(data['names'])}",
    ]
    for split in ("train", "val", "test"):
        if data[split]:
            yaml_lines.append(f"{split}: {data[split]}")

    yaml_path = dataset_root.parent / DATA_CONFIG_FILENAME
    yaml_path.write_text("\n".join(yaml_lines) + "\n")


def main() -> None:
    base_dir = Path(__file__).resolve().parent
    csv_path = base_dir / CSV_FILENAME
    dataset_root = base_dir / OUTPUT_DIRNAME

    if dataset_root.exists():
        raise FileExistsError(f"{dataset_root} already exists. Remove it before rebuilding the dataset.")
    if not csv_path.exists():
        raise FileNotFoundError(f"labels.csv not found at {csv_path}")

    rows = list(load_rows(csv_path))

    images_root = dataset_root / "images"
    labels_root = dataset_root / "labels"
    dataset_root.mkdir(parents=True, exist_ok=True)

    splits_present: set[str] = set()
    missing_images: list[Path] = []
    iterator = tqdm(rows, desc="Building YOLO dataset", unit="img")

    for row in iterator:
        split = normalize_split(row.get("split", "train")) or "train"
        splits_present.add(split)

        img_rel = Path(row["path"])
        src_img = base_dir / img_rel
        if not src_img.exists():
            missing_images.append(src_img)
            continue

        dest_img = images_root / split / img_rel
        ensure_symlink(src_img, dest_img)

        width, height = get_image_size(src_img)
        boxes = [to_box(row, width, height)] if has_ball(row) else []

        dest_label = labels_root / split / img_rel.with_suffix(".txt")
        write_label_file(dest_label, boxes)

    if not any((labels_root / s).exists() for s in splits_present):
        raise RuntimeError("No labels were written. Dataset build aborted.")

    write_data_yaml(dataset_root, splits_present)

    print(
        f"YOLO dataset created at {dataset_root} with splits: {', '.join(sorted(splits_present))}. "
        f"Data config written to {DATA_CONFIG_FILENAME}."
    )
    if missing_images:
        sample = "\n  ".join(str(p) for p in missing_images[:5])
        print(f"Skipped {len(missing_images)} images missing on disk. First few:\n  {sample}")


if __name__ == "__main__":
    main()
