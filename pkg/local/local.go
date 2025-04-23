package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LocalRepo struct {
	Path           string
	Name           string
	IsGitRepo      bool
	IsInMonorepo   bool
	DefaultBranch  string
	LastCommitHash string
}

func SearchRepos(baseDir string, monorepoPath string) ([]LocalRepo, error) {
	var repos []LocalRepo
	err := filepath.Walk(baseDir, createWalkFunction(monorepoPath, &repos))
	return repos, err
}

func createWalkFunction(monorepoPath string, repos *[]LocalRepo) filepath.WalkFunc {
	// Implementation moved from main.go
	return nil
}

func PrintRepos(repos []LocalRepo) {
	fmt.Println("\nFound repositories:")
	fmt.Println("==================")
	for _, repo := range repos {
		status := "Not a Git repo"
		if repo.IsGitRepo {
			if repo.IsInMonorepo {
				status = "In monorepo"
			} else {
				status = fmt.Sprintf("Git repo (branch: %s)", repo.DefaultBranch)
			}
		}
		fmt.Printf("%s\n  Path: %s\n  Status: %s\n\n", repo.Name, repo.Path, status)
	}
}

func SaveReposData(repos []LocalRepo, filename string) error {
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
} 