#!/usr/bin/env python3
"""
Multi-camera ball detection and 3D triangulation.
Supports SAM3 and YOLO detectors.
"""

import os
import sys
import json
import argparse
import numpy as np
import cv2
import torch
from PIL import Image
from typing import Dict, List, Optional, Tuple

# Add project root and sam3 to path
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
sam3_path = os.path.join(project_root, 'sam3')
if os.path.exists(sam3_path):
    sys.path.insert(0, sam3_path)

try:
    from sam3 import build_sam3_image_model
    from sam3.model.sam3_image_processor import Sam3Processor
    HAS_SAM3 = True
except ImportError:
    HAS_SAM3 = False

try:
    from ultralytics import YOLO
    HAS_YOLO = True
except ImportError:
    HAS_YOLO = False

# Camera names
CAMERA_NAMES = ["front", "back", "left", "right"]

# ============================================================================
# Detectors
# ============================================================================

class BaseDetector:
    def detect(self, image_np: np.ndarray) -> List[dict]:
        raise NotImplementedError

class SAM3Detector(BaseDetector):
    def __init__(self, confidence_threshold=0.35):
        if not HAS_SAM3:
            raise ImportError("SAM3 not found. Ensure it's in the project root.")
        
        print("Initializing SAM3 detector...")
        if torch.cuda.is_available():
            torch.backends.cuda.matmul.allow_tf32 = True
            torch.backends.cudnn.allow_tf32 = True
        
        sam3_root = os.path.join(project_root, "sam3")
        bpe_path = os.path.join(sam3_root, "assets", "bpe_simple_vocab_16e6.txt.gz")
        if not os.path.exists(bpe_path):
            bpe_path = None
        
        model = build_sam3_image_model(bpe_path=bpe_path)
        self.processor = Sam3Processor(model, confidence_threshold=confidence_threshold)
        self.prompt = "small tennis ball . tennis ball"
        self.state = {}
        print("SAM3 initialized.")

    def detect(self, image_np: np.ndarray) -> List[dict]:
        # SAM3 expects RGB PIL Image
        image_rgb = cv2.cvtColor(image_np, cv2.COLOR_BGR2RGB)
        pil_img = Image.fromarray(image_rgb)
        
        self.state = self.processor.set_image(pil_img, state=self.state)
        self.processor.reset_all_prompts(self.state)
        self.state = self.processor.set_text_prompt(self.prompt, state=self.state)
        
        boxes = self.state.get("boxes", [])
        scores = self.state.get("scores", [])
        
        if torch.is_tensor(boxes): boxes = boxes.cpu().numpy()
        if torch.is_tensor(scores): scores = scores.cpu().numpy()
        
        detections = []
        for i in range(len(boxes)):
            box = boxes[i]
            score = float(scores[i])
            cx = (box[0] + box[2]) / 2.0
            cy = (box[1] + box[3]) / 2.0
            area = (box[2] - box[0]) * (box[3] - box[1])
            
            detections.append({
                "center": (cx, cy),
                "bbox": box.tolist(),
                "confidence": score,
                "area": area
            })
        return detections

class YOLODetector(BaseDetector):
    def __init__(self, confidence_threshold=0.5):
        if not HAS_YOLO:
            raise ImportError("ultralytics YOLO not found.")
        
        print("Initializing YOLO detector (yolov8n.pt)...")
        self.model = YOLO("yolov8n.pt")
        self.conf = confidence_threshold
        self.ball_class_id = 32 # COCO "sports ball"
        print("YOLO initialized.")

    def detect(self, image_np: np.ndarray) -> List[dict]:
        results = self.model(image_np, conf=self.conf, classes=[self.ball_class_id], verbose=False)
        detections = []
        for result in results:
            for box in result.boxes:
                xyxy = box.xyxy[0].cpu().numpy()
                score = float(box.conf[0])
                cx = (xyxy[0] + xyxy[2]) / 2.0
                cy = (xyxy[1] + xyxy[3]) / 2.0
                area = (xyxy[2] - xyxy[0]) * (xyxy[3] - xyxy[1])
                
                detections.append({
                    "center": (cx, cy),
                    "bbox": xyxy.tolist(),
                    "confidence": score,
                    "area": area
                })
        return detections

# ============================================================================
# Triangulation Logic
# ============================================================================

def triangulate(u_l, v_l, u_r, calib):
    disparity = u_l - u_r
    if disparity <= 0:
        return None, None
    
    Z = (calib["fx"] * calib["baseline"]) / disparity
    X = (u_l - calib["cx"]) * Z / calib["fx"]
    Y = (v_l - calib["cy"]) * Z / calib["fy"]
    
    return [X, Y, Z], disparity

