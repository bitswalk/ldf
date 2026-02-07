package build

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// CompileStage compiles the kernel inside a container or via chroot
type CompileStage struct{}

// NewCompileStage creates a new compile stage
func NewCompileStage() *CompileStage {
	return &CompileStage{}
}

// Name returns the stage name
func (s *CompileStage) Name() db.BuildStageName {
	return db.StageCompile
}

// Validate checks whether this stage can run
func (s *CompileStage) Validate(ctx context.Context, sc *StageContext) error {
	if len(sc.Components) == 0 {
		return fmt.Errorf("no components resolved")
	}

	// Find kernel component
	kernel := s.findKernelComponent(sc.Components)
	if kernel == nil {
		return fmt.Errorf("kernel component not found")
	}
	if kernel.LocalPath == "" {
		return fmt.Errorf("kernel source path not set - prepare stage must run first")
	}

	// Check config file exists
	configPath := filepath.Join(sc.ConfigDir, ".config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("kernel config not found at %s - prepare stage must run first", configPath)
	}

	return nil
}

// Execute compiles the kernel
func (s *CompileStage) Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error {
	progress(0, "Starting kernel compilation")

	kernel := s.findKernelComponent(sc.Components)
	if kernel == nil {
		return fmt.Errorf("kernel component not found")
	}

	configPath := filepath.Join(sc.ConfigDir, ".config")
	outputDir := filepath.Join(sc.WorkspacePath, "kernel-output")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create kernel output directory: %w", err)
	}

	// Parse the config file to determine the mode
	configMode, err := s.parseConfigMode(configPath)
	if err != nil {
		log.Warn("Could not parse config mode, assuming defconfig", "error", err)
		configMode = string(db.KernelConfigModeDefconfig)
	}

	// Determine cross-compile prefix from build environment
	crossCompile := s.getCrossCompilePrefix(sc)
	makeArch := s.getMakeArch(sc)

	progress(5, fmt.Sprintf("Config mode: %s, arch: %s", configMode, sc.TargetArch))

	// Check if executor is available
	executor := sc.Executor
	if executor == nil {
		return fmt.Errorf("build executor not available - no executor configured")
	}
	if !executor.IsAvailable() {
		return fmt.Errorf("build executor not available - please install %s", executor.RuntimeType())
	}

	// Route to appropriate execution method based on runtime type
	if executor.RuntimeType().IsContainerRuntime() {
		return s.executeInContainer(ctx, sc, kernel, configPath, configMode, outputDir, makeArch, crossCompile, progress)
	}

	return s.executeDirect(ctx, sc, kernel, configPath, configMode, outputDir, makeArch, crossCompile, progress)
}

