package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	urlpath "path"
	"path/filepath"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// Downloader handles the actual download of files
type Downloader struct {
	httpClient *http.Client
	storage    storage.Backend
	jobRepo    *db.DownloadJobRepository
}

// ProgressCallback is called with download progress updates
type ProgressCallback func(bytesReceived, totalBytes int64)

// NewDownloader creates a new downloader
func NewDownloader(httpClient *http.Client, storage storage.Backend, jobRepo *db.DownloadJobRepository) *Downloader {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 0, // No timeout for downloads
		}
	}
	return &Downloader{
		httpClient: httpClient,
		storage:    storage,
		jobRepo:    jobRepo,
	}
}

// Download executes a download job
func (d *Downloader) Download(ctx context.Context, job *db.DownloadJob, progressCb ProgressCallback) error {
	// Determine retrieval method
	if job.RetrievalMethod == "git" {
		return d.downloadGit(ctx, job, progressCb)
	}
	return d.downloadHTTP(ctx, job, progressCb)
}

// downloadHTTP downloads a file via HTTP(S)
func (d *Downloader) downloadHTTP(ctx context.Context, job *db.DownloadJob, progressCb ProgressCallback) error {
	// Mark job as started
	if err := d.jobRepo.MarkStarted(job.ID); err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", job.ResolvedURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ldfd/1.0")

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	totalBytes := resp.ContentLength

	// Create temp file
	tempFile, err := os.CreateTemp("", fmt.Sprintf("ldf-download-%s-*", job.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	// Create progress tracking reader
	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	var bytesReceived int64
	buf := make([]byte, 32*1024) // 32KB buffer

	// Throttle progress updates to avoid database lock contention
	// Only update the database every 2 seconds
	lastProgressUpdate := time.Now()
	progressUpdateInterval := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to temp file: %w", writeErr)
			}

			bytesReceived += int64(n)

			// Update progress callback (for logging)
			if progressCb != nil {
				progressCb(bytesReceived, totalBytes)
			}

			// Throttle database progress updates to avoid lock contention
			now := time.Now()
			if now.Sub(lastProgressUpdate) >= progressUpdateInterval {
				if err := d.jobRepo.UpdateProgress(job.ID, bytesReceived, totalBytes); err != nil {
					// Log but don't fail on progress update errors
					log.Warn("Failed to update job progress", "job_id", job.ID, "error", err)
				}
				lastProgressUpdate = now
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}
	}

	// Final progress update to ensure we record 100%
	if err := d.jobRepo.UpdateProgress(job.ID, bytesReceived, totalBytes); err != nil {
		log.Warn("Failed to update final job progress", "job_id", job.ID, "error", err)
	}

	// Calculate checksum
	checksum := hex.EncodeToString(hash.Sum(nil))

	// Seek to beginning for upload
	if _, err := tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek temp file: %w", err)
	}

	// Determine artifact path in storage
	artifactPath := d.buildArtifactPath(job)

	// Get file info for size
	stat, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Upload to storage
	contentType := d.detectContentType(job.ResolvedURL)
	if err := d.storage.Upload(ctx, artifactPath, tempFile, stat.Size(), contentType); err != nil {
		return fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Mark job as completed
	if err := d.jobRepo.MarkCompleted(job.ID, artifactPath, checksum, stat.Size()); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	return nil
}

// DownloadLocal copies a file from a local mirror path into storage
func (d *Downloader) DownloadLocal(ctx context.Context, job *db.DownloadJob, localPath string) error {
	if err := d.jobRepo.MarkStarted(job.ID); err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local mirror file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local mirror file: %w", err)
	}

	// Calculate checksum
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek local file: %w", err)
	}

	artifactPath := d.buildArtifactPath(job)
	contentType := d.detectContentType(localPath)

	if err := d.storage.Upload(ctx, artifactPath, f, stat.Size(), contentType); err != nil {
		return fmt.Errorf("failed to upload to storage: %w", err)
	}

	if err := d.jobRepo.UpdateProgress(job.ID, stat.Size(), stat.Size()); err != nil {
		log.Warn("Failed to update progress for local download", "job_id", job.ID, "error", err)
	}

	if err := d.jobRepo.MarkCompleted(job.ID, artifactPath, checksum, stat.Size()); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	log.Info("Local mirror download completed", "job_id", job.ID, "path", localPath, "size", stat.Size())
	return nil
}

