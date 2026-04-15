package cmd

import (
	"fmt"

	"github.com/ATMackay/agent/constants"
	"github.com/spf13/cobra"
)

const EnvPrefix = "AGENT"

func NewAgentCLICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [subcommand]",
		Short: "CLI for running AI agents and workflows",
		Long: fmt.Sprintf(`Agent CLI

Run and manage AI agents such as code documentors, reviewers, and other workflows.

Version:
  semver: %s
  commit: %s
  build:  %s
`,
			constants.Version,
			constants.GitCommit,
			constants.BuildDate,
		),
		RunE: runHelp,
	}

	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(VersionCmd())
	return cmd
}

func runHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
