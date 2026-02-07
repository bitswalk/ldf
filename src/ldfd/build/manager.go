package build

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

var log = logs.NewDefault()

// SetLogger sets the logger for the build package
func SetLogger(l *logs.Logger) {
	if l != nil {
		log = l
	}
}

// Config holds configuration for the build manager
type Config struct {
	Workers          int           // Number of concurrent build workers
	WorkspaceBase    string        // Base directory for build workspaces
	ContainerImage   string        // Container image for build environment (or sysroot path for chroot)
	ContainerRuntime string        // Container runtime: podman, docker, nerdctl, or chroot
	RetryDelay       time.Duration // Base delay between retries
	MaxRetries       int           // Default max retries per job
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	return Config{
		Workers:          1,
		WorkspaceBase:    "~/.ldfd/cache/builds",
		ContainerImage:   "ldf-builder:latest",
		ContainerRuntime: "podman",
		RetryDelay:       30 * time.Second,
		MaxRetries:       1,
	}
}

// Manager coordinates build jobs across workers
type Manager struct {
	db               *db.Database
	storage          storage.Backend
	buildJobRepo     *db.BuildJobRepository
	distRepo         *db.DistributionRepository
	downloadJobRepo  *db.DownloadJobRepository
	componentRepo    *db.ComponentRepository
	sourceRepo       *db.SourceRepository
	boardProfileRepo *db.BoardProfileRepository
	downloadManager  *download.Manager
	config           Config
	stages           []Stage

	jobQueue    chan *db.BuildJob
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex
	wg          sync.WaitGroup

	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new build manager
func NewManager(database *db.Database, storageBackend storage.Backend,
	downloadMgr *download.Manager, cfg Config) *Manager {

	if cfg.Workers <= 0 {
		cfg.Workers = DefaultConfig().Workers
	}
	if cfg.WorkspaceBase == "" {
		cfg.WorkspaceBase = DefaultConfig().WorkspaceBase
	}
	if cfg.ContainerImage == "" {
		cfg.ContainerImage = DefaultConfig().ContainerImage
	}
	if cfg.ContainerRuntime == "" {
		cfg.ContainerRuntime = DefaultConfig().ContainerRuntime
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = DefaultConfig().RetryDelay
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = DefaultConfig().MaxRetries
	}

	m := &Manager{
		db:               database,
		storage:          storageBackend,
		buildJobRepo:     db.NewBuildJobRepository(database),
		distRepo:         db.NewDistributionRepository(database),
		downloadJobRepo:  db.NewDownloadJobRepository(database),
		componentRepo:    db.NewComponentRepository(database),
		sourceRepo:       db.NewSourceRepository(database),
		boardProfileRepo: db.NewBoardProfileRepository(database),
		downloadManager:  downloadMgr,
		config:           cfg,
		jobQueue:         make(chan *db.BuildJob, cfg.Workers*2),
		cancelFuncs:      make(map[string]context.CancelFunc),
	}

	return m
}

// RegisterStages sets up the ordered build pipeline
func (m *Manager) RegisterStages(stages []Stage) {
	m.stages = stages
}

// RegisterDefaultStages sets up the default build pipeline stages.
// The executor is created per-build in the worker from current config,
// not at registration time, so runtime changes take effect immediately.
func (m *Manager) RegisterDefaultStages() {
	m.stages = []Stage{
		NewResolveStage(m.componentRepo, m.downloadJobRepo, m.boardProfileRepo, m.sourceRepo, m.storage),
		NewDownloadCheckStage(m.downloadJobRepo, m.storage),
		NewPrepareStage(m.storage),
		NewCompileStage(),
		NewAssembleStage(),
		NewPackageStage(m.storage, 4), // 4GB default image size
	}

	log.Info("Registered default build stages",
		"count", len(m.stages),
		"stages", []string{"resolve", "download", "prepare", "compile", "assemble", "package"})
}

// Start begins processing build jobs
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("build manager already running")
	}
	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	log.Info("Build manager starting", "workers", m.config.Workers)

	// Start workers
	for i := 0; i < m.config.Workers; i++ {
		worker := newWorker(i, m, m.jobQueue)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			worker.Run(m.ctx)
		}()
	}

	// Start job dispatcher
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.dispatcher()
	}()

	log.Info("Build manager started")
	return nil
}

