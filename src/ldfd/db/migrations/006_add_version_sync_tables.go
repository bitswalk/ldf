package migrations

import (
	"database/sql"
	"fmt"
)

// migration006AddVersionSyncTables creates the source_versions and version_sync_jobs tables
// for tracking discovered versions from upstream sources
func migration006AddVersionSyncTables() Migration {
	return Migration{
		Version:     6,
		Description: "Add version sync tables (source_versions, version_sync_jobs) for upstream version discovery",
		Up:          migration006Up,
	}
}

func migration006Up(tx *sql.Tx) error {
	// Create source_versions table - cached discovered versions from upstream
	if _, err := tx.Exec(`
		CREATE TABLE source_versions (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			version TEXT NOT NULL,
			release_date DATETIME,
			download_url TEXT,
			checksum TEXT,
			checksum_type TEXT,
			file_size INTEGER,
			is_stable BOOLEAN DEFAULT 1,
			discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_id, source_type, version)
		)
	`); err != nil {
		return fmt.Errorf("failed to create source_versions table: %w", err)
	}

	// Create indexes for source_versions
	if _, err := tx.Exec(`CREATE INDEX idx_source_versions_source ON source_versions(source_id, source_type)`); err != nil {
		return fmt.Errorf("failed to create source_versions source index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_source_versions_version ON source_versions(version)`); err != nil {
		return fmt.Errorf("failed to create source_versions version index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_source_versions_stable ON source_versions(source_id, source_type, is_stable)`); err != nil {
		return fmt.Errorf("failed to create source_versions stable index: %w", err)
	}

	// Create version_sync_jobs table - tracks sync job status
	if _, err := tx.Exec(`
		CREATE TABLE version_sync_jobs (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			versions_found INTEGER DEFAULT 0,
			versions_new INTEGER DEFAULT 0,
			started_at DATETIME,
			completed_at DATETIME,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create version_sync_jobs table: %w", err)
	}

	// Create indexes for version_sync_jobs
	if _, err := tx.Exec(`CREATE INDEX idx_version_sync_jobs_source ON version_sync_jobs(source_id, source_type)`); err != nil {
		return fmt.Errorf("failed to create version_sync_jobs source index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_version_sync_jobs_status ON version_sync_jobs(status)`); err != nil {
		return fmt.Errorf("failed to create version_sync_jobs status index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_version_sync_jobs_created ON version_sync_jobs(created_at DESC)`); err != nil {
		return fmt.Errorf("failed to create version_sync_jobs created index: %w", err)
	}

	return nil
}
