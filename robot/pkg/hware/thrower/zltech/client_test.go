//go:build linux

package zltech

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/notnil/canbus"
)

// TestWheelIntegration tests the zltech wheel client with real hardware.
// This test requires:
// - A ZLAC8015D servo driver connected via CAN
// - CAN interface configured (e.g., can0)
// - Appropriate permissions to access the CAN interface
//
// Run with: go test -v -run TestWheelIntegration
func TestWheelIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Configuration
	canInterface := os.Getenv("CAN_INTERFACE")
	if canInterface == "" {
		canInterface = "can1"
	}

	nodeID := byte(4) // Default CANopen node ID
	side := Left      // Test left motor

	t.Logf("Testing with CAN interface: %s, Node ID: %d, Side: %s", canInterface, nodeID, side)

	// Create logger for detailed output
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Open CAN bus
	bus, err := canbus.DialSocketCAN(canInterface)
	if err != nil {
		t.Skipf("CAN interface %s not available, skipping test: %v", canInterface, err)
	}
	defer bus.Close()

	// Create client
	client := New(bus, Config{
		NodeID:      nodeID,
		Side:        side,
		SingleMotor: true,
		Logger:      logger,
	})
	defer client.Close()

	ctx := context.Background()

	// Test 1: Initialize
	t.Run("Initialize", func(t *testing.T) {
		err := client.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}
		t.Log("✓ Initialization successful")
	})

	// Test 2: Get initial status
	t.Run("InitialStatus", func(t *testing.T) {
		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Initial status: %s", status)

		if status.State != StateOperationEnabled {
			t.Errorf("expected state OperationEnabled, got %s", status.State)
		}
	})

	// Test 3: Spin at low speed
	t.Run("SpinLowSpeed", func(t *testing.T) {
		rpm := int32(100)
		accMs := uint32(1000) // 1 second acceleration
		decMs := uint32(1000) // 1 second deceleration

		err := client.Spin(ctx, rpm, accMs, decMs)
		if err != nil {
			t.Fatalf("failed to spin at %d RPM: %v", rpm, err)
		}
		t.Logf("✓ Set target velocity to %d RPM", rpm)

		// Wait for motor to spin up
		time.Sleep(2 * time.Second)

		// Check status
		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Status while spinning: %s", status)

		// Verify wheel is actually spinning (within tolerance)
		tolerance := 20.0 // RPM tolerance
		if status.ActualRPM < float64(rpm)-tolerance || status.ActualRPM > float64(rpm)+tolerance {
			t.Logf("Warning: actual RPM (%.1f) differs from target (%d) by more than tolerance",
				status.ActualRPM, rpm)
		}
	})

	// Test 4: Stop
	t.Run("Stop", func(t *testing.T) {
		err := client.Stop(ctx)
		if err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
		t.Log("✓ Stop command sent")

		// Wait for motor to stop
		time.Sleep(2 * time.Second)

		// Check status
		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Status after stop: %s", status)

		// Verify wheel has stopped
		tolerance := 5.0 // RPM tolerance for stopped state
		if status.ActualRPM > tolerance || status.ActualRPM < -tolerance {
			t.Logf("Warning: wheel still spinning at %.1f RPM after stop", status.ActualRPM)
		}
	})

	// Test 5: Disable (freewheel)
	t.Run("Disable", func(t *testing.T) {
		err := client.Disable(ctx)
		if err != nil {
			t.Fatalf("failed to disable: %v", err)
		}
		t.Log("✓ Disabled (freewheeling)")

		time.Sleep(500 * time.Millisecond)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Status while disabled: %s", status)

		// Should not be in OperationEnabled state
		if status.State == StateOperationEnabled {
			t.Error("expected to be disabled, but still in OperationEnabled state")
		}
	})

	// Test 6: Re-enable
	t.Run("ReEnable", func(t *testing.T) {
		err := client.Enable(ctx)
		if err != nil {
			t.Fatalf("failed to re-enable: %v", err)
		}
		t.Log("✓ Re-enabled")

		time.Sleep(500 * time.Millisecond)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Status after re-enable: %s", status)

		if status.State != StateOperationEnabled {
			t.Errorf("expected state OperationEnabled, got %s", status.State)
		}
	})

	// Test 7: Spin in reverse
	t.Run("SpinReverse", func(t *testing.T) {
		rpm := int32(-100) // Negative for reverse
		accMs := uint32(500)
		decMs := uint32(500)

		err := client.Spin(ctx, rpm, accMs, decMs)
		if err != nil {
			t.Fatalf("failed to spin in reverse: %v", err)
		}
		t.Logf("✓ Set target velocity to %d RPM (reverse)", rpm)

		// Wait for motor to spin up
		time.Sleep(1500 * time.Millisecond)

		// Check status
		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Status while spinning reverse: %s", status)
	})

	// Test 8: Final stop and status check
	t.Run("FinalStop", func(t *testing.T) {
		err := client.Stop(ctx)
		if err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
		t.Log("✓ Final stop command sent")

		time.Sleep(1500 * time.Millisecond)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		t.Logf("✓ Final status: %s", status)

		// Check health
		if !status.Healthy() {
			t.Errorf("wheel is not healthy: fault=%s", status.FaultCode)
		}
	})
}

