import sys
import os

# Add SAM3 to path before importing
script_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(script_dir)
sam3_path = os.path.join(project_root, 'sam3')
if os.path.exists(sam3_path):
    sys.path.insert(0, sam3_path)

import json
import cv2
import pyzed.sl as sl
import numpy as np
import argparse
import torch
from PIL import Image

# Try to import SAM3 components
try:
    import sam3
    from sam3 import build_sam3_image_model
    from sam3.model.sam3_image_processor import Sam3Processor
    HAS_SAM3 = True
except ImportError:
    HAS_SAM3 = False

def main():
    parser = argparse.ArgumentParser(description="Run ball detection using ZED SDK or SAM3")
    parser.add_argument("--method", choices=["zed", "sam3", "diff", "sam3_custom_od"], default="zed", help="Detection method to use")
    parser.add_argument("--prompt", type=str, default="tennis ball", help="Text prompt for SAM3 detection")
    parser.add_argument("--conf", type=float, default=0.4, help="Confidence threshold for detection")
    args = parser.parse_args()

    # 1. Setup
    input_svo_path = os.path.join(project_root, "data/zed-recordings/HD1080_SN39440864_12-27-57.svo2")
    output_dir = os.path.join(project_root, "data/detections_export")
    
    if not os.path.exists(input_svo_path):
        print(f"Error: Input file {input_svo_path} not found.")
        sys.exit(1)
        
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)

    # SAM3 Setup
    sam3_processor = None
    sam3_inference_state = None
    
    if args.method == "sam3" or args.method == "sam3_custom_od":
        if not HAS_SAM3:
            print("Error: 'sam3' package not found. Please install it or use 'zed' method.")
            sys.exit(1)
            
        print(f"Initializing SAM3 with prompt: '{args.prompt}'...")
        # Setup for Ampere GPUs if available (optional but good practice)
        if torch.cuda.is_available():
            torch.backends.cuda.matmul.allow_tf32 = True
            torch.backends.cudnn.allow_tf32 = True
            
        # Build Model
        # Assuming sam3 is installed in editable mode or we can find assets relative to it
        sam3_root = os.path.join(os.path.dirname(sam3.__file__), "..")
        bpe_path = os.path.join(sam3_root, "assets", "bpe_simple_vocab_16e6.txt.gz")
        
        # If bpe doesn't exist at computed path, try default or let it fail/warn
        if not os.path.exists(bpe_path):
             # Fallback or standard location check if needed
             print(f"Warning: BPE vocab not found at {bpe_path}, trying default loading...")
             bpe_path = None 

        model = build_sam3_image_model(bpe_path=bpe_path)
        
        # Create Processor
        sam3_processor = Sam3Processor(model, confidence_threshold=args.conf)
        print("SAM3 Initialized.")

    # 2. Initialize Camera
    print("Initializing Camera...")
    zed = sl.Camera()
    init_params = sl.InitParameters()
    init_params.set_from_svo_file(input_svo_path)
    init_params.svo_real_time_mode = False 
    init_params.coordinate_units = sl.UNIT.METER
    init_params.depth_mode = sl.DEPTH_MODE.NEURAL_PLUS
    
    err = zed.open(init_params)
    if err != sl.ERROR_CODE.SUCCESS:
        print(f"Error opening SVO file: {err}")
        sys.exit(1)

    # 3. Enable Object Detection (Only for ZED method or for Positional Tracking if needed)
    # Even if using SAM3, we might want positional tracking for 3D transformation if we were doing SLAM,
    # but for simple "relative to camera" 3D from Depth, we just need depth.
    
    obj_runtime_param = sl.ObjectDetectionRuntimeParameters()
    
    if args.method == "zed":
        print("Enabling ZED Object Detection (ACCURATE)...")
        obj_param = sl.ObjectDetectionParameters()
        obj_param.enable_tracking = True
        obj_param.enable_segmentation = False
        obj_param.detection_model = sl.OBJECT_DETECTION_MODEL.MULTI_CLASS_BOX_ACCURATE
        
        # Enable Positional Tracking (Required for Object Tracking)
        if obj_param.enable_tracking:
            pos_tracking_param = sl.PositionalTrackingParameters()
            zed.enable_positional_tracking(pos_tracking_param)

        err = zed.enable_object_detection(obj_param)
        if err != sl.ERROR_CODE.SUCCESS:
            print(f"Error enabling object detection: {err}")
            zed.close()
            sys.exit(1)
            
        obj_runtime_param.detection_confidence_threshold = int(args.conf * 100) # ZED uses 0-100

    elif args.method == "sam3_custom_od":
        print("Enabling Custom Object Detection (SAM3)...")
        obj_param = sl.ObjectDetectionParameters()
        obj_param.enable_tracking = True
        obj_param.enable_segmentation = False
        obj_param.detection_model = sl.OBJECT_DETECTION_MODEL.CUSTOM_BOX_OBJECTS
        
        # Enable Positional Tracking (Required for Object Tracking)
        if obj_param.enable_tracking:
            pos_tracking_param = sl.PositionalTrackingParameters()
            zed.enable_positional_tracking(pos_tracking_param)

        err = zed.enable_object_detection(obj_param)
        if err != sl.ERROR_CODE.SUCCESS:
            print(f"Error enabling object detection: {err}")
            zed.close()
            sys.exit(1)
            
        obj_runtime_param.detection_confidence_threshold = 20 # Low threshold for custom, we filter before ingest

    # Runtime parameters
    runtime_params = sl.RuntimeParameters()

    # Prepare for loop
    image_left = sl.Mat()
    depth_map = sl.Mat()
    objects = sl.Objects()
    
    # Video Writer
    video_writer = None
    video_output_path = os.path.join(output_dir, f"detection_video_{args.method}.mp4")
    fourcc = cv2.VideoWriter_fourcc(*'mp4v') 

    nb_frames = zed.get_svo_number_of_frames()
    print(f"Processing {nb_frames} frames with {args.method}...")

    frame_idx = 0
    prev_gray = None
    
    cam_params = zed.get_camera_information().camera_configuration.calibration_parameters.left_cam
    
    # For sam3_custom_od: track detections across frames to maintain consistent IDs
    prev_detections = []  # List of (cx, cy, unique_id)

    while True:
        if zed.grab(runtime_params) == sl.ERROR_CODE.SUCCESS:
            # Retrieve image and depth
            zed.retrieve_image(image_left, sl.VIEW.LEFT)
            zed.retrieve_measure(depth_map, sl.MEASURE.DEPTH) # Float depth map
            
            frame_cv = image_left.get_data()
            
            # Remove alpha channel
            if frame_cv.shape[2] == 4:
                frame_cv = cv2.cvtColor(frame_cv, cv2.COLOR_BGRA2BGR)
                
            # Initialize video writer
            if video_writer is None:
                h, w = frame_cv.shape[:2]
                fps = zed.get_camera_information().camera_configuration.fps
                if fps == 0: fps = 30
                video_writer = cv2.VideoWriter(video_output_path, fourcc, fps, (w, h))

            detected_balls = []

            if args.method == "zed":
                zed.retrieve_objects(objects, obj_runtime_param)
                if objects.is_new:
                    for obj in objects.object_list:
                        # Check for Sport/Ball
                        is_relevant = (obj.label == sl.OBJECT_CLASS.SPORT)
                        
                        det_dict = {
                            "id": obj.id,
                            "label": str(obj.label),
                            "confidence": obj.confidence,
                            "position": [obj.position[0], obj.position[1], obj.position[2]], # x,y,z
                            "bbox_2d": [[pt[0], pt[1]] for pt in obj.bounding_box_2d]
                        }
                        
                        color = (0, 255, 0) if is_relevant else (255, 0, 0)
                        if is_relevant: # Only save relevant ones if filtering strictly? Or save all? 
                            # Original script logic was slightly loose, let's be specific for "tennis ball" request context
                            # But ZED only gives "SPORT", so we save all SPORT.
                            detected_balls.append(det_dict)

                        # Draw
                        top_left = (int(obj.bounding_box_2d[0][0]), int(obj.bounding_box_2d[0][1]))
                        bottom_right = (int(obj.bounding_box_2d[2][0]), int(obj.bounding_box_2d[2][1]))
                        cv2.rectangle(frame_cv, top_left, bottom_right, color, 2)
                        cv2.putText(frame_cv, f"{str(obj.label).split('.')[-1]}", (top_left[0], top_left[1]-10), 
                                    cv2.FONT_HERSHEY_SIMPLEX, 0.5, color, 2)

            elif args.method == "sam3":
                # Convert to RGB for SAM3 (PIL)
                frame_rgb = cv2.cvtColor(frame_cv, cv2.COLOR_BGR2RGB)
                pil_image = Image.fromarray(frame_rgb)
                
                # Set image
                sam3_inference_state = sam3_processor.set_image(pil_image)
                
                # Reset prompts and Set text prompt
                sam3_processor.reset_all_prompts(sam3_inference_state)
                sam3_inference_state = sam3_processor.set_text_prompt(state=sam3_inference_state, prompt=args.prompt)
                
                # Get results from state
                boxes = sam3_inference_state["boxes"] # [N, 4] in xyxy format
                scores = sam3_inference_state["scores"] # [N]
                
                # Process detections
                boxes_cpu = boxes.cpu().numpy()
                scores_cpu = scores.cpu().numpy()
                
                # Get depth data for 3D calculation
                depth_data = depth_map.get_data()
                
                for i, box in enumerate(boxes_cpu):
                    x1, y1, x2, y2 = map(int, box)
                    score = float(scores_cpu[i])
                    
                    # Calculate center for 3D position
                    cx = (x1 + x2) // 2
                    cy = (y1 + y2) // 2
                    
                    # Clamp to image bounds
                    h, w = depth_data.shape
                    cx = max(0, min(cx, w-1))
                    cy = max(0, min(cy, h-1))
                    
                    # Get depth at center
                    z = depth_data[cy, cx]
                    
                    # Reproject to 3D (using pinhole model approximation or ZED helper if available without SDK objects)
                    # x = (u - cx) * z / fx
                    # y = (v - cy) * z / fy
                    fx = cam_params.fx
                    fy = cam_params.fy
                    cx_cam = cam_params.cx
                    cy_cam = cam_params.cy
                    
                    x_world = (cx - cx_cam) * z / fx
                    y_world = (cy - cy_cam) * z / fy
                    
                    # If depth is valid/finite
                    if np.isfinite(z) and not np.isnan(z):
                         pos = [float(x_world), float(y_world), float(z)]
                    else:
                         pos = [0.0, 0.0, 0.0]

                    det_dict = {
                        "label": args.prompt,
                        "confidence": score * 100,
                        "position": pos,
                        "bbox_2d": [[x1, y1], [x2, y1], [x2, y2], [x1, y2]] # consistent format
                    }
                    detected_balls.append(det_dict)
                    
                    # Draw
                    cv2.rectangle(frame_cv, (x1, y1), (x2, y2), (0, 255, 0), 2)
                    cv2.putText(frame_cv, f"{args.prompt} {int(score*100)}%", (x1, y1-10), 
                                cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 2)

            elif args.method == "sam3_custom_od":
                # Convert to RGB for SAM3 (PIL)
                frame_rgb = cv2.cvtColor(frame_cv, cv2.COLOR_BGR2RGB)
                pil_image = Image.fromarray(frame_rgb)
                
                # Set image
                sam3_inference_state = sam3_processor.set_image(pil_image)
                
                # Reset prompts and Set text prompt
                sam3_processor.reset_all_prompts(sam3_inference_state)
                sam3_inference_state = sam3_processor.set_text_prompt(state=sam3_inference_state, prompt=args.prompt)
                
                # Get results from state
                boxes = sam3_inference_state["boxes"] # [N, 4] in xyxy format
                scores = sam3_inference_state["scores"] # [N]
                
                boxes_cpu = boxes.cpu().numpy()
                scores_cpu = scores.cpu().numpy()
                
                # Track detections across frames for consistent IDs
                current_detections = []
                objects_in = []
                
                for i, box in enumerate(boxes_cpu):
                    score = float(scores_cpu[i])
                    if score < args.conf:
                        continue
                        
                    x1, y1, x2, y2 = map(int, box)
                    cx = (x1 + x2) // 2
                    cy = (y1 + y2) // 2
                    
                    # Find matching previous detection (simple nearest neighbor within threshold)
                    matched_id = None
                    min_dist = float('inf')
                    match_threshold = 100  # pixels
                    
                    for prev_cx, prev_cy, prev_id in prev_detections:
                        dist = np.sqrt((cx - prev_cx)**2 + (cy - prev_cy)**2)
                        if dist < min_dist and dist < match_threshold:
                            min_dist = dist
                            matched_id = prev_id
                    
                    # Use matched ID or generate new one
                    if matched_id is None:
                        unique_id = sl.generate_unique_id()
                    else:
                        unique_id = matched_id
                    
                    current_detections.append((cx, cy, unique_id))
                    
                    tmp = sl.CustomBoxObjectData()
                    tmp.unique_object_id = unique_id
                    tmp.probability = score
                    tmp.label = 1
                    
                    # Bounding box 2D: 4 corners [ (x,y), (x,y), (x,y), (x,y) ]
                    # Order: Top-Left, Top-Right, Bottom-Right, Bottom-Left
                    tmp.bounding_box_2d = np.array([
                        [x1, y1], 
                        [x2, y1], 
                        [x2, y2], 
                        [x1, y2]
                    ], dtype=np.uint32)
                    
                    tmp.is_grounded = False 
                    objects_in.append(tmp)
                
                # Update prev_detections for next frame
                prev_detections = current_detections
                
                # Ingest to ZED SDK
                zed.ingest_custom_box_objects(objects_in)
                
                # Retrieve from ZED SDK (with tracking and depth)
                zed.retrieve_objects(objects, obj_runtime_param)
                
                for obj in objects.object_list:
                    # Collect Data
                    pos = obj.position
                    
                    # Handle NaN positions gracefully
                    if np.isnan(pos[0]) or np.isnan(pos[1]) or np.isnan(pos[2]):
                        dist = 0.0
                        pos_valid = False
                    else:
                        dist = np.sqrt(pos[0]**2 + pos[1]**2 + pos[2]**2)
                        pos_valid = True
                    
                    det_dict = {
                        "id": obj.id,
                        "label": args.prompt,
                        "confidence": obj.confidence,
                        "position": [pos[0], pos[1], pos[2]],
                        "bbox_2d": [[pt[0], pt[1]] for pt in obj.bounding_box_2d]
                    }
                    detected_balls.append(det_dict)
                    
                    # Visualization
                    # Box
                    top_left = (int(obj.bounding_box_2d[0][0]), int(obj.bounding_box_2d[0][1]))
                    bottom_right = (int(obj.bounding_box_2d[2][0]), int(obj.bounding_box_2d[2][1]))
                    
                    # Tracking color (Green = OK, Red = Searching/Terminated)
                    color = (0, 255, 0) if obj.tracking_state == sl.OBJECT_TRACKING_STATE.OK else (0, 0, 255)
                    
                    cv2.rectangle(frame_cv, top_left, bottom_right, color, 2)
                    
                    # Label: ID and Depth only (show "N/A" if depth invalid)
                    if pos_valid:
                        label_text = f"ID:{obj.id} {dist:.2f}m"
                    else:
                        label_text = f"ID:{obj.id} N/A"
                    cv2.putText(frame_cv, label_text, (top_left[0], top_left[1]-10), 
                                cv2.FONT_HERSHEY_SIMPLEX, 0.5, color, 2)

            elif args.method == "diff":
                # Circle Detection on both RGB and Depth, then union with depth filtering
                depth_data = depth_map.get_data()
                
                detected_circles = []  # Store all circles (cx, cy, cr)
                
                # 1. Circle Detection on RGB image
                gray = cv2.cvtColor(frame_cv, cv2.COLOR_BGR2GRAY)
                gray_blur = cv2.GaussianBlur(gray, (9, 9), 2)
                
                rgb_circles = cv2.HoughCircles(gray_blur, cv2.HOUGH_GRADIENT, dp=1.2, minDist=30,
                                             param1=50, param2=30, minRadius=5, maxRadius=25)
                
                if rgb_circles is not None:
                    rgb_circles = np.uint16(np.around(rgb_circles))
                    for c in rgb_circles[0, :]:
                        detected_circles.append((int(c[0]), int(c[1]), int(c[2]), "rgb"))
                
                # 2. Circle Detection on Depth Map
                depth_clipped = np.clip(depth_data, 0.5, 15.0)
                depth_norm = cv2.normalize(depth_clipped, None, 0, 255, cv2.NORM_MINMAX, dtype=cv2.CV_8U)
                
                depth_circles = cv2.HoughCircles(depth_norm, cv2.HOUGH_GRADIENT, dp=1.5, minDist=30,
                                               param1=100, param2=30, minRadius=5, maxRadius=25)
                
                if depth_circles is not None:
                    depth_circles = np.uint16(np.around(depth_circles))
                    for c in depth_circles[0, :]:
                        detected_circles.append((int(c[0]), int(c[1]), int(c[2]), "depth"))
                
                # 3. Process all detected circles (union) with depth filtering
                h_img, w_img = depth_data.shape
                
                for cx, cy, cr, source in detected_circles:
                    # Clamp to image bounds
                    cx_clamped = max(0, min(cx, w_img-1))
                    cy_clamped = max(0, min(cy, h_img-1))
                    
                    # Get depth at center
                    z = depth_data[cy_clamped, cx_clamped]
                    
                    # Filter: only objects with valid depth <= 15m
                    if not (np.isfinite(z) and not np.isnan(z) and z <= 15.0):
                        continue
                    
                    # Calculate 3D Position
                    fx = cam_params.fx
                    fy = cam_params.fy
                    cx_cam = cam_params.cx
                    cy_cam = cam_params.cy
                    
                    x_world = (cx_clamped - cx_cam) * z / fx
                    y_world = (cy_clamped - cy_cam) * z / fy
                    
                    pos = [float(x_world), float(y_world), float(z)]
                    
                    det_dict = {
                        "label": f"ball_{source}",
                        "confidence": 100.0,
                        "position": pos,
                        "bbox_2d": [[cx-cr, cy-cr], [cx+cr, cy-cr], 
                                    [cx+cr, cy+cr], [cx-cr, cy+cr]]
                    }
                    detected_balls.append(det_dict)
                    
                    # Draw detection with different colors based on source
                    color = (0, 255, 255) if source == "rgb" else (255, 0, 255)  # Cyan for RGB, Magenta for depth
                    cv2.circle(frame_cv, (cx, cy), cr, color, 2)
                    cv2.putText(frame_cv, f"{source} {z:.1f}m", (cx-10, cy-10),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.4, color, 1)

            # Export
            # 1. Save Image
            cv2.imwrite(os.path.join(output_dir, f"frame_{frame_idx:06d}.jpg"), frame_cv)
            
            # 2. Save JSON
            with open(os.path.join(output_dir, f"frame_{frame_idx:06d}.json"), 'w') as f:
                json.dump(detected_balls, f, indent=4)
            
            # 3. Write to Video
            if video_writer:
                video_writer.write(frame_cv)

            print(f"Processed frame {frame_idx}/{nb_frames}", end='\r')
            frame_idx += 1
            
        else:
            break

    # Cleanup
    if video_writer:
        video_writer.release()
        
    if args.method == "zed":
        zed.disable_object_detection()
    zed.close()
    print(f"\nDone! Processed {frame_idx} frames.")
    print(f"Results saved to {output_dir}")

if __name__ == "__main__":
    main()
