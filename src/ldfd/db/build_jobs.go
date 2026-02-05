package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BuildJobRepository handles build job database operations
type BuildJobRepository struct {
	db *Database
}

// NewBuildJobRepository creates a new build job repository
func NewBuildJobRepository(db *Database) *BuildJobRepository {
	return &BuildJobRepository{db: db}
}

// Create inserts a new build job
func (r *BuildJobRepository) Create(job *BuildJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	job.CreatedAt = time.Now()
	if job.Status == "" {
		job.Status = BuildStatusPending
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 1
	}
	if job.TargetArch == "" {
		job.TargetArch = ArchX86_64
	}
	if job.ImageFormat == "" {
		job.ImageFormat = ImageFormatRaw
	}

	query := `
		INSERT INTO build_jobs (id, distribution_id, owner_id, status, current_stage,
			target_arch, image_format, progress_percent, workspace_path,
			artifact_path, artifact_checksum, artifact_size,
			error_message, error_stage, retry_count, max_retries,
			clear_cache, config_snapshot, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query,
		job.ID, job.DistributionID, job.OwnerID, job.Status, job.CurrentStage,
		job.TargetArch, job.ImageFormat, job.ProgressPercent, job.WorkspacePath,
		job.ArtifactPath, job.ArtifactChecksum, job.ArtifactSize,
		job.ErrorMessage, job.ErrorStage, job.RetryCount, job.MaxRetries,
		job.ClearCache, job.ConfigSnapshot, job.CreatedAt, job.StartedAt, job.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create build job: %w", err)
	}

	return nil
}

// selectBuildJobsQuery is the base SELECT query for build jobs
const selectBuildJobsQuery = `
	SELECT id, distribution_id, owner_id, status, current_stage,
		target_arch, image_format, progress_percent, workspace_path,
		artifact_path, artifact_checksum, artifact_size,
		error_message, error_stage, retry_count, max_retries,
		clear_cache, config_snapshot, created_at, started_at, completed_at
	FROM build_jobs
`

// GetByID retrieves a build job by ID
func (r *BuildJobRepository) GetByID(id string) (*BuildJob, error) {
	query := selectBuildJobsQuery + ` WHERE id = ?`
	row := r.db.DB().QueryRow(query, id)
	return r.scanJob(row)
}

// ListByDistribution retrieves all build jobs for a distribution
func (r *BuildJobRepository) ListByDistribution(distributionID string) ([]BuildJob, error) {
	query := selectBuildJobsQuery + ` WHERE distribution_id = ? ORDER BY created_at DESC`
	rows, err := r.db.DB().Query(query, distributionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list build jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListByStatus retrieves all build jobs with a specific status
func (r *BuildJobRepository) ListByStatus(status BuildJobStatus) ([]BuildJob, error) {
	query := selectBuildJobsQuery + ` WHERE status = ? ORDER BY created_at ASC`
	rows, err := r.db.DB().Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list build jobs by status: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// ListPending retrieves all pending build jobs
func (r *BuildJobRepository) ListPending() ([]BuildJob, error) {
	return r.ListByStatus(BuildStatusPending)
}

// ListActive retrieves all active build jobs
func (r *BuildJobRepository) ListActive() ([]BuildJob, error) {
	query := selectBuildJobsQuery + ` WHERE status IN (?, ?, ?, ?, ?, ?) ORDER BY created_at ASC`
	rows, err := r.db.DB().Query(query,
		BuildStatusPending, BuildStatusResolving, BuildStatusPreparing,
		BuildStatusCompiling, BuildStatusAssembling, BuildStatusPackaging,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list active build jobs: %w", err)
	}
	defer rows.Close()

	return r.scanJobs(rows)
}

// Delete removes a build job by ID
func (r *BuildJobRepository) Delete(id string) error {
	result, err := r.db.DB().Exec("DELETE FROM build_jobs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete build job: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// DeleteByDistribution removes all build jobs for a distribution
func (r *BuildJobRepository) DeleteByDistribution(distributionID string) error {
	_, err := r.db.DB().Exec("DELETE FROM build_jobs WHERE distribution_id = ?", distributionID)
	if err != nil {
		return fmt.Errorf("failed to delete build jobs: %w", err)
	}
	return nil
}

// MarkStarted marks a build job as started
func (r *BuildJobRepository) MarkStarted(id string) error {
	now := time.Now()
	query := `UPDATE build_jobs SET status = ?, started_at = ? WHERE id = ?`
	result, err := r.db.DB().Exec(query, BuildStatusResolving, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark build started: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// UpdateStage updates the current stage and progress of a build job
func (r *BuildJobRepository) UpdateStage(id string, stage string, progressPercent int) error {
	query := `UPDATE build_jobs SET current_stage = ?, progress_percent = ?, status = ? WHERE id = ?`

	// Map stage names to build status
	status := BuildStatusPending
	switch BuildStageName(stage) {
	case StageResolve:
		status = BuildStatusResolving
	case StageDownload, StagePrepare:
		status = BuildStatusPreparing
	case StageCompile:
		status = BuildStatusCompiling
	case StageAssemble:
		status = BuildStatusAssembling
	case StagePackage:
		status = BuildStatusPackaging
	}

	result, err := r.db.DB().Exec(query, stage, progressPercent, status, id)
	if err != nil {
		return fmt.Errorf("failed to update build stage: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// MarkCompleted marks a build job as completed
func (r *BuildJobRepository) MarkCompleted(id, artifactPath, checksum string, size int64) error {
	now := time.Now()
	query := `
		UPDATE build_jobs
		SET status = ?, completed_at = ?, artifact_path = ?, artifact_checksum = ?,
			artifact_size = ?, progress_percent = 100, error_message = ''
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, BuildStatusCompleted, now, artifactPath, checksum, size, id)
	if err != nil {
		return fmt.Errorf("failed to mark build completed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// MarkFailed marks a build job as failed
func (r *BuildJobRepository) MarkFailed(id, errorMsg, errorStage string) error {
	now := time.Now()
	query := `
		UPDATE build_jobs
		SET status = ?, completed_at = ?, error_message = ?, error_stage = ?
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, BuildStatusFailed, now, errorMsg, errorStage, id)
	if err != nil {
		return fmt.Errorf("failed to mark build failed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// MarkCancelled marks a build job as cancelled
func (r *BuildJobRepository) MarkCancelled(id string) error {
	now := time.Now()
	query := `
		UPDATE build_jobs
		SET status = ?, completed_at = ?, error_message = 'Cancelled by user'
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, BuildStatusCancelled, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark build cancelled: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// IncrementRetry increments the retry count and resets status to pending
func (r *BuildJobRepository) IncrementRetry(id string) error {
	query := `
		UPDATE build_jobs
		SET retry_count = retry_count + 1, status = ?, error_message = '',
			error_stage = '', current_stage = '', progress_percent = 0
		WHERE id = ?
	`
	result, err := r.db.DB().Exec(query, BuildStatusPending, id)
	if err != nil {
		return fmt.Errorf("failed to increment retry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("build job not found: %s", id)
	}

	return nil
}

// CreateStage inserts a new build stage record
func (r *BuildJobRepository) CreateStage(stage *BuildStage) error {
	query := `
		INSERT INTO build_stages (build_id, name, status, progress_percent,
			started_at, completed_at, duration_ms, error_message, log_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.DB().Exec(query,
		stage.BuildID, stage.Name, stage.Status, stage.ProgressPercent,
		stage.StartedAt, stage.CompletedAt, stage.DurationMs, stage.ErrorMessage, stage.LogPath,
	)
	if err != nil {
		return fmt.Errorf("failed to create build stage: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	stage.ID = id

	return nil
}

// UpdateStageStatus updates the status of a build stage
func (r *BuildJobRepository) UpdateStageStatus(buildID string, stageName BuildStageName, status string) error {
	now := time.Now()
	query := `UPDATE build_stages SET status = ?, started_at = COALESCE(started_at, ?) WHERE build_id = ? AND name = ?`
	_, err := r.db.DB().Exec(query, status, now, buildID, stageName)
	if err != nil {
		return fmt.Errorf("failed to update stage status: %w", err)
	}
	return nil
}

// MarkStageCompleted marks a build stage as completed
func (r *BuildJobRepository) MarkStageCompleted(buildID string, stageName BuildStageName, durationMs int64) error {
	now := time.Now()
	query := `
		UPDATE build_stages
		SET status = 'completed', completed_at = ?, duration_ms = ?, progress_percent = 100
		WHERE build_id = ? AND name = ?
	`
	_, err := r.db.DB().Exec(query, now, durationMs, buildID, stageName)
	if err != nil {
		return fmt.Errorf("failed to mark stage completed: %w", err)
	}
	return nil
}

// MarkStageFailed marks a build stage as failed
func (r *BuildJobRepository) MarkStageFailed(buildID string, stageName BuildStageName, errMsg string) error {
	now := time.Now()
	query := `
		UPDATE build_stages
		SET status = 'failed', completed_at = ?, error_message = ?
		WHERE build_id = ? AND name = ?
	`
	_, err := r.db.DB().Exec(query, now, errMsg, buildID, stageName)
	if err != nil {
		return fmt.Errorf("failed to mark stage failed: %w", err)
	}
	return nil
}

// GetStages retrieves all stages for a build job
func (r *BuildJobRepository) GetStages(buildID string) ([]BuildStage, error) {
	query := `
		SELECT id, build_id, name, status, progress_percent,
			started_at, completed_at, duration_ms, error_message, log_path
		FROM build_stages
		WHERE build_id = ?
		ORDER BY id ASC
	`
	rows, err := r.db.DB().Query(query, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get build stages: %w", err)
	}
	defer rows.Close()

	var stages []BuildStage
	for rows.Next() {
		var stage BuildStage
		var startedAt, completedAt sql.NullTime
		var errorMsg, logPath sql.NullString

		if err := rows.Scan(
			&stage.ID, &stage.BuildID, &stage.Name, &stage.Status, &stage.ProgressPercent,
			&startedAt, &completedAt, &stage.DurationMs, &errorMsg, &logPath,
		); err != nil {
			return nil, fmt.Errorf("failed to scan build stage: %w", err)
		}

		if startedAt.Valid {
			stage.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			stage.CompletedAt = &completedAt.Time
		}
		stage.ErrorMessage = errorMsg.String
		stage.LogPath = logPath.String

		stages = append(stages, stage)
	}

	return stages, rows.Err()
}

// AppendLog appends a log entry for a build
func (r *BuildJobRepository) AppendLog(buildID, stage, level, message string) error {
	query := `
		INSERT INTO build_logs (build_id, stage, level, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.DB().Exec(query, buildID, stage, level, message, time.Now())
	if err != nil {
		return fmt.Errorf("failed to append build log: %w", err)
	}
	return nil
}

// GetLogs retrieves log entries for a build job
func (r *BuildJobRepository) GetLogs(buildID string, limit, offset int) ([]BuildLog, error) {
	query := `
		SELECT id, build_id, stage, level, message, created_at
		FROM build_logs
		WHERE build_id = ?
		ORDER BY id ASC
		LIMIT ? OFFSET ?
	`
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.db.DB().Query(query, buildID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get build logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetLogsByStage retrieves log entries for a specific stage
func (r *BuildJobRepository) GetLogsByStage(buildID, stage string) ([]BuildLog, error) {
	query := `
		SELECT id, build_id, stage, level, message, created_at
		FROM build_logs
		WHERE build_id = ? AND stage = ?
		ORDER BY id ASC
	`
	rows, err := r.db.DB().Query(query, buildID, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to get build logs by stage: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetLogsSince retrieves log entries after a given ID (for streaming)
func (r *BuildJobRepository) GetLogsSince(buildID string, afterID int64) ([]BuildLog, error) {
	query := `
		SELECT id, build_id, stage, level, message, created_at
		FROM build_logs
		WHERE build_id = ? AND id > ?
		ORDER BY id ASC
	`
	rows, err := r.db.DB().Query(query, buildID, afterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get build logs since: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// scanJob scans a single build job row
func (r *BuildJobRepository) scanJob(row *sql.Row) (*BuildJob, error) {
	var job BuildJob
	var startedAt, completedAt sql.NullTime
	var workspacePath, artifactPath, artifactChecksum sql.NullString
	var errorMsg, errorStage, configSnapshot, currentStage sql.NullString

	err := row.Scan(
		&job.ID, &job.DistributionID, &job.OwnerID, &job.Status, &currentStage,
		&job.TargetArch, &job.ImageFormat, &job.ProgressPercent, &workspacePath,
		&artifactPath, &artifactChecksum, &job.ArtifactSize,
		&errorMsg, &errorStage, &job.RetryCount, &job.MaxRetries,
		&job.ClearCache, &configSnapshot, &job.CreatedAt, &startedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan build job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	job.CurrentStage = currentStage.String
	job.WorkspacePath = workspacePath.String
	job.ArtifactPath = artifactPath.String
	job.ArtifactChecksum = artifactChecksum.String
	job.ErrorMessage = errorMsg.String
	job.ErrorStage = errorStage.String
	job.ConfigSnapshot = configSnapshot.String

	return &job, nil
}

// scanJobs scans multiple build job rows
func (r *BuildJobRepository) scanJobs(rows *sql.Rows) ([]BuildJob, error) {
	var jobs []BuildJob

	for rows.Next() {
		var job BuildJob
		var startedAt, completedAt sql.NullTime
		var workspacePath, artifactPath, artifactChecksum sql.NullString
		var errorMsg, errorStage, configSnapshot, currentStage sql.NullString

		if err := rows.Scan(
			&job.ID, &job.DistributionID, &job.OwnerID, &job.Status, &currentStage,
			&job.TargetArch, &job.ImageFormat, &job.ProgressPercent, &workspacePath,
			&artifactPath, &artifactChecksum, &job.ArtifactSize,
			&errorMsg, &errorStage, &job.RetryCount, &job.MaxRetries,
			&job.ClearCache, &configSnapshot, &job.CreatedAt, &startedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan build job: %w", err)
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		job.CurrentStage = currentStage.String
		job.WorkspacePath = workspacePath.String
		job.ArtifactPath = artifactPath.String
		job.ArtifactChecksum = artifactChecksum.String
		job.ErrorMessage = errorMsg.String
		job.ErrorStage = errorStage.String
		job.ConfigSnapshot = configSnapshot.String

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating build jobs: %w", err)
	}

	return jobs, nil
}

// scanLogs scans multiple build log rows
func (r *BuildJobRepository) scanLogs(rows *sql.Rows) ([]BuildLog, error) {
	var logs []BuildLog

	for rows.Next() {
		var entry BuildLog
		if err := rows.Scan(
			&entry.ID, &entry.BuildID, &entry.Stage, &entry.Level, &entry.Message, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan build log: %w", err)
		}
		logs = append(logs, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating build logs: %w", err)
	}

	return logs, nil
}
