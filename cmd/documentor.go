package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ATMackay/agent/agents/documentor"
	"github.com/ATMackay/agent/model"
	"github.com/ATMackay/agent/state"
	"github.com/ATMackay/agent/workflow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/adk/session"
)

func NewDocumentorCmd() *cobra.Command {
	var repoURL string
	var ref string
	var pathPrefix string
	var output string
	var maxFiles int
	var modelName, modelProvider string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "documentor",
		Short: "Run the code documentation agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prefer explicit flag, then env vars via Viper.
			apiKey = viper.GetString("api-key")
			if apiKey == "" {
				return fmt.Errorf("google gemini or claude api key is required; set --api-key or export API_KEY")
			}
			if repoURL == "" {
				return fmt.Errorf("--repo is required")
			}
			if output == "" {
				return fmt.Errorf("--output is required")
			}

			workDir, err := os.MkdirTemp("", "agent-documentor-*")
			if err != nil {
				return fmt.Errorf("create work dir: %w", err)
			}
			defer func() {
				if err := os.RemoveAll(workDir); err != nil {
					slog.Error("error removing body", "err", err)
				}
			}()

			cfg := &documentor.Config{WorkDir: workDir}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			ctx := cmd.Context()

			slog.Info(
				"creating agent",
				"agent_name", documentor.AgentName,
				"dir", workDir,
				"model", modelName,
				"provider", modelProvider,
				"output", output,
				"repoURL", repoURL,
			)

			// Select model provider. Supported providers: 'claude' or gemini.
			modelCfg := &model.Config{
				Provider: model.Provider(modelProvider),
				Model:    modelName,
			}
			mod, err := model.New(ctx, modelCfg.WithAPIKey(apiKey))
			if err != nil {
				return fmt.Errorf("create model: %w", err)
			}

			docAgent, err := documentor.NewDocumentor(ctx, cfg, mod)
			if err != nil {
				return fmt.Errorf("create agent: %w", err)
			}

			slog.Info(
				"created agent",
				"agent_name", docAgent.Name(),
				"agent_description", docAgent.Description(),
			)

			initState := map[string]any{
				state.StateRepoURL:    repoURL,
				state.StateRepoRef:    ref,
				state.StateOutputPath: output,
				state.StateMaxFiles:   maxFiles,
			}
			if pathPrefix != "" {
				initState[state.StateSubPath] = pathPrefix
			}

			s, err := workflow.New(
				ctx,
				documentor.AgentName,
				session.InMemoryService(),
				docAgent,
				initState)
			if err != nil {
				return err
			}

			userMsg := documentor.UserMessage()

			if err := s.Start(ctx, userCLI, userMsg); err != nil {
				return err
			}

			if _, err := os.Stat(output); err != nil {
				return fmt.Errorf("agent finished but output file was not created: %w", err)
			}

			slog.Info("Documentation written to", "output_file", output)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "repo", "", "GitHub repository URL")
	cmd.Flags().StringVar(&ref, "ref", "", "Optional branch, tag, or commit")
	cmd.Flags().StringVar(&pathPrefix, "path", "", "Optional subdirectory to document")
	cmd.Flags().StringVar(&output, "output", "doc.agentcli.md", "Output file path for the generated markdown")
	cmd.Flags().IntVar(&maxFiles, "max-files", 50, "Maximum number of files to read")
	cmd.Flags().StringVar(&modelName, "model", "claude-opus-4-1-20250805", "Language model to use")
	cmd.Flags().StringVar(&modelProvider, "provider", "claude", "LLM provider to use (claude or gemini)")

	// Bind flags to environment variables
	must(viper.BindPFlag("repo", cmd.Flags().Lookup("repo")))
	must(viper.BindPFlag("ref", cmd.Flags().Lookup("ref")))
	must(viper.BindPFlag("path", cmd.Flags().Lookup("path")))
	must(viper.BindPFlag("output", cmd.Flags().Lookup("output")))
	must(viper.BindPFlag("max-files", cmd.Flags().Lookup("max-files")))
	must(viper.BindPFlag("model", cmd.Flags().Lookup("model")))
	must(viper.BindPFlag("provider", cmd.Flags().Lookup("provider")))

	// API_KEY is preferred, GOOGLE_API_KEY, GEMINI_API_KEY, CLAUDE_API_KEY are accepted as fallback.
	must(viper.BindEnv("api-key", "API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY", "CLAUDE_API_KEY"))

	return cmd
}
