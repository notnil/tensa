# Drillsx - Tennis Drill Plugin System

A plugin-based system for authoring and executing tennis drills on the robotic tennis ball machine. Drills are compiled as Go plugins (`.so` files) and loaded dynamically at runtime, enabling independent development and deployment of new drill routines.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Package Structure](#package-structure)
- [Design Philosophy](#design-philosophy)
- [Quick Start](#quick-start)
- [Building Plugins](#building-plugins)
- [Plugin Registries](#plugin-registries)
- [Adding New Features](#adding-new-features)
- [Rebuild Requirements](#rebuild-requirements)
- [Platform Support](#platform-support)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

The drillsx system consists of three main components:

1. **API Package** (`api/`) - Pure interface boundary with zero external dependencies
2. **Drillutil Package** (`drillutil/`) - Optional utility functions for plugin authors
3. **Loader Package** (`loader/`) - Plugin loading infrastructure with multiple registry implementations

### Key Innovation: Dependency Isolation

The system uses **interface redeclaration** to achieve true plugin independence:

```
┌─────────────────────────────────────────────────────────────┐
│ Main Application (tensactl)                                  │
│  Uses concrete types: navigation.Navigator, thrower.Thrower │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓ Passes interfaces
┌─────────────────────────────────────────────────────────────┐
│ API Boundary (pkg/ai/drillsx/api)                           │
│  Redeclared interfaces: api.Navigator, api.Thrower          │
│  100% stdlib dependencies only!                              │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ↓ Plugin imports api package
┌─────────────────────────────────────────────────────────────┐
│ Drill Plugin (.so file)                                     │
│  Uses api.Runtime interface                                  │
│  Independent from main app dependencies                      │
└─────────────────────────────────────────────────────────────┘
```

**Result**: Plugins don't need to match internal package versions - they only need to implement the stable stdlib-based API!

## Package Structure

```
pkg/ai/drillsx/
├── README.md                    # This file
├── api/                         # Pure API boundary (100% stdlib)
│   ├── api.go                  # Drill interface, Runtime struct
│   ├── types.go                # Mirrored types and interfaces
│   ├── event.go                # Event types for shot tracking
│   └── shot_tracker.go         # Shot tracking implementation
│
├── drillutil/                   # Optional utilities
│   ├── helpers.go              # Sampling functions
│   ├── prepare.go              # PrepareThrow helper
│   ├── convert.go              # Type conversion utilities
│   └── README.md               # Drillutil documentation
│
├── loader/                      # Plugin loading infrastructure
│   ├── loader.go               # Core loading logic
│   ├── registry.go             # Registry interface
│   ├── gcs_registry.go         # Google Cloud Storage registry
│   └── example_test.go         # Usage examples
│
├── examples/                    # Example drill plugins
│   ├── <uuid>/main.go          # Individual drill implementations
│   ├── build/                  # Compiled .so files
│   ├── Makefile                # Build system
│   └── rebuild-all.sh          # Rebuild script for compatibility
│
└── drills.json                  # Drill metadata
```

## Design Philosophy

### API = Boundary Contract

The `api` package defines the **contract** between the main application and plugins:

- **Stability First**: Only stdlib dependencies
- **Interface Only**: No concrete implementations
- **Minimal Surface**: Only what plugins need to see
- **Version Independent**: Plugins work across updates

### Drillutil = Optional Convenience

The `drillutil` package provides **optional helpers** for plugin authors:

- **Not Required**: Plugins can skip it entirely
- **Can Use External Deps**: Uses `golang.org/x/sync/errgroup`
- **Independently Versioned**: Plugins can vendor different versions
- **Easy to Replace**: Plugins can implement their own helpers

## Quick Start

### 1. Create a New Drill Plugin

```bash
cd pkg/ai/drillsx/examples
mkdir my-drill
cd my-drill
```

Create `main.go`:

```go
package main

import (
    "context"
    "time"
    
    "github.com/notnil/tensa/pkg/ai/drillsx/api"
    "github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
)

type myDrill struct{}

func (d *myDrill) Run(ctx context.Context, rt api.Runtime) error {
    rt.Log.Info("Starting my drill")
    
    // Use helper to create location
    loc := drillutil.MakeLoc(api.Point{X: 0, Y: -10}, 1.57)
    
    // Navigate to position
    if err := rt.Nav.Navigate(ctx, loc); err != nil {
        return err
    }
    
    // Configure thrower
    if err := rt.Thrower.Set(api.Settings{
        Top: 200, Bottom: 200, Angle: 0.5,
    }); err != nil {
        return err
    }
    
    // Load and throw ball
    if err := rt.Thrower.Load(ctx); err != nil {
        return err
    }
    
    return rt.Thrower.Throw(ctx)
}

// Drill is the exported symbol that plugins must provide
var Drill api.Drill = &myDrill{}
```

### 2. Build the Plugin

```bash
cd ../  # Back to examples/
# Add your drill to Makefile PLUGINS list
make my-drill
```

### 3. Load and Execute

```go
import "github.com/notnil/tensa/pkg/ai/drillsx/loader"

registry := loader.NewFSRegistry("examples/build/", log)
drill, _ := registry.GetDrill("my-drill")
drill.Run(ctx, runtime)
```

## Building Plugins

### Local Development (macOS/Linux)

```bash
cd pkg/ai/drillsx/examples

# Build single plugin
make my-drill

# Build all plugins
make all

# Clean build artifacts
make clean
```

### For Jetson (ARM64 Linux)

```bash
cd pkg/ai/drillsx/examples

# Build for Jetson
make jetson

# Or build all for Jetson
GOOS=linux GOARCH=arm64 make all
```

### Build Flags

Plugins are built with:
```bash
CGO_ENABLED=1 go build \
  -buildmode=plugin \
  -ldflags="-s -w" \
  -o drill.so \
  ./path/to/drill
```

- `-buildmode=plugin`: Compile as Go plugin
- `-ldflags="-s -w"`: Strip symbols for smaller size
- `CGO_ENABLED=1`: Required for plugin support

## Plugin Registries

### Filesystem Registry

Load plugins from local directory:

```go
registry := loader.NewFSRegistry("/opt/tensa/drills", log)
drill, err := registry.GetDrill("crosscourt")
```

### Google Cloud Storage Registry

Load plugins from GCS with local caching. Supports both uncompressed `.so` and gzip-compressed `.so.gz` files (automatically decompressed during download):

```go
registry, err := loader.NewGCSRegistry(
    ctx,
    "gs://example-public-bucket/tensa/drills/plugins",
    "/var/cache/drills",
    log,
)

// Sync all plugins from GCS (downloads and decompresses .so.gz files)
registry.Sync(ctx)

// Load plugin (from cache)
drill, err := registry.GetDrill("crosscourt")
```

**GCS File Structure:**
```
gs://bucket/drills/
  ├── drill-uuid-1.so.gz    (recommended - compressed)
  ├── drill-uuid-2.so       (also supported - uncompressed)
  └── ... (add more .so.gz or .so files)
```

**Note:** No metadata files required - just the `.so` or `.so.gz` plugin files.

### Mock Registry (Testing)

```go
registry := loader.NewMockRegistry()
registry.Register("test-drill", &myTestDrill{})
drill, _ := registry.GetDrill("test-drill")
```

## Adding New Features

### Adding to API Package (Rare - Breaking)

**When**: Adding new capabilities that plugins need access to

**Impact**: ALL plugins must be rebuilt

**Steps**:
1. Add interface to `api/types.go`:
   ```go
   type NewInterface interface {
       NewMethod(ctx context.Context) error
   }
   ```

2. Add to `api.Runtime`:
   ```go
   type Runtime struct {
       // ... existing fields
       NewThing NewInterface
   }
   ```

3. **REBUILD EVERYTHING**:
   ```bash
   cd pkg/ai/drillsx/examples
   ./rebuild-all.sh
   ```

**Why Rebuild**: API changes affect the plugin interface contract

### Adding to Drillutil Package (Common - Non-Breaking)

**When**: Adding helper functions for plugin convenience

**Impact**: Plugins can adopt at their own pace

**Steps**:
1. Add function to `drillutil/helpers.go`:
   ```go
   func MyNewHelper(rt api.Runtime, arg float64) (api.Point, error) {
       // Implementation
   }
   ```

2. Update plugin to use it:
   ```go
   import "github.com/notnil/tensa/pkg/ai/drillsx/drillutil"
   
   point := drillutil.MyNewHelper(rt, 5.0)
   ```

3. **Rebuild just updated plugins**:
   ```bash
   cd pkg/ai/drillsx/examples
   make my-drill  # Only rebuild plugins that use the new helper
   ```

**Why No Full Rebuild**: Drillutil is optional, doesn't affect API contract

### Adding Conversion Functions (Common - Non-Breaking)

**When**: Supporting new external types

**Impact**: None - conversions are utilities

**Steps**:
1. Add to `drillutil/convert.go`:
   ```go
   func NewTypeToAPI(t external.Type) api.Type {
       // Convert
   }
   ```

2. Use in plugins as needed

3. **No rebuild required** for plugins not using it

## Rebuild Requirements

### When Main App Changes

| Change Type | Rebuild Main | Rebuild Plugins | Reason |
|------------|--------------|-----------------|---------|
| Add feature to main code | ✅ Yes | ❌ No | Plugins don't depend on main code |
| Update internal packages | ✅ Yes | ❌ No | Plugins use API interface only |
| Update API interface | ✅ Yes | ✅ Yes | Changes plugin contract |
| Update API types | ✅ Yes | ✅ Yes | Changes data structures |

### When Helper Code Changes

| Change Type | Rebuild Main | Rebuild Plugins | Reason |
|------------|--------------|-----------------|---------|
| Add drillutil function | ❌ No | ⚙️ Optional | Only plugins using it |
| Change drillutil function | ❌ No | ⚙️ Optional | Only plugins using it |
| Add conversion | ❌ No | ❌ No | Pure utility |

### When Plugin Changes

| Change Type | Rebuild Main | Rebuild Plugins | Reason |
|------------|--------------|-----------------|---------|
| Modify plugin logic | ❌ No | ✅ That plugin | Plugin independent |
| Add new plugin | ❌ No | ✅ New plugin | Plugin independent |

### Using rebuild-all.sh

The safe approach when unsure:

```bash
cd pkg/ai/drillsx/examples
./rebuild-all.sh
```

This script:
1. Cleans all old builds
2. Rebuilds main application (tensactl)
3. Rebuilds all drill plugins
4. Ensures version compatibility

**When to use**:
- After updating Go version
- After running `go get -u` or `go mod tidy`
- After pulling changes that modify `go.mod`
- After changing API package
- When experiencing plugin load errors

## Platform Support

### Supported Platforms

- ✅ Linux (amd64, arm64) - Primary target (Jetson)
- ✅ macOS (darwin/arm64, darwin/amd64) - Development
- ❌ Windows - Go plugins not supported

### Cross-Compilation

Build for Jetson from Mac/Linux:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 \
  go build -buildmode=plugin -o drill.so ./drill
```

**Note**: Cross-compiling plugins with CGO can be challenging. Best practice is to build on the target platform or use a compatible build environment.

## Examples

The `examples/` directory contains several reference implementations:

### Basic Examples
- **crosscourt** - Basic cross-court drill
- **down-the-line-backhand** - Baseline placement drill
- **progressive-volley** - Speed/position progression drill

### Advanced Examples
- **criss-cross** - Alternating position drill with audio
- **drop-lob** - Pattern variation drill

### Examining Examples

```bash
# View a drill implementation
cat examples/crosscourt_forehand/main.go

# Build and test locally
cd examples
make crosscourt_forehand
```

## Troubleshooting

### Plugin Compatibility Errors

**Error**: `plugin was built with a different version of package`

**Cause**: Version mismatch between main app and plugins

**Solution**:
```bash
cd pkg/ai/drillsx/examples
./rebuild-all.sh
```

See `examples/FIX-PLUGIN-COMPATIBILITY.md` for details.

### Plugin Load Failures

**Error**: `plugin.Open: symbol not found`

**Cause**: API interface mismatch

**Solution**:
1. Verify plugin implements `api.Drill` interface
2. Check exported symbol name is exactly `Drill`
3. Rebuild plugin with correct Go version

### Build Errors

**Error**: `buildmode=plugin not supported`

**Cause**: Trying to build on Windows or without CGO

**Solution**:
- Use Linux or macOS
- Ensure `CGO_ENABLED=1`
- Install C compiler (gcc/clang)

## API Reference

### Core Interfaces

#### Drill Interface

```go
type Drill interface {
    Run(ctx context.Context, rt Runtime) error
}
```

Every plugin must implement this interface and export it as `Drill`.

#### Runtime Struct

```go
type Runtime struct {
    Nav            Navigator        // Navigation system
    Thrower        Thrower          // Ball throwing mechanism  
    Audio          AudioPlayer      // Audio playback
    Events         EventSub         // System events
    Metrics        ShotMetricWriter // Shot metrics
    PlayerProvider PlayerProvider   // Player tracking system
    Log            *slog.Logger     // Structured logger
    Rnd            *rand.Rand       // Random number generator
}
```

### Key Types

```go
type Point struct {
    X float64
    Y float64
}

type Location struct {
    Point    Point
    Rotation float64
}

type Settings struct {
    Top    float64  // Top motor speed (rad/s)
    Bottom float64  // Bottom motor speed (rad/s)
    Angle  float64  // Throw angle (radians)
}

type Range struct {
    Min float64
    Max float64
}
```

### Drillutil Helpers

```go
// Sampling
func Uniform(r *rand.Rand, min, max float64) (float64, error)
func SampleRange(rng *rand.Rand, r api.Range) (float64, error)
func SampleBox(r *rand.Rand, min, max api.Point) (api.Point, error)
func SamplePolygon(r *rand.Rand, polygon api.Polygon) (api.Point, error)

// Location creation
func MakeLoc(point api.Point, rotation float64) api.Location

// Throw preparation
func PrepareThrow(ctx context.Context, rt api.Runtime, params PrepareParams, log *slog.Logger) error
```

## Best Practices

1. **Always respect context cancellation** - Check `ctx.Done()` in loops
2. **Use PrepareThrow for concurrent setup** - It's faster than sequential
3. **Log important events** - Use `rt.Log` for debugging
4. **Handle errors properly** - Return errors to abort execution
5. **Test with MockRegistry first** - Before building actual plugins
6. **Version control your drills** - Use git for drill source code
7. **Document your drill logic** - Future you will thank you

## Contributing

When contributing new drills:

1. Follow the existing structure
2. Add your drill to `PLUGINS` list in `Makefile`
3. Test locally before committing
4. Document special requirements
5. Consider whether changes belong in `api` (rare) or `drillutil` (common)

## License

Part of the Tensa Sports tennis ball machine project.
