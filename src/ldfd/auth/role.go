package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrRoleNotFound is returned when a role is not found
	ErrRoleNotFound = errors.New("role not found")
	// ErrRoleNameExists is returned when trying to create a role with a name that already exists
	ErrRoleNameExists = errors.New("role name already exists")
	// ErrSystemRole is returned when trying to modify or delete a system role
	ErrSystemRole = errors.New("cannot modify system role")
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

// GetRoleByID retrieves a role by ID
func (r *Repository) GetRoleByID(id string) (*RoleRecord, error) {
	var role RoleRecord
	var description, parentRoleID sql.NullString

	err := r.db.QueryRow(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles WHERE id = ?
	`, id).Scan(
		&role.ID, &role.Name, &description,
		&role.Permissions.CanRead, &role.Permissions.CanWrite, &role.Permissions.CanDelete, &role.Permissions.CanAdmin,
		&role.IsSystem, &parentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if description.Valid {
		role.Description = description.String
	}
	if parentRoleID.Valid {
		role.ParentRoleID = parentRoleID.String
	}

	return &role, nil
}

// GetRoleByName retrieves a role by name
func (r *Repository) GetRoleByName(name string) (*RoleRecord, error) {
	var role RoleRecord
	var description, parentRoleID sql.NullString

	err := r.db.QueryRow(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles WHERE name = ?
	`, name).Scan(
		&role.ID, &role.Name, &description,
		&role.Permissions.CanRead, &role.Permissions.CanWrite, &role.Permissions.CanDelete, &role.Permissions.CanAdmin,
		&role.IsSystem, &parentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if description.Valid {
		role.Description = description.String
	}
	if parentRoleID.Valid {
		role.ParentRoleID = parentRoleID.String
	}

	return &role, nil
}

// ListRoles retrieves all roles
func (r *Repository) ListRoles() ([]RoleRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles ORDER BY is_system DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []RoleRecord
	for rows.Next() {
		var role RoleRecord
		var description, parentRoleID sql.NullString

		err := rows.Scan(
			&role.ID, &role.Name, &description,
			&role.Permissions.CanRead, &role.Permissions.CanWrite, &role.Permissions.CanDelete, &role.Permissions.CanAdmin,
			&role.IsSystem, &parentRoleID, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		if description.Valid {
			role.Description = description.String
		}
		if parentRoleID.Valid {
			role.ParentRoleID = parentRoleID.String
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// CreateRole creates a new custom role
func (r *Repository) CreateRole(role *RoleRecord) error {
	// Check if name already exists
	_, err := r.GetRoleByName(role.Name)
	if err == nil {
		return ErrRoleNameExists
	}
	if !errors.Is(err, ErrRoleNotFound) {
		return err
	}

	// Handle optional parent_role_id
	var parentRoleID interface{}
	if role.ParentRoleID != "" {
		parentRoleID = role.ParentRoleID
	}

	_, err = r.db.Exec(`
		INSERT INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, role.ID, role.Name, role.Description,
		role.Permissions.CanRead, role.Permissions.CanWrite, role.Permissions.CanDelete, role.Permissions.CanAdmin,
		role.IsSystem, parentRoleID, role.CreatedAt, role.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// UpdateRole updates a custom role (system roles cannot be modified)
func (r *Repository) UpdateRole(role *RoleRecord) error {
	// Check if role exists and is not a system role
	existing, err := r.GetRoleByID(role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrSystemRole
	}

	// Handle optional parent_role_id
	var parentRoleID interface{}
	if role.ParentRoleID != "" {
		parentRoleID = role.ParentRoleID
	}

	role.UpdatedAt = time.Now().UTC()

	result, err := r.db.Exec(`
		UPDATE roles SET name = ?, description = ?, can_read = ?, can_write = ?, can_delete = ?, can_admin = ?, parent_role_id = ?, updated_at = ?
		WHERE id = ? AND is_system = 0
	`, role.Name, role.Description,
		role.Permissions.CanRead, role.Permissions.CanWrite, role.Permissions.CanDelete, role.Permissions.CanAdmin,
		parentRoleID, role.UpdatedAt, role.ID)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrSystemRole
	}

	return nil
}

// DeleteRole deletes a custom role (system roles cannot be deleted)
func (r *Repository) DeleteRole(id string) error {
	// Check if role exists and is not a system role
	existing, err := r.GetRoleByID(id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrSystemRole
	}

	result, err := r.db.Exec("DELETE FROM roles WHERE id = ? AND is_system = 0", id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrSystemRole
	}

	return nil
}

// GetDefaultRoleID returns the ID of the default role for new users
func GetDefaultRoleID() string {
	return RoleIDDeveloper
}

// IsSystemRoleName checks if a role name is a system role
func IsSystemRoleName(name string) bool {
	return name == "root" || name == "developer" || name == "anonymous"
}
