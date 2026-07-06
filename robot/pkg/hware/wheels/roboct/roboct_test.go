//go:build linux && hardware

package roboct_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/tensa/pkg/hware/wheels/roboct"
)

func boolPtr(b bool) *bool {
	return &b
}

// TestEnableSpinDisable tests the sequence: enable (lock) -> spin -> disable (unlock)
func TestEnableSpinDisable(t *testing.T) {
	bus, err := canbus.DialSocketCANWithOptions("can1",
		&canbus.SocketCANOptions{
			Loopback:           boolPtr(false),
			ReceiveOwnMessages: boolPtr(false),
			SendBufferBytes:    1 << 20,
			ReceiveBufferBytes: 1 << 20,
		})
	if err != nil {
		t.Skipf("CAN interface can2 not available, skipping test: %v", err)
	}
	defer bus.Close()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus = canbus.NewLoggedBus(bus, logger, slog.LevelDebug, canbus.LogAll)
	mux := canbus.NewMux(bus)

	// Node ID 1 for test
	client := roboct.NewWithMux(bus, mux, 0x02, roboct.WithLogger(logger))
	// No explicit Close on roboct client, but bus closes.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init involves starting node, clearing faults, setting mode, configuring RPDOs, enabling
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Phase 1: Enable wheels (Init already enables)
	t.Log("Phase 1: Enabled (Init implicitly enables)")
	time.Sleep(5 * time.Second)
	logDiagnostics(t, ctx, client, "enabled")

	// Phase 2: Spin at 10 RPM for 5 seconds
	t.Log("Phase 2: Spinning wheel at 10 RPM")
	if err := client.SetVelocity(ctx, 10); err != nil {
		t.Fatalf("SetVelocity failed: %v", err)
	}
	time.Sleep(5 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")

	// Phase 3: Disable wheels (should unlock them for freewheeling)
	t.Log("Phase 3: Disabling wheel - motor should freewheel")
	if err := client.Disable(ctx); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	time.Sleep(5 * time.Second) // Brief wait to verify disable state
	logDiagnostics(t, ctx, client, "disabled")
}

// TestDiagnostics validates basic diagnostic reads: enabled state, speed, current, and fault code.
func TestDiagnostics(t *testing.T) {
	bus, err := canbus.DialSocketCAN("can1")
	if err != nil {
		t.Skipf("CAN interface can1 not available, skipping test: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := roboct.NewWithMux(bus, mux, 0x02, roboct.WithLogger(logger))
	ctx := context.Background()

	// Ensure clean state
	if err := client.ClearFault(ctx); err != nil {
		t.Fatalf("ClearFault failed: %v", err)
	}
	if err := client.Init(ctx); err != nil { // also enables
		t.Fatalf("Init failed: %v", err)
	}

	// Log all diagnostics via helper
	logDiagnostics(t, ctx, client, "final")
}

// logDiagnostics prints a consolidated snapshot of the controller diagnostics.
// It is best-effort: errors are logged and the function continues collecting others.
func logDiagnostics(t *testing.T, ctx context.Context, client *roboct.Client, label string) {
	status, err := client.Status(ctx)
	if err != nil {
		t.Logf("[%s] Status error: %v", label, err)
		return
	}

	// Check if enabled (StatusWord bit 1 "Switched On" and bit 2 "Operation Enabled")
	// Similar logic as in Wheels.Status
	enabled := (status.StatusWord & 0x07) == 0x07

	t.Logf("[%s] Diagnostics: statusWord=0x%04X enabled=%t faults=0x%04X speed_rpm=%d current_mA=%d voltage_V=%.1f temp_C=%d",
		label, status.StatusWord, enabled, status.FaultCode, status.RPM, status.Current, status.Voltage, status.Temperature)
}

func TestFaults(t *testing.T) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		t.Skipf("CAN interface can0 not available, skipping test: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := roboct.NewWithMux(bus, mux, 0x01, roboct.WithLogger(logger))
	ctx := context.Background()

	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Requires external intervention to purposefully fault the controller if not already faulted
	t.Log("Phase 1: Spinning wheel at 10 RPM")
	if err := client.SetVelocity(ctx, 10); err != nil {
		t.Fatalf("SetVelocity failed: %v", err)
	}
	time.Sleep(10 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")

	t.Log("Phase 2: Clear Faults (if any)")
	if err := client.ClearFault(ctx); err != nil {
		t.Fatalf("ClearFault failed: %v", err)
	}
	// Re-enable might be needed after clearing fault depending on drive state machine
	if err := client.Enable(ctx); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	time.Sleep(1 * time.Second)
	logDiagnostics(t, ctx, client, "cleared")

	t.Log("Phase 3: Spinning wheel at 10 RPM")
	if err := client.SetVelocity(ctx, 10); err != nil {
		t.Fatalf("SetVelocity failed: %v", err)
	}
	time.Sleep(10 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")

	if err := client.Halt(ctx); err != nil {
		t.Fatalf("Halt failed: %v", err)
	}
}

// BenchmarkSetVelocity measures CAN dispatch cost of SetVelocity.
func BenchmarkSetVelocity(b *testing.B) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		b.Skipf("CAN interface can0 not available, skipping benchmark: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	client := roboct.NewWithMux(bus, mux, 0x01, roboct.WithLogger(logger))

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		b.Fatalf("Init failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var rpm int16 = 10
		if i%2 == 1 {
			rpm = 0
		}
		if err := client.SetVelocity(ctx, rpm); err != nil {
			b.Fatalf("SetVelocity failed: %v", err)
		}
	}
}
