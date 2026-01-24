package distributions

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// StorageAdapter adapts the storage.Backend interface to StorageManager
type StorageAdapter struct {
	backend storage.Backend
}

// NewStorageAdapter creates a new storage adapter
func NewStorageAdapter(backend storage.Backend) *StorageAdapter {
	if backend == nil {
		return nil
	}
	return &StorageAdapter{backend: backend}
}

// ListByDistribution lists all artifact keys for a distribution
func (a *StorageAdapter) ListByDistribution(distributionID string) ([]string, error) {
	if a.backend == nil {
		return nil, nil
	}

	// Artifacts are stored under distribution/{owner_id}/{distribution_id}/
	// We need to search for all objects with the distribution ID in the path
	ctx := context.Background()

	// List all distribution artifacts
	allArtifacts, err := a.backend.List(ctx, "distribution/")
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	var keys []string
	for _, obj := range allArtifacts {
		// Parse the key to extract distribution ID
		// Format: distribution/{owner_id}/{distribution_id}/{path}
		parts := strings.SplitN(obj.Key, "/", 4)
		if len(parts) >= 3 && parts[2] == distributionID {
			// Return the artifact path (last part)
			if len(parts) >= 4 {
				keys = append(keys, parts[3])
			}
		}
	}

	return keys, nil
}

// DeleteByDistribution deletes all artifacts for a distribution
// Returns the number of deleted artifacts and total bytes freed
func (a *StorageAdapter) DeleteByDistribution(distributionID string) (int, int64, error) {
	if a.backend == nil {
		return 0, 0, nil
	}

	ctx := context.Background()

	// List all distribution artifacts
	allArtifacts, err := a.backend.List(ctx, "distribution/")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list artifacts: %w", err)
	}

	var deletedCount int
	var deletedBytes int64

	for _, obj := range allArtifacts {
		// Parse the key to extract distribution ID
		// Format: distribution/{owner_id}/{distribution_id}/{path}
		parts := strings.SplitN(obj.Key, "/", 4)
		if len(parts) >= 3 && parts[2] == distributionID {
			// Delete this artifact
			if err := a.backend.Delete(ctx, obj.Key); err != nil {
				// Log error but continue with other deletions
				continue
			}
			deletedCount++
			deletedBytes += obj.Size
		}
	}

	return deletedCount, deletedBytes, nil
}
