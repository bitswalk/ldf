package cmd

import (
	"context"
	"fmt"

	"github.com/bitswalk/ldf/src/ldfctl/internal/output"
	"github.com/spf13/cobra"
)

// VersionResponse matches the server's /v1/version response
type VersionResponse struct {
	Version        string `json:"version"`
	ReleaseName    string `json:"release_name"`
	ReleaseVersion string `json:"release_version"`
	BuildDate      string `json:"build_date"`
	GitCommit      string `json:"git_commit"`
	GoVersion      string `json:"go_version"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Shows the ldfctl client version and optionally the server version.`,
	RunE:  runVersion,
}

func init() {
	versionCmd.Flags().Bool("server", false, "Also show server version")
}

func runVersion(cmd *cobra.Command, args []string) error {
	showServer, _ := cmd.Flags().GetBool("server")

	format := getOutputFormat()
	if format == "json" || format == "yaml" {
		result := map[string]interface{}{
			"client": VersionInfo.Map(),
		}
		if showServer {
			if err := initConfig(); err != nil {
				return err
			}
			serverInfo, err := fetchServerVersion()
			if err != nil {
				result["server_error"] = err.Error()
			} else {
				result["server"] = serverInfo
			}
		}
		switch format {
		case "json":
			return output.PrintJSON(result)
		case "yaml":
			return output.PrintYAML(result)
		}
	}

	fmt.Printf("Client: %s\n", VersionInfo.Full())

	if showServer {
		if err := initConfig(); err != nil {
			return err
		}
		serverInfo, err := fetchServerVersion()
		if err != nil {
			fmt.Printf("\nServer: error: %v\n", err)
		} else {
			fmt.Printf("\nServer: %s\n", serverInfo.Version)
			fmt.Printf("  Release:    %s\n", serverInfo.ReleaseName)
			fmt.Printf("  Version:    %s\n", serverInfo.ReleaseVersion)
			fmt.Printf("  Build Date: %s\n", serverInfo.BuildDate)
			fmt.Printf("  Git Commit: %s\n", serverInfo.GitCommit)
			fmt.Printf("  Go Version: %s\n", serverInfo.GoVersion)
		}
	}

	return nil
}

func fetchServerVersion() (*VersionResponse, error) {
	c := getClient()
	ctx := context.Background()

	var resp VersionResponse
	if err := c.Get(ctx, "/v1/version", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
