package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/notnil/tensa/pkg/pubsubx"
)

type Metric[T any] struct {
	Value     T         `json:"value"`
	Timestamp time.Time `json:"ts"`
}

func NewMetric[T any](value T) Metric[T] {
	return Metric[T]{
		Value:     value,
		Timestamp: time.Now(),
	}
}

func Log[T any](ctx context.Context, logger *slog.Logger, sub pubsubx.Sub[Metric[T]]) error {
	ch := make(chan Metric[T])
	errCh := make(chan error)
	go func() {
		err := sub.Subscribe(ctx, ch)
		errCh <- err
		close(errCh)
	}()
	for {
		select {
		case <-ctx.Done():
			close(ch)
			return nil
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("failed to subscribe to metrics: %v", err)
			}
		case m := <-ch:
			logger.Info("metric", slog.Any("metric", m))
		}
	}
}

// FilterByTimestamp returns a new slice containing only the metrics whose timestamps are between start and end (inclusive).
func FilterByTimestamp[T any](metrics []Metric[T], start, end time.Time) []Metric[T] {
	var filtered []Metric[T]
	for _, m := range metrics {
		if !m.Timestamp.Before(start) && !m.Timestamp.After(end) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
