package tools

// Session state keys. TODO agent specific, might refactor...
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
