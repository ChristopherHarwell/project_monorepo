package github

import (
	"christopherharwell/project_monorepo/pkg/types"
	"context"
	"net/http"
	"time"
)

func NewClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func Headers(token string) map[string]string {
	return map[string]string{
		"Authorization": "token " + token,
		"Accept":        "application/vnd.github+json",
	}
}

func FetchRepos(ctx context.Context, token string) []types.Repo {
	client := NewClient()
	headers := Headers(token)

	userRepos := fetchUserRepos(client, headers)
	orgRepos := fetchOrgRepos(client, headers)

	return append(userRepos, orgRepos...)
}

func fetchUserRepos(client *http.Client, headers map[string]string) []types.Repo {
	// Implementation moved from main.go
	return nil
}

func fetchOrgRepos(client *http.Client, headers map[string]string) []types.Repo {
	// Implementation moved from main.go
	return nil
} 