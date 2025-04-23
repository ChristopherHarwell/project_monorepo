package types

// Repo represents a Git repository with its essential metadata.
// This type is used to standardize repository information across different
// Git providers (GitHub, GitLab) and local repositories.
type Repo struct {
	// Name is the repository name without the owner/organization prefix
	Name string

	// SSHURL is the Git SSH URL used for cloning the repository
	SSHURL string

	// DefaultBranch is the name of the repository's default branch (e.g., "main", "master")
	DefaultBranch string
} 