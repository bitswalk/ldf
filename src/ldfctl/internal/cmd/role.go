package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles",
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all roles",
	RunE:  runRoleList,
}

var roleGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a role by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleGet,
}

var roleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new role",
	RunE:  runRoleCreate,
}

var roleUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleUpdate,
}

var roleDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleDelete,
}

func init() {
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleGetCmd)
	roleCmd.AddCommand(roleCreateCmd)
	roleCmd.AddCommand(roleUpdateCmd)
	roleCmd.AddCommand(roleDeleteCmd)

	roleCreateCmd.Flags().String("name", "", "Role name (required)")
	roleCreateCmd.Flags().String("description", "", "Role description")
	roleCreateCmd.Flags().String("parent-role-id", "", "Parent role ID")
	roleCreateCmd.Flags().Bool("can-read", false, "Read permission")
	roleCreateCmd.Flags().Bool("can-write", false, "Write permission")
	roleCreateCmd.Flags().Bool("can-delete", false, "Delete permission")
	roleCreateCmd.Flags().Bool("can-admin", false, "Admin permission")
	_ = roleCreateCmd.MarkFlagRequired("name")

	roleUpdateCmd.Flags().String("name", "", "Role name")
	roleUpdateCmd.Flags().String("description", "", "Role description")
	roleUpdateCmd.Flags().Bool("can-read", false, "Read permission")
	roleUpdateCmd.Flags().Bool("can-write", false, "Write permission")
	roleUpdateCmd.Flags().Bool("can-delete", false, "Delete permission")
	roleUpdateCmd.Flags().Bool("can-admin", false, "Admin permission")
}

func runRoleList(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.ListRoles(ctx)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		if len(resp.Roles) == 0 {
			output.PrintMessage("No roles found.")
			return nil
		}

		rows := make([][]string, len(resp.Roles))
		for i, r := range resp.Roles {
			perms := ""
			if r.Permissions.CanRead {
				perms += "R"
			}
			if r.Permissions.CanWrite {
				perms += "W"
			}
			if r.Permissions.CanDelete {
				perms += "D"
			}
			if r.Permissions.CanAdmin {
				perms += "A"
			}
			system := ""
			if r.IsSystem {
				system = "system"
			}
			rows[i] = []string{r.ID, r.Name, perms, system}
		}
		output.PrintTable([]string{"ID", "NAME", "PERMISSIONS", "TYPE"}, rows)
		return nil
	})
}

func runRoleGet(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	resp, err := c.GetRole(ctx, args[0])
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		r := resp.Role
		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", r.ID},
				{"Name", r.Name},
				{"Description", r.Description},
				{"System", strconv.FormatBool(r.IsSystem)},
				{"Can Read", strconv.FormatBool(r.Permissions.CanRead)},
				{"Can Write", strconv.FormatBool(r.Permissions.CanWrite)},
				{"Can Delete", strconv.FormatBool(r.Permissions.CanDelete)},
				{"Can Admin", strconv.FormatBool(r.Permissions.CanAdmin)},
			},
		)
		return nil
	})
}

func runRoleCreate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	parentRoleID, _ := cmd.Flags().GetString("parent-role-id")
	canRead, _ := cmd.Flags().GetBool("can-read")
	canWrite, _ := cmd.Flags().GetBool("can-write")
	canDelete, _ := cmd.Flags().GetBool("can-delete")
	canAdmin, _ := cmd.Flags().GetBool("can-admin")

	req := &client.CreateRoleRequest{
		Name:         name,
		Description:  description,
		ParentRoleID: parentRoleID,
	}
	req.Permissions.CanRead = canRead
	req.Permissions.CanWrite = canWrite
	req.Permissions.CanDelete = canDelete
	req.Permissions.CanAdmin = canAdmin

	resp, err := c.CreateRole(ctx, req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Role %q created (ID: %s)", resp.Role.Name, resp.Role.ID))
		return nil
	})
}

func runRoleUpdate(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	req := &client.UpdateRoleRequest{}
	if cmd.Flags().Changed("name") {
		v, _ := cmd.Flags().GetString("name")
		req.Name = v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		req.Description = v
	}

	if cmd.Flags().Changed("can-read") || cmd.Flags().Changed("can-write") ||
		cmd.Flags().Changed("can-delete") || cmd.Flags().Changed("can-admin") {
		canRead, _ := cmd.Flags().GetBool("can-read")
		canWrite, _ := cmd.Flags().GetBool("can-write")
		canDelete, _ := cmd.Flags().GetBool("can-delete")
		canAdmin, _ := cmd.Flags().GetBool("can-admin")
		req.Permissions = &struct {
			CanRead   bool `json:"can_read"`
			CanWrite  bool `json:"can_write"`
			CanDelete bool `json:"can_delete"`
			CanAdmin  bool `json:"can_admin"`
		}{
			CanRead:   canRead,
			CanWrite:  canWrite,
			CanDelete: canDelete,
			CanAdmin:  canAdmin,
		}
	}

	resp, err := c.UpdateRole(ctx, args[0], req)
	if err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintMessage(fmt.Sprintf("Role %q updated.", resp.Role.Name))
		return nil
	})
}

func runRoleDelete(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	if err := c.DeleteRole(ctx, args[0]); err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), map[string]string{"message": "Role deleted", "id": args[0]}, func() error {

		output.PrintMessage(fmt.Sprintf("Role %s deleted.", args[0]))
		return nil
	})
}
