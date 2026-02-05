package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var artifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage artifacts",
}

var artifactListCmd = &cobra.Command{
	Use:   "list <distribution-id>",
	Short: "List artifacts for a distribution",
	Args:  cobra.ExactArgs(1),
	RunE:  runArtifactList,
}

var artifactUploadCmd = &cobra.Command{
	Use:   "upload <distribution-id> <file>",
	Short: "Upload an artifact",
	Args:  cobra.ExactArgs(2),
	RunE:  runArtifactUpload,
}

var artifactDownloadCmd = &cobra.Command{
	Use:   "download <distribution-id> <path> [dest]",
	Short: "Download an artifact",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runArtifactDownload,
}

var artifactDeleteCmd = &cobra.Command{
	Use:   "delete <distribution-id> <path>",
	Short: "Delete an artifact",
	Args:  cobra.ExactArgs(2),
	RunE:  runArtifactDelete,
}

var artifactURLCmd = &cobra.Command{
	Use:   "url <distribution-id> <path>",
	Short: "Get a direct download URL for an artifact",
	Args:  cobra.ExactArgs(2),
	RunE:  runArtifactURL,
}

var artifactStorageStatusCmd = &cobra.Command{
	Use:   "storage-status",
	Short: "Show storage backend status",
	RunE:  runArtifactStorageStatus,
}

var artifactListAllCmd = &cobra.Command{
	Use:   "list-all",
	Short: "List all artifacts across all distributions",
	RunE:  runArtifactListAll,
}

func init() {
	artifactCmd.AddCommand(artifactListCmd)
	artifactCmd.AddCommand(artifactUploadCmd)
	artifactCmd.AddCommand(artifactDownloadCmd)
	artifactCmd.AddCommand(artifactDeleteCmd)
	artifactCmd.AddCommand(artifactURLCmd)
	artifactCmd.AddCommand(artifactStorageStatusCmd)
	artifactCmd.AddCommand(artifactListAllCmd)
}

func runArtifactList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListArtifacts(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	if resp.Count == 0 {
		output.PrintMessage("No artifacts found.")
		return nil
	}

	rows := make([][]string, len(resp.Artifacts))
	for i, a := range resp.Artifacts {
		rows[i] = []string{a.Path, fmt.Sprintf("%d", a.Size)}
	}
	output.PrintTable([]string{"PATH", "SIZE"}, rows)
	return nil
}

func runArtifactUpload(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	distID := args[0]
	filePath := args[1]

	if err := c.UploadArtifact(ctx, distID, filePath); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Artifact uploaded", "distribution_id": distID, "file": filePath})
	}

	output.PrintMessage(fmt.Sprintf("Artifact %q uploaded to distribution %s.", filepath.Base(filePath), distID))
	return nil
}

func runArtifactDownload(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	distID := args[0]
	artifactPath := args[1]
	destPath := filepath.Base(artifactPath)
	if len(args) > 2 {
		destPath = args[2]
	}

	if err := c.DownloadArtifact(ctx, distID, artifactPath, destPath); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Artifact downloaded", "path": destPath})
	}

	output.PrintMessage(fmt.Sprintf("Artifact downloaded to %s.", destPath))
	return nil
}

func runArtifactDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	distID := args[0]
	artifactPath := args[1]

	if err := c.DeleteArtifact(ctx, distID, artifactPath); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Artifact deleted", "path": artifactPath})
	}

	output.PrintMessage(fmt.Sprintf("Artifact %q deleted.", artifactPath))
	return nil
}

func runArtifactURL(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetArtifactURL(ctx, args[0], args[1])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("%v", resp))
	return nil
}

func runArtifactStorageStatus(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetStorageStatus(ctx)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	available := "no"
	if resp.Available {
		available = "yes"
	}

	rows := [][]string{
		{"Type", resp.Type},
		{"Available", available},
	}
	if resp.Path != "" {
		rows = append(rows, []string{"Path", resp.Path})
	}
	if resp.Bucket != "" {
		rows = append(rows, []string{"Bucket", resp.Bucket})
	}
	if resp.Endpoint != "" {
		rows = append(rows, []string{"Endpoint", resp.Endpoint})
	}
	output.PrintTable([]string{"FIELD", "VALUE"}, rows)
	return nil
}

func runArtifactListAll(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListAllArtifacts(ctx)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	if resp.Count == 0 {
		output.PrintMessage("No artifacts found.")
		return nil
	}

	rows := make([][]string, len(resp.Artifacts))
	for i, a := range resp.Artifacts {
		rows[i] = []string{a.Path, fmt.Sprintf("%d", a.Size)}
	}
	output.PrintTable([]string{"PATH", "SIZE"}, rows)
	return nil
}
