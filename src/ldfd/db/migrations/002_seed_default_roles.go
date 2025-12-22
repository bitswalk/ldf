package migrations

import "database/sql"

// System role IDs - must match constants in auth/role.go
const (
	RoleIDRoot      = "908b291e-61fb-4d95-98db-0b76c0afd6b4"
	RoleIDDeveloper = "91db9f27-b8a2-4452-9b80-5f6ab1096da8"
	RoleIDAnonymous = "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0"
)

// DefaultRole represents a system role to be seeded
type DefaultRole struct {
	ID          string
	Name        string
	Description string
	CanRead     bool
	CanWrite    bool
	CanDelete   bool
	CanAdmin    bool
}

// DefaultRoles returns the list of default system roles
func DefaultRoles() []DefaultRole {
	return []DefaultRole{
		{
			ID:          RoleIDRoot,
			Name:        "root",
			Description: "Administrator role with full system access",
			CanRead:     true,
			CanWrite:    true,
			CanDelete:   true,
			CanAdmin:    true,
		},
		{
			ID:          RoleIDDeveloper,
			Name:        "developer",
			Description: "Standard user with read/write access to owned resources",
			CanRead:     true,
			CanWrite:    true,
			CanDelete:   true,
			CanAdmin:    false,
		},
		{
			ID:          RoleIDAnonymous,
			Name:        "anonymous",
			Description: "Read-only access to public resources",
			CanRead:     true,
			CanWrite:    false,
			CanDelete:   false,
			CanAdmin:    false,
		},
	}
}

// migration002SeedDefaultRoles seeds the default system roles
func migration002SeedDefaultRoles() Migration {
	return Migration{
		Version:     2,
		Description: "Seed default system roles (root, developer, anonymous)",
		Up:          migration002Up,
	}
}

func migration002Up(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, role := range DefaultRoles() {
		if _, err := stmt.Exec(
			role.ID,
			role.Name,
			role.Description,
			role.CanRead,
			role.CanWrite,
			role.CanDelete,
			role.CanAdmin,
		); err != nil {
			return err
		}
	}

	return nil
}
