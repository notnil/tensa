import argparse
import json
import os
import sys
from pathlib import Path
from typing import Iterable, List
import multiprocessing as mp
from concurrent.futures import ProcessPoolExecutor

import numpy as np
import torch
from PIL import Image, ImageDraw
from torch.utils.data import DataLoader, Dataset
from torchvision.transforms import v2

REPO_ROOT = Path(__file__).resolve().parent
PKG_PARENT = REPO_ROOT / "sam3"
if str(PKG_PARENT) not in sys.path:
    sys.path.insert(0, str(PKG_PARENT))

from tqdm.auto import tqdm

from sam3.model.data_misc import FindStage
from sam3.model_builder import build_sam3_image_model
from sam3.model.sam3_image_processor import Sam3Processor


IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".webp"}


def to_numpy_array(data) -> np.ndarray:
    """Convert tensors/lists to numpy arrays on CPU."""
    if data is None:
        return np.empty((0,), dtype=np.float32)
    if torch.is_tensor(data):
        return data.detach().cpu().numpy()
    return np.asarray(data)


def stack_masks(masks) -> np.ndarray:
    """Normalize masks into a numpy array."""
    if masks is None:
        return np.empty((0, 0, 0), dtype=np.uint8)
    if torch.is_tensor(masks):
        mask_arr = masks.detach().cpu().numpy()
        return mask_arr.astype(np.uint8)
    if isinstance(masks, list):
        collected = []
        for mask in masks:
            if torch.is_tensor(mask):
                mask = mask.detach().cpu().numpy()
            collected.append(np.asarray(mask))
        if not collected:
            return np.empty((0, 0, 0), dtype=np.uint8)
        return np.stack(collected, axis=0).astype(np.uint8)
    return np.asarray(masks).astype(np.uint8)


def mask_to_rle(mask: np.ndarray) -> dict:
    """
    Convert a binary mask to a simple run-length encoding that is JSON friendly.
    The encoding follows the COCO convention of flattening in column-major order.
    """
    if mask.ndim == 3:
        mask = np.squeeze(mask, axis=0)
    if mask.size == 0:
        return {"size": [int(dim) for dim in mask.shape], "counts": []}
    mask = np.asarray(mask, dtype=np.uint8, order="F")
    flat = mask.ravel(order="F")
    diff = np.diff(flat)
    changes = np.flatnonzero(diff) + 1
    boundaries = np.concatenate(([0], changes, [flat.size]))
    counts = np.diff(boundaries).tolist()
    if flat[0] == 1:
        counts = [0] + counts
    return {"size": [int(dim) for dim in mask.shape], "counts": counts}


def serialize_detections(output: dict) -> list:
    boxes = to_numpy_array(output.get("boxes"))
    scores = to_numpy_array(output.get("scores"))
    masks = stack_masks(output.get("masks"))

    detections = []
    num_detections = min(len(scores), len(boxes))
    for idx in range(num_detections):
        detections.append(
            {
                "score": float(scores[idx]),
                "box": [float(coord) for coord in boxes[idx]],
                "mask_rle": mask_to_rle(masks[idx]),
            }
        )

    return detections


def serialize_prompt_output(output: dict, prompt: str) -> dict:
    return {
        "prompt": prompt,
        "detections": serialize_detections(output),
    }


def serialize_image_output(image_path: Path, prompt_outputs: list) -> dict:
    return {
        "data": str(image_path),
        "prompts": prompt_outputs,
    }


def iter_images(data_dir: Path) -> Iterable[Path]:
    for path in sorted(data_dir.rglob("*")):
        if path.is_file() and path.suffix.lower() in IMAGE_EXTENSIONS:
            yield path


def overlay_predictions(image: Image.Image, boxes: np.ndarray, scores: np.ndarray, masks: np.ndarray) -> Image.Image:
    """Create a QC overlay image with boxes."""
    base = image.convert("RGBA")
    draw = ImageDraw.Draw(base)

    for box, score in zip(boxes, scores):
        x0, y0, x1, y1 = box
        draw.rectangle([x0, y0, x1, y1], outline="red", width=2)
        draw.text((x0 + 4, y0 + 4), f"{score:.2f}", fill="red")

    return base.convert("RGB")


