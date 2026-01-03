package db

import "time"

// DistributionConfig represents the full configuration for building a distribution
type DistributionConfig struct {
	Core     CoreConfig     `json:"core"`
	System   SystemConfig   `json:"system"`
	Security SecurityConfig `json:"security"`
	Runtime  RuntimeConfig  `json:"runtime"`
	Target   TargetConfig   `json:"target"`
}

// CoreConfig contains core system configuration
type CoreConfig struct {
	Kernel       KernelConfig       `json:"kernel"`
	Bootloader   string             `json:"bootloader"`
	Partitioning PartitioningConfig `json:"partitioning"`
}

// KernelConfig contains kernel configuration
type KernelConfig struct {
	Version string `json:"version"`
}

// PartitioningConfig contains partitioning configuration
type PartitioningConfig struct {
	Type string `json:"type"`
	Mode string `json:"mode"`
}

// SystemConfig contains system services configuration
type SystemConfig struct {
	Init           string           `json:"init"`
	Filesystem     FilesystemConfig `json:"filesystem"`
	PackageManager string           `json:"packageManager"`
}

// FilesystemConfig contains filesystem configuration
type FilesystemConfig struct {
	Type      string `json:"type"`
	Hierarchy string `json:"hierarchy"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	System string `json:"system"`
}

// RuntimeConfig contains runtime configuration
type RuntimeConfig struct {
	Container      string `json:"container"`
	Virtualization string `json:"virtualization"`
}

// TargetConfig contains target environment configuration
type TargetConfig struct {
	Type    string         `json:"type"`
	Desktop *DesktopConfig `json:"desktop,omitempty"`
}

// DesktopConfig contains desktop environment configuration
type DesktopConfig struct {
	Environment   string `json:"environment"`
	DisplayServer string `json:"displayServer"`
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
	Priority        int       `json:"priority"`
	Enabled         bool      `json:"enabled"`
	IsSystem        bool      `json:"is_system"`
	OwnerID         string    `json:"owner_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SourceDefault is an alias for UpstreamSource for backwards compatibility
// Deprecated: Use UpstreamSource instead
type SourceDefault = UpstreamSource

// UserSource is an alias for UpstreamSource for backwards compatibility
// Deprecated: Use UpstreamSource instead
type UserSource = UpstreamSource

// Source is an alias for UpstreamSource for API responses
// This maintains backwards compatibility with existing API consumers
type Source = UpstreamSource

// Component represents a downloadable component in the registry
type Component struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	Category                 string    `json:"category"`
	DisplayName              string    `json:"display_name"`
	Description              string    `json:"description,omitempty"`
	ArtifactPattern          string    `json:"artifact_pattern,omitempty"`
	DefaultURLTemplate       string    `json:"default_url_template,omitempty"`
	GitHubNormalizedTemplate string    `json:"github_normalized_template,omitempty"`
	IsOptional               bool      `json:"is_optional"`
	IsSystem                 bool      `json:"is_system"`
	OwnerID                  string    `json:"owner_id,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
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
	SourceID        string            `json:"source_id"`
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

// DistributionSourceOverride represents a per-distribution source binding
type DistributionSourceOverride struct {
	ID             string    `json:"id"`
	DistributionID string    `json:"distribution_id"`
	ComponentID    string    `json:"component_id"`
	SourceID       string    `json:"source_id"`
	SourceType     string    `json:"source_type"`
	CreatedAt      time.Time `json:"created_at"`
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
