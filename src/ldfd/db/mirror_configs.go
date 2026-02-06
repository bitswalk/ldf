package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MirrorConfigEntry represents a configured download mirror
type MirrorConfigEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URLPrefix string    `json:"url_prefix"`
	MirrorURL string    `json:"mirror_url"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MirrorConfigRepository handles mirror configuration database operations
type MirrorConfigRepository struct {
	db *Database
}

// NewMirrorConfigRepository creates a new mirror config repository
func NewMirrorConfigRepository(db *Database) *MirrorConfigRepository {
	return &MirrorConfigRepository{db: db}
}

// Create inserts a new mirror configuration
func (r *MirrorConfigRepository) Create(entry *MirrorConfigEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()

	_, err := r.db.DB().Exec(`
		INSERT INTO mirror_configs (id, name, url_prefix, mirror_url, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.Name, entry.URLPrefix, entry.MirrorURL,
		entry.Priority, entry.Enabled, entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create mirror config: %w", err)
	}
	return nil
}

// GetByID retrieves a mirror configuration by ID
func (r *MirrorConfigRepository) GetByID(id string) (*MirrorConfigEntry, error) {
	row := r.db.DB().QueryRow(`
		SELECT id, name, url_prefix, mirror_url, priority, enabled, created_at, updated_at
		FROM mirror_configs WHERE id = ?`, id,
	)
	return r.scanEntry(row)
}

// List returns all mirror configurations ordered by priority
func (r *MirrorConfigRepository) List() ([]MirrorConfigEntry, error) {
	rows, err := r.db.DB().Query(`
		SELECT id, name, url_prefix, mirror_url, priority, enabled, created_at, updated_at
		FROM mirror_configs ORDER BY priority ASC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list mirror configs: %w", err)
	}
	defer rows.Close()

	var entries []MirrorConfigEntry
	for rows.Next() {
		var e MirrorConfigEntry
		if err := rows.Scan(
			&e.ID, &e.Name, &e.URLPrefix, &e.MirrorURL,
			&e.Priority, &e.Enabled, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan mirror config: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListEnabled returns only enabled mirror configurations ordered by priority
func (r *MirrorConfigRepository) ListEnabled() ([]MirrorConfigEntry, error) {
	rows, err := r.db.DB().Query(`
		SELECT id, name, url_prefix, mirror_url, priority, enabled, created_at, updated_at
		FROM mirror_configs WHERE enabled = 1 ORDER BY priority ASC, name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled mirror configs: %w", err)
	}
	defer rows.Close()

	var entries []MirrorConfigEntry
	for rows.Next() {
		var e MirrorConfigEntry
		if err := rows.Scan(
			&e.ID, &e.Name, &e.URLPrefix, &e.MirrorURL,
			&e.Priority, &e.Enabled, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan mirror config: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Update modifies an existing mirror configuration
func (r *MirrorConfigRepository) Update(entry *MirrorConfigEntry) error {
	entry.UpdatedAt = time.Now()

	result, err := r.db.DB().Exec(`
		UPDATE mirror_configs SET name = ?, url_prefix = ?, mirror_url = ?,
			priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		entry.Name, entry.URLPrefix, entry.MirrorURL,
		entry.Priority, entry.Enabled, entry.UpdatedAt, entry.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update mirror config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("mirror config not found: %s", entry.ID)
	}
	return nil
}

// Delete removes a mirror configuration by ID
func (r *MirrorConfigRepository) Delete(id string) error {
	result, err := r.db.DB().Exec(`DELETE FROM mirror_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete mirror config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("mirror config not found: %s", id)
	}
	return nil
}

// scanEntry scans a single row into a MirrorConfigEntry
func (r *MirrorConfigRepository) scanEntry(row *sql.Row) (*MirrorConfigEntry, error) {
	var e MirrorConfigEntry
	err := row.Scan(
		&e.ID, &e.Name, &e.URLPrefix, &e.MirrorURL,
		&e.Priority, &e.Enabled, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan mirror config: %w", err)
	}
	return &e, nil
}
