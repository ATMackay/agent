package state

// Session state keys.
const (
	// Shared state keys.
	StateOutputPath = "output_path"

	// Documentor agent state keys.
	StateRepoURL   = "repo_url"
	StateRepoRef   = "repo_ref"
	StateSubPath   = "sub_path"
	StateMaxFiles  = "max_files"

	StateRepoManifest  = "temp_repo_manifest"
	StateRepoLocalPath = "temp_repo_local_path"
	StateLoadedFiles   = "temp_loaded_files"

	StateDocumentation = "documentation_markdown"

	// Analyzer agent state keys.
	StateWorkDir = "work_dir"
)
