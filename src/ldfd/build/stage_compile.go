package build

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// CompileStage compiles the kernel inside a container or via chroot
type CompileStage struct {
	executor Executor
}

// NewCompileStage creates a new compile stage
func NewCompileStage(executor Executor) *CompileStage {
	return &CompileStage{executor: executor}
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
	if s.executor == nil || !s.executor.IsAvailable() {
		return fmt.Errorf("build executor not available - please install %s", s.executor.RuntimeType())
	}

	// Route to appropriate execution method based on runtime type
	if s.executor.RuntimeType().IsContainerRuntime() {
		return s.executeInContainer(ctx, sc, kernel, configPath, configMode, outputDir, makeArch, crossCompile, progress)
	}

	return s.executeDirect(ctx, sc, kernel, configPath, configMode, outputDir, makeArch, crossCompile, progress)
}

// executeInContainer runs compilation inside a Podman container
func (s *CompileStage) executeInContainer(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, configPath, configMode, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	progress(10, "Preparing container build environment")

	// Generate the build script based on config mode
	buildScript := s.generateBuildScript(configMode, makeArch, crossCompile)
	buildScriptPath := filepath.Join(sc.WorkspacePath, "scripts", "compile-kernel.sh")
	if err := os.WriteFile(buildScriptPath, []byte(buildScript), 0755); err != nil {
		return fmt.Errorf("failed to write build script: %w", err)
	}

	// Setup container mounts
	mounts := []Mount{
		{Source: kernel.LocalPath, Target: "/src/kernel", ReadOnly: false},
		{Source: configPath, Target: "/config/.config", ReadOnly: true},
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
	containerImage := s.executor.DefaultImage()
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

	if err := s.executor.Run(ctx, opts); err != nil {
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

// compileDeviceTrees compiles device tree sources specified by the board profile
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
	dtbContainerImage := s.executor.DefaultImage()
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

	if err := s.executor.Run(ctx, opts); err != nil {
		return fmt.Errorf("DTB compilation failed: %w", err)
	}

	progress(97, "Device tree compilation complete")
	return nil
}

// generateDTBBuildScript creates a script to compile device tree blobs
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

// executeDirect runs compilation directly on the host via chroot executor
func (s *CompileStage) executeDirect(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, configPath, configMode, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	progress(10, "Preparing direct build environment")

	// Generate the build script
	buildScript := s.generateBuildScript(configMode, makeArch, crossCompile)
	buildScriptPath := filepath.Join(sc.WorkspacePath, "scripts", "compile-kernel.sh")
	if err := os.WriteFile(buildScriptPath, []byte(buildScript), 0755); err != nil {
		return fmt.Errorf("failed to write build script: %w", err)
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
		maxPercent:  95,
		logFile:     logFile,
		logWriter:   sc.LogWriter,
	}

	progress(15, "Running kernel compilation directly on host")

	// Set up mounts and environment for the chroot executor
	mounts := []Mount{
		{Source: kernel.LocalPath, Target: "/src/kernel", ReadOnly: false},
		{Source: configPath, Target: "/config/.config", ReadOnly: true},
		{Source: outputDir, Target: "/output", ReadOnly: false},
		{Source: filepath.Join(sc.WorkspacePath, "scripts"), Target: "/scripts", ReadOnly: true},
	}

	opts := ContainerRunOpts{
		Mounts:  mounts,
		WorkDir: kernel.LocalPath,
		Env: map[string]string{
			"ARCH":          makeArch,
			"CROSS_COMPILE": crossCompile,
			"NPROC":         "0",
		},
		Command: []string{"/bin/bash", buildScriptPath},
		Stdout:  progressWriter,
		Stderr:  progressWriter,
	}

	if err := s.executor.Run(ctx, opts); err != nil {
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
		if err := s.compileDeviceTreesDirect(ctx, sc, kernel, outputDir, makeArch, crossCompile, progress); err != nil {
			return fmt.Errorf("device tree compilation failed: %w", err)
		}
	}

	progress(100, "Kernel compilation complete")
	return nil
}

// compileDeviceTreesDirect compiles device trees directly on the host
func (s *CompileStage) compileDeviceTreesDirect(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, outputDir, makeArch, crossCompile string, progress ProgressFunc) error {
	dtbScript := s.generateDTBBuildScript(sc.BoardProfile.Config.DeviceTrees, makeArch, crossCompile)
	dtbScriptPath := filepath.Join(sc.WorkspacePath, "scripts", "compile-dtbs.sh")
	if err := os.WriteFile(dtbScriptPath, []byte(dtbScript), 0755); err != nil {
		return fmt.Errorf("failed to write DTB build script: %w", err)
	}

	logPath := filepath.Join(sc.WorkspacePath, "logs", "dtb-compile.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create DTB log file: %w", err)
	}
	defer logFile.Close()

	mounts := []Mount{
		{Source: kernel.LocalPath, Target: "/src/kernel", ReadOnly: false},
		{Source: outputDir, Target: "/output", ReadOnly: false},
		{Source: filepath.Join(sc.WorkspacePath, "scripts"), Target: "/scripts", ReadOnly: true},
	}

	progress(94, fmt.Sprintf("Compiling %d device tree(s)", len(sc.BoardProfile.Config.DeviceTrees)))

	opts := ContainerRunOpts{
		Mounts:  mounts,
		WorkDir: kernel.LocalPath,
		Env: map[string]string{
			"ARCH":          makeArch,
			"CROSS_COMPILE": crossCompile,
		},
		Command: []string{"/bin/bash", dtbScriptPath},
		Stdout:  logFile,
		Stderr:  logFile,
	}

	if err := s.executor.Run(ctx, opts); err != nil {
		return fmt.Errorf("DTB compilation failed: %w", err)
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

	// Add config handling based on mode
	switch configMode {
	case string(db.KernelConfigModeDefconfig):
		script += `
# Generate default config for architecture
echo "Generating defconfig for ${ARCH}..."
if [ "${ARCH}" = "x86" ] || [ "${ARCH}" = "x86_64" ]; then
    make ARCH=x86 CROSS_COMPILE="${CROSS_COMPILE}" x86_64_defconfig
else
    make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" defconfig
fi

# Apply recommended options from LDF config if present
if grep -q "^# Recommended options" /config/.config 2>/dev/null; then
    echo "Applying recommended options..."
    while IFS= read -r line; do
        if [[ "$line" =~ ^#[[:space:]]*(CONFIG_[A-Z0-9_]+)=(.*)$ ]]; then
            KEY="${BASH_REMATCH[1]}"
            VALUE="${BASH_REMATCH[2]}"
            ./scripts/config --set-val "$KEY" "$VALUE" || true
        fi
    done < /config/.config
fi
`

	case string(db.KernelConfigModeOptions):
		script += `
# Generate default config first
echo "Generating defconfig for ${ARCH}..."
if [ "${ARCH}" = "x86" ] || [ "${ARCH}" = "x86_64" ]; then
    make ARCH=x86 CROSS_COMPILE="${CROSS_COMPILE}" x86_64_defconfig
else
    make ARCH="${ARCH}" CROSS_COMPILE="${CROSS_COMPILE}" defconfig
fi

# Apply custom options from config file
echo "Applying custom options..."
while IFS= read -r line; do
    # Skip comments and empty lines
    [[ "$line" =~ ^# ]] && continue
    [[ -z "$line" ]] && continue

    # Skip LDF metadata
    [[ "$line" =~ ^LDF_ ]] && continue

    if [[ "$line" =~ ^(CONFIG_[A-Z0-9_]+)=(.*)$ ]]; then
        KEY="${BASH_REMATCH[1]}"
        VALUE="${BASH_REMATCH[2]}"
        # Remove quotes if present
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
`

	case string(db.KernelConfigModeCustom):
		script += `
# Copy user-provided config (skip header comments)
echo "Using custom kernel config..."
grep -v "^# LDF" /config/.config | grep -v "^LDF_" > .config || true
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
