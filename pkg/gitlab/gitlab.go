package gitlab

import (
	"christopherharwell/project_monorepo/pkg/types"
	"context"
	"net/http"
)

const gitlabAPIURL = "https://gitlab.com/api/v4"

func FetchRepos(ctx context.Context, token string) []types.Repo {
	req := createRequest(token)
	resp, err := executeRequest(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	return parseResponse(resp, token)
}

func createRequest(token string) *http.Request {
	// Implementation moved from main.go
	return nil
}

func executeRequest(req *http.Request) (*http.Response, error) {
	// Implementation moved from main.go
	return nil, nil
}

func parseResponse(resp *http.Response, token string) []types.Repo {
	// Implementation moved from main.go
	return nil
} 