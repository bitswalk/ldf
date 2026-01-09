package downloads

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
)

// Handler handles download-related HTTP requests
type Handler struct {
	distRepo        *db.DistributionRepository
	componentRepo   *db.ComponentRepository
	downloadManager *download.Manager
}

// Config contains configuration options for the Handler
type Config struct {
	DistRepo        *db.DistributionRepository
	ComponentRepo   *db.ComponentRepository
	DownloadManager *download.Manager
}

// DownloadJobResponse represents a download job with additional info
type DownloadJobResponse struct {
	db.DownloadJob
	ComponentName string  `json:"component_name,omitempty"`
	Progress      float64 `json:"progress"`
}

// DownloadJobsListResponse represents a list of download jobs
type DownloadJobsListResponse struct {
	Count int                   `json:"count"`
	Jobs  []DownloadJobResponse `json:"jobs"`
}

// StartDownloadsRequest represents the request to start downloads
type StartDownloadsRequest struct {
	Components []string `json:"components"`
}

// StartDownloadsResponse represents the response after starting downloads
type StartDownloadsResponse struct {
	Count int                   `json:"count"`
	Jobs  []DownloadJobResponse `json:"jobs"`
}