// executeInContainer runs compilation inside an OCI container
func (s *CompileStage) executeInContainer(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, configPath, configMode, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	progress(10, "Preparing container build environment")

	// Generate the build script based on config mode
	buildScript := s.generateBuildScript(configMode, makeArch, crossCompile)
	buildScriptPath := filepath.Join(sc.WorkspacePath, "scripts", "compile-kernel.sh")
	if err := os.WriteFile(buildScriptPath, []byte(buildScript), 0755); err != nil {
		return fmt.Errorf("failed to write build script: %w", err)
	}

	// Generate board profile kernel overlay file if present (for container to consume)
	if sc.BoardProfile != nil && len(sc.BoardProfile.Config.KernelOverlay) > 0 {
		overlayPath := filepath.Join(sc.ConfigDir, ".config.board-overlay")
		if err := GenerateConfigFragment(sc.BoardProfile.Config.KernelOverlay, overlayPath); err != nil {
			return fmt.Errorf("failed to generate board kernel overlay: %w", err)
		}
	}

	// Setup container mounts — mount the entire config dir so overlay files are accessible
	mounts := []Mount{
		{Source: kernel.LocalPath, Target: "/src/kernel", ReadOnly: false},
		{Source: sc.ConfigDir, Target: "/config", ReadOnly: true},
		{Source: outputDir, Target: "/output", ReadOnly: false},
		{Source: filepath.Join(sc.WorkspacePath, "scripts"), Target: "/scripts", ReadOnly: true},
	}

	// Create a log file for build output
	logPath := filepath.Join(sc.WorkspacePath, "logs", "kernel-compile.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Create a progress-tracking writer
	progressWriter := &buildProgressWriter{
		progress:    progress,
		basePercent: 10,
		maxPercent:  95,
		logFile:     logFile,
		logWriter:   sc.LogWriter,
	}

	progress(15, "Running kernel compilation in container")

	// Use BuildEnv container image if available, otherwise fall back to default
	containerImage := sc.Executor.DefaultImage()
	var platformFlag string
	if sc.BuildEnv != nil {
		containerImage = sc.BuildEnv.ContainerImage
		platformFlag = sc.BuildEnv.ContainerPlatformFlag
	}

	opts := ContainerRunOpts{
		Image:    containerImage,
		Mounts:   mounts,
		WorkDir:  "/src/kernel",
		Platform: platformFlag,
		Env: map[string]string{
			"ARCH":          makeArch,
			"CROSS_COMPILE": crossCompile,
			"NPROC":         "0", // 0 means auto-detect
		},
		Command: []string{"/bin/bash", "/scripts/compile-kernel.sh"},
		Stdout:  progressWriter,
		Stderr:  progressWriter,
	}

	if err := sc.Executor.Run(ctx, opts); err != nil {
		return fmt.Errorf("kernel compilation failed: %w", err)
	}

	progress(90, "Verifying build outputs")

	// Verify kernel image was built
	kernelImage := filepath.Join(outputDir, "boot", "vmlinuz")
	if _, err := os.Stat(kernelImage); os.IsNotExist(err) {
		return fmt.Errorf("kernel image not found at %s", kernelImage)
	}

	// Compile device trees if board profile specifies them
	if sc.BoardProfile != nil && len(sc.BoardProfile.Config.DeviceTrees) > 0 {
		progress(92, "Compiling device tree blobs")
		if err := s.compileDeviceTrees(ctx, sc, kernel, outputDir, makeArch, crossCompile, progress); err != nil {
			return fmt.Errorf("device tree compilation failed: %w", err)
		}
	}

	progress(100, "Kernel compilation complete")
	return nil
}

// compileDeviceTrees compiles device tree sources specified by the board profile (container mode)
func (s *CompileStage) compileDeviceTrees(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	dtbScript := s.generateDTBBuildScript(sc.BoardProfile.Config.DeviceTrees, makeArch, crossCompile)
	dtbScriptPath := filepath.Join(sc.WorkspacePath, "scripts", "compile-dtbs.sh")
	if err := os.WriteFile(dtbScriptPath, []byte(dtbScript), 0755); err != nil {
		return fmt.Errorf("failed to write DTB build script: %w", err)
	}

	mounts := []Mount{
		{Source: kernel.LocalPath, Target: "/src/kernel", ReadOnly: false},
		{Source: outputDir, Target: "/output", ReadOnly: false},
		{Source: filepath.Join(sc.WorkspacePath, "scripts"), Target: "/scripts", ReadOnly: true},
	}

	logPath := filepath.Join(sc.WorkspacePath, "logs", "dtb-compile.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create DTB log file: %w", err)
	}
	defer logFile.Close()

	// Use BuildEnv container image if available, otherwise fall back to default
	dtbContainerImage := sc.Executor.DefaultImage()
	var dtbPlatformFlag string
	if sc.BuildEnv != nil {
		dtbContainerImage = sc.BuildEnv.ContainerImage
		dtbPlatformFlag = sc.BuildEnv.ContainerPlatformFlag
	}

	opts := ContainerRunOpts{
		Image:    dtbContainerImage,
		Mounts:   mounts,
		WorkDir:  "/src/kernel",
		Platform: dtbPlatformFlag,
		Env: map[string]string{
			"ARCH":          makeArch,
			"CROSS_COMPILE": crossCompile,
		},
		Command: []string{"/bin/bash", "/scripts/compile-dtbs.sh"},
		Stdout:  logFile,
		Stderr:  logFile,
	}

	progress(94, fmt.Sprintf("Compiling %d device tree(s)", len(sc.BoardProfile.Config.DeviceTrees)))

	if err := sc.Executor.Run(ctx, opts); err != nil {
		return fmt.Errorf("DTB compilation failed: %w", err)
	}

	progress(97, "Device tree compilation complete")
	return nil
}

