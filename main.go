package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Repo struct {
	Name          string
	SSHURL        string
	DefaultBranch string
}

type LocalRepo struct {
	Path           string
	Name           string
	IsGitRepo      bool
	IsInMonorepo   bool
	DefaultBranch  string
	LastCommitHash string
}

type Config struct {
	GitHubToken  string `json:"github_token"`
	GitLabToken  string `json:"gitlab_token"`
	UseSubtree   bool   `json:"use_subtree"`
	AutoMode     bool   `json:"auto_mode"`
	UpdateMode   bool   `json:"update_mode"`
	PushMode     bool   `json:"push_mode"`
	ScanLocal    bool   `json:"scan_local"`
	BaseDir      string `json:"base_dir"`
	MonorepoPath string `json:"monorepo_path"`
}

var (
	cacheFile   = "repo_cache.json"
	monorepoDir = "monorepo"
	useSubtree  = false

	autoMode   = false
	updateMode = false
	pushMode   = false

	configFile = "config.json"

	githubAPIURL = "https://api.github.com"
	gitlabAPIURL = "https://gitlab.com/api/v4"
)

func main() {
	cfg := loadConfig()
	setupConfig(cfg)
	ctx := context.Background()

	if cfg.ScanLocal {
		handleLocalRepos(cfg)
	}

	repos := getRepositories(ctx, cfg)
	selected := selectRepositories(repos, cfg.AutoMode)
	initMonorepo()
	processRepositories(selected, cfg)
}

func setupConfig(cfg Config) {
	useSubtree = cfg.UseSubtree
	autoMode = cfg.AutoMode
	updateMode = cfg.UpdateMode
	pushMode = cfg.PushMode
}

func ThrowLocalScanErr(err error) {
	if err != nil {
		fmt.Printf("Error scanning local repositories: %v\n", err)
		os.Exit(1)
	}
}
func handleLocalRepos(cfg Config) {
	fmt.Println("Scanning local repositories...")
	if cfg.BaseDir == "" || cfg.MonorepoPath == "" {
		fmt.Println("Error: 'base_dir' and 'monorepo_path' must be set in config.json when scan_local is true")
		os.Exit(1)
	}

	localRepos, err := searchLocalRepos(cfg.BaseDir, cfg.MonorepoPath)
	ThrowLocalScanErr(err)

	printLocalRepos(localRepos)
	saveLocalReposData(localRepos)

	if !autoMode {
		promptContinue()
	}
}

func printLocalRepos(localRepos []LocalRepo) {
	fmt.Println("\nFound repositories:")
	fmt.Println("==================")
	for _, repo := range localRepos {
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

func saveLocalReposData(localRepos []LocalRepo) {
	data, _ := json.MarshalIndent(localRepos, "", "  ")
	if err := os.WriteFile("local_repos.json", data, 0644); err != nil {
		fmt.Printf("Error saving local repository data: %v\n", err)
	} else {
		fmt.Println("Local repository data saved to local_repos.json")
	}
}

func promptContinue() {
	fmt.Print("Press Enter to continue with remote repository scanning or Ctrl+C to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func getRepositories(ctx context.Context, cfg Config) []Repo {
	repos := loadCachedRepos()
	if len(repos) == 0 {
		fmt.Println("Fetching repos from GitHub and GitLab...")
		repos = fetchAllRepos(ctx, cfg)
		cacheRepos(repos)
	}
	return repos
}

func selectRepositories(repos []Repo, autoMode bool) []Repo {
	if autoMode {
		return repos
	}

	selected := interactiveSelectRepos(repos)
	if len(selected) == 0 {
		fmt.Println("No selection made. Defaulting to all repositories.")
		return repos
	}
	return selected
}

func processRepositories(selected []Repo, cfg Config) {
	if !cfg.AutoMode {
		selectIntegrationMethod()
	}

	addRepos(selected)
	handleUpdatesAndPushes(selected)
}

func handleUpdatesAndPushes(selected []Repo) {
	if useSubtree {
		if updateMode {
			updateSubtrees(selected)
		}
		if pushMode {
			pushSubtrees(selected)
		}
	}
}

func ThrowMissingConfigError(err error) {
	if err != nil {
		panic("Missing config.json file with GitHub and GitLab tokens")
	}
}

func ThrowConfigJsonError(err error) {
	if err != nil {
		panic(err)
	}
}

func loadConfig() Config {
	data, err := os.ReadFile(configFile)

	ThrowMissingConfigError(err)
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	ThrowConfigJsonError(err)
	return cfg
}

func fetchAllRepos(ctx context.Context, cfg Config) []Repo {
	var wg sync.WaitGroup
	ch := make(chan []Repo, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		ch <- fetchGitHubRepos(cfg.GitHubToken)
	}()
	go func() {
		defer wg.Done()
		ch <- fetchGitLabRepos(cfg.GitLabToken)
	}()

	wg.Wait()
	close(ch)

	var allRepos []Repo
	for r := range ch {
		allRepos = append(allRepos, r...)
	}
	return allRepos
}

func fetchGitHubRepos(token string) []Repo {
	client := createGitHubClient()
	headers := githubHeaders(token)

	userRepos := fetchUserRepos(client, headers)
	orgRepos := fetchOrgRepos(client, headers)

	return append(userRepos, orgRepos...)
}

func createGitHubClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func githubHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "token " + token,
		"Accept":        "application/vnd.github+json",
	}
}

func fetchUserRepos(client *http.Client, headers map[string]string) []Repo {
	return fetchGitHubRepoList(client, headers, githubAPIURL+"/user/repos?per_page=100")
}

func fetchOrgRepos(client *http.Client, headers map[string]string) []Repo {
	var orgRepos []Repo
	orgs := fetchOrganizations(client, headers)

	for _, org := range orgs {
		orgURL := fmt.Sprintf(githubAPIURL+"/orgs/%s/repos?per_page=100", org["login"].(string))
		orgRepos = append(orgRepos, fetchGitHubRepoList(client, headers, orgURL)...)
	}
	return orgRepos
}

func fetchOrganizations(client *http.Client, headers map[string]string) []map[string]interface{} {
	req := createRequest("GET", githubAPIURL+"/user/orgs", nil, headers)
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var orgs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orgs)
	return orgs
}

