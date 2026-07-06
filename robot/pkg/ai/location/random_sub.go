package location

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/numeric"
	"github.com/notnil/tensa/pkg/util/rotation"
)

// DefaultPositionDelta is the default range for position changes in the random walk.
var DefaultPositionDelta = numeric.Range[float64]{Min: 0.0, Max: 0.5}

// DefaultRotationDelta is the default range for rotation changes in the random walk.
var DefaultRotationDelta = numeric.Range[float64]{Min: -math.Pi / 18, Max: math.Pi / 18}

// RandomWalkConfig contains configuration for the random walk subscriber.
type RandomWalkConfig struct {
	Polygon         court2d.Polygon
	Interval        time.Duration
	Seed            int64
	PositionDelta   numeric.Range[float64]
	RotationDelta   numeric.Range[float64]
	InitialPoint    *court2d.Point
	InitialRotation *float64
}

// RandomWalkSub generates random walk Loc metrics within a specified polygon at a given interval.
// It simulates a random walk movement pattern within the bounds of the polygon.
type RandomWalkSub struct {
	polygon       court2d.Polygon
	interval      time.Duration
	current       Loc
	rand          *rand.Rand
	positionDelta numeric.Range[float64]
	rotationDelta numeric.Range[float64]
}

// NewRandomWalkSub creates a new RandomWalkSub with the given configuration.
func NewRandomWalkSub(cfg RandomWalkConfig) *RandomWalkSub {
	r := rand.New(rand.NewSource(cfg.Seed))

	var initialPoint court2d.Point
	if cfg.InitialPoint != nil {
		initialPoint = *cfg.InitialPoint
	} else {
		initialPoint = cfg.Polygon.RandomPoint(r)
	}

	var initialRotation float64
	if cfg.InitialRotation != nil {
		initialRotation = *cfg.InitialRotation
	} else {
		initialRotation = r.Float64() * math.Pi * 2
	}

	return &RandomWalkSub{
		polygon:       cfg.Polygon,
		interval:      cfg.Interval,
		positionDelta: cfg.PositionDelta,
		rotationDelta: cfg.RotationDelta,
		current: Loc{
			Location: initialPoint,
			Rotation: initialRotation,
		},
		rand: r,
	}
}

// Subscribe starts generating random walk Loc messages and sending them to the provided channel.
// It stops when the context is done.
func (s *RandomWalkSub) Subscribe(ctx context.Context, ch chan<- metrics.Metric[Loc]) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.updateLocation()
			ch <- metrics.Metric[Loc]{
				Value:     s.current,
				Timestamp: time.Now(),
			}
		}
	}
}

func (s *RandomWalkSub) updateLocation() {
	deltaPos := s.positionDelta.Sample(s.rand)
	moveDir := s.rand.Float64() * math.Pi * 2
	pt := s.current.Location.Destination(moveDir, deltaPos)
	if !s.polygon.ContainsPoint(pt) {
		s.updateLocation()
		return
	}

	deltaRotation := s.rotationDelta.Sample(s.rand)
	rotation := rotation.Add(s.current.Rotation, deltaRotation)
	s.current.Rotation = rotation
	s.current.Location = pt
}
