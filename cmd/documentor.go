package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ATMackay/agent/agents/documentor"
	"github.com/ATMackay/agent/model"
	"github.com/ATMackay/agent/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	agentpkg "google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const userCLI = "cli-user"

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
				"creating documentor agent",
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

			sessService := session.InMemoryService()
			r, err := runner.New(runner.Config{
				AppName:        "documentor",
				Agent:          docAgent,
				SessionService: sessService,
			})
			if err != nil {
				return fmt.Errorf("create runner: %w", err)
			}

			initState := map[string]any{
				tools.StateRepoURL:    repoURL,
				tools.StateRepoRef:    ref,
				tools.StateOutputPath: output,
				tools.StateMaxFiles:   maxFiles,
			}
			if pathPrefix != "" {
				initState[tools.StateSubPath] = pathPrefix
			}

			resp, err := sessService.Create(ctx, &session.CreateRequest{
				AppName: "documentor",
				UserID:  userCLI,
				State:   initState,
			})
			if err != nil {
				return fmt.Errorf("create session: %w", err)
			}

			userMsg := &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					{
						Text: "Generate detailed code documentation for the configured repository. " +
							"Use fetch_repo_tree first, then read relevant files, then write the markdown output file.",
					},
				},
			}

			slog.Info(
				"running documentor agent",
				"session_id", resp.Session.ID(),
			)

			start := time.Now()
			for event, err := range r.Run(ctx, userCLI, resp.Session.ID(), userMsg, agentpkg.RunConfig{}) {
				if err != nil {
					return fmt.Errorf("agent error: %w", err)
				}
				// handle event (log)
				if event.UsageMetadata == nil {
					continue
				}
				slog.Info("tokens_used",
					"event_id", event.ID,
					"author", event.Author,
					"total_tokens", event.UsageMetadata.TotalTokenCount,
					"prompt_tokens", event.UsageMetadata.PromptTokenCount,
					"tool_use_token_count", event.UsageMetadata.ToolUsePromptTokenCount,
					"thought_token_count", event.UsageMetadata.ThoughtsTokenCount,
				)
				if event.Content == nil {
					continue
				}
				for _, p := range event.Content.Parts {
					slog.Debug("response_content",
						"event_id", event.ID,
						"role", event.Content.Role,
						"text", p.Text,
						"function_call", p.FunctionCall,
						"function_response", p.FunctionResponse,
					)
				}
			}
			slog.Info("Agent execution complete", "time_taken", time.Since(start))

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
