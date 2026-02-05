package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var langpackCmd = &cobra.Command{
	Use:     "langpack",
	Aliases: []string{"lp"},
	Short:   "Manage language packs",
}

var langpackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all language packs",
	RunE:  runLangpackList,
}

var langpackGetCmd = &cobra.Command{
	Use:   "get <locale>",
	Short: "Get a language pack by locale",
	Args:  cobra.ExactArgs(1),
	RunE:  runLangpackGet,
}

var langpackUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload a language pack archive (.tar.xz, .tar.gz)",
	Args:  cobra.ExactArgs(1),
	RunE:  runLangpackUpload,
}

var langpackDeleteCmd = &cobra.Command{
	Use:   "delete <locale>",
	Short: "Delete a language pack",
	Args:  cobra.ExactArgs(1),
	RunE:  runLangpackDelete,
}

func init() {
	langpackCmd.AddCommand(langpackListCmd)
	langpackCmd.AddCommand(langpackGetCmd)
	langpackCmd.AddCommand(langpackUploadCmd)
	langpackCmd.AddCommand(langpackDeleteCmd)
}

func runLangpackList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListLangPacks(ctx)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	if len(resp.LanguagePacks) == 0 {
		output.PrintMessage("No language packs found.")
		return nil
	}

	rows := make([][]string, len(resp.LanguagePacks))
	for i, lp := range resp.LanguagePacks {
		rows[i] = []string{lp.Locale, lp.Name, lp.Version, lp.Author}
	}
	output.PrintTable([]string{"LOCALE", "NAME", "VERSION", "AUTHOR"}, rows)
	return nil
}

func runLangpackGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetLangPack(ctx, args[0])
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
			{"Locale", resp.Locale},
			{"Name", resp.Name},
			{"Version", resp.Version},
			{"Author", resp.Author},
		},
	)
	return nil
}

func runLangpackUpload(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	filePath := args[0]

	if err := c.UploadLangPack(ctx, filePath); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Language pack uploaded", "file": filePath})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Language pack uploaded", "file": filePath})
	}

	output.PrintMessage(fmt.Sprintf("Language pack %q uploaded.", filepath.Base(filePath)))
	return nil
}

func runLangpackDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteLangPack(ctx, args[0]); err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(map[string]string{"message": "Language pack deleted", "locale": args[0]})
	case "yaml":
		return output.PrintYAML(map[string]string{"message": "Language pack deleted", "locale": args[0]})
	}

	output.PrintMessage(fmt.Sprintf("Language pack %q deleted.", args[0]))
	return nil
}