// generateDTBBuildScript creates a script to compile device tree blobs (container mode)
func (s *CompileStage) generateDTBBuildScript(deviceTrees []db.DeviceTreeSpec, makeArch, crossCompile string) string {
	var sb strings.Builder
	sb.WriteString(`#!/bin/bash
set -e

echo "=== LDF Device Tree Build ==="
cd /src/kernel
mkdir -p /output/boot/dtbs
`)

	for _, dt := range deviceTrees {
		// Compile the main DTB
		dtbPath := strings.TrimSuffix(dt.Source, ".dts") + ".dtb"
		sb.WriteString(fmt.Sprintf(`
echo "Building DTB: %s"
make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" %s
cp %s /output/boot/dtbs/
`, dt.Source, dtbPath, dtbPath))

		// Compile overlays if specified
		if len(dt.Overlays) > 0 {
			sb.WriteString("\nmkdir -p /output/boot/dtbs/overlays\n")
			for _, overlay := range dt.Overlays {
				dtboPath := strings.TrimSuffix(overlay, ".dts") + ".dtbo"
				sb.WriteString(fmt.Sprintf(`
echo "Building DT overlay: %s"
make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" %s
cp %s /output/boot/dtbs/overlays/
`, overlay, dtboPath, dtboPath))
			}
		}
	}

	sb.WriteString(`
echo ""
echo "=== Device tree build complete ==="
ls -la /output/boot/dtbs/
`)
	return sb.String()
}

