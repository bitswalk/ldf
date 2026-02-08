package distributions

import (
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/build/kernel"
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles distribution-related HTTP requests
type Handler struct {
	distRepo        *db.DistributionRepository
	downloadJobRepo *db.DownloadJobRepository
	buildJobRepo    *db.BuildJobRepository
	sourceRepo      *db.SourceRepository
	jwtService      *auth.JWTService
	storageManager  StorageManager
	kernelConfigSvc *kernel.KernelConfigService
}

// StorageManager interface for artifact storage operations
type StorageManager interface {
	ListByDistribution(distributionID string) ([]string, error)
	DeleteByDistribution(distributionID string) (int, int64, error)
}

// Config contains configuration options for the Handler
type Config struct {
	DistRepo        *db.DistributionRepository
	DownloadJobRepo *db.DownloadJobRepository
	BuildJobRepo    *db.BuildJobRepository
	SourceRepo      *db.SourceRepository
	JWTService      *auth.JWTService
	StorageManager  StorageManager
	KernelConfigSvc *kernel.KernelConfigService
}

// CreateDistributionRequest represents the request to create a distribution
type CreateDistributionRequest struct {
	Name       string                 `json:"name" binding:"required" example:"ubuntu-22.04"`
	Version    string                 `json:"version" example:"22.04.3"`
	Visibility string                 `json:"visibility" example:"private"`
	Config     *db.DistributionConfig `json:"config"`
	SourceURL  string                 `json:"source_url" example:"https://releases.ubuntu.com/22.04/ubuntu-22.04.3-live-server-amd64.iso"`
	Checksum   string                 `json:"checksum" example:"sha256:a4acfda10b18da50e2ec50ccaf860d7f20b389df8765611142305c0e911d16fd"`
}

// UpdateDistributionRequest represents the request to update a distribution
type UpdateDistributionRequest struct {
	Name       string                 `json:"name" example:"ubuntu-22.04"`
	Version    string                 `json:"version" example:"22.04.3"`
	Status     string                 `json:"status" example:"ready"`
	Visibility string                 `json:"visibility" example:"public"`
	SourceURL  string                 `json:"source_url" example:"https://releases.ubuntu.com/22.04/ubuntu-22.04.3-live-server-amd64.iso"`
	Checksum   string                 `json:"checksum" example:"sha256:a4acfda10b18da50e2ec50ccaf860d7f20b389df8765611142305c0e911d16fd"`
	SizeBytes  int64                  `json:"size_bytes" example:"2048576000"`
	Config     *db.DistributionConfig `json:"config,omitempty"`
}

// DistributionListResponse represents a list of distributions
type DistributionListResponse struct {
	Count         int               `json:"count" example:"5"`
	Distributions []db.Distribution `json:"distributions"`
}

// DistributionStatsResponse represents distribution statistics
type DistributionStatsResponse struct {
	Total int64            `json:"total" example:"10"`
	Stats map[string]int64 `json:"stats"`
}

// DeletionPreviewResponse represents what will be deleted when a distribution is removed
type DeletionPreviewResponse struct {
	Distribution   db.Distribution        `json:"distribution"`
	DownloadJobs   DeletionPreviewCount   `json:"download_jobs"`
	Artifacts      DeletionPreviewCount   `json:"artifacts"`
	UserSources    DeletionPreviewSources `json:"user_sources"`
	TotalSizeBytes int64                  `json:"total_size_bytes"`
}

// DeletionPreviewCount represents a count with optional details
type DeletionPreviewCount struct {
	Count int      `json:"count"`
	Items []string `json:"items,omitempty"`
}

// DeletionPreviewSources represents user sources that will be deleted
type DeletionPreviewSources struct {
	Count   int                     `json:"count"`
	Sources []DeletionSourceSummary `json:"sources,omitempty"`
}

// DeletionSourceSummary represents a summary of a source to be deleted
type DeletionSourceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
