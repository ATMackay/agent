package tools

type EditFileArgs struct {
	Path      string `json:"path"`
	StartLine string `json:"start_line"`
	EndLine   int    `json:"end_line,omitempty"`
	Content   string `json:"content"` // multi-line content separated by '\n'
}

type EditFileResult struct {
	// TODO
}
