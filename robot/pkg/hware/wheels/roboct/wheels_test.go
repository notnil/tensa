//go:build linux && hardware

package roboct_test

import (
	"context"
	"log/slog"
	"math"
	"os"
	"testing"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/tensa/pkg/hware/wheels"
	"github.com/notnil/tensa/pkg/hware/wheels/roboct"
	"github.com/notnil/tensa/pkg/util/rotation"
)

// TestWheels_EnableMoveDisable mirrors the client-level enable/spin/disable flow
// but exercises the roboct Wheels implementation (four-node mecanum translation).
func TestWheels_EnableMoveDisable(t *testing.T) {
	// Setup Bus
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		t.Skipf("CAN interface can0 not available, skipping test: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Setup 4 clients with assumed IDs 1, 2, 3, 4
	// Assuming typical setup: FL=1, FR=2, RL=3, RR=4
	fl := roboct.NewWithMux(bus, mux, 1, roboct.WithLogger(logger))
	fr := roboct.NewWithMux(bus, mux, 2, roboct.WithLogger(logger))
	rl := roboct.NewWithMux(bus, mux, 3, roboct.WithLogger(logger))
	rr := roboct.NewWithMux(bus, mux, 4, roboct.WithLogger(logger))

	wh := roboct.NewWheels(fl, fr, rl, rr, roboct.WithWheelsLogger(logger))

	// No Close method on roboct.Wheels that closes clients (it relies on user owning clients),
	// but we should probably stop/disable at end.
	defer func() {
		_ = wh.Stop()
		_ = wh.Disable()
	}()

	ctx := context.Background()
	if err := wh.Init(ctx); err != nil {
		t.Fatalf("wheels Init failed: %v", err)
	}

	// Convert ~10 RPM to rad/s for wheels API
	speedRad := 10.0 / 9.549297
	dirs := []struct {
		name string
		dir  float64
	}{
		{name: "North", dir: rotation.North},
		{name: "East", dir: rotation.East},
		{name: "South", dir: rotation.South},
		{name: "West", dir: rotation.West},
	}
	for _, d := range dirs {
		// Enable
		if err := wh.Enable(); err != nil {
			t.Fatalf("wheels Enable failed (%s): %v", d.name, err)
		}
		time.Sleep(2 * time.Second)
		logWheelsStatus(t, wh, "enabled-"+d.name)

		// Move
		if err := wh.Move(d.dir, speedRad); err != nil {
			t.Fatalf("wheels Move failed (%s): %v", d.name, err)
		}
		time.Sleep(5 * time.Second)
		logWheelsStatus(t, wh, "moving-"+d.name)

		// Disable
		if err := wh.Disable(); err != nil {
			t.Fatalf("wheels Disable failed (%s): %v", d.name, err)
		}
		time.Sleep(1 * time.Second)
		logWheelsStatus(t, wh, "disabled-"+d.name)
	}
}

func logWheelsStatus(t *testing.T, wh wheels.Wheels, label string) {
	st, err := wh.Status()
	if err != nil {
		t.Logf("[%s] Status error: %v", label, err)
		return
	}
	// Round speeds for compact logging
	r := func(v float64) float64 { return math.Round(v*10) / 10 }
	t.Logf("[%s] FL{en=%t sp=%.1f cur=%.1f err=%q} FR{en=%t sp=%.1f cur=%.1f err=%q} RL{en=%t sp=%.1f cur=%.1f err=%q} RR{en=%t sp=%.1f cur=%.1f err=%q}",
		label,
		st.FrontLeft.Enabled, r(st.FrontLeft.Speed), st.FrontLeft.Current, st.FrontLeft.Error,
		st.FrontRight.Enabled, r(st.FrontRight.Speed), st.FrontRight.Current, st.FrontRight.Error,
		st.RearLeft.Enabled, r(st.RearLeft.Speed), st.RearLeft.Current, st.RearLeft.Error,
		st.RearRight.Enabled, r(st.RearRight.Speed), st.RearRight.Current, st.RearRight.Error,
	)
}

// BenchmarkWheelsStatus measures the cost of retrieving wheel status via the
// roboct Wheels implementation. Skips if CAN is unavailable.
func BenchmarkWheelsStatus(b *testing.B) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		b.Skipf("CAN interface can0 not available, skipping benchmark: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	fl := roboct.NewWithMux(bus, mux, 1, roboct.WithLogger(logger))
	fr := roboct.NewWithMux(bus, mux, 2, roboct.WithLogger(logger))
	rl := roboct.NewWithMux(bus, mux, 3, roboct.WithLogger(logger))
	rr := roboct.NewWithMux(bus, mux, 4, roboct.WithLogger(logger))

	wh := roboct.NewWheels(fl, fr, rl, rr, roboct.WithWheelsLogger(logger))

	ctx := context.Background()
	if err := wh.Init(ctx); err != nil {
		b.Fatalf("wheels Init failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := wh.Status()
		if err != nil {
			b.Fatalf("Status failed: %v", err)
		}
		b.Logf("status: %s", s.String())
	}
}
