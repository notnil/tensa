// Package loader provides utilities for loading and executing drill plugins.
package loader

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/notnil/tensa/pkg/ai/drillsx/api"
	"google.golang.org/api/iterator"
)

// GCSRegistry is a Google Cloud Storage-based implementation of Registry.
// It syncs drill plugins from GCS .so or .so.gz files to a local cache directory and loads them on demand.
// The expected file naming convention is <drill-id>.so or <drill-id>.so.gz (compressed files are automatically decompressed).
//
// Usage pattern:
//  1. Create registry with NewGCSRegistry()
//  2. Call Sync() to download all plugins from GCS to local cache
//  3. Call GetDrill() to load plugins from local cache
type GCSRegistry struct {
	bucket   string
	rootPath string
	loader   *Loader
	client   *storage.Client
	cache    map[string]api.Drill
	cacheDir string
	mu       sync.RWMutex
	log      *slog.Logger
}

// NewGCSRegistry creates a new GCSRegistry that loads plugins from the specified GCS location.
// The gcsURI should be in the format: gs://bucket-name/path/to/plugins
// The cachePath is a persistent directory where plugins will be stored across reboots.
// Call Sync() to download plugins from GCS before calling GetDrill().
func NewGCSRegistry(ctx context.Context, gcsURI string, cachePath string, log *slog.Logger) (*GCSRegistry, error) {
	// Parse the GCS URI
	bucket, rootPath, err := parseGCSURI(gcsURI)
	if err != nil {
		return nil, fmt.Errorf("invalid GCS URI: %w", err)
	}

	// Create GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Create the cache directory if it doesn't exist
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	log.Info("initialized GCS registry", "bucket", bucket, "rootPath", rootPath, "cacheDir", cachePath)

	return &GCSRegistry{
		bucket:   bucket,
		rootPath: rootPath,
		loader:   New(log),
		client:   client,
		cache:    make(map[string]api.Drill),
		cacheDir: cachePath,
		log:      log,
	}, nil
}

// parseGCSURI parses a GCS URI in the format gs://bucket-name/path/to/object
// and returns the bucket name and object path separately.
func parseGCSURI(uri string) (bucket, path string, err error) {
	if !strings.HasPrefix(uri, "gs://") {
		return "", "", fmt.Errorf("URI must start with gs://")
	}

	// Remove the gs:// prefix
	trimmed := strings.TrimPrefix(uri, "gs://")

	// Split on the first /
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid GCS URI format: missing bucket name")
	}

	bucket = parts[0]
	if len(parts) == 2 {
		path = parts[1]
	}

	return bucket, path, nil
}

// Sync downloads all .so and .so.gz plugin files from the GCS bucket to the local cache directory.
// This should be called before GetDrill() to ensure plugins are available locally.
// Compressed .so.gz files are automatically decompressed during download.
// Sync will skip files that already exist in the cache to avoid unnecessary downloads.
func (g *GCSRegistry) Sync(ctx context.Context) error {
	g.log.Info("syncing plugins from GCS", "bucket", g.bucket, "rootPath", g.rootPath)

	// List all objects in the bucket
	query := &storage.Query{
		Prefix: g.rootPath,
	}

	it := g.client.Bucket(g.bucket).Objects(ctx, query)

	var downloadCount, skipCount int
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		// Skip files that are neither .so nor .so.gz
		if !strings.HasSuffix(attrs.Name, ".so") && !strings.HasSuffix(attrs.Name, ".so.gz") {
			continue
		}

		// Extract the drill ID (remove .so or .so.gz extension)
		filename := filepath.Base(attrs.Name)
		drillID := strings.TrimSuffix(strings.TrimSuffix(filename, ".gz"), ".so")
		localPath := filepath.Join(g.cacheDir, fmt.Sprintf("%s.so", drillID))

		// Check if file already exists locally
		if stat, err := os.Stat(localPath); err == nil {
			// File exists - for simplicity, we skip it
			// In the future, we could compare timestamps or checksums
			g.log.Debug("skipping existing plugin", "drillID", drillID, "localSize", stat.Size, "gcsFile", filename)
			skipCount++
			continue
		}

		// Download the plugin (decompressing if needed)
		isCompressed := strings.HasSuffix(attrs.Name, ".gz")
		g.log.Info("downloading plugin", "drillID", drillID, "gcsFile", filename, "size", attrs.Size, "compressed", isCompressed)
		if err := g.downloadPlugin(ctx, attrs.Name, localPath); err != nil {
			return fmt.Errorf("failed to download plugin %q: %w", drillID, err)
		}
		downloadCount++
	}

	g.log.Info("sync completed", "downloaded", downloadCount, "skipped", skipCount, "total", downloadCount+skipCount)
	return nil
}

// GetDrill retrieves a drill by its ID by loading the corresponding .so plugin file from the local cache.
// The plugin file must exist at <cacheDir>/<id>.so. Call Sync() first to ensure plugins are downloaded.
// Loaded drills are cached in memory to avoid reloading the same plugin multiple times.
func (g *GCSRegistry) GetDrill(id string) (api.Drill, error) {
	// Check memory cache first
	g.mu.RLock()
	if drill, exists := g.cache[id]; exists {
		g.mu.RUnlock()
		return drill, nil
	}
	g.mu.RUnlock()

	// Construct the local cache path
	localPath := filepath.Join(g.cacheDir, fmt.Sprintf("%s.so", id))

	// Check if the plugin exists locally
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("drill %q not found in cache (did you call Sync()?): %w", id, err)
	}

	g.log.Debug("loading drill plugin from cache", "id", id, "localPath", localPath)

	// Load the plugin from local cache
	drill, err := g.loader.Load(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load drill %q: %w", id, err)
	}

	// Cache the loaded drill in memory
	g.mu.Lock()
	g.cache[id] = drill
	g.mu.Unlock()

	return drill, nil
}

// downloadPlugin downloads a plugin file from GCS to the local filesystem.
// If the objectPath ends with .gz, the file is automatically decompressed during download.
// The localPath should always be the final uncompressed .so path.
func (g *GCSRegistry) downloadPlugin(ctx context.Context, objectPath, localPath string) error {
	// Get the GCS object
	obj := g.client.Bucket(g.bucket).Object(objectPath)

	// Open a reader for the object
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer reader.Close()

	// Wrap in gzip reader if the object is compressed
	var dataReader io.Reader = reader
	if strings.HasSuffix(objectPath, ".gz") {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		dataReader = gzReader
		g.log.Debug("decompressing plugin during download", "objectPath", objectPath)
	}

	// Create the local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy the contents (decompressing if wrapped in gzip reader)
	bytesWritten, err := io.Copy(file, dataReader)
	if err != nil {
		return fmt.Errorf("failed to copy object contents: %w", err)
	}

	// Make the file executable
	if err := os.Chmod(localPath, 0755); err != nil {
		return fmt.Errorf("failed to make file executable: %w", err)
	}

	g.log.Debug("plugin downloaded and written", "localPath", localPath, "bytesWritten", bytesWritten)
	return nil
}

// Close cleans up resources used by the GCSRegistry.
// It closes the GCS client but preserves the cache directory for future use.
func (g *GCSRegistry) Close() error {
	// Close the GCS client
	if err := g.client.Close(); err != nil {
		return fmt.Errorf("failed to close GCS client: %w", err)
	}

	g.log.Info("closed GCS registry", "bucket", g.bucket, "rootPath", g.rootPath, "cacheDir", g.cacheDir)
	return nil
}
