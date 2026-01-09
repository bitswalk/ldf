package auth

import (
	"database/sql"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/google/uuid"
)

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

// GetDefaultRoleID returns the ID of the default role for new users
func GetDefaultRoleID() string {
	return RoleIDDeveloper
}

// IsSystemRoleName checks if a role name is a system role
func IsSystemRoleName(name string) bool {
	return name == "root" || name == "developer" || name == "anonymous"
}

// GetRoleByID retrieves a role by ID
func (m *UserManager) GetRoleByID(id string) (*RoleRecord, error) {
	var role RoleRecord
	var description, parentRoleID sql.NullString

	err := m.db.QueryRow(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles WHERE id = ?
	`, id).Scan(
		&role.ID, &role.Name, &description,
		&role.Permissions.CanRead, &role.Permissions.CanWrite, &role.Permissions.CanDelete, &role.Permissions.CanAdmin,
		&role.IsSystem, &parentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRoleNotFound
	}
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
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
func (m *UserManager) GetRoleByName(name string) (*RoleRecord, error) {
	var role RoleRecord
	var description, parentRoleID sql.NullString

	err := m.db.QueryRow(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles WHERE name = ?
	`, name).Scan(
		&role.ID, &role.Name, &description,
		&role.Permissions.CanRead, &role.Permissions.CanWrite, &role.Permissions.CanDelete, &role.Permissions.CanAdmin,
		&role.IsSystem, &parentRoleID, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrRoleNotFound
	}
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
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
func (m *UserManager) ListRoles() ([]RoleRecord, error) {
	rows, err := m.db.Query(`
		SELECT id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at
		FROM roles ORDER BY is_system DESC, name ASC
	`)
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
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
			return nil, errors.ErrDatabaseQuery.WithCause(err)
		}

		if description.Valid {
			role.Description = description.String
		}
		if parentRoleID.Valid {
			role.ParentRoleID = parentRoleID.String
		}

		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
	}

	return roles, nil
}

// CreateRole creates a new custom role in the database
func (m *UserManager) CreateRole(role *RoleRecord) error {
	// Check if name already exists
	_, err := m.GetRoleByName(role.Name)
	if err == nil {
		return errors.ErrRoleAlreadyExists
	}
	if !errors.Is(err, errors.ErrRoleNotFound) {
		return err
	}

	// Handle optional parent_role_id
	var parentRoleID interface{}
	if role.ParentRoleID != "" {
		parentRoleID = role.ParentRoleID
	}

	_, err = m.db.Exec(`
		INSERT INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system, parent_role_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, role.ID, role.Name, role.Description,
		role.Permissions.CanRead, role.Permissions.CanWrite, role.Permissions.CanDelete, role.Permissions.CanAdmin,
		role.IsSystem, parentRoleID, role.CreatedAt, role.UpdatedAt)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// UpdateRole updates a custom role (system roles cannot be modified)
func (m *UserManager) UpdateRole(role *RoleRecord) error {
	// Check if role exists and is not a system role
	existing, err := m.GetRoleByID(role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return errors.ErrSystemRoleModification
	}

	// Handle optional parent_role_id
	var parentRoleID interface{}
	if role.ParentRoleID != "" {
		parentRoleID = role.ParentRoleID
	}

	role.UpdatedAt = time.Now().UTC()

	result, err := m.db.Exec(`
		UPDATE roles SET name = ?, description = ?, can_read = ?, can_write = ?, can_delete = ?, can_admin = ?, parent_role_id = ?, updated_at = ?
		WHERE id = ? AND is_system = 0
	`, role.Name, role.Description,
		role.Permissions.CanRead, role.Permissions.CanWrite, role.Permissions.CanDelete, role.Permissions.CanAdmin,
		parentRoleID, role.UpdatedAt, role.ID)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.ErrSystemRoleModification
	}

	return nil
}

// DeleteRole deletes a custom role (system roles cannot be deleted)
func (m *UserManager) DeleteRole(id string) error {
	// Check if role exists and is not a system role
	existing, err := m.GetRoleByID(id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return errors.ErrSystemRoleDeletion
	}

	result, err := m.db.Exec("DELETE FROM roles WHERE id = ? AND is_system = 0", id)
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.ErrSystemRoleDeletion
	}

	return nil
}
