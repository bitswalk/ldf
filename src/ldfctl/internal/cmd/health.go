package cmd

import (
	"context"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

// HealthResponse matches the server's /v1/health response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	Long:  `Checks the health status of the LDF server.`,
	RunE:  runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	c := getClient()
	ctx := context.Background()

	var resp HealthResponse
	if err := c.Get(ctx, "/v1/health", &resp); err != nil {
		return err
	}

	return output.PrintFormatted(getOutputFormat(), resp, func() error {

		output.PrintTable(
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"Status", resp.Status},
				{"Timestamp", resp.Timestamp},
			},
		)
		return nil
	})
}
