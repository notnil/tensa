# Ball Tracking

## Running Online Inference

```bash
python online_runner.py \
  --svo path/to/recording.svo \
  --weights yolo_ball_detector.pt
```

FP16 half-precision is used by default. Pass `--fp32` to disable it.

To cap the run to a specific number of frames (useful for debugging):

```bash
python online_runner.py --svo recording.svo --weights yolo_ball_detector.pt --max-frames 500
```

## TensorRT Model Export

The online runner supports TensorRT-optimized models for faster GPU inference.
Export must be done on the machine with the target GPU, as the resulting engine
is architecture-specific.

### Export

The batch size for the engine must be `2 * --inference-batch-size` (default 4,
so batch=8). The runner batches multiple stereo frame-pairs into a single YOLO
call and pads partial batches to this fixed size.

```bash
yolo export model=yolo_ball_detector.pt format=engine half=True imgsz=1280 batch=8
```

This produces `yolo_ball_detector.engine` in the same directory. The export
takes a few minutes on first run as TensorRT builds and benchmarks kernels for
your specific GPU.

### Usage

Pass the `.engine` file as `--weights`:

```bash
python online_runner.py --svo input.svo --weights yolo_ball_detector.engine
```

FP16 inference is enabled by default. If you exported with `half=True` (recommended),
this matches automatically. Use `--fp32` only if you exported without `half=True`.

### Performance Tuning

The runner prefetches frames in a background thread and batches multiple
frame-pairs into a single YOLO inference call. The key flags:

- `--inference-batch-size N` — max stereo frame-pairs per YOLO call (default 4,
  meaning up to 8 images per batch). The TensorRT engine must be compiled with
  `batch=2*N`.
- `--profile` — prints a wall-clock stage timing summary at the end of the run,
  useful for identifying bottlenecks.

### Notes

- We use a fixed (not dynamic) TensorRT batch size because it allows TensorRT
  to pick kernel implementations and memory layouts optimized for that exact
  shape. When fewer frames are available, the runner pads with zero images to
  fill the batch; the padding results are discarded.
- The engine file is tied to the exact GPU model and TensorRT version it was
  built on. Re-export if you change hardware or update TensorRT.
- The `imgsz` used during export must match the `--imgsz` passed to the runner
  (default: 1280).
- Batch size is baked into the engine. If you change `--inference-batch-size`,
  you must re-export with the corresponding `batch=2*N`.
