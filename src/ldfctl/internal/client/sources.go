package client

import (
	"context"
	"fmt"
)

// Source represents a source resource
type Source struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	ForgeType      string `json:"forge_type"`
	ComponentID    string `json:"component_id"`
	IsSystem       bool   `json:"is_system"`
	OwnerID        string `json:"owner_id"`
	VersionFilter  string `json:"version_filter"`
	LastSyncAt     string `json:"last_sync_at"`
	LastSyncStatus string `json:"last_sync_status"`
	VersionCount   int    `json:"version_count"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// SourceListResponse represents a list of sources
type SourceListResponse struct {
	Count   int      `json:"count"`
	Sources []Source `json:"sources"`
}

// SourceVersion represents a discovered version
type SourceVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Type    string `json:"type"`
}

// SourceVersionListResponse represents a list of versions
type SourceVersionListResponse struct {
	Count    int             `json:"count"`
	Versions []SourceVersion `json:"versions"`
}

// SyncTriggerResponse represents the response from triggering a sync
type SyncTriggerResponse struct {
	Message  string `json:"message"`
	SourceID string `json:"source_id"`
	Status   string `json:"status"`
}

// SyncStatusResponse represents the sync status
type SyncStatusResponse struct {
	SourceID     string `json:"source_id"`
	Status       string `json:"status"`
	LastSyncAt   string `json:"last_sync_at"`
	VersionCount int    `json:"version_count"`
	Error        string `json:"error,omitempty"`
}

// CreateSourceRequest represents the request to create a source
type CreateSourceRequest struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	ComponentID   string `json:"component_id"`
	VersionFilter string `json:"version_filter,omitempty"`
}

// UpdateSourceRequest represents the request to update a source
type UpdateSourceRequest struct {
	Name          string `json:"name,omitempty"`
	URL           string `json:"url,omitempty"`
	VersionFilter string `json:"version_filter,omitempty"`
}

// ListSources returns all sources
func (c *Client) ListSources(ctx context.Context) (*SourceListResponse, error) {
	var resp SourceListResponse
	if err := c.Get(ctx, "/v1/sources", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSource returns a single source by ID
func (c *Client) GetSource(ctx context.Context, id string) (*Source, error) {
	var resp Source
	if err := c.Get(ctx, fmt.Sprintf("/v1/sources/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateSource creates a new user source
func (c *Client) CreateSource(ctx context.Context, req *CreateSourceRequest) (*Source, error) {
	var resp Source
	if err := c.Post(ctx, "/v1/sources", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSource updates an existing source
func (c *Client) UpdateSource(ctx context.Context, id string, req *UpdateSourceRequest) (*Source, error) {
	var resp Source
	if err := c.Put(ctx, fmt.Sprintf("/v1/sources/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSource deletes a source
func (c *Client) DeleteSource(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/sources/%s", id), nil)
}

// ListSourceVersions returns discovered versions for a source
func (c *Client) ListSourceVersions(ctx context.Context, id string) (*SourceVersionListResponse, error) {
	var resp SourceVersionListResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/sources/%s/versions", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SyncSource triggers a version sync for a source
func (c *Client) SyncSource(ctx context.Context, id string) (*SyncTriggerResponse, error) {
	var resp SyncTriggerResponse
	if err := c.Post(ctx, fmt.Sprintf("/v1/sources/%s/sync", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSyncStatus returns the sync status for a source
func (c *Client) GetSyncStatus(ctx context.Context, id string) (*SyncStatusResponse, error) {
	var resp SyncStatusResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/sources/%s/sync/status", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ClearSourceVersions deletes all cached versions for a source
func (c *Client) ClearSourceVersions(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/sources/%s/versions", id), nil)
}
