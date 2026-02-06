package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ArtifactCacheEntry represents a cached artifact shared across distributions
type ArtifactCacheEntry struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id"`
	Version     string    `json:"version"`
	Checksum    string    `json:"checksum"`
	CachePath   string    `json:"cache_path"`
	SizeBytes   int64     `json:"size_bytes"`
	ContentType string    `json:"content_type"`
	ResolvedURL string    `json:"resolved_url"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	UseCount    int       `json:"use_count"`
}

// ArtifactCacheRepository handles artifact cache database operations
type ArtifactCacheRepository struct {
	db *Database
}

// NewArtifactCacheRepository creates a new artifact cache repository
func NewArtifactCacheRepository(db *Database) *ArtifactCacheRepository {
	return &ArtifactCacheRepository{db: db}
}

// Create inserts a new artifact cache entry
func (r *ArtifactCacheRepository) Create(entry *ArtifactCacheEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	entry.CreatedAt = time.Now()
	entry.LastUsedAt = time.Now()
	if entry.UseCount == 0 {
		entry.UseCount = 1
	}

	_, err := r.db.DB().Exec(`
		INSERT INTO artifact_cache (id, source_id, version, checksum, cache_path,
			size_bytes, content_type, resolved_url, created_at, last_used_at, use_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.SourceID, entry.Version, entry.Checksum, entry.CachePath,
		entry.SizeBytes, entry.ContentType, entry.ResolvedURL,
		entry.CreatedAt, entry.LastUsedAt, entry.UseCount,
	)
	if err != nil {
		return fmt.Errorf("failed to create artifact cache entry: %w", err)
	}
	return nil
}

// GetBySourceAndVersion retrieves a cache entry by source ID and version
func (r *ArtifactCacheRepository) GetBySourceAndVersion(sourceID, version string) (*ArtifactCacheEntry, error) {
	row := r.db.DB().QueryRow(`
		SELECT id, source_id, version, checksum, cache_path, size_bytes,
			content_type, resolved_url, created_at, last_used_at, use_count
		FROM artifact_cache WHERE source_id = ? AND version = ?`,
		sourceID, version,
	)
	return r.scanEntry(row)
}

// TouchLastUsed updates last_used_at and increments use_count
func (r *ArtifactCacheRepository) TouchLastUsed(id string) error {
	_, err := r.db.DB().Exec(`
		UPDATE artifact_cache SET last_used_at = ?, use_count = use_count + 1 WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

// Delete removes a cache entry by ID
func (r *ArtifactCacheRepository) Delete(id string) error {
	result, err := r.db.DB().Exec(`DELETE FROM artifact_cache WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete artifact cache entry: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("artifact cache entry not found: %s", id)
	}
	return nil
}

// ListLRU returns cache entries ordered by least recently used (oldest first)
func (r *ArtifactCacheRepository) ListLRU(limit int) ([]ArtifactCacheEntry, error) {
	rows, err := r.db.DB().Query(`
		SELECT id, source_id, version, checksum, cache_path, size_bytes,
			content_type, resolved_url, created_at, last_used_at, use_count
		FROM artifact_cache ORDER BY last_used_at ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact cache entries: %w", err)
	}
	defer rows.Close()

	var entries []ArtifactCacheEntry
	for rows.Next() {
		var e ArtifactCacheEntry
		if err := rows.Scan(
			&e.ID, &e.SourceID, &e.Version, &e.Checksum, &e.CachePath,
			&e.SizeBytes, &e.ContentType, &e.ResolvedURL,
			&e.CreatedAt, &e.LastUsedAt, &e.UseCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan artifact cache entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// TotalSize returns the sum of all cached artifact sizes in bytes
func (r *ArtifactCacheRepository) TotalSize() (int64, error) {
	var total sql.NullInt64
	err := r.db.DB().QueryRow(`SELECT SUM(size_bytes) FROM artifact_cache`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total cache size: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// Count returns the number of entries in the cache
func (r *ArtifactCacheRepository) Count() (int, error) {
	var count int
	err := r.db.DB().QueryRow(`SELECT COUNT(*) FROM artifact_cache`).Scan(&count)
	return count, err
}

// scanEntry scans a single row into an ArtifactCacheEntry
func (r *ArtifactCacheRepository) scanEntry(row *sql.Row) (*ArtifactCacheEntry, error) {
	var e ArtifactCacheEntry
	err := row.Scan(
		&e.ID, &e.SourceID, &e.Version, &e.Checksum, &e.CachePath,
		&e.SizeBytes, &e.ContentType, &e.ResolvedURL,
		&e.CreatedAt, &e.LastUsedAt, &e.UseCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan artifact cache entry: %w", err)
	}
	return &e, nil
}
