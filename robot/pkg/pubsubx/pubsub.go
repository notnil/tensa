// Package pubsubx provides generic interfaces and implementations for publish-subscribe messaging patterns.
package pubsubx

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Pub is a generic publisher that sends messages of type T.
type Pub[T any] interface {
	Publish(message T) error
}

// Sub is a generic subscription interface which defines a method for subscribing to a stream of messages of type T.
type Sub[T any] interface {
	// Subscribe registers a subscription using the provided context. Received messages are sent to the provided channel.
	Subscribe(ctx context.Context, ch chan<- T) error
}

// IntervalSub implements Sub[T] by invoking a callback at a regular interval.
type IntervalSub[T any] struct {
	Interval time.Duration
	Callback func() T
}

// NewIntervalSub returns a Sub that calls the provided callback every interval.
func NewIntervalSub[T any](interval time.Duration, callback func() T) *IntervalSub[T] {
	return &IntervalSub[T]{Interval: interval, Callback: callback}
}

// Subscribe starts a ticker at the specified interval and sends each callback result to ch.
// It stops when ctx is done.
func (s *IntervalSub[T]) Subscribe(ctx context.Context, ch chan<- T) error {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ch <- s.Callback()
		}
	}
}

// NoOpSub is a subscriber implementation that does nothing.
// It implements the Sub[T] interface but never sends any messages.
// Useful for testing or when a subscriber is required but not needed.
type NoOpSub[T any] struct{}

// NewNoOpSub returns a new NoOpSub.
func NewNoOpSub[T any]() *NoOpSub[T] {
	return &NoOpSub[T]{}
}

// Subscribe implements the Sub[T] interface but does nothing.
// It immediately returns nil when the context is done.
func (s *NoOpSub[T]) Subscribe(ctx context.Context, ch chan<- T) error {
	<-ctx.Done()
	return nil
}

// Relay reads messages from a subscriber (src) and publishes them to a publisher (dst)
// until the context is cancelled or an error occurs.
// It subscribes to the source in a goroutine and publishes messages received on a channel.
// If the context is cancelled, it returns ctx.Err(). If the subscriber or publisher
// returns an error, Relay returns that error.
func Relay[T any](ctx context.Context, src Sub[T], dst Pub[T]) error {
	// Channel to receive messages from the subscriber
	msgCh := make(chan T)
	// Channel to receive errors from the subscriber goroutine
	errCh := make(chan error, 1) // Use a buffered channel for the error

	// Start the subscription in a goroutine
	go func() {
		// Subscribe will block until ctx is done or an error occurs
		err := src.Subscribe(ctx, msgCh)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			// Send the error to the error channel, unless it's just context cancellation
			errCh <- err
		}
		// Close the message channel when the subscription goroutine exits
		// This signals the main loop that the subscriber is done.
		close(msgCh)
	}()

	// Loop to receive messages and publish them
	for {
		select {
		case <-ctx.Done():
			// Context cancelled, stop relaying
			return ctx.Err()
		case err := <-errCh:
			// An error occurred in the subscriber goroutine
			return err
		case msg, ok := <-msgCh:
			if !ok {
				// Channel closed, subscriber goroutine exited.
				// This should typically happen after ctx.Done() or an error in errCh.
				// If we reach here and ctx is not done and errCh is empty, it indicates
				// the subscriber finished without error but also without context cancellation.
				// This might be unexpected depending on the Sub implementation, but we should exit.
				return nil // Subscriber finished successfully
			}
			// Received a message, publish it
			if err := dst.Publish(msg); err != nil {
				// Publishing failed, return the error
				return fmt.Errorf("failed to publish message: %w", err)
			}
		}
	}
}
