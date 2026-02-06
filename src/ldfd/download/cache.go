package download

import (
	"context"
	"fmt"
	"path"
	"sync"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// CacheConfig holds configuration for the artifact cache
type CacheConfig struct {
	Enabled   bool // Enable/disable caching (default: true)
	MaxSizeGB int  // Max cache size in GB (0 = unlimited)
}

// DefaultCacheConfig returns sensible defaults for cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:   true,
		MaxSizeGB: 0,
	}
}

// Cache manages a shared artifact cache across distributions.
// Artifacts are stored under cache/artifacts/{sourceID}/{version}/{filename}
// and copied to distribution-specific paths on cache hits.
type Cache struct {
	repo    *db.ArtifactCacheRepository
	storage storage.Backend
	config  CacheConfig
	mu      sync.RWMutex
}

// NewCache creates a new artifact cache manager
func NewCache(repo *db.ArtifactCacheRepository, storage storage.Backend, cfg CacheConfig) *Cache {
	return &Cache{
		repo:    repo,
		storage: storage,
		config:  cfg,
	}
}

// Lookup checks if an artifact exists in cache for the given source+version.
// Returns the cache entry if found and the artifact exists in storage, nil otherwise.
func (c *Cache) Lookup(ctx context.Context, sourceID, version string) (*db.ArtifactCacheEntry, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	entry, err := c.repo.GetBySourceAndVersion(sourceID, version)
	if err != nil {
		return nil, fmt.Errorf("cache lookup failed: %w", err)
	}
	if entry == nil {
		return nil, nil
	}

	// Verify the artifact still exists in storage
	exists, err := c.storage.Exists(ctx, entry.CachePath)
	if err != nil {
		log.Warn("Failed to verify cached artifact", "cache_path", entry.CachePath, "error", err)
		return nil, nil
	}
	if !exists {
		// Cache entry is stale — remove it
		log.Info("Removing stale cache entry", "source_id", sourceID, "version", version)
		if err := c.repo.Delete(entry.ID); err != nil {
			log.Warn("Failed to delete stale cache entry", "id", entry.ID, "error", err)
		}
		return nil, nil
	}

	// Touch the entry to update LRU tracking
	if err := c.repo.TouchLastUsed(entry.ID); err != nil {
		log.Warn("Failed to touch cache entry", "id", entry.ID, "error", err)
	}

	return entry, nil
}

// Store adds a downloaded artifact to the cache by copying it from its
// distribution-specific path to the shared cache namespace.
func (c *Cache) Store(ctx context.Context, sourceID, version, artifactPath, checksum string, sizeBytes int64, contentType string) error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already cached
	existing, err := c.repo.GetBySourceAndVersion(sourceID, version)
	if err == nil && existing != nil {
		// Already cached, just touch it
		return c.repo.TouchLastUsed(existing.ID)
	}

	// Build cache path
	cachePath := c.buildCachePath(sourceID, version, artifactPath)

	// Copy artifact to cache namespace
	if err := c.storage.Copy(ctx, artifactPath, cachePath); err != nil {
		return fmt.Errorf("failed to copy artifact to cache: %w", err)
	}

	// Create cache entry
	entry := &db.ArtifactCacheEntry{
		SourceID:    sourceID,
		Version:     version,
		Checksum:    checksum,
		CachePath:   cachePath,
		SizeBytes:   sizeBytes,
		ContentType: contentType,
		ResolvedURL: artifactPath, // store original path for reference
	}
	if err := c.repo.Create(entry); err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	log.Info("Artifact cached",
		"source_id", sourceID,
		"version", version,
		"cache_path", cachePath,
		"size", sizeBytes,
	)

	// Run eviction if needed
	if err := c.evict(ctx); err != nil {
		log.Warn("Cache eviction error", "error", err)
	}

	return nil
}

// CopyToDistribution copies a cached artifact to a distribution's artifact path.
func (c *Cache) CopyToDistribution(ctx context.Context, entry *db.ArtifactCacheEntry, dstPath string) error {
	return c.storage.Copy(ctx, entry.CachePath, dstPath)
}

// evict removes old cache entries to keep total size under MaxSizeGB.
func (c *Cache) evict(ctx context.Context) error {
	if c.config.MaxSizeGB <= 0 {
		return nil // No size limit
	}

	maxBytes := int64(c.config.MaxSizeGB) * 1024 * 1024 * 1024

	for {
		totalSize, err := c.repo.TotalSize()
		if err != nil {
			return fmt.Errorf("failed to get cache size: %w", err)
		}
		if totalSize <= maxBytes {
			return nil
		}

		// Get the least recently used entries
		entries, err := c.repo.ListLRU(5)
		if err != nil {
			return fmt.Errorf("failed to list LRU entries: %w", err)
		}
		if len(entries) == 0 {
			return nil
		}

		for _, entry := range entries {
			// Delete from storage
			if err := c.storage.Delete(ctx, entry.CachePath); err != nil {
				log.Warn("Failed to delete cached artifact", "cache_path", entry.CachePath, "error", err)
			}
			// Delete from database
			if err := c.repo.Delete(entry.ID); err != nil {
				log.Warn("Failed to delete cache entry", "id", entry.ID, "error", err)
			}
			log.Info("Evicted cache entry",
				"source_id", entry.SourceID,
				"version", entry.Version,
				"size", entry.SizeBytes,
			)

			// Recheck after each deletion
			totalSize, err = c.repo.TotalSize()
			if err != nil {
				return err
			}
			if totalSize <= maxBytes {
				return nil
			}
		}
	}
}

// Stats returns cache statistics
func (c *Cache) Stats(ctx context.Context) (totalSize int64, entryCount int, err error) {
	totalSize, err = c.repo.TotalSize()
	if err != nil {
		return 0, 0, err
	}
	entryCount, err = c.repo.Count()
	if err != nil {
		return 0, 0, err
	}
	return totalSize, entryCount, nil
}

// buildCachePath constructs the cache storage key:
// cache/artifacts/{sourceID}/{version}/{filename}
func (c *Cache) buildCachePath(sourceID, version, originalPath string) string {
	filename := path.Base(originalPath)
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("%s-%s.tar.gz", sourceID, version)
	}
	return fmt.Sprintf("cache/artifacts/%s/%s/%s", sourceID, version, filename)
}
