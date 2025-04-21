package sourceimpl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ka2n/miru/api/source"
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

// fetchGitlab fetches the README content from a GitLab repository
// Returns the content, related sources, and any error
func fetchGitlab(pkgPath string) (string, []source.RelatedReference, error) {
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
		return "", nil, failure.New(ErrInvalidPackagePath,
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
	sources := extractRelatedSources(docContent, repo)

	return docContent, sources, nil
}

// Implementation of GitLab Investigator
type GitLabInvestigator struct{}

func (i *GitLabInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from GitLab
	content, RelatedSources, err := fetchGitlab(packagePath)
	if err != nil {
		return source.Data{}, err
	}

	// Generate browser URL
	browserURL, _ := url.Parse(i.GetURL(packagePath))

	return source.Data{
		Contents:       map[string]string{"README.md": content},
		FetchedAt:      time.Now(),
		RelatedSources: RelatedSources,
		BrowserURL:     browserURL,
	}, nil
}

func (i *GitLabInvestigator) GetURL(packagePath string) string {
	// Strip ".*gitlab.com/" prefix from package path
	pos := strings.Index(packagePath, "gitlab.com/")
	if pos != -1 {
		packagePath = packagePath[pos+len("gitlab.com/"):]
	}
	return fmt.Sprintf("https://gitlab.com/%s", packagePath)
}

func (i *GitLabInvestigator) GetSourceType() source.Type {
	return source.TypeGitLab
}

func (i *GitLabInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from GitLab URL
	// Example: https://gitlab.com/username/repo -> username/repo
	prefix := "https://gitlab.com/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid GitLab package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
