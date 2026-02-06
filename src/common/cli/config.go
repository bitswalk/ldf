// Package cli provides common CLI utilities for ldf applications using Cobra and Viper.
package cli

import (
	"fmt"
	"strings"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigOptions holds options for configuration initialization
type ConfigOptions struct {
	// ConfigFile is the path to the config file (if specified via flag)
	ConfigFile string

	// ConfigName is the name of the config file (without extension)
	ConfigName string

	// ConfigType is the type of config file (yaml, json, toml)
	ConfigType string

	// EnvPrefix is the prefix for environment variables (e.g., "LDFD" -> LDFD_SERVER_PORT)
	EnvPrefix string

	// SearchPaths are additional paths to search for the config file
	SearchPaths []string
}

// DefaultConfigOptions returns default configuration options
func DefaultConfigOptions(configName, envPrefix string) ConfigOptions {
	return ConfigOptions{
		ConfigName: configName,
		ConfigType: "yaml",
		EnvPrefix:  envPrefix,
		SearchPaths: []string{
			"/etc/ldf",
			"$HOME/.config/ldf",
			".",
		},
	}
}

// InitConfig initializes Viper configuration with standard LDF patterns.
// It searches for config files, binds environment variables, and sets up defaults.
func InitConfig(opts ConfigOptions) error {
	if opts.ConfigFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(paths.Expand(opts.ConfigFile))
	} else {
		// Search for config in standard locations
		viper.SetConfigName(opts.ConfigName)
		viper.SetConfigType(opts.ConfigType)

		for _, searchPath := range opts.SearchPaths {
			viper.AddConfigPath(paths.Expand(searchPath))
		}
	}

	// Read environment variables with prefix
	if opts.EnvPrefix != "" {
		viper.SetEnvPrefix(opts.EnvPrefix)
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv()
	}

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is not an error - we use defaults
	}

	return nil
}

// RegisterLogFlags registers common logging flags on a Cobra command
func RegisterLogFlags(cmd *cobra.Command) {
	cmd.Flags().String("log-output", "auto", "Log output destination (auto, stdout, journald)")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")

	_ = viper.BindPFlag("log.output", cmd.Flags().Lookup("log-output"))
	_ = viper.BindPFlag("log.level", cmd.Flags().Lookup("log-level"))

	viper.SetDefault("log.output", "auto")
	viper.SetDefault("log.level", "info")
}

// RegisterConfigFlag registers the --config flag on a Cobra command
func RegisterConfigFlag(cmd *cobra.Command, cfgFile *string, defaultPath string) {
	cmd.PersistentFlags().StringVar(cfgFile, "config", "", fmt.Sprintf("config file (default: %s)", defaultPath))
}

// InitLogger creates and returns a logger based on Viper configuration.
// Should be called after InitConfig.
func InitLogger(prefix string) *logs.Logger {
	logOutput := logs.LogOutput(viper.GetString("log.output"))
	return logs.New(logs.Config{
		Output: logOutput,
		Level:  viper.GetString("log.level"),
		Prefix: prefix,
	})
}

// BindFlag binds a Cobra flag to a Viper config key
func BindFlag(cmd *cobra.Command, flagName, viperKey string) error {
	return viper.BindPFlag(viperKey, cmd.Flags().Lookup(flagName))
}

// BindPersistentFlag binds a Cobra persistent flag to a Viper config key
func BindPersistentFlag(cmd *cobra.Command, flagName, viperKey string) error {
	return viper.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(flagName))
}

// GetExpandedString gets a string from Viper and expands path prefixes
func GetExpandedString(key string) string {
	return paths.Expand(viper.GetString(key))
}
