package sources

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
)

// Handler handles source-related HTTP requests
type Handler struct {
	sourceRepo        *db.SourceRepository
	sourceVersionRepo *db.SourceVersionRepository
	versionDiscovery  *download.VersionDiscovery
}

// Config contains configuration options for the Handler
type Config struct {
	SourceRepo        *db.SourceRepository
	SourceVersionRepo *db.SourceVersionRepository
	VersionDiscovery  *download.VersionDiscovery
}

// CreateSourceRequest represents the request to create a source
type CreateSourceRequest struct {
	Name            string   `json:"name" binding:"required" example:"Ubuntu Releases"`
	URL             string   `json:"url" binding:"required,url" example:"https://releases.ubuntu.com"`
	ComponentIDs    []string `json:"component_ids" example:"[\"uuid-of-kernel-component\"]"`
	RetrievalMethod string   `json:"retrieval_method" example:"release"`
	URLTemplate     string   `json:"url_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	ForgeType       string   `json:"forge_type" example:"github"`
	VersionFilter   string   `json:"version_filter" example:"!*-rc*,!*alpha*,!*beta*"`
	Priority        int      `json:"priority" example:"10"`
	Enabled         *bool    `json:"enabled" example:"true"`
	IsSystem        *bool    `json:"is_system" example:"false"`
}

// UpdateSourceRequest represents the request to update a source
type UpdateSourceRequest struct {
	Name            string   `json:"name" example:"Ubuntu Releases"`
	URL             string   `json:"url" example:"https://releases.ubuntu.com"`
	ComponentIDs    []string `json:"component_ids" example:"[\"uuid-of-kernel-component\"]"`
	RetrievalMethod string   `json:"retrieval_method" example:"release"`
	URLTemplate     string   `json:"url_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	ForgeType       *string  `json:"forge_type" example:"github"`
	VersionFilter   *string  `json:"version_filter" example:"!*-rc*,!*alpha*,!*beta*"`
	Priority        *int     `json:"priority" example:"10"`
	Enabled         *bool    `json:"enabled" example:"true"`
}

// SourceListResponse represents a list of sources
type SourceListResponse struct {
	Count   int                 `json:"count" example:"5"`
	Sources []db.UpstreamSource `json:"sources"`
}

// DefaultSourceListResponse represents a list of default/system sources
type DefaultSourceListResponse struct {
	Count   int                 `json:"count" example:"3"`
	Sources []db.UpstreamSource `json:"sources"`
}

// SourceVersionListResponse represents a list of source versions
type SourceVersionListResponse struct {
	Count    int                `json:"count"`
	Total    int                `json:"total,omitempty"`
	Versions []db.SourceVersion `json:"versions"`
	SyncJob  *db.VersionSyncJob `json:"sync_job,omitempty"`
}

// SyncTriggerResponse represents the response when triggering a sync
type SyncTriggerResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// SyncStatusResponse represents the status of a sync job
type SyncStatusResponse struct {
	Job *db.VersionSyncJob `json:"job"`
}

// ClearVersionsResponse represents the response when clearing version cache
type ClearVersionsResponse struct {
	Message string `json:"message"`
}
