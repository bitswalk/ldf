package forge

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/db"
)

var log *logs.Logger

// SetLogger sets the logger for the forge package
func SetLogger(l *logs.Logger) {
	log = l
}

// ForgeType represents the type of git forge
type ForgeType string

const (
	ForgeTypeGitHub   ForgeType = "github"
	ForgeTypeGitLab   ForgeType = "gitlab"
	ForgeTypeGitea    ForgeType = "gitea"
	ForgeTypeCodeberg ForgeType = "codeberg"
	ForgeTypeForgejo  ForgeType = "forgejo"
	ForgeTypeGeneric  ForgeType = "generic"
)

// AllForgeTypes returns all available forge types for UI display
func AllForgeTypes() []ForgeType {
	return []ForgeType{
		ForgeTypeGitHub,
		ForgeTypeGitLab,
		ForgeTypeGitea,
		ForgeTypeCodeberg,
		ForgeTypeForgejo,
		ForgeTypeGeneric,
	}
}

// ForgeTypeInfo provides display information about a forge type
type ForgeTypeInfo struct {
	Type        ForgeType `json:"type"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
}

// GetForgeTypeInfo returns display information for all forge types
func GetForgeTypeInfo() []ForgeTypeInfo {
	return []ForgeTypeInfo{
		{ForgeTypeGitHub, "GitHub", "GitHub.com repositories"},
		{ForgeTypeGitLab, "GitLab", "GitLab.com or self-hosted GitLab instances"},
		{ForgeTypeGitea, "Gitea", "Gitea self-hosted instances"},
		{ForgeTypeCodeberg, "Codeberg", "Codeberg.org repositories"},
		{ForgeTypeForgejo, "Forgejo", "Forgejo self-hosted instances"},
		{ForgeTypeGeneric, "Generic", "Generic Git/HTTP sources (kernel.org style)"},
	}
}

// RepoInfo contains parsed repository information
type RepoInfo struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BaseURL    string `json:"base_url"`
	APIBaseURL string `json:"api_base_url"`
}

// DiscoveredVersion represents a version found from an upstream source
type DiscoveredVersion struct {
	Version      string
	VersionType  db.VersionType
	ReleaseDate  *time.Time
	DownloadURL  string
	IsStable     bool
	IsPrerelease bool // Upstream prerelease flag (from API)
}

// ForgeDefaults contains calculated defaults from a forge
type ForgeDefaults struct {
	URLTemplate   string `json:"url_template"`
	VersionFilter string `json:"version_filter"`
	FilterSource  string `json:"filter_source"` // "upstream" or "default"
}

// Provider defines the interface for forge-specific operations
type Provider interface {
	// Name returns the forge type name
	Name() ForgeType

	// Detect checks if this provider can handle the given URL
	Detect(url string) bool

	// ParseRepoInfo extracts repository information from a URL
	ParseRepoInfo(url string) (*RepoInfo, error)

	// DiscoverVersions fetches available versions from the upstream source
	DiscoverVersions(ctx context.Context, url string) ([]DiscoveredVersion, error)

	// DefaultURLTemplate returns the default URL template for this forge
	DefaultURLTemplate(repoInfo *RepoInfo) string

	// DefaultVersionFilter calculates the default version filter from upstream
	// This queries the API to determine which patterns should be excluded
	DefaultVersionFilter(ctx context.Context, repoInfo *RepoInfo) (string, error)

	// GetDefaults returns both URL template and version filter defaults
	GetDefaults(ctx context.Context, url string) (*ForgeDefaults, error)
}

// Registry manages forge providers
type Registry struct {
	providers  []Provider
	httpClient *http.Client
}

// NewRegistry creates a new forge provider registry with all built-in providers
func NewRegistry() *Registry {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	r := &Registry{
		httpClient: client,
		providers:  make([]Provider, 0),
	}

	// Register providers in detection priority order
	// More specific providers first, generic last
	r.Register(NewGitHubProvider(client))
	r.Register(NewCodebergProvider(client))
	r.Register(NewGitLabProvider(client))
	r.Register(NewGiteaProvider(client))
	r.Register(NewForgejoProvider(client))
	r.Register(NewGenericProvider(client))

	return r
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

// DetectForge detects the forge type for a given URL
func (r *Registry) DetectForge(url string) ForgeType {
	for _, p := range r.providers {
		if p.Detect(url) {
			return p.Name()
		}
	}
	return ForgeTypeGeneric
}

// GetProvider returns the provider for a given forge type
func (r *Registry) GetProvider(forgeType ForgeType) Provider {
	for _, p := range r.providers {
		if p.Name() == forgeType {
			return p
		}
	}
	// Fallback to generic
	return r.providers[len(r.providers)-1]
}

// GetProviderForURL returns the appropriate provider for a URL
func (r *Registry) GetProviderForURL(url string) Provider {
	forgeType := r.DetectForge(url)
	return r.GetProvider(forgeType)
}

// DetectAndGetDefaults detects forge type and returns defaults for a URL
func (r *Registry) DetectAndGetDefaults(ctx context.Context, url string) (ForgeType, *ForgeDefaults, error) {
	forgeType := r.DetectForge(url)
	provider := r.GetProvider(forgeType)

	defaults, err := provider.GetDefaults(ctx, url)
	if err != nil {
		return forgeType, nil, err
	}

	return forgeType, defaults, nil
}

// DiscoverVersions discovers versions using the appropriate provider
func (r *Registry) DiscoverVersions(ctx context.Context, url string, forgeType ForgeType) ([]DiscoveredVersion, error) {
	provider := r.GetProvider(forgeType)
	return provider.DiscoverVersions(ctx, url)
}

// Helper functions

// normalizeVersion removes common prefixes from version strings
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

// isVersionString checks if a string looks like a version number
func isVersionString(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

// isPrerelease checks if a version string indicates a prerelease
func isPrerelease(version string) bool {
	lower := strings.ToLower(version)
	return strings.Contains(lower, "-rc") ||
		strings.Contains(lower, "-alpha") ||
		strings.Contains(lower, "-beta") ||
		strings.Contains(lower, "-dev") ||
		strings.Contains(lower, "-pre") ||
		strings.Contains(lower, ".rc") ||
		strings.Contains(lower, "_rc") ||
		strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "beta")
}

// determineVersionType determines the version type based on prerelease status
func determineVersionType(version string, isUpstreamPrerelease bool) db.VersionType {
	if isUpstreamPrerelease || isPrerelease(version) {
		return db.VersionTypeMainline
	}
	return db.VersionTypeStable
}

// extractExcludePatterns analyzes versions to build exclude patterns
func extractExcludePatterns(versions []DiscoveredVersion) []string {
	patterns := make(map[string]bool)

	for _, v := range versions {
		if !v.IsPrerelease {
			continue
		}

		lower := strings.ToLower(v.Version)

		// Check for common prerelease patterns
		if strings.Contains(lower, "-rc") || strings.Contains(lower, ".rc") || strings.Contains(lower, "_rc") {
			patterns["!*-rc*"] = true
			patterns["!*.rc*"] = true
			patterns["!*_rc*"] = true
		}
		if strings.Contains(lower, "-alpha") || strings.Contains(lower, "alpha") {
			patterns["!*alpha*"] = true
		}
		if strings.Contains(lower, "-beta") || strings.Contains(lower, "beta") {
			patterns["!*beta*"] = true
		}
		if strings.Contains(lower, "-dev") {
			patterns["!*-dev*"] = true
		}
		if strings.Contains(lower, "-pre") {
			patterns["!*-pre*"] = true
		}
		if strings.Contains(lower, "-snapshot") {
			patterns["!*-snapshot*"] = true
		}
		if strings.Contains(lower, "-nightly") {
			patterns["!*-nightly*"] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(patterns))
	for p := range patterns {
		result = append(result, p)
	}

	return result
}
