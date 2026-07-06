# Drill Plugin Build Guide

## Quick Reference

### Local Development (Default)
```bash
make                    # Build all plugins (uncompressed)
make local              # Same as above
make clean              # Clean all build artifacts
```

### Compress for Distribution
```bash
make compress           # Build and compress all plugins
make clean-compressed   # Clean only compressed files
```

### Build and Upload to GCS
```bash
make gcs                # Build, compress, and upload all plugins
```

### Individual Plugin Operations
```bash
# Build single plugin (uncompressed)
make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d

# Compress single plugin
make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.compress

# Upload single plugin to GCS
make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.upload
```

### Cross-Platform Builds

#### Build for Jetson and Upload
```bash
make jetson gcs
# Equivalent to: GOOS=linux GOARCH=arm64 make gcs
```

#### Build for specific platforms
```bash
make linux              # Build for Linux (current GOARCH)
make darwin             # Build for macOS (current GOARCH)
make amd64              # Build for x86_64 (current GOOS)
```

## Directory Structure

```
pkg/ai/drillsx/examples/
├── build/
│   ├── *.so                    # Uncompressed plugins (local use)
│   └── compressed/
│       └── *.so.gz             # Compressed plugins (GCS upload)
├── <plugin-uuid>/
│   ├── main.go                 # Drill implementation
│   └── *.mp3                   # Audio files (embedded via go:embed)
└── Makefile
```

## Text-to-Speech Generation

Drills can include audio instructions that are generated from text comments using Google Cloud Text-to-Speech API.

### Adding TTS to a Drill

1. **Add the go:generate directive** at the top of your drill's `main.go`:
   ```go
   //go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go
   
   // Package main implements your drill as a plugin.
   package main
   ```

2. **Add TTS comments** for each audio file you want to generate:
   ```go
   //tts:filename=intro.mp3 Starting criss cross drill. Get ready to move at the baseline.
   //tts:filename=tip1.mp3 Focus on footwork between shots.
   //tts:filename=tip2.mp3 Recover to the center after each shot.
   ```

3. **Add go:embed directives** to embed the generated MP3 files:
   ```go
   //go:embed intro.mp3
   var introSound []byte
   
   //go:embed tip1.mp3
   var tip1Sound []byte
   ```

4. **Generate the audio files**:
   ```bash
   cd <plugin-uuid>/
   go generate
   ```

5. **Use the audio in your drill**:
   ```go
   func (d *myDrill) Run(ctx context.Context, rt api.Runtime) error {
       go func() {
           buf := bytes.NewBuffer(introSound)
           if err := rt.Audio.Play(buf); err != nil {
               rt.Log.Error("failed to play audio", "error", err)
           }
       }()
       // ... rest of drill logic
   }
   ```

### TTS Comment Format

**Syntax**: `//tts:filename=<name.mp3> <text to speak>`

- The text after the filename will be converted to speech
- Multiple lines with the same filename will be concatenated with spaces
- The generated MP3 file will be created in the same directory as `main.go`

**Example with multi-line text**:
```go
//tts:filename=instructions.mp3 This drill will test your footwork.
//tts:filename=instructions.mp3 Move quickly between shots and stay balanced.
```
Result: One `instructions.mp3` file with both sentences spoken.

### Requirements

- **Google Cloud credentials**: Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable
- **Network access**: The tool makes API calls to Google Cloud Text-to-Speech
- **Go installed**: Required to run `go generate`

### Setup Google Cloud Authentication

```bash
# Set up credentials (one-time setup)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/credentials.json"

# Or use gcloud CLI
gcloud auth application-default login
```

### Voice Configuration

The TTS generator uses:
- **Voice**: `en-US-Studio-O` (high-quality female voice)
- **Language**: `en-US`
- **Sample Rate**: 48000 Hz
- **Format**: MP3

To customize the voice, edit `cmd/scripts/speech/tts_generator.go` in the `GenerateFromText` method.

### Example: Complete Drill with TTS

```go
//go:generate go run ../../../../../cmd/scripts/drillsx/generate-tts/main.go

// Package main implements the example drill as a plugin.
package main

import (
    "bytes"
    "context"
    _ "embed"
    
    "github.com/notnil/tensa/pkg/ai/drillsx/api"
)

//tts:filename=intro.mp3 Welcome to the cross-court forehand drill.
//tts:filename=tip.mp3 Aim past the service line and recover quickly.

//go:embed intro.mp3
var introSound []byte

//go:embed tip.mp3
var tipSound []byte

type exampleDrill struct{}

func (d *exampleDrill) Run(ctx context.Context, rt api.Runtime) error {
    // Play intro
    go func() {
        buf := bytes.NewBuffer(introSound)
        rt.Audio.Play(buf)
    }()
    
    // ... drill implementation ...
    
    return nil
}

var Drill api.Drill = &exampleDrill{}
```

### Workflow

1. **Write text** in `//tts:filename=` comments
2. **Run** `go generate` in the drill directory
3. **MP3 files** are created automatically
4. **Commit** the MP3 files to the repository (they are source files)
5. **Build** the plugin with `make` (MP3s are embedded via `go:embed`)

### Troubleshooting

**Error**: `GOOGLE_APPLICATION_CREDENTIALS not set`
```bash
# Set the environment variable
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials.json"
```

**Error**: `No //tts: comments found`
- Check that your comments use the exact format: `//tts:filename=name.mp3 Text`
- Ensure there's no space before `//tts:`
- Verify the file is named `main.go` or has a `.go` extension

**Error**: `Failed to synthesize speech`
- Check your Google Cloud credentials are valid
- Verify you have Text-to-Speech API enabled in your GCP project
- Ensure you have network connectivity