// executeDirect runs kernel compilation directly on the host using sequential
// executor.Run calls with real paths. No bash script generation needed.
func (s *CompileStage) executeDirect(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, configPath, configMode, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	progress(10, "Preparing direct build environment")

	kernelDir := kernel.LocalPath
	nproc := fmt.Sprintf("%d", runtime.NumCPU())

	// Common make arguments
	makeEnv := map[string]string{
		"ARCH":          makeArch,
		"CROSS_COMPILE": crossCompile,
	}

	// Create a log file for build output
	logPath := filepath.Join(sc.WorkspacePath, "logs", "kernel-compile.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	progressWriter := &buildProgressWriter{
		progress:    progress,
		basePercent: 10,
		maxPercent:  90,
		logFile:     logFile,
		logWriter:   sc.LogWriter,
	}

	// Helper to run a make command in the kernel source directory
	runMake := func(args ...string) error {
		cmd := append([]string{"make"}, args...)
		return sc.Executor.Run(ctx, ContainerRunOpts{
			WorkDir: kernelDir,
			Env:     makeEnv,
			Command: cmd,
			Stdout:  progressWriter,
			Stderr:  progressWriter,
		})
	}

	// Step 1: Generate kernel config based on mode
	progress(12, fmt.Sprintf("Generating kernel config (%s)", configMode))

	archFlag := fmt.Sprintf("ARCH=%s", makeArch)
	crossFlag := fmt.Sprintf("CROSS_COMPILE=%s", crossCompile)

	if configMode == string(db.KernelConfigModeCustom) {
		// Full config: copy directly, stripping LDF metadata headers
		if err := s.copyCustomConfig(kernelDir, configPath); err != nil {
			return fmt.Errorf("failed to copy custom config: %w", err)
		}
	} else {
		// Fragment (defconfig or options): run defconfig then merge fragment on top
		defconfigName := GetDefconfigName(sc.BoardProfile, sc.TargetArch)
		defconfigTarget := defconfigName
		if defconfigName != "defconfig" {
			defconfigTarget = defconfigName + "_defconfig"
		}
		if err := runMake(archFlag, crossFlag, defconfigTarget); err != nil {
			return fmt.Errorf("defconfig generation failed: %w", err)
		}

		// Merge the stored config fragment (recommended + user options)
		if err := s.applyKconfigOptions(kernelDir, configPath); err != nil {
			return fmt.Errorf("failed to apply config fragment: %w", err)
		}

		// Apply board profile kernel overlay on top (if present)
		if sc.BoardProfile != nil && len(sc.BoardProfile.Config.KernelOverlay) > 0 {
			log.Info("Applying board profile kernel overlay",
				"board", sc.BoardProfile.Name,
				"options", len(sc.BoardProfile.Config.KernelOverlay))
			overlayPath := filepath.Join(sc.ConfigDir, ".config.board-overlay")
			if err := GenerateConfigFragment(sc.BoardProfile.Config.KernelOverlay, overlayPath); err != nil {
				return fmt.Errorf("failed to generate board kernel overlay: %w", err)
			}
			if err := s.applyKconfigOptions(kernelDir, overlayPath); err != nil {
				return fmt.Errorf("failed to apply board kernel overlay: %w", err)
			}
		}
	}

	// Step 2: Resolve config dependencies
	progress(20, "Resolving config dependencies")
	if err := runMake(archFlag, crossFlag, "olddefconfig"); err != nil {
		return fmt.Errorf("olddefconfig failed: %w", err)
	}

	// Step 3: Build kernel image
	kernelTarget := "Image"
	if makeArch == "x86" || makeArch == "x86_64" {
		kernelTarget = "bzImage"
	}
	progress(25, fmt.Sprintf("Building %s", kernelTarget))
	if err := runMake(archFlag, crossFlag, fmt.Sprintf("-j%s", nproc), kernelTarget); err != nil {
		return fmt.Errorf("kernel image build failed: %w", err)
	}

	// Step 4: Build modules
	progress(60, "Building modules")
	if err := runMake(archFlag, crossFlag, fmt.Sprintf("-j%s", nproc), "modules"); err != nil {
		return fmt.Errorf("module build failed: %w", err)
	}

	// Step 5: Install modules to output directory
	progress(75, "Installing modules")
	modInstallPath := fmt.Sprintf("INSTALL_MOD_PATH=%s/modules", outputDir)
	if err := runMake(archFlag, crossFlag, modInstallPath, "modules_install"); err != nil {
		return fmt.Errorf("module install failed: %w", err)
	}

	// Step 6: Copy build artifacts to output directory
	progress(85, "Copying build artifacts")

	bootDir := filepath.Join(outputDir, "boot")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("failed to create boot output directory: %w", err)
	}

	// Copy kernel image
	var kernelImageSrc string
	if makeArch == "x86" || makeArch == "x86_64" {
		kernelImageSrc = filepath.Join(kernelDir, "arch/x86/boot/bzImage")
	} else {
		kernelImageSrc = filepath.Join(kernelDir, "arch", makeArch, "boot/Image")
	}
	if err := copyFile(kernelImageSrc, filepath.Join(bootDir, "vmlinuz")); err != nil {
		return fmt.Errorf("failed to copy kernel image: %w", err)
	}

	// Copy System.map
	if err := copyFile(filepath.Join(kernelDir, "System.map"), filepath.Join(bootDir, "System.map")); err != nil {
		log.Warn("Failed to copy System.map", "error", err)
	}

	// Copy .config
	if err := copyFile(filepath.Join(kernelDir, ".config"), filepath.Join(bootDir, "config")); err != nil {
		log.Warn("Failed to copy kernel config", "error", err)
	}

	progress(90, "Verifying build outputs")

	// Verify kernel image was built
	kernelImage := filepath.Join(bootDir, "vmlinuz")
	if _, err := os.Stat(kernelImage); os.IsNotExist(err) {
		return fmt.Errorf("kernel image not found at %s", kernelImage)
	}

	// Compile device trees if board profile specifies them
	if sc.BoardProfile != nil && len(sc.BoardProfile.Config.DeviceTrees) > 0 {
		progress(92, "Compiling device tree blobs")
		if err := s.compileDeviceTreesDirect(ctx, sc, kernel, outputDir, makeArch, crossCompile, nproc, progress); err != nil {
			return fmt.Errorf("device tree compilation failed: %w", err)
		}
	}

	progress(100, "Kernel compilation complete")
	return nil
}

// applyKconfigOptions reads LDF config options from configPath and merges them
// into the kernel's .config file using pure Go text manipulation. This avoids
// shelling out to scripts/config and keeps all file operations in Go.
func (s *CompileStage) applyKconfigOptions(kernelDir, configPath string) error {
	// Parse desired options from the LDF config file
	options := make(map[string]string) // CONFIG_FOO -> "y"|"m"|"n"|value
	ldfFile, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer ldfFile.Close()

	scanner := bufio.NewScanner(ldfFile)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments, empty lines, and LDF metadata
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "LDF_") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}

		key := line[:idx]
		value := strings.Trim(line[idx+1:], "\"")

		if !strings.HasPrefix(key, "CONFIG_") {
			continue
		}

		options[key] = value
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if len(options) == 0 {
		return nil
	}

	// Read the kernel's current .config generated by defconfig
	kconfigPath := filepath.Join(kernelDir, ".config")
	data, err := os.ReadFile(kconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kernel .config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	seen := make(map[string]bool)

	// Pass 1: update existing lines that reference any of our options
	for i, line := range lines {
		for key, value := range options {
			// Match "CONFIG_FOO=..." or "# CONFIG_FOO is not set"
			if strings.HasPrefix(line, key+"=") || line == "# "+key+" is not set" {
				seen[key] = true
				switch value {
				case "n":
					lines[i] = "# " + key + " is not set"
				default:
					lines[i] = key + "=" + value
				}
				break
			}
		}
	}

	// Pass 2: append options that weren't already present in .config
	for key, value := range options {
		if seen[key] {
			continue
		}
		switch value {
		case "n":
			lines = append(lines, "# "+key+" is not set")
		default:
			lines = append(lines, key+"="+value)
		}
	}

	return os.WriteFile(kconfigPath, []byte(strings.Join(lines, "\n")), 0644)
}

