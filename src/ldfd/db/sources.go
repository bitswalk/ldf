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

// List retrieves all sources ordered by priority
func (r *SourceRepository) List() ([]UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// ListDefaults retrieves all system/default sources ordered by priority
func (r *SourceRepository) ListDefaults() ([]UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE is_system = 1
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list default sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// ListDefaultsByComponent retrieves all system sources for a specific component
func (r *SourceRepository) ListDefaultsByComponent(componentID string) ([]UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE is_system = 1 AND EXISTS (SELECT 1 FROM json_each(component_ids) WHERE value = ?)
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list default sources by component: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// GetDefaultByID retrieves a system source by ID
func (r *SourceRepository) GetDefaultByID(id string) (*UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE id = ? AND is_system = 1
	`
	row := r.db.DB().QueryRow(query, id)

	s, err := r.scanSource(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default source: %w", err)
	}

	return s, nil
}

// GetByID retrieves any source by ID (system or user)
func (r *SourceRepository) GetByID(id string) (*UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	s, err := r.scanSource(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return s, nil
}

// CreateDefault inserts a new system/default source
func (r *SourceRepository) CreateDefault(s *UpstreamSource) error {
	s.IsSystem = true
	s.OwnerID = ""
	return r.Create(s)
}

// Create inserts a new source
func (r *SourceRepository) Create(s *UpstreamSource) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.RetrievalMethod == "" {
		s.RetrievalMethod = "release"
	}
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}
	if s.ForgeType == "" {
		s.ForgeType = "generic"
	}

	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		INSERT INTO upstream_sources (id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, s.ID, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.ForgeType, s.VersionFilter, s.Priority, s.Enabled, s.IsSystem, nullString(s.OwnerID), s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	return nil
}

// UpdateDefault updates an existing system source
func (r *SourceRepository) UpdateDefault(s *UpstreamSource) error {
	s.IsSystem = true
	return r.Update(s)
}

// Update updates an existing source
func (r *SourceRepository) Update(s *UpstreamSource) error {
	s.UpdatedAt = time.Now()
	if s.ComponentIDs == nil {
		s.ComponentIDs = []string{}
	}
	if s.ForgeType == "" {
		s.ForgeType = "generic"
	}

	componentIDsJSON := serializeComponentIDs(s.ComponentIDs)

	query := `
		UPDATE upstream_sources
		SET name = ?, url = ?, component_ids = ?, retrieval_method = ?, url_template = ?, forge_type = ?, version_filter = ?, priority = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, s.Name, s.URL, componentIDsJSON, s.RetrievalMethod,
		nullString(s.URLTemplate), s.ForgeType, s.VersionFilter, s.Priority, s.Enabled, s.UpdatedAt, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("source not found: %s", s.ID)
	}

	return nil
}

// DeleteDefault removes a system source by ID
func (r *SourceRepository) DeleteDefault(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM upstream_sources WHERE id = ? AND is_system = 1", id)
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

// Delete removes any source by ID
func (r *SourceRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM upstream_sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("source not found: %s", id)
	}

	return nil
}

// ListUserSources retrieves all user sources for a specific user ordered by priority
func (r *SourceRepository) ListUserSources(ownerID string) ([]UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE is_system = 0 AND owner_id = ?
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// ListUserSourcesByComponent retrieves all user sources for a specific component
func (r *SourceRepository) ListUserSourcesByComponent(ownerID, componentID string) ([]UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE is_system = 0 AND owner_id = ? AND EXISTS (SELECT 1 FROM json_each(component_ids) WHERE value = ?)
		ORDER BY priority ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sources by component: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// GetUserSourceByID retrieves a user source by ID
func (r *SourceRepository) GetUserSourceByID(id string) (*UpstreamSource, error) {
	query := `
		SELECT id, name, url, component_ids, retrieval_method, url_template, forge_type, version_filter, priority, enabled, is_system, owner_id, created_at, updated_at
		FROM upstream_sources
		WHERE id = ? AND is_system = 0
	`
	row := r.db.DB().QueryRow(query, id)

	s, err := r.scanSource(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user source: %w", err)
	}

	return s, nil
}

// CreateUserSource inserts a new user source
func (r *SourceRepository) CreateUserSource(s *UpstreamSource) error {
	s.IsSystem = false
	return r.Create(s)
}

// UpdateUserSource updates an existing user source
func (r *SourceRepository) UpdateUserSource(s *UpstreamSource) error {
	s.IsSystem = false
	return r.Update(s)
}

// DeleteUserSource removes a user source by ID
func (r *SourceRepository) DeleteUserSource(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM upstream_sources WHERE id = ? AND is_system = 0", id)
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

// DeleteUserSourcesByOwner removes all user sources for a specific owner
func (r *SourceRepository) DeleteUserSourcesByOwner(ownerID string) (int64, error) {
	result, err := r.db.DB().Exec("DELETE FROM upstream_sources WHERE is_system = 0 AND owner_id = ?", ownerID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user sources: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return affected, nil
}

// GetMergedSources returns all sources (defaults + user-specific) merged and sorted by priority
func (r *SourceRepository) GetMergedSources(userID string) ([]UpstreamSource, error) {
	var sources []UpstreamSource

	// Get all system sources
	defaults, err := r.ListDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to get default sources: %w", err)
	}
	sources = append(sources, defaults...)

	// Get user sources if userID is provided
	if userID != "" {
		userSources, err := r.ListUserSources(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user sources: %w", err)
		}
		sources = append(sources, userSources...)
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
func (r *SourceRepository) GetMergedSourcesByComponent(userID, componentID string) ([]UpstreamSource, error) {
	var sources []UpstreamSource

	// Get default sources for this component
	defaults, err := r.ListDefaultsByComponent(componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default sources: %w", err)
	}
	sources = append(sources, defaults...)

	// Get user sources for this component if userID is provided
	if userID != "" {
		userSources, err := r.ListUserSourcesByComponent(userID, componentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user sources: %w", err)
		}
		sources = append(sources, userSources...)
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

// GetEffectiveSource returns the effective source for a component based on priority
// Sources are selected by priority (ascending) - the first enabled source wins
func (r *SourceRepository) GetEffectiveSource(componentID, userID string) (*UpstreamSource, error) {
	// Get sources for this component sorted by priority
	sources, err := r.GetMergedSourcesByComponent(userID, componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merged sources: %w", err)
	}

	// Return the first enabled source (highest priority)
	for i := range sources {
		if sources[i].Enabled {
			return &sources[i], nil
		}
	}

	return nil, nil
}

// Helper functions

// scanSources scans multiple source rows
func (r *SourceRepository) scanSources(rows *sql.Rows) ([]UpstreamSource, error) {
	var sources []UpstreamSource
	for rows.Next() {
		var s UpstreamSource
		var componentIDsJSON, urlTemplate, forgeType, versionFilter, ownerID sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
			&forgeType, &versionFilter, &s.Priority, &s.Enabled, &s.IsSystem, &ownerID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
		s.URLTemplate = urlTemplate.String
		s.ForgeType = forgeType.String
		if s.ForgeType == "" {
			s.ForgeType = "generic"
		}
		s.VersionFilter = versionFilter.String
		s.OwnerID = ownerID.String
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sources: %w", err)
	}

	return sources, nil
}

// scanSource scans a single source row
func (r *SourceRepository) scanSource(row *sql.Row) (*UpstreamSource, error) {
	var s UpstreamSource
	var componentIDsJSON, urlTemplate, forgeType, versionFilter, ownerID sql.NullString
	err := row.Scan(&s.ID, &s.Name, &s.URL, &componentIDsJSON, &s.RetrievalMethod, &urlTemplate,
		&forgeType, &versionFilter, &s.Priority, &s.Enabled, &s.IsSystem, &ownerID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.ComponentIDs = parseComponentIDs(componentIDsJSON.String)
	s.URLTemplate = urlTemplate.String
	s.ForgeType = forgeType.String
	if s.ForgeType == "" {
		s.ForgeType = "generic"
	}
	s.VersionFilter = versionFilter.String
	s.OwnerID = ownerID.String
	return &s, nil
}

// Deprecated scan functions for backwards compatibility
// These are aliases to the new unified scan functions

// scanDefaultSources is deprecated, use scanSources instead
func (r *SourceRepository) scanDefaultSources(rows *sql.Rows) ([]UpstreamSource, error) {
	return r.scanSources(rows)
}

// scanUserSources is deprecated, use scanSources instead
func (r *SourceRepository) scanUserSources(rows *sql.Rows) ([]UpstreamSource, error) {
	return r.scanSources(rows)
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
