package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitInstaller defines the interface for init system installers
type InitInstaller interface {
	// Name returns the init system name
	Name() string
	// Install installs the init system to the rootfs
	Install(rootfsPath string, component *ResolvedComponent) error
	// Configure configures the init system
	Configure(rootfsPath string) error
	// EnableService enables a service to start at boot
	EnableService(rootfsPath, serviceName string) error
}

// SystemdInstaller installs and configures systemd
type SystemdInstaller struct{}

// NewSystemdInstaller creates a new systemd installer
func NewSystemdInstaller() *SystemdInstaller {
	return &SystemdInstaller{}
}

// Name returns the init system name
func (i *SystemdInstaller) Name() string {
	return "systemd"
}

// Install installs systemd to the rootfs
func (i *SystemdInstaller) Install(rootfsPath string, component *ResolvedComponent) error {
	// Create systemd directories
	dirs := []string{
		"/etc/systemd",
		"/etc/systemd/system",
		"/etc/systemd/system/multi-user.target.wants",
		"/etc/systemd/system/sockets.target.wants",
		"/etc/systemd/system/sysinit.target.wants",
		"/etc/systemd/system/basic.target.wants",
		"/etc/systemd/system/getty.target.wants",
		"/etc/systemd/network",
		"/etc/systemd/resolved.conf.d",
		"/usr/lib/systemd",
		"/usr/lib/systemd/system",
		"/var/lib/systemd",
		"/var/log/journal",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(rootfsPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create systemd directory %s: %w", dir, err)
		}
	}

	// If component has extracted source, copy binaries
	if component != nil && component.LocalPath != "" {
		log.Info("Installing systemd from source", "path", component.LocalPath)
		// In a real implementation, we would:
		// 1. Build systemd from source or
		// 2. Extract pre-built binaries
		// For now, we create the structure for a minimal boot
	}

	// Create essential symlinks for systemd
	symlinks := []struct {
		target string
		link   string
	}{
		{"/usr/lib/systemd/systemd", "/sbin/init"},
		{"../systemd-networkd.service", "/etc/systemd/system/multi-user.target.wants/systemd-networkd.service"},
		{"../systemd-resolved.service", "/etc/systemd/system/multi-user.target.wants/systemd-resolved.service"},
	}

	for _, s := range symlinks {
		linkPath := filepath.Join(rootfsPath, s.link)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent dir for symlink: %w", err)
		}
		// Remove existing
		os.Remove(linkPath)
		if err := os.Symlink(s.target, linkPath); err != nil {
			// Non-fatal, the target might not exist yet
			log.Warn("Could not create symlink", "link", s.link, "target", s.target, "error", err)
		}
	}

	log.Info("Installed systemd structure")
	return nil
}

// Configure configures systemd
func (i *SystemdInstaller) Configure(rootfsPath string) error {
	// Create default target
	defaultTarget := filepath.Join(rootfsPath, "etc", "systemd", "system", "default.target")
	os.Remove(defaultTarget)
	if err := os.Symlink("/usr/lib/systemd/system/multi-user.target", defaultTarget); err != nil {
		log.Warn("Could not create default.target symlink", "error", err)
	}

	// Create machine-id placeholder
	machineID := filepath.Join(rootfsPath, "etc", "machine-id")
	if err := os.WriteFile(machineID, []byte(""), 0444); err != nil {
		return fmt.Errorf("failed to create machine-id: %w", err)
	}

	// Configure journald
	journaldConf := `[Journal]
Storage=persistent
Compress=yes
SystemMaxUse=500M
`
	journaldPath := filepath.Join(rootfsPath, "etc", "systemd", "journald.conf")
	if err := os.WriteFile(journaldPath, []byte(journaldConf), 0644); err != nil {
		return fmt.Errorf("failed to write journald.conf: %w", err)
	}

	// Configure networkd for DHCP
	networkConf := `[Match]
Name=*

[Network]
DHCP=yes
`
	networkPath := filepath.Join(rootfsPath, "etc", "systemd", "network", "80-dhcp.network")
	if err := os.WriteFile(networkPath, []byte(networkConf), 0644); err != nil {
		return fmt.Errorf("failed to write network config: %w", err)
	}

	// Create locale.conf
	localeConf := `LANG=en_US.UTF-8
`
	localePath := filepath.Join(rootfsPath, "etc", "locale.conf")
	if err := os.WriteFile(localePath, []byte(localeConf), 0644); err != nil {
		return fmt.Errorf("failed to write locale.conf: %w", err)
	}

	// Create vconsole.conf
	vconsoleConf := `KEYMAP=us
`
	vconsolePath := filepath.Join(rootfsPath, "etc", "vconsole.conf")
	if err := os.WriteFile(vconsolePath, []byte(vconsoleConf), 0644); err != nil {
		return fmt.Errorf("failed to write vconsole.conf: %w", err)
	}

	log.Info("Configured systemd")
	return nil
}

// EnableService enables a systemd service
func (i *SystemdInstaller) EnableService(rootfsPath, serviceName string) error {
	// Determine target directory based on service type
	targetDir := filepath.Join(rootfsPath, "etc", "systemd", "system", "multi-user.target.wants")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Create symlink
	serviceFile := fmt.Sprintf("/usr/lib/systemd/system/%s", serviceName)
	linkPath := filepath.Join(targetDir, serviceName)

	os.Remove(linkPath)
	if err := os.Symlink(serviceFile, linkPath); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
	}

	log.Info("Enabled systemd service", "service", serviceName)
	return nil
}