// Stop gracefully stops the build manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	log.Info("Build manager stopping")

	// Cancel all pending operations
	m.cancel()

	// Close job queue
	close(m.jobQueue)

	// Wait for workers to finish
	m.wg.Wait()

	log.Info("Build manager stopped")
	return nil
}

// dispatcher polls for pending jobs and dispatches them to workers
func (m *Manager) dispatcher() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.dispatchPendingJobs()
		}
	}
}

// dispatchPendingJobs fetches and dispatches pending build jobs
func (m *Manager) dispatchPendingJobs() {
	jobs, err := m.buildJobRepo.ListPending()
	if err != nil {
		log.Error("Failed to list pending build jobs", "error", err)
		return
	}

	for _, job := range jobs {
		j := job // capture loop variable
		select {
		case <-m.ctx.Done():
			return
		case m.jobQueue <- &j:
			log.Debug("Dispatched build job", "build_id", j.ID)
		default:
			log.Debug("Build job queue full, will retry", "build_id", j.ID)
			return
		}
	}
}

// SubmitBuild creates a build job for a distribution
func (m *Manager) SubmitBuild(dist *db.Distribution, userID string, arch db.TargetArch, format db.ImageFormat, clearCache bool) (*db.BuildJob, error) {
	if dist.Config == nil {
		return nil, fmt.Errorf("distribution has no configuration")
	}

	// Snapshot the config at build time
	configJSON, err := json.Marshal(dist.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot config: %w", err)
	}

	job := &db.BuildJob{
		DistributionID: dist.ID,
		OwnerID:        userID,
		TargetArch:     arch,
		ImageFormat:    format,
		Status:         db.BuildStatusPending,
		MaxRetries:     m.config.MaxRetries,
		ClearCache:     clearCache,
		ConfigSnapshot: string(configJSON),
	}

	if err := m.buildJobRepo.Create(job); err != nil {
		return nil, fmt.Errorf("failed to create build job: %w", err)
	}

	log.Info("Build job submitted",
		"build_id", job.ID,
		"distribution_id", dist.ID,
		"arch", arch,
		"format", format,
	)

	// Try to dispatch immediately
	select {
	case m.jobQueue <- job:
		log.Debug("Build job dispatched immediately", "build_id", job.ID)
	default:
		log.Debug("Build job queued for later dispatch", "build_id", job.ID)
	}

	return job, nil
}

// CancelBuild cancels a running or pending build
func (m *Manager) CancelBuild(buildID string) error {
	m.mu.RLock()
	cancel, exists := m.cancelFuncs[buildID]
	m.mu.RUnlock()

	if exists {
		cancel()
	}

	return m.buildJobRepo.MarkCancelled(buildID)
}

// RetryBuild retries a failed build
func (m *Manager) RetryBuild(buildID string) error {
	job, err := m.buildJobRepo.GetByID(buildID)
	if err != nil {
		return fmt.Errorf("failed to get build: %w", err)
	}
	if job == nil {
		return fmt.Errorf("build not found: %s", buildID)
	}

	if job.Status != db.BuildStatusFailed && job.Status != db.BuildStatusCancelled {
		return fmt.Errorf("can only retry failed or cancelled builds")
	}

	return m.buildJobRepo.IncrementRetry(buildID)
}

// GetBuildStatus returns the current status of a build
func (m *Manager) GetBuildStatus(buildID string) (*db.BuildJob, error) {
	return m.buildJobRepo.GetByID(buildID)
}

// GetConfig returns the build manager configuration
func (m *Manager) GetConfig() Config {
	return m.config
}

// BuildJobRepo returns the build job repository
func (m *Manager) BuildJobRepo() *db.BuildJobRepository {
	return m.buildJobRepo
}

// DistRepo returns the distribution repository
func (m *Manager) DistRepo() *db.DistributionRepository {
	return m.distRepo
}

// registerCancel registers a cancel function for a build
func (m *Manager) registerCancel(buildID string, cancel context.CancelFunc) {
	m.mu.Lock()
	m.cancelFuncs[buildID] = cancel
	m.mu.Unlock()
}

// unregisterCancel removes a cancel function for a build
func (m *Manager) unregisterCancel(buildID string) {
	m.mu.Lock()
	delete(m.cancelFuncs, buildID)
	m.mu.Unlock()
}
