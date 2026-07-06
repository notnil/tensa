import numpy as np

KeypointCandidate = tuple[float, float] | tuple[None, None]


class Evaluator:
    def __init__(self, dist_threshold_px) -> None:
        self.trackers = {}
        self.dist_threshold_px = dist_threshold_px

    def eval_single_frame(
        self, group, gt_kps: list[KeypointCandidate], pred_kps: list[KeypointCandidate]
    ):
        if group not in self.trackers:
            self.trackers[group] = PerfTracker(self.dist_threshold_px)
        return self.trackers[group].eval_single_frame(gt_kps, pred_kps)


class PerfTracker(object):
    def __init__(self, dist_threshold_px):
        self.dist_threshold_px = dist_threshold_px
        self.frames = 0
        self.tp = 0
        self.fp1 = 0
        self.fp2 = 0
        self.tn = 0
        self.fn = 0
        self.l1_errors = []

    def eval_single_frame(
        self, gt_kps: list[KeypointCandidate], pred_kps: list[KeypointCandidate]
    ):
        self.frames += 1
        for gt, pred in zip(gt_kps, pred_kps):
            self.eval_keypoint(gt, pred)

    def eval_keypoint(self, xy_gt: KeypointCandidate, xy_pred: KeypointCandidate):
        tp, fp1, fp2, tn, fn = 0, 0, 0, 0, 0
        visi_gt = is_sane_nonnegative(xy_gt[0]) and is_sane_nonnegative(xy_gt[1])
        visi_pred = is_sane_nonnegative(xy_pred[0]) and is_sane_nonnegative(xy_pred[1])
        if visi_gt:
            if visi_pred:
                eucl_distance = np.linalg.norm(np.array(xy_pred) - np.array(xy_gt))
                if eucl_distance < self.dist_threshold_px:
                    tp += 1
                else:
                    fp1 += 1
                self.l1_errors.append(eucl_distance)
            else:
                fn += 1
        else:
            if visi_pred:
                fp2 += 1
            else:
                tn += 1
        self.tp += tp
        self.fp1 += fp1
        self.fp2 += fp2
        self.tn += tn
        self.fn += fn

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

    @property
    def rmse(self):
        if len(self.l1_errors) == 0:
            return np.nan
        sq_errs = np.square(np.array(self.l1_errors))
        _rmse = np.sqrt(np.array(sq_errs).mean())
        return _rmse

    @property
    def MAE(self):
        if len(self.l1_errors) == 0:
            return np.nan
        return np.array(self.l1_errors).mean()

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
            RMSE=self.rmse,
            MAE=self.MAE,
            num_distances_computed=len(self.l1_errors),
            num_frames=self.frames,
        )


def is_sane_nonnegative(n):
    if n is None:
        return False
    return n >= 0
