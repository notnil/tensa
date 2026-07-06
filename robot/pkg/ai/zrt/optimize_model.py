import onnx
import onnx_graphsurgeon as gs
import numpy as np
import argparse

def optimize_model(input_model_path, output_model_path):
    print(f"Loading model from {input_model_path}...")
    graph = gs.import_onnx(onnx.load(input_model_path))

    # ==================================================================================
    # PART 1: Input Optimization (Zero-Copy ZED Support)
    # ==================================================================================
    
    # Find the original input
    original_input = graph.inputs[0]
    print(f"Original input: {original_input.name}, shape: {original_input.shape}, dtype: {original_input.dtype}")
    
    # Determine H and W
    h_dim = "H"
    w_dim = "W"
    if len(original_input.shape) == 4:
        # If original is [1, 3, H, W], we try to preserve H/W if they are fixed integers
        if isinstance(original_input.shape[2], int):
            h_dim = original_input.shape[2]
        if isinstance(original_input.shape[3], int):
            w_dim = original_input.shape[3]

    # Create new input for ZED: [Batch, Height, Width, Channels=4] (BGRA), uint8
    new_input = gs.Variable(name="zed_input_bgra", dtype=np.uint8, shape=[1, h_dim, w_dim, 4])
    
    # 1. Cast to Float (TensorRT requirement for Gather)
    cast_to_float = gs.Variable(name="bgra_float", dtype=np.float32)
    cast_node_1 = gs.Node(op="Cast", inputs=[new_input], outputs=[cast_to_float], attrs={"to": onnx.TensorProto.FLOAT})
    graph.nodes.append(cast_node_1)
    
    # 2. Gather to swap BGR->RGB and drop Alpha
    indices = gs.Constant(name="indices_rgb", values=np.array([2, 1, 0], dtype=np.int64))
    gather_out = gs.Variable(name="rgb_float", dtype=np.float32)
    gather_node = gs.Node(op="Gather", inputs=[cast_to_float, indices], outputs=[gather_out], attrs={"axis": 3})
    graph.nodes.append(gather_node)
    
    # 3. Div (Normalize)
    div_const = gs.Constant(name="div_255", values=np.array([255.0], dtype=np.float32))
    normalized = gs.Variable(name="rgb_normalized", dtype=np.float32)
    div_node = gs.Node(op="Div", inputs=[gather_out, div_const], outputs=[normalized])
    graph.nodes.append(div_node)
    
    # 4. Transpose HWC -> CHW
    transpose_out = gs.Variable(name="normalized_input", dtype=np.float32)
    transpose_node = gs.Node(op="Transpose", inputs=[normalized], outputs=[transpose_out], attrs={"perm": [0, 3, 1, 2]})
    graph.nodes.append(transpose_node)
    
    # Replace the original input in the graph
    for node in graph.nodes:
        for i, inp in enumerate(node.inputs):
            if inp.name == original_input.name:
                node.inputs[i] = transpose_out
    
    # Update graph inputs
    graph.inputs = [new_input]

    # Cleanup before NMS part
    graph.cleanup().toposort()

    # ==================================================================================
    # PART 2: Output Optimization (EfficientNMS_TRT)
    # ==================================================================================
    
    # Locate the model output
    # YOLOv8 output is typically [1, 84, 8400] (4 coords + 80 classes)
    original_output = graph.outputs[0]
    print(f"Original output: {original_output.name}, shape: {original_output.shape}")
    
    # Disconnect original output
    graph.outputs = [] 
    
    # 1. Transpose [1, 84, N] -> [1, N, 84]
    # We need channels last for splitting
    nms_input_transposed = gs.Variable(name="nms_input_transposed", dtype=np.float32)
    transpose_nms = gs.Node(op="Transpose", inputs=[original_output], outputs=[nms_input_transposed], attrs={"perm": [0, 2, 1]})
    graph.nodes.append(transpose_nms)
    
    # 2. Slice into Boxes and Scores
    # Boxes: [0, 1, 2, 3], Scores: [4 ... 83]
    # We use 'Split' or 'Slice'. Split is easier if lengths are known, but Slice is safer.
    # Let's use Slice.
    
    # Boxes: start=0, end=4, axis=2
    boxes = gs.Variable(name="nms_boxes", dtype=np.float32)
    # For Slice opset >= 10, we need inputs: data, starts, ends, axes, steps
    # To keep it simple with GraphSurgeon, we can use Constant inputs.
    
    # Slice Boxes
    starts_box = gs.Constant(name="slice_box_starts", values=np.array([0], dtype=np.int64))
    ends_box = gs.Constant(name="slice_box_ends", values=np.array([4], dtype=np.int64))
    axes_box = gs.Constant(name="slice_box_axes", values=np.array([2], dtype=np.int64))
    
    slice_box_node = gs.Node(op="Slice", inputs=[nms_input_transposed, starts_box, ends_box, axes_box], outputs=[boxes])
    graph.nodes.append(slice_box_node)
    
    # Slice Scores
    scores = gs.Variable(name="nms_scores", dtype=np.float32)
    starts_score = gs.Constant(name="slice_score_starts", values=np.array([4], dtype=np.int64))
    ends_score = gs.Constant(name="slice_score_ends", values=np.array([2147483647], dtype=np.int64)) # INT_MAX
    axes_score = gs.Constant(name="slice_score_axes", values=np.array([2], dtype=np.int64))
    
    slice_score_node = gs.Node(op="Slice", inputs=[nms_input_transposed, starts_score, ends_score, axes_score], outputs=[scores])
    graph.nodes.append(slice_score_node)
    
    # 3. EfficientNMS_TRT Plugin
    # Outputs: num_detections, detection_boxes, detection_scores, detection_classes
    num_detections = gs.Variable(name="num_detections", dtype=np.int32)
    det_boxes = gs.Variable(name="det_boxes", dtype=np.float32)
    det_scores = gs.Variable(name="det_scores", dtype=np.float32)
    det_classes = gs.Variable(name="det_classes", dtype=np.int32) # Plugin output is usually Int32 for classes
    
    # Configuration
    MAX_BOXES = 100
    nms_node = gs.Node(
        op="EfficientNMS_TRT",
        inputs=[boxes, scores],
        outputs=[num_detections, det_boxes, det_scores, det_classes],
        attrs={
            "plugin_version": "1",
            "background_class": -1,
            "max_output_boxes": MAX_BOXES,
            "score_threshold": 0.25,
            "iou_threshold": 0.45,
            "score_activation": 0, # 0: None (Input is prob), 1: Sigmoid. YOLOv8 export is usually prob? Safe to assume prob if no Sigmoid in graph?
                                   # Actually YOLOv8 usually exports with Sigmoid embedded or Softmax. 
                                   # If unsure, 0 is safest for standard ONNX exports which include activation.
            "box_coding": 1,       # 0: Corners (x1,y1,x2,y2), 1: Center (x,y,w,h). YOLOv8 output is x,y,w,h.
        }
    )
    graph.nodes.append(nms_node)
    
    # 4. Reshape/Cast for Concatenation
    # Bridge expects [N, 6] -> [x, y, w, h, conf, class]
    # EfficientNMS outputs:
    # det_boxes: [Batch, MaxBoxes, 4]
    # det_scores: [Batch, MaxBoxes]
    # det_classes: [Batch, MaxBoxes]
    
    # Unsqueeze scores and classes to [Batch, MaxBoxes, 1]
    det_scores_expanded = gs.Variable(name="det_scores_expanded", dtype=np.float32)
    unsqueeze_scores = gs.Node(op="Unsqueeze", inputs=[det_scores, gs.Constant(name="axes_3", values=np.array([2], dtype=np.int64))], outputs=[det_scores_expanded])
    graph.nodes.append(unsqueeze_scores)
    
    det_classes_float = gs.Variable(name="det_classes_float", dtype=np.float32)
    cast_classes = gs.Node(op="Cast", inputs=[det_classes], outputs=[det_classes_float], attrs={"to": onnx.TensorProto.FLOAT})
    graph.nodes.append(cast_classes)
    
    det_classes_expanded = gs.Variable(name="det_classes_expanded", dtype=np.float32)
    unsqueeze_classes = gs.Node(op="Unsqueeze", inputs=[det_classes_float, gs.Constant(name="axes_3_cls", values=np.array([2], dtype=np.int64))], outputs=[det_classes_expanded])
    graph.nodes.append(unsqueeze_classes)
    
    # Concat: [Boxes(4), Scores(1), Classes(1)] -> [Batch, MaxBoxes, 6]
    final_output = gs.Variable(name="final_detections", dtype=np.float32, shape=[1, MAX_BOXES, 6])
    concat_node = gs.Node(op="Concat", inputs=[det_boxes, det_scores_expanded, det_classes_expanded], outputs=[final_output], attrs={"axis": 2})
    graph.nodes.append(concat_node)
    
    graph.outputs = [final_output]
    
    # Final Cleanup
    graph.cleanup().toposort()
    
    # Validate FP16 compatibility
    print("Validating model for FP16 compatibility...")
    problematic_ops = ['ReduceSum', 'ReduceMean', 'Softmax']
    found_problematic = False
    for node in graph.nodes:
        if node.op in problematic_ops:
            print(f"  Note: Found {node.op} operation which may have precision sensitivity in FP16")
            found_problematic = True
    
    if not found_problematic:
        print("  No known FP16 problematic operations found. Model should work well with FP16.")

    print(f"Saving optimized model to {output_model_path}...")
    model = gs.export_onnx(graph)
    model.ir_version = 9 # Force IR version 9
    onnx.save(model, output_model_path)
    print("Done.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Optimize ONNX model for Zero-Copy ZED pipeline")
    parser.add_argument("--input", type=str, required=True, help="Path to input ONNX model")
    parser.add_argument("--output", type=str, required=True, help="Path to output ONNX model")
    args = parser.parse_args()
    
    optimize_model(args.input, args.output)
