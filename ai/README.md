# AI and Perception

Tensa's AI work centered on making a moving tennis robot understand a real court quickly enough to aim, move, and react.

## Problem Shape

The robot needed perception that worked from moving cameras, on real courts, with
small fast tennis balls, glare, motion blur, fences, players, and changing light.
The useful output was not just a detection box. The rest of the robot needed
court-space positions: where the robot is, where the ball is, where players are,
and where a shot or movement command should go.

## Included Work

- `balltracking/`: online ZED SVO inference, stereo matching, 3D triangulation, trajectory tracking, bounce refinement, and Rerun visualization.
- `../docs/ai/ball-tracking-methodology.md`: detailed notes on stereo triangulation, filtering, tracking, and physics-informed refinement.
- `../docs/ai/localization-methodology.md`: notes on machine localization with camera/court geometry.
- `training/`: training/evaluation code skeletons for ball detection and court keypoint detection, with heavyweight weights and private datasets removed.

## Main Threads

### Ball Tracking

The ball tracker detects candidates in stereo image pairs, matches left/right
detections with geometric constraints, triangulates 3D points from disparity,
and associates those points into trajectories. A physics-aware refinement step
uses gravity and bounce constraints to clean noisy detections.

### Court Localization

The localization code maps camera observations into a tennis-court coordinate
frame. Known court geometry, camera intrinsics, camera extrinsics, and solved
machine pose let downstream systems reason in meters instead of pixels.

### Training and Evaluation

Training folders preserve the detector experiments and evaluators used while
iterating on ball and court-keypoint perception. Private images, labels, model
weights, and generated runs are omitted.

## Model Assets

Large model weights and datasets are intentionally not committed. The code expects you to provide local weights such as `yolo_ball_detector.pt` or a TensorRT `.engine` built for the target GPU.

```bash
cd ai/balltracking
python online_runner.py --svo path/to/recording.svo --weights path/to/yolo_ball_detector.pt
```
