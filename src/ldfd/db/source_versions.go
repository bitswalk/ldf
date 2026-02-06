package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SourceVersionRepository handles source version database operations
type SourceVersionRepository struct {
	db *Database
}

// NewSourceVersionRepository creates a new source version repository
func NewSourceVersionRepository(db *Database) *SourceVersionRepository {
	return &SourceVersionRepository{db: db}
}

// ListBySource retrieves all versions for a specific source ordered by version descending
func (r *SourceVersionRepository) ListBySource(sourceID, sourceType string) ([]SourceVersion, error) {
	query := `
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE source_id = ? AND source_type = ?
		ORDER BY discovered_at DESC, version DESC
	`
	rows, err := r.db.DB().Query(query, sourceID, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to list source versions: %w", err)
	}
	defer rows.Close()

	return r.scanVersions(rows)
}

// ListBySourceStable retrieves only stable versions for a specific source
func (r *SourceVersionRepository) ListBySourceStable(sourceID, sourceType string) ([]SourceVersion, error) {
	query := `
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE source_id = ? AND source_type = ? AND is_stable = 1
		ORDER BY discovered_at DESC, version DESC
	`
	rows, err := r.db.DB().Query(query, sourceID, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to list stable source versions: %w", err)
	}
	defer rows.Close()

	return r.scanVersions(rows)
}

// ListBySourcePaginated retrieves versions with pagination and optional version type filter
func (r *SourceVersionRepository) ListBySourcePaginated(sourceID, sourceType string, limit, offset int, versionTypeFilter string) ([]SourceVersion, int, error) {
	// Build where clause
	whereClause := "source_id = ? AND source_type = ?"
	args := []interface{}{sourceID, sourceType}

	// Filter by version type if specified
	if versionTypeFilter != "" && versionTypeFilter != "all" {
		whereClause += " AND version_type = ?"
		args = append(args, versionTypeFilter)
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM source_versions WHERE %s", whereClause)
	if err := r.db.DB().QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count source versions: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE %s
		ORDER BY discovered_at DESC, version DESC
		LIMIT ? OFFSET ?
	`, whereClause)
	queryArgs := append(args, limit, offset)
	rows, err := r.db.DB().Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list source versions paginated: %w", err)
	}
	defer rows.Close()

	versions, err := r.scanVersions(rows)
	if err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

// GetByID retrieves a source version by ID
func (r *SourceVersionRepository) GetByID(id string) (*SourceVersion, error) {
	query := `
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	v, err := r.scanVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source version: %w", err)
	}

	return v, nil
}

// GetByVersion retrieves a specific version for a source
func (r *SourceVersionRepository) GetByVersion(sourceID, sourceType, version string) (*SourceVersion, error) {
	query := `
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE source_id = ? AND source_type = ? AND version = ?
	`
	row := r.db.DB().QueryRow(query, sourceID, sourceType, version)

	v, err := r.scanVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source version: %w", err)
	}

	return v, nil
}

// GetLatestStable retrieves the most recent stable version for a source
func (r *SourceVersionRepository) GetLatestStable(sourceID, sourceType string) (*SourceVersion, error) {
	query := `
		SELECT id, source_id, source_type, version, version_type, release_date, download_url,
		       checksum, checksum_type, file_size, is_stable, discovered_at
		FROM source_versions
		WHERE source_id = ? AND source_type = ? AND is_stable = 1
		ORDER BY discovered_at DESC, version DESC
		LIMIT 1
	`
	row := r.db.DB().QueryRow(query, sourceID, sourceType)

	v, err := r.scanVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest stable version: %w", err)
	}

	return v, nil
}

