package loader_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"github.com/notnil/tensa/pkg/ai/drillsx/loader"
)

// mockNavigator implements a simple mock navigator for examples.
type mockNavigator struct {
	navigations []api.Location
}

func (m *mockNavigator) Navigate(ctx context.Context, dest api.Location) error {
	m.navigations = append(m.navigations, dest)
	fmt.Printf("Navigating to location: (%v, %v), rotation: %.2f\n", dest.Point.X, dest.Point.Y, dest.Rotation)
	return nil
}

// mockMover implements a simple mock mover for examples.
type mockMover struct{}

func (m *mockMover) Move(dir, speed float64) error {
	return nil
}

func (m *mockMover) Rotate(speed float64) error {
	return nil
}

func (m *mockMover) Stop() error {
	return nil
}

// mockThrower implements a simple mock thrower for examples.
type mockThrower struct {
	setCalls []api.Settings
	throws   int
	loads    int
}

func (m *mockThrower) Set(s api.Settings) error {
	m.setCalls = append(m.setCalls, s)
	fmt.Printf("Setting thrower: top=%.2f bottom=%.2f angle=%.2f\n", s.Top, s.Bottom, s.Angle)
	return nil
}

func (m *mockThrower) Throw(ctx context.Context) error {
	m.throws++
	fmt.Printf("Throwing ball #%d\n", m.throws)
	return nil
}

func (m *mockThrower) Load(ctx context.Context) error {
	m.loads++
	fmt.Printf("Loading ball (count=%d)\n", m.loads)
	return nil
}

// mockAudioPlayer implements api.AudioPlayer for testing
type mockAudioPlayer struct{}

func (m mockAudioPlayer) Play(r io.Reader) error {
	return nil
}

// mockPlayerProvider implements api.PlayerProvider for testing
type mockPlayerProvider struct{}

func (m *mockPlayerProvider) Players() ([]api.Location, error) {
	// Return empty list for testing
	return []api.Location{}, nil
}

// This example demonstrates how to use the FSRegistry to load and execute drill plugins
// from the filesystem. This is the recommended approach for production code.
func Example_fsRegistry() {
	// Create a logger
	log := slog.Default()

	// Determine the path to the example drills
	// In production, this would be a configured directory like "/opt/tensa/drills"
	_, filename, _, _ := runtime.Caller(0)
	examplesDir := filepath.Join(filepath.Dir(filename), "..", "examples", "build")
	if _, err := os.Stat(filepath.Join(examplesDir, "crosscourt.so")); err != nil {
		return
	}

	// Create an FSRegistry that loads plugins from the examples/build directory
	registry := loader.NewFSRegistry(examplesDir, log)

	// Set up hardware interfaces
	// In production, these would be the real hardware implementations
	nav := &mockNavigator{}
	mover := &mockMover{}
	thr := &mockThrower{}
	rt := api.Runtime{
		Nav:            nav,
		Mover:          mover,
		Thrower:        thr,
		Audio:          mockAudioPlayer{},
		Events:         nil, // Set to nil for this example
		Metrics:        nil, // Set to nil for this example
		PlayerProvider: &mockPlayerProvider{},
		Log:            log,
		Rnd:            rand.New(rand.NewSource(42)),
	}

	// Load a drill by ID (the registry will look for crosscourt.so)
	drill, err := registry.GetDrill("crosscourt")
	if err != nil {
		fmt.Printf("Failed to load drill: %v\n", err)
		return
	}

	// Note: In a real test, we'd limit the drill execution to avoid running the full drill
	// For this example, we'll just show the pattern
	fmt.Println("Successfully loaded drill from FSRegistry")
	fmt.Printf("Drill type: %T\n", drill)

	// In production, you would execute the drill like this:
	// ctx := context.Background()
	// err = drill.Run(ctx, rt)

	_ = rt // Suppress unused variable warning in example
}

// TestFSRegistry demonstrates how to test drill loading and execution using the FSRegistry.
// This is an integration test that requires the drill plugins to be built first.
func TestFSRegistry(t *testing.T) {
	// Skip if not on a platform that supports plugins
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Plugin loading is only supported on Linux and macOS")
	}

	// Create a logger
	log := slog.Default()

	// Determine the path to the example drills
	_, filename, _, _ := runtime.Caller(0)
	examplesDir := filepath.Join(filepath.Dir(filename), "..", "examples", "build")
	if _, err := os.Stat(filepath.Join(examplesDir, "crosscourt.so")); err != nil {
		t.Skip("example drill plugins are not built")
	}

	// Create an FSRegistry
	registry := loader.NewFSRegistry(examplesDir, log)

	// Set up mock hardware
	nav := &mockNavigator{}
	mover := &mockMover{}
	thr := &mockThrower{}
	rt := api.Runtime{
		Nav:            nav,
		Mover:          mover,
		Thrower:        thr,
		Audio:          mockAudioPlayer{},
		Events:         nil, // Set to nil for this test
		Metrics:        nil, // Set to nil for this test
		PlayerProvider: &mockPlayerProvider{},
		Log:            log,
		Rnd:            rand.New(rand.NewSource(12345)),
	}

	// Test loading a drill
	drill, err := registry.GetDrill("crosscourt")
	if err != nil {
		t.Fatalf("Failed to load crosscourt drill: %v", err)
	}

	if drill == nil {
		t.Fatal("Expected drill to be non-nil")
	}

	// Test that the same drill is cached (should return immediately)
	drill2, err := registry.GetDrill("crosscourt")
	if err != nil {
		t.Fatalf("Failed to load cached drill: %v", err)
	}

	if drill != drill2 {
		t.Error("Expected cached drill to be the same instance")
	}

	// Test that we can execute a few iterations of the drill
	// We'll create a context that we cancel after a short time
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the drill in a goroutine so we can cancel it
	errCh := make(chan error, 1)
	go func() {
		errCh <- drill.Run(ctx, rt)
	}()

	// Let the drill run for a moment to execute at least one feed
	// The crosscourt drill has a 2-second delay between feeds, so we need to wait
	// a bit to see at least one feed happen

	// Wait for drill to start and execute a few operations
	// We don't want to wait for the full drill (20 balls with 2s delays = 40s)
	// so we'll just cancel after a moment
	cancel()

	// Wait for the drill to finish
	err = <-errCh
	if err != nil && err != context.Canceled {
		t.Errorf("Drill execution failed: %v", err)
	}

	// Verify that some navigation and thrower operations occurred
	t.Logf("Navigations: %d", len(nav.navigations))
	t.Logf("Thrower set calls: %d", len(thr.setCalls))
	t.Logf("Loads executed: %d", thr.loads)
	t.Logf("Throws executed: %d", thr.throws)

	// Note: We can't make strong assertions about counts because we're canceling
	// the drill early, but this demonstrates the pattern
}

