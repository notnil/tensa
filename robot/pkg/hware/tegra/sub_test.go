package tegra

import (
	"context"
	"log/slog"
	"os/exec"
	"testing"
	"time"

	"github.com/notnil/tensa/pkg/metrics"
)

func TestStatsStreamer(t *testing.T) {
	if _, err := exec.LookPath("tegrastats"); err != nil {
		t.Skip("tegrastats is only available on Jetson devices")
	}

	parser := NanoParser{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sub := NewStatsSub(slog.Default(), parser)
	errCh := make(chan error, 1)
	statsCh := make(chan metrics.Metric[Stats], 1)
	go func() {
		errCh <- sub.Subscribe(ctx, statsCh)
	}()

	for {
		select {
		case err := <-errCh:
			t.Fatalf("failed to stream stats: %v", err)
		case stats := <-statsCh:
			t.Logf("stats: %v", stats)
			return
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}
}