// Create inserts a new source version
func (r *SourceVersionRepository) Create(v *SourceVersion) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	if v.DiscoveredAt.IsZero() {
		v.DiscoveredAt = time.Now().UTC()
	}
	if v.VersionType == "" {
		v.VersionType = VersionTypeStable
	}

	query := `
		INSERT INTO source_versions (id, source_id, source_type, version, version_type, release_date, download_url,
		                             checksum, checksum_type, file_size, is_stable, discovered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, v.ID, v.SourceID, v.SourceType, v.Version, v.VersionType,
		nullTime(v.ReleaseDate), nullString(v.DownloadURL), nullString(v.Checksum),
		nullString(v.ChecksumType), v.FileSize, v.IsStable, v.DiscoveredAt)
	if err != nil {
		return fmt.Errorf("failed to create source version: %w", err)
	}

	return nil
}

// BulkUpsert inserts or updates multiple versions, returns count of new versions
func (r *SourceVersionRepository) BulkUpsert(versions []SourceVersion) (int, error) {
	if len(versions) == 0 {
		return 0, nil
	}

	tx, err := r.db.DB().Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare the upsert statement
	stmt, err := tx.Prepare(`
		INSERT INTO source_versions (id, source_id, source_type, version, version_type, release_date, download_url,
		                             checksum, checksum_type, file_size, is_stable, discovered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_id, source_type, version) DO UPDATE SET
			version_type = excluded.version_type,
			release_date = excluded.release_date,
			download_url = excluded.download_url,
			checksum = excluded.checksum,
			checksum_type = excluded.checksum_type,
			file_size = excluded.file_size,
			is_stable = excluded.is_stable
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare upsert statement: %w", err)
	}
	defer stmt.Close()

	newCount := 0
	now := time.Now().UTC()

	for _, v := range versions {
		// Check if version exists
		exists, err := r.versionExistsInTx(tx, v.SourceID, v.SourceType, v.Version)
		if err != nil {
			return 0, fmt.Errorf("failed to check version existence: %w", err)
		}
		if !exists {
			newCount++
		}

		if v.ID == "" {
			v.ID = uuid.New().String()
		}
		if v.DiscoveredAt.IsZero() {
			v.DiscoveredAt = now
		}
		if v.VersionType == "" {
			v.VersionType = VersionTypeStable
		}

		_, err = stmt.Exec(v.ID, v.SourceID, v.SourceType, v.Version, v.VersionType,
			nullTime(v.ReleaseDate), nullString(v.DownloadURL), nullString(v.Checksum),
			nullString(v.ChecksumType), v.FileSize, v.IsStable, v.DiscoveredAt)
		if err != nil {
			return 0, fmt.Errorf("failed to upsert version %s: %w", v.Version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newCount, nil
}

// DeleteBySource removes all versions for a source
func (r *SourceVersionRepository) DeleteBySource(sourceID, sourceType string) error {
	_, err := r.db.DB().Exec(
		"DELETE FROM source_versions WHERE source_id = ? AND source_type = ?",
		sourceID, sourceType,
	)
	if err != nil {
		return fmt.Errorf("failed to delete source versions: %w", err)
	}
	return nil
}

// CountBySource returns the number of versions for a source
func (r *SourceVersionRepository) CountBySource(sourceID, sourceType string) (int, error) {
	var count int
	err := r.db.DB().QueryRow(
		"SELECT COUNT(*) FROM source_versions WHERE source_id = ? AND source_type = ?",
		sourceID, sourceType,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count source versions: %w", err)
	}
	return count, nil
}

// Version Sync Job operations

// CreateSyncJob creates a new sync job
func (r *SourceVersionRepository) CreateSyncJob(job *VersionSyncJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.Status == "" {
		job.Status = SyncStatusPending
	}
	job.CreatedAt = time.Now().UTC()

	query := `
		INSERT INTO version_sync_jobs (id, source_id, source_type, status, versions_found, versions_new,
		                               started_at, completed_at, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, job.ID, job.SourceID, job.SourceType, job.Status,
		job.VersionsFound, job.VersionsNew, nullTime(job.StartedAt), nullTime(job.CompletedAt),
		nullString(job.ErrorMessage), job.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	return nil
}

// GetSyncJob retrieves a sync job by ID
func (r *SourceVersionRepository) GetSyncJob(id string) (*VersionSyncJob, error) {
	query := `
		SELECT id, source_id, source_type, status, versions_found, versions_new,
		       started_at, completed_at, error_message, created_at
		FROM version_sync_jobs
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)

	job, err := r.scanSyncJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync job: %w", err)
	}

	return job, nil
}

