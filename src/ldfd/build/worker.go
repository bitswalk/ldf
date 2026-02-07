package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bitswalk/ldf/src/common/paths"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/spf13/viper"
)

// Worker processes build jobs from the queue
type Worker struct {
	id      int
	manager *Manager
	jobChan <-chan *db.BuildJob
}

// newWorker creates a new build worker
func newWorker(id int, manager *Manager, jobChan <-chan *db.BuildJob) *Worker {
	return &Worker{
		id:      id,
		manager: manager,
		jobChan: jobChan,
	}
}

// Run starts the worker loop
func (w *Worker) Run(ctx context.Context) {
	log.Debug("Build worker started", "worker_id", w.id)
	defer log.Debug("Build worker stopped", "worker_id", w.id)

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

// processJob handles a single build job through all pipeline stages
func (w *Worker) processJob(ctx context.Context, job *db.BuildJob) {
	// Recover from panics so the worker goroutine survives and
	// the build job gets marked as failed instead of hanging forever.
	defer func() {
		if r := recover(); r != nil {
			log.Error("Build worker recovered from panic",
				"worker_id", w.id,
				"build_id", job.ID,
				"panic", fmt.Sprintf("%v", r),
			)
			w.handleFailure(job, fmt.Sprintf("internal error (panic): %v", r), job.CurrentStage)
		}
	}()
	log.Info("Processing build job",
		"worker_id", w.id,
		"build_id", job.ID,
		"distribution_id", job.DistributionID,
		"arch", job.TargetArch,
		"format", job.ImageFormat,
	)

	// Create job-specific context with cancellation
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register cancel function with manager
	w.manager.registerCancel(job.ID, cancel)
	defer w.manager.unregisterCancel(job.ID)

	// Mark job as started
	if err := w.manager.buildJobRepo.MarkStarted(job.ID); err != nil {
		log.Error("Failed to mark build started", "build_id", job.ID, "error", err)
		return
	}

	// Update distribution status to building
	if err := w.manager.distRepo.UpdateStatus(job.DistributionID, db.StatusBuilding, ""); err != nil {
		log.Warn("Failed to update distribution status to building", "distribution_id", job.DistributionID, "error", err)
	}

	// Parse the config snapshot
	var config db.DistributionConfig
	if job.ConfigSnapshot != "" {
		if err := json.Unmarshal([]byte(job.ConfigSnapshot), &config); err != nil {
			w.handleFailure(job, fmt.Sprintf("Failed to parse config snapshot: %v", err), "")
			return
		}
	}

	// Validate build environment (architecture, toolchain, container image)
	// Read live config from viper so Settings changes take effect without restart
	liveRuntime := viper.GetString("build.container_runtime")
	if liveRuntime == "" {
		liveRuntime = w.manager.config.ContainerRuntime
	}
	liveImage := viper.GetString("build.container_image")
	if liveImage == "" {
		liveImage = w.manager.config.ContainerImage
	}
	runtime := RuntimeType(liveRuntime)
	buildEnv, err := ValidateBuildEnvironment(runtime, liveImage, job.TargetArch)
	if err != nil {
		w.handleFailure(job, fmt.Sprintf("Build environment validation failed: %v", err), "")
		return
	}

	log.Info("Build environment validated",
		"runtime", runtime,
		"host_arch", buildEnv.HostArch,
		"target_arch", buildEnv.TargetArch,
		"native", buildEnv.IsNative,
		"cross_compile", buildEnv.Toolchain.CrossCompilePrefix,
		"container_image", buildEnv.ContainerImage,
		"qemu_emulation", buildEnv.UseQEMUEmulation,
	)

	// Set up workspace (expand ~ to home directory)
	workspaceBase := paths.Expand(w.manager.config.WorkspaceBase)
	workspacePath := filepath.Join(workspaceBase, job.ID)
	sourcesDir := filepath.Join(workspacePath, "sources")
	rootfsDir := filepath.Join(workspacePath, "rootfs")
	outputDir := filepath.Join(workspacePath, "output")
	configDir := filepath.Join(workspacePath, "config")

	dirs := []string{workspacePath, sourcesDir, rootfsDir, outputDir, configDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			w.handleFailure(job, fmt.Sprintf("Failed to create workspace directory: %v", err), "")
			return
		}
	}

	// Create executor from current config (reads live viper values so
	// runtime changes via the Settings API take effect immediately)
	runtimeName := viper.GetString("build.container_runtime")
	if runtimeName == "" {
		runtimeName = w.manager.config.ContainerRuntime
	}
	containerImage := viper.GetString("build.container_image")
	if containerImage == "" {
		containerImage = w.manager.config.ContainerImage
	}

	execRuntime := RuntimeType(runtimeName)

	// For chroot mode, use direct host execution (empty sysroot).
	// The compile and package stages handle path translation internally.
	if execRuntime == RuntimeChroot {
		containerImage = ""
	}

	executor, err := NewExecutor(execRuntime, containerImage, nil)
	if err != nil {
		w.handleFailure(job, fmt.Sprintf("Failed to create build executor: %v", err), "")
		return
	}

	log.Info("Build executor created from live config",
		"runtime", execRuntime,
		"image", containerImage,
		"build_id", job.ID,
	)

	// Create stage context
	sc := &StageContext{
		BuildID:        job.ID,
		DistributionID: job.DistributionID,
		OwnerID:        job.OwnerID,
		Config:         &config,
		TargetArch:     job.TargetArch,
		ImageFormat:    job.ImageFormat,
		WorkspacePath:  workspacePath,
		SourcesDir:     sourcesDir,
		RootfsDir:      rootfsDir,
		OutputDir:      outputDir,
		ConfigDir:      configDir,
		BuildEnv:       buildEnv,
		Executor:       executor,
	}

	// Create stage records in database
	for _, stage := range w.manager.stages {
		stageRecord := &db.BuildStage{
			BuildID: job.ID,
			Name:    stage.Name(),
			Status:  "pending",
		}
		if err := w.manager.buildJobRepo.CreateStage(stageRecord); err != nil {
			log.Warn("Failed to create stage record", "build_id", job.ID, "stage", stage.Name(), "error", err)
		}
	}

	// Run each stage sequentially
	for i, stage := range w.manager.stages {
		stageName := stage.Name()

		// Check for cancellation
		select {
		case <-jobCtx.Done():
			w.handleFailure(job, "Build cancelled", string(stageName))
			w.cleanup(workspacePath)
			return
		default:
		}

		// Update DB with current stage
		stageProgress := (i * 100) / len(w.manager.stages)
		if err := w.manager.buildJobRepo.UpdateStage(job.ID, string(stageName), stageProgress); err != nil {
			log.Warn("Failed to update build stage", "build_id", job.ID, "error", err)
		}

		// Mark stage as running
		if err := w.manager.buildJobRepo.UpdateStageStatus(job.ID, stageName, "running"); err != nil {
			log.Warn("Failed to update stage status", "build_id", job.ID, "stage", stageName, "error", err)
		}

		if err := w.manager.buildJobRepo.AppendLog(job.ID, string(stageName), "info",
			fmt.Sprintf("Starting stage: %s", stageName)); err != nil {
			log.Warn("Failed to append build log", "build_id", job.ID, "error", err)
		}

		stageStart := time.Now()

		// Validate stage
		if err := stage.Validate(jobCtx, sc); err != nil {
			if logErr := w.manager.buildJobRepo.AppendLog(job.ID, string(stageName), "error",
				fmt.Sprintf("Stage validation failed: %v", err)); logErr != nil {
				log.Warn("Failed to append build log", "build_id", job.ID, "error", logErr)
			}
			if logErr := w.manager.buildJobRepo.MarkStageFailed(job.ID, stageName, err.Error()); logErr != nil {
				log.Warn("Failed to mark stage failed", "build_id", job.ID, "stage", stageName, "error", logErr)
			}
			w.handleFailure(job, fmt.Sprintf("Stage %s validation failed: %v", stageName, err), string(stageName))
			w.cleanup(workspacePath)
			return
		}

		// Execute stage with progress reporting
		progressFunc := func(percent int, message string) {
			// Calculate overall progress
			overallPercent := (i*100 + percent) / len(w.manager.stages)
			if err := w.manager.buildJobRepo.UpdateStage(job.ID, string(stageName), overallPercent); err != nil {
				log.Warn("Failed to update build stage progress", "build_id", job.ID, "error", err)
			}

			if message != "" {
				if err := w.manager.buildJobRepo.AppendLog(job.ID, string(stageName), "info", message); err != nil {
					log.Warn("Failed to append build log", "build_id", job.ID, "error", err)
				}
			}
		}

		if err := stage.Execute(jobCtx, sc, progressFunc); err != nil {
			if logErr := w.manager.buildJobRepo.AppendLog(job.ID, string(stageName), "error",
				fmt.Sprintf("Stage execution failed: %v", err)); logErr != nil {
				log.Warn("Failed to append build log", "build_id", job.ID, "error", logErr)
			}
			if logErr := w.manager.buildJobRepo.MarkStageFailed(job.ID, stageName, err.Error()); logErr != nil {
				log.Warn("Failed to mark stage failed", "build_id", job.ID, "stage", stageName, "error", logErr)
			}
			w.handleFailure(job, fmt.Sprintf("Stage %s failed: %v", stageName, err), string(stageName))
			w.cleanup(workspacePath)
			return
		}

		// Mark stage as completed
		durationMs := time.Since(stageStart).Milliseconds()
		if err := w.manager.buildJobRepo.MarkStageCompleted(job.ID, stageName, durationMs); err != nil {
			log.Warn("Failed to mark stage completed", "build_id", job.ID, "stage", stageName, "error", err)
		}

		if err := w.manager.buildJobRepo.AppendLog(job.ID, string(stageName), "info",
			fmt.Sprintf("Stage completed in %dms", durationMs)); err != nil {
			log.Warn("Failed to append build log", "build_id", job.ID, "error", err)
		}
	}

	// Build completed successfully
	log.Info("Build completed successfully",
		"worker_id", w.id,
		"build_id", job.ID,
		"artifact_path", sc.ArtifactPath,
		"artifact_size", sc.ArtifactSize,
	)

	// Mark job completed with artifact info from package stage
	if err := w.manager.buildJobRepo.MarkCompleted(job.ID, sc.ArtifactPath, sc.ArtifactChecksum, sc.ArtifactSize); err != nil {
		log.Error("Failed to mark build completed", "build_id", job.ID, "error", err)
	}

	// Update distribution status to ready
	if err := w.manager.distRepo.UpdateStatus(job.DistributionID, db.StatusReady, ""); err != nil {
		log.Warn("Failed to update distribution status to ready", "distribution_id", job.DistributionID, "error", err)
	}

	if err := w.manager.buildJobRepo.AppendLog(job.ID, "", "info",
		fmt.Sprintf("Build completed successfully: %s (%d bytes)", sc.ArtifactPath, sc.ArtifactSize)); err != nil {
		log.Warn("Failed to append build log", "build_id", job.ID, "error", err)
	}

	// Cleanup workspace only if clear_cache is enabled
	if job.ClearCache {
		if err := w.manager.buildJobRepo.AppendLog(job.ID, "", "info", "Clearing local build cache as requested"); err != nil {
			log.Warn("Failed to append build log", "build_id", job.ID, "error", err)
		}
		w.cleanup(workspacePath)
	} else {
		log.Debug("Keeping local build cache", "build_id", job.ID, "workspace", workspacePath)
	}
}

// handleFailure marks a build job as failed
func (w *Worker) handleFailure(job *db.BuildJob, errorMsg, errorStage string) {
	log.Error("Build job failed",
		"worker_id", w.id,
		"build_id", job.ID,
		"error", errorMsg,
		"stage", errorStage,
	)

	if err := w.manager.buildJobRepo.MarkFailed(job.ID, errorMsg, errorStage); err != nil {
		log.Error("Failed to mark build as failed", "build_id", job.ID, "error", err)
	}

	// Update distribution status to failed
	if err := w.manager.distRepo.UpdateStatus(job.DistributionID, db.StatusFailed, errorMsg); err != nil {
		log.Warn("Failed to update distribution status to failed", "distribution_id", job.DistributionID, "error", err)
	}
}

// cleanup removes the build workspace directory
func (w *Worker) cleanup(workspacePath string) {
	if workspacePath == "" {
		return
	}
	if err := os.RemoveAll(workspacePath); err != nil {
		log.Warn("Failed to cleanup workspace", "path", workspacePath, "error", err)
	}
}
