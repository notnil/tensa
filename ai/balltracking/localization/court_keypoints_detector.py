from itertools import zip_longest
from pathlib import Path
import numpy as np
import torch
import torchvision.transforms.v2 as T
import torch.nn.functional as F

ImagePoint = tuple[float, float]


class CourtKeypointsDetector:
    def __init__(self, device, score_threshold=0.5, batch_size=6) -> None:
        self.model = torch.jit.load(
            Path(__file__).parent / "court_keypoints_detector.pt"
        )
        self.device = device
        self.model = self.model.to(self.device)
        self.score_threshold = score_threshold
        self.batch_size = batch_size

    TARGET_WIDTH = 1280 
    TARGET_HEIGHT = 720

    # TODO: this thing can potentially be JIT-ed into torchscript
    transformation_pipeline = T.Compose(
        [
            # Ensure we convert to an image tensor first because T.Resize() is a no-op when fed numpy arrays, which future code changes may do.
            T.ToImage(),
            T.Resize(
                (TARGET_HEIGHT, TARGET_WIDTH),
                # Explicitly set antialias to true to ensure consistent behaviour with older versions of Torchvision (changed default).
                antialias=True,
            ),
            T.ToDtype(torch.float32, scale=True),
            T.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )

    @torch.inference_mode()
    def predict(
        self, rgb_frames: list[np.ndarray]
    ) -> list[list[ImagePoint | tuple[None, None]]]:
        if len(rgb_frames) == 0:
            raise ValueError("rgb_frames list is empty.")
        input_shape = rgb_frames[0].shape
        if not all(f.shape == input_shape for f in rgb_frames):
            raise ValueError(
                "Input frames in the same batch need to have the same dimensions."
            )
        if not len(input_shape) == 3 or input_shape[2] != 3:
            raise ValueError("Input arrays need to be (H W C) image tensors")
        res = []
        for group in self.to_groups(rgb_frames, self.batch_size):
            batch = np.stack(group)
            # Pipeline expects (C H W), but cv2 and np.array(Image) will give (H W C).
            batch = np.moveaxis(batch, 3, 1)
            batch = torch.from_numpy(batch)
            res.extend(self.predict_batch(batch))
        return res

    @staticmethod
    def to_groups(rgb_frames, n):
        iterators = [iter(rgb_frames)] * n
        for b in zip_longest(*iterators):
            yield [x for x in b if x is not None]

    @torch.inference_mode()
    def predict_batch(self, batch: torch.Tensor):
        # pipeline expects (N C H W)
        _, chans, input_height, input_width = batch.shape
        assert chans == 3, batch.shape
        batch = batch.to(self.device)
        batch = self.transformation_pipeline(batch)
        yhat = self.model(batch)
        yhat_s = self.upscale(yhat)
        batch_kps = []
        for hm in yhat_s:
            raw_kps = self.heatmaps_to_keypoints(hm)
            # Scale the keypoints back to the original image dimensions.
            kps = self.rescale_kps(raw_kps, input_width, input_height)
            batch_kps.append(kps)
        return batch_kps

    def heatmaps_to_keypoints(self, kp_hms: torch.Tensor):
        # TODO: this function can be optimized dramatically by just using tensor ops instead of a Python for loop but that's not a good use of my time right now.
        # Get the coordinate of each keypoint from its corresponding heatmap by finding the center of mass.
        assert kp_hms.shape == torch.Size(
            (21, self.TARGET_HEIGHT, self.TARGET_WIDTH)
        ), kp_hms.shape
        kp_coords = []
        indices_y, indices_x = torch.meshgrid(
            torch.arange(kp_hms.shape[1], device=kp_hms.device),
            torch.arange(kp_hms.shape[2], device=kp_hms.device),
            indexing="ij",
        )
        kp_hms_threshed = F.threshold(kp_hms, self.score_threshold, 0)
        for i in range(21):
            hm_threshed = kp_hms_threshed[i]
            coords = [torch.nan, torch.nan]
            if torch.any(hm_threshed > self.score_threshold):
                # Calculate center of mass
                hm_normed = hm_threshed / hm_threshed.sum()
                x = (hm_normed * indices_x).sum()
                y = (hm_normed * indices_y).sum()
                coords = [x, y]
            kp_coords.append(coords)

        # Collect all CUDA tensors into a tensor at once which apparently allows for faster GPU->CPU synchronization.
        collected = torch.Tensor(kp_coords, device="cpu").tolist()

        # Use None instead of NaN to prevent downstream confusion (checking for None is generally easier than for np.NaN)
        out = []
        for x, y in collected:
            if not np.isnan(x) and not np.isnan(y):
                out.append((x, y))
            else:
                out.append((None, None))
        return out

    def rescale_kps(self, downscaled_kps, img_width, img_height) -> list[ImagePoint]:
        scale_x = img_width / self.TARGET_WIDTH
        scale_y = img_height / self.TARGET_HEIGHT
        kps = []
        for x, y in downscaled_kps:
            coord = [None, None]
            if x is not None and y is not None:
                coord = [x * scale_x, y * scale_y]
            kps.append(coord)
        return kps

    def upscale(self, t):
        return F.interpolate(
            t,
            size=(self.TARGET_HEIGHT, self.TARGET_WIDTH),
            mode="bilinear",
            align_corners=False,
        )