func fetchGitHubRepoList(client *http.Client, headers map[string]string, url string) []Repo {
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("GitHub API error (URL: %s, Status: %d): %s\n", url, resp.StatusCode, string(body))
		return nil
	}

	var data []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	var repos []Repo
	for _, r := range data {
		repos = append(repos, Repo{
			Name:          r["name"].(string),
			SSHURL:        r["ssh_url"].(string),
			DefaultBranch: r["default_branch"].(string),
		})
	}
	return repos
}

func fetchGitLabRepos(token string) []Repo {
	if token == "" {
		fmt.Println("Warning: GitLab token is empty")
		return nil
	}

	fmt.Println("Fetching GitLab repositories...")
	req := createGitLabRequest(token)
	resp, err := executeGitLabRequest(req)

	if err != nil {
		fmt.Printf("Error connecting to GitLab API: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	return parseGitLabResponse(resp, token)
}

func createGitLabRequest(token string) *http.Request {
	req, _ := http.NewRequest("GET", gitlabAPIURL+"/projects?membership=true&per_page=100", nil)
	req.Header.Add("PRIVATE-TOKEN", token)
	return req
}

func executeGitLabRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func parseGitLabResponse(resp *http.Response, token string) []Repo {
	if resp.StatusCode != http.StatusOK {
		handleGitLabError(resp)
		return nil
	}

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Error decoding GitLab response: %v\n", err)
		return nil
	}

	return processGitLabRepos(data, token)
}

func processGitLabRepos(data []map[string]interface{}, token string) []Repo {
	fmt.Printf("Found %d GitLab repositories\n", len(data))
	var repos []Repo
	for _, r := range data {
		httpURL, ok := r["http_url_to_repo"].(string)
		if !ok {
			fmt.Printf("Warning: Could not get HTTP URL for repo %v\n", r["name"])
			continue
		}

		// Add the token to the HTTPS URL for authentication
		repoURL := strings.Replace(httpURL, "https://", fmt.Sprintf("https://oauth2:%s@", token), 1)

		name, _ := r["name"].(string)
		defaultBranch, _ := r["default_branch"].(string)

		fmt.Printf("Adding GitLab repo: %s (branch: %s)\n", name, defaultBranch)
		fmt.Printf("URL: %s\n", httpURL) // Print the URL without the token for debugging

		repos = append(repos, Repo{
			Name:          name,
			SSHURL:        repoURL,
			DefaultBranch: defaultBranch,
		})
	}

	return repos
}

func cacheRepos(repos []Repo) {
	data, _ := json.MarshalIndent(repos, "", "  ")
	_ = os.WriteFile(cacheFile, data, 0644)
}

func loadCachedRepos() []Repo {
	if _, err := os.Stat(cacheFile); err != nil {
		return nil
	}
	data, _ := os.ReadFile(cacheFile)
	var repos []Repo
	json.Unmarshal(data, &repos)
	return repos
}

func interactiveSelectRepos(repos []Repo) []Repo {
	fmt.Println("Select repositories to include (type name, enter empty to finish):")
	for i, r := range repos {
		fmt.Printf("[%d] %s (default branch: %s)\n", i, r.Name, r.DefaultBranch)
	}

	scanner := bufio.NewScanner(os.Stdin)
	var selected []Repo
	for {
		fmt.Print("Repo name (or enter to finish): ")
		scanner.Scan()
		input := scanner.Text()
		if input == "" {
			break
		}
		for _, r := range repos {
			if strings.EqualFold(r.Name, input) {
				selected = append(selected, r)
				break
			}
		}
	}
	return selected
}

func selectIntegrationMethod() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Choose integration method: [1] Submodule, [2] Subtree")
	fmt.Print("Enter choice (1 or 2): ")
	scanner.Scan()
	choice := scanner.Text()
	if choice == "2" {
		useSubtree = true
	}
}

