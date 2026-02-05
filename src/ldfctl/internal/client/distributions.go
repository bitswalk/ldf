package client

import (
	"context"
	"fmt"
)

// Distribution represents a distribution resource
type Distribution struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Version    string      `json:"version"`
	Status     string      `json:"status"`
	Visibility string      `json:"visibility"`
	SourceURL  string      `json:"source_url"`
	Checksum   string      `json:"checksum"`
	SizeBytes  int64       `json:"size_bytes"`
	Config     interface{} `json:"config,omitempty"`
	CreatedAt  string      `json:"created_at"`
	UpdatedAt  string      `json:"updated_at"`
}

// DistributionListResponse represents a list of distributions
type DistributionListResponse struct {
	Count         int            `json:"count"`
	Distributions []Distribution `json:"distributions"`
}

// DistributionStatsResponse represents distribution statistics
type DistributionStatsResponse struct {
	Total int64            `json:"total"`
	Stats map[string]int64 `json:"stats"`
}

// CreateDistributionRequest represents the request to create a distribution
type CreateDistributionRequest struct {
	Name       string      `json:"name"`
	Version    string      `json:"version,omitempty"`
	Visibility string      `json:"visibility,omitempty"`
	Config     interface{} `json:"config,omitempty"`
	SourceURL  string      `json:"source_url,omitempty"`
	Checksum   string      `json:"checksum,omitempty"`
}

// UpdateDistributionRequest represents the request to update a distribution
type UpdateDistributionRequest struct {
	Name       string      `json:"name,omitempty"`
	Version    string      `json:"version,omitempty"`
	Status     string      `json:"status,omitempty"`
	Visibility string      `json:"visibility,omitempty"`
	SourceURL  string      `json:"source_url,omitempty"`
	Checksum   string      `json:"checksum,omitempty"`
	SizeBytes  int64       `json:"size_bytes,omitempty"`
	Config     interface{} `json:"config,omitempty"`
}

// ListDistributions returns all distributions
func (c *Client) ListDistributions(ctx context.Context, opts *ListOptions) (*DistributionListResponse, error) {
	var resp DistributionListResponse
	if err := c.Get(ctx, "/v1/distributions"+opts.QueryString(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDistribution returns a single distribution by ID
func (c *Client) GetDistribution(ctx context.Context, id string) (*Distribution, error) {
	var resp Distribution
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateDistribution creates a new distribution
func (c *Client) CreateDistribution(ctx context.Context, req *CreateDistributionRequest) (*Distribution, error) {
	var resp Distribution
	if err := c.Post(ctx, "/v1/distributions", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateDistribution updates an existing distribution
func (c *Client) UpdateDistribution(ctx context.Context, id string, req *UpdateDistributionRequest) (*Distribution, error) {
	var resp Distribution
	if err := c.Put(ctx, fmt.Sprintf("/v1/distributions/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteDistribution deletes a distribution
func (c *Client) DeleteDistribution(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/distributions/%s", id), nil)
}

// GetDistributionLogs returns logs for a distribution
func (c *Client) GetDistributionLogs(ctx context.Context, id string) (interface{}, error) {
	var resp interface{}
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/logs", id), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetDistributionStats returns distribution statistics
func (c *Client) GetDistributionStats(ctx context.Context) (*DistributionStatsResponse, error) {
	var resp DistributionStatsResponse
	if err := c.Get(ctx, "/v1/distributions/stats", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDeletionPreview returns a preview of what would be deleted
func (c *Client) GetDeletionPreview(ctx context.Context, id string) (interface{}, error) {
	var resp interface{}
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/deletion-preview", id), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
