package stages

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/build/kernel"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// ResolveStage resolves the distribution configuration into concrete components and versions
type ResolveStage struct {
	componentRepo    *db.ComponentRepository
	downloadJobRepo  *db.DownloadJobRepository
	boardProfileRepo *db.BoardProfileRepository
	sourceRepo       *db.SourceRepository
	storage          storage.Backend
}

// NewResolveStage creates a new resolve stage
func NewResolveStage(componentRepo *db.ComponentRepository, downloadJobRepo *db.DownloadJobRepository, boardProfileRepo *db.BoardProfileRepository, sourceRepo *db.SourceRepository, storageBackend storage.Backend) *ResolveStage {
	return &ResolveStage{
		componentRepo:    componentRepo,
		downloadJobRepo:  downloadJobRepo,
		boardProfileRepo: boardProfileRepo,
		sourceRepo:       sourceRepo,
		storage:          storageBackend,
	}
}

// Name returns the stage name
func (s *ResolveStage) Name() db.BuildStageName {
	return db.StageResolve
}

// Validate checks whether this stage can run
func (s *ResolveStage) Validate(ctx context.Context, sc *build.StageContext) error {
	if sc.Config == nil {
		return fmt.Errorf("distribution config is required")
	}
	if sc.DistributionID == "" {
		return fmt.Errorf("distribution ID is required")
	}
	return nil
}

