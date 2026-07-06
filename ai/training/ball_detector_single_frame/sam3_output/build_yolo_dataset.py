#!/usr/bin/env python3
"""Convert SAM3 predictions into Ultralytics YOLO directory structure with symlinked images."""

from __future__ import annotations

import json
import os
import random
import struct
from pathlib import Path
from typing import Iterable, Tuple

from tqdm import tqdm

DATASET_NAME = "ball_detector_single_frame"
PREDICTIONS_FILENAME = "predictions.jsonl"
OUTPUT_DIRNAME = "yolo_dataset"
DATA_CONFIG_FILENAME = "yolo_dataset.yaml"
CLASS_NAMES = ["tennis ball"]
PROMPTS_TO_USE = {"tennis ball", "small tennis ball"}
NMS_IOU_THRESHOLD = 0.7
VAL_FRACTION = 0.1
SPLIT_SEED = 1337

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


def build_split_map(count: int, val_fraction: float, seed: int) -> list[str]:
    if count < 2 or val_fraction <= 0.0:
        return ["train"] * count
    rng = random.Random(seed)
    indices = list(range(count))
    rng.shuffle(indices)
    val_count = max(1, int(round(count * val_fraction)))
    val_indices = set(indices[:val_count])
    return ["val" if idx in val_indices else "train" for idx in range(count)]


def to_box(box: list[float], width: int, height: int) -> list[float]:
    xmin, ymin, xmax, ymax = map(float, box)
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


def load_predictions(jsonl_path: Path) -> Iterable[dict]:
    with jsonl_path.open() as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            yield json.loads(line)


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


def iou(box_a: list[float], box_b: list[float]) -> float:
    ax1, ay1, ax2, ay2 = box_a
    bx1, by1, bx2, by2 = box_b
    inter_x1 = max(ax1, bx1)
    inter_y1 = max(ay1, by1)
    inter_x2 = min(ax2, bx2)
    inter_y2 = min(ay2, by2)
    inter_w = max(0.0, inter_x2 - inter_x1)
    inter_h = max(0.0, inter_y2 - inter_y1)
    inter_area = inter_w * inter_h
    if inter_area == 0.0:
        return 0.0
    area_a = max(0.0, ax2 - ax1) * max(0.0, ay2 - ay1)
    area_b = max(0.0, bx2 - bx1) * max(0.0, by2 - by1)
    denom = area_a + area_b - inter_area
    return inter_area / denom if denom else 0.0


def nms(detections: list[tuple[list[float], float]], iou_threshold: float) -> list[list[float]]:
    if not detections:
        return []
    detections = sorted(detections, key=lambda item: item[1], reverse=True)
    kept: list[list[float]] = []
    while detections:
        box, _score = detections.pop(0)
        kept.append(box)
        remaining = []
        for candidate_box, candidate_score in detections:
            if iou(box, candidate_box) <= iou_threshold:
                remaining.append((candidate_box, candidate_score))
        detections = remaining
    return kept


def collect_boxes(prediction: dict) -> list[tuple[list[float], float]]:
    detections: list[tuple[list[float], float]] = []
    for prompt_entry in prediction.get("prompts", []):
        prompt = str(prompt_entry.get("prompt", "")).strip().lower()
        if prompt not in PROMPTS_TO_USE:
            continue
        for detection in prompt_entry.get("detections", []):
            box = detection.get("box")
            if not box or len(box) != 4:
                continue
            score = float(detection.get("score", 0.0))
            detections.append((list(map(float, box)), score))
    return detections


def main() -> None:
    script_dir = Path(__file__).resolve().parent
    base_dir = script_dir.parent
    predictions_path = script_dir / PREDICTIONS_FILENAME
    dataset_root = script_dir / OUTPUT_DIRNAME

    if dataset_root.exists():
        raise FileExistsError(f"{dataset_root} already exists. Remove it before rebuilding the dataset.")
    if not predictions_path.exists():
        raise FileNotFoundError(f"predictions.jsonl not found at {predictions_path}")

    images_root = dataset_root / "images"
    labels_root = dataset_root / "labels"
    dataset_root.mkdir(parents=True, exist_ok=True)

    splits_present: set[str] = set()
    missing_images: list[Path] = []
    predictions = list(load_predictions(predictions_path))
    splits = build_split_map(len(predictions), VAL_FRACTION, SPLIT_SEED)
    iterator = tqdm(predictions, desc="Building YOLO dataset", unit="img")

    for prediction, split in zip(iterator, splits):
        splits_present.add(split)

        img_rel = Path(prediction["data"])
        src_img = base_dir / img_rel
        if not src_img.exists():
            missing_images.append(src_img)
            continue

        dest_img = images_root / split / img_rel
        ensure_symlink(src_img, dest_img)

        width, height = get_image_size(src_img)
        detections = collect_boxes(prediction)
        kept_boxes = nms(detections, NMS_IOU_THRESHOLD)
        boxes = [to_box(box, width, height) for box in kept_boxes]

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
