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
func (b *URLBuilder) BuildURL(source *db.Source, component *db.Component, version string) (string, error) {
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
func (b *URLBuilder) selectTemplate(source *db.Source, component *db.Component) string {
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
func (b *URLBuilder) applyTemplate(template, baseURL, version string) string {
	result := template

	// Replace placeholders
	result = strings.ReplaceAll(result, "{base_url}", baseURL)
	result = strings.ReplaceAll(result, "{version}", version)
	result = strings.ReplaceAll(result, "{tag}", "v"+version)

	// Handle major version extraction for kernel-style URLs
	// e.g., kernel 6.12.0 -> major is 6
	if strings.Contains(result, "{major}") {
		major := extractMajorVersion(version)
		result = strings.ReplaceAll(result, "{major}", major)
	}

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

// extractMajorVersion extracts the major version number from a version string
func extractMajorVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return version
}

// BuildGitCloneURL constructs a URL suitable for git clone operations
func (b *URLBuilder) BuildGitCloneURL(source *db.Source) string {
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
func (b *URLBuilder) PreviewURL(source *db.Source, component *db.Component, exampleVersion string) (string, error) {
	if exampleVersion == "" {
		exampleVersion = "1.0.0"
	}
	return b.BuildURL(source, component, exampleVersion)
}
