package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ATMackay/agent/agents/analyzer"
	"github.com/ATMackay/agent/model"
	"github.com/ATMackay/agent/state"
	"github.com/ATMackay/agent/workflow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/adk/session"
)

func NewAnalyzerCmd() *cobra.Command {
	var workDir string
	var task string
	var output string
	var modelName, modelProvider string

	cmd := &cobra.Command{
		Use:   "analyzer",
		Short: "Run the general-purpose analyzer agent",
		Long: `Run the analyzer agent to perform filesystem and command-line tasks.
The agent can read, write, and edit local files, execute shell commands,
and analyze documents (including PDFs, text, source code, and more).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey := viper.GetString("api-key")
			if apiKey == "" {
				return fmt.Errorf("google gemini or claude api key is required; set --api-key or export API_KEY")
			}
			if task == "" {
				return fmt.Errorf("--task is required")
			}

			// Default work directory to the current directory.
			if workDir == "" {
				var err error
				workDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
			}

			ctx := cmd.Context()

			slog.Info(
				"creating agent",
				"agent_name", analyzer.AgentName,
				"work_dir", workDir,
				"model", modelName,
				"provider", modelProvider,
				"output", output,
			)

			modelCfg := &model.Config{
				Provider: model.Provider(modelProvider),
				Model:    modelName,
			}
			mod, err := model.New(ctx, modelCfg.WithAPIKey(apiKey))
			if err != nil {
				return fmt.Errorf("create model: %w", err)
			}

			cfg := &analyzer.Config{WorkDir: workDir}
			ag, err := analyzer.NewAnalyzer(ctx, cfg, mod)
			if err != nil {
				return fmt.Errorf("create agent: %w", err)
			}

			slog.Info(
				"created agent",
				"agent_name", ag.Name(),
				"agent_description", ag.Description(),
			)

			initState := map[string]any{
				state.StateWorkDir:    workDir,
				state.StateOutputPath: output,
			}

			s, err := workflow.New(
				ctx,
				analyzer.AgentName,
				session.InMemoryService(),
				ag,
				initState,
			)
			if err != nil {
				return fmt.Errorf("create workflow: %w", err)
			}

			userMsg := analyzer.UserMessage(task)

			if err := s.Start(ctx, userCLI, userMsg); err != nil {
				return err
			}

			slog.Info("Analyzer complete", "output_file", output)
			return nil
		},
	}

	cmd.Flags().StringVar(&workDir, "work-dir", "", "Working directory for file operations (defaults to current directory)")
	cmd.Flags().StringVar(&task, "task", "", "Task description for the analyzer agent (required)")
	cmd.Flags().StringVar(&output, "output", "analysis.md", "Output file path for the agent's written result")
	cmd.Flags().StringVar(&modelName, "model", "claude-opus-4-1-20250805", "Language model to use")
	cmd.Flags().StringVar(&modelProvider, "provider", "claude", "LLM provider (claude or gemini)")

	must(viper.BindPFlag("work-dir", cmd.Flags().Lookup("work-dir")))
	must(viper.BindPFlag("task", cmd.Flags().Lookup("task")))
	must(viper.BindPFlag("output", cmd.Flags().Lookup("output")))
	must(viper.BindPFlag("model", cmd.Flags().Lookup("model")))
	must(viper.BindPFlag("provider", cmd.Flags().Lookup("provider")))

	must(viper.BindEnv("api-key", "API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY", "CLAUDE_API_KEY"))

	return cmd
}
