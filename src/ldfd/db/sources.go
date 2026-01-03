package db

import (
	"database/sql"
	"encoding/json"
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
		SELECT id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM source_defaults
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list default sources: %w", err)
	}
	defer rows.Close()

	return r.scanDefaultSources(rows)
}

// ListDefaultsByComponent retrieves all default sources for a specific component
func (r *SourceRepository) ListDefaultsByComponent(componentID string) ([]SourceDefault, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM source_defaults
		WHERE EXISTS (SELECT 1 FROM json_each(component_ids) WHERE value = ?)
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list default sources by component: %w", err)
	}
	defer rows.Close()

	return r.scanDefaultSources(rows)
}

// GetDefaultByID retrieves a default source by ID
func (r *SourceRepository) GetDefaultByID(id string) (*SourceDefault, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM source_defaults
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	var s SourceDefault
	var componentIDsJSON, urlTemplate sql.NullString
	err := row.Scan(&s.ID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
		&s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default source: %w", err)
	}

	s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
	s.URLTemplate = urlTemplate.String

	return &s, nil
}

// CreateDefault inserts a new default source
func (r *SourceRepository) CreateDefault(s *SourceDefault) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.RetrievalMethod == "" {
		s.RetrievalMethod = "release"
	}
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		INSERT INTO source_defaults (id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, s.ID, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.Priority, s.Enabled, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create default source: %w", err)
	}

	return nil
}

// UpdateDefault updates an existing default source
func (r *SourceRepository) UpdateDefault(s *SourceDefault) error {
	s.UpdatedAt = time.Now()
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		UPDATE source_defaults
		SET name = ?, url = ?, component_ids = ?, retrieval_method = ?, url_template = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.Priority, s.Enabled, s.UpdatedAt, s.ID)
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
		SELECT id, owner_id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM user_sources
		WHERE owner_id = ?
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sources: %w", err)
	}
	defer rows.Close()

	return r.scanUserSources(rows)
}

// ListUserSourcesByComponent retrieves all user sources for a specific component
func (r *SourceRepository) ListUserSourcesByComponent(ownerID, componentID string) ([]UserSource, error) {
	query := `
		SELECT id, owner_id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM user_sources
		WHERE owner_id = ? AND EXISTS (SELECT 1 FROM json_each(component_ids) WHERE value = ?)
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sources by component: %w", err)
	}
	defer rows.Close()

	return r.scanUserSources(rows)
}

// GetUserSourceByID retrieves a user source by ID
func (r *SourceRepository) GetUserSourceByID(id string) (*UserSource, error) {
	query := `
		SELECT id, owner_id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at
		FROM user_sources
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	var s UserSource
	var componentIDsJSON, urlTemplate sql.NullString
	err := row.Scan(&s.ID, &s.OwnerID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
		&s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user source: %w", err)
	}

	s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
	s.URLTemplate = urlTemplate.String

	return &s, nil
}

// CreateUserSource inserts a new user source
func (r *SourceRepository) CreateUserSource(s *UserSource) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.RetrievalMethod == "" {
		s.RetrievalMethod = "release"
	}
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		INSERT INTO user_sources (id, owner_id, name, url, component_ids, retrieval_method, url_template, priority, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, s.ID, s.OwnerID, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.Priority, s.Enabled, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user source: %w", err)
	}

	return nil
}

// UpdateUserSource updates an existing user source
func (r *SourceRepository) UpdateUserSource(s *UserSource) error {
	s.UpdatedAt = time.Now()
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		UPDATE user_sources
		SET name = ?, url = ?, component_ids = ?, retrieval_method = ?, url_template = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.Priority, s.Enabled, s.UpdatedAt, s.ID)
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
			ID:              d.ID,
			Name:            d.Name,
			URL:             d.URL,
			ComponentIDs:    d.ComponentIDs,
			RetrievalMethod: d.RetrievalMethod,
			URLTemplate:     d.URLTemplate,
			Priority:        d.Priority,
			Enabled:         d.Enabled,
			IsSystem:        true,
			OwnerID:         "",
			CreatedAt:       d.CreatedAt,
			UpdatedAt:       d.UpdatedAt,
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
				ID:              u.ID,
				Name:            u.Name,
				URL:             u.URL,
				ComponentIDs:    u.ComponentIDs,
				RetrievalMethod: u.RetrievalMethod,
				URLTemplate:     u.URLTemplate,
				Priority:        u.Priority,
				Enabled:         u.Enabled,
				IsSystem:        false,
				OwnerID:         u.OwnerID,
				CreatedAt:       u.CreatedAt,
				UpdatedAt:       u.UpdatedAt,
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

