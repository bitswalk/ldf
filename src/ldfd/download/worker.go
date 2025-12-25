package download

import (
	"context"
	"fmt"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Worker processes download jobs from the queue
type Worker struct {
	id         int
	manager    *Manager
	jobChan    <-chan *db.DownloadJob
	downloader *Downloader
	verifier   *Verifier
}

// newWorker creates a new worker
func newWorker(id int, manager *Manager, jobChan <-chan *db.DownloadJob) *Worker {
	return &Worker{
		id:         id,
		manager:    manager,
		jobChan:    jobChan,
		downloader: manager.downloader,
		verifier:   manager.verifier,
	}
}

// Run starts the worker loop
func (w *Worker) Run(ctx context.Context) {
	log.Debug("Worker started", "worker_id", w.id)
	defer log.Debug("Worker stopped", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-w.jobChan:
			if !ok {
				return
			}
			w.processJob(ctx, job)
		}
	}
}

// processJob handles a single download job with retries
func (w *Worker) processJob(ctx context.Context, job *db.DownloadJob) {
	log.Info("Processing download job",
		"worker_id", w.id,
		"job_id", job.ID,
		"component", job.ComponentID,
		"url", job.ResolvedURL,
	)

	// Create job-specific context with cancellation
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register cancel function with manager
	w.manager.registerCancel(job.ID, cancel)
	defer w.manager.unregisterCancel(job.ID)

	// Process with retries
	var lastErr error
	for attempt := 0; attempt <= job.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Info("Retrying download",
				"worker_id", w.id,
				"job_id", job.ID,
				"attempt", attempt,
				"max_retries", job.MaxRetries,
			)

			// Wait before retry with exponential backoff
			delay := w.manager.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-jobCtx.Done():
				w.handleFailure(job, "Job cancelled")
				return
			case <-time.After(delay):
			}

			// Reset job for retry
			if err := w.manager.jobRepo.IncrementRetry(job.ID); err != nil {
				log.Error("Failed to increment retry count", "job_id", job.ID, "error", err)
			}
		}

		// Verify URL exists before downloading
		if err := w.verify(jobCtx, job); err != nil {
			lastErr = err
			log.Warn("Verification failed",
				"worker_id", w.id,
				"job_id", job.ID,
				"error", err,
			)
			continue
		}

		// Execute download
		if err := w.download(jobCtx, job); err != nil {
			lastErr = err
			log.Warn("Download failed",
				"worker_id", w.id,
				"job_id", job.ID,
				"error", err,
			)
			continue
		}

		// Success
		log.Info("Download completed successfully",
			"worker_id", w.id,
			"job_id", job.ID,
			"component", job.ComponentID,
		)
		return
	}

	// All retries exhausted
	w.handleFailure(job, fmt.Sprintf("Max retries exceeded: %v", lastErr))
}

// verify checks that the URL is accessible
func (w *Worker) verify(ctx context.Context, job *db.DownloadJob) error {
	// Update status to verifying
	if err := w.manager.jobRepo.MarkVerifying(job.ID); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Determine retrieval method
	retrievalMethod := "release"
	if job.SourceType == "git" {
		retrievalMethod = "git"
	}

	result, err := w.verifier.Verify(ctx, job.ResolvedURL, retrievalMethod, job.Version)
	if err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	if !result.Exists {
		if result.Error != nil {
			return fmt.Errorf("resource not found: %w", result.Error)
		}
		return fmt.Errorf("resource not found at URL: %s", job.ResolvedURL)
	}

	return nil
}

// download executes the actual download
func (w *Worker) download(ctx context.Context, job *db.DownloadJob) error {
	progressCb := func(bytesReceived, totalBytes int64) {
		// Log progress at intervals
		if totalBytes > 0 {
			percent := float64(bytesReceived) / float64(totalBytes) * 100
			if int(percent)%10 == 0 {
				log.Debug("Download progress",
					"worker_id", w.id,
					"job_id", job.ID,
					"progress", fmt.Sprintf("%.1f%%", percent),
				)
			}
		}
	}

	return w.downloader.Download(ctx, job, progressCb)
}

// handleFailure marks a job as failed
func (w *Worker) handleFailure(job *db.DownloadJob, errorMsg string) {
	log.Error("Download job failed",
		"worker_id", w.id,
		"job_id", job.ID,
		"error", errorMsg,
	)

	if err := w.manager.jobRepo.MarkFailed(job.ID, errorMsg); err != nil {
		log.Error("Failed to mark job as failed", "job_id", job.ID, "error", err)
	}
}