// copyCustomConfig copies a user-provided kernel config, stripping LDF metadata lines.
func (s *CompileStage) copyCustomConfig(kernelDir, configPath string) error {
	input, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var output strings.Builder
	for _, line := range strings.Split(string(input), "\n") {
		if strings.HasPrefix(line, "# LDF") || strings.HasPrefix(line, "LDF_") {
			continue
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}

	return os.WriteFile(filepath.Join(kernelDir, ".config"), []byte(output.String()), 0644)
}

// compileDeviceTreesDirect compiles device trees directly on the host
// using sequential executor.Run calls.
func (s *CompileStage) compileDeviceTreesDirect(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, outputDir, makeArch, crossCompile, nproc string, progress ProgressFunc) error {
	kernelDir := kernel.LocalPath
	dtbsDir := filepath.Join(outputDir, "boot", "dtbs")

	if err := os.MkdirAll(dtbsDir, 0755); err != nil {
		return fmt.Errorf("failed to create dtbs directory: %w", err)
	}

	archFlag := fmt.Sprintf("ARCH=%s", makeArch)
	crossFlag := fmt.Sprintf("CROSS_COMPILE=%s", crossCompile)

	logPath := filepath.Join(sc.WorkspacePath, "logs", "dtb-compile.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create DTB log file: %w", err)
	}
	defer logFile.Close()

	for i, dt := range sc.BoardProfile.Config.DeviceTrees {
		dtbTarget := strings.TrimSuffix(dt.Source, ".dts") + ".dtb"

		progress(94, fmt.Sprintf("Building DTB %d/%d: %s", i+1, len(sc.BoardProfile.Config.DeviceTrees), dt.Source))

		// Build the DTB
		if err := sc.Executor.Run(ctx, ContainerRunOpts{
			WorkDir: kernelDir,
			Env: map[string]string{
				"ARCH":          makeArch,
				"CROSS_COMPILE": crossCompile,
			},
			Command: []string{"make", archFlag, crossFlag, dtbTarget},
			Stdout:  logFile,
			Stderr:  logFile,
		}); err != nil {
			return fmt.Errorf("DTB build failed for %s: %w", dt.Source, err)
		}

		// Copy DTB to output
		if err := copyFile(filepath.Join(kernelDir, dtbTarget), filepath.Join(dtbsDir, filepath.Base(dtbTarget))); err != nil {
			return fmt.Errorf("failed to copy DTB %s: %w", dtbTarget, err)
		}

		// Build overlays if specified
		if len(dt.Overlays) > 0 {
			overlaysDir := filepath.Join(dtbsDir, "overlays")
			if err := os.MkdirAll(overlaysDir, 0755); err != nil {
				return fmt.Errorf("failed to create overlays directory: %w", err)
			}

			for _, overlay := range dt.Overlays {
				dtboTarget := strings.TrimSuffix(overlay, ".dts") + ".dtbo"

				if err := sc.Executor.Run(ctx, ContainerRunOpts{
					WorkDir: kernelDir,
					Env: map[string]string{
						"ARCH":          makeArch,
						"CROSS_COMPILE": crossCompile,
					},
					Command: []string{"make", archFlag, crossFlag, dtboTarget},
					Stdout:  logFile,
					Stderr:  logFile,
				}); err != nil {
					return fmt.Errorf("DT overlay build failed for %s: %w", overlay, err)
				}

				if err := copyFile(filepath.Join(kernelDir, dtboTarget), filepath.Join(overlaysDir, filepath.Base(dtboTarget))); err != nil {
					return fmt.Errorf("failed to copy DT overlay %s: %w", dtboTarget, err)
				}
			}
		}
	}

	progress(97, "Device tree compilation complete")
	return nil
}