// Execute resolves components and populates StageContext.Components
func (s *ResolveStage) Execute(ctx context.Context, sc *build.StageContext, progress build.ProgressFunc) error {
	progress(0, "Resolving required components")

	// Load board profile if configured
	if sc.Config.BoardProfileID != "" && s.boardProfileRepo != nil {
		progress(2, "Loading board profile")
		profile, err := s.boardProfileRepo.GetByID(sc.Config.BoardProfileID)
		if err != nil {
			return fmt.Errorf("failed to load board profile: %w", err)
		}
		if profile == nil {
			return fmt.Errorf("board profile not found: %s", sc.Config.BoardProfileID)
		}
		if profile.Arch != sc.TargetArch {
			return fmt.Errorf("board profile architecture mismatch: profile requires %s but build targets %s", profile.Arch, sc.TargetArch)
		}
		sc.BoardProfile = profile
		progress(5, fmt.Sprintf("Board profile loaded: %s (%s)", profile.DisplayName, profile.Arch))
	}

	// Get required component names from config
	componentNames := s.getRequiredComponents(sc.Config, sc.TargetArch)
	if len(componentNames) == 0 {
		return fmt.Errorf("no components required for this distribution configuration")
	}

	progress(10, fmt.Sprintf("Found %d required components", len(componentNames)))

	// Resolve each component to its concrete version and download artifact
	var resolved []build.ResolvedComponent
	for i, name := range componentNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pct := 10 + (80 * (i + 1) / len(componentNames))
		progress(pct, fmt.Sprintf("Resolving component: %s", name))

		component, err := s.componentRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("failed to lookup component %s: %w", name, err)
		}
		if component == nil {
			return fmt.Errorf("component not found: %s", name)
		}

		// Filter out components incompatible with target architecture
		if !isComponentCompatible(component, sc.TargetArch) {
			log.Info("Skipping component incompatible with target architecture",
				"component", component.Name,
				"target_arch", sc.TargetArch,
				"supported", component.SupportedArchitectures)
			continue
		}

		// Resolve version from config override or component default
		version := s.getComponentVersion(sc.Config, component)
		if version == "" {
			return fmt.Errorf("no version resolved for component %s", name)
		}

		// Find the download job for this component
		jobs, err := s.downloadJobRepo.ListByDistribution(sc.DistributionID)
		if err != nil {
			return fmt.Errorf("failed to list download jobs: %w", err)
		}

		var downloadJob *db.DownloadJob
		for _, job := range jobs {
			// Check if this job is for our component (either primary or in ComponentIDs list)
			if job.ComponentID == component.ID || containsString(job.ComponentIDs, component.ID) {
				if job.Version == version && job.Status == db.JobStatusCompleted {
					downloadJob = &job
					break
				}
			}
		}

		if downloadJob == nil {
			// Fallback: check if artifact exists directly in storage
			artifactPath := s.findArtifactInStorage(ctx, sc, component, version)
			if artifactPath != "" {
				log.Info("Resolved component from storage (no download job)",
					"component", name, "version", version, "artifact", artifactPath)
				resolved = append(resolved, build.ResolvedComponent{
					Component:    *component,
					Version:      version,
					ArtifactPath: artifactPath,
					LocalPath:    "",
				})
				continue
			}
			return fmt.Errorf("no completed download found for component %s version %s", name, version)
		}

		resolved = append(resolved, build.ResolvedComponent{
			Component:    *component,
			Version:      version,
			ArtifactPath: downloadJob.ArtifactPath,
			LocalPath:    "", // Will be set by prepare stage after extraction
		})
	}

	sc.Components = resolved

	// Fetch kernel .config artifact from storage into the workspace
	progress(95, "Fetching kernel config from storage")
	kernelConfigKey := kernel.KernelConfigArtifactPath(sc.OwnerID, sc.DistributionID)
	configPath := filepath.Join(sc.ConfigDir, ".config")

	if err := os.MkdirAll(sc.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := s.fetchKernelConfig(ctx, sc, kernelConfigKey, configPath); err != nil {
		// Config artifact doesn't exist (pre-build-engine distribution) -- generate it
		log.Warn("Kernel config artifact not found, generating default",
			"distribution_id", sc.DistributionID, "error", err)

		progress(97, "Generating default kernel config")
		configSvc := kernel.NewKernelConfigService(s.storage)
		dist := &db.Distribution{
			ID:      sc.DistributionID,
			OwnerID: sc.OwnerID,
			Config:  sc.Config,
		}
		if err := configSvc.GenerateAndStore(ctx, dist); err != nil {
			return fmt.Errorf("failed to generate kernel config artifact: %w", err)
		}

		// Now fetch the freshly generated artifact
		if err := s.fetchKernelConfig(ctx, sc, kernelConfigKey, configPath); err != nil {
			return fmt.Errorf("failed to fetch generated kernel config: %w", err)
		}
	}

	// Validate build toolchain availability (non-container mode only)
	progress(98, "Validating build toolchain")
	toolchain := db.ResolveToolchain(&sc.Config.Core)
	crossPrefix := ""
	if sc.BuildEnv != nil {
		crossPrefix = sc.BuildEnv.Toolchain.CrossCompilePrefix
	}

	// Check if toolchain will be provided by downloaded components
	hasToolchainComponent := false
	for _, rc := range sc.Components {
		if containsCat(rc.Component.Categories, "toolchain") {
			hasToolchainComponent = true
			break
		}
	}

	if hasToolchainComponent {
		log.Info("Skipping toolchain validation (toolchain provided by downloaded components)", "toolchain", toolchain)
	} else if sc.Executor != nil && !sc.Executor.RuntimeType().IsContainerRuntime() {
		deps := build.GetToolchainDeps(toolchain, crossPrefix)
		missing := build.ValidateToolchainAvailability(deps)
		if len(missing) > 0 {
			return fmt.Errorf("missing build toolchain dependencies: %v (install them or use a container-based executor)", missing)
		}
		log.Info("Build toolchain validated", "toolchain", toolchain, "deps", len(deps.All()))
	} else {
		log.Info("Skipping toolchain validation (container mode)", "toolchain", toolchain)
	}

	progress(100, fmt.Sprintf("Resolved %d components", len(resolved)))

	return nil
}