// DownloadHTTPWithOptions downloads a file via HTTP using a resolved URL with optional rate limiters
func (d *Downloader) DownloadHTTPWithOptions(ctx context.Context, job *db.DownloadJob, resolvedURL string, progressCb ProgressCallback, limiters ...*rateLimiter) error {
	if err := d.jobRepo.MarkStarted(job.ID); err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", resolvedURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	totalBytes := resp.ContentLength

	tempFile, err := os.CreateTemp("", fmt.Sprintf("ldf-download-%s-*", job.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	defer tempFile.Close()

	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)

	// Wrap response body with throttled reader if limiters are provided
	var reader io.Reader = resp.Body
	tr := newThrottledReader(ctx, resp.Body, limiters...)
	if len(tr.limiters) > 0 {
		reader = tr
	}

	var bytesReceived int64
	buf := make([]byte, 32*1024)
	lastProgressUpdate := time.Now()
	progressUpdateInterval := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := writer.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			bytesReceived += int64(n)
			if progressCb != nil {
				progressCb(bytesReceived, totalBytes)
			}
			now := time.Now()
			if now.Sub(lastProgressUpdate) >= progressUpdateInterval {
				if err := d.jobRepo.UpdateProgress(job.ID, bytesReceived, totalBytes); err != nil {
					log.Warn("Failed to update job progress", "job_id", job.ID, "error", err)
				}
				lastProgressUpdate = now
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}
	}

	if err := d.jobRepo.UpdateProgress(job.ID, bytesReceived, totalBytes); err != nil {
		log.Warn("Failed to update final job progress", "job_id", job.ID, "error", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	if _, err := tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek temp file: %w", err)
	}

	artifactPath := d.buildArtifactPath(job)
	stat, err := tempFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	contentType := d.detectContentType(resolvedURL)
	if err := d.storage.Upload(ctx, artifactPath, tempFile, stat.Size(), contentType); err != nil {
		return fmt.Errorf("failed to upload to storage: %w", err)
	}

	if err := d.jobRepo.MarkCompleted(job.ID, artifactPath, checksum, stat.Size()); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	return nil
}

// downloadGit clones a git repository
func (d *Downloader) downloadGit(ctx context.Context, job *db.DownloadJob, progressCb ProgressCallback) error {
	// Mark job as started
	if err := d.jobRepo.MarkStarted(job.ID); err != nil {
		return fmt.Errorf("failed to mark job started: %w", err)
	}

	// Create temp directory for clone
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("ldf-git-%s-*", job.ID))
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Determine the git ref (tag)
	ref := "v" + job.Version

	// Clone the repository with the specific tag
	cloneCmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1",
		"--branch", ref,
		job.ResolvedURL,
		tempDir,
	)

	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	// Create archive from the cloned repository
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("ldf-archive-%s.tar.gz", job.ID))
	defer os.Remove(archivePath)

	archiveCmd := exec.CommandContext(ctx, "tar", "-czf", archivePath, "-C", tempDir, ".")
	if output, err := archiveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar archive failed: %w, output: %s", err, string(output))
	}

	// Open archive for reading
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archiveFile.Close()

	// Calculate checksum
	hash := sha256.New()
	if _, err := io.Copy(hash, archiveFile); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	// Seek back to beginning
	if _, err := archiveFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek archive: %w", err)
	}

	// Get file info
	stat, err := archiveFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	// Determine artifact path
	artifactPath := d.buildArtifactPath(job)

	// Upload to storage
	if err := d.storage.Upload(ctx, artifactPath, archiveFile, stat.Size(), "application/gzip"); err != nil {
		return fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Update progress to 100%
	if progressCb != nil {
		progressCb(stat.Size(), stat.Size())
	}

	// Mark job as completed
	if err := d.jobRepo.MarkCompleted(job.ID, artifactPath, checksum, stat.Size()); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	return nil
}

// buildArtifactPath constructs the storage path for a download artifact
// Following the artifact module pattern: distribution/{ownerID}/{distributionID}/{path}
// Artifacts are stored by source ID (not name) to enable deduplication and avoid
// issues with spaces or special characters in source names.
// For release archives: distribution/{ownerID}/{distributionID}/components/{sourceID}/{version}/{filename}
// For git sources: distribution/{ownerID}/{distributionID}/sources/{sourceID}/{version}/{filename}
func (d *Downloader) buildArtifactPath(job *db.DownloadJob) string {
	// Extract filename from URL or construct one
	filename := urlpath.Base(job.ResolvedURL)
	if filename == "" || filename == "." || filename == "/" {
		filename = fmt.Sprintf("%s-%s.tar.gz", job.SourceID, job.Version)
	}

	// Determine the subdirectory based on retrieval method
	// Git sources go to "sources/", release archives go to "components/"
	subdir := "components"
	if job.RetrievalMethod == "git" {
		subdir = "sources"
	}

	// Use source ID for the path to enable deduplication and avoid path issues
	// Fall back to component ID for backward compatibility with older jobs
	pathID := job.SourceID
	if pathID == "" {
		pathID = job.ComponentID
	}

	// Build path following artifact module pattern:
	// distribution/{ownerID}/{distributionID}/{subdir}/{sourceID}/{version}/{filename}
	return fmt.Sprintf("distribution/%s/%s/%s/%s/%s/%s",
		job.OwnerID,
		job.DistributionID,
		subdir,
		pathID,
		job.Version,
		filename,
	)
}

// detectContentType attempts to determine content type from URL
func (d *Downloader) detectContentType(url string) string {
	ext := urlpath.Ext(url)
	switch ext {
	case ".tar.gz", ".tgz":
		return "application/gzip"
	case ".tar.xz":
		return "application/x-xz"
	case ".tar.bz2":
		return "application/x-bzip2"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	default:
		return "application/octet-stream"
	}
}

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	ArtifactPath string
	Checksum     string
	Size         int64
	Duration     time.Duration
}
