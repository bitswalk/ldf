package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BoardProfileRepository handles board profile database operations
type BoardProfileRepository struct {
	db *Database
}

// NewBoardProfileRepository creates a new board profile repository
func NewBoardProfileRepository(db *Database) *BoardProfileRepository {
	return &BoardProfileRepository{db: db}
}

// List retrieves all board profiles ordered by name
func (r *BoardProfileRepository) List() ([]BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list board profiles: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// ListSystem retrieves all system board profiles
func (r *BoardProfileRepository) ListSystem() ([]BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		WHERE is_system = 1
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list system board profiles: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// ListByArch retrieves all board profiles for a specific architecture
func (r *BoardProfileRepository) ListByArch(arch TargetArch) ([]BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		WHERE arch = ?
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query, string(arch))
	if err != nil {
		return nil, fmt.Errorf("failed to list board profiles by arch: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// ListByOwner retrieves all board profiles for a specific owner
func (r *BoardProfileRepository) ListByOwner(ownerID string) ([]BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		WHERE is_system = 0 AND owner_id = ?
		ORDER BY name ASC
	`
	rows, err := r.db.DB().Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list board profiles by owner: %w", err)
	}
	defer rows.Close()

	return r.scanProfiles(rows)
}

// GetByID retrieves a board profile by ID
func (r *BoardProfileRepository) GetByID(id string) (*BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	bp, err := r.scanProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get board profile: %w", err)
	}

	return bp, nil
}

// GetByName retrieves a board profile by unique name
func (r *BoardProfileRepository) GetByName(name string) (*BoardProfile, error) {
	query := `
		SELECT id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at
		FROM board_profiles
		WHERE name = ?
	`
	row := r.db.DB().QueryRow(query, name)

	bp, err := r.scanProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get board profile by name: %w", err)
	}

	return bp, nil
}

// Create inserts a new board profile
func (r *BoardProfileRepository) Create(bp *BoardProfile) error {
	if bp.ID == "" {
		bp.ID = uuid.New().String()
	}

	now := time.Now()
	bp.CreatedAt = now
	bp.UpdatedAt = now

	configJSON, err := json.Marshal(bp.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize board config: %w", err)
	}

	query := `
		INSERT INTO board_profiles (id, name, display_name, description, arch, config, is_system, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.DB().Exec(query, bp.ID, bp.Name, bp.DisplayName, bp.Description,
		string(bp.Arch), string(configJSON), bp.IsSystem, nullString(bp.OwnerID), bp.CreatedAt, bp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create board profile: %w", err)
	}

	return nil
}

// Update updates an existing board profile
func (r *BoardProfileRepository) Update(bp *BoardProfile) error {
	bp.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(bp.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize board config: %w", err)
	}

	query := `
		UPDATE board_profiles
		SET name = ?, display_name = ?, description = ?, config = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, bp.Name, bp.DisplayName, bp.Description,
		string(configJSON), bp.UpdatedAt, bp.ID)
	if err != nil {
		return fmt.Errorf("failed to update board profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("board profile not found: %s", bp.ID)
	}

	return nil
}

// Delete removes a board profile by ID
func (r *BoardProfileRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM board_profiles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete board profile: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("board profile not found: %s", id)
	}

	return nil
}

// scanProfiles scans multiple board profile rows
func (r *BoardProfileRepository) scanProfiles(rows *sql.Rows) ([]BoardProfile, error) {
	var profiles []BoardProfile
	for rows.Next() {
		var bp BoardProfile
		var configJSON, ownerID sql.NullString
		var arch string
		if err := rows.Scan(&bp.ID, &bp.Name, &bp.DisplayName, &bp.Description, &arch,
			&configJSON, &bp.IsSystem, &ownerID, &bp.CreatedAt, &bp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan board profile: %w", err)
		}
		bp.Arch = TargetArch(arch)
		bp.OwnerID = ownerID.String
		if configJSON.Valid && configJSON.String != "" {
			if err := json.Unmarshal([]byte(configJSON.String), &bp.Config); err != nil {
				return nil, fmt.Errorf("failed to deserialize board config: %w", err)
			}
		}
		profiles = append(profiles, bp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating board profiles: %w", err)
	}

	return profiles, nil
}

// scanProfile scans a single board profile row
func (r *BoardProfileRepository) scanProfile(row *sql.Row) (*BoardProfile, error) {
	var bp BoardProfile
	var configJSON, ownerID sql.NullString
	var arch string
	err := row.Scan(&bp.ID, &bp.Name, &bp.DisplayName, &bp.Description, &arch,
		&configJSON, &bp.IsSystem, &ownerID, &bp.CreatedAt, &bp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	bp.Arch = TargetArch(arch)
	bp.OwnerID = ownerID.String
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &bp.Config); err != nil {
			return nil, fmt.Errorf("failed to deserialize board config: %w", err)
		}
	}
	return &bp, nil
}
