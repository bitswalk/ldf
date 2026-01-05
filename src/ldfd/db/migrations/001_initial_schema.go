package migrations

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// System role IDs - must match constants in auth/role.go
const (
	RoleIDRoot      = "908b291e-61fb-4d95-98db-0b76c0afd6b4"
	RoleIDDeveloper = "91db9f27-b8a2-4452-9b80-5f6ab1096da8"
	RoleIDAnonymous = "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0"
)

// migration001InitialSchema creates the complete database schema and seeds default data
func migration001InitialSchema() Migration {
	return Migration{
		Version:     1,
		Description: "Complete initial schema with all tables, indexes, and default data",
		Up:          migration001Up,
	}
}

func migration001Up(tx *sql.Tx) error {
	// Create all tables in dependency order
	if err := createTables(tx); err != nil {
		return err
	}

	// Create all indexes
	if err := createIndexes(tx); err != nil {
		return err
	}

	// Seed default data
	if err := seedDefaultRoles(tx); err != nil {
		return err
	}

	if err := seedDefaultComponents(tx); err != nil {
		return err
	}

	return nil
}

func createTables(tx *sql.Tx) error {
	tables := []string{
		// Settings table (no dependencies)
		`CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Roles table (self-referencing for parent_role_id)
		`CREATE TABLE roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			can_read BOOLEAN NOT NULL DEFAULT 1,
			can_write BOOLEAN NOT NULL DEFAULT 0,
			can_delete BOOLEAN NOT NULL DEFAULT 0,
			can_admin BOOLEAN NOT NULL DEFAULT 0,
			is_system BOOLEAN NOT NULL DEFAULT 0,
			parent_role_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_role_id) REFERENCES roles(id) ON DELETE SET NULL
		)`,

		// Users table (references roles)
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
		)`,

		// Revoked tokens table (references users)
		`CREATE TABLE revoked_tokens (
			token_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Components table (references users for owner)
		`CREATE TABLE components (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			artifact_pattern TEXT,
			default_url_template TEXT,
			github_normalized_template TEXT,
			is_optional BOOLEAN NOT NULL DEFAULT 0,
			is_system BOOLEAN NOT NULL DEFAULT 1,
			owner_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// Distributions table (references users)
		`CREATE TABLE distributions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			visibility TEXT NOT NULL DEFAULT 'private',
			config TEXT,
			source_url TEXT,
			checksum TEXT,
			size_bytes INTEGER DEFAULT 0,
			owner_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			error_message TEXT,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL
		)`,

		// Distribution logs table (references distributions)
		`CREATE TABLE distribution_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			distribution_id TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE
		)`,

		// Source defaults table (references components)
		`CREATE TABLE source_defaults (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			url TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			component_id TEXT,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			url_template TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (component_id) REFERENCES components(id) ON DELETE SET NULL
		)`,

		// User sources table (references users and components)
		`CREATE TABLE user_sources (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			component_id TEXT,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
			url_template TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(owner_id, name),
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (component_id) REFERENCES components(id) ON DELETE SET NULL
		)`,

		// Distribution source overrides table (references distributions and components)
		`CREATE TABLE distribution_source_overrides (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL,
			component_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(distribution_id, component_id),
			FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE,
			FOREIGN KEY (component_id) REFERENCES components(id) ON DELETE CASCADE
		)`,

		// Download jobs table (references distributions and components)
		`CREATE TABLE download_jobs (
			id TEXT PRIMARY KEY,
			distribution_id TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			component_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			retrieval_method TEXT NOT NULL DEFAULT 'release',
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
			max_retries INTEGER DEFAULT 3,
			FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE,
			FOREIGN KEY (component_id) REFERENCES components(id) ON DELETE CASCADE
		)`,

		// Source versions table (for version sync)
		`CREATE TABLE source_versions (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			version TEXT NOT NULL,
			release_date DATETIME,
			download_url TEXT,
			checksum TEXT,
			checksum_type TEXT,
			file_size INTEGER,
			is_stable BOOLEAN DEFAULT 1,
			discovered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_id, source_type, version)
		)`,

		// Version sync jobs table
		`CREATE TABLE version_sync_jobs (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			versions_found INTEGER DEFAULT 0,
			versions_new INTEGER DEFAULT 0,
			started_at DATETIME,
			completed_at DATETIME,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Language packs table
		`CREATE TABLE language_packs (
			locale TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			author TEXT,
			dictionary TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, sql := range tables {
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

func createIndexes(tx *sql.Tx) error {
	indexes := []string{
		// Roles indexes
		`CREATE INDEX idx_roles_name ON roles(name)`,
		`CREATE INDEX idx_roles_parent ON roles(parent_role_id)`,

		// Users indexes
		`CREATE INDEX idx_users_name ON users(name)`,
		`CREATE INDEX idx_users_email ON users(email)`,
		`CREATE INDEX idx_users_role ON users(role_id)`,

		// Revoked tokens indexes
		`CREATE INDEX idx_revoked_tokens_user_id ON revoked_tokens(user_id)`,
		`CREATE INDEX idx_revoked_tokens_expires_at ON revoked_tokens(expires_at)`,

		// Components indexes
		`CREATE INDEX idx_components_name ON components(name)`,
		`CREATE INDEX idx_components_category ON components(category)`,
		`CREATE INDEX idx_components_is_system ON components(is_system)`,
		`CREATE INDEX idx_components_owner ON components(owner_id)`,

		// Distributions indexes
		`CREATE INDEX idx_distributions_name ON distributions(name)`,
		`CREATE INDEX idx_distributions_status ON distributions(status)`,
		`CREATE INDEX idx_distributions_owner ON distributions(owner_id)`,
		`CREATE INDEX idx_distributions_visibility ON distributions(visibility)`,

		// Distribution logs indexes
		`CREATE INDEX idx_distribution_logs_dist_id ON distribution_logs(distribution_id)`,

		// Source defaults indexes
		`CREATE INDEX idx_source_defaults_component ON source_defaults(component_id)`,

		// User sources indexes
		`CREATE INDEX idx_user_sources_owner ON user_sources(owner_id)`,
		`CREATE INDEX idx_user_sources_component ON user_sources(component_id)`,

		// Distribution source overrides indexes
		`CREATE INDEX idx_dist_source_overrides_dist ON distribution_source_overrides(distribution_id)`,
		`CREATE INDEX idx_dist_source_overrides_component ON distribution_source_overrides(component_id)`,

		// Download jobs indexes
		`CREATE INDEX idx_download_jobs_distribution ON download_jobs(distribution_id)`,
		`CREATE INDEX idx_download_jobs_status ON download_jobs(status)`,
		`CREATE INDEX idx_download_jobs_component ON download_jobs(component_id)`,

		// Source versions indexes
		`CREATE INDEX idx_source_versions_source ON source_versions(source_id, source_type)`,
		`CREATE INDEX idx_source_versions_version ON source_versions(version)`,
		`CREATE INDEX idx_source_versions_stable ON source_versions(source_id, source_type, is_stable)`,

		// Version sync jobs indexes
		`CREATE INDEX idx_version_sync_jobs_source ON version_sync_jobs(source_id, source_type)`,
		`CREATE INDEX idx_version_sync_jobs_status ON version_sync_jobs(status)`,
		`CREATE INDEX idx_version_sync_jobs_created ON version_sync_jobs(created_at DESC)`,

		// Language packs indexes
		`CREATE INDEX idx_language_packs_created ON language_packs(created_at DESC)`,
	}

	for _, sql := range indexes {
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// DefaultRole represents a system role to be seeded
type DefaultRole struct {
	ID          string
	Name        string
	Description string
	CanRead     bool
	CanWrite    bool
	CanDelete   bool
	CanAdmin    bool
}

// DefaultRoles returns the list of default system roles
func DefaultRoles() []DefaultRole {
	return []DefaultRole{
		{
			ID:          RoleIDRoot,
			Name:        "root",
			Description: "Administrator role with full system access",
			CanRead:     true,
			CanWrite:    true,
			CanDelete:   true,
			CanAdmin:    true,
		},
		{
			ID:          RoleIDDeveloper,
			Name:        "developer",
			Description: "Standard user with read/write access to owned resources",
			CanRead:     true,
			CanWrite:    true,
			CanDelete:   true,
			CanAdmin:    false,
		},
		{
			ID:          RoleIDAnonymous,
			Name:        "anonymous",
			Description: "Read-only access to public resources",
			CanRead:     true,
			CanWrite:    false,
			CanDelete:   false,
			CanAdmin:    false,
		},
	}
}

func seedDefaultRoles(tx *sql.Tx) error {
	stmt, err := tx.Prepare(`
		INSERT INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare role insert: %w", err)
	}
	defer stmt.Close()

	for _, role := range DefaultRoles() {
		if _, err := stmt.Exec(
			role.ID,
			role.Name,
			role.Description,
			role.CanRead,
			role.CanWrite,
			role.CanDelete,
			role.CanAdmin,
		); err != nil {
			return fmt.Errorf("failed to insert role %s: %w", role.Name, err)
		}
	}

	return nil
}

// DefaultComponent represents a system component to be seeded
type DefaultComponent struct {
	Name                     string
	Category                 string
	DisplayName              string
	Description              string
	ArtifactPattern          string
	DefaultURLTemplate       string
	GithubNormalizedTemplate string
	IsOptional               bool
}

// DefaultComponents returns the list of default system components
func DefaultComponents() []DefaultComponent {
	return []DefaultComponent{
		// Core components
		{
			Name:                     "kernel",
			Category:                 "core",
			DisplayName:              "Linux Kernel",
			Description:              "The Linux kernel source code",
			ArtifactPattern:          "linux-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/linux-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               false,
		},
		// Bootloader components
		{
			Name:                     "systemd-boot",
			Category:                 "bootloader",
			DisplayName:              "systemd-boot",
			Description:              "UEFI boot manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               false,
		},
		{
			Name:                     "u-boot",
			Category:                 "bootloader",
			DisplayName:              "U-Boot",
			Description:              "Universal Boot Loader for embedded systems",
			ArtifactPattern:          "u-boot-{version}.tar.bz2",
			DefaultURLTemplate:       "{base_url}/u-boot-{version}.tar.bz2",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               false,
		},
		{
			Name:                     "grub2",
			Category:                 "bootloader",
			DisplayName:              "GRUB2",
			Description:              "GNU GRand Unified Bootloader version 2",
			ArtifactPattern:          "grub-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/grub-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/grub-{version}.tar.gz",
			IsOptional:               false,
		},
		// Init system components
		{
			Name:                     "systemd",
			Category:                 "init",
			DisplayName:              "systemd",
			Description:              "System and service manager for Linux",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               false,
		},
		{
			Name:                     "systemd-networkd",
			Category:                 "init",
			DisplayName:              "systemd-networkd",
			Description:              "Network configuration manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-resolved",
			Category:                 "init",
			DisplayName:              "systemd-resolved",
			Description:              "Network name resolution manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-sysext",
			Category:                 "init",
			DisplayName:              "systemd-sysext",
			Description:              "System extension image manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-confext",
			Category:                 "init",
			DisplayName:              "systemd-confext",
			Description:              "Configuration extension image manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-vmspawn",
			Category:                 "init",
			DisplayName:              "systemd-vmspawn",
			Description:              "Lightweight VM spawner using QEMU from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-nspawn",
			Category:                 "init",
			DisplayName:              "systemd-nspawn",
			Description:              "Lightweight container manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-repart",
			Category:                 "init",
			DisplayName:              "systemd-repart",
			Description:              "Automatic partition growing and creation from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-udevd",
			Category:                 "init",
			DisplayName:              "systemd-udevd",
			Description:              "Device event managing daemon from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-homed",
			Category:                 "init",
			DisplayName:              "systemd-homed",
			Description:              "Portable home directory manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-machined",
			Category:                 "init",
			DisplayName:              "systemd-machined",
			Description:              "Virtual machine and container registration manager from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-importd",
			Category:                 "init",
			DisplayName:              "systemd-importd",
			Description:              "VM and container image import and export service from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-run",
			Category:                 "init",
			DisplayName:              "systemd-run",
			Description:              "Transient unit execution utility from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "systemd-analyze",
			Category:                 "init",
			DisplayName:              "systemd-analyze",
			Description:              "System boot-up performance analysis tool from systemd",
			ArtifactPattern:          "systemd-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/systemd-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "openrc",
			Category:                 "init",
			DisplayName:              "OpenRC",
			Description:              "Dependency-based init system",
			ArtifactPattern:          "openrc-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/openrc-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			IsOptional:               false,
		},
		// Virtualization components
		{
			Name:                     "cloud-hypervisor",
			Category:                 "virtualization",
			DisplayName:              "Cloud Hypervisor",
			Description:              "Open source Virtual Machine Monitor (VMM) for cloud workloads",
			ArtifactPattern:          "cloud-hypervisor-v{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/cloud-hypervisor-v{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "qemu-kvm-libvirt",
			Category:                 "virtualization",
			DisplayName:              "QEMU/KVM with libvirt",
			Description:              "Full virtualization solution with libvirt management",
			ArtifactPattern:          "qemu-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/qemu-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		// Container components
		{
			Name:                     "docker",
			Category:                 "container",
			DisplayName:              "Docker",
			Description:              "Container runtime and platform for building and running applications",
			ArtifactPattern:          "docker-{version}.tgz",
			DefaultURLTemplate:       "{base_url}/docker-{version}.tgz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "podman",
			Category:                 "container",
			DisplayName:              "Podman",
			Description:              "Daemonless container engine for developing, managing, and running OCI containers",
			ArtifactPattern:          "podman-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/podman-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "runc",
			Category:                 "container",
			DisplayName:              "runC",
			Description:              "CLI tool for running containers according to OCI specification",
			ArtifactPattern:          "runc-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/runc-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "cri-o",
			Category:                 "container",
			DisplayName:              "CRI-O",
			Description:              "Lightweight container runtime for Kubernetes",
			ArtifactPattern:          "cri-o-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/cri-o-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		// Security components
		{
			Name:                     "selinux",
			Category:                 "security",
			DisplayName:              "SELinux",
			Description:              "Security-Enhanced Linux",
			ArtifactPattern:          "selinux-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/selinux-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "apparmor",
			Category:                 "security",
			DisplayName:              "AppArmor",
			Description:              "Linux application security framework",
			ArtifactPattern:          "apparmor-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/apparmor-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		// Desktop environment components
		{
			Name:                     "kde",
			Category:                 "desktop",
			DisplayName:              "KDE Plasma",
			Description:              "KDE Plasma desktop environment",
			ArtifactPattern:          "plasma-desktop-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/plasma-desktop-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "gnome",
			Category:                 "desktop",
			DisplayName:              "GNOME",
			Description:              "GNOME desktop environment",
			ArtifactPattern:          "gnome-shell-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/gnome-shell-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			IsOptional:               true,
		},
		{
			Name:                     "swaywm",
			Category:                 "desktop",
			DisplayName:              "SwayWM",
			Description:              "i3-compatible Wayland compositor",
			ArtifactPattern:          "sway-{version}.tar.gz",
			DefaultURLTemplate:       "{base_url}/sway-{version}.tar.gz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/{version}.tar.gz",
			IsOptional:               true,
		},
	}
}

func seedDefaultComponents(tx *sql.Tx) error {
	now := time.Now().UTC()

	stmt, err := tx.Prepare(`
		INSERT INTO components (id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare component insert: %w", err)
	}
	defer stmt.Close()

	for _, c := range DefaultComponents() {
		id := uuid.New().String()
		if _, err := stmt.Exec(
			id,
			c.Name,
			c.Category,
			c.DisplayName,
			c.Description,
			c.ArtifactPattern,
			c.DefaultURLTemplate,
			c.GithubNormalizedTemplate,
			c.IsOptional,
			now,
			now,
		); err != nil {
			return fmt.Errorf("failed to insert component %s: %w", c.Name, err)
		}
	}

	return nil
}
