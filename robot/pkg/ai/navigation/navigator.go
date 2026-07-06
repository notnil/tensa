package navigation

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/notnil/tensa/pkg/ai/location"
	"github.com/notnil/tensa/pkg/metrics"
	"github.com/notnil/tensa/pkg/pubsubx"
	"github.com/notnil/tensa/pkg/tennis/court2d"
	"github.com/notnil/tensa/pkg/util/concurx"
	"github.com/notnil/tensa/pkg/util/jsonx"
	"github.com/notnil/tensa/pkg/util/rotation"
)

// DefaultSafeZone returns a polygon that represents the safe zone of the court for autonomous navigation.
func DefaultSafeZone() court2d.Polygon {
	return court2d.Polygon{Vertices: []court2d.Point{
		{X: -6.5, Y: -1.0}, {X: 6.5, Y: -1.0}, {X: 6.5, Y: -13.0}, {X: -6.5, Y: -13.0},
	}}
}

// Navigator defines a strategy for navigating a mover to a desired destination.
type Navigator interface {
	Navigate(ctx context.Context, dest location.Loc) error
}

var _ Navigator = &TwoStage{}
var _ Navigator = &Translation{}
var _ Navigator = &Rotation{}

// TwoStageNavigator performs navigation in two sequential stages:
// translation (position adjustment) followed by rotation (orientation alignment).
// RestDuration is used before and after rotation to mitigate momentum.
type TwoStage struct {
	mover        Mover
	sub          pubsubx.Sub[metrics.Metric[location.Loc]]
	log          *slog.Logger
	Translation  Translation    `json:"translation"`
	Rotation     Rotation       `json:"rotation"`
	RestDuration jsonx.Duration `json:"rest_duration"`
	Timeout      jsonx.Duration `json:"timeout"` // Overall timeout for the entire two-stage process
}

// NewTwoStage creates a new TwoStage navigator with the given mover and location subscription.
func NewTwoStage(mover Mover, sub pubsubx.Sub[metrics.Metric[location.Loc]], log *slog.Logger, translation Translation, rotation Rotation, restDuration, timeout jsonx.Duration) *TwoStage {
	// Update the nested navigators with the same mover and sub
	translation.mover = mover
	translation.sub = sub
	translation.log = log
	rotation.mover = mover
	rotation.sub = sub
	rotation.log = log

	return &TwoStage{
		mover:        mover,
		sub:          sub,
		log:          log,
		Translation:  translation,
		Rotation:     rotation,
		RestDuration: restDuration,
		Timeout:      timeout,
	}
}

// Navigate executes the two-stage navigation.
// It first translates the mover to the destination, waits for a rest period,
// then rotates the mover to align with the target orientation, and finally waits again.
func (n *TwoStage) Navigate(ctx context.Context, dest location.Loc) error {
	// Apply overall timeout if configured
	if n.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(n.Timeout))
		defer cancel()
	}

	n.log.Debug("starting translation", "dest", dest)
	// Perform translation navigation to reach near the destination.
	if err := n.Translation.Navigate(ctx, dest); err != nil {
		return err
	}
	// Rest before rotation so momentum doesn't carry over.
	n.log.Debug("sleeping before rotation", "duration", n.RestDuration)
	if err := n.sleep(ctx, time.Duration(n.RestDuration)); err != nil {
		return err
	}
	// Perform rotation navigation to align with the destination orientation.
	n.log.Debug("starting rotation", "dest", dest)
	if err := n.Rotation.Navigate(ctx, dest); err != nil {
		return err
	}
	// Rest after rotation so momentum doesn't affect the next move.
	n.log.Debug("sleeping after rotation", "duration", n.RestDuration)
	if err := n.sleep(ctx, time.Duration(n.RestDuration)); err != nil {
		return err
	}
	n.log.Debug("navigation complete")
	return nil
}

// sleep performs a context-aware sleep that can be cancelled.
func (n *TwoStage) sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Translation is responsible for navigating the mover by adjusting its position.
// It dynamically selects speeds based on the current distance to the destination and
// ensures that the mover stays within a predefined safe zone.
type Translation struct {
	mover       Mover
	sub         pubsubx.Sub[metrics.Metric[location.Loc]]
	log         *slog.Logger
	FarSpeed    float64         `json:"far_speed"`    // Speed used when far from the destination.
	NearSpeed   float64         `json:"near_speed"`   // Reduced speed used when near the destination.
	OnThreshold float64         `json:"on_threshold"` // Distance threshold under which the destination is considered reached.
	SafeZone    court2d.Polygon `json:"safe_zone"`    // Safe operational area for the mover.
	Timeout     jsonx.Duration  `json:"timeout"`      // Timeout for translation operations
}

// NewTranslation creates a new Translation navigator with the given mover and location subscription.
func NewTranslation(mover Mover, sub pubsubx.Sub[metrics.Metric[location.Loc]], log *slog.Logger, farSpeed, nearSpeed, onThreshold float64, safeZone court2d.Polygon, timeout jsonx.Duration) *Translation {
	return &Translation{
		mover:       mover,
		sub:         sub,
		log:         log,
		FarSpeed:    farSpeed,
		NearSpeed:   nearSpeed,
		OnThreshold: onThreshold,
		SafeZone:    safeZone,
		Timeout:     timeout,
	}
}