// GetMergedSourcesByComponent returns sources for a specific component merged and sorted by priority
func (r *SourceRepository) GetMergedSourcesByComponent(userID, componentID string) ([]Source, error) {
	var sources []Source

	// Get default sources for this component
	defaults, err := r.ListDefaultsByComponent(componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default sources: %w", err)
	}

	for _, d := range defaults {
		sources = append(sources, Source{
			ID:              d.ID,
			Name:            d.Name,
			URL:             d.URL,
			ComponentIDs:    d.ComponentIDs,
			RetrievalMethod: d.RetrievalMethod,
			URLTemplate:     d.URLTemplate,
			Priority:        d.Priority,
			Enabled:         d.Enabled,
			IsSystem:        true,
			OwnerID:         "",
			CreatedAt:       d.CreatedAt,
			UpdatedAt:       d.UpdatedAt,
		})
	}

	// Get user sources for this component if userID is provided
	if userID != "" {
		userSources, err := r.ListUserSourcesByComponent(userID, componentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user sources: %w", err)
		}

		for _, u := range userSources {
			sources = append(sources, Source{
				ID:              u.ID,
				Name:            u.Name,
				URL:             u.URL,
				ComponentIDs:    u.ComponentIDs,
				RetrievalMethod: u.RetrievalMethod,
				URLTemplate:     u.URLTemplate,
				Priority:        u.Priority,
				Enabled:         u.Enabled,
				IsSystem:        false,
				OwnerID:         u.OwnerID,
				CreatedAt:       u.CreatedAt,
				UpdatedAt:       u.UpdatedAt,
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

// GetEffectiveSource returns the effective source for a component, considering distribution overrides
func (r *SourceRepository) GetEffectiveSource(distributionID, componentID, userID string) (*Source, error) {
	// First, check for distribution-specific override
	override, err := r.GetDistributionOverride(distributionID, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution override: %w", err)
	}

	if override != nil {
		// Get the source from the override
		if override.SourceType == "default" {
			src, err := r.GetDefaultByID(override.SourceID)
			if err != nil {
				return nil, fmt.Errorf("failed to get default source from override: %w", err)
			}
			if src != nil {
				return &Source{
					ID:              src.ID,
					Name:            src.Name,
					URL:             src.URL,
					ComponentIDs:    src.ComponentIDs,
					RetrievalMethod: src.RetrievalMethod,
					URLTemplate:     src.URLTemplate,
					Priority:        src.Priority,
					Enabled:         src.Enabled,
					IsSystem:        true,
					CreatedAt:       src.CreatedAt,
					UpdatedAt:       src.UpdatedAt,
				}, nil
			}
		} else {
			src, err := r.GetUserSourceByID(override.SourceID)
			if err != nil {
				return nil, fmt.Errorf("failed to get user source from override: %w", err)
			}
			if src != nil {
				return &Source{
					ID:              src.ID,
					Name:            src.Name,
					URL:             src.URL,
					ComponentIDs:    src.ComponentIDs,
					RetrievalMethod: src.RetrievalMethod,
					URLTemplate:     src.URLTemplate,
					Priority:        src.Priority,
					Enabled:         src.Enabled,
					IsSystem:        false,
					OwnerID:         src.OwnerID,
					CreatedAt:       src.CreatedAt,
					UpdatedAt:       src.UpdatedAt,
				}, nil
			}
		}
	}

	// Fall back to priority-based selection from merged sources
	sources, err := r.GetMergedSourcesByComponent(userID, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merged sources: %w", err)
	}

	// Return the first enabled source (highest priority)
	for _, s := range sources {
		if s.Enabled {
			return &s, nil
		}
	}

	return nil, nil
}

// Distribution Source Override operations

// GetDistributionOverride retrieves a distribution source override
func (r *SourceRepository) GetDistributionOverride(distributionID, componentID string) (*DistributionSourceOverride, error) {
	query := `
		SELECT id, distribution_id, component_id, source_id, source_type, created_at
		FROM distribution_source_overrides
		WHERE distribution_id = ? AND component_id = ?
	`
	row := r.db.DB().QueryRow(query, distributionID, componentID)

	var o DistributionSourceOverride
	err := row.Scan(&o.ID, &o.DistributionID, &o.ComponentID, &o.SourceID, &o.SourceType, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution override: %w", err)
	}

	return &o, nil
}

// ListDistributionOverrides retrieves all overrides for a distribution
func (r *SourceRepository) ListDistributionOverrides(distributionID string) ([]DistributionSourceOverride, error) {
	query := `
		SELECT id, distribution_id, component_id, source_id, source_type, created_at
		FROM distribution_source_overrides
		WHERE distribution_id = ?
	`
	rows, err := r.db.DB().Query(query, distributionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list distribution overrides: %w", err)
	}
	defer rows.Close()

	var overrides []DistributionSourceOverride
	for rows.Next() {
		var o DistributionSourceOverride
		if err := rows.Scan(&o.ID, &o.DistributionID, &o.ComponentID, &o.SourceID, &o.SourceType, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan distribution override: %w", err)
		}
		overrides = append(overrides, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating distribution overrides: %w", err)
	}

	return overrides, nil
}

// SetDistributionOverride creates or updates a distribution source override
func (r *SourceRepository) SetDistributionOverride(o *DistributionSourceOverride) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	o.CreatedAt = time.Now()

	// Use INSERT OR REPLACE to handle both create and update
	query := `
		INSERT OR REPLACE INTO distribution_source_overrides (id, distribution_id, component_id, source_id, source_type, created_at)
		VALUES (
			COALESCE((SELECT id FROM distribution_source_overrides WHERE distribution_id = ? AND component_id = ?), ?),
			?, ?, ?, ?, ?
		)
	`
	_, err := r.db.DB().Exec(query,
		o.DistributionID, o.ComponentID, o.ID,
		o.DistributionID, o.ComponentID, o.SourceID, o.SourceType, o.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to set distribution override: %w", err)
	}

	return nil
}

// RemoveDistributionOverride removes a distribution source override
func (r *SourceRepository) RemoveDistributionOverride(distributionID, componentID string) error {
	result, err := r.db.DB().Exec(
		"DELETE FROM distribution_source_overrides WHERE distribution_id = ? AND component_id = ?",
		distributionID, componentID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove distribution override: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("distribution override not found")
	}

	return nil
}

// DeleteDistributionOverrides removes all overrides for a distribution
func (r *SourceRepository) DeleteDistributionOverrides(distributionID string) error {
	_, err := r.db.DB().Exec("DELETE FROM distribution_source_overrides WHERE distribution_id = ?", distributionID)
	if err != nil {
		return fmt.Errorf("failed to delete distribution overrides: %w", err)
	}
	return nil
}

// Helper functions

// scanDefaultSources scans multiple default source rows
func (r *SourceRepository) scanDefaultSources(rows *sql.Rows) ([]SourceDefault, error) {
	var sources []SourceDefault
	for rows.Next() {
		var s SourceDefault
		var componentIDsJSON, urlTemplate sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
			&s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan default source: %w", err)
		}
		s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
		s.URLTemplate = urlTemplate.String
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating default sources: %w", err)
	}

	return sources, nil
}

// scanUserSources scans multiple user source rows
func (r *SourceRepository) scanUserSources(rows *sql.Rows) ([]UserSource, error) {
	var sources []UserSource
	for rows.Next() {
		var s UserSource
		var componentIDsJSON, urlTemplate sql.NullString
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
			&s.Priority, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user source: %w", err)
		}
		s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
		s.URLTemplate = urlTemplate.String
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user sources: %w", err)
	}

	return sources, nil
}

// nullString returns sql.NullString for empty strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// parseComponentIDs parses a JSON array string into a slice of component IDs
func parseComponentIDs(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" {
		return []string{}
	}
	var ids []string
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		return []string{}
	}
	return ids
}

// serializeComponentIDs serializes a slice of component IDs to a JSON array string
func serializeComponentIDs(ids []string) string {
	if ids == nil || len(ids) == 0 {
		return "[]"
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(data)
}
