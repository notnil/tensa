# Training Code

This directory preserves the training and evaluation scaffolding used for the
ball detector and court keypoint detector.

The private datasets, generated YOLO folders, SAM outputs, training runs, and
model weights are intentionally not included. The files here are mainly useful
as implementation evidence: how labels were converted, how models were trained,
and how detection quality was evaluated.

## Contents

- `ball_detector_single_frame/`: single-frame tennis ball detection training
  with hand-labeled and SAM-assisted dataset paths.
- `ball_detector_multi_frame/`: heatmap-style multi-frame detector experiments.
- `court_keypoint_detector/`: court keypoint detector training and evaluation
  utilities used by localization work.

To run these scripts, recreate the expected local dataset layout under each
experiment directory and install the matching `requirements.txt` for that
experiment.
