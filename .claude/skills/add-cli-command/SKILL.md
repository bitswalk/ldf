---
name: add-cli-command
description: Scaffold a new CLI command for ldfctl with Cobra command definitions, flags, API client calls, and output formatting. Use when adding a new resource or subcommand to the CLI.
argument-hint: "[resource-name] [description]"

---

# Add CLI Command

Scaffold a complete CLI command group following the project's established Cobra patterns.

## Arguments

- `$ARGUMENTS[0]` -- Resource name (e.g., `board`, `profile`). This becomes the command name.
- `$ARGUMENTS[1]` -- Short description (e.g., "Manage board profiles")

## Steps

### 1. Create the command file

Create `src/ldfctl/internal/cmd/$0.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

var $0Cmd = &cobra.Command{
	Use:     "$0",
	Aliases: []string{"<short-alias>"},
	Short:   "$1",
}

var $0ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ${0}s",
	RunE:  run${0^}List,
}

var $0GetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a $0 by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  run${0^}Get,
}

var $0CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new $0",
	RunE:  run${0^}Create,
}

var $0UpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a $0",
	Args:  cobra.ExactArgs(1),
	RunE:  run${0^}Update,
}

var $0DeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a $0",
	Args:  cobra.ExactArgs(1),
	RunE:  run${0^}Delete,
}

func init() {
	$0Cmd.AddCommand($0ListCmd)
	$0Cmd.AddCommand($0GetCmd)
	$0Cmd.AddCommand($0CreateCmd)
	$0Cmd.AddCommand($0UpdateCmd)
	$0Cmd.AddCommand($0DeleteCmd)

	// List flags
	$0ListCmd.Flags().Int("limit", 0, "Maximum number of results")
	$0ListCmd.Flags().Int("offset", 0, "Number of results to skip")

	// Create flags
	$0CreateCmd.Flags().String("name", "", "Name (required)")
	$0CreateCmd.MarkFlagRequired("name")

	// Update flags
	$0UpdateCmd.Flags().String("name", "", "Name")
}
```

### 2. Implement RunE handlers

Each handler follows this pattern:

```go
func run${0^}List(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	opts := &client.ListOptions{}
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Offset, _ = cmd.Flags().GetInt("offset")

	resp, err := c.List${0^}s(ctx, opts)
	if err != nil {
		return err
	}

	switch getOutputFormat() {
	case "json":
		return output.PrintJSON(resp)
	case "yaml":
		return output.PrintYAML(resp)
	}

	if len(resp.Items) == 0 {
		output.PrintMessage("No ${0}s found.")
		return nil
	}

	rows := make([][]string, len(resp.Items))
	for i, item := range resp.Items {
		rows[i] = []string{item.ID, item.Name}
	}
	output.PrintTable([]string{"ID", "NAME"}, rows)
	return nil
}
```

### 3. Add client methods

Add the corresponding API client methods to `src/ldfctl/internal/client/`:

```go
func (c *Client) List${0^}s(ctx context.Context, opts *ListOptions) (*${0^}ListResponse, error) {
	var resp ${0^}ListResponse
	err := c.Get(ctx, "/v1/${0}s"+opts.QueryString(), &resp)
	return &resp, err
}
```

### 4. Register in root.go

Edit `src/ldfctl/internal/cmd/root.go`, add to the `init()` function:

```go
rootCmd.AddCommand($0Cmd)
```

And add completions in `registerCompletions()` if applicable:

```go
$0GetCmd.ValidArgsFunction = completion${0^}IDs
$0UpdateCmd.ValidArgsFunction = completion${0^}IDs
$0DeleteCmd.ValidArgsFunction = completion${0^}IDs
```

### 5. Verify

Run: `task build:cli` to confirm compilation.

## Conventions to follow

- Parent command has no `RunE` (grouping only)
- Subcommand vars: `$0ListCmd`, `$0GetCmd`, `$0CreateCmd`, etc.
- RunE functions: `run${0^}List`, `run${0^}Get`, etc. (camelCase resource + PascalCase action)
- Every handler must support `--output json|yaml|table` via `getOutputFormat()` switch
- Use `cobra.ExactArgs(1)` for commands taking a single ID argument
- Use `cmd.Flags().Changed("flag")` for update commands (only send changed fields)
- Use `getClient()` to get the singleton API client
- Use `context.Background()` for the request context
- Success messages via `output.PrintMessage(fmt.Sprintf(...))`
- Delete operations: confirm message then `output.PrintMessage`
