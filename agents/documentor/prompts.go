package documentor

import (
	"fmt"
	"strings"
)

func buildAnalysisPrompt(state State) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Analyze this repository and produce architecture notes.\n\n")
	fmt.Fprintf(&b, "Repository: %s\n", state.Repo.URL)
	fmt.Fprintf(&b, "Ref: %s\n\n", state.Repo.Ref)

	fmt.Fprintf(&b, "Manifest:\n")
	for _, f := range state.Manifest {
		fmt.Fprintf(&b, "- %s (lang=%s, pkg=%s, entry=%t)\n", f.Path, f.Language, f.PackageName, f.IsEntry)
	}

	fmt.Fprintf(&b, "\nSelected file contents:\n")
	for _, f := range state.Selected {
		fmt.Fprintf(&b, "\n===== FILE: %s =====\n%s\n", f.Path, f.Content)
	}

	fmt.Fprintf(&b, `
Write architecture notes covering:
- purpose of the repo/subsystem
- main packages/modules
- entry points
- key types/functions/interfaces
- config flow
- error handling patterns
- extension points
- unclear areas / TODOs

Be precise. Do not invent behavior.
`)
	return b.String()
}

func buildWriterPrompt(state State) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Write high-quality maintainer documentation as markdown.\n\n")
	fmt.Fprintf(&b, "Repository: %s\n", state.Repo.URL)
	fmt.Fprintf(&b, "Ref: %s\n\n", state.Repo.Ref)

	fmt.Fprintf(&b, "Architecture notes:\n%s\n\n", state.AnalysisMD)

	fmt.Fprintf(&b, `
Output requirements:
- Title
- Executive summary
- Architecture overview
- Package/module breakdown
- Key types and functions
- Execution/control flow
- Configuration and dependencies
- Extension points
- Operational notes / caveats
- File index of the most important files

Include file paths and symbol names where possible.
Do not include unsupported claims.
`)
	return b.String()
}
