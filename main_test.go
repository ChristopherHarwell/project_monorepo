package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup test config file
	testConfig := `{
		"github_token": "test_github",
		"gitlab_token": "test_gitlab",
		"use_subtree": true,
		"auto_mode": true
	}`
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Override config file location
	oldConfig := configFile
	configFile = tmpFile.Name()
	defer func() { configFile = oldConfig }()

	cfg := loadConfig()
	if cfg.GitHubToken != "test_github" {
		t.Errorf("Expected GitHub token 'test_github', got '%s'", cfg.GitHubToken)
	}
	if !cfg.UseSubtree {
		t.Error("Expected UseSubtree to be true")
	}
}

func TestFetchGitHubRepos(t *testing.T) {
	// Setup mock GitHub server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{
			"name": "test-repo",
			"ssh_url": "git@github.com:test/test-repo.git",
			"default_branch": "main"
		}]`))
	}))
	defer ts.Close()

	client := createGitHubClient()
	headers := githubHeaders("test-token")
	repos := fetchGitHubRepoList(client, headers, ts.URL)

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, got %d", len(repos))
	}
	if repos[0].Name != "test-repo" {
		t.Errorf("Expected repo name 'test-repo', got '%s'", repos[0].Name)
	}
}

func TestFetchGitLabRepos(t *testing.T) {
	// Setup mock GitLab server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{
			"name": "test-repo",
			"http_url_to_repo": "https://gitlab.com/test/test-repo.git",
			"default_branch": "main"
		}]`))
	}))
	defer ts.Close()

	req := createGitLabRequest("test-token")
	parsedURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	req.URL = parsedURL
	resp, err := executeGitLabRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	repos := parseGitLabResponse(resp, "test-token")
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, got %d", len(repos))
	}
	if repos[0].SSHURL != "https://oauth2:test-token@gitlab.com/test/test-repo.git" {
		t.Errorf("Unexpected SSH URL: %s", repos[0].SSHURL)
	}
}

func TestSearchLocalRepos(t *testing.T) {
	// Create test directory structure
	tmpDir, err := os.MkdirTemp("", "test-repos")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock git repo
	gitRepo := filepath.Join(tmpDir, "git-repo")
	if err := os.Mkdir(gitRepo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", gitRepo, "init").Run(); err != nil {
		t.Fatal(err)
	}

	// Create a non-git directory
	nonGitRepo := filepath.Join(tmpDir, "non-git-repo")
	if err := os.Mkdir(nonGitRepo, 0755); err != nil {
		t.Fatal(err)
	}

	repos, err := searchLocalRepos(tmpDir, filepath.Join(tmpDir, "monorepo"))
	if err != nil {
		t.Fatal(err)
	}

	if len(repos) != 2 {
		t.Fatalf("Expected 2 directories, got %d", len(repos))
	}

	var gitRepoFound bool
	for _, repo := range repos {
		if repo.Name == "git-repo" && repo.IsGitRepo {
			gitRepoFound = true
		}
	}
	if !gitRepoFound {
		t.Error("Git repo not detected correctly")
	}
}

func TestRepoExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-repo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo := Repo{Name: "test-repo"}
	if repoExists(repo) {
		t.Error("Repo should not exist before creation")
	}

	// Create repo directory
	if err := os.Mkdir(filepath.Join(monorepoDir, "repos", repo.Name), 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Join(monorepoDir, "repos", repo.Name))

	if !repoExists(repo) {
		t.Error("Repo should exist after creation")
	}
}

func TestHandleGitLabError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	handleGitLabError(resp) // Should log error but not panic
}

func TestFetchAllRepos(t *testing.T) {
	// Setup mock servers for GitHub and GitLab
	githubTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"name": "github-repo"}]`))
	}))
	defer githubTS.Close()

	gitlabTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"name": "gitlab-repo"}]`))
	}))
	defer gitlabTS.Close()

	cfg := Config{
		GitHubToken: "test",
		GitLabToken: "test",
	}

	// Override API endpoints
	oldGitHubURL := githubAPIURL
	oldGitLabURL := gitlabAPIURL
	githubAPIURL = githubTS.URL
	gitlabAPIURL = gitlabTS.URL
	defer func() {
		githubAPIURL = oldGitHubURL
		gitlabAPIURL = oldGitLabURL
	}()

	repos := fetchAllRepos(context.Background(), cfg)
	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(repos))
	}
} 