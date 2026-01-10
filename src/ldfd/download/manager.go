// Package download provides a download manager for retrieving component artifacts.
package download

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

var log *logs.Logger

// SetLogger sets the logger for the download package
func SetLogger(l *logs.Logger) {
	log = l
}

// Config holds configuration for the download manager
type Config struct {
	Workers        int           // Number of concurrent download workers
	RetryDelay     time.Duration // Base delay between retries
	RequestTimeout time.Duration // HTTP request timeout
	MaxRetries     int           // Default max retries per job
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() Config {
	return Config{
		Workers:        3,
		RetryDelay:     5 * time.Second,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
	}
}

// Manager coordinates download jobs across multiple workers
type Manager struct {
	db                *db.Database
	storage           storage.Backend
	jobRepo           *db.DownloadJobRepository
	componentRepo     *db.ComponentRepository
	sourceRepo        *db.SourceRepository
	sourceVersionRepo *db.SourceVersionRepository
	urlBuilder        *URLBuilder
	verifier          *Verifier
	downloader        *Downloader
	config            Config

	jobQueue    chan *db.DownloadJob
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex
	wg          sync.WaitGroup

	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new download manager
func NewManager(database *db.Database, storageBackend storage.Backend, cfg Config) *Manager {
	if cfg.Workers <= 0 {
		cfg.Workers = DefaultConfig().Workers
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = DefaultConfig().RetryDelay
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = DefaultConfig().RequestTimeout
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = DefaultConfig().MaxRetries
	}

	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	jobRepo := db.NewDownloadJobRepository(database)
	componentRepo := db.NewComponentRepository(database)
	sourceRepo := db.NewSourceRepository(database)
	sourceVersionRepo := db.NewSourceVersionRepository(database)

	m := &Manager{
		db:                database,
		storage:           storageBackend,
		jobRepo:           jobRepo,
		componentRepo:     componentRepo,
		sourceRepo:        sourceRepo,
		sourceVersionRepo: sourceVersionRepo,
		urlBuilder:        NewURLBuilder(componentRepo),
		verifier:          NewVerifier(httpClient),
		downloader:        NewDownloader(nil, storageBackend, jobRepo), // nil client = no timeout for downloads
		config:            cfg,
		jobQueue:          make(chan *db.DownloadJob, cfg.Workers*2),
		cancelFuncs:       make(map[string]context.CancelFunc),
	}

	return m
}

// Start begins processing download jobs
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("manager already running")
	}
	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	log.Info("Download manager starting", "workers", m.config.Workers)

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

	log.Info("Download manager started")
	return nil
}

// Stop gracefully stops the download manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	log.Info("Download manager stopping")

	// Cancel all pending operations
	m.cancel()

	// Close job queue
	close(m.jobQueue)

	// Wait for workers to finish
	m.wg.Wait()

	log.Info("Download manager stopped")
	return nil
}

