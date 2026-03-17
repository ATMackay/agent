package main

import (
	"log/slog"
	"os"

	"github.com/ATMackay/agent/cmd"
)

// @title         Agent CLI
// @version       0.1.0
// @description   CLI for AI code/document analysis agents
// @schemes       TODO
// @host          TODO
func main() {
	command := cmd.NewAgentCLICmd()
	if err := command.Execute(); err != nil {
		slog.Error("main: execution failed", "error", err)
		os.Exit(1)
	}
}
