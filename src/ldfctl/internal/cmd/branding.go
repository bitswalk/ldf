package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var brandingCmd = &cobra.Command{
	Use:   "branding",
	Short: "Manage branding assets",
}

var brandingGetCmd = &cobra.Command{
	Use:   "get <asset> [dest]",
	Short: "Download a branding asset (logo, favicon)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBrandingGet,
}

var brandingInfoCmd = &cobra.Command{
	Use:   "info <asset>",
	Short: "Get branding asset metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrandingInfo,
}

var brandingUploadCmd = &cobra.Command{
	Use:   "upload <asset> <file>",
	Short: "Upload a branding asset",
	Args:  cobra.ExactArgs(2),
	RunE:  runBrandingUpload,
}

var brandingDeleteCmd = &cobra.Command{
	Use:   "delete <asset>",
	Short: "Delete a branding asset",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrandingDelete,
}

func init() {
	brandingCmd.AddCommand(brandingGetCmd)
	brandingCmd.AddCommand(brandingInfoCmd)
	brandingCmd.AddCommand(brandingUploadCmd)
	brandingCmd.AddCommand(brandingDeleteCmd)
}

func runBrandingGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	asset := args[0]
	destPath := asset
	if len(args) > 1 {
		destPath = args[1]
	}

	if err := c.GetBrandingAsset(ctx, asset, destPath); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Asset downloaded", "asset": asset, "path": destPath})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Asset downloaded", "asset": asset, "path": destPath})
	}

	output.PrintMessage(fmt.Sprintf("Branding asset %q downloaded to %s.", asset, destPath))
	return nil
}

func runBrandingInfo(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetBrandingAssetInfo(ctx, args[0])
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"Asset", resp.Asset},
			{"Exists", strconv.FormatBool(resp.Exists)},
			{"URL", resp.URL},
			{"Content Type", resp.ContentType},
			{"Size", fmt.Sprintf("%d", resp.Size)},
		},
	)
	return nil
}

func runBrandingUpload(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	asset := args[0]
	filePath := args[1]

	if err := c.UploadBrandingAsset(ctx, asset, filePath); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Asset uploaded", "asset": asset, "file": filePath})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Asset uploaded", "asset": asset, "file": filePath})
	}

	output.PrintMessage(fmt.Sprintf("Branding asset %q uploaded from %s.", asset, filepath.Base(filePath)))
	return nil
}

func runBrandingDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteBrandingAsset(ctx, args[0]); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Asset deleted", "asset": args[0]})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Asset deleted", "asset": args[0]})
	}

	output.PrintMessage(fmt.Sprintf("Branding asset %q deleted.", args[0]))
	return nil
}