// dispatcher polls for pending jobs and dispatches them to workers
func (m *Manager) dispatcher() {
	ticker := time.NewTicker(5 * time.Second)
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

// dispatchPendingJobs fetches and dispatches pending jobs
func (m *Manager) dispatchPendingJobs() {
	jobs, err := m.jobRepo.ListPending()
	if err != nil {
		log.Error("Failed to list pending jobs", "error", err)
		return
	}

	for _, job := range jobs {
		select {
		case <-m.ctx.Done():
			return
		case m.jobQueue <- &job:
			log.Debug("Dispatched job", "job_id", job.ID)
		default:
			// Queue full, will try again next tick
			log.Debug("Job queue full, will retry", "job_id", job.ID)
			return
		}
	}
}

// SubmitJob adds a new job to be processed
func (m *Manager) SubmitJob(job *db.DownloadJob) error {
	// Set defaults
	if job.MaxRetries == 0 {
		job.MaxRetries = m.config.MaxRetries
	}
	if job.Status == "" {
		job.Status = db.JobStatusPending
	}

	// Create job in database
	if err := m.jobRepo.Create(job); err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	log.Info("Job submitted", "job_id", job.ID, "component", job.ComponentID)

	// Try to dispatch immediately if queue has space
	select {
	case m.jobQueue <- job:
		log.Debug("Job dispatched immediately", "job_id", job.ID)
	default:
		// Queue full, dispatcher will pick it up
		log.Debug("Job queued for later dispatch", "job_id", job.ID)
	}

	return nil
}

// CancelJob cancels a running or pending job
func (m *Manager) CancelJob(jobID string) error {
	m.mu.RLock()
	cancel, exists := m.cancelFuncs[jobID]
	m.mu.RUnlock()

	if exists {
		cancel()
	}

	return m.jobRepo.MarkCancelled(jobID)
}

// RetryJob retries a failed job
func (m *Manager) RetryJob(jobID string) error {
	job, err := m.jobRepo.GetByID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status != db.JobStatusFailed && job.Status != db.JobStatusCancelled {
		return fmt.Errorf("can only retry failed or cancelled jobs")
	}

	// Reset job for retry
	return m.jobRepo.IncrementRetry(jobID)
}

// GetJobStatus returns the current status of a job
func (m *Manager) GetJobStatus(jobID string) (*db.DownloadJob, error) {
	return m.jobRepo.GetByID(jobID)
}

// CreateJobsForDistribution creates download jobs for all components needed by a distribution
func (m *Manager) CreateJobsForDistribution(dist *db.Distribution, userID string) ([]db.DownloadJob, error) {
	if dist.Config == nil {
		return nil, fmt.Errorf("distribution has no configuration")
	}

	// Determine which components are needed based on distribution config
	componentNames := m.getRequiredComponents(dist.Config)

	var jobs []db.DownloadJob

	for _, componentName := range componentNames {
		job, err := m.createJobForComponent(dist, componentName, userID)
		if err != nil {
			log.Warn("Failed to create job for component", "component", componentName, "error", err)
			continue
		}
		if job != nil {
			jobs = append(jobs, *job)
		}
	}

	return jobs, nil
}

// createJobForComponent creates a download job for a specific component
func (m *Manager) createJobForComponent(dist *db.Distribution, componentName string, userID string) (*db.DownloadJob, error) {
	// Get component from registry
	component, err := m.componentRepo.GetByName(componentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get component: %w", err)
	}
	if component == nil {
		return nil, fmt.Errorf("component not found: %s", componentName)
	}

	// Get effective source for this component (priority-based selection)
	source, err := m.sourceRepo.GetEffectiveSource(component.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective source: %w", err)
	}
	if source == nil {
		return nil, fmt.Errorf("no source available for component: %s", componentName)
	}

	// Get version for this component
	version := m.getComponentVersion(dist.Config, componentName)
	if version == "" {
		return nil, fmt.Errorf("no version specified for component: %s", componentName)
	}

	// Build the resolved URL
	resolvedURL, err := m.urlBuilder.BuildURL(source, component, version)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Determine retrieval method
	retrievalMethod := source.RetrievalMethod
	if retrievalMethod == "" {
		retrievalMethod = "release"
	}

	// Create the job
	job := &db.DownloadJob{
		DistributionID:  dist.ID,
		OwnerID:         dist.OwnerID,
		ComponentID:     component.ID,
		ComponentName:   component.Name,
		SourceID:        source.ID,
		SourceType:      m.getSourceType(source),
		RetrievalMethod: retrievalMethod,
		ResolvedURL:     resolvedURL,
		Version:         version,
		Status:          db.JobStatusPending,
		MaxRetries:      m.config.MaxRetries,
	}

	if err := m.jobRepo.Create(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// getRequiredComponents determines which components are needed based on distribution config
// It dynamically looks up components from the database by category and config value
func (m *Manager) getRequiredComponents(config *db.DistributionConfig) []string {
	var components []string

	// Helper to find component by category and config value
	findComponent := func(category, configValue string) {
		if configValue == "" {
			return
		}
		component, err := m.componentRepo.GetByCategoryAndNameContains(category, configValue)
		if err != nil {
			log.Warn("Failed to lookup component", "category", category, "configValue", configValue, "error", err)
			return
		}
		if component != nil {
			components = append(components, component.Name)
		} else {
			log.Warn("No component found for config", "category", category, "configValue", configValue)
		}
	}

	// Core - kernel is always required (look it up by category)
	kernel, err := m.componentRepo.GetByCategoryAndNameContains("core", "kernel")
	if err != nil || kernel == nil {
		// Fallback to hardcoded name if not found
		components = append(components, "kernel")
	} else {
		components = append(components, kernel.Name)
	}

	// Bootloader - lookup by category and config value
	if config.Core.Bootloader != "" {
		findComponent("bootloader", config.Core.Bootloader)
	}

	// Init system
	if config.System.Init != "" {
		findComponent("init", config.System.Init)
	}

	// Virtualization
	if config.Runtime.Virtualization != "" {
		findComponent("runtime", config.Runtime.Virtualization)
	}

	// Container
	if config.Runtime.Container != "" {
		findComponent("runtime", config.Runtime.Container)
	}

	// Security
	if config.Security.System != "" {
		findComponent("security", config.Security.System)
	}

	// Desktop (only if target is desktop)
	if config.Target.Type == "desktop" && config.Target.Desktop != nil && config.Target.Desktop.Environment != "" {
		findComponent("desktop", config.Target.Desktop.Environment)
	}

	return components
}

// getComponentVersion extracts the version for a component from distribution config
// Priority: 1) Distribution config override 2) Component default (pinned or resolved from rule)
func (m *Manager) getComponentVersion(config *db.DistributionConfig, componentName string) string {
	// First check if distribution config has an explicit version override
	overrideVersion := m.getDistributionVersionOverride(config, componentName)
	if overrideVersion != "" {
		return overrideVersion
	}

	// No override, resolve from component's default
	return m.resolveComponentDefaultVersion(componentName)
}

// getDistributionVersionOverride gets explicit version override from distribution config
func (m *Manager) getDistributionVersionOverride(config *db.DistributionConfig, componentName string) string {
	switch componentName {
	case "kernel":
		return config.Core.Kernel.Version
	default:
		// Check bootloader version
		if config.Core.Bootloader != "" && containsIgnoreCase(componentName, config.Core.Bootloader) {
			return config.Core.BootloaderVersion
		}
		// Check init system version
		if config.System.Init != "" && containsIgnoreCase(componentName, config.System.Init) {
			return config.System.InitVersion
		}
		// Check filesystem version
		if config.System.Filesystem.Type != "" && containsIgnoreCase(componentName, config.System.Filesystem.Type) {
			return config.System.FilesystemVersion
		}
		// Check package manager version
		if config.System.PackageManager != "" && containsIgnoreCase(componentName, config.System.PackageManager) {
			return config.System.PackageManagerVersion
		}
		// Check security system version
		if config.Security.System != "" && containsIgnoreCase(componentName, config.Security.System) {
			return config.Security.SystemVersion
		}
		// Check container runtime version
		if config.Runtime.Container != "" && containsIgnoreCase(componentName, config.Runtime.Container) {
			return config.Runtime.ContainerVersion
		}
		// Check virtualization version
		if config.Runtime.Virtualization != "" && containsIgnoreCase(componentName, config.Runtime.Virtualization) {
			return config.Runtime.VirtualizationVersion
		}
		// Check desktop environment version
		if config.Target.Desktop != nil && config.Target.Desktop.Environment != "" &&
			containsIgnoreCase(componentName, config.Target.Desktop.Environment) {
			return config.Target.Desktop.EnvironmentVersion
		}
		// Check display server version
		if config.Target.Desktop != nil && config.Target.Desktop.DisplayServer != "" &&
			containsIgnoreCase(componentName, config.Target.Desktop.DisplayServer) {
			return config.Target.Desktop.DisplayServerVersion
		}
	}
	return ""
}

// resolveComponentDefaultVersion resolves the default version for a component
func (m *Manager) resolveComponentDefaultVersion(componentName string) string {
	component, err := m.componentRepo.GetByName(componentName)
	if err != nil || component == nil {
		log.Warn("Failed to get component for version resolution", "component", componentName, "error", err)
		return ""
	}

	// If pinned version, use it directly
	if component.DefaultVersionRule == db.VersionRulePinned {
		return component.DefaultVersion
	}

	// Otherwise, resolve the rule to an actual version
	var version *db.SourceVersion

	switch component.DefaultVersionRule {
	case db.VersionRuleLatestStable:
		version, err = m.sourceVersionRepo.GetLatestStableByComponent(component.ID)
	case db.VersionRuleLatestLTS:
		version, err = m.sourceVersionRepo.GetLatestLongtermByComponent(component.ID)
	default:
		// Default to latest stable if no rule specified
		version, err = m.sourceVersionRepo.GetLatestStableByComponent(component.ID)
	}

	if err != nil {
		log.Warn("Failed to resolve version for component", "component", componentName, "rule", component.DefaultVersionRule, "error", err)
		return ""
	}

	if version == nil {
		log.Warn("No version found for component", "component", componentName, "rule", component.DefaultVersionRule)
		return ""
	}

	return version.Version
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) > 0 && (strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}

// getSourceType determines if a source is "default" or "user"
// Deprecated: Use db.GetSourceType() directly instead
func (m *Manager) getSourceType(source *db.UpstreamSource) string {
	return db.GetSourceType(source)
}

// registerCancel registers a cancel function for a job
func (m *Manager) registerCancel(jobID string, cancel context.CancelFunc) {
	m.mu.Lock()
	m.cancelFuncs[jobID] = cancel
	m.mu.Unlock()
}

// unregisterCancel removes a cancel function for a job
func (m *Manager) unregisterCancel(jobID string) {
	m.mu.Lock()
	delete(m.cancelFuncs, jobID)
	m.mu.Unlock()
}

// JobRepo returns the job repository for external access
func (m *Manager) JobRepo() *db.DownloadJobRepository {
	return m.jobRepo
}

// ComponentRepo returns the component repository for external access
func (m *Manager) ComponentRepo() *db.ComponentRepository {
	return m.componentRepo
}

// SourceRepo returns the source repository for external access
func (m *Manager) SourceRepo() *db.SourceRepository {
	return m.sourceRepo
}

// URLBuilder returns the URL builder for external access
func (m *Manager) URLBuilder() *URLBuilder {
	return m.urlBuilder
}

// GetKernelModulesForDistribution returns kernel modules needed for a distribution
func (m *Manager) GetKernelModulesForDistribution(dist *db.Distribution) ([]db.Component, error) {
	componentNames := m.getRequiredComponents(dist.Config)

	var kernelModules []db.Component
	for _, name := range componentNames {
		component, err := m.componentRepo.GetByName(name)
		if err != nil || component == nil {
			continue
		}
		if component.IsKernelModule {
			kernelModules = append(kernelModules, *component)
		}
	}
	return kernelModules, nil
}

// GetUserspaceComponentsForDistribution returns userspace components for a distribution
func (m *Manager) GetUserspaceComponentsForDistribution(dist *db.Distribution) ([]db.Component, error) {
	componentNames := m.getRequiredComponents(dist.Config)

	var userspaceComponents []db.Component
	for _, name := range componentNames {
		component, err := m.componentRepo.GetByName(name)
		if err != nil || component == nil {
			continue
		}
		if component.IsUserspace {
			userspaceComponents = append(userspaceComponents, *component)
		}
	}
	return userspaceComponents, nil
}

// GetHybridComponentsForDistribution returns components that are both kernel modules and userspace
func (m *Manager) GetHybridComponentsForDistribution(dist *db.Distribution) ([]db.Component, error) {
	componentNames := m.getRequiredComponents(dist.Config)

	var hybridComponents []db.Component
	for _, name := range componentNames {
		component, err := m.componentRepo.GetByName(name)
		if err != nil || component == nil {
			continue
		}
		if component.IsKernelModule && component.IsUserspace {
			hybridComponents = append(hybridComponents, *component)
		}
	}
	return hybridComponents, nil
}
