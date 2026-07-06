// Package loader provides utilities for loading and executing drill plugins.
// It handles the runtime loading of .so files and symbol resolution.
package loader

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"plugin"

	"github.com/notnil/tensa/pkg/ai/drillsx/api"
)

// Loader is responsible for loading and executing drill plugins.
type Loader struct {
	log *slog.Logger
}

// New creates a new Loader with the given logger.
func New(log *slog.Logger) *Loader {
	return &Loader{log: log}
}

// Load loads a drill plugin from the specified .so file path.
// It returns the Drill interface that can be used to execute the drill.
func (l *Loader) Load(path string) (api.Drill, error) {
	l.log.Info("loading drill plugin", "path", path)

	// Open the plugin file
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %q: %w", path, err)
	}

	// Look up the "Drill" symbol
	sym, err := p.Lookup("Drill")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup 'Drill' symbol in plugin %q: %w", path, err)
	}

	// The plugin.Lookup returns a pointer to the exported variable.
	// If the plugin exports "var Drill api.Drill", we get *api.Drill.
	// We need to dereference it to get the actual interface value.
	drillPtr, ok := sym.(*api.Drill)
	if !ok {
		return nil, fmt.Errorf("symbol 'Drill' in plugin %q is not of type *api.Drill (type: %T)", path, sym)
	}

	// Dereference to get the actual drill implementation
	drill := *drillPtr
	if drill == nil {
		return nil, fmt.Errorf("symbol 'Drill' in plugin %q is nil", path)
	}

	l.log.Info("successfully loaded drill plugin", "path", path)
	return drill, nil
}

// Execute loads a drill plugin from the specified path and executes it with the provided runtime.
// This is a convenience method that combines Load and Run.
func (l *Loader) Execute(ctx context.Context, path string, rt api.Runtime) error {
	drill, err := l.Load(path)
	if err != nil {
		return err
	}

	l.log.Info("executing drill", "path", path)
	if err := drill.Run(ctx, rt); err != nil {
		return fmt.Errorf("drill execution failed: %w", err)
	}

	l.log.Info("drill execution completed", "path", path)
	return nil
}

// ExecuteWithRand loads a drill plugin from the specified path and executes it with the provided runtime.
// Deprecated: Use Execute instead. The Runtime struct now contains the random number generator.
func (l *Loader) ExecuteWithRand(ctx context.Context, path string, rt api.Runtime, rnd *rand.Rand) error {
	drill, err := l.Load(path)
	if err != nil {
		return err
	}

	l.log.Info("executing drill", "path", path)
	if err := drill.Run(ctx, rt); err != nil {
		return fmt.Errorf("drill execution failed: %w", err)
	}

	l.log.Info("drill execution completed", "path", path)
	return nil
}
