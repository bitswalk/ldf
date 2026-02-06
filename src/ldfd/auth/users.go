package auth

import (
	"database/sql"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/google/uuid"
)

// NewUserManager creates a new user manager
func NewUserManager(db *sql.DB) *UserManager {
	return &UserManager{db: db}
}

// NewUser creates a new user with a generated UUID
func NewUser(name, email, passwordHash, roleID string) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		RoleID:       roleID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateUser creates a new user in the database using a transaction to ensure atomicity
func (m *UserManager) CreateUser(user *User) error {
	tx, err := m.db.Begin()
	if err != nil {
		return errors.ErrDatabaseTransaction.WithCause(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check if email already exists
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", user.Email).Scan(&count); err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}
	if count > 0 {
		return errors.ErrEmailAlreadyExists
	}

	// Check if username already exists
	if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", user.Name).Scan(&count); err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}
	if count > 0 {
		return errors.ErrUserAlreadyExists
	}

	// If role is root, check if a root user already exists
	if user.RoleID == RoleIDRoot {
		if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = ?", RoleIDRoot).Scan(&count); err != nil {
			return errors.ErrDatabaseQuery.WithCause(err)
		}
		if count > 0 {
			return errors.ErrRootUserExists
		}
	}

	// Insert the new user
	_, err = tx.Exec(`
		INSERT INTO users (id, name, email, password_hash, role_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Name, user.Email, user.PasswordHash, user.RoleID, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	if err := tx.Commit(); err != nil {
		return errors.ErrDatabaseTransaction.WithCause(err)
	}

	return nil
}

// GetUserByName retrieves a user by username with role information
func (m *UserManager) GetUserByName(name string) (*User, error) {
	user := &User{}
	err := m.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.name = ?
	`, name).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID with role information
func (m *UserManager) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := m.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = ?
	`, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email with role information
func (m *UserManager) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := m.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = ?
	`, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.ErrUserNotFound
	}
	if err != nil {
		return nil, errors.ErrDatabaseQuery.WithCause(err)
	}

	return user, nil
}

// CountUsers returns the total number of users
func (m *UserManager) CountUsers() (int, error) {
	var count int
	if err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return 0, errors.ErrDatabaseQuery.WithCause(err)
	}
	return count, nil
}

// HasRootUser checks if a root user exists
func (m *UserManager) HasRootUser() (bool, error) {
	var count int
	if err := m.db.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = ?", RoleIDRoot).Scan(&count); err != nil {
		return false, errors.ErrDatabaseQuery.WithCause(err)
	}
	return count > 0, nil
}

// GetUserWithRole retrieves a user with full role information
func (m *UserManager) GetUserWithRole(userID string) (*User, *RoleRecord, error) {
	user, err := m.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}

	role, err := m.GetRoleByID(user.RoleID)
	if err != nil {
		return nil, nil, err
	}

	return user, role, nil
}
