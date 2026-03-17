package cmd

import (
	"fmt"

	"github.com/ATMackay/agent/constants"
	"github.com/spf13/cobra"
)

const EnvPrefix = "AGENT"

func NewAgentCLICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "agent [subcommand]",
		Short: fmt.Sprintf("agent command line interface.\n\nVERSION:\n  semver: %s\n  commit: %s\n  compilation date: %s",
			constants.Version, constants.GitCommit, constants.BuildDate),
		RunE: runHelp,
	}

	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(VersionCmd())
	return cmd
}

func runHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
