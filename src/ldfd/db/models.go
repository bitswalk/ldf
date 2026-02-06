package db

import "time"

// DistributionConfig represents the full configuration for building a distribution
type DistributionConfig struct {
	Core           CoreConfig     `json:"core"`
	System         SystemConfig   `json:"system"`
	Security       SecurityConfig `json:"security"`
	Runtime        RuntimeConfig  `json:"runtime"`
	Target         TargetConfig   `json:"target"`
	BoardProfileID string         `json:"board_profile_id,omitempty"`
}

// CoreConfig contains core system configuration
type CoreConfig struct {
	Kernel            KernelConfig       `json:"kernel"`
	Bootloader        string             `json:"bootloader"`
	BootloaderVersion string             `json:"bootloader_version,omitempty"`
	Partitioning      PartitioningConfig `json:"partitioning"`
}

// KernelConfigMode represents the kernel configuration mode
type KernelConfigMode string

const (
	// KernelConfigModeDefconfig uses architecture default config (make defconfig)
	KernelConfigModeDefconfig KernelConfigMode = "defconfig"
	// KernelConfigModeOptions uses defconfig with additional options applied
	KernelConfigModeOptions KernelConfigMode = "options"
	// KernelConfigModeCustom uses a user-provided complete .config file
	KernelConfigModeCustom KernelConfigMode = "custom"
)

// KernelConfig contains kernel configuration
type KernelConfig struct {
	Version string `json:"version"`
	// ConfigMode determines how the kernel .config is generated
	// "defconfig" - use arch default, "options" - defconfig + custom options, "custom" - user-provided file
	ConfigMode KernelConfigMode `json:"config_mode,omitempty"`
	// ConfigOptions are key-value pairs applied on top of defconfig (when ConfigMode is "options")
	// Example: {"CONFIG_EXT4_FS": "y", "CONFIG_BTRFS_FS": "m", "CONFIG_DEBUG_INFO": "n"}
	ConfigOptions map[string]string `json:"config_options,omitempty"`
	// CustomConfigPath is the storage path to a user-uploaded .config file (when ConfigMode is "custom")
	CustomConfigPath string `json:"custom_config_path,omitempty"`
}

// PartitioningConfig contains partitioning configuration
type PartitioningConfig struct {
	Type string `json:"type"`
	Mode string `json:"mode"`
}

// SystemConfig contains system services configuration
type SystemConfig struct {
	Init                  string           `json:"init"`
	InitVersion           string           `json:"init_version,omitempty"`
	Filesystem            FilesystemConfig `json:"filesystem"`
	FilesystemVersion     string           `json:"filesystem_version,omitempty"`
	FilesystemUserspace   bool             `json:"filesystem_userspace,omitempty"` // Include userspace tools for hybrid filesystem components
	PackageManager        string           `json:"packageManager"`
	PackageManagerVersion string           `json:"package_manager_version,omitempty"`
}

