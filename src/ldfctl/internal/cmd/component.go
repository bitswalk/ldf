package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var componentCmd = &cobra.Command{
	Use:     "component",
	Aliases: []string{"comp"},
	Short:   "Manage components",
}

var componentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all components",
	RunE:  runComponentList,
}

var componentGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a component by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentGet,
}

var componentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new component",
	RunE:  runComponentCreate,
}

var componentUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a component",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentUpdate,
}

var componentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a component",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentDelete,
}

func init() {
	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentGetCmd)
	componentCmd.AddCommand(componentCreateCmd)
	componentCmd.AddCommand(componentUpdateCmd)
	componentCmd.AddCommand(componentDeleteCmd)

	componentCreateCmd.Flags().String("name", "", "Component name (required)")
	componentCreateCmd.Flags().String("category", "", "Component category")
	componentCreateCmd.Flags().String("description", "", "Component description")
	componentCreateCmd.Flags().String("source-url", "", "Source URL")
	componentCreateCmd.Flags().String("license", "", "License")
	componentCreateCmd.MarkFlagRequired("name")

	componentUpdateCmd.Flags().String("name", "", "Component name")
	componentUpdateCmd.Flags().String("category", "", "Component category")
	componentUpdateCmd.Flags().String("description", "", "Component description")
	componentUpdateCmd.Flags().String("source-url", "", "Source URL")
	componentUpdateCmd.Flags().String("license", "", "License")
}

func runComponentList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListComponents(ctx)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	if resp.Count == 0 {
		output.PrintMessage("No components found.")
		return nil
	}

	rows := make([][]string, len(resp.Components))
	for i, comp := range resp.Components {
		system := ""
		if comp.IsSystem {
			system = "system"
		}
		rows[i] = []string{comp.ID, comp.Name, comp.Category, system}
	}
	output.PrintTable([]string{"ID", "NAME", "CATEGORY", "TYPE"}, rows)
	return nil
}

func runComponentGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetComponent(ctx, args[0])
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	isSystem := "no"
	if resp.IsSystem {
		isSystem = "yes"
	}

	output.PrintTable(
		[]string{"FIELD", "VALUE"},
		[][]string{
			{"ID", resp.ID},
			{"Name", resp.Name},
			{"Category", resp.Category},
			{"Description", resp.Description},
			{"Source URL", resp.SourceURL},
			{"License", resp.License},
			{"System", isSystem},
			{"Created", resp.CreatedAt},
			{"Updated", resp.UpdatedAt},
		},
	)
	return nil
}

func runComponentCreate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	name, _ := cmd.Flags().GetString("name")
	category, _ := cmd.Flags().GetString("category")
	description, _ := cmd.Flags().GetString("description")
	sourceURL, _ := cmd.Flags().GetString("source-url")
	license, _ := cmd.Flags().GetString("license")

	req := &client.CreateComponentRequest{
		Name:        name,
		Category:    category,
		Description: description,
		SourceURL:   sourceURL,
		License:     license,
	}

	resp, err := c.CreateComponent(ctx, req)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("Component %q created (ID: %s)", resp.Name, resp.ID))
	return nil
}

func runComponentUpdate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	req := &client.UpdateComponentRequest{}
	if cmd.Flags().Changed("name") {
		v, _ := cmd.Flags().GetString("name")
		req.Name = v
	}
	if cmd.Flags().Changed("category") {
		v, _ := cmd.Flags().GetString("category")
		req.Category = v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		req.Description = v
	}
	if cmd.Flags().Changed("source-url") {
		v, _ := cmd.Flags().GetString("source-url")
		req.SourceURL = v
	}
	if cmd.Flags().Changed("license") {
		v, _ := cmd.Flags().GetString("license")
		req.License = v
	}

	resp, err := c.UpdateComponent(ctx, args[0], req)
	if err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(resp)
	}

	output.PrintMessage(fmt.Sprintf("Component %q updated.", resp.Name))
	return nil
}

func runComponentDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteComponent(ctx, args[0]); err != nil {
		return err
	}

	if getOutputFormat() == "json" {
		return output.PrintJSON(map[string]string{"message": "Component deleted", "id": args[0]})
	}

	output.PrintMessage(fmt.Sprintf("Component %s deleted.", args[0]))
	return nil
}
