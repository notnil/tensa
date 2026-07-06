# Makefile for building drill plugins
# Build plugins with -buildmode=plugin (Linux/macOS)

# Determine the target OS/architecture (defaults to host)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Compiler flags
CGO_ENABLED := 1
LDFLAGS := -s -w
BUILD_FLAGS := -buildmode=plugin -ldflags="$(LDFLAGS)"

# Output directories
OUT_DIR := ./build
COMPRESSED_DIR := ./build/compressed

# GCS configuration
GCS_BUCKET := gs://example-public-bucket/tensa/drills/plugins

# Automatically discover plugin directories (any directory with a main.go file)
# Exclude build directory and any dot-directories
PLUGIN_DIRS := $(shell find . -maxdepth 1 -type d -not -name '.' -not -name 'build' -not -name '.*' -exec test -f {}/main.go \; -print)
PLUGINS := $(patsubst ./%,%,$(PLUGIN_DIRS))

# Define phony targets
COMPRESS_TARGETS := $(addsuffix .compress,$(PLUGINS))
UPLOAD_TARGETS := $(addsuffix .upload,$(PLUGINS))

.PHONY: all local gcs compress upload clean help linux darwin mac jetson amd64 $(PLUGINS) $(COMPRESS_TARGETS) $(UPLOAD_TARGETS)

# Default target - build for local filesystem
all: local

# Build uncompressed .so files for local development
local: $(PLUGINS)
	@echo "✓ All plugins built successfully in $(OUT_DIR)/"

# Build, compress, and upload to GCS
gcs: $(UPLOAD_TARGETS)
	@echo "✓ All plugins built, compressed, and uploaded to $(GCS_BUCKET)/"

# Build and compress (without uploading)
compress: $(COMPRESS_TARGETS)
	@echo "✓ All plugins built and compressed in $(COMPRESSED_DIR)/"

# Build each plugin (uncompressed for local use)
$(PLUGINS):
	@echo "Building $@ plugin for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUT_DIR)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build $(BUILD_FLAGS) -o $(OUT_DIR)/$@.so ./$@/
	@echo "  → $(OUT_DIR)/$@.so"

# Compress each plugin
$(COMPRESS_TARGETS): %.compress: %
	@echo "Compressing $* plugin..."
	@mkdir -p $(COMPRESSED_DIR)
	@gzip -9 -c $(OUT_DIR)/$*.so > $(COMPRESSED_DIR)/$*.so.gz
	@SIZE=$$(du -h $(COMPRESSED_DIR)/$*.so.gz | cut -f1); \
		echo "  → $(COMPRESSED_DIR)/$*.so.gz ($$SIZE)"

# Upload each plugin to GCS
$(UPLOAD_TARGETS): %.upload: %.compress
	@echo "Uploading $* to GCS..."
	@gsutil cp $(COMPRESSED_DIR)/$*.so.gz $(GCS_BUCKET)/$*.so.gz
	@echo "  → $(GCS_BUCKET)/$*.so.gz"

# Build for Jetson (ARM64)
jetson:
	@$(MAKE) GOARCH=arm64

# Build for x86_64
amd64:
	@$(MAKE) GOARCH=amd64

# Build for Linux (respect current GOARCH)
linux:
	@$(MAKE) GOOS=linux

# Build for macOS (respect current GOARCH)
darwin:
	@$(MAKE) GOOS=darwin

# Alias for macOS
mac: darwin


# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUT_DIR)

# Clean only compressed files
clean-compressed:
	@echo "Cleaning compressed files..."
	@rm -rf $(COMPRESSED_DIR)

# Show help
help:
	@echo "Drill Plugin Build System"
	@echo "========================="
	@echo ""
	@echo "Build Targets:"
	@echo "  all (default)  - Build uncompressed .so files for local use"
	@echo "  local          - Same as 'all' - build for local filesystem"
	@echo "  compress       - Build and gzip compress all plugins"
	@echo "  gcs            - Build, compress, and upload to GCS"
	@echo ""
	@echo "Platform Targets:"
	@echo "  linux          - Build for Linux (uses current GOARCH)"
	@echo "  darwin         - Build for macOS (uses current GOARCH)"
	@echo "  mac            - Alias for darwin"
	@echo "  jetson         - Build for Jetson (linux/arm64)"
	@echo "  amd64          - Build for x86_64 (uses current GOOS)"
	@echo ""
	@echo "Individual Plugin Targets:"
	@echo "  <plugin-id>            - Build uncompressed .so file"
	@echo "  <plugin-id>.compress   - Build and compress plugin"
	@echo "  <plugin-id>.upload     - Build, compress, and upload plugin"
	@echo ""
	@echo "Cleanup Targets:"
	@echo "  clean              - Remove all build artifacts"
	@echo "  clean-compressed   - Remove only compressed files"
	@echo ""
	@echo "Configuration:"
	@echo "  GOOS=$(GOOS)"
	@echo "  GOARCH=$(GOARCH)"
	@echo "  GCS_BUCKET=$(GCS_BUCKET)"
	@echo ""
	@echo "Examples:"
	@echo "  make                                           # Build all plugins locally"
	@echo "  make gcs                                       # Build, compress, upload all"
	@echo "  make criss-cross                              # Build one plugin"
	@echo "  make criss-cross.upload                       # Upload one plugin"
	@echo "  make jetson gcs                                # Build for Jetson and upload"
	@echo ""
	@echo "Available Plugins:"
	@echo "  $(PLUGINS)"
