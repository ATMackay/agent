package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ATMackay/agent/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: fmt.Sprintf("Start the %s", constants.ServiceName),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read configuration from Viper
			logLevel := viper.GetString("log-level")
			logFormat := viper.GetString("log-format")
			//
			// Execute the main application lifecycle
			//
			// Initialize logger
			if err := initLogging(logLevel, logFormat); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			if isBuildDirty() {
				// Warn if the build contains uncommitted changes
				slog.Warn("running a DIRTY build (uncommitted changes present) — do not run in production")
			}
			slog.Info(fmt.Sprintf("starting %s", constants.ServiceName),
				"compilation_date", constants.BuildDate,
				"commit", constants.GitCommit,
				"version", constants.Version,
			)

			return nil
		},
	}

	// Add subcommands
	cmd.AddCommand(NewDocumentorCmd())
	// TODO - more agent types

	// Bind flags and ENV vars
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error, fatal, panic)")
	cmd.Flags().String("log-format", "text", "Log format (text, json)")

	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	// Bind flags to environment variables
	must(viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level")))
	must(viper.BindPFlag("log-format", cmd.Flags().Lookup("log-format")))

	// Set environment variable prefix and read from environment
	viper.SetEnvPrefix(EnvPrefix) // Environment variables will be prefixed with CHECKOUT_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // Automatically read environment variables
	return cmd

}
