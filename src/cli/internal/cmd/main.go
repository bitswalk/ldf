package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	logger  *log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "ldfctl",
	Short: "Linux Distribution Factory - Build custom Linux distributions",
	Long: `Linux Distribution Factory (LDF) is a modern tool for creating custom Linux distributions.
It provides an easy-to-use interface for configuring and building Linux distributions
with your choice of kernel, init system, filesystem, and other components.`,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/ldfctl/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")

	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Initialize logger
	logger = log.New(os.Stderr)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home + "/.config/ldfctl")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("LDF")

	if err := viper.ReadInConfig(); err == nil {
		logger.Info("Using config file", "file", viper.ConfigFileUsed())
	}

	// Set log level
	switch viper.GetString("log.level") {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.InfoLevel)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
