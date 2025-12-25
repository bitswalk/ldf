package db

import (
	"database/sql"
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

	query := `
		INSERT INTO download_jobs (id, distribution_id, owner_id, component_id, component_name,
			source_id, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query,
		job.ID, job.DistributionID, job.OwnerID, job.ComponentID, job.ComponentName,
		job.SourceID, job.SourceType, job.RetrievalMethod, job.ResolvedURL, job.Version, job.Status,
		job.ProgressBytes, job.TotalBytes, job.CreatedAt, job.StartedAt, job.CompletedAt,
		job.ArtifactPath, job.Checksum, job.ErrorMessage, job.RetryCount, job.MaxRetries,
	)
	if err != nil {
		return fmt.Errorf("failed to create download job: %w", err)
	}

	return nil
}

// GetByID retrieves a download job by ID
func (r *DownloadJobRepository) GetByID(id string) (*DownloadJob, error) {
	query := `
		SELECT id, distribution_id, owner_id, component_id, component_name,
			source_id, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries
		FROM download_jobs
		WHERE id = ?
	`
	row := r.db.DB().QueryRow(query, id)
	return r.scanJob(row)
}

// ListByDistribution retrieves all download jobs for a distribution
func (r *DownloadJobRepository) ListByDistribution(distributionID string) ([]DownloadJob, error) {
	query := `
		SELECT id, distribution_id, owner_id, component_id, component_name,
			source_id, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries
		FROM download_jobs
		WHERE distribution_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.DB().Query(query, distributionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list download jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListByStatus retrieves all download jobs with a specific status
func (r *DownloadJobRepository) ListByStatus(status DownloadJobStatus) ([]DownloadJob, error) {
	query := `
		SELECT id, distribution_id, owner_id, component_id, component_name,
			source_id, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries
		FROM download_jobs
		WHERE status = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.DB().Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list download jobs by status: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListPending retrieves all pending download jobs ordered by creation time
func (r *DownloadJobRepository) ListPending() ([]DownloadJob, error) {
	return r.ListByStatus(JobStatusPending)
}

// ListActive retrieves all active (pending, verifying, downloading) download jobs
func (r *DownloadJobRepository) ListActive() ([]DownloadJob, error) {
	query := `
		SELECT id, distribution_id, owner_id, component_id, component_name,
			source_id, source_type, retrieval_method, resolved_url, version, status,
			progress_bytes, total_bytes, created_at, started_at, completed_at,
			artifact_path, checksum, error_message, retry_count, max_retries
		FROM download_jobs
		WHERE status IN (?, ?, ?)
		ORDER BY created_at ASC
	`
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
	var artifactPath, checksum, errorMsg sql.NullString

	err := row.Scan(
		&job.ID, &job.DistributionID, &job.OwnerID, &job.ComponentID, &job.ComponentName,
		&job.SourceID, &job.SourceType, &job.RetrievalMethod, &job.ResolvedURL, &job.Version, &job.Status,
		&job.ProgressBytes, &job.TotalBytes, &job.CreatedAt, &startedAt, &completedAt,
		&artifactPath, &checksum, &errorMsg, &job.RetryCount, &job.MaxRetries,
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

	return &job, nil
}

// scanJobs scans multiple download job rows
func (r *DownloadJobRepository) scanJobs(rows *sql.Rows) ([]DownloadJob, error) {
	var jobs []DownloadJob

	for rows.Next() {
		var job DownloadJob
		var startedAt, completedAt sql.NullTime
		var artifactPath, checksum, errorMsg sql.NullString

		if err := rows.Scan(
			&job.ID, &job.DistributionID, &job.OwnerID, &job.ComponentID, &job.ComponentName,
			&job.SourceID, &job.SourceType, &job.RetrievalMethod, &job.ResolvedURL, &job.Version, &job.Status,
			&job.ProgressBytes, &job.TotalBytes, &job.CreatedAt, &startedAt, &completedAt,
			&artifactPath, &checksum, &errorMsg, &job.RetryCount, &job.MaxRetries,
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

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating download jobs: %w", err)
	}

	return jobs, nil
}
