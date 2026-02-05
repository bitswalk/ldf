package client

import (
	"context"
	"fmt"
)

// Component represents a component resource
type Component struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url"`
	License     string `json:"license"`
	IsSystem    bool   `json:"is_system"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ComponentListResponse represents a list of components
type ComponentListResponse struct {
	Count      int         `json:"count"`
	Components []Component `json:"components"`
}

// ComponentVersionsResponse represents versions for a component
type ComponentVersionsResponse struct {
	ComponentID string   `json:"component_id"`
	Versions    []string `json:"versions"`
}

// ResolvedVersionResponse represents a resolved version
type ResolvedVersionResponse struct {
	ComponentID string `json:"component_id"`
	Version     string `json:"version"`
	ResolvedURL string `json:"resolved_url"`
}

// CreateComponentRequest represents the request to create a component
type CreateComponentRequest struct {
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	License     string `json:"license,omitempty"`
}

// UpdateComponentRequest represents the request to update a component
type UpdateComponentRequest struct {
	Name        string `json:"name,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	License     string `json:"license,omitempty"`
}

// ListComponents returns all components
func (c *Client) ListComponents(ctx context.Context) (*ComponentListResponse, error) {
	var resp ComponentListResponse
	if err := c.Get(ctx, "/v1/components", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetComponent returns a single component by ID
func (c *Client) GetComponent(ctx context.Context, id string) (*Component, error) {
	var resp Component
	if err := c.Get(ctx, fmt.Sprintf("/v1/components/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateComponent creates a new component
func (c *Client) CreateComponent(ctx context.Context, req *CreateComponentRequest) (*Component, error) {
	var resp Component
	if err := c.Post(ctx, "/v1/components", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateComponent updates an existing component
func (c *Client) UpdateComponent(ctx context.Context, id string, req *UpdateComponentRequest) (*Component, error) {
	var resp Component
	if err := c.Put(ctx, fmt.Sprintf("/v1/components/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteComponent deletes a component
func (c *Client) DeleteComponent(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/components/%s", id), nil)
}

// GetComponentVersions returns versions for a component
func (c *Client) GetComponentVersions(ctx context.Context, id string) (*ComponentVersionsResponse, error) {
	var resp ComponentVersionsResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/components/%s/versions", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResolveVersion resolves the best version for a component
func (c *Client) ResolveVersion(ctx context.Context, id string) (*ResolvedVersionResponse, error) {
	var resp ResolvedVersionResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/components/%s/resolve-version", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
