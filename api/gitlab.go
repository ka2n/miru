package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrGLabCommandNotFound represents an error when the glab command is not found
	ErrGLabCommandNotFound ErrorCode = "GLabCommandNotFound"
	// ErrGLabCommandFailed represents an error when the glab command fails
	ErrGLabCommandFailed ErrorCode = "GLabCommandFailed"

	// EnvGLabCommand is the environment variable name for specifying glab command path
	EnvGLabCommand = "MIRU_GLAB_BIN"
	// DefaultGLabCommand is the default command name for GitLab CLI
	DefaultGLabCommand = "glab"
)

// gitlabContentsResponse represents the GitLab API response for repository contents
type gitlabContentsResponse struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

// FetchGitLabReadme fetches the README content from a GitLab repository
// Returns the content, DocSource with related sources, and any error
func FetchGitLabReadme(pkgPath string) (string, *DocSource, error) {
	pos := strings.Index(pkgPath, "gitlab.com/")
	if pos != -1 {
		pkgPath = pkgPath[pos+len("gitlab.com/"):]
	}

	// Get glab command path from environment variable or use default
	glabCmd := DefaultGLabCommand
	if cmd := os.Getenv(EnvGLabCommand); cmd != "" {
		glabCmd = cmd
	}

	// Check if glab command exists
	if _, err := exec.LookPath(glabCmd); err != nil {
		return "", nil, failure.New(ErrGLabCommandNotFound,
			failure.Message(fmt.Sprintf("glab command not found at %s. Please install GitLab CLI: https://gitlab.com/gitlab-org/cli or set %s environment variable", glabCmd, EnvGLabCommand)),
			failure.Context{
				"error": err.Error(),
				"path":  glabCmd,
			},
		)
	}

	// Extract owner and repo from package path (already trimmed of gitlab.com/)
	parts := strings.Split(pkgPath, "/")
	if len(parts) < 2 {
		return "", nil, failure.New(ErrDocumentationFetch,
			failure.Message("Invalid GitLab package path"),
			failure.Context{"path": pkgPath},
		)
	}
	owner := parts[0]
	repo := parts[1]

	// Get repository contents using glab api with pagination
	cmd := exec.Command(glabCmd, "api", fmt.Sprintf("/projects/%s%%2F%s/repository/tree", owner, repo), "--paginate")
	output, err := cmd.Output()
	if err != nil {
		return "", nil, failure.New(ErrGLabCommandFailed,
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
		return "", nil, failure.Wrap(err)
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
		return "", nil, failure.New(ErrREADMENotFound,
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
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Extract related sources from content
	docContent := string(content)
	sources := ExtractRelatedSources(docContent, repo)

	// Create DocSource with related sources
	result := &DocSource{
		Type:           SourceTypeGitLab,
		PackagePath:    pkgPath,
		RelatedSources: sources,
	}

	return docContent, result, nil
}
