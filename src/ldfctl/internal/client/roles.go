package client

import (
	"context"
	"fmt"
)

// Role represents a user role
type Role struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsSystem    bool            `json:"is_system"`
	ParentID    string          `json:"parent_role_id"`
	Permissions RolePermissions `json:"permissions"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// RolePermissions represents role permission flags
type RolePermissions struct {
	CanRead   bool `json:"can_read"`
	CanWrite  bool `json:"can_write"`
	CanDelete bool `json:"can_delete"`
	CanAdmin  bool `json:"can_admin"`
}

// RoleListResponse represents a list of roles
type RoleListResponse struct {
	Roles []Role `json:"roles"`
}

// RoleResponse wraps a single role
type RoleResponse struct {
	Role Role `json:"role"`
}

// CreateRoleRequest represents the request to create a role
type CreateRoleRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ParentRoleID string `json:"parent_role_id,omitempty"`
	Permissions  struct {
		CanRead   bool `json:"can_read"`
		CanWrite  bool `json:"can_write"`
		CanDelete bool `json:"can_delete"`
		CanAdmin  bool `json:"can_admin"`
	} `json:"permissions"`
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Permissions *struct {
		CanRead   bool `json:"can_read"`
		CanWrite  bool `json:"can_write"`
		CanDelete bool `json:"can_delete"`
		CanAdmin  bool `json:"can_admin"`
	} `json:"permissions,omitempty"`
}

// ListRoles returns all roles
func (c *Client) ListRoles(ctx context.Context) (*RoleListResponse, error) {
	var resp RoleListResponse
	if err := c.Get(ctx, "/v1/roles", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRole returns a single role by ID
func (c *Client) GetRole(ctx context.Context, id string) (*RoleResponse, error) {
	var resp RoleResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/roles/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateRole creates a new role
func (c *Client) CreateRole(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
	var resp RoleResponse
	if err := c.Post(ctx, "/v1/roles", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateRole updates an existing role
func (c *Client) UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*RoleResponse, error) {
	var resp RoleResponse
	if err := c.Put(ctx, fmt.Sprintf("/v1/roles/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteRole deletes a role
func (c *Client) DeleteRole(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/roles/%s", id), nil)
}