// GetLatestSyncJob retrieves the most recent sync job for a source
func (r *SourceVersionRepository) GetLatestSyncJob(sourceID, sourceType string) (*VersionSyncJob, error) {
	query := `
		SELECT id, source_id, source_type, status, versions_found, versions_new,
		       started_at, completed_at, error_message, created_at
		FROM version_sync_jobs
		WHERE source_id = ? AND source_type = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.DB().QueryRow(query, sourceID, sourceType)

	job, err := r.scanSyncJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest sync job: %w", err)
	}

	return job, nil
}

// GetRunningSyncJob retrieves any currently running sync job for a source
func (r *SourceVersionRepository) GetRunningSyncJob(sourceID, sourceType string) (*VersionSyncJob, error) {
	query := `
		SELECT id, source_id, source_type, status, versions_found, versions_new,
		       started_at, completed_at, error_message, created_at
		FROM version_sync_jobs
		WHERE source_id = ? AND source_type = ? AND status IN ('pending', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.DB().QueryRow(query, sourceID, sourceType)

	job, err := r.scanSyncJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get running sync job: %w", err)
	}

	return job, nil
}

// UpdateSyncJob updates an existing sync job
func (r *SourceVersionRepository) UpdateSyncJob(job *VersionSyncJob) error {
	query := `
		UPDATE version_sync_jobs
		SET status = ?, versions_found = ?, versions_new = ?,
		    started_at = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, job.Status, job.VersionsFound, job.VersionsNew,
		nullTime(job.StartedAt), nullTime(job.CompletedAt), nullString(job.ErrorMessage), job.ID)
	if err != nil {
		return fmt.Errorf("failed to update sync job: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("sync job not found: %s", job.ID)
	}

	return nil
}

// MarkSyncJobRunning marks a sync job as running
func (r *SourceVersionRepository) MarkSyncJobRunning(id string) error {
	now := time.Now().UTC()
	query := `UPDATE version_sync_jobs SET status = ?, started_at = ? WHERE id = ?`
	_, err := r.db.DB().Exec(query, SyncStatusRunning, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark sync job running: %w", err)
	}
	return nil
}

// MarkSyncJobCompleted marks a sync job as completed
func (r *SourceVersionRepository) MarkSyncJobCompleted(id string, versionsFound, versionsNew int) error {
	now := time.Now().UTC()
	query := `UPDATE version_sync_jobs SET status = ?, completed_at = ?, versions_found = ?, versions_new = ? WHERE id = ?`
	_, err := r.db.DB().Exec(query, SyncStatusCompleted, now, versionsFound, versionsNew, id)
	if err != nil {
		return fmt.Errorf("failed to mark sync job completed: %w", err)
	}
	return nil
}

// MarkSyncJobFailed marks a sync job as failed
func (r *SourceVersionRepository) MarkSyncJobFailed(id, errorMessage string) error {
	now := time.Now().UTC()
	query := `UPDATE version_sync_jobs SET status = ?, completed_at = ?, error_message = ? WHERE id = ?`
	_, err := r.db.DB().Exec(query, SyncStatusFailed, now, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to mark sync job failed: %w", err)
	}
	return nil
}

// ListSyncJobsBySource retrieves all sync jobs for a source
func (r *SourceVersionRepository) ListSyncJobsBySource(sourceID, sourceType string, limit int) ([]VersionSyncJob, error) {
	query := `
		SELECT id, source_id, source_type, status, versions_found, versions_new,
		       started_at, completed_at, error_message, created_at
		FROM version_sync_jobs
		WHERE source_id = ? AND source_type = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := r.db.DB().Query(query, sourceID, sourceType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync jobs: %w", err)
	}
	defer rows.Close()

	var jobs []VersionSyncJob
	for rows.Next() {
		job, err := r.scanSyncJobRow(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sync jobs: %w", err)
	}

	return jobs, nil
}

// Helper functions

func (r *SourceVersionRepository) versionExistsInTx(tx *sql.Tx, sourceID, sourceType, version string) (bool, error) {
	var exists bool
	err := tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM source_versions WHERE source_id = ? AND source_type = ? AND version = ?)",
		sourceID, sourceType, version,
	).Scan(&exists)
	return exists, err
}

