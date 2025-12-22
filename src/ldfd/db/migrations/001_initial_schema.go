package migrations

import "database/sql"

// migration001InitialSchema creates the initial database schema
func migration001InitialSchema() Migration {
	return Migration{
		Version:     1,
		Description: "Initial schema with distributions, users, roles, and settings",
		Up:          migration001Up,
	}
}

func migration001Up(tx *sql.Tx) error {
	// Create roles table first (referenced by users)
	if _, err := tx.Exec(rolesTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(rolesIndexesSQL); err != nil {
		return err
	}

	// Create users table (referenced by distributions and revoked_tokens)
	if _, err := tx.Exec(usersTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(usersIndexesSQL); err != nil {
		return err
	}

	// Create distributions table
	if _, err := tx.Exec(distributionsTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(distributionsIndexesSQL); err != nil {
		return err
	}

	// Create distribution_logs table
	if _, err := tx.Exec(distributionLogsTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(distributionLogsIndexesSQL); err != nil {
		return err
	}

	// Create revoked_tokens table
	if _, err := tx.Exec(revokedTokensTableSQL); err != nil {
		return err
	}
	if _, err := tx.Exec(revokedTokensIndexesSQL); err != nil {
		return err
	}

	// Create settings table
	if _, err := tx.Exec(settingsTableSQL); err != nil {
		return err
	}

	return nil
}

// Table creation SQL statements

const rolesTableSQL = `
CREATE TABLE IF NOT EXISTS roles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT,
	can_read BOOLEAN NOT NULL DEFAULT 1,
	can_write BOOLEAN NOT NULL DEFAULT 0,
	can_delete BOOLEAN NOT NULL DEFAULT 0,
	can_admin BOOLEAN NOT NULL DEFAULT 0,
	is_system BOOLEAN NOT NULL DEFAULT 0,
	parent_role_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (parent_role_id) REFERENCES roles(id) ON DELETE SET NULL
)`

const rolesIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_roles_parent ON roles(parent_role_id)`

const usersTableSQL = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
)`

const usersIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id)`

const distributionsTableSQL = `
CREATE TABLE IF NOT EXISTS distributions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	version TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	visibility TEXT NOT NULL DEFAULT 'private',
	config TEXT,
	source_url TEXT,
	checksum TEXT,
	size_bytes INTEGER DEFAULT 0,
	owner_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	started_at DATETIME,
	completed_at DATETIME,
	error_message TEXT,
	FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL
)`

const distributionsIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_distributions_status ON distributions(status);
CREATE INDEX IF NOT EXISTS idx_distributions_name ON distributions(name);
CREATE INDEX IF NOT EXISTS idx_distributions_owner ON distributions(owner_id);
CREATE INDEX IF NOT EXISTS idx_distributions_visibility ON distributions(visibility)`

const distributionLogsTableSQL = `
CREATE TABLE IF NOT EXISTS distribution_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	distribution_id INTEGER NOT NULL,
	level TEXT NOT NULL,
	message TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE
)`

const distributionLogsIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_distribution_logs_dist_id ON distribution_logs(distribution_id)`

const revokedTokensTableSQL = `
CREATE TABLE IF NOT EXISTS revoked_tokens (
	token_id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	expires_at DATETIME NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)`

const revokedTokensIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id ON revoked_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at)`

const settingsTableSQL = `
CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`
