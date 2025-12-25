package migrations

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// migration005AddDownloadManagerTables creates the components, distribution_source_overrides,
// and download_jobs tables, and extends source tables with component fields
func migration005AddDownloadManagerTables() Migration {
	return Migration{
		Version:     5,
		Description: "Add download manager tables (components, source overrides, download jobs) and extend sources with component fields",
		Up:          migration005Up,
	}
}

func migration005Up(tx *sql.Tx) error {
	// Create components table (component registry)
	if _, err := tx.Exec(`
		CREATE TABLE components (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			artifact_pattern TEXT,
			default_url_template TEXT,
			github_normalized_template TEXT,
			is_optional BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create components table: %w", err)
	}

	// Create indexes for components
	if _, err := tx.Exec(`CREATE INDEX idx_components_category ON components(category)`); err != nil {
		return fmt.Errorf("failed to create components category index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_components_name ON components(name)`); err != nil {
		return fmt.Errorf("failed to create components name index: %w", err)
	}

	// Extend source_defaults with component fields
	if _, err := tx.Exec(`ALTER TABLE source_defaults ADD COLUMN component_id TEXT REFERENCES components(id)`); err != nil {
		return fmt.Errorf("failed to add component_id to source_defaults: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE source_defaults ADD COLUMN retrieval_method TEXT NOT NULL DEFAULT 'release'`); err != nil {
		return fmt.Errorf("failed to add retrieval_method to source_defaults: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE source_defaults ADD COLUMN url_template TEXT`); err != nil {
		return fmt.Errorf("failed to add url_template to source_defaults: %w", err)
	}

	// Create index for source_defaults component lookup
	if _, err := tx.Exec(`CREATE INDEX idx_source_defaults_component ON source_defaults(component_id)`); err != nil {
		return fmt.Errorf("failed to create source_defaults component index: %w", err)
	}

	// Extend user_sources with component fields
	if _, err := tx.Exec(`ALTER TABLE user_sources ADD COLUMN component_id TEXT REFERENCES components(id)`); err != nil {
		return fmt.Errorf("failed to add component_id to user_sources: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE user_sources ADD COLUMN retrieval_method TEXT NOT NULL DEFAULT 'release'`); err != nil {
		return fmt.Errorf("failed to add retrieval_method to user_sources: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE user_sources ADD COLUMN url_template TEXT`); err != nil {
		return fmt.Errorf("failed to add url_template to user_sources: %w", err)
	}

	// Create index for user_sources component lookup
	if _, err := tx.Exec(`CREATE INDEX idx_user_sources_component ON user_sources(component_id)`); err != nil {
		return fmt.Errorf("failed to create user_sources component index: %w", err)
	}

	// Create distribution_source_overrides table
	if _, err := tx.Exec(`
		CREATE TABLE distribution_source_overrides (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
			component_id TEXT NOT NULL REFERENCES components(id),
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(distribution_id, component_id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create distribution_source_overrides table: %w", err)
	}

	// Create indexes for distribution_source_overrides
	if _, err := tx.Exec(`CREATE INDEX idx_dist_source_overrides_dist ON distribution_source_overrides(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create distribution_source_overrides distribution index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_dist_source_overrides_component ON distribution_source_overrides(component_id)`); err != nil {
		return fmt.Errorf("failed to create distribution_source_overrides component index: %w", err)
	}

	// Create download_jobs table
	if _, err := tx.Exec(`
		CREATE TABLE download_jobs (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
			component_id TEXT NOT NULL REFERENCES components(id),
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			resolved_url TEXT NOT NULL,
			version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			progress_bytes INTEGER DEFAULT 0,
			total_bytes INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			artifact_path TEXT,
			checksum TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3
		)
	`); err != nil {
		return fmt.Errorf("failed to create download_jobs table: %w", err)
	}

	// Create indexes for download_jobs
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_distribution ON download_jobs(distribution_id)`); err != nil {
		return fmt.Errorf("failed to create download_jobs distribution index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_status ON download_jobs(status)`); err != nil {
		return fmt.Errorf("failed to create download_jobs status index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_download_jobs_component ON download_jobs(component_id)`); err != nil {
		return fmt.Errorf("failed to create download_jobs component index: %w", err)
	}

	// Seed default components
	if err := seedComponents(tx); err != nil {
		return fmt.Errorf("failed to seed components: %w", err)
	}

	return nil
}

// seedComponents inserts the default component definitions
func seedComponents(tx *sql.Tx) error {
	now := time.Now().UTC()

	components := []struct {
		name                     string
		category                 string
		displayName              string
		description              string
		artifactPattern          string
		defaultURLTemplate       string
		githubNormalizedTemplate string
		isOptional               bool
	}{
		// Core components
		{
			name:                     "kernel",
			category:                 "core",
			displayName:              "Linux Kernel",
			description:              "The Linux kernel source code",
			artifactPattern:          "linux-{version}.tar.xz",
			defaultURLTemplate:       "{base_url}/linux-{version}.tar.xz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               false,
		},
		// Bootloader components
		{
			name:                     "bootloader-systemd-boot",
			category:                 "bootloader",
			displayName:              "systemd-boot",
			description:              "UEFI boot manager from systemd",
			artifactPattern:          "systemd-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               false,
		},
		{
			name:                     "bootloader-u-boot",
			category:                 "bootloader",
			displayName:              "U-Boot",
			description:              "Universal Boot Loader for embedded systems",
			artifactPattern:          "u-boot-{version}.tar.bz2",
			defaultURLTemplate:       "{base_url}/u-boot-{version}.tar.bz2",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               false,
		},
		{
			name:                     "bootloader-grub2",
			category:                 "bootloader",
			displayName:              "GRUB2",
			description:              "GNU GRand Unified Bootloader version 2",
			artifactPattern:          "grub-{version}.tar.xz",
			defaultURLTemplate:       "{base_url}/grub-{version}.tar.xz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/grub-{version}.tar.gz",
			isOptional:               false,
		},
		// Init system components
		{
			name:                     "init-systemd",
			category:                 "init",
			displayName:              "systemd",
			description:              "System and service manager for Linux",
			artifactPattern:          "systemd-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               false,
		},
		{
			name:                     "init-openrc",
			category:                 "init",
			displayName:              "OpenRC",
			description:              "Dependency-based init system",
			artifactPattern:          "openrc-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/openrc-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			isOptional:               false,
		},
		// Runtime - Virtualization components
		{
			name:                     "virtualization-cloud-hypervisor",
			category:                 "runtime",
			displayName:              "Cloud Hypervisor",
			description:              "Open source Virtual Machine Monitor (VMM) for cloud workloads",
			artifactPattern:          "cloud-hypervisor-v{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/cloud-hypervisor-v{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "virtualization-qemu-kvm-libvirt",
			category:                 "runtime",
			displayName:              "QEMU/KVM with libvirt",
			description:              "Full virtualization solution with libvirt management",
			artifactPattern:          "qemu-{version}.tar.xz",
			defaultURLTemplate:       "{base_url}/qemu-{version}.tar.xz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		// Runtime - Container components
		{
			name:                     "container-docker-podman",
			category:                 "runtime",
			displayName:              "Docker/Podman",
			description:              "Container runtime with Docker/Podman compatibility",
			artifactPattern:          "podman-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/podman-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "container-runc",
			category:                 "runtime",
			displayName:              "runC",
			description:              "CLI tool for running containers according to OCI specification",
			artifactPattern:          "runc-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/runc-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "container-cri-o",
			category:                 "runtime",
			displayName:              "CRI-O",
			description:              "Lightweight container runtime for Kubernetes",
			artifactPattern:          "cri-o-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/cri-o-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		// Security components
		{
			name:                     "security-selinux",
			category:                 "security",
			displayName:              "SELinux",
			description:              "Security-Enhanced Linux",
			artifactPattern:          "selinux-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/selinux-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "security-apparmor",
			category:                 "security",
			displayName:              "AppArmor",
			description:              "Linux application security framework",
			artifactPattern:          "apparmor-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/apparmor-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		// Desktop environment components
		{
			name:                     "desktop-kde",
			category:                 "desktop",
			displayName:              "KDE Plasma",
			description:              "KDE Plasma desktop environment",
			artifactPattern:          "plasma-desktop-{version}.tar.xz",
			defaultURLTemplate:       "{base_url}/plasma-desktop-{version}.tar.xz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "desktop-gnome",
			category:                 "desktop",
			displayName:              "GNOME",
			description:              "GNOME desktop environment",
			artifactPattern:          "gnome-shell-{version}.tar.xz",
			defaultURLTemplate:       "{base_url}/gnome-shell-{version}.tar.xz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			isOptional:               true,
		},
		{
			name:                     "desktop-swaywm",
			category:                 "desktop",
			displayName:              "SwayWM",
			description:              "i3-compatible Wayland compositor",
			artifactPattern:          "sway-{version}.tar.gz",
			defaultURLTemplate:       "{base_url}/sway-{version}.tar.gz",
			githubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			isOptional:               true,
		},
	}

	stmt, err := tx.Prepare(`
		INSERT INTO components (id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare component insert statement: %w", err)
	}
	defer stmt.Close()

	for _, c := range components {
		id := uuid.New().String()
		_, err := stmt.Exec(
			id,
			c.name,
			c.category,
			c.displayName,
			c.description,
			c.artifactPattern,
			c.defaultURLTemplate,
			c.githubNormalizedTemplate,
			c.isOptional,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert component %s: %w", c.name, err)
		}
	}

	return nil
}