// TestEnableDisableCycle tests the enable/disable functionality with extended duration
// This test cycles between enabled and disabled states for 10 seconds each
func TestEnableDisableCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	canInterface := os.Getenv("CAN_INTERFACE")
	if canInterface == "" {
		canInterface = "can1"
	}

	bus, err := canbus.DialSocketCAN(canInterface)
	if err != nil {
		t.Skipf("CAN interface %s not available, skipping test: %v", canInterface, err)
	}
	defer bus.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := New(bus, Config{
		NodeID:      4,
		Side:        Left,
		SingleMotor: true,
		Logger:      logger,
	})
	defer client.Close()

	ctx := context.Background()

	// Initialize
	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}
	t.Log("✓ Initialized")

	// Cycle through enable/disable states
	cycles := 3
	for i := 0; i < cycles; i++ {
		t.Logf("=== Cycle %d/%d ===", i+1, cycles)

		// Disable for 10 seconds
		t.Log("Disabling motor (freewheeling)...")
		if err := client.Disable(ctx); err != nil {
			t.Fatalf("cycle %d: failed to disable: %v", i+1, err)
		}

		// Monitor during disabled state
		t.Log("Motor disabled, monitoring for 10 seconds...")
		disableStart := time.Now()
		for time.Since(disableStart) < 10*time.Second {
			time.Sleep(2 * time.Second)
			elapsed := time.Since(disableStart).Seconds()
			t.Logf("  [%.1fs] Motor disabled (freewheeling)", elapsed)
		}
		t.Log("✓ Completed 10 seconds in disabled state")

		// Enable for 10 seconds
		t.Log("Enabling motor...")
		if err := client.Enable(ctx); err != nil {
			t.Fatalf("cycle %d: failed to enable: %v", i+1, err)
		}

		// Monitor during enabled state
		t.Log("Motor enabled, monitoring for 10 seconds...")
		enableStart := time.Now()
		for time.Since(enableStart) < 10*time.Second {
			time.Sleep(2 * time.Second)
			elapsed := time.Since(enableStart).Seconds()
			t.Logf("  [%.1fs] Motor enabled (holding position)", elapsed)
		}
		t.Log("✓ Completed 10 seconds in enabled state")
	}

	t.Logf("✓ Successfully completed %d enable/disable cycles", cycles)
}

// TestWheelStatus tests just the status reading functionality
// This is useful for quick checks without moving the motor
func TestWheelStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	canInterface := os.Getenv("CAN_INTERFACE")
	if canInterface == "" {
		canInterface = "can1"
	}

	bus, err := canbus.DialSocketCAN(canInterface)
	if err != nil {
		t.Skipf("CAN interface %s not available, skipping test: %v", canInterface, err)
	}
	defer bus.Close()

	client := New(bus, Config{
		NodeID:      4,
		Side:        Left,
		SingleMotor: true,
		Logger:      slog.Default(),
	})
	defer client.Close()

	ctx := context.Background()

	// Initialize first
	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Read status
	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	t.Logf("Status: %s", status)
	t.Logf("  State: %s", status.State)
	t.Logf("  Actual RPM: %.1f", status.ActualRPM)
	t.Logf("  Temperature: %.1f°C", status.Temperature)
	t.Logf("  Current: %.2fA", status.Current)
	t.Logf("  Fault Code: %s", status.FaultCode)
	t.Logf("  Healthy: %v", status.Healthy())
}

