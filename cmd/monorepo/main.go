// Package main provides the entry point for the monorepo management tool.
// This tool helps manage multiple Git repositories by integrating them into a single monorepo,
// supporting both GitHub and GitLab repositories, as well as local repositories.
package main

import (
	"context"
	"fmt"
	"os"

	"christopherharwell/project_monorepo/pkg/config"
	"christopherharwell/project_monorepo/pkg/git"
	"christopherharwell/project_monorepo/pkg/github"
	"christopherharwell/project_monorepo/pkg/gitlab"
	"christopherharwell/project_monorepo/pkg/local"
	"christopherharwell/project_monorepo/pkg/types"
)

const (
	// configFile is the default path to the configuration file
	configFile = "config.json"
)

// main is the entry point of the application.
// It loads the configuration, handles local repositories if configured,
// fetches remote repositories, and processes them according to the configuration.
func main() {
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if cfg.ScanLocal {
		handleLocalRepos(cfg)
	}

	repos := getRepositories(ctx, cfg)
	selected := selectRepositories(repos, cfg.AutoMode)
	
	if err := git.InitMonorepo(); err != nil {
		fmt.Printf("Error initializing monorepo: %v\n", err)
		os.Exit(1)
	}

	processRepositories(selected, cfg)
}

// handleLocalRepos scans and processes local repositories based on the configuration.
// It prints information about found repositories and saves the data to a file.
//
// Parameters:
//   - cfg: The application configuration
func handleLocalRepos(cfg types.Config) {
	fmt.Println("Scanning local repositories...")
	if cfg.BaseDir == "" || cfg.MonorepoPath == "" {
		fmt.Println("Error: 'base_dir' and 'monorepo_path' must be set in config.json when scan_local is true")
		os.Exit(1)
	}

	localRepos, err := local.SearchRepos(cfg.BaseDir, cfg.MonorepoPath)
	if err != nil {
		fmt.Printf("Error scanning local repositories: %v\n", err)
		os.Exit(1)
	}

	local.PrintRepos(localRepos)
	if err := local.SaveReposData(localRepos, "local_repos.json"); err != nil {
		fmt.Printf("Error saving local repository data: %v\n", err)
	}

	if !cfg.AutoMode {
		promptContinue()
	}
}

// getRepositories fetches repositories from both GitHub and GitLab based on the configuration.
//
// Parameters:
//   - ctx: Context for the operation
//   - cfg: The application configuration
//
// Returns:
//   - []types.Repo: A slice of repositories from both sources
func getRepositories(ctx context.Context, cfg types.Config) []types.Repo {
	var allRepos []types.Repo
	
	// Fetch from GitHub
	githubRepos := github.FetchRepos(ctx, cfg.GitHubToken)
	allRepos = append(allRepos, githubRepos...)
	
	// Fetch from GitLab
	gitlabRepos := gitlab.FetchRepos(ctx, cfg.GitLabToken)
	allRepos = append(allRepos, gitlabRepos...)
	
	return allRepos
}

// selectRepositories filters repositories based on the auto mode setting.
// In auto mode, all repositories are selected. Otherwise, it prompts for user selection.
//
// Parameters:
//   - repos: The list of repositories to select from
//   - autoMode: Whether to automatically select all repositories
//
// Returns:
//   - []types.Repo: The selected repositories
func selectRepositories(repos []types.Repo, autoMode bool) []types.Repo {
	if autoMode {
		return repos
	}
	
	// TODO: Implement interactive selection
	return repos
}

// processRepositories handles the integration of selected repositories into the monorepo.
//
// Parameters:
//   - selected: The repositories to process
//   - cfg: The application configuration
func processRepositories(selected []types.Repo, cfg types.Config) {
	// TODO: Implement repository processing
}

// promptContinue waits for user input before proceeding with remote repository scanning.
func promptContinue() {
	fmt.Print("Press Enter to continue with remote repository scanning or Ctrl+C to exit...")
	fmt.Scanln()
} 