func (r *SourceVersionRepository) scanVersions(rows *sql.Rows) ([]SourceVersion, error) {
	var versions []SourceVersion
	for rows.Next() {
		var v SourceVersion
		var releaseDate sql.NullTime
		var downloadURL, checksum, checksumType, versionType sql.NullString

		if err := rows.Scan(&v.ID, &v.SourceID, &v.SourceType, &v.Version, &versionType, &releaseDate,
			&downloadURL, &checksum, &checksumType, &v.FileSize, &v.IsStable, &v.DiscoveredAt); err != nil {
			return nil, fmt.Errorf("failed to scan source version: %w", err)
		}

		if releaseDate.Valid {
			v.ReleaseDate = &releaseDate.Time
		}
		v.DownloadURL = downloadURL.String
		v.Checksum = checksum.String
		v.ChecksumType = checksumType.String
		v.VersionType = VersionType(versionType.String)
		if v.VersionType == "" {
			v.VersionType = VersionTypeStable
		}

		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating source versions: %w", err)
	}

	return versions, nil
}

func (r *SourceVersionRepository) scanVersion(row *sql.Row) (*SourceVersion, error) {
	var v SourceVersion
	var releaseDate sql.NullTime
	var downloadURL, checksum, checksumType, versionType sql.NullString

	err := row.Scan(&v.ID, &v.SourceID, &v.SourceType, &v.Version, &versionType, &releaseDate,
		&downloadURL, &checksum, &checksumType, &v.FileSize, &v.IsStable, &v.DiscoveredAt)
	if err != nil {
		return nil, err
	}

	if releaseDate.Valid {
		v.ReleaseDate = &releaseDate.Time
	}
	v.DownloadURL = downloadURL.String
	v.Checksum = checksum.String
	v.ChecksumType = checksumType.String
	v.VersionType = VersionType(versionType.String)
	if v.VersionType == "" {
		v.VersionType = VersionTypeStable
	}

	return &v, nil
}

func (r *SourceVersionRepository) scanSyncJob(row *sql.Row) (*VersionSyncJob, error) {
	var job VersionSyncJob
	var startedAt, completedAt sql.NullTime
	var errorMessage sql.NullString

	err := row.Scan(&job.ID, &job.SourceID, &job.SourceType, &job.Status,
		&job.VersionsFound, &job.VersionsNew, &startedAt, &completedAt, &errorMessage, &job.CreatedAt)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.ErrorMessage = errorMessage.String

	return &job, nil
}

