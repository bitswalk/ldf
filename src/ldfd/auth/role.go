package auth

import (
	"time"

	"github.com/google/uuid"
)

// Default role IDs for the three system roles (fixed UUIDs for consistency)
const (
	RoleIDRoot      = "908b291e-61fb-4d95-98db-0b76c0afd6b4"
	RoleIDDeveloper = "91db9f27-b8a2-4452-9b80-5f6ab1096da8"
	RoleIDAnonymous = "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0"
)

// RolePermissions defines the permission flags for a role
type RolePermissions struct {
	CanRead   bool `json:"can_read"`
	CanWrite  bool `json:"can_write"`
	CanDelete bool `json:"can_delete"`
	CanAdmin  bool `json:"can_admin"`
}

// RoleRecord represents a role stored in the database
type RoleRecord struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Permissions  RolePermissions `json:"permissions"`
	IsSystem     bool            `json:"is_system"`
	ParentRoleID string          `json:"parent_role_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// NewRole creates a new custom role
func NewRole(name, description string, permissions RolePermissions, parentRoleID string) *RoleRecord {
	now := time.Now().UTC()
	return &RoleRecord{
		ID:           uuid.New().String(),
		Name:         name,
		Description:  description,
		Permissions:  permissions,
		IsSystem:     false,
		ParentRoleID: parentRoleID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// HasWriteAccess returns true if the role has write permissions
func (r *RoleRecord) HasWriteAccess() bool {
	return r.Permissions.CanWrite
}

// HasDeleteAccess returns true if the role has delete permissions
func (r *RoleRecord) HasDeleteAccess() bool {
	return r.Permissions.CanDelete
}

// HasAdminAccess returns true if the role has admin permissions
func (r *RoleRecord) HasAdminAccess() bool {
	return r.Permissions.CanAdmin
}

// GetDefaultRoleID returns the ID of the default role for new users
func GetDefaultRoleID() string {
	return RoleIDDeveloper
}

// IsSystemRoleName checks if a role name is a system role
func IsSystemRoleName(name string) bool {
	return name == "root" || name == "developer" || name == "anonymous"
}
