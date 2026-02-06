// Package core provides the core command and server functionality for ldfd.
package core

import (
	"fmt"
	"os"

	"github.com/bitswalk/ldf/src/common/cli"
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// VersionInfo holds version information - set at build time via ldflags
	VersionInfo = version.New()

	// Global logger instance
	log *logs.Logger

	// Configuration file path
	cfgFile string
)

// Linker variables - these are set via ldflags at build time
// They must be initialized as empty strings or literals for ldflags to work
var (
	Version        = "dev"
	ReleaseName    = "Phoenix"
	ReleaseVersion = "0.0.0"
	BuildDate      = "unknown"
	GitCommit      = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ldfd",
	Short: "LDF API Server",
	Long: `ldfd is the core API server for the LDF platform.

It exposes REST APIs on port 8443 compatible with OpenAPI 3.2 standard.
The API is versioned and discoverable through the root endpoint.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

// Execute runs the root command
func Execute() {
	// Populate VersionInfo from linker variables
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
	// Configuration file flag
	cli.RegisterConfigFlag(rootCmd, &cfgFile, "/etc/ldfd/ldfd.yaml")

	// Server flags
	rootCmd.Flags().IntP("port", "p", 8443, "Port to listen on")
	rootCmd.Flags().StringP("bind", "b", "0.0.0.0", "Address to bind to")

	// Logging flags (using common helper)
	cli.RegisterLogFlags(rootCmd)

	// Database flags
	rootCmd.Flags().String("db-path", "~/.ldfd/ldfd.db", "Path to persist database on shutdown")

	// Storage flags
	rootCmd.Flags().String("storage-type", "local", "Storage backend type: 'local' or 's3'")
	rootCmd.Flags().String("storage-path", "~/.ldfd/artifacts", "Local storage path (for local backend)")

	// Build flags
	rootCmd.Flags().String("build-workspace", "~/.ldfd/cache/builds", "Base directory for build workspaces")
	rootCmd.Flags().Int("build-workers", 1, "Number of concurrent build workers")

	// TLS flags
	rootCmd.Flags().Bool("tls-enabled", false, "Enable native HTTPS/TLS support")
	rootCmd.Flags().String("tls-cert", "", "Path to TLS certificate file (PEM)")
	rootCmd.Flags().String("tls-key", "", "Path to TLS private key file (PEM)")

	// S3 Storage flags
	rootCmd.Flags().String("s3-endpoint", "", "S3-compatible storage endpoint URL")
	rootCmd.Flags().String("s3-region", "us-east-1", "S3 region")
	rootCmd.Flags().String("s3-bucket", "ldf-distributions", "S3 bucket for distribution artifacts")
	rootCmd.Flags().String("s3-access-key", "", "S3 access key ID")
	rootCmd.Flags().String("s3-secret-key", "", "S3 secret access key")
	rootCmd.Flags().Bool("s3-path-style", true, "Use path-style addressing for S3")

	// Bind flags to viper
	_ = viper.BindPFlag("server.port", rootCmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("server.bind", rootCmd.Flags().Lookup("bind"))
	_ = viper.BindPFlag("database.path", rootCmd.Flags().Lookup("db-path"))
	_ = viper.BindPFlag("storage.type", rootCmd.Flags().Lookup("storage-type"))
	_ = viper.BindPFlag("storage.local.path", rootCmd.Flags().Lookup("storage-path"))
	_ = viper.BindPFlag("storage.s3.endpoint", rootCmd.Flags().Lookup("s3-endpoint"))
	_ = viper.BindPFlag("storage.s3.region", rootCmd.Flags().Lookup("s3-region"))
	_ = viper.BindPFlag("storage.s3.bucket", rootCmd.Flags().Lookup("s3-bucket"))
	_ = viper.BindPFlag("storage.s3.access_key", rootCmd.Flags().Lookup("s3-access-key"))
	_ = viper.BindPFlag("storage.s3.secret_key", rootCmd.Flags().Lookup("s3-secret-key"))
	_ = viper.BindPFlag("storage.s3.path_style", rootCmd.Flags().Lookup("s3-path-style"))
	_ = viper.BindPFlag("build.workspace", rootCmd.Flags().Lookup("build-workspace"))
	_ = viper.BindPFlag("build.workers", rootCmd.Flags().Lookup("build-workers"))
	_ = viper.BindPFlag("server.tls.enabled", rootCmd.Flags().Lookup("tls-enabled"))
	_ = viper.BindPFlag("server.tls.cert_path", rootCmd.Flags().Lookup("tls-cert"))
	_ = viper.BindPFlag("server.tls.key_path", rootCmd.Flags().Lookup("tls-key"))

	// Set defaults
	viper.SetDefault("server.port", 8443)
	viper.SetDefault("server.bind", "0.0.0.0")
	viper.SetDefault("database.path", "~/.ldfd/ldfd.db")
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.local.path", "~/.ldfd/artifacts")
	viper.SetDefault("storage.s3.region", "us-east-1")
	viper.SetDefault("storage.s3.bucket", "ldf-distributions")
	viper.SetDefault("storage.s3.path_style", true)
	viper.SetDefault("build.workspace", "~/.ldfd/cache/builds")
	viper.SetDefault("build.workers", 1)
	viper.SetDefault("sync.cache_duration", 60) // 60 minutes default

	// Security defaults
	viper.SetDefault("security.rate_limit.enabled", true)
	viper.SetDefault("security.rate_limit.auth_per_min", 10)
	viper.SetDefault("security.rate_limit.api_per_min", 120)
	viper.SetDefault("security.rate_limit.trust_proxy", false)
	viper.SetDefault("security.max_refresh_tokens_per_user", 5)
	viper.SetDefault("security.master_key_path", "~/.ldfd/master.key")

	// TLS defaults
	viper.SetDefault("server.tls.enabled", false)
	viper.SetDefault("server.tls.cert_path", "")
	viper.SetDefault("server.tls.key_path", "")
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	// Use common config initialization with ldfd-specific search paths
	opts := cli.ConfigOptions{
		ConfigName: "ldfd",
		ConfigType: "yaml",
		EnvPrefix:  "LDFD",
		SearchPaths: []string{
			"/etc/ldfd",
			"/opt/ldfd",
			"~/.ldfd",
		},
	}
	opts.ConfigFile = cfgFile

	if err := cli.InitConfig(opts); err != nil {
		return err
	}

	// Initialize logger using common helper
	log = cli.InitLogger("ldfd")

	return nil
}
