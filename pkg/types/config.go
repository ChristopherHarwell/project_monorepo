package types

// Config represents the application's configuration settings.
// These settings control how the monorepo tool operates and interacts with
// different Git providers and local repositories.
type Config struct {
	// GitHubToken is the personal access token used for GitHub API authentication
	GitHubToken string `json:"github_token"`

	// GitLabToken is the personal access token used for GitLab API authentication
	GitLabToken string `json:"gitlab_token"`

	// UseSubtree determines whether to use Git subtree for repository integration
	// instead of submodules
	UseSubtree bool `json:"use_subtree"`

	// AutoMode enables automatic operation without user interaction
	AutoMode bool `json:"auto_mode"`

	// UpdateMode enables automatic updates of integrated repositories
	UpdateMode bool `json:"update_mode"`

	// PushMode enables automatic pushing of changes to remote repositories
	PushMode bool `json:"push_mode"`

	// ScanLocal enables scanning of local repositories for integration
	ScanLocal bool `json:"scan_local"`

	// BaseDir is the root directory to scan for local repositories
	BaseDir string `json:"base_dir"`

	// MonorepoPath is the path to the monorepo where repositories will be integrated
	MonorepoPath string `json:"monorepo_path"`
} 