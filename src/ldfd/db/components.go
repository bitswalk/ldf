package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ComponentRepository handles component database operations
type ComponentRepository struct {
	db *Database
}

// NewComponentRepository creates a new component repository
func NewComponentRepository(db *Database) *ComponentRepository {
	return &ComponentRepository{db: db}
}

// List retrieves all components ordered by category and name
func (r *ComponentRepository) List() ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list components: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// ListByOwner retrieves all components owned by a specific user
func (r *ComponentRepository) ListByOwner(ownerID string) ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE owner_id = ?
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list components by owner: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// ListSystem retrieves all system (default) components
func (r *ComponentRepository) ListSystem() ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE is_system = 1
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list system components: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// GetByID retrieves a component by ID
func (r *ComponentRepository) GetByID(id string) (*Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)
	return r.scanComponent(row)
}

// GetByName retrieves a component by name
func (r *ComponentRepository) GetByName(name string) (*Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE name = ?
	`
	row := r.db.DB().QueryRow(query, name)
	return r.scanComponent(row)
}

// GetByCategory retrieves all components in a category
func (r *ComponentRepository) GetByCategory(category string) ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE category = ?
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to list components by category: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// GetByCategoryAndNameContains finds a component by category where name contains the given value
// This is used for dynamic component lookup based on distribution config values
// e.g., category="bootloader", nameContains="systemd-boot" could match "bootloader-systemd-boot" or "my-systemd-boot-variant"
func (r *ComponentRepository) GetByCategoryAndNameContains(category, nameContains string) (*Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at
		FROM components
		WHERE category = ? AND name LIKE ?
		ORDER BY is_system DESC, name ASC
		LIMIT 1
	`
	// Use LIKE pattern to find components containing the config value
	pattern := "%" + nameContains + "%"
	row := r.db.DB().QueryRow(query, category, pattern)
	return r.scanComponent(row)
}

// Create inserts a new component
func (r *ComponentRepository) Create(c *Component) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}

	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	// Handle nullable owner_id
	var ownerID interface{}
	if c.OwnerID != "" {
		ownerID = c.OwnerID
	}

	// Set default version rule if not specified
	if c.DefaultVersionRule == "" {
		c.DefaultVersionRule = VersionRuleLatestStable
	}

	query := `
		INSERT INTO components (id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, owner_id,
			default_version, default_version_rule, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query,
		c.ID, c.Name, c.Category, c.DisplayName, c.Description, c.ArtifactPattern,
		c.DefaultURLTemplate, c.GitHubNormalizedTemplate, c.IsOptional, c.IsSystem, ownerID,
		c.DefaultVersion, c.DefaultVersionRule, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create component: %w", err)
	}

	return nil
}

// Update updates an existing component
func (r *ComponentRepository) Update(c *Component) error {
	c.UpdatedAt = time.Now()

	query := `
		UPDATE components
		SET name = ?, category = ?, display_name = ?, description = ?, artifact_pattern = ?,
			default_url_template = ?, github_normalized_template = ?, is_optional = ?,
			default_version = ?, default_version_rule = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query,
		c.Name, c.Category, c.DisplayName, c.Description, c.ArtifactPattern,
		c.DefaultURLTemplate, c.GitHubNormalizedTemplate, c.IsOptional,
		c.DefaultVersion, c.DefaultVersionRule, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update component: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("component not found: %s", c.ID)
	}

	return nil
}

// Delete removes a component by ID
func (r *ComponentRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM components WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete component: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("component not found: %s", id)
	}

	return nil
}

// GetCategories returns all distinct component categories
func (r *ComponentRepository) GetCategories() ([]string, error) {
	query := `SELECT DISTINCT category FROM components ORDER BY category ASC`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	return categories, nil
}

// scanComponent scans a single component row
func (r *ComponentRepository) scanComponent(row *sql.Row) (*Component, error) {
	var c Component
	var description, artifactPattern, defaultTemplate, githubTemplate, ownerID sql.NullString
	var defaultVersion, defaultVersionRule sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &c.Category, &c.DisplayName,
		&description, &artifactPattern, &defaultTemplate, &githubTemplate,
		&c.IsOptional, &c.IsSystem, &ownerID,
		&defaultVersion, &defaultVersionRule, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan component: %w", err)
	}

	c.Description = description.String
	c.ArtifactPattern = artifactPattern.String
	c.DefaultURLTemplate = defaultTemplate.String
	c.GitHubNormalizedTemplate = githubTemplate.String
	c.OwnerID = ownerID.String
	c.DefaultVersion = defaultVersion.String
	c.DefaultVersionRule = VersionRule(defaultVersionRule.String)

	return &c, nil
}

// scanComponents scans multiple component rows
func (r *ComponentRepository) scanComponents(rows *sql.Rows) ([]Component, error) {
	var components []Component

	for rows.Next() {
		var c Component
		var description, artifactPattern, defaultTemplate, githubTemplate, ownerID sql.NullString
		var defaultVersion, defaultVersionRule sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Name, &c.Category, &c.DisplayName,
			&description, &artifactPattern, &defaultTemplate, &githubTemplate,
			&c.IsOptional, &c.IsSystem, &ownerID,
			&defaultVersion, &defaultVersionRule, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan component: %w", err)
		}

		c.Description = description.String
		c.ArtifactPattern = artifactPattern.String
		c.DefaultURLTemplate = defaultTemplate.String
		c.GitHubNormalizedTemplate = githubTemplate.String
		c.OwnerID = ownerID.String
		c.DefaultVersion = defaultVersion.String
		c.DefaultVersionRule = VersionRule(defaultVersionRule.String)

		components = append(components, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating components: %w", err)
	}

	return components, nil
}
