package cmd

import (
	"fmt"
	"os"

	agentpkg "google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/ATMackay/agent/agents/documentor"
	"github.com/spf13/cobra"
)

func NewDocumentorCmd() *cobra.Command {
	var repoURL string
	var ref string
	var pathPrefix string
	var output string
	var maxFiles int
	var model string

	cmd := &cobra.Command{
		Use:   "documentor",
		Short: "Run the code documentation agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey := os.Getenv("GOOGLE_API_KEY")
			if apiKey == "" {
				return fmt.Errorf("GOOGLE_API_KEY is required")
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
			defer os.RemoveAll(workDir)

			ctx := cmd.Context()

			doc, err := documentor.NewDocumentorAgent(ctx, documentor.Config{
				ModelName: model,
				APIKey:    apiKey,
				WorkDir:   workDir,
			})
			if err != nil {
				return fmt.Errorf("create agent: %w", err)
			}

			sessService := session.InMemoryService()
			r, err := runner.New(runner.Config{
				AppName:        "documentor",
				Agent:          doc,
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
				_ = event
			}

			if _, err := os.Stat(output); err != nil {
				return fmt.Errorf("agent finished but output file was not created: %w", err)
			}

			fmt.Printf("Documentation written to %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "repo", "", "GitHub repository URL")
	cmd.Flags().StringVar(&ref, "ref", "", "Optional branch, tag, or commit")
	cmd.Flags().StringVar(&pathPrefix, "path", "", "Optional subdirectory to document")
	cmd.Flags().StringVar(&output, "output", "", "Output file path for the generated markdown")
	cmd.Flags().IntVar(&maxFiles, "max-files", 20, "Maximum number of files to read")
	cmd.Flags().StringVar(&model, "model", "gemini-2.5-pro", "Gemini model to use")

	return cmd
}