// FilesystemConfig contains filesystem configuration
type FilesystemConfig struct {
	Type      string `json:"type"`
	Hierarchy string `json:"hierarchy"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	System          string `json:"system"`
	SystemVersion   string `json:"system_version,omitempty"`
	SystemUserspace bool   `json:"system_userspace,omitempty"` // Include userspace tools for hybrid security components (SELinux, AppArmor)
}

// RuntimeConfig contains runtime configuration
type RuntimeConfig struct {
	Container             string `json:"container"`
	ContainerVersion      string `json:"container_version,omitempty"`
	Virtualization        string `json:"virtualization"`
	VirtualizationVersion string `json:"virtualization_version,omitempty"`
}

// TargetConfig contains target environment configuration
type TargetConfig struct {
	Type    string         `json:"type"`
	Desktop *DesktopConfig `json:"desktop,omitempty"`
}

// DesktopConfig contains desktop environment configuration
type DesktopConfig struct {
	Environment          string `json:"environment"`
	EnvironmentVersion   string `json:"environment_version,omitempty"`
	DisplayServer        string `json:"displayServer"`
	DisplayServerVersion string `json:"display_server_version,omitempty"`
}

// DistributionStatus represents the status of a distribution
type DistributionStatus string

const (
	StatusPending     DistributionStatus = "pending"
	StatusDownloading DistributionStatus = "downloading"
	StatusValidating  DistributionStatus = "validating"
	StatusReady       DistributionStatus = "ready"
	StatusFailed      DistributionStatus = "failed"
	StatusDeleted     DistributionStatus = "deleted"
)

// Visibility represents the visibility level of a distribution
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Distribution represents a distribution record
type Distribution struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Status       DistributionStatus  `json:"status"`
	Visibility   Visibility          `json:"visibility"`
	Config       *DistributionConfig `json:"config,omitempty"`
	SourceURL    string              `json:"source_url,omitempty"`
	Checksum     string              `json:"checksum,omitempty"`
	SizeBytes    int64               `json:"size_bytes"`
	OwnerID      string              `json:"owner_id,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

// DistributionLog represents a log entry for a distribution
type DistributionLog struct {
	ID             int64     `json:"id"`
	DistributionID string    `json:"distribution_id"`
	Level          string    `json:"level"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// UpstreamSource represents a unified source (both system and user sources)
type UpstreamSource struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	ComponentIDs    []string  `json:"component_ids"`
	RetrievalMethod string    `json:"retrieval_method"`
	URLTemplate     string    `json:"url_template,omitempty"`
	ForgeType       string    `json:"forge_type"`
	VersionFilter   string    `json:"version_filter,omitempty"`
	DefaultVersion  string    `json:"default_version,omitempty"` // Default/recommended version for this source
	Priority        int       `json:"priority"`
	Enabled         bool      `json:"enabled"`
	IsSystem        bool      `json:"is_system"`
	OwnerID         string    `json:"owner_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// VersionRule represents how default version is determined for a component
type VersionRule string

const (
	VersionRulePinned       VersionRule = "pinned"        // Use exact default_version value
	VersionRuleLatestStable VersionRule = "latest-stable" // Resolve to newest stable version
	VersionRuleLatestLTS    VersionRule = "latest-lts"    // Resolve to newest longterm/LTS version
)

// Component represents a downloadable component in the registry
type Component struct {
	ID                       string       `json:"id"`
	Name                     string       `json:"name"`
	Category                 string       `json:"category"`             // Primary category (first in categories list)
	Categories               []string     `json:"categories,omitempty"` // All categories (stored as comma-separated in DB)
	DisplayName              string       `json:"display_name"`
	Description              string       `json:"description,omitempty"`
	ArtifactPattern          string       `json:"artifact_pattern,omitempty"`
	DefaultURLTemplate       string       `json:"default_url_template,omitempty"`
	GitHubNormalizedTemplate string       `json:"github_normalized_template,omitempty"`
	IsOptional               bool         `json:"is_optional"`
	IsSystem                 bool         `json:"is_system"`
	IsKernelModule           bool         `json:"is_kernel_module"` // Requires kernel configuration at build time
	IsUserspace              bool         `json:"is_userspace"`     // Needs to be built as userspace binary
	OwnerID                  string       `json:"owner_id,omitempty"`
	DefaultVersion           string       `json:"default_version,omitempty"`         // Pinned version or resolved value
	DefaultVersionRule       VersionRule  `json:"default_version_rule,omitempty"`    // "pinned", "latest-stable", "latest-lts"
	SupportedArchitectures   []TargetArch `json:"supported_architectures,omitempty"` // When empty, supports all architectures
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}

// DownloadJobStatus represents the status of a download job
type DownloadJobStatus string

const (
	JobStatusPending     DownloadJobStatus = "pending"
	JobStatusVerifying   DownloadJobStatus = "verifying"
	JobStatusDownloading DownloadJobStatus = "downloading"
	JobStatusCompleted   DownloadJobStatus = "completed"
	JobStatusFailed      DownloadJobStatus = "failed"
	JobStatusCancelled   DownloadJobStatus = "cancelled"
)

// DownloadJob represents a download task for a component
type DownloadJob struct {
	ID              string            `json:"id"`
	DistributionID  string            `json:"distribution_id"`
	OwnerID         string            `json:"owner_id"`
	ComponentID     string            `json:"component_id"`
	ComponentName   string            `json:"component_name"`
	ComponentIDs    []string          `json:"component_ids,omitempty"` // All components sharing this artifact
	SourceID        string            `json:"source_id"`
	SourceName      string            `json:"source_name,omitempty"` // Source name for artifact path
	SourceType      string            `json:"source_type"`
	RetrievalMethod string            `json:"retrieval_method"` // "release" or "git"
	ResolvedURL     string            `json:"resolved_url"`
	Version         string            `json:"version"`
	Status          DownloadJobStatus `json:"status"`
	ProgressBytes   int64             `json:"progress_bytes"`
	TotalBytes      int64             `json:"total_bytes"`
	CreatedAt       time.Time         `json:"created_at"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	ArtifactPath    string            `json:"artifact_path,omitempty"`
	Checksum        string            `json:"checksum,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
}

// VersionType represents the type/category of a version
type VersionType string

const (
	VersionTypeMainline  VersionType = "mainline"
	VersionTypeStable    VersionType = "stable"
	VersionTypeLongterm  VersionType = "longterm"
	VersionTypeLinuxNext VersionType = "linux-next"
)

// SourceVersion represents a discovered version from an upstream source
type SourceVersion struct {
	ID           string      `json:"id"`
	SourceID     string      `json:"source_id"`
	SourceType   string      `json:"source_type"` // "default" or "user"
	Version      string      `json:"version"`
	VersionType  VersionType `json:"version_type"` // "mainline", "stable", "longterm", "linux-next"
	ReleaseDate  *time.Time  `json:"release_date,omitempty"`
	DownloadURL  string      `json:"download_url,omitempty"`
	Checksum     string      `json:"checksum,omitempty"`
	ChecksumType string      `json:"checksum_type,omitempty"`
	FileSize     int64       `json:"file_size,omitempty"`
	IsStable     bool        `json:"is_stable"` // Kept for backwards compatibility
	DiscoveredAt time.Time   `json:"discovered_at"`
}

// VersionSyncStatus represents the status of a version sync job
type VersionSyncStatus string

const (
	SyncStatusPending   VersionSyncStatus = "pending"
	SyncStatusRunning   VersionSyncStatus = "running"
	SyncStatusCompleted VersionSyncStatus = "completed"
	SyncStatusFailed    VersionSyncStatus = "failed"
)

// VersionSyncJob represents a version sync job for a source
type VersionSyncJob struct {
	ID            string            `json:"id"`
	SourceID      string            `json:"source_id"`
	SourceType    string            `json:"source_type"` // "default" or "user"
	Status        VersionSyncStatus `json:"status"`
	VersionsFound int               `json:"versions_found"`
	VersionsNew   int               `json:"versions_new"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// BuildJobStatus represents the status of a build job
type BuildJobStatus string

const (
	BuildStatusPending    BuildJobStatus = "pending"
	BuildStatusResolving  BuildJobStatus = "resolving"
	BuildStatusPreparing  BuildJobStatus = "preparing"
	BuildStatusCompiling  BuildJobStatus = "compiling"
	BuildStatusAssembling BuildJobStatus = "assembling"
	BuildStatusPackaging  BuildJobStatus = "packaging"
	BuildStatusCompleted  BuildJobStatus = "completed"
	BuildStatusFailed     BuildJobStatus = "failed"
	BuildStatusCancelled  BuildJobStatus = "cancelled"
)

// BuildStageName defines the pipeline stages
type BuildStageName string

const (
	StageResolve  BuildStageName = "resolve"
	StageDownload BuildStageName = "download"
	StagePrepare  BuildStageName = "prepare"
	StageCompile  BuildStageName = "compile"
	StageAssemble BuildStageName = "assemble"
	StagePackage  BuildStageName = "package"
)

// TargetArch represents a supported target architecture
type TargetArch string

const (
	ArchX86_64  TargetArch = "x86_64"
	ArchAARCH64 TargetArch = "aarch64"
)

// ImageFormat represents the output image format
type ImageFormat string

const (
	ImageFormatRaw   ImageFormat = "raw"
	ImageFormatQCOW2 ImageFormat = "qcow2"
	ImageFormatISO   ImageFormat = "iso"
)

// BuildJob represents a build task for a distribution
type BuildJob struct {
	ID               string         `json:"id"`
	DistributionID   string         `json:"distribution_id"`
	OwnerID          string         `json:"owner_id"`
	Status           BuildJobStatus `json:"status"`
	CurrentStage     string         `json:"current_stage"`
	TargetArch       TargetArch     `json:"target_arch"`
	ImageFormat      ImageFormat    `json:"image_format"`
	ProgressPercent  int            `json:"progress_percent"`
	WorkspacePath    string         `json:"workspace_path,omitempty"`
	ArtifactPath     string         `json:"artifact_path,omitempty"`
	ArtifactChecksum string         `json:"artifact_checksum,omitempty"`
	ArtifactSize     int64          `json:"artifact_size"`
	ErrorMessage     string         `json:"error_message,omitempty"`
	ErrorStage       string         `json:"error_stage,omitempty"`
	RetryCount       int            `json:"retry_count"`
	MaxRetries       int            `json:"max_retries"`
	ClearCache       bool           `json:"clear_cache"`
	ConfigSnapshot   string         `json:"config_snapshot,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
}

// BuildStage represents a single stage in the build pipeline
type BuildStage struct {
	ID              int64          `json:"id"`
	BuildID         string         `json:"build_id"`
	Name            BuildStageName `json:"name"`
	Status          string         `json:"status"`
	ProgressPercent int            `json:"progress_percent"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	DurationMs      int64          `json:"duration_ms"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	LogPath         string         `json:"log_path,omitempty"`
}

// BuildLog represents a log entry for a build
type BuildLog struct {
	ID        int64     `json:"id"`
	BuildID   string    `json:"build_id"`
	Stage     string    `json:"stage"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// LanguagePack represents a custom language pack for i18n
type LanguagePack struct {
	Locale     string    `json:"locale"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Author     string    `json:"author,omitempty"`
	Dictionary string    `json:"dictionary"` // JSON blob
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// LanguagePackMeta represents metadata about a language pack (without full dictionary)
type LanguagePackMeta struct {
	Locale    string    `json:"locale"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Author    string    `json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BoardProfile represents a hardware board profile for targeted builds
type BoardProfile struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"` // unique slug: "rpi4", "generic-x86_64"
	DisplayName string      `json:"display_name"`
	Description string      `json:"description,omitempty"`
	Arch        TargetArch  `json:"arch"` // architecture constraint
	Config      BoardConfig `json:"config"`
	IsSystem    bool        `json:"is_system"`
	OwnerID     string      `json:"owner_id,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// BoardConfig holds all board-specific build parameters
type BoardConfig struct {
	DeviceTrees     []DeviceTreeSpec  `json:"device_trees,omitempty"`
	KernelOverlay   map[string]string `json:"kernel_overlay,omitempty"`   // CONFIG_ options merged on top
	KernelDefconfig string            `json:"kernel_defconfig,omitempty"` // board-specific defconfig (e.g. "bcm2711_defconfig")
	BootParams      BoardBootParams   `json:"boot_params,omitempty"`
	Firmware        []BoardFirmware   `json:"firmware,omitempty"`
	KernelCmdline   string            `json:"kernel_cmdline,omitempty"`
}

// DeviceTreeSpec defines a device tree source to compile and include
type DeviceTreeSpec struct {
	Source   string   `json:"source"`             // path relative to kernel source tree
	Overlays []string `json:"overlays,omitempty"` // optional DT overlay paths
}

// BoardBootParams holds board-specific bootloader parameters
type BoardBootParams struct {
	BootloaderOverride string            `json:"bootloader_override,omitempty"` // override distro bootloader choice
	UBootBoard         string            `json:"uboot_board,omitempty"`         // U-Boot board config name
	ExtraFiles         map[string]string `json:"extra_files,omitempty"`         // extra files to place (dest -> content)
	ConfigTxt          string            `json:"config_txt,omitempty"`          // RPi config.txt content
}

// BoardFirmware describes firmware blobs required by the board
type BoardFirmware struct {
	Name        string `json:"name"`
	ComponentID string `json:"component_id,omitempty"` // optional reference to component registry
	Path        string `json:"path,omitempty"`         // install path in rootfs
	Description string `json:"description,omitempty"`
}
