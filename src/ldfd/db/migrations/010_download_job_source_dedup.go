package migrations

import (
	"database/sql"
)

// migration010DownloadJobSourceDedup adds source_name and component_ids columns
// to download_jobs table for artifact deduplication across components sharing the same source
func migration010DownloadJobSourceDedup() Migration {
	return Migration{
		Version:     10,
		Description: "Add source_name and component_ids columns to download_jobs for deduplication",
		Up: func(tx *sql.Tx) error {
			// Add source_name column for artifact path construction
			_, err := tx.Exec(`ALTER TABLE download_jobs ADD COLUMN source_name TEXT`)
			if err != nil {
				return err
			}

			// Add component_ids column (JSON array) to track all components sharing this artifact
			_, err = tx.Exec(`ALTER TABLE download_jobs ADD COLUMN component_ids TEXT`)
			if err != nil {
				return err
			}

			// Add index for efficient lookup by source_id + version (for deduplication checks)
			_, err = tx.Exec(`CREATE INDEX idx_download_jobs_source_version ON download_jobs(distribution_id, source_id, version)`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
