package components

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles component-related HTTP requests
type Handler struct {
	componentRepo     *db.ComponentRepository
	sourceVersionRepo *db.SourceVersionRepository
}

// Config contains configuration options for the Handler
type Config struct {
	ComponentRepo     *db.ComponentRepository
	SourceVersionRepo *db.SourceVersionRepository
}

// ComponentListResponse represents a list of components
type ComponentListResponse struct {
	Count      int            `json:"count" example:"16"`
	Components []db.Component `json:"components"`
}

// CreateComponentRequest represents the request to create a component
type CreateComponentRequest struct {
	Name                     string `json:"name" binding:"required" example:"kernel"`
	Category                 string `json:"category" binding:"required" example:"core"`
	DisplayName              string `json:"display_name" binding:"required" example:"Linux Kernel"`
	Description              string `json:"description" example:"The Linux kernel source code"`
	ArtifactPattern          string `json:"artifact_pattern" example:"linux-{version}.tar.xz"`
	DefaultURLTemplate       string `json:"default_url_template" example:"{base_url}/linux-{version}.tar.xz"`
	GitHubNormalizedTemplate string `json:"github_normalized_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	IsOptional               *bool  `json:"is_optional" example:"false"`
	DefaultVersion           string `json:"default_version" example:"6.12.1"`
	DefaultVersionRule       string `json:"default_version_rule" example:"latest-stable"`
}

// UpdateComponentRequest represents the request to update a component
type UpdateComponentRequest struct {
	Name                     string `json:"name" example:"kernel"`
	Category                 string `json:"category" example:"core"`
	DisplayName              string `json:"display_name" example:"Linux Kernel"`
	Description              string `json:"description" example:"The Linux kernel source code"`
	ArtifactPattern          string `json:"artifact_pattern" example:"linux-{version}.tar.xz"`
	DefaultURLTemplate       string `json:"default_url_template" example:"{base_url}/linux-{version}.tar.xz"`
	GitHubNormalizedTemplate string `json:"github_normalized_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	IsOptional               *bool  `json:"is_optional" example:"false"`
	DefaultVersion           string `json:"default_version" example:"6.12.1"`
	DefaultVersionRule       string `json:"default_version_rule" example:"latest-stable"`
}

// ComponentVersionsResponse represents a paginated list of versions for a component
type ComponentVersionsResponse struct {
	Versions []db.SourceVersion `json:"versions"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

// ResolvedVersionResponse represents a resolved version from a rule
type ResolvedVersionResponse struct {
	Rule            string            `json:"rule"`
	ResolvedVersion string            `json:"resolved_version"`
	Version         *db.SourceVersion `json:"version,omitempty"`
}
