package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Composite release workflow commands",
	Long: `Composite commands for managing distribution releases.

A release is a versioned distribution with a specific configuration of
components (kernel, init system, filesystem, security, etc.).`,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new distribution release",
	Long: `Creates a new distribution with a version and optional initial configuration.

Example:
  ldfctl release create --name my-distro --version 0.0.1 --visibility private`,
	RunE: runReleaseCreate,
}

var releaseConfigureCmd = &cobra.Command{
	Use:   "configure <distribution-id>",
	Short: "Configure components for a distribution release",
	Long: `Sets the component configuration for an existing distribution.
Only specified flags are applied; unset flags leave the existing configuration unchanged.

Example:
  ldfctl release configure abc123 --kernel 6.7.1 --init systemd --filesystem ext4 --target-type server`,
	Args: cobra.ExactArgs(1),
	RunE: runReleaseConfigure,
}

var releaseShowCmd = &cobra.Command{
	Use:   "show <distribution-id>",
	Short: "Show the full configuration of a distribution release",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseShow,
}

func init() {
	releaseCmd.AddCommand(releaseCreateCmd)
	releaseCmd.AddCommand(releaseConfigureCmd)
	releaseCmd.AddCommand(releaseShowCmd)

	// Create flags
	releaseCreateCmd.Flags().String("name", "", "Distribution name (required)")
	releaseCreateCmd.Flags().String("version", "", "Distribution version (required)")
	releaseCreateCmd.Flags().String("visibility", "private", "Visibility (public, private)")
	_ = releaseCreateCmd.MarkFlagRequired("name")
	_ = releaseCreateCmd.MarkFlagRequired("version")

	// Configure flags -- core
	releaseConfigureCmd.Flags().String("kernel", "", "Kernel version")
	releaseConfigureCmd.Flags().String("bootloader", "", "Bootloader (e.g., grub, systemd-boot)")
	releaseConfigureCmd.Flags().String("bootloader-version", "", "Bootloader version")
	releaseConfigureCmd.Flags().String("partitioning-type", "", "Partitioning type (e.g., gpt, mbr)")
	releaseConfigureCmd.Flags().String("partitioning-mode", "", "Partitioning mode (e.g., auto, manual)")

	// Configure flags -- system
	releaseConfigureCmd.Flags().String("init", "", "Init system (e.g., systemd, openrc)")
	releaseConfigureCmd.Flags().String("init-version", "", "Init system version")
	releaseConfigureCmd.Flags().String("filesystem", "", "Filesystem type (e.g., ext4, btrfs, xfs)")
	releaseConfigureCmd.Flags().String("filesystem-hierarchy", "", "Filesystem hierarchy (e.g., fhs, usr-merge)")
	releaseConfigureCmd.Flags().String("filesystem-version", "", "Filesystem version")
	releaseConfigureCmd.Flags().String("package-manager", "", "Package manager (e.g., apt, dnf, pacman)")
	releaseConfigureCmd.Flags().String("package-manager-version", "", "Package manager version")

	// Configure flags -- security
	releaseConfigureCmd.Flags().String("security", "", "Security system (e.g., selinux, apparmor)")
	releaseConfigureCmd.Flags().String("security-version", "", "Security system version")

	// Configure flags -- runtime
	releaseConfigureCmd.Flags().String("container", "", "Container runtime (e.g., docker, podman)")
	releaseConfigureCmd.Flags().String("container-version", "", "Container runtime version")
	releaseConfigureCmd.Flags().String("virtualization", "", "Virtualization system (e.g., kvm, xen)")
	releaseConfigureCmd.Flags().String("virtualization-version", "", "Virtualization system version")

	// Configure flags -- target
	releaseConfigureCmd.Flags().String("target-type", "", "Target type (server, desktop)")
	releaseConfigureCmd.Flags().String("desktop-env", "", "Desktop environment (e.g., gnome, kde)")
	releaseConfigureCmd.Flags().String("desktop-env-version", "", "Desktop environment version")
	releaseConfigureCmd.Flags().String("display-server", "", "Display server (e.g., wayland, x11)")
	releaseConfigureCmd.Flags().String("display-server-version", "", "Display server version")
}

