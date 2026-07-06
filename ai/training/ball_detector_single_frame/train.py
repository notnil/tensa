from ultralytics import YOLO

model = YOLO("yolo26m.pt")

results = model.train(
    # resume=True,
    data="./sam3_output/yolo_dataset.yaml",
    epochs=100,

    # Target image size for training. Images are resized to squares with sides equal to the specified value (if rect=False), preserving aspect ratio for YOLO models but not RT-DETR. Affects model accuracy and computational complexity.
    imgsz=1280,

    # Treats all classes in multi-class datasets as a single class during training. Useful for binary classification tasks or when focusing on object presence rather than classification.
    single_cls=True,

    # Enables multi-scale training by increasing/decreasing imgsz by up to a factor of 0.5 during training. Trains the model to be more accurate with multiple imgsz during inference.
    multi_scale=True,

    # No need for determinism. Speed please!
    deterministic=False,

    # Inspection plots
    plots=True,
)

print(results)