func initMonorepo() {
	absPath := getMonorepoPath()
	createMonorepoDirectories(absPath)

	if !isGitInitialized(absPath) {
		initializeNewMonorepo(absPath)
	} else {
		ensureMainBranchExists(absPath)
	}

	verifyCleanWorkingTree()
}

func createMonorepoDirectories(absPath string) {
	// Create the monorepo directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		fmt.Printf("Error creating monorepo directory: %v\n", err)
		os.Exit(1)
	}

	// Create the repos subdirectory
	if err := os.MkdirAll(filepath.Join(absPath, "repos"), 0755); err != nil {
		fmt.Printf("Error creating repos directory: %v\n", err)
		os.Exit(1)
	}
}

func initializeNewMonorepo(absPath string) {
	fmt.Printf("Initializing git repository in %s\n", absPath)

	// Initialize git repository
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error initializing git repository: %v\n", err)
		os.Exit(1)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "monorepo@example.com")
	cmd.Dir = absPath
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Monorepo")
	cmd.Dir = absPath
	cmd.Run()

	// Create initial file
	initialFile := filepath.Join(absPath, ".gitkeep")
	if err := os.WriteFile(initialFile, []byte("initial"), 0644); err != nil {
		fmt.Printf("Error creating initial file: %v\n", err)
		os.Exit(1)
	}

	// Add and commit initial file
	cmd = exec.Command("git", "add", ".gitkeep")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error adding initial file: %v\n", err)
		os.Exit(1)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = absPath
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error committing initial file: %v\n", err)
		os.Exit(1)
	}
}

func ensureMainBranchExists(absPath string) {
	// If git is already initialized, ensure we have a main branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = absPath
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		// Create main branch if it doesn't exist
		cmd = exec.Command("git", "branch", "-M", "main")
		cmd.Dir = absPath
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error creating main branch: %v\n", err)
			os.Exit(1)
		}
	}
}

func verifyCleanWorkingTree() {
	if !isCleanWorkingTree() {
		fmt.Println("❌ Working tree is dirty. Please commit or stash changes before continuing.")
		os.Exit(1)
	}
}

