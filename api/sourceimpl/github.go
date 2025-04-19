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
	// ErrGHCommandNotFound represents an error when the gh command is not found
	ErrGHCommandNotFound ErrCode = "GHCommandNotFound"
	// ErrGHCommandFailed represents an error when the gh command fails
	ErrGHCommandFailed ErrCode = "GHCommandFailed"
	// ErrREADMENotFound represents an error when README is not found
	ErrREADMENotFound ErrCode = "READMENotFound"

	// EnvGHCommand is the environment variable name for specifying gh command path
	EnvGHCommand = "MIRU_GH_BIN"
	// DefaultGHCommand is the default command name for GitHub CLI
	DefaultGHCommand = "gh"
)

// githubRepoResponse represents the GitHub API response for repository information
type githubRepoResponse struct {
	Homepage string `json:"homepage"`
}

// githubContentsResponse represents the GitHub API response for repository contents
type githubContentsResponse struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

// fetchGitHub fetches the README content from a GitHub repository
// Returns the content, related sources, and any error
func fetchGitHub(pkgPath string) (string, []source.RelatedReference, error) {
	// Strip ".*github.com/" prefix from package path
	pos := strings.Index(pkgPath, "github.com/")
	if pos != -1 {
		pkgPath = pkgPath[pos+len("github.com/"):]
	}

	// Get gh command path from environment variable or use default
	ghCmd := DefaultGHCommand
	if cmd := os.Getenv(EnvGHCommand); cmd != "" {
		ghCmd = cmd
	}

	// Check if gh command exists
	if _, err := exec.LookPath(ghCmd); err != nil {
		return "", nil, failure.New(ErrGHCommandNotFound,
			failure.Message(fmt.Sprintf("gh command not found at %s. Please install GitHub CLI: https://cli.github.com/ or set %s environment variable", ghCmd, EnvGHCommand)),
			failure.Context{
				"error": err.Error(),
				"path":  ghCmd,
			},
		)
	}

	// Extract owner and repo from package path (already trimmed of github.com/)
	parts := strings.Split(pkgPath, "/")
	if len(parts) < 2 {
		return "", nil, failure.New(ErrInvalidPackagePath,
			failure.Message("Invalid GitHub package path"),
			failure.Context{"path": pkgPath},
		)
	}
	owner := parts[0]
	repo := parts[1]

	// Get repository information using gh api
	cmd := exec.Command(ghCmd, "api", fmt.Sprintf("/repos/%s/%s", owner, repo))
	output, err := cmd.Output()
	if err != nil {
		return "", nil, failure.New(ErrGHCommandFailed,
			failure.Message("Failed to fetch repository information"),
			failure.Context{
				"error": err.Error(),
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	// Parse JSON response for repository information
	var repoInfo githubRepoResponse
	if err := json.Unmarshal(output, &repoInfo); err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Get repository contents using gh api
	cmd = exec.Command(ghCmd, "api", fmt.Sprintf("/repos/%s/%s/contents", owner, repo))
	output, err = cmd.Output()
	if err != nil {
		return "", nil, failure.New(ErrGHCommandFailed,
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
		return "", nil, failure.Wrap(err)
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

	// Add homepage if available
	if repoInfo.Homepage != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.TypeHomepage,
			URL:  repoInfo.Homepage,
			From: "api",
		})
	}

	return docContent, sources, nil
}

// Implementation of GitHub Investigator
type GitHubInvestigator struct{}

func (i *GitHubInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from GitHub
	content, RelatedSources, err := fetchGitHub(packagePath)
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

func (i *GitHubInvestigator) GetURL(packagePath string) string {
	// Strip ".*github.com/" prefix from package path
	pos := strings.Index(packagePath, "github.com/")
	if pos != -1 {
		packagePath = packagePath[pos+len("github.com/"):]
	}
	return fmt.Sprintf("https://github.com/%s", packagePath)
}

func (i *GitHubInvestigator) GetSourceType() source.Type {
	return source.TypeGitHub
}

func (i *GitHubInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from GitHub URL
	// Example: https://github.com/username/repo -> username/repo
	prefix := "https://github.com/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid GitHub package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
