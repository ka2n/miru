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
	// ErrGLabCommandNotFound represents an error when the glab command is not found
	ErrGLabCommandNotFound ErrorCode = "GLabCommandNotFound"
	// ErrGLabCommandFailed represents an error when the glab command fails
	ErrGLabCommandFailed ErrorCode = "GLabCommandFailed"
)

// gitlabContentsResponse represents the GitLab API response for repository contents
type gitlabContentsResponse struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

// FetchGitLabReadme fetches the README content from a GitLab repository
func FetchGitLabReadme(pkgPath string) (string, error) {
	// Check if glab command exists
	if _, err := exec.LookPath("glab"); err != nil {
		return "", failure.New(ErrGLabCommandNotFound,
			failure.Message("glab command not found. Please install GitLab CLI: https://gitlab.com/gitlab-org/cli"),
			failure.Context{"error": err.Error()},
		)
	}

	// Extract owner and repo from package path (already trimmed of gitlab.com/)
	parts := strings.Split(pkgPath, "/")
	if len(parts) < 2 {
		return "", failure.New(ErrDocumentationFetch,
			failure.Message("Invalid GitLab package path"),
			failure.Context{"path": pkgPath},
		)
	}
	owner := parts[0]
	repo := parts[1]

	// Get repository contents using glab api with pagination
	cmd := exec.Command("glab", "api", fmt.Sprintf("/projects/%s%%2F%s/repository/tree", owner, repo), "--paginate")
	output, err := cmd.Output()
	if err != nil {
		return "", failure.New(ErrGLabCommandFailed,
			failure.Message("Failed to fetch repository contents"),
			failure.Context{
				"error": err.Error(),
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	// Parse JSON response
	var allContents []gitlabContentsResponse
	if err := json.Unmarshal(output, &allContents); err != nil {
		return "", failure.Wrap(err)
	}

	// Find README file
	var readmeURL string
	for _, file := range allContents {
		if strings.HasPrefix(strings.ToLower(file.Name), "readme.") || strings.ToLower(file.Name) == "readme" {
			// GitLab API doesn't provide direct download URLs in the tree endpoint
			// Construct the raw content URL
			readmeURL = fmt.Sprintf("https://gitlab.com/%s/%s/-/raw/main/%s", owner, repo, file.Name)
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