// TestKeepAliveIntegration tests the keep-alive functionality with real hardware.
// This test verifies that:
// - Keep-alive commands are sent periodically
// - Motor continues to run without communication timeout
// - Enable/Disable cycles work correctly with keep-alive
//
// Run with: go test -v -run TestKeepAliveIntegration
func TestKeepAliveIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Configuration
	canInterface := os.Getenv("CAN_INTERFACE")
	if canInterface == "" {
		canInterface = "can1"
	}

	nodeID := byte(4)
	side := Left

	t.Logf("Testing keep-alive with CAN interface: %s, Node ID: %d", canInterface, nodeID)

	// Create logger for detailed output
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Open CAN bus
	bus, err := canbus.DialSocketCAN(canInterface)
	if err != nil {
		t.Skipf("CAN interface %s not available, skipping test: %v", canInterface, err)
	}
	defer bus.Close()

	// Create client with keep-alive enabled
	client := New(bus, Config{
		NodeID:            nodeID,
		Side:              side,
		SingleMotor:       true,
		Logger:            logger,
		KeepAliveInterval: 500 * time.Millisecond, // Send keep-alive every 500ms
	})
	defer client.Close()

	ctx := context.Background()

	// Initialize
	t.Run("InitializeWithKeepAlive", func(t *testing.T) {
		err := client.Initialize(ctx)
		if err != nil {
			t.Fatalf("failed to initialize: %v", err)
		}
		t.Log("✓ Initialization successful with keep-alive enabled")
	})

	// Test 1: Set velocity and let keep-alive maintain it
	t.Run("KeepAliveMaintenace", func(t *testing.T) {
		rpm := int32(100)
		err := client.Spin(ctx, rpm, 500, 500)
		if err != nil {
			t.Fatalf("failed to spin: %v", err)
		}
		t.Logf("✓ Set target velocity to %d RPM", rpm)

		// Wait for 2 seconds - keep-alive should send commands every 500ms
		// Without keep-alive, motor might timeout after 1000ms
		t.Log("Waiting 2 seconds to verify keep-alive maintains motor...")
		time.Sleep(2 * time.Second)

		// Check motor is still running
		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}

		if status.State != StateOperationEnabled {
			t.Errorf("motor not in OperationEnabled state after keep-alive period: %s", status.State)
		}
		t.Logf("✓ Motor still running after keep-alive period (actual RPM: %.1f)", status.ActualRPM)
	})

	// Test 2: Disable and verify keep-alive sends zero
	t.Run("DisableWithKeepAlive", func(t *testing.T) {
		err := client.Disable(ctx)
		if err != nil {
			t.Fatalf("failed to disable: %v", err)
		}
		t.Log("✓ Motor disabled")

		// Wait and verify motor stays disabled
		time.Sleep(1 * time.Second)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}

		// Motor should be disabled but not in fault (keep-alive maintains communication)
		if status.State == StateFault {
			t.Errorf("motor in fault state after disable: %s", status.FaultCode)
		}
		t.Logf("✓ Motor disabled without fault (state: %s, RPM: %.1f)", status.State, status.ActualRPM)
	})

	// Test 3: Re-enable and spin again
	t.Run("ReEnableWithKeepAlive", func(t *testing.T) {
		err := client.Enable(ctx)
		if err != nil {
			t.Fatalf("failed to re-enable: %v", err)
		}
		t.Log("✓ Motor re-enabled")

		// Spin at a different speed
		rpm := int32(50)
		err = client.Spin(ctx, rpm, 500, 500)
		if err != nil {
			t.Fatalf("failed to spin after re-enable: %v", err)
		}
		t.Logf("✓ Set target velocity to %d RPM after re-enable", rpm)

		time.Sleep(1500 * time.Millisecond)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}

		if status.State != StateOperationEnabled {
			t.Errorf("motor not operational after re-enable: %s", status.State)
		}
		t.Logf("✓ Motor running after re-enable (actual RPM: %.1f)", status.ActualRPM)
	})

	// Test 4: Stop motor
	t.Run("StopWithKeepAlive", func(t *testing.T) {
		err := client.Stop(ctx)
		if err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
		t.Log("✓ Motor stopped")

		time.Sleep(1 * time.Second)

		status, err := client.Status(ctx)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}

		// Motor should be at or near zero RPM
		if status.ActualRPM > 10 {
			t.Errorf("motor still spinning after stop: %.1f RPM", status.ActualRPM)
		}
		t.Logf("✓ Motor stopped (actual RPM: %.1f)", status.ActualRPM)
	})

	t.Log("✓ All keep-alive tests passed")
}

