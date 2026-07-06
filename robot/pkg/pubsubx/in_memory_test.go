package pubsubx_test

import (
	"bytes"
	"context"
	"encoding"
	"encoding/binary"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/pubsubx"
)

func TestInMemoryPubSub(t *testing.T) {
	subject := "test"
	ps := pubsubx.NewInMemoryPubSub(0)
	sub := pubsubx.NewInMemorySubscriber[string](ps, subject)
	pub := pubsubx.NewInMemoryPublisher[string](ps, subject)

	// Create a channel to receive published messages.
	ch := make(chan string, 1)

	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run the subscriber in a separate goroutine.
	subErrCh := make(chan error, 1)
	go func() {
		subErrCh <- sub.Subscribe(ctx, ch)
	}()

	// Allow brief time for subscription to be established.
	time.Sleep(10 * time.Millisecond)

	// Publish a test message.
	testMsg := "hello world"
	if err := pub.Publish(testMsg); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Wait for the message or timeout.
	select {
	case msg := <-ch:
		if msg != testMsg {
			t.Fatalf("Received unexpected message: got %q, want %q", msg, testMsg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Cancel the context to allow the subscription to end.
	cancel()

	// Verify the subscriber returned a cancellation-related error.
	err := <-subErrCh
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("Subscriber returned unexpected error: %v", err)
	}
}

func BenchmarkInMemoryPubSub_RoundTrip(b *testing.B) {
	ps := pubsubx.NewInMemoryPubSub(10)
	sub := pubsubx.NewInMemorySubscriber[[]byte](ps, "testing")
	pub := pubsubx.NewInMemoryPublisher[[]byte](ps, "testing")

	// Create a channel to receive messages
	ch := make(chan []byte, 10)

	// Create context for subscription
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start subscriber
	go func() {
		sub.Subscribe(ctx, ch)
	}()
	// Allow brief time for subscription to be established.
	time.Sleep(10 * time.Millisecond)

	// Create 1MB test payload
	data := make([]byte, 1<<20) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := pub.Publish(data); err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
		<-ch
	}
}

func TestInMemoryPubSub_BinaryMarshaling(t *testing.T) {
	ps := pubsubx.NewInMemoryPubSub(0)
	sub := pubsubx.NewInMemorySubscriber[binaryTestMessage](ps, "binary-test")
	pub := pubsubx.NewInMemoryPublisher[binaryTestMessage](ps, "binary-test")

	// Create a channel to receive messages
	ch := make(chan binaryTestMessage, 1)

	// Create context for subscription
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start subscriber in a goroutine
	subErrCh := make(chan error, 1)
	go func() {
		subErrCh <- sub.Subscribe(ctx, ch)
	}()

	// Allow brief time for subscription to be established
	time.Sleep(10 * time.Millisecond)

	// Create test message with binary marshaling capability
	testMsg := binaryTestMessage{
		ID:      42,
		Name:    "test message",
		Payload: []byte("binary payload"),
	}

	// Publish the message
	if err := pub.Publish(testMsg); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Wait for the message to be received
	select {
	case receivedMsg := <-ch:
		if receivedMsg.ID != testMsg.ID {
			t.Errorf("Received message with wrong ID: got %d, want %d", receivedMsg.ID, testMsg.ID)
		}
		if receivedMsg.Name != testMsg.Name {
			t.Errorf("Received message with wrong Name: got %s, want %s", receivedMsg.Name, testMsg.Name)
		}
		if !bytes.Equal(receivedMsg.Payload, testMsg.Payload) {
			t.Errorf("Received message with wrong Payload: got %v, want %v", receivedMsg.Payload, testMsg.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message")
	}

	// Cancel the context to allow the subscription to end
	cancel()

	// Verify the subscriber returned a cancellation-related error
	err := <-subErrCh
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("Subscriber returned unexpected error: %v", err)
	}
}

// binaryTestMessage implements encoding.BinaryMarshaler and encoding.BinaryUnmarshaler
type binaryTestMessage struct {
	ID      int
	Name    string
	Payload []byte
}

var _ encoding.BinaryMarshaler = binaryTestMessage{}
var _ encoding.BinaryUnmarshaler = &binaryTestMessage{}

func (m binaryTestMessage) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write ID
	if err := binary.Write(buf, binary.LittleEndian, int64(m.ID)); err != nil {
		return nil, err
	}

	// Write Name length and Name
	nameBytes := []byte(m.Name)
	if err := binary.Write(buf, binary.LittleEndian, int32(len(nameBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(nameBytes); err != nil {
		return nil, err
	}

	// Write Payload length and Payload
	if err := binary.Write(buf, binary.LittleEndian, int32(len(m.Payload))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(m.Payload); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *binaryTestMessage) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	// Read ID
	var id int64
	if err := binary.Read(buf, binary.LittleEndian, &id); err != nil {
		return err
	}
	m.ID = int(id)

	// Read Name length and Name
	var nameLen int32
	if err := binary.Read(buf, binary.LittleEndian, &nameLen); err != nil {
		return err
	}
	nameBytes := make([]byte, nameLen)
	if _, err := buf.Read(nameBytes); err != nil {
		return err
	}
	m.Name = string(nameBytes)

	// Read Payload length and Payload
	var payloadLen int32
	if err := binary.Read(buf, binary.LittleEndian, &payloadLen); err != nil {
		return err
	}
	m.Payload = make([]byte, payloadLen)
	if _, err := buf.Read(m.Payload); err != nil {
		return err
	}

	return nil
}