// generateBuildScript creates the kernel build script for container execution
func (s *CompileStage) generateBuildScript(configMode, makeArch, crossCompile string) string {
	script := `#!/bin/bash
set -e

echo "=== LDF Kernel Build ==="
echo "Architecture: ${ARCH:-x86}"
echo "Cross-compile: ${CROSS_COMPILE:-none}"
echo "Config mode: ` + configMode + `"
echo ""

cd /src/kernel

# Determine number of parallel jobs
if [ "${NPROC}" = "0" ] || [ -z "${NPROC}" ]; then
    NPROC=$(nproc)
fi
echo "Using ${NPROC} parallel jobs"

`

	// Config handling: two paths — custom (full config) vs fragment (defconfig/options)
	if configMode == string(db.KernelConfigModeCustom) {
		script += `
# Custom mode: copy user-provided full config, stripping LDF metadata
echo "Using custom kernel config..."
grep -v "^# LDF" /config/.config | grep -v "^LDF_" > .config || true
`
	} else {
		script += `
# Fragment mode: run defconfig then merge stored config fragment
echo "Generating defconfig for ${ARCH}..."
if [ "${ARCH}" = "x86" ] || [ "${ARCH}" = "x86_64" ]; then
    make ARCH=x86 CROSS_COMPILE="${CROSS_COMPILE}" x86_64_defconfig
else
    make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" defconfig
fi

# Merge config fragment (recommended + user options) on top of defconfig
echo "Applying config fragment..."
while IFS= read -r line; do
    # Skip comments, empty lines, and LDF metadata
    [[ "$line" =~ ^# ]] && continue
    [[ -z "$line" ]] && continue
    [[ "$line" =~ ^LDF_ ]] && continue

    if [[ "$line" =~ ^(CONFIG_[A-Z0-9_]+)=(.*)$ ]]; then
        KEY="${BASH_REMATCH[1]}"
        VALUE="${BASH_REMATCH[2]}"
        VALUE="${VALUE%\"}"
        VALUE="${VALUE#\"}"

        case "$VALUE" in
            y) ./scripts/config --enable "$KEY" ;;
            m) ./scripts/config --module "$KEY" ;;
            n) ./scripts/config --disable "$KEY" ;;
            *) ./scripts/config --set-str "$KEY" "$VALUE" ;;
        esac
    elif [[ "$line" =~ ^"# "(CONFIG_[A-Z0-9_]+)" is not set"$ ]]; then
        KEY="${BASH_REMATCH[1]}"
        ./scripts/config --disable "$KEY"
    fi
done < /config/.config

# Apply board profile kernel overlay if present
if [ -f /config/.config.board-overlay ]; then
    echo "Applying board profile kernel overlay..."
    while IFS= read -r line; do
        [[ "$line" =~ ^# ]] && continue
        [[ -z "$line" ]] && continue

        if [[ "$line" =~ ^(CONFIG_[A-Z0-9_]+)=(.*)$ ]]; then
            KEY="${BASH_REMATCH[1]}"
            VALUE="${BASH_REMATCH[2]}"
            VALUE="${VALUE%\"}"
            VALUE="${VALUE#\"}"

            case "$VALUE" in
                y) ./scripts/config --enable "$KEY" ;;
                m) ./scripts/config --module "$KEY" ;;
                n) ./scripts/config --disable "$KEY" ;;
                *) ./scripts/config --set-str "$KEY" "$VALUE" ;;
            esac
        elif [[ "$line" =~ ^"# "(CONFIG_[A-Z0-9_]+)" is not set"$ ]]; then
            KEY="${BASH_REMATCH[1]}"
            ./scripts/config --disable "$KEY"
        fi
    done < /config/.config.board-overlay
fi
`
	}

	// Common build steps
	script += `
# Update config to resolve dependencies
echo "Resolving config dependencies..."
make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" olddefconfig

echo ""
echo "=== Starting kernel build ==="
echo ""

# Build kernel image
if [ "${ARCH}" = "x86" ] || [ "${ARCH}" = "x86_64" ]; then
    echo "Building bzImage..."
    make ARCH=x86 CROSS_COMPILE="${CROSS_COMPILE}" -j${NPROC} bzImage
else
    echo "Building Image..."
    make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" -j${NPROC} Image
fi

# Build modules
echo ""
echo "Building modules..."
make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" -j${NPROC} modules

# Install modules
echo ""
echo "Installing modules..."
make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" INSTALL_MOD_PATH=/output/modules modules_install

# Copy kernel image
echo ""
echo "Copying kernel image..."
mkdir -p /output/boot
if [ "${ARCH}" = "x86" ] || [ "${ARCH}" = "x86_64" ]; then
    cp arch/x86/boot/bzImage /output/boot/vmlinuz
else
    cp arch/${ARCH}/boot/Image /output/boot/vmlinuz
fi

# Copy System.map and config
cp System.map /output/boot/
cp .config /output/boot/config

echo ""
echo "=== Kernel build complete ==="
ls -la /output/boot/
`

	return script
}