func (r *SourceVersionRepository) scanSyncJobRow(rows *sql.Rows) (*VersionSyncJob, error) {
	var job VersionSyncJob
	var startedAt, completedAt sql.NullTime
	var errorMessage sql.NullString

	err := rows.Scan(&job.ID, &job.SourceID, &job.SourceType, &job.Status,
		&job.VersionsFound, &job.VersionsNew, &startedAt, &completedAt, &errorMessage, &job.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan sync job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.ErrorMessage = errorMessage.String

	return &job, nil
}

// nullTime returns sql.NullTime for nil time pointers
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// GetSourceType returns "default" for system sources and "user" for user sources
// This helper is useful for deriving the source_type from an UpstreamSource
func GetSourceType(source *UpstreamSource) string {
	if source == nil {
		return ""
	}
	if source.IsSystem {
		return "default"
	}
	return "user"
}

// ListByComponentPaginated retrieves versions for all sources linked to a component
// Returns deduplicated versions ordered by discovered_at desc, version desc
func (r *SourceVersionRepository) ListByComponentPaginated(componentID string, limit, offset int, versionTypeFilter string) ([]SourceVersion, int, error) {
	// Build the query to get versions from sources linked to this component
	// We use a subquery to find source IDs that contain this component
	baseWhere := `
		EXISTS (
			SELECT 1 FROM upstream_sources us
			WHERE (us.id = sv.source_id)
			AND EXISTS (SELECT 1 FROM json_each(us.component_ids) WHERE value = ?)
		)
	`
	args := []interface{}{componentID}

	// Add version type filter if specified
	if versionTypeFilter != "" && versionTypeFilter != "all" {
		baseWhere += " AND sv.version_type = ?"
		args = append(args, versionTypeFilter)
	}

	// Get total count (distinct by version to avoid duplicates from multiple sources)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT sv.version)
		FROM source_versions sv
		WHERE %s
	`, baseWhere)
	var total int
	if err := r.db.DB().QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count component versions: %w", err)
	}

	// Get paginated results, deduplicated by version (keep the one with latest discovered_at)
	query := fmt.Sprintf(`
		SELECT sv.id, sv.source_id, sv.source_type, sv.version, sv.version_type, sv.release_date,
		       sv.download_url, sv.checksum, sv.checksum_type, sv.file_size, sv.is_stable, sv.discovered_at
		FROM source_versions sv
		WHERE %s
		GROUP BY sv.version
		ORDER BY sv.discovered_at DESC, sv.version DESC
		LIMIT ? OFFSET ?
	`, baseWhere)
	queryArgs := append(args, limit, offset)

	rows, err := r.db.DB().Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list component versions: %w", err)
	}
	defer rows.Close()

	versions, err := r.scanVersions(rows)
	if err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

// GetLatestByComponentAndType retrieves the most recent version of a specific type for a component
func (r *SourceVersionRepository) GetLatestByComponentAndType(componentID string, versionType VersionType) (*SourceVersion, error) {
	query := `
		SELECT sv.id, sv.source_id, sv.source_type, sv.version, sv.version_type, sv.release_date,
		       sv.download_url, sv.checksum, sv.checksum_type, sv.file_size, sv.is_stable, sv.discovered_at
		FROM source_versions sv
		WHERE EXISTS (
			SELECT 1 FROM upstream_sources us
			WHERE us.id = sv.source_id
			AND EXISTS (SELECT 1 FROM json_each(us.component_ids) WHERE value = ?)
		)
		AND sv.version_type = ?
		ORDER BY sv.discovered_at DESC, sv.version DESC
		LIMIT 1
	`
	row := r.db.DB().QueryRow(query, componentID, string(versionType))

	v, err := r.scanVersion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version by type: %w", err)
	}

	return v, nil
}

// GetLatestStableByComponent retrieves the most recent stable version for a component
func (r *SourceVersionRepository) GetLatestStableByComponent(componentID string) (*SourceVersion, error) {
	return r.GetLatestByComponentAndType(componentID, VersionTypeStable)
}

// GetLatestLongtermByComponent retrieves the most recent longterm version for a component
func (r *SourceVersionRepository) GetLatestLongtermByComponent(componentID string) (*SourceVersion, error) {
	return r.GetLatestByComponentAndType(componentID, VersionTypeLongterm)
}

// GetDistinctVersionTypes retrieves all distinct version types for a source
func (r *SourceVersionRepository) GetDistinctVersionTypes(sourceID, sourceType string) ([]string, error) {
	query := `
		SELECT DISTINCT version_type
		FROM source_versions
		WHERE source_id = ? AND source_type = ? AND version_type != ''
		ORDER BY version_type ASC
	`
	rows, err := r.db.DB().Query(query, sourceID, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct version types: %w", err)
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var vt string
		if err := rows.Scan(&vt); err != nil {
			return nil, fmt.Errorf("failed to scan version type: %w", err)
		}
		types = append(types, vt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating version types: %w", err)
	}

	return types, nil
}
