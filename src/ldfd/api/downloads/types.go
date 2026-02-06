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
	Priority   int      `json:"priority,omitempty"`
}

// StartDownloadsResponse represents the response after starting downloads
type StartDownloadsResponse struct {
	Count int                   `json:"count"`
	Jobs  []DownloadJobResponse `json:"jobs"`
}

// MirrorResponse represents a mirror configuration in API responses
type MirrorResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URLPrefix string `json:"url_prefix"`
	MirrorURL string `json:"mirror_url"`
	Priority  int    `json:"priority"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// MirrorListResponse represents a list of mirror configurations
type MirrorListResponse struct {
	Count   int              `json:"count"`
	Mirrors []MirrorResponse `json:"mirrors"`
}

// CreateMirrorRequest represents the request to create a mirror
type CreateMirrorRequest struct {
	Name      string `json:"name" binding:"required"`
	URLPrefix string `json:"url_prefix" binding:"required"`
	MirrorURL string `json:"mirror_url" binding:"required"`
	Priority  int    `json:"priority"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

// UpdateMirrorRequest represents the request to update a mirror
type UpdateMirrorRequest struct {
	Name      string `json:"name,omitempty"`
	URLPrefix string `json:"url_prefix,omitempty"`
	MirrorURL string `json:"mirror_url,omitempty"`
	Priority  *int   `json:"priority,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}
