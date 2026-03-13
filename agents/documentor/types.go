package documentor

// Session state keys used by the documentor agent.
const (
	StateRepoURL    = "repo_url"
	StateRepoRef    = "repo_ref"
	StateSubPath    = "sub_path"
	StateOutputPath = "output_path"
	StateMaxFiles   = "max_files"

	StateRepoManifest  = "temp_repo_manifest"
	StateRepoLocalPath = "temp_repo_local_path"
	StateLoadedFiles   = "temp_loaded_files"

	StateDocumentation = "documentation_markdown"
)

type FileInfo struct {
	Path        string
	Size        int64
	Language    string
	PackageName string
	IsEntry     bool
	IsTest      bool
	IsGenerated bool
}

type SourceFile struct {
	Path    string
	Content string
}

type RepoMetadata struct {
	Owner string
	Name  string
	Ref   string
	URL   string
}

type State struct {
	Repo          RepoMetadata
	LocalPath     string
	OutputPath    string
	Manifest      []FileInfo
	Selected      []SourceFile
	AnalysisMD    string
	Documentation string
}
