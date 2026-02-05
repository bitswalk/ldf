package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var forgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge detection and filtering",
}

var forgeDetectCmd = &cobra.Command{
	Use:   "detect <url>",
	Short: "Detect forge type for a URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runForgeDetect,
}

var forgePreviewFilterCmd = &cobra.Command{
	Use:   "preview-filter <url>",
	Short: "Preview version filtering for a URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runForgePreviewFilter,
}

var forgeTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List supported forge types",
	RunE:  runForgeTypes,
}

var forgeFiltersCmd = &cobra.Command{
	Use:   "filters",
	Short: "List common filter presets",
	RunE:  runForgeFilters,
}

func init() {
	forgeCmd.AddCommand(forgeDetectCmd)
	forgeCmd.AddCommand(forgePreviewFilterCmd)
	forgeCmd.AddCommand(forgeTypesCmd)
	forgeCmd.AddCommand(forgeFiltersCmd)

	forgePreviewFilterCmd.Flags().String("forge-type", "", "Override forge type")
	forgePreviewFilterCmd.Flags().String("version-filter", "", "Version filter pattern")
}

func runForgeDetect(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.DetectForge(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"Forge Type", resp.ForgeType},
		},
	)
	return nil
}

func runForgePreviewFilter(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	forgeType, _ := cmd.Flags().GetString("forge-type")
	versionFilter, _ := cmd.Flags().GetString("version-filter")

	req := &client.PreviewFilterRequest{
		URL:           args[0],
		ForgeType:     forgeType,
		VersionFilter: versionFilter,
	}

	resp, err := c.PreviewFilter(ctx, req)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("Filter: %s (source: %s)", resp.AppliedFilter, resp.FilterSource))
	output.PrintMessage(fmt.Sprintf("Total: %d, Included: %d, Excluded: %d\n",
		resp.TotalVersions, resp.IncludedVersions, resp.ExcludedVersions))

	rows := make([][]string, len(resp.Versions))
	for i, v := range resp.Versions {
		included := "yes"
		if !v.Included {
			included = "no"
		}
		rows[i] = []string{v.Version, included, v.Reason}
	}
	output.PrintTable([]string{"VERSION", "INCLUDED", "REASON"}, rows)
	return nil
}

func runForgeTypes(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListForgeTypes(ctx)
	if err != nil {
		return err
	}

	return output.PrintJSON(resp)
}

func runForgeFilters(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetCommonFilters(ctx)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	rows := make([][]string, 0, len(resp.Filters))
	for name, pattern := range resp.Filters {
		rows = append(rows, []string{name, pattern})
	}
	output.PrintTable([]string{"NAME", "PATTERN"}, rows)
	return nil
}
