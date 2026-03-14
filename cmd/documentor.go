package cmd

import (
	"fmt"
	"log/slog"
	"os"

	agentpkg "google.golang.org/adk/agent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/ATMackay/agent/agents/documentor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDocumentorCmd() *cobra.Command {
	var repoURL string
	var ref string
	var pathPrefix string
	var output string
	var maxFiles int
	var modelName string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "documentor",
		Short: "Run the code documentation agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prefer explicit flag, then env vars via Viper.
			apiKey = viper.GetString("google-api-key")
			if apiKey == "" {
				return fmt.Errorf("google api key is required; set --google-api-key or export GOOGLE_API_KEY")
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
			cfg = cfg.SetDefaults()
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			ctx := cmd.Context()

			slog.Info(
				"creating documentor agent",
				"dir", workDir,
				"model", modelName,
				"output", output,
				"repoURL", repoURL,
			)

			// Start with Gemini models
			// TODO create abstraction and package to support arbitrary model types.
			mod, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
				APIKey: apiKey,
			})
			if err != nil {
				return fmt.Errorf("create model: %w", err)
			}

			docAgent, err := documentor.NewDocumentorAgent(ctx, cfg, mod)
			if err != nil {
				return fmt.Errorf("create agent: %w", err)
			}

			slog.Info(
				"crrated agent",
				"agent_name", docAgent.Agent().Name(),
				"agent_description", docAgent.Agent().Description(),
			)

			sessService := session.InMemoryService()
			r, err := runner.New(runner.Config{
				AppName:        "documentor",
				Agent:          docAgent.Agent(),
				SessionService: sessService,
			})
			if err != nil {
				return fmt.Errorf("create runner: %w", err)
			}

			initState := map[string]any{
				documentor.StateRepoURL:    repoURL,
				documentor.StateRepoRef:    ref,
				documentor.StateOutputPath: output,
				documentor.StateMaxFiles:   maxFiles,
			}
			if pathPrefix != "" {
				initState[documentor.StateSubPath] = pathPrefix
			}

			resp, err := sessService.Create(ctx, &session.CreateRequest{
				AppName: "documentor",
				UserID:  "cli-user",
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

			for event, err := range r.Run(ctx, "cli-user", resp.Session.ID(), userMsg, agentpkg.RunConfig{}) {
				if err != nil {
					return fmt.Errorf("agent error: %w", err)
				}
				// handle event (log)
				slog.Info("event", "response_content", event.Content, "branch", event.Branch)
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
	cmd.Flags().StringVar(&output, "output", "", "Output file path for the generated markdown")
	cmd.Flags().IntVar(&maxFiles, "max-files", 20, "Maximum number of files to read")
	cmd.Flags().StringVar(&modelName, "model", "gemini-2.5-pro", "Gemini model to use")

	// Bind flags to environment variables
	must(viper.BindPFlag("repo", cmd.Flags().Lookup("repo")))
	must(viper.BindPFlag("ref", cmd.Flags().Lookup("ref")))
	must(viper.BindPFlag("path", cmd.Flags().Lookup("path")))
	must(viper.BindPFlag("output", cmd.Flags().Lookup("output")))
	must(viper.BindPFlag("max-files", cmd.Flags().Lookup("max-files")))
	must(viper.BindPFlag("model", cmd.Flags().Lookup("model")))

	// GOOGLE_API_KEY is preferred, GEMINI_API_KEY is accepted as fallback.
	must(viper.BindEnv("google-api-key", "GOOGLE_API_KEY", "GEMINI_API_KEY"))

	return cmd
}
