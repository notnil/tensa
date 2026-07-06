package pubsubx_test

import (
	"context"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/pubsubx"
)

func TestRelay(t *testing.T) {
	// Use a context with a timeout to prevent the test from hanging indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an in-memory pubsub system with some capacity.
	ps := pubsubx.NewInMemoryPubSub(10)

	// Define source and destination topics.
	srcTopic := "source_messages"
	dstTopic := "relayed_messages"

	// Create components for the relay test
	srcSub := pubsubx.NewInMemorySubscriber[string](ps, srcTopic)
	dstPub := pubsubx.NewInMemoryPublisher[string](ps, dstTopic)
	dstSub := pubsubx.NewInMemorySubscriber[string](ps, dstTopic)
	srcPub := pubsubx.NewInMemoryPublisher[string](ps, srcTopic)

	// Create a channel to receive messages from destination subscriber
	dstCh := make(chan string, 10)

	// Start the destination subscriber
	dstSubErrCh := make(chan error, 1)
	dstCtx, dstCancel := context.WithCancel(ctx)
	go func() {
		dstSubErrCh <- dstSub.Subscribe(dstCtx, dstCh)
	}()

	// Run the Relay function in a goroutine.
	relayErrCh := make(chan error, 1)
	relayCtx, relayCancel := context.WithCancel(ctx)
	go func() {
		relayErrCh <- pubsubx.Relay(relayCtx, srcSub, dstPub)
	}()

	// Allow time for subscribers and Relay to start.
	time.Sleep(100 * time.Millisecond)

	// Test messages to send.
	testMessages := []string{"msg1", "msg2", "msg3"}

	// Publish messages to the source topic
	for _, msg := range testMessages {
		if err := srcPub.Publish(msg); err != nil {
			t.Fatalf("Failed to publish message: %v", err)
		}
	}

	// Wait for the messages to be relayed and received on the destination topic.
	receivedMessages := []string{}
	timeout := time.After(5 * time.Second)

	for i := 0; i < len(testMessages); i++ {
		select {
		case receivedMsg := <-dstCh:
			receivedMessages = append(receivedMessages, receivedMsg)
		case err := <-relayErrCh:
			t.Fatalf("Relay goroutine exited with error: %v", err)
		case err := <-dstSubErrCh:
			t.Fatalf("Destination subscriber exited with error: %v", err)
		case <-timeout:
			t.Fatalf("Timed out waiting for message %d/%d. Received: %v", i+1, len(testMessages), receivedMessages)
		}
	}

	// Verify that all messages were received
	if len(receivedMessages) != len(testMessages) {
		t.Fatalf("Expected %d messages, but received %d. Received: %v", len(testMessages), len(receivedMessages), receivedMessages)
	}
	for i := range testMessages {
		if receivedMessages[i] != testMessages[i] {
			t.Errorf("Message %d mismatch: expected %q, got %q", i, testMessages[i], receivedMessages[i])
		}
	}

	// Signal Relay to stop by cancelling the context.
	relayCancel()
	dstCancel()

	// Wait for the Relay goroutine to exit cleanly.
	select {
	case err := <-relayErrCh:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Relay goroutine exited with unexpected error after context cancellation: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Relay goroutine did not exit after context cancellation")
	}
}
