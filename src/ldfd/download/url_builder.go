package download

import (
	"fmt"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// URLBuilder handles URL construction from templates
type URLBuilder struct {
	componentRepo *db.ComponentRepository
}

// NewURLBuilder creates a new URL builder
func NewURLBuilder(componentRepo *db.ComponentRepository) *URLBuilder {
	return &URLBuilder{
		componentRepo: componentRepo,
	}
}

// BuildURL constructs the final download URL from a source and component
func (b *URLBuilder) BuildURL(source *db.UpstreamSource, component *db.Component, version string) (string, error) {
	if source == nil {
		return "", fmt.Errorf("source is nil")
	}
	if component == nil {
		return "", fmt.Errorf("component is nil")
	}
	if version == "" {
		return "", fmt.Errorf("version is empty")
	}

	baseURL := b.normalizeBaseURL(source.URL)

	// Determine which template to use
	template := b.selectTemplate(source, component)
	if template == "" {
		// No template available, use base URL directly
		return baseURL, nil
	}

	// Apply template
	return b.applyTemplate(template, baseURL, version), nil
}

// selectTemplate determines which template to use based on source and component
func (b *URLBuilder) selectTemplate(source *db.UpstreamSource, component *db.Component) string {
	// Priority:
	// 1. Source-specific URL template (user-defined override)
	// 2. GitHub normalized template (if URL is GitHub)
	// 3. Component default URL template
	// 4. Empty (use base URL as-is)

	if source.URLTemplate != "" {
		return source.URLTemplate
	}

	if b.isGitHubURL(source.URL) && component.GitHubNormalizedTemplate != "" {
		return component.GitHubNormalizedTemplate
	}

	if component.DefaultURLTemplate != "" {
		return component.DefaultURLTemplate
	}

	return ""
}

// applyTemplate replaces placeholders in the template with actual values
// Available placeholders:
//   - {base_url}    : The source base URL
//   - {version}     : Full version string (e.g., "1.0.0", "6.12.5")
//   - {tag}         : Version with 'v' prefix (e.g., "v1.0.0", "v6.12.5")
//   - {tag_short}   : Short tag with major.minor only (e.g., "v1.0", "v6.12")
//   - {tag_compact} : Compact tag without dots (e.g., "v100", "v6125")
//   - {major}       : Major version number only (e.g., "1", "6")
//   - {minor}       : Minor version number only (e.g., "0", "12")
//   - {patch}       : Patch version number only (e.g., "0", "5")
//   - {major_x}     : Major version with .x suffix for kernel.org style (e.g., "6.x")
func (b *URLBuilder) applyTemplate(template, baseURL, version string) string {
	result := template

	// Parse version components
	major, minor, patch := parseVersionComponents(version)

	// Replace basic placeholders
	result = strings.ReplaceAll(result, "{base_url}", baseURL)
	result = strings.ReplaceAll(result, "{version}", version)
	result = strings.ReplaceAll(result, "{tag}", "v"+version)

	// Short tag: v{major}.{minor} (e.g., "v1.0", "v6.12")
	tagShort := fmt.Sprintf("v%s.%s", major, minor)
	result = strings.ReplaceAll(result, "{tag_short}", tagShort)

	// Compact tag: v{major}{minor}{patch} without dots (e.g., "v100", "v259")
	tagCompact := "v" + buildCompactVersion(version)
	result = strings.ReplaceAll(result, "{tag_compact}", tagCompact)

	// Individual version components
	result = strings.ReplaceAll(result, "{major}", major)
	result = strings.ReplaceAll(result, "{minor}", minor)
	result = strings.ReplaceAll(result, "{patch}", patch)

	// Major with .x suffix for kernel.org style URLs (e.g., "6.x")
	result = strings.ReplaceAll(result, "{major_x}", major+".x")

	return result
}

// normalizeBaseURL cleans up the base URL
func (b *URLBuilder) normalizeBaseURL(url string) string {
	// Remove trailing slashes
	url = strings.TrimSuffix(url, "/")

	// Remove .git suffix for GitHub URLs
	url = strings.TrimSuffix(url, ".git")

	return url
}

// isGitHubURL checks if the URL is a GitHub URL
func (b *URLBuilder) isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com")
}

// IsGitLabURL checks if the URL is a GitLab URL
func (b *URLBuilder) IsGitLabURL(url string) bool {
	return strings.Contains(url, "gitlab.com") || strings.Contains(url, "gitlab.")
}

// parseVersionComponents extracts major, minor, and patch from a version string
// Returns ("0", "0", "0") for invalid versions
func parseVersionComponents(version string) (major, minor, patch string) {
	parts := strings.Split(version, ".")

	major = "0"
	minor = "0"
	patch = "0"

	if len(parts) > 0 && parts[0] != "" {
		major = parts[0]
	}
	if len(parts) > 1 && parts[1] != "" {
		minor = parts[1]
	}
	if len(parts) > 2 && parts[2] != "" {
		patch = parts[2]
	}

	return major, minor, patch
}

// buildCompactVersion creates a compact version string without dots
// Examples:
//   - "1.0.0" -> "100"
//   - "6.12.5" -> "6125"
//   - "259" -> "259" (already compact, like systemd)
func buildCompactVersion(version string) string {
	// Remove all dots to create compact version
	return strings.ReplaceAll(version, ".", "")
}

// BuildGitCloneURL constructs a URL suitable for git clone operations
func (b *URLBuilder) BuildGitCloneURL(source *db.UpstreamSource) string {
	baseURL := b.normalizeBaseURL(source.URL)

	// Ensure .git suffix for clone operations
	if !strings.HasSuffix(baseURL, ".git") {
		baseURL = baseURL + ".git"
	}

	return baseURL
}

// BuildGitRef constructs the git reference (tag or branch) for checkout
func (b *URLBuilder) BuildGitRef(version string) string {
	// Default to tag format v{version}
	return "v" + version
}

// PreviewURL generates a preview of what the final URL would look like
func (b *URLBuilder) PreviewURL(source *db.UpstreamSource, component *db.Component, exampleVersion string) (string, error) {
	if exampleVersion == "" {
		exampleVersion = "1.0.0"
	}
	return b.BuildURL(source, component, exampleVersion)
}
