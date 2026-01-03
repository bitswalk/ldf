package migrations

import (
	"database/sql"
	"fmt"
)

// migration004MergeSources merges source_defaults and user_sources into a single upstream_sources table
func migration004MergeSources() Migration {
	return Migration{
		Version:     4,
		Description: "Merge source_defaults and user_sources into upstream_sources table",
		Up:          migration004Up,
	}
}

func migration004Up(tx *sql.Tx) error {
	// Step 1: Create the new unified upstream_sources table
	_, err := tx.Exec(`
		CREATE TABLE upstream_sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			component_ids TEXT DEFAULT '[]',
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			url_template TEXT,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			is_system INTEGER NOT NULL DEFAULT 0,
			owner_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create upstream_sources table: %w", err)
	}

	// Step 2: Migrate data from source_defaults (system sources)
	_, err = tx.Exec(`
		INSERT INTO upstream_sources (id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
		SELECT id, name, url, COALESCE(component_ids, '[]'), retrieval_method, url_template, priority, enabled, 1, NULL, created_at, updated_at
		FROM source_defaults
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate source_defaults data: %w", err)
	}

	// Step 3: Migrate data from user_sources (user sources)
	_, err = tx.Exec(`
		INSERT INTO upstream_sources (id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
		SELECT id, name, url, COALESCE(component_ids, '[]'), retrieval_method, url_template, priority, enabled, 0, owner_id, created_at, updated_at
		FROM user_sources
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate user_sources data: %w", err)
	}

	// Step 4: Create indexes for efficient queries
	indexes := []string{
		`CREATE INDEX idx_upstream_sources_is_system ON upstream_sources(is_system)`,
		`CREATE INDEX idx_upstream_sources_owner ON upstream_sources(owner_id)`,
		`CREATE INDEX idx_upstream_sources_priority ON upstream_sources(priority)`,
		`CREATE UNIQUE INDEX idx_upstream_sources_owner_name ON upstream_sources(owner_id, name) WHERE owner_id IS NOT NULL`,
		`CREATE UNIQUE INDEX idx_upstream_sources_system_name ON upstream_sources(name) WHERE is_system = 1`,
	}

	for _, sql := range indexes {
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Step 5: Update distribution_source_overrides to remove source_type
	// We'll keep the source_type column but it will be deprecated
	// The source_id now uniquely identifies the source in upstream_sources

	// Step 6: Drop the old tables
	_, err = tx.Exec(`DROP TABLE source_defaults`)
	if err != nil {
		return fmt.Errorf("failed to drop source_defaults table: %w", err)
	}

	_, err = tx.Exec(`DROP TABLE user_sources`)
	if err != nil {
		return fmt.Errorf("failed to drop user_sources table: %w", err)
	}

	return nil
}