// Navigate processes localization updates to move the mover towards the destination.
// It subscribes to localization metrics, validates that the mover remains within the safe zone,
// computes the necessary direction and speed, and commands the mover to move until it reaches the destination.
func (n *Translation) Navigate(ctx context.Context, dest location.Loc) error {
	// Apply timeout if configured
	if n.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(n.Timeout))
		defer cancel()
	}

	// Create channels for receiving localization metrics and any subscription errors.
	ch := make(chan metrics.Metric[location.Loc])
	errCh := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer n.mover.Rotate(0)

	// Launch a goroutine to subscribe to localization updates.
	go func() {
		err := n.sub.Subscribe(ctx, ch)
		if err != nil {
			errCh <- err
		}
	}()

	// Continuously process localization data.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errCh:
			// Return an error if the subscription fails while the channel is still open.
			if ok && err != nil {
				return fmt.Errorf("navigation: error subscribing to localization: %v", err)
			}
		case m := <-ch:
			// Obtain the most recent localization update.
			loc := m.Value
			latest := concurx.Drain(ch)
			if !latest.Value.IsZero() {
				loc = latest.Value
			}
			n.log.Debug("translation update", "loc", loc)
			// Verify that the current location is within the designated safe zone.
			if n.SafeZone.ContainsPoint(loc.Location) == false {
				return fmt.Errorf("navigation: out of safe zone %v", loc.Location)
			}
			// Compute the distance to the destination.
			dist := loc.Location.Distance(dest.Location)
			// Check if the mover is on target.
			isOn := dist < n.OnThreshold
			n.log.Debug("translation check", "dist", dist, "on", isOn)
			if isOn {
				return nil
			}
			// Adjust the speed based on proximity to the destination.
			maxDist := court2d.KP13.Point().Distance(court2d.KP1.Point())
			speed := n.NearSpeed + (n.FarSpeed-n.NearSpeed)*(dist/maxDist)
			speed = math.Min(speed, n.FarSpeed)
			// Calculate the direction from the current location to the destination.
			dir := loc.Location.Angle(dest.Location)
			// Adjust by 90 degrees to align with robotics conventions.
			dir = dir + math.Pi/2
			// Correct for current rotation to get the relative direction.
			dir = dir - loc.Rotation
			// Normalize the resulting direction.
			dir = rotation.Normalize(dir)
			n.log.Debug("translation moving", "dir", dir, "speed", speed)
			// Command the mover to move in the computed direction at the determined speed.
			if err := n.mover.Move(dir, speed); err != nil {
				return fmt.Errorf("navigation: error moving: %v", err)
			}
		}
	}
}

// Rotation is responsible for aligning the mover's orientation with the desired target.
// It calculates the angular difference and commands rotation adjustments until alignment is achieved.
type Rotation struct {
	mover       Mover
	sub         pubsubx.Sub[metrics.Metric[location.Loc]]
	log         *slog.Logger
	MaxSpeed    float64        `json:"max_speed"`    // Maximum rotation speed when the angle difference is large.
	MinSpeed    float64        `json:"min_speed"`    // Minimum rotation speed when the angle difference is small.
	OnThreshold float64        `json:"on_threshold"` // Angular threshold under which alignment is considered complete.
	Timeout     jsonx.Duration `json:"timeout"`      // Timeout for rotation operations
}

// NewRotation creates a new Rotation navigator with the given mover and location subscription.
func NewRotation(mover Mover, sub pubsubx.Sub[metrics.Metric[location.Loc]], log *slog.Logger, maxSpeed, minSpeed, onThreshold float64, timeout jsonx.Duration) *Rotation {
	return &Rotation{
		mover:       mover,
		sub:         sub,
		log:         log,
		MaxSpeed:    maxSpeed,
		MinSpeed:    minSpeed,
		OnThreshold: onThreshold,
		Timeout:     timeout,
	}
}

// Navigate processes localization updates to rotate the mover until its orientation
// matches that of the destination. It calculates the angular difference and applies
// rotation commands scaled by the determined speed and direction.
func (n *Rotation) Navigate(ctx context.Context, dest location.Loc) error {
	// Apply timeout if configured
	if n.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(n.Timeout))
		defer cancel()
	}

	// Create channels for receiving localization metrics and any subscription errors.
	ch := make(chan metrics.Metric[location.Loc])
	errCh := make(chan error)
	// Launch a goroutine to subscribe to localization updates.
	go func() {
		err := n.sub.Subscribe(ctx, ch)
		if err != nil {
			errCh <- err
		}
	}()
	// Ensure channel closure and stop rotation when function exits.
	defer n.mover.Rotate(0)

	// Continuously process localization data.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errCh:
			// Return an error if the subscription fails while the channel is open.
			if ok && err != nil {
				return fmt.Errorf("navigation: error subscribing to localization: %v", err)
			}
		case m := <-ch:
			// Obtain the most recent localization update.
			loc := m.Value
			latest := concurx.Drain(ch)
			if !latest.Value.IsZero() {
				loc = latest.Value
			}
			n.log.Debug("rotation update", "loc", loc)
			// Compute the angular difference and the direction needed to align with the destination orientation.
			dir, rot := rotation.Diff(loc.Rotation, dest.Rotation)
			// If already aligned within threshold, finish rotation.
			isOn := rot < n.OnThreshold
			n.log.Debug("rotation check", "current_rot", loc.Rotation, "dest_rot", dest.Rotation, "dir", dir, "rot", rot, "threshold", n.OnThreshold, "on", isOn)
			if isOn {
				return nil
			}
			// Linearly interpolate the speed: n.MinSpeed at zero rotation up to n.MaxSpeed at a rotation of pi/2.
			speed := n.MinSpeed + (n.MaxSpeed-n.MinSpeed)*(rot/(math.Pi/2))
			n.log.Debug("rotation rotating", "dir", dir, "speed", speed)
			// Command the mover to rotate using the computed speed and directional factor.
			if err := n.mover.Rotate(speed * float64(dir)); err != nil {
				return fmt.Errorf("navigation: error rotating: %v", err)
			}
		}
	}
}
