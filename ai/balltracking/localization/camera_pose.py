from dataclasses import dataclass
import court_dims as dimensions
import numpy as np
import cv2 as cv

# Catch old OpenCV versions early by importing solvePnP from the fisheye module separately.
from cv2.fisheye import undistortPoints as fisheyeUndistortPoints


@dataclass
class CameraPoseEstimate:
    # Uses the same coordinate system as `court2d.KeyPoint`: the origin is the center of the court (KP11).
    # The x-axis is parallel to the net and the y-axis is perpendicular to the net.
    # The x-axis is positive to the right and the y-axis is positive towards the far end of the court.
    x: float
    y: float
    altitude: float
    yaw: float

    # Rotation matrix as returned by cv.solvePnP and cv.Rodrigues.
    # Can be used in conjunction with the translation vector to transform a 3D point expressed in the world frame into the camera frame.
    rotation_matrix: np.ndarray
    translation_vector: np.array

    # Parameters used to create the estimate.
    # Attached to this model to facilitate downstream use, as the exact same camera matrix has to be used for e.g. 3D unprojection.
    camera_matrix: np.ndarray
    distortion_coeffs: np.ndarray | None
    fisheye: bool = True


# We add a third dimension to the court space keypoints to indicate their altitude (0).
KP_COURT_SPACE_COORDS = [(*e.coordinates, 0) for e in dimensions.KeyPoint]


def estimate_camera_pose(
    kps: list[dimensions.ImagePoint | tuple[None, None]],
    camera_matrix: np.ndarray,
    distortion_coeffs: np.ndarray | None = None,
    fisheye=True,
) -> CameraPoseEstimate | None:
    assert len(kps) == 21
    assert camera_matrix.shape == (3, 3), camera_matrix.shape
    if distortion_coeffs is None:
        distortion_coeffs = np.zeros((4,), dtype=np.float32)
    distortion_coeffs = np.asarray(distortion_coeffs, dtype=np.float32).reshape(-1)

    object_points = []
    image_points = []
    for ckp, kp in zip(KP_COURT_SPACE_COORDS, kps):
        kpx, kpy = kp
        if (
            kpx is not None
            and kpy is not None
            and not np.isnan(kpx)
            and not np.isnan(kpy)
        ):
            object_points.append(ckp)
            image_points.append(kp)

    # Solver results using fewer than 4 keypoints are generally too inaccurate to use, despite solver claiming success.
    MIN_KPS = 4
    if len(object_points) < MIN_KPS:
        return None

    object_points = np.array(object_points, dtype=np.float32)
    image_points = np.array(image_points, dtype=np.float32)
    success = False
    rotation_vector = None
    translation_vector = None

    # Real cameras are calibrated with the fisheye camera model, but sim cameras work best with the regular model.
    if fisheye:
        try:
            undistorted_kps = fisheyeUndistortPoints(
                np.expand_dims(np.array(image_points), -2),
                camera_matrix,
                distortion_coeffs,
            )
            success, rotation_vector, translation_vector, _ = cv.solvePnPRansac(
                object_points,
                undistorted_kps,
                # Identity intrinsics because points are already undistorted.
                np.eye(3),
                np.zeros((1, 5)),
                flags=cv.SOLVEPNP_SQPNP,
            )
        except cv.error as e:
            print(f"Warning: OpenCV error in solvePnPRansac: {e}")
            # success remains False, will lead to returning None
            pass
    else:
        try:
            success, rotation_vector, translation_vector, _ = cv.solvePnPRansac(
                object_points,
                image_points,
                camera_matrix,
                distortion_coeffs,
                flags=cv.SOLVEPNP_SQPNP,
            )
        except cv.error as e:
            print(f"Warning: OpenCV error in solvePnPRansac: {e}")
            pass

    if success and rotation_vector is not None and translation_vector is not None:
        rotation_matrix, _ = cv.Rodrigues(rotation_vector)
        position = -rotation_matrix.T @ translation_vector
        x, y, z = position.reshape(3).tolist()
        yaw, _, _ = rotation_matrix_to_euler_angles(rotation_matrix)
        return CameraPoseEstimate(
            x=float(x),
            y=float(y),
            altitude=float(z),
            yaw=float(yaw),
            rotation_matrix=rotation_matrix,
            translation_vector=translation_vector,
            camera_matrix=camera_matrix,
            distortion_coeffs=distortion_coeffs,
            fisheye=fisheye,
        )


def rotation_matrix_to_euler_angles(R):
    limit = 1 - 1e-6
    if R[1, 0] > limit:  # singularity at north pole
        yaw = np.arctan2(R[0, 2], R[2, 2])
        pitch = np.pi / 2
        roll = 0
        return yaw, pitch, roll

    if R[1, 0] < -limit:  # singularity at south pole
        yaw = np.arctan2(R[0, 2], R[2, 2])
        pitch = -np.pi / 2
        roll = 0
        return yaw, pitch, roll

    yaw = np.arctan2(-R[2, 0], R[0, 0])
    roll = np.arctan2(-R[1, 2], R[1, 1])
    pitch = np.arcsin(R[1, 0])

    return yaw, pitch, roll
