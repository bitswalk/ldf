package migrations

import (
	"database/sql"
)

func migration012BuildJobs() Migration {
	return Migration{
		Version:     12,
		Description: "Add build_jobs, build_stages, and build_logs tables",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE build_jobs (
					id TEXT PRIMARY KEY,
					distribution_id TEXT NOT NULL,
					owner_id TEXT NOT NULL,
					status TEXT NOT NULL DEFAULT 'pending',
					current_stage TEXT DEFAULT '',
					target_arch TEXT NOT NULL DEFAULT 'x86_64',
					image_format TEXT NOT NULL DEFAULT 'raw',
					progress_percent INTEGER DEFAULT 0,
					workspace_path TEXT DEFAULT '',
					artifact_path TEXT DEFAULT '',
					artifact_checksum TEXT DEFAULT '',
					artifact_size INTEGER DEFAULT 0,
					error_message TEXT DEFAULT '',
					error_stage TEXT DEFAULT '',
					retry_count INTEGER DEFAULT 0,
					max_retries INTEGER DEFAULT 1,
					config_snapshot TEXT DEFAULT '',
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					started_at DATETIME,
					completed_at DATETIME,
					FOREIGN KEY (distribution_id) REFERENCES distributions(id)
				)
			`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
				CREATE TABLE build_stages (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					build_id TEXT NOT NULL,
					name TEXT NOT NULL,
					status TEXT NOT NULL DEFAULT 'pending',
					progress_percent INTEGER DEFAULT 0,
					started_at DATETIME,
					completed_at DATETIME,
					duration_ms INTEGER DEFAULT 0,
					error_message TEXT DEFAULT '',
					log_path TEXT DEFAULT '',
					FOREIGN KEY (build_id) REFERENCES build_jobs(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
				CREATE TABLE build_logs (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					build_id TEXT NOT NULL,
					stage TEXT NOT NULL DEFAULT '',
					level TEXT NOT NULL DEFAULT 'info',
					message TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (build_id) REFERENCES build_jobs(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return err
			}

			// Indexes
			_, err = tx.Exec(`CREATE INDEX idx_build_jobs_distribution ON build_jobs(distribution_id)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_build_jobs_status ON build_jobs(status)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_build_stages_build ON build_stages(build_id)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_build_logs_build ON build_logs(build_id)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_build_logs_build_stage ON build_logs(build_id, stage)`)
			return err
		},
	}
}