func runReleaseCreate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	name, _ := cmd.Flags().GetString("name")
	version, _ := cmd.Flags().GetString("version")
	visibility, _ := cmd.Flags().GetString("visibility")

	req := &client.CreateDistributionRequest{
		Name:       name,
		Version:    version,
		Visibility: visibility,
	}

	resp, err := c.CreateDistribution(ctx, req)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	output.PrintMessage(fmt.Sprintf("Release %q v%s created (ID: %s)", resp.Name, resp.Version, resp.ID))
	return nil
}

func runReleaseConfigure(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()
	distID := args[0]

	// Fetch current distribution to get existing config
	dist, err := c.GetDistribution(ctx, distID)
	if err != nil {
		return fmt.Errorf("failed to fetch distribution: %w", err)
	}

	// Parse existing config into a map we can merge into
	config := make(map[string]interface{})
	if dist.Config != nil {
		raw, _ := json.Marshal(dist.Config)
		_ = json.Unmarshal(raw, &config)
	}

	// Build config from flags, merging with existing
	changed := false

	// Core
	if cmd.Flags().Changed("kernel") {
		v, _ := cmd.Flags().GetString("kernel")
		ensureMap(config, "core")
		coreMap := config["core"].(map[string]interface{})
		coreMap["kernel"] = map[string]interface{}{"version": v}
		changed = true
	}
	if cmd.Flags().Changed("bootloader") {
		v, _ := cmd.Flags().GetString("bootloader")
		ensureMap(config, "core")
		config["core"].(map[string]interface{})["bootloader"] = v
		changed = true
	}
	if cmd.Flags().Changed("bootloader-version") {
		v, _ := cmd.Flags().GetString("bootloader-version")
		ensureMap(config, "core")
		config["core"].(map[string]interface{})["bootloader_version"] = v
		changed = true
	}
	if cmd.Flags().Changed("partitioning-type") || cmd.Flags().Changed("partitioning-mode") {
		ensureMap(config, "core")
		coreMap := config["core"].(map[string]interface{})
		partMap, ok := coreMap["partitioning"].(map[string]interface{})
		if !ok {
			partMap = make(map[string]interface{})
		}
		if cmd.Flags().Changed("partitioning-type") {
			v, _ := cmd.Flags().GetString("partitioning-type")
			partMap["type"] = v
		}
		if cmd.Flags().Changed("partitioning-mode") {
			v, _ := cmd.Flags().GetString("partitioning-mode")
			partMap["mode"] = v
		}
		coreMap["partitioning"] = partMap
		changed = true
	}

	// System
	if cmd.Flags().Changed("init") {
		v, _ := cmd.Flags().GetString("init")
		ensureMap(config, "system")
		config["system"].(map[string]interface{})["init"] = v
		changed = true
	}
	if cmd.Flags().Changed("init-version") {
		v, _ := cmd.Flags().GetString("init-version")
		ensureMap(config, "system")
		config["system"].(map[string]interface{})["init_version"] = v
		changed = true
	}
	if cmd.Flags().Changed("filesystem") || cmd.Flags().Changed("filesystem-hierarchy") {
		ensureMap(config, "system")
		sysMap := config["system"].(map[string]interface{})
		fsMap, ok := sysMap["filesystem"].(map[string]interface{})
		if !ok {
			fsMap = make(map[string]interface{})
		}
		if cmd.Flags().Changed("filesystem") {
			v, _ := cmd.Flags().GetString("filesystem")
			fsMap["type"] = v
		}
		if cmd.Flags().Changed("filesystem-hierarchy") {
			v, _ := cmd.Flags().GetString("filesystem-hierarchy")
			fsMap["hierarchy"] = v
		}
		sysMap["filesystem"] = fsMap
		changed = true
	}
	if cmd.Flags().Changed("filesystem-version") {
		v, _ := cmd.Flags().GetString("filesystem-version")
		ensureMap(config, "system")
		config["system"].(map[string]interface{})["filesystem_version"] = v
		changed = true
	}
	if cmd.Flags().Changed("package-manager") {
		v, _ := cmd.Flags().GetString("package-manager")
		ensureMap(config, "system")
		config["system"].(map[string]interface{})["packageManager"] = v
		changed = true
	}
	if cmd.Flags().Changed("package-manager-version") {
		v, _ := cmd.Flags().GetString("package-manager-version")
		ensureMap(config, "system")
		config["system"].(map[string]interface{})["package_manager_version"] = v
		changed = true
	}

	// Security
	if cmd.Flags().Changed("security") {
		v, _ := cmd.Flags().GetString("security")
		ensureMap(config, "security")
		config["security"].(map[string]interface{})["system"] = v
		changed = true
	}
	if cmd.Flags().Changed("security-version") {
		v, _ := cmd.Flags().GetString("security-version")
		ensureMap(config, "security")
		config["security"].(map[string]interface{})["system_version"] = v
		changed = true
	}

	// Runtime
	if cmd.Flags().Changed("container") {
		v, _ := cmd.Flags().GetString("container")
		ensureMap(config, "runtime")
		config["runtime"].(map[string]interface{})["container"] = v
		changed = true
	}
	if cmd.Flags().Changed("container-version") {
		v, _ := cmd.Flags().GetString("container-version")
		ensureMap(config, "runtime")
		config["runtime"].(map[string]interface{})["container_version"] = v
		changed = true
	}
	if cmd.Flags().Changed("virtualization") {
		v, _ := cmd.Flags().GetString("virtualization")
		ensureMap(config, "runtime")
		config["runtime"].(map[string]interface{})["virtualization"] = v
		changed = true
	}
	if cmd.Flags().Changed("virtualization-version") {
		v, _ := cmd.Flags().GetString("virtualization-version")
		ensureMap(config, "runtime")
		config["runtime"].(map[string]interface{})["virtualization_version"] = v
		changed = true
	}

	// Target
	if cmd.Flags().Changed("target-type") {
		v, _ := cmd.Flags().GetString("target-type")
		ensureMap(config, "target")
		config["target"].(map[string]interface{})["type"] = v
		changed = true
	}
	if cmd.Flags().Changed("desktop-env") || cmd.Flags().Changed("desktop-env-version") ||
		cmd.Flags().Changed("display-server") || cmd.Flags().Changed("display-server-version") {
		ensureMap(config, "target")
		targetMap := config["target"].(map[string]interface{})
		deskMap, ok := targetMap["desktop"].(map[string]interface{})
		if !ok {
			deskMap = make(map[string]interface{})
		}
		if cmd.Flags().Changed("desktop-env") {
			v, _ := cmd.Flags().GetString("desktop-env")
			deskMap["environment"] = v
		}
		if cmd.Flags().Changed("desktop-env-version") {
			v, _ := cmd.Flags().GetString("desktop-env-version")
			deskMap["environment_version"] = v
		}
		if cmd.Flags().Changed("display-server") {
			v, _ := cmd.Flags().GetString("display-server")
			deskMap["displayServer"] = v
		}
		if cmd.Flags().Changed("display-server-version") {
			v, _ := cmd.Flags().GetString("display-server-version")
			deskMap["display_server_version"] = v
		}
		targetMap["desktop"] = deskMap
		changed = true
	}

	if !changed {
		output.PrintMessage("No configuration flags specified. Use --help to see available options.")
		return nil
	}

	req := &client.UpdateDistributionRequest{
		Config: config,
	}

	resp, err := c.UpdateDistribution(ctx, distID, req)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	output.PrintMessage(fmt.Sprintf("Release %q configured.", resp.Name))
	return nil
}

func runReleaseShow(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDistribution(ctx, args[0])
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	// Print distribution info
	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"ID", resp.ID},
			{"Name", resp.Name},
			{"Version", resp.Version},
			{"Status", resp.Status},
			{"Visibility", resp.Visibility},
		},
	)

	// Print config if present
	if resp.Config != nil {
		fmt.Println()
		output.PrintMessage("Configuration:")
		configBytes, err := json.MarshalIndent(resp.Config, "  ", "  ")
		if err == nil {
			fmt.Printf("  %s\n", string(configBytes))
		}
	} else {
		fmt.Println()
		output.PrintMessage("No configuration set. Use 'ldfctl release configure' to set components.")
	}

	return nil
}

// ensureMap ensures a key exists in the map as a map[string]interface{}
func ensureMap(m map[string]interface{}, key string) {
	if _, ok := m[key]; !ok {
		m[key] = make(map[string]interface{})
	}
	if _, ok := m[key].(map[string]interface{}); !ok {
		m[key] = make(map[string]interface{})
	}
}
