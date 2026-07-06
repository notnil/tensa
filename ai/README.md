# AI and Perception

Tensa's AI work centered on making a moving tennis robot understand a real court quickly enough to aim, move, and react.

## Included Work

- `balltracking/`: online ZED SVO inference, stereo matching, 3D triangulation, trajectory tracking, bounce refinement, and Rerun visualization.
- `balltracking/logan_scripts/BALL_TRACKING_METHODOLOGY.md`: detailed notes on stereo triangulation, filtering, tracking, and physics-informed refinement.
- `balltracking/logan_scripts/LOCALIZATION_METHODOLOGY.md`: notes on machine localization with camera/court geometry.
- `training/`: training/evaluation code skeletons for ball detection and court keypoint detection, with heavyweight weights and private datasets removed.

## Model Assets

Large model weights and datasets are intentionally not committed. The code expects you to provide local weights such as `yolo_ball_detector.pt` or a TensorRT `.engine` built for the target GPU.

```bash
cd ai/balltracking
python online_runner.py --svo path/to/recording.svo --weights path/to/yolo_ball_detector.pt
```
