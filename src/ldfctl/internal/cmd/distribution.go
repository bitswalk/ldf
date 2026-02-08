package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var distributionCmd = &cobra.Command{
	Use:     "distribution",
	Aliases: []string{"dist"},
	Short:   "Manage distributions",
}

var distributionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all distributions",
	RunE:  runDistributionList,
}

var distributionGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a distribution by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDistributionGet,
}

var distributionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new distribution",
	RunE:  runDistributionCreate,
}

var distributionUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runDistributionUpdate,
}

var distributionDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runDistributionDelete,
}

var distributionLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Get logs for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runDistributionLogs,
}

var distributionStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Get distribution statistics",
	RunE:  runDistributionStats,
}

var distributionDeletionPreviewCmd = &cobra.Command{
	Use:   "deletion-preview <id>",
	Short: "Preview what will be deleted",
	Args:  cobra.ExactArgs(1),
	RunE:  runDistributionDeletionPreview,
}

func init() {
	distributionCmd.AddCommand(distributionListCmd)
	distributionCmd.AddCommand(distributionGetCmd)
	distributionCmd.AddCommand(distributionCreateCmd)
	distributionCmd.AddCommand(distributionUpdateCmd)
	distributionCmd.AddCommand(distributionDeleteCmd)
	distributionCmd.AddCommand(distributionLogsCmd)
	distributionCmd.AddCommand(distributionStatsCmd)
	distributionCmd.AddCommand(distributionDeletionPreviewCmd)

	// Create flags
	distributionCreateCmd.Flags().String("name", "", "Distribution name (required)")
	distributionCreateCmd.Flags().String("version", "", "Distribution version")
	distributionCreateCmd.Flags().String("visibility", "", "Visibility (public, private)")
	distributionCreateCmd.Flags().String("source-url", "", "Source URL")
	distributionCreateCmd.Flags().String("checksum", "", "Checksum")
	distributionCreateCmd.Flags().String("toolchain", "gcc", "Build toolchain (gcc, llvm)")
	_ = distributionCreateCmd.MarkFlagRequired("name")

	// Update flags
	distributionUpdateCmd.Flags().String("name", "", "Distribution name")
	distributionUpdateCmd.Flags().String("version", "", "Distribution version")
	distributionUpdateCmd.Flags().String("status", "", "Distribution status")
	distributionUpdateCmd.Flags().String("visibility", "", "Visibility (public, private)")
	distributionUpdateCmd.Flags().String("source-url", "", "Source URL")
	distributionUpdateCmd.Flags().String("checksum", "", "Checksum")
	distributionUpdateCmd.Flags().String("toolchain", "", "Build toolchain (gcc, llvm)")

	// List flags
	distributionListCmd.Flags().Int("limit", 0, "Maximum number of results")
	distributionListCmd.Flags().Int("offset", 0, "Number of results to skip")
	distributionListCmd.Flags().String("status", "", "Filter by status (pending, downloading, validating, ready, failed)")
}

func runDistributionList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	opts := &client.ListOptions{}
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Offset, _ = cmd.Flags().GetInt("offset")
	opts.Status, _ = cmd.Flags().GetString("status")

	resp, err := c.ListDistributions(ctx, opts)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		if resp.Count == 0 {
			output.PrintMessage("No distributions found.")
			return nil
		}

		rows := make([][]string, len(resp.Distributions))
		for i, d := range resp.Distributions {
			rows[i] = []string{d.ID, d.Name, d.Version, d.Status, d.Visibility}
		}
		output.PrintTable([]string{"ID", "NAME", "VERSION", "STATUS", "VISIBILITY"}, rows)
		return nil
	})
}

func runDistributionGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDistribution(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", resp.ID},
				{"Name", resp.Name},
				{"Version", resp.Version},
				{"Status", resp.Status},
				{"Visibility", resp.Visibility},
				{"Source URL", resp.SourceURL},
				{"Created", resp.CreatedAt},
				{"Updated", resp.UpdatedAt},
			},
		)
		return nil
	})
}

func runDistributionCreate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	name, _ := cmd.Flags().GetString("name")
	version, _ := cmd.Flags().GetString("version")
	visibility, _ := cmd.Flags().GetString("visibility")
	sourceURL, _ := cmd.Flags().GetString("source-url")
	checksum, _ := cmd.Flags().GetString("checksum")
	toolchain, _ := cmd.Flags().GetString("toolchain")

	req := &client.CreateDistributionRequest{
		Name:       name,
		Version:    version,
		Visibility: visibility,
		SourceURL:  sourceURL,
		Checksum:   checksum,
	}

	if toolchain != "" && toolchain != "gcc" {
		req.Config = map[string]interface{}{
			"core": map[string]interface{}{
				"toolchain": toolchain,
			},
		}
	}

	resp, err := c.CreateDistribution(ctx, req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Distribution %q created (ID: %s)", resp.Name, resp.ID))
		return nil
	})
}

func runDistributionUpdate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	req := &client.UpdateDistributionRequest{}
	setStringIfChanged(cmd, "name", &req.Name)
	setStringIfChanged(cmd, "version", &req.Version)
	setStringIfChanged(cmd, "status", &req.Status)
	setStringIfChanged(cmd, "visibility", &req.Visibility)
	setStringIfChanged(cmd, "source-url", &req.SourceURL)
	setStringIfChanged(cmd, "checksum", &req.Checksum)
	if cmd.Flags().Changed("toolchain") {
		v, _ := cmd.Flags().GetString("toolchain")
		req.Config = map[string]interface{}{
			"core": map[string]interface{}{
				"toolchain": v,
			},
		}
	}

	resp, err := c.UpdateDistribution(ctx, args[0], req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Distribution %q updated.", resp.Name))
		return nil
	})
}

func runDistributionDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteDistribution(ctx, args[0]); err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), map[string]string{"message": "Distribution deleted", "id": args[0]}, func() error {

		output.PrintMessage(fmt.Sprintf("Distribution %s deleted.", args[0]))
		return nil
	})
}

func runDistributionLogs(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDistributionLogs(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintJSON(resp)
}

func runDistributionStats(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDistributionStats(ctx)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Total distributions: %d", resp.Total))
		if len(resp.Stats) > 0 {
			rows := make([][]string, 0, len(resp.Stats))
			for status, count := range resp.Stats {
				rows = append(rows, []string{status, fmt.Sprintf("%d", count)})
			}
			output.PrintTable([]string{"STATUS", "COUNT"}, rows)
		}
		return nil
	})
}

func runDistributionDeletionPreview(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDeletionPreview(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintJSON(resp)
}
