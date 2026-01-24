package forge

import (
	"github.com/bitswalk/ldf/src/ldfd/forge"
)

// Handler handles forge-related HTTP requests
type Handler struct {
	registry *forge.Registry
}

// Config contains configuration options for the Handler
type Config struct {
	Registry *forge.Registry
}

// DetectRequest represents a request to detect forge type
type DetectRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// DetectResponse represents the response from forge detection
type DetectResponse struct {
	ForgeType  forge.ForgeType       `json:"forge_type"`
	RepoInfo   *forge.RepoInfo       `json:"repo_info,omitempty"`
	Defaults   *forge.ForgeDefaults  `json:"defaults,omitempty"`
	ForgeTypes []forge.ForgeTypeInfo `json:"forge_types"`
}

// PreviewFilterRequest represents a request to preview version filtering
type PreviewFilterRequest struct {
	URL           string `json:"url" binding:"required,url"`
	ForgeType     string `json:"forge_type"`
	VersionFilter string `json:"version_filter"`
}

// VersionPreview represents a version with filter result
type VersionPreview struct {
	Version      string `json:"version"`
	Included     bool   `json:"included"`
	Reason       string `json:"reason,omitempty"`
	IsPrerelease bool   `json:"is_prerelease"`
}

// PreviewFilterResponse represents the response from filter preview
type PreviewFilterResponse struct {
	TotalVersions    int              `json:"total_versions"`
	IncludedVersions int              `json:"included_versions"`
	ExcludedVersions int              `json:"excluded_versions"`
	Versions         []VersionPreview `json:"versions"`
	AppliedFilter    string           `json:"applied_filter"`
	FilterSource     string           `json:"filter_source"` // "custom", "upstream", or "default"
}

// CommonFiltersResponse represents available common filter presets
type CommonFiltersResponse struct {
	Filters map[string]string `json:"filters"`
}
