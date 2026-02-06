package db

import (
	"database/sql"
	"fmt"
	"strings"
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE name = ?
	`
	row := r.db.DB().QueryRow(query, name)
	return r.scanComponent(row)
}

// GetByCategory retrieves all components that have the given category
// This handles comma-separated categories by matching any component where the category field contains the given value
func (r *ComponentRepository) GetByCategory(category string) ([]Component, error) {
	// Match exact category or category in comma-separated list
	// Uses LIKE patterns: "category" OR "category,%" OR "%,category" OR "%,category,%"
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE category = ?
		   OR category LIKE ?
		   OR category LIKE ?
		   OR category LIKE ?
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query, category, category+",%", "%,"+category, "%,"+category+",%")
	if err != nil {
		return nil, fmt.Errorf("failed to list components by category: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// GetByCategoryAndNameContains finds a component by category where name contains the given value
// This is used for dynamic component lookup based on distribution config values
// e.g., category="bootloader", nameContains="systemd-boot" could match "bootloader-systemd-boot" or "my-systemd-boot-variant"
// This handles comma-separated categories by matching any component that includes the given category
func (r *ComponentRepository) GetByCategoryAndNameContains(category, nameContains string) (*Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE (category = ? OR category LIKE ? OR category LIKE ? OR category LIKE ?)
		  AND name LIKE ?
		ORDER BY is_system DESC, name ASC
		LIMIT 1
	`
	// Use LIKE pattern to find components containing the config value
	namePattern := "%" + nameContains + "%"
	row := r.db.DB().QueryRow(query, category, category+",%", "%,"+category, "%,"+category+",%", namePattern)
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
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query,
		c.ID, c.Name, c.Category, c.DisplayName, c.Description, c.ArtifactPattern,
		c.DefaultURLTemplate, c.GitHubNormalizedTemplate, c.IsOptional, c.IsSystem,
		c.IsKernelModule, c.IsUserspace, ownerID,
		c.DefaultVersion, c.DefaultVersionRule, joinArchitectures(c.SupportedArchitectures),
		c.CreatedAt, c.UpdatedAt,
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
			is_kernel_module = ?, is_userspace = ?,
			default_version = ?, default_version_rule = ?, supported_architectures = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query,
		c.Name, c.Category, c.DisplayName, c.Description, c.ArtifactPattern,
		c.DefaultURLTemplate, c.GitHubNormalizedTemplate, c.IsOptional,
		c.IsKernelModule, c.IsUserspace,
		c.DefaultVersion, c.DefaultVersionRule, joinArchitectures(c.SupportedArchitectures),
		c.UpdatedAt, c.ID,
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
// This handles comma-separated categories by extracting each unique category
func (r *ComponentRepository) GetCategories() ([]string, error) {
	query := `SELECT DISTINCT category FROM components`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	// Use a map to collect unique categories
	categorySet := make(map[string]struct{})
	for rows.Next() {
		var categoryRaw string
		if err := rows.Scan(&categoryRaw); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		// Parse comma-separated categories and add each unique one
		for _, cat := range parseCategories(categoryRaw) {
			categorySet[cat] = struct{}{}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	// Convert map to sorted slice
	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	// Sort alphabetically
	for i := 0; i < len(categories)-1; i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[i] > categories[j] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	return categories, nil
}

// parseArchitectures splits a comma-separated architecture string into a typed slice
func parseArchitectures(raw string) []TargetArch {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	archs := make([]TargetArch, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			archs = append(archs, TargetArch(trimmed))
		}
	}
	return archs
}

// joinArchitectures converts a typed architecture slice to a comma-separated string
func joinArchitectures(archs []TargetArch) string {
	if len(archs) == 0 {
		return ""
	}
	parts := make([]string, len(archs))
	for i, a := range archs {
		parts[i] = string(a)
	}
	return strings.Join(parts, ",")
}

// parseCategories splits a comma-separated category string into a slice
func parseCategories(category string) []string {
	if category == "" {
		return nil
	}
	parts := strings.Split(category, ",")
	categories := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			categories = append(categories, trimmed)
		}
	}
	return categories
}

// scanComponent scans a single component row
func (r *ComponentRepository) scanComponent(row *sql.Row) (*Component, error) {
	var c Component
	var categoryRaw string
	var description, artifactPattern, defaultTemplate, githubTemplate, ownerID sql.NullString
	var defaultVersion, defaultVersionRule, supportedArchs sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &categoryRaw, &c.DisplayName,
		&description, &artifactPattern, &defaultTemplate, &githubTemplate,
		&c.IsOptional, &c.IsSystem, &c.IsKernelModule, &c.IsUserspace, &ownerID,
		&defaultVersion, &defaultVersionRule, &supportedArchs, &c.CreatedAt, &c.UpdatedAt,
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
	c.SupportedArchitectures = parseArchitectures(supportedArchs.String)

	// Parse categories from comma-separated string
	c.Categories = parseCategories(categoryRaw)
	// Set primary category to first one
	if len(c.Categories) > 0 {
		c.Category = c.Categories[0]
	} else {
		c.Category = categoryRaw
	}

	return &c, nil
}

// scanComponents scans multiple component rows
func (r *ComponentRepository) scanComponents(rows *sql.Rows) ([]Component, error) {
	var components []Component

	for rows.Next() {
		var c Component
		var categoryRaw string
		var description, artifactPattern, defaultTemplate, githubTemplate, ownerID sql.NullString
		var defaultVersion, defaultVersionRule, supportedArchs sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Name, &categoryRaw, &c.DisplayName,
			&description, &artifactPattern, &defaultTemplate, &githubTemplate,
			&c.IsOptional, &c.IsSystem, &c.IsKernelModule, &c.IsUserspace, &ownerID,
			&defaultVersion, &defaultVersionRule, &supportedArchs, &c.CreatedAt, &c.UpdatedAt,
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
		c.SupportedArchitectures = parseArchitectures(supportedArchs.String)

		// Parse categories from comma-separated string
		c.Categories = parseCategories(categoryRaw)
		// Set primary category to first one
		if len(c.Categories) > 0 {
			c.Category = c.Categories[0]
		} else {
			c.Category = categoryRaw
		}

		components = append(components, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating components: %w", err)
	}

	return components, nil
}

// ListKernelModules retrieves all components that are kernel modules
func (r *ComponentRepository) ListKernelModules() ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE is_kernel_module = 1
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list kernel modules: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// ListUserspace retrieves all components that are userspace tools
func (r *ComponentRepository) ListUserspace() ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE is_userspace = 1
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list userspace components: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}

// ListHybrid retrieves components that are both kernel modules AND userspace tools
func (r *ComponentRepository) ListHybrid() ([]Component, error) {
	query := `
		SELECT id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, owner_id,
			default_version, default_version_rule, supported_architectures, created_at, updated_at
		FROM components
		WHERE is_kernel_module = 1 AND is_userspace = 1
		ORDER BY category ASC, name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list hybrid components: %w", err)
	}
	defer rows.Close()

	return r.scanComponents(rows)
}