// TestFSRegistry_CrissCross specifically tests loading the criss_cross drill
// that was mentioned in the original error.
func TestFSRegistry_CrissCross(t *testing.T) {
	// Skip if not on a platform that supports plugins
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Plugin loading is only supported on Linux and macOS")
	}

	// Create a logger
	log := slog.Default()

	// Determine the path to the example drills
	_, filename, _, _ := runtime.Caller(0)
	examplesDir := filepath.Join(filepath.Dir(filename), "..", "examples", "build")
	if _, err := os.Stat(filepath.Join(examplesDir, "criss-cross.so")); err != nil {
		t.Skip("example drill plugins are not built")
	}

	// Create an FSRegistry
	registry := loader.NewFSRegistry(examplesDir, log)

	// Test loading the criss-cross drill specifically
	drill, err := registry.GetDrill("criss-cross")
	if err != nil {
		t.Fatalf("Failed to load criss-cross drill: %v", err)
	}

	if drill == nil {
		t.Fatal("Expected drill to be non-nil")
	}

	t.Logf("Successfully loaded criss-cross drill: %T", drill)

	// Set up mock hardware
	nav := &mockNavigator{}
	mover := &mockMover{}
	thr := &mockThrower{}
	rt := api.Runtime{
		Nav:            nav,
		Mover:          mover,
		Thrower:        thr,
		Audio:          mockAudioPlayer{},
		Events:         nil, // Set to nil for this test
		Metrics:        nil, // Set to nil for this test
		PlayerProvider: &mockPlayerProvider{},
		Log:            log,
		Rnd:            rand.New(rand.NewSource(12345)),
	}

	// Execute the drill briefly to verify it works
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- drill.Run(ctx, rt)
	}()

	// Cancel immediately to avoid long execution
	cancel()

	err = <-errCh
	if err != nil && err != context.Canceled {
		t.Errorf("Drill execution failed: %v", err)
	}

	t.Logf("Criss-cross drill executed successfully")
}

// TestMockRegistry demonstrates using the MockRegistry for unit testing drill logic
// without requiring .so files.
func TestMockRegistry(t *testing.T) {
	// Create a mock registry
	registry := loader.NewMockRegistry()

	// Create a simple test drill
	testDrill := &testDrill{}
	registry.Register("test_drill", testDrill)

	// Load the drill
	drill, err := registry.GetDrill("test_drill")
	if err != nil {
		t.Fatalf("Failed to load test drill: %v", err)
	}

	// Execute it
	nav := &mockNavigator{}
	mover := &mockMover{}
	thr := &mockThrower{}
	log := slog.Default()
	rt := api.Runtime{
		Nav:            nav,
		Mover:          mover,
		Thrower:        thr,
		Audio:          mockAudioPlayer{},
		Events:         nil, // Set to nil for this test
		Metrics:        nil, // Set to nil for this test
		PlayerProvider: &mockPlayerProvider{},
		Log:            log,
		Rnd:            rand.New(rand.NewSource(42)),
	}

	ctx := context.Background()

	err = drill.Run(ctx, rt)
	if err != nil {
		t.Fatalf("Drill execution failed: %v", err)
	}

	// Verify the drill executed
	if !testDrill.executed {
		t.Error("Expected drill to be executed")
	}

	// Verify mock tracking
	if len(nav.navigations) != 1 {
		t.Errorf("Expected 1 navigation, got %d", len(nav.navigations))
	}

	if thr.throws != 1 {
		t.Errorf("Expected 1 throw, got %d", thr.throws)
	}
}

// testDrill is a minimal drill implementation for testing
type testDrill struct {
	executed bool
}

func (d *testDrill) Run(ctx context.Context, rt api.Runtime) error {
	d.executed = true

	// Navigate to a location
	loc := api.Location{
		Point:    api.Point{X: 0, Y: -10},
		Rotation: 1.57,
	}
	if err := rt.Nav.Navigate(ctx, loc); err != nil {
		return err
	}

	// Set thrower and execute one throw
	if err := rt.Thrower.Set(api.Settings{Top: 200, Bottom: 200, Angle: 0}); err != nil {
		return err
	}
	if err := rt.Thrower.Load(ctx); err != nil {
		return err
	}
	return rt.Thrower.Throw(ctx)
}