def slugify_prompt(prompt: str, max_len: int = 48) -> str:
    cleaned = []
    for char in prompt.strip().lower():
        if char.isalnum():
            cleaned.append(char)
        elif char in {" ", "-", "_"}:
            cleaned.append("_")
    slug = "".join(cleaned).strip("_")
    if not slug:
        slug = "prompt"
    return slug[:max_len]


def serialize_outputs_to_jsonl_line(
    image_path: str, prompts: List[str], outputs: list
) -> str:
    prompt_outputs = [
        serialize_prompt_output(output, prompt)
        for output, prompt in zip(outputs, prompts)
    ]
    output_data = serialize_image_output(Path(image_path), prompt_outputs)
    return json.dumps(output_data)


class ImagePathDataset(Dataset):
    def __init__(self, image_paths: List[Path]):
        self.image_paths = image_paths

    def __len__(self) -> int:
        return len(self.image_paths)

    def __getitem__(self, idx: int):
        path = self.image_paths[idx]
        with Image.open(path) as image:
            rgb_image = image.convert("RGB")
            rgb_image.load()
        return path, rgb_image


def collate_batch(batch):
    paths, images = zip(*batch)
    return list(paths), list(images)


@torch.inference_mode()
def predict_batch_multi_prompt(processor: Sam3Processor, images: List[Image.Image], prompts: List[str]):
    if not isinstance(images, list) or len(images) == 0:
        raise ValueError("Images must be a non-empty list of PIL images")
    if not isinstance(prompts, list) or len(prompts) == 0:
        raise ValueError("Prompts must be a non-empty list of strings")

    original_sizes = [(img.height, img.width) for img in images]
    processed_images = [
        processor.transform(v2.functional.to_image(image).to(processor.device))
        for image in images
    ]
    image_batch = torch.stack(processed_images, dim=0)
    backbone_out = processor.model.backbone.forward_image(image_batch)

    results_by_prompt = []
    target_sizes = torch.tensor(original_sizes, device=processor.device)

    for prompt in prompts:
        prompt_list = [prompt] * len(images)
        text_outputs = processor.model.backbone.forward_text(
            prompt_list, device=processor.device
        )
        backbone_out_with_text = {**backbone_out, **text_outputs}

        geometric_prompt = processor.model._get_dummy_prompt(num_prompts=len(prompt_list))
        find_stage = FindStage(
            img_ids=torch.arange(len(images), device=processor.device, dtype=torch.long),
            text_ids=torch.arange(len(prompt_list), device=processor.device, dtype=torch.long),
            input_boxes=None,
            input_boxes_mask=None,
            input_boxes_label=None,
            input_points=None,
            input_points_mask=None,
        )

        outputs = processor.model.forward_grounding(
            backbone_out=backbone_out_with_text,
            find_input=find_stage,
            geometric_prompt=geometric_prompt,
            find_target=None,
        )

        results = processor._postprocessor(
            outputs,
            target_sizes_boxes=target_sizes,
            target_sizes_masks=target_sizes,
            consistent=False,
        )
        results_by_prompt.append(results)

    return results_by_prompt


