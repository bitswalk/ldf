package migrations

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// migration003ConvertDistributionIDToUUID converts the distributions.id column from INTEGER to TEXT UUID
func migration003ConvertDistributionIDToUUID() Migration {
	return Migration{
		Version:     3,
		Description: "Convert distributions.id from INTEGER to TEXT UUID for multi-tenancy support",
		Up:          migration003Up,
	}
}

func migration003Up(tx *sql.Tx) error {
	// SQLite doesn't support ALTER COLUMN, so we need to recreate the table
	// Step 1: Get existing distributions with their integer IDs
	rows, err := tx.Query(`
		SELECT id, name, version, status, visibility, config, source_url, checksum,
		       size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message
		FROM distributions
	`)
	if err != nil {
		return fmt.Errorf("failed to query existing distributions: %w", err)
	}

	type distRecord struct {
		OldID        int64
		NewID        string
		Name         string
		Version      string
		Status       string
		Visibility   string
		Config       sql.NullString
		SourceURL    sql.NullString
		Checksum     sql.NullString
		SizeBytes    int64
		OwnerID      sql.NullString
		CreatedAt    string
		UpdatedAt    string
		StartedAt    sql.NullString
		CompletedAt  sql.NullString
		ErrorMessage sql.NullString
	}

	var distributions []distRecord
	for rows.Next() {
		var d distRecord
		if err := rows.Scan(
			&d.OldID, &d.Name, &d.Version, &d.Status, &d.Visibility, &d.Config,
			&d.SourceURL, &d.Checksum, &d.SizeBytes, &d.OwnerID,
			&d.CreatedAt, &d.UpdatedAt, &d.StartedAt, &d.CompletedAt, &d.ErrorMessage,
		); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan distribution: %w", err)
		}
		d.NewID = uuid.New().String()
		distributions = append(distributions, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating distributions: %w", err)
	}

	// Build old ID to new UUID mapping for updating foreign keys
	idMapping := make(map[int64]string)
	for _, d := range distributions {
		idMapping[d.OldID] = d.NewID
	}

	// Step 2: Get existing distribution logs
	logRows, err := tx.Query(`SELECT id, distribution_id, level, message, created_at FROM distribution_logs`)
	if err != nil {
		return fmt.Errorf("failed to query distribution logs: %w", err)
	}

	type logRecord struct {
		ID             int64
		DistributionID int64
		Level          string
		Message        string
		CreatedAt      string
	}

	var logs []logRecord
	for logRows.Next() {
		var l logRecord
		if err := logRows.Scan(&l.ID, &l.DistributionID, &l.Level, &l.Message, &l.CreatedAt); err != nil {
			logRows.Close()
			return fmt.Errorf("failed to scan log: %w", err)
		}
		logs = append(logs, l)
	}
	logRows.Close()

	// Step 3: Drop the old tables (logs first due to FK constraint)
	if _, err := tx.Exec(`DROP TABLE IF EXISTS distribution_logs`); err != nil {
		return fmt.Errorf("failed to drop distribution_logs table: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE IF EXISTS distributions`); err != nil {
		return fmt.Errorf("failed to drop distributions table: %w", err)
	}

	// Step 4: Recreate distributions table with TEXT id
	if _, err := tx.Exec(`
		CREATE TABLE distributions (
			id TEXT PRIMARY KEY,
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
		)
	`); err != nil {
		return fmt.Errorf("failed to create new distributions table: %w", err)
	}

	// Step 5: Recreate indexes
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_distributions_status ON distributions(status)`); err != nil {
		return fmt.Errorf("failed to create status index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_distributions_name ON distributions(name)`); err != nil {
		return fmt.Errorf("failed to create name index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_distributions_owner ON distributions(owner_id)`); err != nil {
		return fmt.Errorf("failed to create owner index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_distributions_visibility ON distributions(visibility)`); err != nil {
		return fmt.Errorf("failed to create visibility index: %w", err)
	}

	// Step 6: Recreate distribution_logs table with TEXT distribution_id
	if _, err := tx.Exec(`
		CREATE TABLE distribution_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			distribution_id TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create new distribution_logs table: %w", err)
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_distribution_logs_dist_id ON distribution_logs(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create distribution_logs index: %w", err)
	}

	// Step 7: Re-insert distributions with new UUIDs
	for _, d := range distributions {
		if _, err := tx.Exec(`
			INSERT INTO distributions (id, name, version, status, visibility, config, source_url, checksum,
			                          size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, d.NewID, d.Name, d.Version, d.Status, d.Visibility, d.Config, d.SourceURL, d.Checksum,
			d.SizeBytes, d.OwnerID, d.CreatedAt, d.UpdatedAt, d.StartedAt, d.CompletedAt, d.ErrorMessage,
		); err != nil {
			return fmt.Errorf("failed to insert distribution %s: %w", d.Name, err)
		}
	}

	// Step 8: Re-insert logs with updated distribution IDs
	for _, l := range logs {
		newDistID, ok := idMapping[l.DistributionID]
		if !ok {
			continue // Skip orphaned logs
		}
		if _, err := tx.Exec(`
			INSERT INTO distribution_logs (distribution_id, level, message, created_at)
			VALUES (?, ?, ?, ?)
		`, newDistID, l.Level, l.Message, l.CreatedAt); err != nil {
			return fmt.Errorf("failed to insert log: %w", err)
		}
	}

	return nil
}
