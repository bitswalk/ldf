package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var settingCmd = &cobra.Command{
	Use:   "setting",
	Short: "Manage server settings",
}

var settingListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all settings",
	RunE:  runSettingList,
}

var settingGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a setting value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSettingGet,
}

var settingSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a setting value",
	Args:  cobra.ExactArgs(2),
	RunE:  runSettingSet,
}

var settingResetDBCmd = &cobra.Command{
	Use:   "reset-db",
	Short: "Reset the database (requires confirmation)",
	RunE:  runSettingResetDB,
}

func init() {
	settingCmd.AddCommand(settingListCmd)
	settingCmd.AddCommand(settingGetCmd)
	settingCmd.AddCommand(settingSetCmd)
	settingCmd.AddCommand(settingResetDBCmd)

	settingResetDBCmd.Flags().Bool("yes", false, "Skip confirmation prompt")
}

func runSettingList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListSettings(ctx)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	rows := [][]string{}
	for key, s := range resp.Settings {
		val := fmt.Sprintf("%v", s.Value)
		rows = append(rows, []string{key, val, s.Type})
	}
	output.PrintTable([]string{"KEY", "VALUE", "TYPE"}, rows)
	return nil
}

func runSettingGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetSetting(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	reboot := "no"
	if resp.Reboot {
		reboot = "yes"
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"Key", resp.Key},
			{"Value", fmt.Sprintf("%v", resp.Value)},
			{"Type", resp.Type},
			{"Default", fmt.Sprintf("%v", resp.Default)},
			{"Description", resp.Description},
			{"Requires Reboot", reboot},
		},
	)
	return nil
}

func runSettingSet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	key := args[0]
	rawValue := args[1]

	// Try to parse the value as JSON first (handles booleans, numbers, arrays)
	var value interface{}
	if err := json.Unmarshal([]byte(rawValue), &value); err != nil {
		// If it's not valid JSON, treat it as a string
		value = rawValue
	}

	resp, err := c.UpdateSetting(ctx, key, value)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("Setting %q updated to %v.", key, resp.Value))
	return nil
}

func runSettingResetDB(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Print("WARNING: This will reset the entire database. All data will be lost.\nType 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			output.PrintMessage("Database reset cancelled.")
			return nil
		}
	}

	c := getClient()
	ctx := context.Background()

	if err := c.ResetDatabase(ctx); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Database reset successfully"})
	}

	output.PrintMessage("Database reset successfully.")
	return nil
}
