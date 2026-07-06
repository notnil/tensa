from enum import Enum

""" 
CourtPoint is a tuple of two floats representing the x and y coordinates of a point on the tennis court.
The origin is the center of the court (KP11).  The x-axis is parallel to the net and the y-axis is perpendicular to the net.
The x-axis is positive to the right and the y-axis is positive towards the far end of the court.
"""
CourtPoint = tuple[float, float]

"""
ImagePoint is a tuple of two floats representing the x and y coordinates of a point in an image with potential sub-pixel resolution.
"""
ImagePoint = tuple[float, float]

# Net post is 0.91m (3ft) outside of doubles court.
NET_X = 6.3964

NET_HEIGHT = 1.07


class KeyPoint(Enum):
    """Enum for the key points of the tennis court.
    Key points are used as reference points for the perspective transform and navigation.
    Each key point has a number and a tuple of coordinates (x, y)."""

    KP1 = (1, (-5.4864, 11.8872))
    KP2 = (2, (-4.1148, 11.8872))
    KP3 = (3, (0, 11.8872))
    KP4 = (4, (4.1148, 11.8872))
    KP5 = (5, (5.4864, 11.8872))
    KP6 = (6, (-4.1148, 6.4008))
    KP7 = (7, (0, 6.4008))
    KP8 = (8, (4.1148, 6.4008))
    KP9 = (9, (-5.4864, 0))
    KP10 = (10, (-4.1148, 0))
    KP11 = (11, (0, 0))
    KP12 = (12, (4.1148, 0))
    KP13 = (13, (5.4864, 0))
    KP14 = (14, (-4.1148, -6.4008))
    KP15 = (15, (0, -6.4008))
    KP16 = (16, (4.1148, -6.4008))
    KP17 = (17, (-5.4864, -11.8872))
    KP18 = (18, (-4.1148, -11.8872))
    KP19 = (19, (0, -11.8872))
    KP20 = (20, (4.1148, -11.8872))
    KP21 = (21, (5.4864, -11.8872))

    def __init__(self, number: int, coordinates: tuple[float, float]):
        self.number = number
        self.coordinates = coordinates


class Section(Enum):
    """Enum for the sections of the tennis court.  Sections are rectangular areas of
    the court that may have different rules or be used for different purposes.  For
    example serves need to go in one of the service boxes.  Each section is defined
    by a tuple of key points."""

    FarAdDoublesAlley = (KeyPoint.KP4, KeyPoint.KP5, KeyPoint.KP13, KeyPoint.KP12)
    FarAdServiceBox = (KeyPoint.KP7, KeyPoint.KP8, KeyPoint.KP12, KeyPoint.KP11)
    FarBackcourt = (KeyPoint.KP2, KeyPoint.KP4, KeyPoint.KP8, KeyPoint.KP6)
    FarDeuceDoublesAlley = (KeyPoint.KP1, KeyPoint.KP2, KeyPoint.KP10, KeyPoint.KP9)
    FarDeuceServiceBox = (KeyPoint.KP6, KeyPoint.KP7, KeyPoint.KP11, KeyPoint.KP10)
    NearAdDoublesAlley = (KeyPoint.KP9, KeyPoint.KP10, KeyPoint.KP18, KeyPoint.KP17)
    NearAdServiceBox = (KeyPoint.KP10, KeyPoint.KP11, KeyPoint.KP15, KeyPoint.KP14)
    NearBackcourt = (KeyPoint.KP14, KeyPoint.KP16, KeyPoint.KP20, KeyPoint.KP18)
    NearDeuceDoublesAlley = (KeyPoint.KP12, KeyPoint.KP13, KeyPoint.KP21, KeyPoint.KP20)
    NearDeuceServiceBox = (KeyPoint.KP11, KeyPoint.KP12, KeyPoint.KP16, KeyPoint.KP15)
    FarSinglesCourt = (KeyPoint.KP2, KeyPoint.KP4, KeyPoint.KP12, KeyPoint.KP10)
    FarDoublesCourt = (KeyPoint.KP1, KeyPoint.KP5, KeyPoint.KP13, KeyPoint.KP9)
    NearSinglesCourt = (KeyPoint.KP10, KeyPoint.KP12, KeyPoint.KP20, KeyPoint.KP18)
    NearDoublesCourt = (KeyPoint.KP9, KeyPoint.KP13, KeyPoint.KP21, KeyPoint.KP17)
    SinglesCourt = (KeyPoint.KP2, KeyPoint.KP4, KeyPoint.KP20, KeyPoint.KP18)
    DoublesCourt = (KeyPoint.KP1, KeyPoint.KP5, KeyPoint.KP21, KeyPoint.KP17)

    def __init__(self, *key_points):
        self.key_points = key_points

    @classmethod
    def primary_sections(cls):
        """Returns a list of the primary sections of the tennis court.  Primary sections
        are the sections have no overlap with other sections."""
        return [
            cls.FarAdDoublesAlley,
            cls.FarAdServiceBox,
            cls.FarBackcourt,
            cls.FarDeuceDoublesAlley,
            cls.FarDeuceServiceBox,
            cls.NearAdDoublesAlley,
            cls.NearAdServiceBox,
            cls.NearBackcourt,
            cls.NearDeuceDoublesAlley,
            cls.NearDeuceServiceBox,
        ]

    def contains(self, pt: CourtPoint) -> bool:
        """Returns True if the point is within the section."""
        x, y = pt
        return (
            self.key_points[0].coordinates[0] <= x <= self.key_points[2].coordinates[0]
            and self.key_points[0].coordinates[1]
            <= y
            <= self.key_points[2].coordinates[1]
        )