// fetchKernelConfig links or downloads the kernel config artifact into the workspace.
// For local storage, creates a symlink to avoid duplication; for S3, downloads the file.
func (s *ResolveStage) fetchKernelConfig(ctx context.Context, sc *build.StageContext, key, configPath string) error {
	if resolver, ok := s.storage.(storage.LocalPathResolver); ok {
		srcPath := resolver.ResolvePath(key)
		if _, err := os.Stat(srcPath); err != nil {
			return fmt.Errorf("kernel config artifact not found: %w", err)
		}
		if err := os.Symlink(srcPath, configPath); err != nil {
			log.Warn("Symlink failed, falling back to copy", "error", err)
			return s.downloadToFile(ctx, key, configPath)
		}
		return nil
	}
	return s.downloadToFile(ctx, key, configPath)
}

// downloadToFile downloads a storage object to a local file.
func (s *ResolveStage) downloadToFile(ctx context.Context, key, localPath string) error {
	reader, _, err := s.storage.Download(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", key, err)
	}
	defer reader.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", localPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write %s: %w", localPath, err)
	}

	return nil
}

// findArtifactInStorage checks if an artifact exists in the storage backend
// at the expected distribution path, without requiring a download_jobs record.
func (s *ResolveStage) findArtifactInStorage(ctx context.Context, sc *build.StageContext, component *db.Component, version string) string {
	if s.storage == nil || s.sourceRepo == nil {
		return ""
	}

	// Get the effective source for this component to determine sourceID
	source, err := s.sourceRepo.GetEffectiveSource(component.ID, sc.OwnerID)
	if err != nil || source == nil {
		return ""
	}

	sourceID := source.ID

	// Try both subdirectories: components/ (release) and sources/ (git)
	subdirs := []string{"components", "sources"}
	for _, subdir := range subdirs {
		prefix := fmt.Sprintf("distribution/%s/%s/%s/%s/%s/",
			sc.OwnerID, sc.DistributionID, subdir, sourceID, version)

		objects, err := s.storage.List(ctx, prefix)
		if err != nil {
			log.Warn("Storage list failed during resolve fallback",
				"prefix", prefix, "error", err)
			continue
		}

		for _, obj := range objects {
			if obj.Size > 0 {
				return obj.Key
			}
		}
	}

	return ""
}

