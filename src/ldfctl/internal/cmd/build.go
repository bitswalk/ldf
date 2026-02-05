package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Manage builds",
}

var buildStartCmd = &cobra.Command{
	Use:   "start <distribution-id>",
	Short: "Start a build for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildStart,
}

var buildGetCmd = &cobra.Command{
	Use:   "get <build-id>",
	Short: "Get a build job by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildGet,
}

var buildListCmd = &cobra.Command{
	Use:   "list <distribution-id>",
	Short: "List builds for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildList,
}

var buildLogsCmd = &cobra.Command{
	Use:   "logs <build-id>",
	Short: "Get build logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildLogs,
}

var buildCancelCmd = &cobra.Command{
	Use:   "cancel <build-id>",
	Short: "Cancel a running build",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildCancel,
}

var buildRetryCmd = &cobra.Command{
	Use:   "retry <build-id>",
	Short: "Retry a failed build",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildRetry,
}

var buildActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "List all active builds",
	RunE:  runBuildActive,
}

func init() {
	buildCmd.AddCommand(buildStartCmd)
	buildCmd.AddCommand(buildGetCmd)
	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildLogsCmd)
	buildCmd.AddCommand(buildCancelCmd)
	buildCmd.AddCommand(buildRetryCmd)
	buildCmd.AddCommand(buildActiveCmd)

	// Start flags
	buildStartCmd.Flags().String("arch", "x86_64", "Target architecture (x86_64, aarch64)")
	buildStartCmd.Flags().String("format", "raw", "Image format (raw, qcow2, iso)")

	// List flags
	buildListCmd.Flags().Int("limit", 0, "Maximum number of results")
	buildListCmd.Flags().Int("offset", 0, "Number of results to skip")
}

func runBuildStart(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	arch, _ := cmd.Flags().GetString("arch")
	format, _ := cmd.Flags().GetString("format")

	req := &client.StartBuildRequest{
		Arch:   arch,
		Format: format,
	}

	resp, err := c.StartBuild(ctx, args[0], req)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	default:
		output.PrintMessage(fmt.Sprintf("Build %s started for distribution %s.", resp.ID, args[0]))
		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"Build ID", resp.ID},
				{"Architecture", resp.TargetArch},
				{"Format", resp.ImageFormat},
				{"Status", resp.Status},
			},
		)
		return nil
	}
}

func runBuildGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetBuild(ctx, args[0])
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	default:
		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", resp.ID},
				{"Distribution", resp.DistributionID},
				{"Status", resp.Status},
				{"Current Stage", resp.CurrentStage},
				{"Progress", fmt.Sprintf("%d%%", resp.ProgressPercent)},
				{"Architecture", resp.TargetArch},
				{"Format", resp.ImageFormat},
				{"Error", resp.ErrorMessage},
				{"Error Stage", resp.ErrorStage},
				{"Created", resp.CreatedAt},
				{"Started", resp.StartedAt},
				{"Completed", resp.CompletedAt},
			},
		)

		if len(resp.Stages) > 0 {
			fmt.Println()
			output.PrintMessage("Stages:")
			rows := make([][]string, len(resp.Stages))
			for i, s := range resp.Stages {
				duration := ""
				if s.DurationMs > 0 {
					duration = fmt.Sprintf("%dms", s.DurationMs)
				}
				rows[i] = []string{s.Name, s.Status, fmt.Sprintf("%d%%", s.ProgressPercent), duration, s.ErrorMessage}
			}
			output.PrintTable([]string{"STAGE", "STATUS", "PROGRESS", "DURATION", "ERROR"}, rows)
		}
		return nil
	}
}

func runBuildList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	opts := &client.ListOptions{}
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Offset, _ = cmd.Flags().GetInt("offset")

	resp, err := c.ListDistributionBuilds(ctx, args[0], opts)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	default:
		if resp.Count == 0 {
			output.PrintMessage("No builds found.")
			return nil
		}

		rows := make([][]string, len(resp.Builds))
		for i, b := range resp.Builds {
			rows[i] = []string{b.ID, b.Status, b.CurrentStage, fmt.Sprintf("%d%%", b.ProgressPercent), b.TargetArch, b.ImageFormat}
		}
		output.PrintTable([]string{"ID", "STATUS", "STAGE", "PROGRESS", "ARCH", "FORMAT"}, rows)
		return nil
	}
}

func runBuildLogs(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetBuildLogs(ctx, args[0])
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	default:
		if resp.Count == 0 {
			output.PrintMessage("No build logs found.")
			return nil
		}

		rows := make([][]string, len(resp.Logs))
		for i, l := range resp.Logs {
			rows[i] = []string{l.CreatedAt, l.Stage, l.Level, l.Message}
		}
		output.PrintTable([]string{"TIME", "STAGE", "LEVEL", "MESSAGE"}, rows)
		return nil
	}
}

func runBuildCancel(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.CancelBuild(ctx, args[0]); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Build cancelled", "id": args[0]})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Build cancelled", "id": args[0]})
	default:
		output.PrintMessage(fmt.Sprintf("Build %s cancelled.", args[0]))
		return nil
	}
}

func runBuildRetry(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.RetryBuild(ctx, args[0]); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Build retry started", "id": args[0]})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Build retry started", "id": args[0]})
	default:
		output.PrintMessage(fmt.Sprintf("Build %s retry started.", args[0]))
		return nil
	}
}

func runBuildActive(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListActiveBuilds(ctx)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	default:
		if resp.Count == 0 {
			output.PrintMessage("No active builds.")
			return nil
		}

		rows := make([][]string, len(resp.Builds))
		for i, b := range resp.Builds {
			rows[i] = []string{b.ID, b.DistributionID, b.Status, b.CurrentStage, fmt.Sprintf("%d%%", b.ProgressPercent), b.TargetArch}
		}
		output.PrintTable([]string{"ID", "DISTRIBUTION", "STATUS", "STAGE", "PROGRESS", "ARCH"}, rows)
		return nil
	}
}