def match_stereo_detections(left_detections, right_detections, y_tolerance=5.0, area_tolerance=0.3):
    matches = []
    used_right = set()
    
    for left_det in left_detections:
        left_cx, left_cy = left_det["center"]
        left_area = left_det["area"]
        
        best_match = None
        best_score = float('inf')
        
        for i, right_det in enumerate(right_detections):
            if i in used_right:
                continue
            
            right_cx, right_cy = right_det["center"]
            right_area = right_det["area"]
            
            y_diff = abs(left_cy - right_cy)
            if y_diff > y_tolerance:
                continue
            
            if left_cx <= right_cx:
                continue
            
            area_ratio = min(left_area, right_area) / max(left_area, right_area)
            if area_ratio < (1 - area_tolerance):
                continue
            
            score = y_diff + (1 - area_ratio) * 10
            if score < best_score:
                best_score = score
                best_match = (i, right_det)
        
        if best_match is not None:
            used_right.add(best_match[0])
            matches.append((left_det, best_match[1]))
    
    return matches

# ============================================================================
# Main Processing
# ============================================================================

def main():
    parser = argparse.ArgumentParser(description="Multi-camera ball detection and triangulation.")
    parser.add_argument("input_dir", help="Directory containing exported frames and calibration.json")
    parser.add_argument("--detector", choices=["sam3", "yolo"], default="sam3", help="Detector to use")
    parser.add_argument("--conf", type=float, default=None, help="Confidence threshold")
    args = parser.parse_args()

    # Load calibration
    calib_path = os.path.join(args.input_dir, "calibration.json")
    if not os.path.exists(calib_path):
        print(f"Error: calibration.json not found in {args.input_dir}")
        sys.exit(1)
    with open(calib_path, 'r') as f:
        calibration = json.load(f)

    # Initialize detector
    if args.detector == "sam3":
        conf = args.conf or 0.35
        detector = SAM3Detector(confidence_threshold=conf)
    else:
        conf = args.conf or 0.5
        detector = YOLODetector(confidence_threshold=conf)

    # Get timestamp folders
    folders = sorted([f for f in os.listdir(args.input_dir) if f.isdigit()], key=lambda x: int(x))
    print(f"Found {len(folders)} frames to process.")

    # Results container: cam -> list of {timestamp, detections}
    results = {cam: [] for cam in CAMERA_NAMES}

    for i, folder in enumerate(folders):
        folder_path = os.path.join(args.input_dir, folder)
        timestamp = int(folder)
        
        for cam in CAMERA_NAMES:
            left_path = os.path.join(folder_path, f"{cam}.jpg")
            right_path = os.path.join(folder_path, f"{cam}_right.jpg")
            
            if not os.path.exists(left_path) or not os.path.exists(right_path):
                continue
            
            left_img = cv2.imread(left_path)
            right_img = cv2.imread(right_path)
            
            left_dets = detector.detect(left_img)
            right_dets = detector.detect(right_img)
            
            matches = match_stereo_detections(left_dets, right_dets)
            
            triangulated = []
            for left_det, right_det in matches:
                l_cx, l_cy = left_det["center"]
                r_cx, r_cy = right_det["center"]
                xyz, disparity = triangulate(l_cx, l_cy, r_cx, calibration[cam])
                
                if xyz:
                    triangulated.append({
                        "left_center": [l_cx, l_cy],
                        "right_center": [r_cx, r_cy],
                        "left_bbox": left_det["bbox"],
                        "right_bbox": right_det["bbox"],
                        "disparity": disparity,
                        "triangulated_xyz": xyz,
                        "confidence": (left_det["confidence"] + right_det["confidence"]) / 2.0
                    })
            
            results[cam].append({
                "timestamp": timestamp,
                "detections": triangulated
            })
            
        if (i + 1) % 10 == 0:
            print(f"Processed {i + 1}/{len(folders)} frames...")

    # Write results per camera
    for cam in CAMERA_NAMES:
        cam_dir = os.path.join(args.input_dir, cam)
        os.makedirs(cam_dir, exist_ok=True)
        out_path = os.path.join(cam_dir, "triangulated_detections.json")
        with open(out_path, 'w') as f:
            json.dump(results[cam], f, indent=2)
        print(f"Wrote {len(results[cam])} frames of detections for {cam} to {out_path}")

if __name__ == "__main__":
    main()