// getRequiredComponents returns the list of component names required by the config.
// This mirrors download/manager.go's getRequiredComponents but returns names directly.
func (s *ResolveStage) getRequiredComponents(config *db.DistributionConfig, targetArch db.TargetArch) []string {
	var components []string

	// Helper to find component by category and config value
	findComponent := func(category, configValue string) {
		if configValue == "" {
			return
		}
		component, err := s.componentRepo.GetByCategoryAndNameContains(category, configValue)
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

	// Core - kernel is always required
	kernel, err := s.componentRepo.GetByCategoryAndNameContains("core", "kernel")
	if err != nil || kernel == nil {
		components = append(components, "kernel")
	} else {
		components = append(components, kernel.Name)
	}

	// Bootloader
	if config.Core.Bootloader != "" {
		findComponent("bootloader", config.Core.Bootloader)
	}

	// Init system
	if config.System.Init != "" {
		findComponent("init", config.System.Init)
	}

	// Filesystem userspace tools (only if userspace flag is set)
	if config.System.Filesystem.Type != "" && config.System.FilesystemUserspace {
		findComponent("filesystem", config.System.Filesystem.Type)
	}

	// Virtualization
	if config.Runtime.Virtualization != "" {
		findComponent("runtime", config.Runtime.Virtualization)
	}

	// Container
	if config.Runtime.Container != "" {
		findComponent("runtime", config.Runtime.Container)
	}

	// Security userspace tools (only if userspace flag is set)
	if config.Security.System != "" && config.Security.System != "none" && config.Security.SystemUserspace {
		findComponent("security", config.Security.System)
	}

	// Desktop (only if target is desktop)
	if config.Target.Type == "desktop" && config.Target.Desktop != nil && config.Target.Desktop.Environment != "" {
		findComponent("desktop", config.Target.Desktop.Environment)
	}

	// Board profile firmware components (resolved via board profile on StageContext)
	// Note: This is handled separately since we need the board profile loaded first.
	// Firmware with ComponentID references will be resolved during Execute after
	// the board profile is loaded into StageContext.

	// Toolchain components
	toolchain := db.ResolveToolchain(&config.Core)
	isNative := build.IsNativeBuild(build.DetectHostArch(), targetArch)
	switch toolchain {
	case db.ToolchainLLVM:
		findComponent("toolchain", "llvm")
	default: // GCC
		if !isNative && targetArch == db.ArchAARCH64 {
			findComponent("toolchain", "gcc-cross-aarch64")
		} else {
			findComponent("toolchain", "gcc-native")
		}
	}
	findComponent("toolchain", "build-essentials")

	return components
}

// getComponentVersion resolves the version for a component from config or default
func (s *ResolveStage) getComponentVersion(config *db.DistributionConfig, component *db.Component) string {
	// First check distribution config for explicit version override
	override := s.getDistributionVersionOverride(config, component.Name)
	if override != "" {
		return override
	}

	// Fall back to component's default version
	return component.DefaultVersion
}

// getDistributionVersionOverride checks distribution config for explicit version
func (s *ResolveStage) getDistributionVersionOverride(config *db.DistributionConfig, componentName string) string {
	lowerName := strings.ToLower(componentName)

	// Kernel version
	if strings.Contains(lowerName, "kernel") {
		return config.Core.Kernel.Version
	}

	// Bootloader version
	if config.Core.Bootloader != "" && strings.Contains(lowerName, strings.ToLower(config.Core.Bootloader)) {
		return config.Core.BootloaderVersion
	}

	// Init system version
	if config.System.Init != "" && strings.Contains(lowerName, strings.ToLower(config.System.Init)) {
		return config.System.InitVersion
	}

	// Filesystem version
	if config.System.Filesystem.Type != "" && strings.Contains(lowerName, strings.ToLower(config.System.Filesystem.Type)) {
		return config.System.FilesystemVersion
	}

	// Package manager version
	if config.System.PackageManager != "" && strings.Contains(lowerName, strings.ToLower(config.System.PackageManager)) {
		return config.System.PackageManagerVersion
	}

	// Container version
	if config.Runtime.Container != "" && strings.Contains(lowerName, strings.ToLower(config.Runtime.Container)) {
		return config.Runtime.ContainerVersion
	}

	// Virtualization version
	if config.Runtime.Virtualization != "" && strings.Contains(lowerName, strings.ToLower(config.Runtime.Virtualization)) {
		return config.Runtime.VirtualizationVersion
	}

	// Security system version
	if config.Security.System != "" && strings.Contains(lowerName, strings.ToLower(config.Security.System)) {
		return config.Security.SystemVersion
	}

	// Desktop environment version
	if config.Target.Desktop != nil {
		if config.Target.Desktop.Environment != "" && strings.Contains(lowerName, strings.ToLower(config.Target.Desktop.Environment)) {
			return config.Target.Desktop.EnvironmentVersion
		}
		if config.Target.Desktop.DisplayServer != "" && strings.Contains(lowerName, strings.ToLower(config.Target.Desktop.DisplayServer)) {
			return config.Target.Desktop.DisplayServerVersion
		}
	}

	return ""
}

// isComponentCompatible returns true if the component supports the given target
// architecture. An empty SupportedArchitectures list means the component supports
// all architectures (backward compatible default).
func isComponentCompatible(c *db.Component, arch db.TargetArch) bool {
	if len(c.SupportedArchitectures) == 0 {
		return true
	}
	for _, a := range c.SupportedArchitectures {
		if a == arch {
			return true
		}
	}
	return false
}

// containsString checks if a slice contains a string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
