package navigation

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// Mover is an interface representing an entity capable of moving and rotating.
// It abstracts the underlying hardware movement commands.
type Mover interface {
	// Move commands the mover to travel in a specified direction (in radians relative to its frame)
	// at the specified speed.
	Move(dir, speed float64) error
	// Rotate commands the mover to rotate at the specified speed.
	Rotate(speed float64) error
	// Stop commands the mover to stop moving.
	Stop() error
}

// RateLimitedMover is a Mover that limits the rate of its actions.
type RateLimitedMover struct {
	mover   Mover
	limiter *rate.Limiter
}

// NewRateLimitedMover wraps an existing Mover with rate limiting.
// It creates a limiter that allows one action per minInterval.
func NewRateLimitedMover(mover Mover, minInterval time.Duration) Mover {
	return &RateLimitedMover{
		mover:   mover,
		limiter: rate.NewLimiter(rate.Every(minInterval), 1),
	}
}

// Move commands the mover to travel in a specified direction (in radians relative to its frame)
// at the specified speed. It blocks until the limiter permits a new move attempt.
func (r *RateLimitedMover) Move(dir, speed float64) error {
	// Block until the limiter permits a new move attempt.
	if err := r.limiter.Wait(context.Background()); err != nil {
		return fmt.Errorf("rate limit exceeded during Move: %w", err)
	}
	return r.mover.Move(dir, speed)
}

// Rotate commands the mover to rotate at the specified speed. It blocks until the limiter permits a new rotate attempt.
func (r *RateLimitedMover) Rotate(speed float64) error {
	// Block until the limiter permits a new rotate attempt.
	if err := r.limiter.Wait(context.Background()); err != nil {
		return fmt.Errorf("rate limit exceeded during Rotate: %w", err)
	}
	return r.mover.Rotate(speed)
}

// Stop commands the mover to stop moving. It doesn't block for safety.
func (r *RateLimitedMover) Stop() error {
	return r.mover.Stop()
}