func isCleanWorkingTree() bool {
	absPath, err := filepath.Abs(monorepoDir)
	if err != nil {
		return false
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = absPath
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}

func isGitInitialized(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func addRepos(selected []Repo) {
	if !isCleanWorkingTree() {
		exitWithDirtyTree()
	}

	failed, success := attemptInitialAdd(selected)
	_, success = retryFailedRepos(failed, success)
	commitSuccessfulAdds(success)
}

func attemptInitialAdd(repos []Repo) ([]Repo, []Repo) {
	var failed, success []Repo
	for _, r := range repos {
		if repoExists(r) {
			fmt.Printf("Skipping %s: already exists\n", r.Name)
			continue
		}

		if addSingleRepo(r) {
			success = append(success, r)
		} else {
			failed = append(failed, r)
		}
	}
	return failed, success
}

func retryFailedRepos(failed, success []Repo) ([]Repo, []Repo) {
	if len(failed) == 0 {
		return failed, success
	}

	fmt.Println("\nRetrying failed repositories...")
	var newFailed []Repo
	for _, r := range failed {
		if repoExists(r) {
			continue
		}

		if addSingleRepo(r) {
			success = append(success, r)
		} else {
			newFailed = append(newFailed, r)
		}
	}
	return newFailed, success
}

func commitSuccessfulAdds(success []Repo) {
	if len(success) > 0 {
		cmd := exec.Command("git", "commit", "-m", "Add selected repos")
		cmd.Dir = monorepoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error committing added repositories: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("No repositories were successfully added.")
	}
}

func addSingleRepo(r Repo) bool {
	dir := filepath.Join(monorepoDir, "repos", r.Name)
	if _, err := os.Stat(dir); err == nil {
		fmt.Printf("Skipping %s: already exists\n", r.Name)
		return false
	}

	fmt.Printf("\nAttempting to add repository: %s\n", r.Name)
	// Print URL with redacted token
	sanitizedURL := strings.Replace(r.SSHURL, "oauth2:"+strings.Split(r.SSHURL, "@")[0][7:], "oauth2:***", 1)
	fmt.Printf("Using URL: %s\n", sanitizedURL)

	var cmd *exec.Cmd
	if useSubtree {
		cmd = exec.Command("git", "subtree", "add", "--prefix", filepath.Join("repos", r.Name), r.SSHURL, r.DefaultBranch, "--squash")
	} else {
		cmd = exec.Command("git", "submodule", "add", "-b", r.DefaultBranch, r.SSHURL, filepath.Join("repos", r.Name))
	}
	cmd.Dir = monorepoDir

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Print the exact command being run (with redacted token)
	cmdStr := strings.Join(cmd.Args, " ")
	cmdStr = strings.Replace(cmdStr, r.SSHURL, sanitizedURL, 1)
	fmt.Printf("Running command: git %s\n", cmdStr)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error adding repository %s:\n", r.Name)
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Stdout: %s\n", stdout.String())
		fmt.Printf("Stderr: %s\n", stderr.String())
		return false
	}

	fmt.Printf("Successfully added %s\n", r.Name)
	return true
}

func repoExists(r Repo) bool {
	dir := filepath.Join(monorepoDir, "repos", r.Name)
	_, err := os.Stat(dir)
	return err == nil
}

func exitWithDirtyTree() {
	fmt.Println("❌ Working tree is dirty. Please commit or stash changes before continuing.")
	os.Exit(1)
}

func updateSubtrees(repos []Repo) {
	for _, r := range repos {
		fmt.Printf("Updating subtree: %s\n", r.Name)
		cmd := exec.Command("git", "subtree", "pull", "--prefix", filepath.Join("repos", r.Name), r.SSHURL, r.DefaultBranch, "--squash")
		cmd.Dir = monorepoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
	fmt.Println("Subtree updates complete.")
}

func pushSubtrees(repos []Repo) {
	for _, r := range repos {
		fmt.Printf("Pushing subtree: %s\n", r.Name)
		cmd := exec.Command("git", "subtree", "push", "--prefix", filepath.Join("repos", r.Name), r.SSHURL, r.DefaultBranch)
		cmd.Dir = monorepoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
	fmt.Println("Subtree pushes complete.")
}

func searchLocalRepos(baseDir string, monorepoPath string) ([]LocalRepo, error) {
	monorepoPath = resolveMonorepoPath(monorepoPath)
	var repos []LocalRepo

	err := filepath.Walk(baseDir, createWalkFunction(monorepoPath, &repos))
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}
	return repos, nil
}

func createWalkFunction(monorepoPath string, repos *[]LocalRepo) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || skipProcessing(path, monorepoPath, info) {
			return err
		}

		repo := analyzeDirectory(path, info, monorepoPath)
		*repos = append(*repos, repo)
		return nil
	}
}

func skipProcessing(path string, monorepoPath string, info os.FileInfo) bool {
	// Skip the monorepo itself
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	if absPath == monorepoPath {
		return true
	}

	// Skip non-directories
	if !info.IsDir() {
		return true
	}

	// Check if this is a git repository
	isGitRepo := isGitRepository(path)

	// If it's not a git repo but contains a .git directory, skip its subdirectories
	if !isGitRepo && containsDotGit(path) {
		return true
	}

	return false
}

func analyzeDirectory(path string, info os.FileInfo, monorepoPath string) LocalRepo {
	repo := LocalRepo{
		Path:         path,
		Name:         info.Name(),
		IsGitRepo:    isGitRepository(path),
		IsInMonorepo: isRepoInMonorepo(path, monorepoPath),
	}

	if repo.IsGitRepo {
		// Get default branch
		if branch, err := getDefaultBranch(path); err == nil {
			repo.DefaultBranch = branch
		}

		// Get last commit hash
		if hash, err := getLastCommitHash(path); err == nil {
			repo.LastCommitHash = hash
		}
	} else {
		// Initialize git repository if it's not one
		if err := initializeGitRepo(path); err == nil {
			repo.IsGitRepo = true
			repo.DefaultBranch = "main"
		}
	}

	return repo
}

func isGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = path
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func containsDotGit(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func getDefaultBranch(path string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getLastCommitHash(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func initializeGitRepo(path string) error {
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "monorepo@example.com")
	cmd.Dir = path
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Monorepo")
	cmd.Dir = path
	cmd.Run()

	return nil
}

func isRepoInMonorepo(repoPath string, monorepoPath string) bool {
	return strings.HasPrefix(repoPath, monorepoPath)
}

func getMonorepoPath() string {
	absPath, err := filepath.Abs(monorepoDir)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	return absPath
}

func resolveMonorepoPath(monorepoPath string) string {
	absPath, err := filepath.Abs(monorepoPath)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		os.Exit(1)
	}
	return absPath
}

func createRequest(method, url string, body io.Reader, headers map[string]string) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	return req
}

func handleGitLabError(resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("GitLab API error (status %d): %s\n", resp.StatusCode, string(body))
}
