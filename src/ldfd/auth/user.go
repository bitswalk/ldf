package auth

import "database/sql"

// UserManager handles user, role, and token persistence
type UserManager struct {
	db *sql.DB
}

// NewUserManager creates a new user manager
func NewUserManager(db *sql.DB) *UserManager {
	return &UserManager{db: db}
}