def main():
    parser = argparse.ArgumentParser(
        description="Run SAM 3 over every image in a directory and save JSON outputs."
    )
    parser.add_argument(
        "--prompt",
        action="append",
        required=True,
        help="Text prompt to segment. Repeat --prompt for multiple prompts.",
    )
    parser.add_argument(
        "--data-dir",
        type=Path,
        default=Path("data"),
        help="Directory containing input images.",
    )
    parser.add_argument(
        "--overlay-dir",
        type=Path,
        default=Path("output"),
        help="Where overlay outputs will be written.",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("output") / "predictions.jsonl",
        help="Where JSONL outputs will be written.",
    )
    parser.add_argument(
        "--device",
        default="cuda" if torch.cuda.is_available() else "cpu",
        help="Device to run the model on (cuda or cpu).",
    )
    parser.add_argument(
        "--confidence-threshold",
        type=float,
        default=0.5,
        help="Filter predictions below this score.",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=1,
        help="Number of images to process together.",
    )
    parser.add_argument(
        "--num-workers",
        type=int,
        default=max(1, min(8, os.cpu_count() or 1)),
        help="Number of worker processes for image loading.",
    )
    parser.add_argument(
        "--prefetch-factor",
        type=int,
        default=2,
        help="Number of batches prefetched per worker.",
    )
    parser.add_argument(
        "--postprocess-workers",
        type=int,
        default=max(1, min(4, os.cpu_count() or 1)),
        help="Worker processes for JSON serialization (0 to disable).",
    )
    parser.add_argument(
        "--overlay-format",
        default="jpg",
        choices=["png", "jpg", "jpeg"],
        help="Image format for QC overlays.",
    )
    parser.add_argument(
        "--overlays",
        action="store_true",
        help="Write overlay images alongside JSON outputs.",
    )
    args = parser.parse_args()

    if not args.data_dir.exists():
        raise FileNotFoundError(f"Data directory not found: {args.data_dir}")

    args.overlay_dir.mkdir(parents=True, exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)

    model = build_sam3_image_model(device=args.device)
    processor = Sam3Processor(
        model, device=args.device, confidence_threshold=args.confidence_threshold
    )

    image_paths = list(iter_images(args.data_dir))
    if not image_paths:
        raise RuntimeError(f"No images found in {args.data_dir}")

    batch_size = max(1, args.batch_size)
    prompts = args.prompt

    dataset = ImagePathDataset(image_paths)
    loader_kwargs = dict(
        batch_size=batch_size,
        shuffle=False,
        num_workers=max(0, args.num_workers),
        collate_fn=collate_batch,
    )
    if args.num_workers > 0:
        loader_kwargs["prefetch_factor"] = max(1, args.prefetch_factor)
        loader_kwargs["persistent_workers"] = True

    dataloader = DataLoader(dataset, **loader_kwargs)

    mp_context = mp.get_context("spawn")
    postprocess_workers = max(0, args.postprocess_workers)
    executor = None
    if postprocess_workers > 0:
        executor = ProcessPoolExecutor(
            max_workers=postprocess_workers, mp_context=mp_context
        )
    pending = []
    max_pending = max(1, postprocess_workers * 4)

    output_f = args.output.open("w")
    with output_f, tqdm(total=len(dataset), desc="Processing images") as pbar:
        try:
            for batch_paths, rgb_images in dataloader:
                outputs_by_prompt = predict_batch_multi_prompt(
                    processor, rgb_images, prompts
                )

                if args.overlays:
                    for prompt, outputs in zip(prompts, outputs_by_prompt):
                        for image_path, output, rgb_image in zip(
                            batch_paths, outputs, rgb_images
                        ):
                            boxes = to_numpy_array(output.get("boxes"))
                            scores = to_numpy_array(output.get("scores"))
                            masks = stack_masks(output.get("masks"))

                            overlay_image = overlay_predictions(
                                rgb_image, boxes, scores, masks
                            )
                            overlay_suffix = (
                                f"_{slugify_prompt(prompt)}" if len(prompts) > 1 else ""
                            )
                            overlay_path = (
                                args.overlay_dir
                                / f"{image_path.stem}_overlay{overlay_suffix}.{args.overlay_format}"
                            )
                            overlay_image.save(overlay_path)

                for idx, image_path in enumerate(batch_paths):
                    outputs_for_image = [
                        outputs[idx] for outputs in outputs_by_prompt
                    ]
                    if executor is None:
                        line = serialize_outputs_to_jsonl_line(
                            str(image_path), prompts, outputs_for_image
                        )
                        output_f.write(line + "\n")
                    else:
                        pending.append(
                            executor.submit(
                                serialize_outputs_to_jsonl_line,
                                str(image_path),
                                prompts,
                                outputs_for_image,
                            )
                        )
                        if len(pending) >= max_pending:
                            output_f.write(pending.pop(0).result() + "\n")

                pbar.update(len(batch_paths))
        finally:
            if executor is not None:
                for future in pending:
                    output_f.write(future.result() + "\n")
                executor.shutdown()


if __name__ == "__main__":
    main()
