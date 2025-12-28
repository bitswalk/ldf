package migrations

import (
	"database/sql"
	"fmt"
)

// migration008RemoveComponentNameFromDownloadJobs removes the redundant component_name
// column from download_jobs table. The component name is now retrieved via JOIN
// with the components table using component_id.
func migration008RemoveComponentNameFromDownloadJobs() Migration {
	return Migration{
		Version:     8,
		Description: "Remove redundant component_name column from download_jobs (use JOIN instead)",
		Up:          migration008Up,
	}
}

func migration008Up(tx *sql.Tx) error {
	// SQLite doesn't support DROP COLUMN directly, so we need to:
	// 1. Create a new table without the column
	// 2. Copy data from the old table
	// 3. Drop the old table
	// 4. Rename the new table

	// Create new table without component_name
	if _, err := tx.Exec(`
		CREATE TABLE download_jobs_new (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
			owner_id TEXT NOT NULL,
			component_id TEXT NOT NULL REFERENCES components(id),
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

	// Copy data from old table (excluding component_name)
	if _, err := tx.Exec(`
		INSERT INTO download_jobs_new (
			id, distribution_id, owner_id, component_id, source_id, source_type,
			retrieval_method, resolved_url, version, status, progress_bytes, total_bytes,
			created_at, started_at, completed_at, artifact_path, checksum, error_message,
			retry_count, max_retries
		)
		SELECT
			id, distribution_id, owner_id, component_id, source_id, source_type,
			retrieval_method, resolved_url, version, status, progress_bytes, total_bytes,
			created_at, started_at, completed_at, artifact_path, checksum, error_message,
			retry_count, max_retries
		FROM download_jobs
	`); err != nil {
		return fmt.Errorf("failed to copy data to new download_jobs table: %w", err)
	}

	// Drop old table
	if _, err := tx.Exec(`DROP TABLE download_jobs`); err != nil {
		return fmt.Errorf("failed to drop old download_jobs table: %w", err)
	}

	// Rename new table
	if _, err := tx.Exec(`ALTER TABLE download_jobs_new RENAME TO download_jobs`); err != nil {
		return fmt.Errorf("failed to rename download_jobs table: %w", err)
	}

	// Recreate indexes
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_distribution ON download_jobs(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create download_jobs distribution index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_status ON download_jobs(status)`); err != nil {
		return fmt.Errorf("failed to create download_jobs status index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_component ON download_jobs(component_id)`); err != nil {
		return fmt.Errorf("failed to create download_jobs component index: %w", err)
	}

	return nil
}
