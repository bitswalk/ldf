package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when trying to create a user that already exists
	ErrUserExists = errors.New("user already exists")
	// ErrEmailExists is returned when the email is already taken
	ErrEmailExists = errors.New("email already exists")
	// ErrRootExists is returned when trying to create a root user but one already exists
	ErrRootExists = errors.New("root user already exists")
	// ErrTokenRevoked is returned when a token has been revoked
	ErrTokenRevoked = errors.New("token has been revoked")
)

// Repository handles user and token persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository
// Note: The auth tables (users, revoked_tokens) are created by db.Database.initSchema()
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user in the database using a transaction to ensure atomicity
func (r *Repository) CreateUser(user *User) error {
	// Start a transaction to ensure atomic check-and-insert
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be no-op if commit succeeds

	// Check if email already exists
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", user.Email).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if count > 0 {
		return ErrEmailExists
	}

	// Check if username already exists
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE name = ?", user.Name).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if count > 0 {
		return ErrUserExists
	}

	// If role is root, check if a root user already exists
	if user.RoleID == RoleIDRoot {
		err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = ?", RoleIDRoot).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check root user existence: %w", err)
		}
		if count > 0 {
			return ErrRootExists
		}
	}

	// Insert the new user
	_, err = tx.Exec(`
		INSERT INTO users (id, name, email, password_hash, role_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Name, user.Email, user.PasswordHash, user.RoleID, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserByName retrieves a user by username with role information
func (r *Repository) GetUserByName(name string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.name = ?
	`, name).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID with role information
func (r *Repository) GetUserByID(id string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = ?
	`, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email with role information
func (r *Repository) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, r.name, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = ?
	`, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.RoleName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// RevokeToken adds a token to the revoked tokens list
func (r *Repository) RevokeToken(tokenID, userID string, expiresAt time.Time) error {
	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO revoked_tokens (token_id, user_id, revoked_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, tokenID, userID, time.Now().UTC(), expiresAt)

	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// IsTokenRevoked checks if a token has been revoked
func (r *Repository) IsTokenRevoked(tokenID string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM revoked_tokens WHERE token_id = ?", tokenID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return count > 0, nil
}

// CleanupExpiredTokens removes revoked tokens that have passed their expiry time
func (r *Repository) CleanupExpiredTokens() error {
	_, err := r.db.Exec("DELETE FROM revoked_tokens WHERE expires_at < ?", time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}

// CountUsers returns the total number of users
func (r *Repository) CountUsers() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// HasRootUser checks if a root user exists
func (r *Repository) HasRootUser() (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = ?", RoleIDRoot).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check root user: %w", err)
	}
	return count > 0, nil
}

// GetUserWithRole retrieves a user with full role information
func (r *Repository) GetUserWithRole(userID string) (*User, *RoleRecord, error) {
	user, err := r.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}

	role, err := r.GetRoleByID(user.RoleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user role: %w", err)
	}

	return user, role, nil
}
