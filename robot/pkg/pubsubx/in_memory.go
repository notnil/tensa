package pubsubx

import (
	"context"

	"github.com/cskr/pubsub/v2"
)

var _ Pub[any] = &InMemoryPub[any]{}
var _ Sub[any] = &InMemorySub[any]{}

// NewInMemoryPubSub creates a new instance of the cskr/pubsub PubSub with string keys and any values.
// The capacity parameter determines the buffer size for each subscription channel.
func NewInMemoryPubSub(capacity int) *pubsub.PubSub[string, any] {
	return pubsub.New[string, any](capacity)
}

// InMemoryPub is a generic publisher that sends messages of type T using the cskr/pubsub library.
type InMemoryPub[T any] struct {
	ps    *pubsub.PubSub[string, any]
	topic string
}

// NewInMemoryPublisher creates and returns a new InMemoryPub instance for the provided topic.
func NewInMemoryPublisher[T any](ps *pubsub.PubSub[string, any], topic string) *InMemoryPub[T] {
	return &InMemoryPub[T]{
		ps:    ps,
		topic: topic,
	}
}

// Publish sends a message of type T to the associated topic.
// The message is passed directly without serialization.
func (p *InMemoryPub[T]) Publish(message T) error {
	p.ps.Pub(message, p.topic)
	return nil
}

// InMemorySub is an implementation of the Sub interface using the cskr/pubsub library.
// It subscribes to a specific topic and handles type assertion of incoming messages.
type InMemorySub[T any] struct {
	ps    *pubsub.PubSub[string, any]
	topic string
}

// NewInMemorySubscriber creates and returns a new InMemorySub instance for the provided topic.
func NewInMemorySubscriber[T any](ps *pubsub.PubSub[string, any], topic string) *InMemorySub[T] {
	return &InMemorySub[T]{
		ps:    ps,
		topic: topic,
	}
}

// Subscribe registers a subscription to the topic.
// Incoming messages are type-asserted to type T and sent to the provided channel.
// If a message cannot be type-asserted to T, it is silently skipped.
// The subscription continues until the context is cancelled.
func (s *InMemorySub[T]) Subscribe(ctx context.Context, ch chan<- T) error {
	subCh := s.ps.Sub(s.topic)
	defer func() {
		go s.ps.Unsub(subCh, s.topic)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-subCh:
			// Type assert the message to T
			payload, ok := msg.(T)
			if !ok {
				// If type assertion fails, skip this message
				// This could happen if messages of different types are published to the same topic
				continue
			}

			// Use a nested select to avoid blocking indefinitely if the output channel is full
			select {
			case ch <- payload:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