// parseConfigMode reads the LDF_CONFIG_MODE from the config file
func (s *CompileStage) parseConfigMode(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "LDF_CONFIG_MODE=") {
			return strings.TrimPrefix(line, "LDF_CONFIG_MODE="), nil
		}
	}

	return "", fmt.Errorf("LDF_CONFIG_MODE not found in config")
}

// getCrossCompilePrefix returns the cross-compile prefix from BuildEnvironment,
// falling back to toolchain registry lookup if BuildEnv is not populated.
func (s *CompileStage) getCrossCompilePrefix(sc *StageContext) string {
	if sc.BuildEnv != nil {
		return sc.BuildEnv.Toolchain.CrossCompilePrefix
	}
	tc, err := GetToolchain(DetectHostArch(), sc.TargetArch)
	if err != nil {
		return ""
	}
	return tc.CrossCompilePrefix
}

// getMakeArch returns the ARCH value for make from BuildEnvironment,
// falling back to toolchain registry lookup if BuildEnv is not populated.
func (s *CompileStage) getMakeArch(sc *StageContext) string {
	if sc.BuildEnv != nil {
		return sc.BuildEnv.Toolchain.MakeArch
	}
	tc, err := GetToolchain(DetectHostArch(), sc.TargetArch)
	if err != nil {
		return "x86"
	}
	return tc.MakeArch
}

// findKernelComponent finds the kernel component in the resolved list
func (s *CompileStage) findKernelComponent(components []ResolvedComponent) *ResolvedComponent {
	for i := range components {
		if strings.Contains(strings.ToLower(components[i].Component.Name), "kernel") {
			return &components[i]
		}
	}
	return nil
}

// buildProgressWriter parses build output and updates progress
type buildProgressWriter struct {
	progress    ProgressFunc
	basePercent int
	maxPercent  int
	logFile     *os.File
	logWriter   io.Writer
	lastPercent int
}

func (w *buildProgressWriter) Write(p []byte) (n int, err error) {
	// Write to log file
	if w.logFile != nil {
		if _, err := w.logFile.Write(p); err != nil {
			log.Warn("Failed to write to build log file", "error", err)
		}
	}

	// Write to log writer if available
	if w.logWriter != nil {
		if _, err := w.logWriter.Write(p); err != nil {
			log.Warn("Failed to write to build log writer", "error", err)
		}
	}

	// Parse output for progress indicators
	text := string(p)
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		// Look for make progress indicators like [  1%], [ 10%], [100%]
		re := regexp.MustCompile(`\[\s*(\d+)%\]`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			if pct, err := strconv.Atoi(matches[1]); err == nil {
				// Scale to our progress range
				scaledPct := w.basePercent + (pct * (w.maxPercent - w.basePercent) / 100)
				if scaledPct > w.lastPercent {
					w.lastPercent = scaledPct
					w.progress(scaledPct, fmt.Sprintf("Compiling kernel... %d%%", pct))
				}
			}
		}

		// Also look for stage markers
		if strings.Contains(line, "Building bzImage") || strings.Contains(line, "Building Image") {
			w.progress(w.basePercent+20, "Building kernel image...")
		} else if strings.Contains(line, "Building modules") {
			w.progress(w.basePercent+60, "Building modules...")
		} else if strings.Contains(line, "Installing modules") {
			w.progress(w.basePercent+80, "Installing modules...")
		}
	}

	return len(p), nil
}
