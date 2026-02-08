package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ToolchainProfileRepository handles toolchain profile database operations
type ToolchainProfileRepository struct {
	db *Database
}

// NewToolchainProfileRepository creates a new toolchain profile repository
func NewToolchainProfileRepository(db *Database) *ToolchainProfileRepository {
	return &ToolchainProfileRepository{db: db}
}

// List retrieves all toolchain profiles ordered by name
func (r *ToolchainProfileRepository) List() ([]ToolchainProfile, error) {
	query := `
		SELECT id, name, display_name, description, type, config, is_system, owner_id, created_at, updated_at
		FROM toolchain_profiles
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list toolchain profiles: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// ListByType retrieves all toolchain profiles for a specific type (gcc or llvm)
func (r *ToolchainProfileRepository) ListByType(toolchainType string) ([]ToolchainProfile, error) {
	query := `
		SELECT id, name, display_name, description, type, config, is_system, owner_id, created_at, updated_at
		FROM toolchain_profiles
		WHERE type = ?
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query, toolchainType)
	if err != nil {
		return nil, fmt.Errorf("failed to list toolchain profiles by type: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// GetByID retrieves a toolchain profile by ID
func (r *ToolchainProfileRepository) GetByID(id string) (*ToolchainProfile, error) {
	query := `
		SELECT id, name, display_name, description, type, config, is_system, owner_id, created_at, updated_at
		FROM toolchain_profiles
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	tp, err := r.scanProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get toolchain profile: %w", err)
	}

	return tp, nil
}

// GetByName retrieves a toolchain profile by unique name
func (r *ToolchainProfileRepository) GetByName(name string) (*ToolchainProfile, error) {
	query := `
		SELECT id, name, display_name, description, type, config, is_system, owner_id, created_at, updated_at
		FROM toolchain_profiles
		WHERE name = ?
	`
	row := r.db.DB().QueryRow(query, name)

	tp, err := r.scanProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get toolchain profile by name: %w", err)
	}

	return tp, nil
}

// Create inserts a new toolchain profile
func (r *ToolchainProfileRepository) Create(tp *ToolchainProfile) error {
	if tp.ID == "" {
		tp.ID = uuid.New().String()
	}

	now := time.Now()
	tp.CreatedAt = now
	tp.UpdatedAt = now

	configJSON, err := json.Marshal(tp.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize toolchain config: %w", err)
	}

	query := `
		INSERT INTO toolchain_profiles (id, name, display_name, description, type, config, is_system, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.DB().Exec(query, tp.ID, tp.Name, tp.DisplayName, tp.Description,
		tp.Type, string(configJSON), tp.IsSystem, nullString(tp.OwnerID), tp.CreatedAt, tp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create toolchain profile: %w", err)
	}

	return nil
}

// Update updates an existing toolchain profile
func (r *ToolchainProfileRepository) Update(tp *ToolchainProfile) error {
	tp.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(tp.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize toolchain config: %w", err)
	}

	query := `
		UPDATE toolchain_profiles
		SET name = ?, display_name = ?, description = ?, config = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, tp.Name, tp.DisplayName, tp.Description,
		string(configJSON), tp.UpdatedAt, tp.ID)
	if err != nil {
		return fmt.Errorf("failed to update toolchain profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("toolchain profile not found: %s", tp.ID)
	}

	return nil
}

// Delete removes a toolchain profile by ID
func (r *ToolchainProfileRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM toolchain_profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete toolchain profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("toolchain profile not found: %s", id)
	}

	return nil
}

// scanProfiles scans multiple toolchain profile rows
func (r *ToolchainProfileRepository) scanProfiles(rows *sql.Rows) ([]ToolchainProfile, error) {
	var profiles []ToolchainProfile
	for rows.Next() {
		var tp ToolchainProfile
		var configJSON, ownerID sql.NullString
		if err := rows.Scan(&tp.ID, &tp.Name, &tp.DisplayName, &tp.Description, &tp.Type,
			&configJSON, &tp.IsSystem, &ownerID, &tp.CreatedAt, &tp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan toolchain profile: %w", err)
		}
		tp.OwnerID = ownerID.String
		if configJSON.Valid && configJSON.String != "" {
			if err := json.Unmarshal([]byte(configJSON.String), &tp.Config); err != nil {
				return nil, fmt.Errorf("failed to deserialize toolchain config: %w", err)
			}
		}
		profiles = append(profiles, tp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating toolchain profiles: %w", err)
	}

	return profiles, nil
}

// scanProfile scans a single toolchain profile row
func (r *ToolchainProfileRepository) scanProfile(row *sql.Row) (*ToolchainProfile, error) {
	var tp ToolchainProfile
	var configJSON, ownerID sql.NullString
	err := row.Scan(&tp.ID, &tp.Name, &tp.DisplayName, &tp.Description, &tp.Type,
		&configJSON, &tp.IsSystem, &ownerID, &tp.CreatedAt, &tp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	tp.OwnerID = ownerID.String
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &tp.Config); err != nil {
			return nil, fmt.Errorf("failed to deserialize toolchain config: %w", err)
		}
	}
	return &tp, nil
}