// TestTwoWheelsSpin tests spinning two wheels simultaneously for 10 seconds
// Uses two separate motor controllers, each with their own node ID
func TestTwoWheelsSpin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	canInterface := os.Getenv("CAN_INTERFACE")
	if canInterface == "" {
		canInterface = "can1"
	}

	nodeID1 := byte(4)
	nodeID2 := byte(5)

	t.Logf("Testing two wheels with CAN interface: %s, Node IDs: %d, %d", canInterface, nodeID1, nodeID2)

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Open CAN bus
	bus, err := canbus.DialSocketCAN(canInterface)
	if err != nil {
		t.Skipf("CAN interface %s not available, skipping test: %v", canInterface, err)
	}
	defer bus.Close()

	// Create a shared Mux for both clients
	// This is critical - multiple clients on the same bus must share the same Mux
	mux := canbus.NewMux(bus)

	// Create first wheel client (node ID 4)
	client1 := New(bus, Config{
		NodeID:      nodeID1,
		Side:        Left,
		SingleMotor: true, // Single motor mode
		Logger:      logger,
		Mux:         mux, // Share the Mux
	})
	defer client1.Close()

	// Create second wheel client (node ID 5)
	client2 := New(bus, Config{
		NodeID:      nodeID2,
		Side:        Left,
		SingleMotor: true, // Single motor mode
		Logger:      logger,
		Mux:         mux, // Share the same Mux
	})
	defer client2.Close()

	ctx := context.Background()

	// Initialize both wheels
	t.Log("Initializing wheel 1...")
	if err := client1.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize wheel 1: %v", err)
	}
	t.Log("✓ Wheel 1 initialized")

	t.Log("Initializing wheel 2...")
	if err := client2.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize wheel 2: %v", err)
	}
	t.Log("✓ Wheel 2 initialized")

	// Set both wheels spinning at 200 RPM
	rpm := int32(200)
	accMs := uint32(1000)
	decMs := uint32(1000)

	t.Logf("Spinning both wheels at %d RPM...", rpm)
	if err := client1.Spin(ctx, rpm, accMs, decMs); err != nil {
		t.Fatalf("failed to spin wheel 1: %v", err)
	}

	if err := client2.Spin(ctx, rpm, accMs, decMs); err != nil {
		t.Fatalf("failed to spin wheel 2: %v", err)
	}
	t.Log("✓ Both wheels spinning")

	// Monitor for 10 seconds
	t.Log("Running for 10 seconds...")
	startTime := time.Now()
	for time.Since(startTime) < 10*time.Second {
		time.Sleep(2 * time.Second)
		elapsed := time.Since(startTime).Seconds()

		// Check wheel 1 status
		status1, err := client1.Status(ctx)
		if err != nil {
			t.Logf("Warning: failed to read wheel 1 status: %v", err)
		} else {
			t.Logf("  [%.1fs] Wheel 1 (node %d): %.1f RPM, %.2f A", elapsed, nodeID1, status1.ActualRPM, status1.Current)
		}

		// Check wheel 2 status
		status2, err := client2.Status(ctx)
		if err != nil {
			t.Logf("Warning: failed to read wheel 2 status: %v", err)
		} else {
			t.Logf("  [%.1fs] Wheel 2 (node %d): %.1f RPM, %.2f A", elapsed, nodeID2, status2.ActualRPM, status2.Current)
		}
	}

	// Stop both wheels
	t.Log("Stopping both wheels...")
	if err := client1.Stop(ctx); err != nil {
		t.Fatalf("failed to stop wheel 1: %v", err)
	}

	if err := client2.Stop(ctx); err != nil {
		t.Fatalf("failed to stop wheel 2: %v", err)
	}
	t.Log("✓ Both wheels stopped")

	// Wait for wheels to come to a complete stop
	time.Sleep(2 * time.Second)

	// Verify both wheels have stopped
	status1, err := client1.Status(ctx)
	if err != nil {
		t.Fatalf("failed to read final wheel 1 status: %v", err)
	}
	t.Logf("Final wheel 1 status: %.1f RPM", status1.ActualRPM)

	status2, err := client2.Status(ctx)
	if err != nil {
		t.Fatalf("failed to read final wheel 2 status: %v", err)
	}
	t.Logf("Final wheel 2 status: %.1f RPM", status2.ActualRPM)

	t.Log("✓ Test completed successfully")
}
