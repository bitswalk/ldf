package client

import "context"

// DetectRequest represents a forge detection request
type DetectRequest struct {
	URL string `json:"url"`
}

// DetectResponse represents a forge detection response
type DetectResponse struct {
	ForgeType  string      `json:"forge_type"`
	RepoInfo   interface{} `json:"repo_info,omitempty"`
	Defaults   interface{} `json:"defaults,omitempty"`
	ForgeTypes interface{} `json:"forge_types"`
}

// PreviewFilterRequest represents a filter preview request
type PreviewFilterRequest struct {
	URL           string `json:"url"`
	ForgeType     string `json:"forge_type,omitempty"`
	VersionFilter string `json:"version_filter,omitempty"`
}

// VersionPreview represents a version with filter result
type VersionPreview struct {
	Version      string `json:"version"`
	Included     bool   `json:"included"`
	Reason       string `json:"reason,omitempty"`
	IsPrerelease bool   `json:"is_prerelease"`
}

// PreviewFilterResponse represents a filter preview response
type PreviewFilterResponse struct {
	TotalVersions    int              `json:"total_versions"`
	IncludedVersions int              `json:"included_versions"`
	ExcludedVersions int              `json:"excluded_versions"`
	Versions         []VersionPreview `json:"versions"`
	AppliedFilter    string           `json:"applied_filter"`
	FilterSource     string           `json:"filter_source"`
}

// ForgeTypesResponse represents the list of forge types
type ForgeTypesResponse struct {
	ForgeTypes interface{} `json:"forge_types"`
}

// CommonFiltersResponse represents the common filter presets
type CommonFiltersResponse struct {
	Filters map[string]string `json:"filters"`
}

// DetectForge detects the forge type for a URL
func (c *Client) DetectForge(ctx context.Context, url string) (*DetectResponse, error) {
	req := DetectRequest{URL: url}
	var resp DetectResponse
	if err := c.Post(ctx, "/v1/forge/detect", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PreviewFilter previews version filtering for a URL
func (c *Client) PreviewFilter(ctx context.Context, req *PreviewFilterRequest) (*PreviewFilterResponse, error) {
	var resp PreviewFilterResponse
	if err := c.Post(ctx, "/v1/forge/preview-filter", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListForgeTypes returns all supported forge types
func (c *Client) ListForgeTypes(ctx context.Context) (*ForgeTypesResponse, error) {
	var resp ForgeTypesResponse
	if err := c.Get(ctx, "/v1/forge/types", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCommonFilters returns the common filter presets
func (c *Client) GetCommonFilters(ctx context.Context) (*CommonFiltersResponse, error) {
	var resp CommonFiltersResponse
	if err := c.Get(ctx, "/v1/forge/common-filters", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
