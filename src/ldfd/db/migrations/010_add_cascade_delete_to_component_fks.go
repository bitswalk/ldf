package migrations

import (
	"database/sql"
	"fmt"
)

func migration010AddCascadeDeleteToComponentFKs() Migration {
	return Migration{
		Version:     10,
		Description: "Add ON DELETE CASCADE to component foreign keys",
		Up:          migration010Up,
	}
}

// migration010Up adds ON DELETE CASCADE to all foreign keys referencing components.
// SQLite doesn't support ALTER CONSTRAINT, so we need to recreate the tables.
func migration010Up(tx *sql.Tx) error {
	// 1. Recreate distribution_source_overrides with CASCADE on component_id
	if _, err := tx.Exec(`
		CREATE TABLE distribution_source_overrides_new (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
			component_id TEXT NOT NULL REFERENCES components(id) ON DELETE CASCADE,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(distribution_id, component_id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create new distribution_source_overrides table: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO distribution_source_overrides_new
		SELECT * FROM distribution_source_overrides
	`); err != nil {
		return fmt.Errorf("failed to copy distribution_source_overrides data: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE distribution_source_overrides`); err != nil {
		return fmt.Errorf("failed to drop old distribution_source_overrides table: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE distribution_source_overrides_new RENAME TO distribution_source_overrides`); err != nil {
		return fmt.Errorf("failed to rename distribution_source_overrides table: %w", err)
	}

	// Recreate indexes
	if _, err := tx.Exec(`CREATE INDEX idx_dist_source_overrides_dist ON distribution_source_overrides(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create distribution index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_dist_source_overrides_component ON distribution_source_overrides(component_id)`); err != nil {
		return fmt.Errorf("failed to create component index: %w", err)
	}

	// 2. Recreate download_jobs with CASCADE on component_id
	if _, err := tx.Exec(`
		CREATE TABLE download_jobs_new (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
			owner_id TEXT NOT NULL,
			component_id TEXT NOT NULL REFERENCES components(id) ON DELETE CASCADE,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			resolved_url TEXT NOT NULL,
			version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			progress_bytes INTEGER DEFAULT 0,
			total_bytes INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			artifact_path TEXT,
			checksum TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3
		)
	`); err != nil {
		return fmt.Errorf("failed to create new download_jobs table: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO download_jobs_new
		SELECT id, distribution_id, owner_id, component_id, source_id, source_type,
		       retrieval_method, resolved_url, version, status, progress_bytes, total_bytes,
		       created_at, started_at, completed_at, artifact_path, checksum, error_message,
		       retry_count, max_retries
		FROM download_jobs
	`); err != nil {
		return fmt.Errorf("failed to copy download_jobs data: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE download_jobs`); err != nil {
		return fmt.Errorf("failed to drop old download_jobs table: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE download_jobs_new RENAME TO download_jobs`); err != nil {
		return fmt.Errorf("failed to rename download_jobs table: %w", err)
	}

	// Recreate indexes
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_distribution ON download_jobs(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create distribution index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_component ON download_jobs(component_id)`); err != nil {
		return fmt.Errorf("failed to create component index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_status ON download_jobs(status)`); err != nil {
		return fmt.Errorf("failed to create status index: %w", err)
	}

	// 3. For source_defaults and user_sources, we need to handle nullable component_id
	// These use SET NULL behavior since sources can exist without a component
	// SQLite doesn't support ON DELETE SET NULL with ALTER, so recreate

	// Recreate source_defaults
	if _, err := tx.Exec(`
		CREATE TABLE source_defaults_new (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			component_id TEXT REFERENCES components(id) ON DELETE SET NULL,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			url_template TEXT
		)
	`); err != nil {
		return fmt.Errorf("failed to create new source_defaults table: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO source_defaults_new
		SELECT id, name, url, priority, enabled, created_at, updated_at,
		       component_id, retrieval_method, url_template
		FROM source_defaults
	`); err != nil {
		return fmt.Errorf("failed to copy source_defaults data: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE source_defaults`); err != nil {
		return fmt.Errorf("failed to drop old source_defaults table: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE source_defaults_new RENAME TO source_defaults`); err != nil {
		return fmt.Errorf("failed to rename source_defaults table: %w", err)
	}

	if _, err := tx.Exec(`CREATE INDEX idx_source_defaults_component ON source_defaults(component_id)`); err != nil {
		return fmt.Errorf("failed to create source_defaults component index: %w", err)
	}

	// Recreate user_sources
	if _, err := tx.Exec(`
		CREATE TABLE user_sources_new (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			component_id TEXT REFERENCES components(id) ON DELETE SET NULL,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			url_template TEXT,
			UNIQUE(owner_id, name)
		)
	`); err != nil {
		return fmt.Errorf("failed to create new user_sources table: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT INTO user_sources_new
		SELECT id, owner_id, name, url, priority, enabled, created_at, updated_at,
		       component_id, retrieval_method, url_template
		FROM user_sources
	`); err != nil {
		return fmt.Errorf("failed to copy user_sources data: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE user_sources`); err != nil {
		return fmt.Errorf("failed to drop old user_sources table: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE user_sources_new RENAME TO user_sources`); err != nil {
		return fmt.Errorf("failed to rename user_sources table: %w", err)
	}

	if _, err := tx.Exec(`CREATE INDEX idx_user_sources_owner ON user_sources(owner_id)`); err != nil {
		return fmt.Errorf("failed to create user_sources owner index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_user_sources_component ON user_sources(component_id)`); err != nil {
		return fmt.Errorf("failed to create user_sources component index: %w", err)
	}

	return nil
}
