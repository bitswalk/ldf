package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"dl"},
	Short:   "Manage downloads",
}

var downloadListCmd = &cobra.Command{
	Use:   "list <distribution-id>",
	Short: "List downloads for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runDownloadList,
}

var downloadGetCmd = &cobra.Command{
	Use:   "get <job-id>",
	Short: "Get a download job by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runDownloadGet,
}

var downloadStartCmd = &cobra.Command{
	Use:   "start <distribution-id>",
	Short: "Start downloads for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runDownloadStart,
}

var downloadCancelCmd = &cobra.Command{
	Use:   "cancel <job-id>",
	Short: "Cancel a running download",
	Args:  cobra.ExactArgs(1),
	RunE:  runDownloadCancel,
}

var downloadRetryCmd = &cobra.Command{
	Use:   "retry <job-id>",
	Short: "Retry a failed download",
	Args:  cobra.ExactArgs(1),
	RunE:  runDownloadRetry,
}

func init() {
	downloadCmd.AddCommand(downloadListCmd)
	downloadCmd.AddCommand(downloadGetCmd)
	downloadCmd.AddCommand(downloadStartCmd)
	downloadCmd.AddCommand(downloadCancelCmd)
	downloadCmd.AddCommand(downloadRetryCmd)
}

func runDownloadList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListDistributionDownloads(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	if resp.Count == 0 {
		output.PrintMessage("No downloads found.")
		return nil
	}

	rows := make([][]string, len(resp.Jobs))
	for i, j := range resp.Jobs {
		rows[i] = []string{j.ID, j.Status, fmt.Sprintf("%d%%", j.Progress), j.SourceURL}
	}
	output.PrintTable([]string{"ID", "STATUS", "PROGRESS", "SOURCE"}, rows)
	return nil
}

func runDownloadGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetDownloadJob(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"ID", resp.ID},
			{"Distribution", resp.DistributionID},
			{"Component", resp.ComponentID},
			{"Source URL", resp.SourceURL},
			{"Status", resp.Status},
			{"Progress", fmt.Sprintf("%d%%", resp.Progress)},
			{"Total Bytes", fmt.Sprintf("%d", resp.TotalBytes)},
			{"Downloaded", fmt.Sprintf("%d", resp.DownloadedBytes)},
			{"Error", resp.Error},
			{"Created", resp.CreatedAt},
			{"Updated", resp.UpdatedAt},
		},
	)
	return nil
}

func runDownloadStart(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.StartDistributionDownloads(ctx, args[0], nil)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("Downloads started for distribution %s.", args[0]))
	return nil
}

func runDownloadCancel(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.CancelDownload(ctx, args[0]); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Download cancelled", "id": args[0]})
	}

	output.PrintMessage(fmt.Sprintf("Download %s cancelled.", args[0]))
	return nil
}

func runDownloadRetry(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.RetryDownload(ctx, args[0]); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Download retry started", "id": args[0]})
	}

	output.PrintMessage(fmt.Sprintf("Download %s retry started.", args[0]))
	return nil
}
