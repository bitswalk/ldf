package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DownloadJobRepository handles download job database operations
type DownloadJobRepository struct {
	db *Database
}

// NewDownloadJobRepository creates a new download job repository
func NewDownloadJobRepository(db *Database) *DownloadJobRepository {
	return &DownloadJobRepository{db: db}
}

// Create inserts a new download job
func (r *DownloadJobRepository) Create(job *DownloadJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	job.CreatedAt = time.Now()
	if job.Status == "" {
		job.Status = JobStatusPending
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	// Serialize component_ids as JSON
	var componentIDsJSON []byte
	var err error
	if len(job.ComponentIDs) > 0 {
		componentIDsJSON, err = json.Marshal(job.ComponentIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal component_ids: %w", err)
		}
	}

	query := `
		INSERT INTO download_jobs (id, distribution_id, owner_id, component_id,
			source_id, source_name, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries, component_ids,
			priority, cache_hit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.DB().Exec(query,
		job.ID, job.DistributionID, job.OwnerID, job.ComponentID,
		job.SourceID, job.SourceName, job.SourceType, job.RetrievalMethod, job.ResolvedURL, job.Version, job.Status,
		job.ProgressBytes, job.TotalBytes, job.CreatedAt, job.StartedAt, job.CompletedAt,
		job.ArtifactPath, job.Checksum, job.ErrorMessage, job.RetryCount, job.MaxRetries,
		string(componentIDsJSON), job.Priority, job.CacheHit,
	)
	if err != nil {
		return fmt.Errorf("failed to create download job: %w", err)
	}

	return nil
}

// GetBySourceAndVersion retrieves an existing job for the same distribution, source, and version
// Returns nil if no matching job exists
func (r *DownloadJobRepository) GetBySourceAndVersion(distributionID, sourceID, version string) (*DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.distribution_id = ? AND dj.source_id = ? AND dj.version = ? LIMIT 1`
	row := r.db.DB().QueryRow(query, distributionID, sourceID, version)
	return r.scanJob(row)
}

// GetCompletedBySourceAndVersion finds any completed job across all distributions
// for the given source+version. Used for cross-distribution cache population.
func (r *DownloadJobRepository) GetCompletedBySourceAndVersion(sourceID, version string) (*DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.source_id = ? AND dj.version = ? AND dj.status = 'completed' LIMIT 1`
	row := r.db.DB().QueryRow(query, sourceID, version)
	return r.scanJob(row)
}

// selectJobsQuery is the base SELECT query with JOIN to get component name
const selectJobsQuery = `
	SELECT dj.id, dj.distribution_id, dj.owner_id, dj.component_id, c.name as component_name,
		dj.source_id, dj.source_name, dj.source_type, dj.retrieval_method, dj.resolved_url, dj.version, dj.status,
		dj.progress_bytes, dj.total_bytes, dj.created_at, dj.started_at, dj.completed_at,
		dj.artifact_path, dj.checksum, dj.error_message, dj.retry_count, dj.max_retries, dj.component_ids,
		dj.priority, dj.cache_hit
	FROM download_jobs dj
	LEFT JOIN components c ON dj.component_id = c.id
`

// GetByID retrieves a download job by ID
func (r *DownloadJobRepository) GetByID(id string) (*DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.id = ?`
	row := r.db.DB().QueryRow(query, id)
	return r.scanJob(row)
}

// ListByDistribution retrieves all download jobs for a distribution
func (r *DownloadJobRepository) ListByDistribution(distributionID string) ([]DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.distribution_id = ? ORDER BY dj.created_at DESC`
	rows, err := r.db.DB().Query(query, distributionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list download jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListByStatus retrieves all download jobs with a specific status
func (r *DownloadJobRepository) ListByStatus(status DownloadJobStatus) ([]DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.status = ? ORDER BY dj.created_at ASC`
	rows, err := r.db.DB().Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list download jobs by status: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListPending retrieves all pending download jobs ordered by priority (highest first), then creation time
func (r *DownloadJobRepository) ListPending() ([]DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.status = ? ORDER BY dj.priority DESC, dj.created_at ASC`
	rows, err := r.db.DB().Query(query, JobStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending download jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListActive retrieves all active (pending, verifying, downloading) download jobs
func (r *DownloadJobRepository) ListActive() ([]DownloadJob, error) {
	query := selectJobsQuery + ` WHERE dj.status IN (?, ?, ?) ORDER BY dj.created_at ASC`
	rows, err := r.db.DB().Query(query, JobStatusPending, JobStatusVerifying, JobStatusDownloading)
	if err != nil {
		return nil, fmt.Errorf("failed to list active download jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// UpdateStatus updates the status of a download job
func (r *DownloadJobRepository) UpdateStatus(id string, status DownloadJobStatus, errorMsg string) error {
	query := `UPDATE download_jobs SET status = ?, error_message = ? WHERE id = ?`
	result, err := r.db.DB().Exec(query, status, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// UpdateProgress updates the progress of a download job
func (r *DownloadJobRepository) UpdateProgress(id string, progressBytes, totalBytes int64) error {
	query := `UPDATE download_jobs SET progress_bytes = ?, total_bytes = ? WHERE id = ?`
	result, err := r.db.DB().Exec(query, progressBytes, totalBytes, id)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// MarkStarted marks a job as started
func (r *DownloadJobRepository) MarkStarted(id string) error {
	now := time.Now()
	query := `UPDATE download_jobs SET status = ?, started_at = ? WHERE id = ?`
	result, err := r.db.DB().Exec(query, JobStatusDownloading, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// MarkVerifying marks a job as verifying
func (r *DownloadJobRepository) MarkVerifying(id string) error {
	query := `UPDATE download_jobs SET status = ? WHERE id = ?`
	result, err := r.db.DB().Exec(query, JobStatusVerifying, id)
	if err != nil {
		return fmt.Errorf("failed to mark job verifying: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// MarkCompleted marks a job as completed with artifact path and checksum
func (r *DownloadJobRepository) MarkCompleted(id, artifactPath, checksum string) error {
	now := time.Now()
	query := `
		UPDATE download_jobs
		SET status = ?, completed_at = ?, artifact_path = ?, checksum = ?, error_message = ''
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, JobStatusCompleted, now, artifactPath, checksum, id)
	if err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// MarkFailed marks a job as failed with an error message
func (r *DownloadJobRepository) MarkFailed(id, errorMsg string) error {
	now := time.Now()
	query := `
		UPDATE download_jobs
		SET status = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, JobStatusFailed, now, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to mark job failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// MarkCancelled marks a job as cancelled
func (r *DownloadJobRepository) MarkCancelled(id string) error {
	now := time.Now()
	query := `
		UPDATE download_jobs
		SET status = ?, completed_at = ?, error_message = 'Cancelled by user'
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, JobStatusCancelled, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark job cancelled: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// IncrementRetry increments the retry count for a job and resets status to pending
func (r *DownloadJobRepository) IncrementRetry(id string) error {
	query := `
		UPDATE download_jobs
		SET retry_count = retry_count + 1, status = ?, error_message = ''
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, JobStatusPending, id)
	if err != nil {
		return fmt.Errorf("failed to increment retry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// Delete removes a download job by ID
func (r *DownloadJobRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM download_jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete download job: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("download job not found: %s", id)
	}

	return nil
}

// DeleteByDistribution removes all download jobs for a distribution
func (r *DownloadJobRepository) DeleteByDistribution(distributionID string) error {
	_, err := r.db.DB().Exec("DELETE FROM download_jobs WHERE distribution_id = ?", distributionID)
	if err != nil {
		return fmt.Errorf("failed to delete download jobs: %w", err)
	}
	return nil
}

// scanJob scans a single download job row
func (r *DownloadJobRepository) scanJob(row *sql.Row) (*DownloadJob, error) {
	var job DownloadJob
	var startedAt, completedAt sql.NullTime
	var artifactPath, checksum, errorMsg, componentName, sourceName, componentIDsJSON sql.NullString

	err := row.Scan(
		&job.ID, &job.DistributionID, &job.OwnerID, &job.ComponentID, &componentName,
		&job.SourceID, &sourceName, &job.SourceType, &job.RetrievalMethod, &job.ResolvedURL, &job.Version, &job.Status,
		&job.ProgressBytes, &job.TotalBytes, &job.CreatedAt, &startedAt, &completedAt,
		&artifactPath, &checksum, &errorMsg, &job.RetryCount, &job.MaxRetries, &componentIDsJSON,
		&job.Priority, &job.CacheHit,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan download job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.ArtifactPath = artifactPath.String
	job.Checksum = checksum.String
	job.ErrorMessage = errorMsg.String
	job.ComponentName = componentName.String
	job.SourceName = sourceName.String

	// Parse component_ids JSON
	if componentIDsJSON.Valid && componentIDsJSON.String != "" {
		if err := json.Unmarshal([]byte(componentIDsJSON.String), &job.ComponentIDs); err != nil {
			// Log error but don't fail - just leave ComponentIDs empty
			job.ComponentIDs = nil
		}
	}

	return &job, nil
}

// scanJobs scans multiple download job rows
func (r *DownloadJobRepository) scanJobs(rows *sql.Rows) ([]DownloadJob, error) {
	var jobs []DownloadJob

	for rows.Next() {
		var job DownloadJob
		var startedAt, completedAt sql.NullTime
		var artifactPath, checksum, errorMsg, componentName, sourceName, componentIDsJSON sql.NullString

		if err := rows.Scan(
			&job.ID, &job.DistributionID, &job.OwnerID, &job.ComponentID, &componentName,
			&job.SourceID, &sourceName, &job.SourceType, &job.RetrievalMethod, &job.ResolvedURL, &job.Version, &job.Status,
			&job.ProgressBytes, &job.TotalBytes, &job.CreatedAt, &startedAt, &completedAt,
			&artifactPath, &checksum, &errorMsg, &job.RetryCount, &job.MaxRetries, &componentIDsJSON,
			&job.Priority, &job.CacheHit,
		); err != nil {
			return nil, fmt.Errorf("failed to scan download job: %w", err)
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		job.ArtifactPath = artifactPath.String
		job.Checksum = checksum.String
		job.ErrorMessage = errorMsg.String
		job.ComponentName = componentName.String
		job.SourceName = sourceName.String

		// Parse component_ids JSON
		if componentIDsJSON.Valid && componentIDsJSON.String != "" {
			if err := json.Unmarshal([]byte(componentIDsJSON.String), &job.ComponentIDs); err != nil {
				// Log error but don't fail - just leave ComponentIDs empty
				job.ComponentIDs = nil
			}
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating download jobs: %w", err)
	}

	return jobs, nil
}

// AddComponentToJob adds a component ID to an existing job's component_ids list
func (r *DownloadJobRepository) AddComponentToJob(jobID, componentID string) error {
	// Get the current job
	job, err := r.GetByID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Check if component already in list
	for _, id := range job.ComponentIDs {
		if id == componentID {
			return nil // Already added
		}
	}

	// Add component to list
	job.ComponentIDs = append(job.ComponentIDs, componentID)

	// Serialize and update
	componentIDsJSON, err := json.Marshal(job.ComponentIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal component_ids: %w", err)
	}

	query := `UPDATE download_jobs SET component_ids = ? WHERE id = ?`
	_, err = r.db.DB().Exec(query, string(componentIDsJSON), jobID)
	if err != nil {
		return fmt.Errorf("failed to update component_ids: %w", err)
	}

	return nil
}
