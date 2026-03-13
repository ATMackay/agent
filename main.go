package main

import (
	"log/slog"
	"os"

	"github.com/ATMackay/agent/cmd"
)

// TODO
// TODO
// This is a toy project.... Building AI agents with Google's ADK performing various tasks
//
//  Features
//
// Cobra cli framework
// Google ADK for AI agent development
// Agents include...
// Static Code analysis agent
// Documentation agent

// @title         Agent API
// @version       0.1.0
// @description   API for running code analysis agents
// @schemes       TODO
// @host          TODO

// @securityDefinitions.apikey  XAuthPassword
// @in                          header
// @name                        X-Auth-Password

func main() {
	command := cmd.NewAgentCmd()
	if err := command.Execute(); err != nil {
		slog.Error("main: execution failed", "error", err)
		os.Exit(1)
	}
}
