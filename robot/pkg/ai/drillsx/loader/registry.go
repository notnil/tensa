// Package loader provides utilities for loading and executing drill plugins.
package loader

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

// Registry defines the interface for retrieving drill implementations by their identifier.
// Implementations may load drills from various sources such as filesystem plugins,
// in-memory maps, or remote registries.
type Registry interface {
	// GetDrill retrieves a drill by its unique identifier.
	// Returns an error if the drill is not found or cannot be loaded.
	GetDrill(id string) (api.Drill, error)
}

// MockRegistry is an in-memory implementation of Registry for testing.
// It stores drills in a map and allows programmatic registration.
type MockRegistry struct {
	drills map[string]api.Drill
}

// NewMockRegistry creates a new MockRegistry with an empty drill map.
func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		drills: make(map[string]api.Drill),
	}
}

// Register adds a drill to the registry with the specified ID.
// This method is useful for setting up test scenarios.
func (m *MockRegistry) Register(id string, drill api.Drill) {
	m.drills[id] = drill
}

// GetDrill retrieves a drill by its ID from the in-memory map.
// Returns an error if the drill is not found.
func (m *MockRegistry) GetDrill(id string) (api.Drill, error) {
	drill, exists := m.drills[id]
	if !exists {
		return nil, fmt.Errorf("drill not found: %s", id)
	}
	return drill, nil
}

// FSRegistry is a filesystem-based implementation of Registry.
// It loads drill plugins from .so files located in a specified directory.
// The expected file naming convention is <drill-id>.so.
type FSRegistry struct {
	baseDir string
	loader  *Loader
	cache   map[string]api.Drill
	mu      sync.RWMutex
}

// NewFSRegistry creates a new FSRegistry that loads plugins from the specified base directory.
// The logger is used for logging plugin loading operations.
func NewFSRegistry(baseDir string, log *slog.Logger) *FSRegistry {
	return &FSRegistry{
		baseDir: baseDir,
		loader:  New(log),
		cache:   make(map[string]api.Drill),
	}
}

// GetDrill retrieves a drill by its ID by loading the corresponding .so plugin file.
// The plugin file is expected to be located at <baseDir>/<id>.so.
// Loaded drills are cached to avoid reloading the same plugin multiple times.
func (f *FSRegistry) GetDrill(id string) (api.Drill, error) {
	// Check cache first
	f.mu.RLock()
	if drill, exists := f.cache[id]; exists {
		f.mu.RUnlock()
		return drill, nil
	}
	f.mu.RUnlock()

	// Construct the plugin path
	pluginPath := filepath.Join(f.baseDir, fmt.Sprintf("%s.so", id))

	// Load the plugin
	drill, err := f.loader.Load(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load drill %q: %w", id, err)
	}

	// Cache the loaded drill
	f.mu.Lock()
	f.cache[id] = drill
	f.mu.Unlock()

	return drill, nil
}
