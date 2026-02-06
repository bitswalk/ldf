package migrations

import "database/sql"

func migration016DownloadCache() Migration {
	return Migration{
		Version:     16,
		Description: "Add artifact cache, mirror configs, and download job enhancements",
		Up: func(tx *sql.Tx) error {
			// Shared artifact cache keyed by source+version
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS artifact_cache (
					id TEXT PRIMARY KEY,
					source_id TEXT NOT NULL,
					version TEXT NOT NULL,
					checksum TEXT NOT NULL DEFAULT '',
					cache_path TEXT NOT NULL,
					size_bytes INTEGER DEFAULT 0,
					content_type TEXT DEFAULT '',
					resolved_url TEXT DEFAULT '',
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					use_count INTEGER DEFAULT 1,
					UNIQUE(source_id, version)
				)
			`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_artifact_cache_source_version ON artifact_cache(source_id, version)`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_artifact_cache_last_used ON artifact_cache(last_used_at)`)
			if err != nil {
				return err
			}

			// Mirror configuration table
			_, err = tx.Exec(`
				CREATE TABLE IF NOT EXISTS mirror_configs (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					url_prefix TEXT NOT NULL,
					mirror_url TEXT NOT NULL,
					priority INTEGER DEFAULT 0,
					enabled INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				return err
			}

			// Download job enhancements
			_, err = tx.Exec(`ALTER TABLE download_jobs ADD COLUMN priority INTEGER DEFAULT 0`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE download_jobs ADD COLUMN cache_hit INTEGER DEFAULT 0`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
