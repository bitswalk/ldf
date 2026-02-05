package build

import (
	"context"
	"fmt"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// DownloadCheckStage verifies that all required component downloads are complete
type DownloadCheckStage struct {
	downloadJobRepo *db.DownloadJobRepository
	storage         storage.Backend
}

// NewDownloadCheckStage creates a new download check stage
func NewDownloadCheckStage(downloadJobRepo *db.DownloadJobRepository, storage storage.Backend) *DownloadCheckStage {
	return &DownloadCheckStage{
		downloadJobRepo: downloadJobRepo,
		storage:         storage,
	}
}

// Name returns the stage name
func (s *DownloadCheckStage) Name() db.BuildStageName {
	return db.StageDownload
}

// Validate checks whether this stage can run
func (s *DownloadCheckStage) Validate(ctx context.Context, sc *StageContext) error {
	if len(sc.Components) == 0 {
		return fmt.Errorf("no components resolved - resolve stage must run first")
	}
	return nil
}

// Execute verifies all component downloads are complete and artifacts exist
func (s *DownloadCheckStage) Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error {
	progress(0, "Verifying component downloads")

	totalComponents := len(sc.Components)
	var pendingDownloads []string

	for i, rc := range sc.Components {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pct := (90 * (i + 1)) / totalComponents
		progress(pct, fmt.Sprintf("Checking download: %s v%s", rc.Component.Name, rc.Version))

		// Verify artifact path is set
		if rc.ArtifactPath == "" {
			pendingDownloads = append(pendingDownloads, fmt.Sprintf("%s v%s (no artifact path)", rc.Component.Name, rc.Version))
			continue
		}

		// Verify artifact exists in storage
		if s.storage != nil {
			exists, err := s.storage.Exists(ctx, rc.ArtifactPath)
			if err != nil {
				log.Warn("Failed to check artifact existence",
					"component", rc.Component.Name,
					"version", rc.Version,
					"path", rc.ArtifactPath,
					"error", err)
				pendingDownloads = append(pendingDownloads, fmt.Sprintf("%s v%s (storage check failed: %v)", rc.Component.Name, rc.Version, err))
				continue
			}
			if !exists {
				pendingDownloads = append(pendingDownloads, fmt.Sprintf("%s v%s (artifact not found at %s)", rc.Component.Name, rc.Version, rc.ArtifactPath))
				continue
			}
		}

		log.Info("Download verified",
			"component", rc.Component.Name,
			"version", rc.Version,
			"artifact", rc.ArtifactPath)
	}

	if len(pendingDownloads) > 0 {
		progress(95, fmt.Sprintf("Missing downloads: %d", len(pendingDownloads)))
		return fmt.Errorf("missing downloads for build:\n  - %s\n\nPlease ensure all component downloads are complete before building",
			joinStrings(pendingDownloads, "\n  - "))
	}

	progress(100, fmt.Sprintf("All %d downloads verified", totalComponents))
	return nil
}

// WaitForDownloads waits for pending downloads to complete (for future use when we want to auto-trigger downloads)
func (s *DownloadCheckStage) WaitForDownloads(ctx context.Context, distributionID string, componentIDs []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		allComplete := true
		jobs, err := s.downloadJobRepo.ListByDistribution(distributionID)
		if err != nil {
			return fmt.Errorf("failed to list download jobs: %w", err)
		}

		for _, componentID := range componentIDs {
			found := false
			for _, job := range jobs {
				if job.ComponentID == componentID || containsString(job.ComponentIDs, componentID) {
					found = true
					if job.Status != db.JobStatusCompleted {
						allComplete = false
						if job.Status == db.JobStatusFailed {
							return fmt.Errorf("download failed for component %s: %s", componentID, job.ErrorMessage)
						}
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("no download job found for component %s", componentID)
			}
		}

		if allComplete {
			return nil
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for downloads to complete")
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