**Regenerating audio**:
```bash
# Delete old MP3 files and regenerate
rm *.mp3
go generate
```

## Workflow Examples

### Typical Local Development
```bash
# 1. Edit drill code
vim a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d/main.go

# 2. Build locally
make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d

# 3. Test with tensactl (uses local .so file)
cd ../../../cmd/tensactl
./tensactl ...
```

### Deploy to Production
```bash
# 1. Build for Jetson and upload to GCS
make jetson gcs

# 2. Robots will download compressed .so.gz files from GCS
# 3. GCSRegistry automatically decompresses during Sync()
```

### Update Single Drill in Production
```bash
# 1. Build, compress, and upload just one drill
make jetson a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.upload

# 2. Verify upload
gsutil ls -lh gs://example-public-bucket/tensa/drills/plugins/a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.so.gz
```

## File Formats

### Local (.so files)
- **Purpose**: Local development and testing
- **Location**: `./build/*.so`
- **Used by**: Local `FSRegistry`, direct plugin loading
- **Size**: ~3 MB per plugin (uncompressed)

### GCS (.so.gz files)
- **Purpose**: Production distribution
- **Location**: `gs://example-public-bucket/tensa/drills/plugins/*.so.gz`
- **Used by**: `GCSRegistry` (auto-decompresses), mobile app BLE upload
- **Size**: ~1.2 MB per plugin (60% compression)
- **Format**: gzip -9 (best compression)

## Compression Details

### Compression Command
```bash
gzip -9 -c input.so > output.so.gz
```
- `-9`: Best compression level
- `-c`: Write to stdout (preserves original)
- Output is piped to `.so.gz` file

### Typical Compression Ratios
- Original: 3.0 MB
- Compressed: 1.2 MB
- **Ratio: 60% reduction**

### Benefits
- ✅ 60% smaller files for GCS storage
- ✅ 60% faster downloads from GCS
- ✅ 3x faster BLE transfers to robot
- ✅ Lower GCS egress costs
- ✅ Better mobile UX over cellular

## GCS Upload

### Configuration
```makefile
GCS_BUCKET := gs://example-public-bucket/tensa/drills/plugins
```

### Requirements
- `gsutil` CLI tool installed
- GCS credentials configured: `gcloud auth login`
- Write permissions to bucket

### Verify Upload
```bash
# List all plugins in GCS
gsutil ls gs://example-public-bucket/tensa/drills/plugins/

# Check specific plugin
gsutil ls -lh gs://example-public-bucket/tensa/drills/plugins/<plugin-id>.so.gz

# Download and test locally
gsutil cp gs://example-public-bucket/tensa/drills/plugins/<plugin-id>.so.gz .
gunzip <plugin-id>.so.gz
file <plugin-id>.so  # Should show: Mach-O 64-bit dynamically linked shared library arm64
```

## Troubleshooting

### Build Errors

**Error**: `plugin was built with a different version of package`
```bash
# Solution: Rebuild main app and all plugins
cd ../../..
make clean
make
cd pkg/ai/drillsx/examples
make clean
make
```

**Error**: `buildmode=plugin not supported`
- Cause: Building on Windows or without CGO
- Solution: Use Linux or macOS, ensure `CGO_ENABLED=1`

### Upload Errors

**Error**: `gsutil: command not found`
```bash
# Install Google Cloud SDK
brew install google-cloud-sdk
# Or: https://cloud.google.com/sdk/docs/install

# Authenticate
gcloud auth login
```

**Error**: `AccessDeniedException: 403`
```bash
# Verify credentials
gcloud auth list

# Check bucket permissions
gsutil iam get gs://example-public-bucket/tensa
```

### Compression Issues

**Error**: `gzip: stdin: not in gzip format`
- File is already compressed or corrupted
- Clean and rebuild:
  ```bash
  make clean
  make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d.compress
  ```

**File size too large after compression**
- Check plugin dependencies (remove unused imports)
- Verify build flags include `-ldflags="-s -w"` (strip symbols)

## Advanced Usage

### Override GCS Bucket
```bash
make gcs GCS_BUCKET=gs://my-custom-bucket/drills
```

### Build Verbose
```bash
make a3b4c5d6-7e8f-4a9b-0c1d-2e3f4a5b6c7d LDFLAGS=""
```

### Parallel Builds
```bash
make -j8 local      # Build 8 plugins in parallel
make -j8 compress   # Compress 8 plugins in parallel
```

### Custom Platform
```bash
GOOS=linux GOARCH=amd64 make local
GOOS=linux GOARCH=arm64 make gcs
```

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Build and upload drills
  run: |
    cd pkg/ai/drillsx/examples
    make jetson gcs
  env:
    GOOS: linux
    GOARCH: arm64
    GOOGLE_APPLICATION_CREDENTIALS: <GCS_SERVICE_ACCOUNT_JSON>
```

### Automated Deployment
```bash
#!/bin/bash
# deploy-drills.sh

set -e

cd pkg/ai/drillsx/examples

# Build for Jetson
echo "Building plugins for Jetson..."
make jetson clean
make jetson compress

# Upload to GCS
echo "Uploading to GCS..."
make jetson gcs

# Verify uploads
echo "Verifying uploads..."
for plugin in $(echo $PLUGINS); do
    gsutil stat gs://example-public-bucket/tensa/drills/plugins/$plugin.so.gz
done

echo "✓ All plugins deployed successfully!"
```

## Plugin Discovery

The Makefile automatically discovers all plugin directories by searching for folders containing a `main.go` file. This means you don't need to manually update the Makefile when adding new drills - just create a new directory with a `main.go` file and the build system will pick it up automatically.

To see all available plugins, run:
```bash
make help
```

This will display the "Available Plugins" section with all discovered plugin directories.

