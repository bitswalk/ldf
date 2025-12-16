package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DistributionConfig represents the full configuration for building a distribution
type DistributionConfig struct {
	Core     CoreConfig     `json:"core"`
	System   SystemConfig   `json:"system"`
	Security SecurityConfig `json:"security"`
	Runtime  RuntimeConfig  `json:"runtime"`
	Target   TargetConfig   `json:"target"`
}

// CoreConfig contains core system configuration
type CoreConfig struct {
	Kernel       KernelConfig       `json:"kernel"`
	Bootloader   string             `json:"bootloader"`
	Partitioning PartitioningConfig `json:"partitioning"`
}

// KernelConfig contains kernel configuration
type KernelConfig struct {
	Version string `json:"version"`
}

// PartitioningConfig contains partitioning configuration
type PartitioningConfig struct {
	Type string `json:"type"`
	Mode string `json:"mode"`
}

// SystemConfig contains system services configuration
type SystemConfig struct {
	Init           string           `json:"init"`
	Filesystem     FilesystemConfig `json:"filesystem"`
	PackageManager string           `json:"packageManager"`
}

// FilesystemConfig contains filesystem configuration
type FilesystemConfig struct {
	Type      string `json:"type"`
	Hierarchy string `json:"hierarchy"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	System string `json:"system"`
}

// RuntimeConfig contains runtime configuration
type RuntimeConfig struct {
	Container      string `json:"container"`
	Virtualization string `json:"virtualization"`
}

// TargetConfig contains target environment configuration
type TargetConfig struct {
	Type    string         `json:"type"`
	Desktop *DesktopConfig `json:"desktop,omitempty"`
}

// DesktopConfig contains desktop environment configuration
type DesktopConfig struct {
	Environment   string `json:"environment"`
	DisplayServer string `json:"displayServer"`
}

// DistributionStatus represents the status of a distribution
type DistributionStatus string

const (
	StatusPending     DistributionStatus = "pending"
	StatusDownloading DistributionStatus = "downloading"
	StatusValidating  DistributionStatus = "validating"
	StatusReady       DistributionStatus = "ready"
	StatusFailed      DistributionStatus = "failed"
	StatusDeleted     DistributionStatus = "deleted"
)

// Visibility represents the visibility level of a distribution
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Distribution represents a distribution record
type Distribution struct {
	ID           int64               `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Status       DistributionStatus  `json:"status"`
	Visibility   Visibility          `json:"visibility"`
	Config       *DistributionConfig `json:"config,omitempty"`
	SourceURL    string              `json:"source_url,omitempty"`
	Checksum     string              `json:"checksum,omitempty"`
	SizeBytes    int64               `json:"size_bytes"`
	OwnerID      string              `json:"owner_id,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

// DistributionLog represents a log entry for a distribution
type DistributionLog struct {
	ID             int64     `json:"id"`
	DistributionID int64     `json:"distribution_id"`
	Level          string    `json:"level"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// DistributionRepository handles distribution database operations
type DistributionRepository struct {
	db *Database
}

// NewDistributionRepository creates a new distribution repository
func NewDistributionRepository(db *Database) *DistributionRepository {
	return &DistributionRepository{db: db}
}

// Create inserts a new distribution
func (r *DistributionRepository) Create(d *Distribution) error {
	var configJSON []byte
	var err error
	if d.Config != nil {
		configJSON, err = json.Marshal(d.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	// Handle optional owner_id
	var ownerID interface{}
	if d.OwnerID != "" {
		ownerID = d.OwnerID
	} else {
		ownerID = nil
	}

	// Default to private visibility if not set
	if d.Visibility == "" {
		d.Visibility = VisibilityPrivate
	}

	query := `
		INSERT INTO distributions (name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.DB().Exec(query, d.Name, d.Version, d.Status, d.Visibility, configJSON, d.SourceURL, d.Checksum, d.SizeBytes, ownerID)
	if err != nil {
		return fmt.Errorf("failed to create distribution: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	d.ID = id

	return nil
}

// GetByID retrieves a distribution by ID
func (r *DistributionRepository) GetByID(id int64) (*Distribution, error) {
	query := `
		SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
		       created_at, updated_at, started_at, completed_at, error_message
		FROM distributions
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)
	return r.scanDistribution(row)
}

// GetByName retrieves a distribution by name
func (r *DistributionRepository) GetByName(name string) (*Distribution, error) {
	query := `
		SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
		       created_at, updated_at, started_at, completed_at, error_message
		FROM distributions
		WHERE name = ?
	`
	row := r.db.DB().QueryRow(query, name)
	return r.scanDistribution(row)
}

// List retrieves all distributions with optional status filter (admin use only)
func (r *DistributionRepository) List(status *DistributionStatus) ([]Distribution, error) {
	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
			       created_at, updated_at, started_at, completed_at, error_message
			FROM distributions
			WHERE status = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{*status}
	} else {
		query = `
			SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
			       created_at, updated_at, started_at, completed_at, error_message
			FROM distributions
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.DB().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list distributions: %w", err)
	}
	defer rows.Close()

	var distributions []Distribution
	for rows.Next() {
		d, err := r.scanDistributionRows(rows)
		if err != nil {
			return nil, err
		}
		distributions = append(distributions, *d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating distributions: %w", err)
	}

	return distributions, nil
}

// ListAccessible retrieves distributions accessible to a user
// - If userID is empty (anonymous): only public distributions
// - If userID is set: public distributions + private distributions owned by the user
// - If isAdmin is true: all distributions
func (r *DistributionRepository) ListAccessible(userID string, isAdmin bool, status *DistributionStatus) ([]Distribution, error) {
	var query string
	var args []interface{}

	if isAdmin {
		// Admin sees everything
		return r.List(status)
	}

	if userID == "" {
		// Anonymous: only public
		if status != nil {
			query = `
				SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
				       created_at, updated_at, started_at, completed_at, error_message
				FROM distributions
				WHERE visibility = 'public' AND status = ?
				ORDER BY created_at DESC
			`
			args = []interface{}{*status}
		} else {
			query = `
				SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
				       created_at, updated_at, started_at, completed_at, error_message
				FROM distributions
				WHERE visibility = 'public'
				ORDER BY created_at DESC
			`
		}
	} else {
		// Authenticated user: public + own private
		if status != nil {
			query = `
				SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
				       created_at, updated_at, started_at, completed_at, error_message
				FROM distributions
				WHERE (visibility = 'public' OR owner_id = ?) AND status = ?
				ORDER BY created_at DESC
			`
			args = []interface{}{userID, *status}
		} else {
			query = `
				SELECT id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id,
				       created_at, updated_at, started_at, completed_at, error_message
				FROM distributions
				WHERE visibility = 'public' OR owner_id = ?
				ORDER BY created_at DESC
			`
			args = []interface{}{userID}
		}
	}

	rows, err := r.db.DB().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list distributions: %w", err)
	}
	defer rows.Close()

	var distributions []Distribution
	for rows.Next() {
		d, err := r.scanDistributionRows(rows)
		if err != nil {
			return nil, err
		}
		distributions = append(distributions, *d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating distributions: %w", err)
	}

	return distributions, nil
}

// CanUserAccess checks if a user can access a specific distribution
func (r *DistributionRepository) CanUserAccess(distributionID int64, userID string, isAdmin bool) (bool, error) {
	if isAdmin {
		return true, nil
	}

	d, err := r.GetByID(distributionID)
	if err != nil {
		return false, err
	}
	if d == nil {
		return false, nil
	}

	// Public distributions are accessible to anyone
	if d.Visibility == VisibilityPublic {
		return true, nil
	}

	// Private distributions are only accessible to the owner
	if userID != "" && d.OwnerID == userID {
		return true, nil
	}

	return false, nil
}

// UpdateStatus updates the status of a distribution
func (r *DistributionRepository) UpdateStatus(id int64, status DistributionStatus, errorMsg string) error {
	var query string
	var args []interface{}

	now := time.Now()

	switch status {
	case StatusDownloading, StatusValidating:
		query = `
			UPDATE distributions
			SET status = ?, started_at = ?, updated_at = ?, error_message = ?
			WHERE id = ?
		`
		args = []interface{}{status, now, now, errorMsg, id}
	case StatusReady:
		query = `
			UPDATE distributions
			SET status = ?, completed_at = ?, updated_at = ?, error_message = ?
			WHERE id = ?
		`
		args = []interface{}{status, now, now, errorMsg, id}
	case StatusFailed:
		query = `
			UPDATE distributions
			SET status = ?, completed_at = ?, updated_at = ?, error_message = ?
			WHERE id = ?
		`
		args = []interface{}{status, now, now, errorMsg, id}
	default:
		query = `
			UPDATE distributions
			SET status = ?, updated_at = ?, error_message = ?
			WHERE id = ?
		`
		args = []interface{}{status, now, errorMsg, id}
	}

	result, err := r.db.DB().Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update distribution status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("distribution not found: %d", id)
	}

	return nil
}

// Update updates a distribution
func (r *DistributionRepository) Update(d *Distribution) error {
	query := `
		UPDATE distributions
		SET name = ?, version = ?, status = ?, visibility = ?, source_url = ?, checksum = ?,
		    size_bytes = ?, updated_at = ?, error_message = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query,
		d.Name, d.Version, d.Status, d.Visibility, d.SourceURL, d.Checksum,
		d.SizeBytes, time.Now(), d.ErrorMessage, d.ID)
	if err != nil {
		return fmt.Errorf("failed to update distribution: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("distribution not found: %d", d.ID)
	}

	return nil
}

// Delete removes a distribution by ID
func (r *DistributionRepository) Delete(id int64) error {
	result, err := r.db.DB().Exec("DELETE FROM distributions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete distribution: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("distribution not found: %d", id)
	}

	return nil
}

// AddLog adds a log entry for a distribution
func (r *DistributionRepository) AddLog(distributionID int64, level, message string) error {
	query := `INSERT INTO distribution_logs (distribution_id, level, message) VALUES (?, ?, ?)`
	_, err := r.db.DB().Exec(query, distributionID, level, message)
	if err != nil {
		return fmt.Errorf("failed to add distribution log: %w", err)
	}
	return nil
}

// GetLogs retrieves logs for a distribution
func (r *DistributionRepository) GetLogs(distributionID int64, limit int) ([]DistributionLog, error) {
	query := `
		SELECT id, distribution_id, level, message, created_at
		FROM distribution_logs
		WHERE distribution_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := r.db.DB().Query(query, distributionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution logs: %w", err)
	}
	defer rows.Close()

	var logs []DistributionLog
	for rows.Next() {
		var log DistributionLog
		if err := rows.Scan(&log.ID, &log.DistributionID, &log.Level, &log.Message, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan distribution log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetStats returns distribution statistics
func (r *DistributionRepository) GetStats() (map[string]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM distributions
		GROUP BY status
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}

// scanDistribution scans a single row into a Distribution
func (r *DistributionRepository) scanDistribution(row *sql.Row) (*Distribution, error) {
	var d Distribution
	var configJSON, sourceURL, checksum, ownerID, errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime

	err := row.Scan(
		&d.ID, &d.Name, &d.Version, &d.Status, &d.Visibility, &configJSON, &sourceURL, &checksum, &d.SizeBytes, &ownerID,
		&d.CreatedAt, &d.UpdatedAt, &startedAt, &completedAt, &errorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan distribution: %w", err)
	}

	if configJSON.Valid && configJSON.String != "" {
		var config DistributionConfig
		if err := json.Unmarshal([]byte(configJSON.String), &config); err == nil {
			d.Config = &config
		}
	}
	if sourceURL.Valid {
		d.SourceURL = sourceURL.String
	}
	if checksum.Valid {
		d.Checksum = checksum.String
	}
	if ownerID.Valid {
		d.OwnerID = ownerID.String
	}
	if errorMessage.Valid {
		d.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}

	return &d, nil
}

// scanDistributionRows scans rows into a Distribution
func (r *DistributionRepository) scanDistributionRows(rows *sql.Rows) (*Distribution, error) {
	var d Distribution
	var configJSON, sourceURL, checksum, ownerID, errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime

	err := rows.Scan(
		&d.ID, &d.Name, &d.Version, &d.Status, &d.Visibility, &configJSON, &sourceURL, &checksum, &d.SizeBytes, &ownerID,
		&d.CreatedAt, &d.UpdatedAt, &startedAt, &completedAt, &errorMessage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan distribution: %w", err)
	}

	if configJSON.Valid && configJSON.String != "" {
		var config DistributionConfig
		if err := json.Unmarshal([]byte(configJSON.String), &config); err == nil {
			d.Config = &config
		}
	}
	if sourceURL.Valid {
		d.SourceURL = sourceURL.String
	}
	if checksum.Valid {
		d.Checksum = checksum.String
	}
	if ownerID.Valid {
		d.OwnerID = ownerID.String
	}
	if errorMessage.Valid {
		d.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		d.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		d.CompletedAt = &completedAt.Time
	}

	return &d, nil
}
