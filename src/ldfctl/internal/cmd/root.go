package cmd

import (
	"fmt"
	"os"

	"github.com/bitswalk/ldf/src/common/cli"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/bitswalk/ldf/src/ldfctl/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// VersionInfo holds version information - set at build time via ldflags
	VersionInfo = version.New()

	// Configuration file path
	cfgFile string

	// Output format (json or table)
	outputFormat string

	// API client instance
	apiClient *client.Client
)

// Linker variables - set via ldflags at build time
var (
	Version        = "dev"
	ReleaseName    = "Phoenix"
	ReleaseVersion = "0.0.0"
	BuildDate      = "unknown"
	GitCommit      = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "ldfctl",
	Short: "LDF CLI Client",
	Long: `ldfctl is the command-line client for the LDF platform.

It communicates with the ldfd API server to manage distributions,
components, sources, downloads, artifacts, and settings.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config init for version command without --server flag
		if cmd.Name() == "version" && !cmd.Flags().Changed("server") {
			return nil
		}
		return initConfig()
	},
}

// Execute runs the root command
func Execute() {
	VersionInfo.Version = Version
	VersionInfo.ReleaseName = ReleaseName
	VersionInfo.ReleaseVersion = ReleaseVersion
	VersionInfo.BuildDate = BuildDate
	VersionInfo.GitCommit = GitCommit

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cli.RegisterConfigFlag(rootCmd, &cfgFile, "~/.ldfctl/ldfctl.yaml")

	rootCmd.PersistentFlags().StringP("server", "s", "", "LDF server URL (default: http://localhost:8443)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json")

	cli.RegisterLogFlags(rootCmd)

	viper.BindPFlag("server.url", rootCmd.PersistentFlags().Lookup("server"))

	viper.SetDefault("server.url", "http://localhost:8443")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(distributionCmd)
	rootCmd.AddCommand(componentCmd)
	rootCmd.AddCommand(sourceCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(artifactCmd)
	rootCmd.AddCommand(settingCmd)
	rootCmd.AddCommand(roleCmd)
	rootCmd.AddCommand(forgeCmd)
	rootCmd.AddCommand(brandingCmd)
	rootCmd.AddCommand(langpackCmd)
}

func initConfig() error {
	opts := cli.ConfigOptions{
		ConfigName: "ldfctl",
		ConfigType: "yaml",
		EnvPrefix:  "LDFCTL",
		SearchPaths: []string{
			"/etc/ldfctl",
			"~/.ldfctl",
		},
	}
	opts.ConfigFile = cfgFile

	if err := cli.InitConfig(opts); err != nil {
		return err
	}

	return nil
}

// getClient returns the API client, creating it if needed.
// It loads the stored token for authentication.
func getClient() *client.Client {
	if apiClient == nil {
		serverURL := viper.GetString("server.url")
		apiClient = client.New(serverURL)

		// Load stored token
		tokenData, err := config.LoadToken()
		if err == nil && tokenData.AccessToken != "" {
			apiClient.Token = tokenData.AccessToken
			apiClient.RefreshToken = tokenData.RefreshToken
		}
	}
	return apiClient
}

// getOutputFormat returns the current output format
func getOutputFormat() string {
	return outputFormat
}
