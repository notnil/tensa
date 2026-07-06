import numpy as np
import cv2
import torch
from localization.camera_pose import CameraPoseEstimate, estimate_camera_pose
from localization.court_keypoints_detector import CourtKeypointsDetector
from dataclasses import dataclass


def angle_mod(angle_rad: float) -> float:
    return float((angle_rad + np.pi) % (2 * np.pi) - np.pi)


@dataclass
class CameraPose:
    cam_name: str
    x: float
    y: float
    altitude: float
    yaw: float


@dataclass
class CameraLocalizerResult:
    name: str
    keypoints: list[tuple[float, float] | tuple[None, None]]
    pose: CameraPose | None = None

    _raw: CameraPoseEstimate | None = None


class LocalizerModel:
    def __init__(
        self,
        device,
        camera_intrinsics: dict[str, dict[str, np.ndarray | bool]] | None = None,
        batch_size=6,
    ):
        self.device = device
        self.model = CourtKeypointsDetector(device, batch_size=batch_size)
        self.camera_intrinsics = camera_intrinsics or {}

    def correct_pose(self, p):
        # Images taken close to the net but facing backwards may couse the court keypoint detection model to output far court keypoints, so we assume a camera pose is actually in the near court.
        x, y, yaw = p.x, p.y, p.yaw
        if p.y > 0:
            yaw = angle_mod(p.yaw - np.pi)
            x = -x
            y = -y
        return x, y, yaw

    def decode_img_as_rgb_tensor(self, img: bytes | np.ndarray | torch.Tensor):
        if isinstance(img, torch.Tensor):
            tensor = img
        else:
            if isinstance(img, bytes):
                # TODO: instead of this abomination, use `torchvision.io.decode_jpeg` once our base container supports it (GPU-accelerated).
                arr = cv2.cvtColor(
                    cv2.imdecode(np.frombuffer(img, np.uint8), cv2.IMREAD_COLOR),
                    cv2.COLOR_BGR2RGB,
                )
                tensor = torch.from_numpy(arr)
            elif isinstance(img, np.ndarray):
                arr = img
                tensor = torch.from_numpy(arr)
            else:
                raise Exception("input frame to be bytes, np.ndarray or torch.Tensor")
            # Pipeline expects (C H W), but cv2 will give (H W C).
            tensor = torch.movedim(tensor, 2, 0)
        chans = tensor.shape[0]
        assert chans == 3, tensor.shape
        return tensor

    def predict(
        self, images: list[bytes | np.ndarray | torch.Tensor], camera_names: list[str]
    ) -> list[CameraLocalizerResult]:
        assert len(images) == len(camera_names), (len(images), len(camera_names))
        batch = torch.stack([self.decode_img_as_rgb_tensor(b) for b in images])
        _, _, h, w = batch.shape
        batch_kps = self.model.predict_batch(batch)
        results = []
        for cam_name, kps in zip(camera_names, batch_kps):
            if cam_name in self.camera_intrinsics:
                intr = self.camera_intrinsics[cam_name]
                mat = np.asarray(intr["mat"], dtype=np.float32)
                dist = np.asarray(intr.get("dist", np.zeros((4,), dtype=np.float32)))
                fisheye = bool(intr.get("fisheye", False))
            else:
                # Fallback to simple pinhole intrinsics when calibration is unavailable.
                focal = 0.95 * max(w, h)
                mat = np.array(
                    [[focal, 0.0, w / 2.0], [0.0, focal, h / 2.0], [0.0, 0.0, 1.0]],
                    dtype=np.float32,
                )
                dist = np.zeros((4,), dtype=np.float32)
                fisheye = False
            p = estimate_camera_pose(kps, mat, dist, fisheye)
            r = CameraLocalizerResult(name=cam_name, keypoints=kps)
            r._raw = p
            if p:
                x, y, yaw = self.correct_pose(p)
                pose = CameraPose(
                    cam_name=cam_name, x=x, y=y, yaw=yaw, altitude=p.altitude
                )
                r.pose = pose
            results.append(r)
        return results