// OpenRCInstaller installs and configures OpenRC
type OpenRCInstaller struct{}

// NewOpenRCInstaller creates a new OpenRC installer
func NewOpenRCInstaller() *OpenRCInstaller {
	return &OpenRCInstaller{}
}

// Name returns the init system name
func (i *OpenRCInstaller) Name() string {
	return "openrc"
}

// Install installs OpenRC to the rootfs
func (i *OpenRCInstaller) Install(rootfsPath string, component *ResolvedComponent) error {
	// Create OpenRC directories
	dirs := []string{
		"/etc/init.d",
		"/etc/conf.d",
		"/etc/runlevels",
		"/etc/runlevels/boot",
		"/etc/runlevels/default",
		"/etc/runlevels/nonetwork",
		"/etc/runlevels/shutdown",
		"/etc/runlevels/sysinit",
		"/run/openrc",
		"/var/lib/misc",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(rootfsPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create OpenRC directory %s: %w", dir, err)
		}
	}

	// If component has extracted source, copy binaries
	if component != nil && component.LocalPath != "" {
		log.Info("Installing OpenRC from source", "path", component.LocalPath)
	}

	// Create /sbin/init symlink to openrc-init
	initLink := filepath.Join(rootfsPath, "sbin", "init")
	os.Remove(initLink)
	if err := os.Symlink("/sbin/openrc-init", initLink); err != nil {
		log.Warn("Could not create init symlink", "error", err)
	}

	log.Info("Installed OpenRC structure")
	return nil
}

// Configure configures OpenRC
func (i *OpenRCInstaller) Configure(rootfsPath string) error {
	// Create openrc.conf
	openrcConf := `# OpenRC configuration
rc_parallel="YES"
rc_logger="YES"
rc_log_path="/var/log/rc.log"
`
	openrcPath := filepath.Join(rootfsPath, "etc", "rc.conf")
	if err := os.WriteFile(openrcPath, []byte(openrcConf), 0644); err != nil {
		return fmt.Errorf("failed to write rc.conf: %w", err)
	}

	// Create hostname init script
	hostnameInit := `#!/sbin/openrc-run
description="Set system hostname"

depend() {
    need localmount
}

start() {
    ebegin "Setting hostname"
    hostname -F /etc/hostname
    eend $?
}
`
	hostnamePath := filepath.Join(rootfsPath, "etc", "init.d", "hostname")
	if err := os.WriteFile(hostnamePath, []byte(hostnameInit), 0755); err != nil {
		return fmt.Errorf("failed to write hostname init script: %w", err)
	}

	// Create network init script
	networkInit := `#!/sbin/openrc-run
description="Network configuration"

depend() {
    need localmount hostname
    after bootmisc
}

start() {
    ebegin "Configuring network interfaces"
    ip link set lo up
    # DHCP for primary interface
    if command -v dhcpcd >/dev/null 2>&1; then
        dhcpcd -b eth0
    elif command -v udhcpc >/dev/null 2>&1; then
        udhcpc -i eth0 -b
    fi
    eend $?
}

stop() {
    ebegin "Deconfiguring network interfaces"
    ip link set lo down
    eend $?
}
`
	networkPath := filepath.Join(rootfsPath, "etc", "init.d", "network")
	if err := os.WriteFile(networkPath, []byte(networkInit), 0755); err != nil {
		return fmt.Errorf("failed to write network init script: %w", err)
	}

	// Enable essential services in boot runlevel
	bootServices := []string{"hostname", "network"}
	for _, svc := range bootServices {
		if err := i.EnableService(rootfsPath, svc); err != nil {
			log.Warn("Could not enable boot service", "service", svc, "error", err)
		}
	}

	log.Info("Configured OpenRC")
	return nil
}

// EnableService enables an OpenRC service
func (i *OpenRCInstaller) EnableService(rootfsPath, serviceName string) error {
	// Determine runlevel (default to 'default')
	runlevel := "default"
	if serviceName == "hostname" || serviceName == "network" {
		runlevel = "boot"
	}

	runlevelDir := filepath.Join(rootfsPath, "etc", "runlevels", runlevel)
	if err := os.MkdirAll(runlevelDir, 0755); err != nil {
		return err
	}

	// Create symlink to init script
	initScript := fmt.Sprintf("/etc/init.d/%s", serviceName)
	linkPath := filepath.Join(runlevelDir, serviceName)

	os.Remove(linkPath)
	if err := os.Symlink(initScript, linkPath); err != nil {
		return fmt.Errorf("failed to enable service %s in %s: %w", serviceName, runlevel, err)
	}

	log.Info("Enabled OpenRC service", "service", serviceName, "runlevel", runlevel)
	return nil
}

// GetInitInstaller returns the appropriate init installer for the config
func GetInitInstaller(initSystem string) InitInstaller {
	switch strings.ToLower(initSystem) {
	case "systemd":
		return NewSystemdInstaller()
	case "openrc":
		return NewOpenRCInstaller()
	default:
		// Default to systemd
		return NewSystemdInstaller()
	}
}
