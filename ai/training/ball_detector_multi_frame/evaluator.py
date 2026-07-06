from dataclasses import dataclass
import numpy as np


@dataclass
class BallDetection:
    x: int
    y: int
    score: float
    apparent_radius: int


class Evaluator:
    def __init__(self, dist_threshold_perc) -> None:
        self.trackers = {}
        self.dist_threshold_perc = dist_threshold_perc

    def eval_single_frame(
        self,
        group,
        xyr_gt: tuple[float, float, float],
        preds: list[BallDetection],
        frame_dims: tuple[float, float],
    ):
        if group not in self.trackers:
            self.trackers[group] = PerfTracker(self.dist_threshold_perc)
        return self.trackers[group].eval_single_frame(xyr_gt, preds, frame_dims)


class PerfTracker(object):
    def __init__(self, dist_threshold_perc):
        self.dist_threshold_perc = dist_threshold_perc
        self.frames = 0
        self.tp = 0
        self.fp1 = 0
        self.fp2 = 0
        self.tn = 0
        self.fn = 0
        self.centroid_perc_errors = []
        self.radius_errors = []
        self._scores = []

    def eval_single_frame(
        self,
        xyr_gt: tuple[float, float, float],
        preds: list[BallDetection],
        frame_dims: tuple[float, float],
    ):
        diagonal_pixels = np.sqrt((frame_dims[0] ** 2) + (frame_dims[1] ** 2))
        tp, fp1, fp2, tn, fn = 0, 0, 0, 0, 0
        gt_visible = xyr_gt[0] >= 0 and xyr_gt[1] >= 0 and xyr_gt[2] >= 0
        pred_visible = len(preds) > 0
        if gt_visible:
            if pred_visible:
                for p in preds:
                    # TODO: with this approach it is theoretically possible to double-count true positives. Unlikely given the current heatmap-to-detection implementation, though.
                    gt_x, gt_y, gt_r = xyr_gt
                    eucl_distance = np.linalg.norm(
                        np.array((p.x, p.y)) - np.array((gt_x, gt_y))
                    )
                    perc_error = (eucl_distance / diagonal_pixels) * 100
                    self.centroid_perc_errors.append(perc_error)
                    self.radius_errors.append(
                        ((p.apparent_radius - gt_r) / diagonal_pixels) * 100
                    )
                    self._scores.append(p.score)
                    if perc_error < self.dist_threshold_perc:
                        tp += 1
                    else:
                        fp1 += 1
            else:
                fn += 1
        else:
            if pred_visible:
                fp2 += 1
            else:
                tn += 1
        self.tp += tp
        self.fp1 += fp1
        self.fp2 += fp2
        self.tn += tn
        self.fn += fn
        self.frames += 1
        return

    @property
    def fp_all(self):
        return self.fp1 + self.fp2

    @property
    def prec(self):
        prec = 0.0
        if (self.tp + self.fp_all) > 0.0:
            prec = self.tp / (self.tp + self.fp_all)
        return prec

    @property
    def recall(self):
        recall = 0.0
        if (self.tp + self.fn) > 0.0:
            recall = self.tp / (self.tp + self.fn)
        return recall

    @property
    def f1(self):
        f1 = 0.0
        if self.prec + self.recall > 0.0:
            f1 = 2 * self.prec * self.recall / (self.prec + self.recall)
        return f1

    @property
    def accuracy(self):
        accuracy = 0.0
        if self.tp + self.tn + self.fp_all + self.fn > 0.0:
            accuracy = (self.tp + self.tn) / (self.tp + self.tn + self.fp_all + self.fn)
        return accuracy

    def to_dict(self):
        return dict(
            TP=self.tp,
            TN=self.tn,
            FP1=self.fp1,
            FP2=self.fp2,
            FP=self.fp_all,
            FN=self.fn,
            precision=self.prec,
            recall=self.recall,
            F1=self.f1,
            accuracy=self.accuracy,
            mean_perc_error_centroid=mean(self.centroid_perc_errors),
            mean_perc_error_radius=mean(self.radius_errors),
            num_distances_computed=len(self.centroid_perc_errors),
            num_frames=self.frames,
        )


def mean(errors):
    if len(errors) == 0:
        return np.nan
    return np.array(errors).mean()


def is_sane_nonnegative(n):
    if n is None:
        return False
    return n >= 0
