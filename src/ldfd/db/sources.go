package db

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// SourceRepository handles source database operations
type SourceRepository struct {
	db *Database
}

// NewSourceRepository creates a new source repository
func NewSourceRepository(db *Database) *SourceRepository {
	return &SourceRepository{db: db}
}

// ListDefaults retrieves all default sources ordered by priority
func (r *SourceRepository) ListDefaults() ([]SourceDefault, error) {
	query := `
		SELECT id, name, url, priority, enabled, created_at, updated_at
		FROM source_defaults
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list default sources: %w", err)
	}
	defer rows.Close()

	var sources []SourceDefault
	for rows.Next() {
		var s SourceDefault
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan default source: %w", err)
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating default sources: %w", err)
	}

	return sources, nil
}

// GetDefaultByID retrieves a default source by ID
func (r *SourceRepository) GetDefaultByID(id string) (*SourceDefault, error) {
	query := `
		SELECT id, name, url, priority, enabled, created_at, updated_at
		FROM source_defaults
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	var s SourceDefault
	err := row.Scan(&s.ID, &s.Name, &s.URL, &s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default source: %w", err)
	}

	return &s, nil
}

// CreateDefault inserts a new default source
func (r *SourceRepository) CreateDefault(s *SourceDefault) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	query := `
		INSERT INTO source_defaults (id, name, url, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, s.ID, s.Name, s.URL, s.Priority, s.Enabled, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create default source: %w", err)
	}

	return nil
}

// UpdateDefault updates an existing default source
func (r *SourceRepository) UpdateDefault(s *SourceDefault) error {
	s.UpdatedAt = time.Now()

	query := `
		UPDATE source_defaults
		SET name = ?, url = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, s.Name, s.URL, s.Priority, s.Enabled, s.UpdatedAt, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update default source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("default source not found: %s", s.ID)
	}

	return nil
}

// DeleteDefault removes a default source by ID
func (r *SourceRepository) DeleteDefault(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM source_defaults WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete default source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("default source not found: %s", id)
	}

	return nil
}

// ListUserSources retrieves all sources for a specific user ordered by priority
func (r *SourceRepository) ListUserSources(ownerID string) ([]UserSource, error) {
	query := `
		SELECT id, owner_id, name, url, priority, enabled, created_at, updated_at
		FROM user_sources
		WHERE owner_id = ?
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sources: %w", err)
	}
	defer rows.Close()

	var sources []UserSource
	for rows.Next() {
		var s UserSource
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.Name, &s.URL, &s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user source: %w", err)
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user sources: %w", err)
	}

	return sources, nil
}

// GetUserSourceByID retrieves a user source by ID
func (r *SourceRepository) GetUserSourceByID(id string) (*UserSource, error) {
	query := `
		SELECT id, owner_id, name, url, priority, enabled, created_at, updated_at
		FROM user_sources
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	var s UserSource
	err := row.Scan(&s.ID, &s.OwnerID, &s.Name, &s.URL, &s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user source: %w", err)
	}

	return &s, nil
}

// CreateUserSource inserts a new user source
func (r *SourceRepository) CreateUserSource(s *UserSource) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	query := `
		INSERT INTO user_sources (id, owner_id, name, url, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, s.ID, s.OwnerID, s.Name, s.URL, s.Priority, s.Enabled, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user source: %w", err)
	}

	return nil
}

// UpdateUserSource updates an existing user source
func (r *SourceRepository) UpdateUserSource(s *UserSource) error {
	s.UpdatedAt = time.Now()

	query := `
		UPDATE user_sources
		SET name = ?, url = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, s.Name, s.URL, s.Priority, s.Enabled, s.UpdatedAt, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update user source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("user source not found: %s", s.ID)
	}

	return nil
}

// DeleteUserSource removes a user source by ID
func (r *SourceRepository) DeleteUserSource(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM user_sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete user source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("user source not found: %s", id)
	}

	return nil
}

// GetMergedSources returns all sources (defaults + user-specific) merged and sorted by priority
func (r *SourceRepository) GetMergedSources(userID string) ([]Source, error) {
	var sources []Source

	// Get default sources
	defaults, err := r.ListDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to get default sources: %w", err)
	}

	for _, d := range defaults {
		sources = append(sources, Source{
			ID:        d.ID,
			Name:      d.Name,
			URL:       d.URL,
			Priority:  d.Priority,
			Enabled:   d.Enabled,
			IsSystem:  true,
			OwnerID:   "",
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}

	// Get user sources if userID is provided
	if userID != "" {
		userSources, err := r.ListUserSources(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user sources: %w", err)
		}

		for _, u := range userSources {
			sources = append(sources, Source{
				ID:        u.ID,
				Name:      u.Name,
				URL:       u.URL,
				Priority:  u.Priority,
				Enabled:   u.Enabled,
				IsSystem:  false,
				OwnerID:   u.OwnerID,
				CreatedAt: u.CreatedAt,
				UpdatedAt: u.UpdatedAt,
			})
		}
	}

	// Sort by priority (ascending), then by name
	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Priority != sources[j].Priority {
			return sources[i].Priority < sources[j].Priority
		}
		return sources[i].Name < sources[j].Name
	})

	return sources, nil
}
