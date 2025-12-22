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
	ID           int64               `json:"id"`
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
	DistributionID int64     `json:"distribution_id"`
	Level          string    `json:"level"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}
