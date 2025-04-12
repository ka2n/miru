package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrGHCommandNotFound represents an error when the gh command is not found
	ErrGHCommandNotFound ErrorCode = "GHCommandNotFound"
	// ErrGHCommandFailed represents an error when the gh command fails
	ErrGHCommandFailed ErrorCode = "GHCommandFailed"
	// ErrREADMENotFound represents an error when README is not found
	ErrREADMENotFound ErrorCode = "READMENotFound"
)

// githubContentsResponse represents the GitHub API response for repository contents
type githubContentsResponse struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

// FetchGitHubReadme fetches the README content from a GitHub repository
func FetchGitHubReadme(pkgPath string) (string, error) {
	// Check if gh command exists
	if _, err := exec.LookPath("gh"); err != nil {
		return "", failure.New(ErrGHCommandNotFound,
			failure.Message("gh command not found. Please install GitHub CLI: https://cli.github.com/"),
			failure.Context{"error": err.Error()},
		)
	}

	// Extract owner and repo from package path (already trimmed of github.com/)
	parts := strings.Split(pkgPath, "/")
	if len(parts) < 2 {
		return "", failure.New(ErrDocumentationFetch,
			failure.Message("Invalid GitHub package path"),
			failure.Context{"path": pkgPath},
		)
	}
	owner := parts[0]
	repo := parts[1]

	// Get repository contents using gh api
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/%s/contents", owner, repo))
	output, err := cmd.Output()
	if err != nil {
		return "", failure.New(ErrGHCommandFailed,
			failure.Message("Failed to fetch repository contents"),
			failure.Context{
				"error": err.Error(),
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	// Parse JSON response
	var contents []githubContentsResponse
	if err := json.Unmarshal(output, &contents); err != nil {
		return "", failure.Wrap(err)
	}

	// Find README file
	var readmeURL string
	for _, file := range contents {
		if strings.HasPrefix(strings.ToLower(file.Name), "readme.") || strings.ToLower(file.Name) == "readme" {
			readmeURL = file.DownloadURL
			break
		}
	}

	if readmeURL == "" {
		return "", failure.New(ErrREADMENotFound,
			failure.Message("README not found in repository"),
			failure.Context{
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	// Download README content
	resp, err := http.Get(readmeURL)
	if err != nil {
		return "", failure.Wrap(err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", failure.Wrap(err)
	}

	return string(content), nil
}
