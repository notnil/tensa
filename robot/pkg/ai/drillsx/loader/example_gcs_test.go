package loader_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/notnil/tensa/pkg/ai/drillsx/loader"
)

// ExampleGCSRegistry demonstrates the typical workflow for using GCSRegistry.
func ExampleGCSRegistry() {
	ctx := context.Background()
	log := slog.Default()

	// Create a GCS registry with a persistent cache directory
	registry, err := loader.NewGCSRegistry(
		ctx,
		"gs://example-public-bucket/tensa/drills/plugins",
		"/var/cache/drills",
		log,
	)
	if err != nil {
		fmt.Printf("Failed to create registry: %v\n", err)
		return
	}
	defer registry.Close()

	// Sync all plugins from GCS to local cache
	// This is typically done once at startup or on-demand
	if err := registry.Sync(ctx); err != nil {
		fmt.Printf("Failed to sync plugins: %v\n", err)
		return
	}

	// Load a drill by ID (from local cache)
	drill, err := registry.GetDrill("crosscourt")
	if err != nil {
		fmt.Printf("Failed to load drill: %v\n", err)
		return
	}

	// The drill is now ready to execute
	_ = drill // Use the drill for execution
	fmt.Println("Successfully loaded drill from GCS registry")
}

// Example_gCSRegistry_offlineMode demonstrates using GCSRegistry in offline mode.
// If plugins have been synced previously, the cache directory persists across reboots
// and you can load drills without calling Sync() again.
func Example_gCSRegistry_offlineMode() {
	ctx := context.Background()
	log := slog.Default()

	// Create registry with existing cache directory
	// No need to call Sync() if plugins were already downloaded
	registry, err := loader.NewGCSRegistry(
		ctx,
		"gs://example-public-bucket/tensa/drills/plugins",
		"/var/cache/drills",
		log,
	)
	if err != nil {
		fmt.Printf("Failed to create registry: %v\n", err)
		return
	}
	defer registry.Close()

	// Load drill directly from cache (no network call)
	drill, err := registry.GetDrill("crosscourt")
	if err != nil {
		// If this fails, you might need to call Sync() first
		fmt.Printf("Failed to load drill (try calling Sync first): %v\n", err)
		return
	}

	_ = drill
	fmt.Println("Successfully loaded drill from local cache")
}

// Example_gCSRegistry_updatePlugins shows how to update plugins by calling Sync again.
// Sync will only download files that have changed (based on size comparison).
func Example_gCSRegistry_updatePlugins() {
	ctx := context.Background()
	log := slog.Default()

	registry, err := loader.NewGCSRegistry(
		ctx,
		"gs://example-public-bucket/tensa/drills/plugins",
		"/var/cache/drills",
		log,
	)
	if err != nil {
		fmt.Printf("Failed to create registry: %v\n", err)
		return
	}
	defer registry.Close()

	// Initial sync
	if err := registry.Sync(ctx); err != nil {
		fmt.Printf("Failed to sync plugins: %v\n", err)
		return
	}

	// ... time passes, plugins are updated in GCS ...

	// Sync again to get updates
	// Only changed files will be downloaded
	if err := registry.Sync(ctx); err != nil {
		fmt.Printf("Failed to sync updates: %v\n", err)
		return
	}

	fmt.Println("Plugins updated successfully")
}

// Example_gCSRegistry_customCachePath shows how to use a custom cache directory.
// This is useful for controlling where plugins are stored on your system.
func Example_gCSRegistry_customCachePath() {
	ctx := context.Background()
	log := slog.Default()

	// Use a custom cache directory
	cacheDir := "/opt/tensa/drill-plugins"

	// Ensure the directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("Failed to create cache dir: %v\n", err)
		return
	}

	registry, err := loader.NewGCSRegistry(
		ctx,
		"gs://example-public-bucket/tensa/drills/plugins",
		cacheDir,
		log,
	)
	if err != nil {
		fmt.Printf("Failed to create registry: %v\n", err)
		return
	}
	defer registry.Close()

	if err := registry.Sync(ctx); err != nil {
		fmt.Printf("Failed to sync plugins: %v\n", err)
		return
	}

	fmt.Println("Plugins synced to custom directory")
}
