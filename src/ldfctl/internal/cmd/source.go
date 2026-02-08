package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var sourceCmd = &cobra.Command{
	Use:     "source",
	Aliases: []string{"src"},
	Short:   "Manage sources",
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sources",
	RunE:  runSourceList,
}

var sourceGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a source by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceGet,
}

var sourceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new source",
	RunE:  runSourceCreate,
}

var sourceUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceUpdate,
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceDelete,
}

var sourceSyncCmd = &cobra.Command{
	Use:   "sync <id>",
	Short: "Trigger version sync for a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceSync,
}

var sourceVersionsCmd = &cobra.Command{
	Use:   "versions <id>",
	Short: "List discovered versions for a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceVersions,
}

var sourceSyncStatusCmd = &cobra.Command{
	Use:   "sync-status <id>",
	Short: "Get sync status for a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceSyncStatus,
}

var sourceClearVersionsCmd = &cobra.Command{
	Use:   "clear-versions <id>",
	Short: "Clear cached versions for a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceClearVersions,
}

func init() {
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceGetCmd)
	sourceCmd.AddCommand(sourceCreateCmd)
	sourceCmd.AddCommand(sourceUpdateCmd)
	sourceCmd.AddCommand(sourceDeleteCmd)
	sourceCmd.AddCommand(sourceSyncCmd)
	sourceCmd.AddCommand(sourceVersionsCmd)
	sourceCmd.AddCommand(sourceSyncStatusCmd)
	sourceCmd.AddCommand(sourceClearVersionsCmd)

	sourceCreateCmd.Flags().String("name", "", "Source name (required)")
	sourceCreateCmd.Flags().String("url", "", "Source URL (required)")
	sourceCreateCmd.Flags().String("component-id", "", "Component ID (required)")
	sourceCreateCmd.Flags().String("version-filter", "", "Version filter pattern")
	_ = sourceCreateCmd.MarkFlagRequired("name")
	_ = sourceCreateCmd.MarkFlagRequired("url")
	_ = sourceCreateCmd.MarkFlagRequired("component-id")

	sourceUpdateCmd.Flags().String("name", "", "Source name")
	sourceUpdateCmd.Flags().String("url", "", "Source URL")
	sourceUpdateCmd.Flags().String("version-filter", "", "Version filter pattern")

	// List flags
	sourceListCmd.Flags().Int("limit", 0, "Maximum number of results")
	sourceListCmd.Flags().Int("offset", 0, "Number of results to skip")

	// Versions flags
	sourceVersionsCmd.Flags().Int("limit", 0, "Maximum number of results")
	sourceVersionsCmd.Flags().Int("offset", 0, "Number of results to skip")
	sourceVersionsCmd.Flags().String("version-type", "", "Filter by version type")
}

func runSourceList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	opts := &client.ListOptions{}
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Offset, _ = cmd.Flags().GetInt("offset")

	resp, err := c.ListSources(ctx, opts)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		if resp.Count == 0 {
			output.PrintMessage("No sources found.")
			return nil
		}

		rows := make([][]string, len(resp.Sources))
		for i, s := range resp.Sources {
			srcType := "user"
			if s.IsSystem {
				srcType = "system"
			}
			rows[i] = []string{s.ID, s.Name, s.ForgeType, srcType, s.LastSyncStatus, fmt.Sprintf("%d", s.VersionCount)}
		}
		output.PrintTable([]string{"ID", "NAME", "FORGE", "TYPE", "SYNC STATUS", "VERSIONS"}, rows)
		return nil
	})
}

func runSourceGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetSource(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		isSystem := "no"
		if resp.IsSystem {
			isSystem = "yes"
		}

		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", resp.ID},
				{"Name", resp.Name},
				{"URL", resp.URL},
				{"Forge Type", resp.ForgeType},
				{"Component ID", resp.ComponentID},
				{"System", isSystem},
				{"Version Filter", resp.VersionFilter},
				{"Last Sync", resp.LastSyncAt},
				{"Sync Status", resp.LastSyncStatus},
				{"Versions", fmt.Sprintf("%d", resp.VersionCount)},
				{"Created", resp.CreatedAt},
				{"Updated", resp.UpdatedAt},
			},
		)
		return nil
	})
}

func runSourceCreate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	name, _ := cmd.Flags().GetString("name")
	url, _ := cmd.Flags().GetString("url")
	componentID, _ := cmd.Flags().GetString("component-id")
	versionFilter, _ := cmd.Flags().GetString("version-filter")

	req := &client.CreateSourceRequest{
		Name:          name,
		URL:           url,
		ComponentID:   componentID,
		VersionFilter: versionFilter,
	}

	resp, err := c.CreateSource(ctx, req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Source %q created (ID: %s)", resp.Name, resp.ID))
		return nil
	})
}

func runSourceUpdate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	req := &client.UpdateSourceRequest{}
	if cmd.Flags().Changed("name") {
		v, _ := cmd.Flags().GetString("name")
		req.Name = v
	}
	if cmd.Flags().Changed("url") {
		v, _ := cmd.Flags().GetString("url")
		req.URL = v
	}
	if cmd.Flags().Changed("version-filter") {
		v, _ := cmd.Flags().GetString("version-filter")
		req.VersionFilter = v
	}

	resp, err := c.UpdateSource(ctx, args[0], req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Source %q updated.", resp.Name))
		return nil
	})
}

func runSourceDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteSource(ctx, args[0]); err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), map[string]string{"message": "Source deleted", "id": args[0]}, func() error {

		output.PrintMessage(fmt.Sprintf("Source %s deleted.", args[0]))
		return nil
	})
}

func runSourceSync(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.SyncSource(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Sync triggered for source %s: %s", resp.SourceID, resp.Message))
		return nil
	})
}

func runSourceVersions(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	opts := &client.ListOptions{}
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Offset, _ = cmd.Flags().GetInt("offset")
	opts.VersionType, _ = cmd.Flags().GetString("version-type")

	resp, err := c.ListSourceVersions(ctx, args[0], opts)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		if resp.Count == 0 {
			output.PrintMessage("No versions found.")
			return nil
		}

		rows := make([][]string, len(resp.Versions))
		for i, v := range resp.Versions {
			rows[i] = []string{v.Version, v.Type, v.URL}
		}
		output.PrintTable([]string{"VERSION", "TYPE", "URL"}, rows)
		return nil
	})
}

func runSourceSyncStatus(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetSyncStatus(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"Source ID", resp.SourceID},
				{"Status", resp.Status},
				{"Last Sync", resp.LastSyncAt},
				{"Versions", fmt.Sprintf("%d", resp.VersionCount)},
				{"Error", resp.Error},
			},
		)
		return nil
	})
}

func runSourceClearVersions(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.ClearSourceVersions(ctx, args[0]); err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), map[string]string{"message": "Versions cleared", "source_id": args[0]}, func() error {

		output.PrintMessage(fmt.Sprintf("Versions cleared for source %s.", args[0]))
		return nil
	})
}
