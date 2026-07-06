//go:build linux

package zltech_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/notnil/canbus"
	"github.com/notnil/tensa/pkg/hware/wheels/zltech"
)

func boolPtr(b bool) *bool {
	return &b
}

// TestEnableSpinDisable tests the sequence: enable (lock) -> spin -> disable (unlock)
func TestEnableSpinDisable(t *testing.T) {
	bus, err := canbus.DialSocketCANWithOptions("can2",
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

	client := zltech.NewWithMux(bus, mux, 0x01, zltech.WithLogger(logger), zltech.WithSyncPeriod(5*time.Millisecond))
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	// SYNC is started during Init

	// Phase 1: Enable wheels (should lock them in place)
	t.Log("Phase 1: Enabling wheels - motors should be locked")
	// if err := client.Enable(ctx); err != nil {
	// 	t.Fatalf("Enable failed: %v", err)
	// }
	time.Sleep(5 * time.Second)
	logDiagnostics(t, ctx, client, "enabled")

	// Phase 2: Spin at 10 RPM for 5 seconds
	t.Log("Phase 2: Spinning wheels at 10 RPM")
	if err := client.SetProfileVelocity(ctx, 10, 10, 10, 10); err != nil {
		t.Fatalf("SetProfileVelocity failed: %v", err)
	}
	time.Sleep(5 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")

	// Phase 3: Disable wheels (should unlock them for freewheeling)
	t.Log("Phase 3: Disabling wheels - motors should freewheel")
	if err := client.Disable(ctx); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	time.Sleep(2 * time.Second) // Brief wait to verify disable state
	logDiagnostics(t, ctx, client, "disabled")
}

// TestDiagnostics validates basic diagnostic reads: enabled state, speed, current, and fault code.
func TestDiagnostics(t *testing.T) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		t.Skipf("CAN interface can0 not available, skipping test: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := zltech.NewWithMux(bus, mux, 0x01, zltech.WithLogger(logger), zltech.WithSyncPeriod(5*time.Millisecond))
	defer client.Close()
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
func logDiagnostics(t *testing.T, ctx context.Context, client *zltech.Client, label string) {
	status, err := client.StatusWord(ctx)
	if err != nil {
		t.Logf("[%s] StatusWord error: %v", label, err)
	}
	state := status.State()
	// Treat both OperationEnabled and SwitchedOn as enabled for our use‑case since
	// some firmware reports quick‑stop/target‑reached patterns while still driving.
	enabled := state == zltech.StateOperationEnabled || state == zltech.StateSwitchedOn

	lFault, rFault, err := client.FaultCodes(ctx)
	if err != nil {
		t.Logf("[%s] FaultCodes error: %v", label, err)
	}

	lSpeed, err := client.Speed(ctx, zltech.Left)
	if err != nil {
		t.Logf("[%s] Speed(left) error: %v", label, err)
	}
	rSpeed, err := client.Speed(ctx, zltech.Right)
	if err != nil {
		t.Logf("[%s] Speed(right) error: %v", label, err)
	}

	lCur, err := client.Current(ctx, zltech.Left)
	if err != nil {
		t.Logf("[%s] Current(left) error: %v", label, err)
	}
	rCur, err := client.Current(ctx, zltech.Right)
	if err != nil {
		t.Logf("[%s] Current(right) error: %v", label, err)
	}

	// Temperatures
	lTemp, err := client.Temperature(ctx, zltech.Left)
	if err != nil {
		t.Logf("[%s] Temp(left) error: %v", label, err)
	}
	rTemp, err := client.Temperature(ctx, zltech.Right)
	if err != nil {
		t.Logf("[%s] Temp(right) error: %v", label, err)
	}
	ctrlTemp, err := client.ControllerTemperature(ctx)
	if err != nil {
		t.Logf("[%s] Temp(controller) error: %v", label, err)
	}

	t.Logf("[%s] Diagnostics: status=%s enabled=%t state=%s faults[L=%v(0x%04X) R=%v(0x%04X)] speed_rpm[L=%.1f R=%.1f] current_A[L=%.1f R=%.1f] temp_C[L=%.1f R=%.1f CTRL=%.1f]",
		label, status, enabled, state, lFault, uint16(lFault), rFault, uint16(rFault), lSpeed, rSpeed, lCur, rCur, lTemp, rTemp, ctrlTemp)
}

func TestFaults(t *testing.T) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		t.Skipf("CAN interface can0 not available, skipping test: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := zltech.NewWithMux(bus, mux, 0x01, zltech.WithLogger(logger), zltech.WithSyncPeriod(5*time.Millisecond))
	defer client.Close()
	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	// Requires sheldon to purposefully fault the controller
	t.Log("Phase 1: Spinning wheels at 10 RPM")
	if err := client.SetProfileVelocity(ctx, 10, 10, 10, 10); err != nil {
		t.Fatalf("SetProfileVelocity failed: %v", err)
	}
	time.Sleep(10 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")

	t.Log("Phase 2: Recover from Faults")
	if err := client.RecoverFromFault(ctx); err != nil {
		t.Fatalf("RecoverFromFault failed: %v", err)
	}
	time.Sleep(1 * time.Second)
	logDiagnostics(t, ctx, client, "recovered")

	t.Log("Phase 3: Spinning wheels at 10 RPM")
	if err := client.SetProfileVelocity(ctx, 10, 10, 10, 10); err != nil {
		t.Fatalf("SetProfileVelocity failed: %v", err)
	}
	time.Sleep(10 * time.Second)
	logDiagnostics(t, ctx, client, "spinning")
	if err := client.Halt(ctx); err != nil {
		t.Fatalf("Halt failed: %v", err)
	}
}

// BenchmarkSetProfileVelocity measures CAN dispatch cost of SetProfileVelocity.
func BenchmarkSetProfileVelocity(b *testing.B) {
	bus, err := canbus.DialSocketCAN("can0")
	if err != nil {
		b.Skipf("CAN interface can0 not available, skipping benchmark: %v", err)
	}
	defer bus.Close()
	mux := canbus.NewMux(bus)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	client := zltech.NewWithMux(bus, mux, 0x01, zltech.WithLogger(logger), zltech.WithVelocityViaRPDO())
	defer client.Close()

	ctx := context.Background()
	if err := client.Init(ctx); err != nil {
		b.Fatalf("Init failed: %v", err)
	}

	const accMs, decMs uint32 = 10, 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var rpm int32 = 10
		if i%2 == 1 {
			rpm = 0
		}
		if err := client.SetProfileVelocity(ctx, rpm, rpm, accMs, decMs); err != nil {
			b.Fatalf("SetProfileVelocity failed: %v", err)
		}
	}
}
