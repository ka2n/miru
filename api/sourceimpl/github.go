package sourceimpl

import (
	"encoding/base64"
	"fmt"
	"io"
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
	ErrGHCommandNotFound ErrorCode = "GHCommandNotFound"
	// ErrGHCommandFailed represents an error when the gh command fails
	ErrGHCommandFailed ErrorCode = "GHCommandFailed"
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
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
}

type githubContentResponse struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
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

	// Remove query parameters or fragments from repo name
	if idx := strings.Index(repo, "?"); idx != -1 {
		repo = repo[:idx]
	}
	if idx := strings.Index(repo, "#"); idx != -1 {
		repo = repo[:idx]
	}
	if repo == "" {
		return "", nil, failure.New(ErrInvalidPackagePath,
			failure.Message("Invalid GitHub package path"),
			failure.Context{"path": pkgPath},
		)
	}

	// Get repository information using gh api
	reqpath := fmt.Sprintf("/repos/%s/%s", owner, repo)
	var info githubRepoResponse
	if err := execCmdJSON(ghCmd, []string{"api", reqpath}, &info); err != nil {
		return "", nil, failure.New(ErrGHCommandFailed,
			failure.Message("Failed to fetch repository information"),
			failure.Context{
				"error": err.Error(),
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	// Get repository contents using gh api
	reqpath = fmt.Sprintf("/repos/%s/%s/contents", owner, repo)
	var contents []githubContentsResponse
	if err := execCmdJSON(ghCmd, []string{"api", reqpath}, &contents); err != nil {
		return "", nil, failure.New(ErrGHCommandFailed,
			failure.Message("Failed to fetch repository contents"),
			failure.Context{
				"error": err.Error(),
				"owner": owner,
				"repo":  repo,
			},
		)
	}

	sources := make([]source.RelatedReference, 0)

	// Find README file
	var docContent string
	var readmePath string
	for _, file := range contents {
		if strings.HasPrefix(strings.ToLower(file.Name), "readme.") || strings.ToLower(file.Name) == "readme" {
			readmePath = file.Path
			break
		}
	}

	// Download README by GitHub API content if found.
	// We don't use download_url here, because GitHub API provides symbolic resolution for symlinked files.
	if readmePath != "" {
		// Get repository contents using gh api
		reqpath = fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, readmePath)
		var content githubContentResponse
		if err := execCmdJSON(ghCmd, []string{"api", reqpath}, &content); err != nil {
			return "", nil, failure.New(ErrGHCommandFailed,
				failure.Message("Failed to fetch README content"),
				failure.Context{
					"error": err.Error(),
					"owner": owner,
					"repo":  repo,
				},
			)
		}

		r, err := content.GetContent()
		if err != nil {
			return "", nil, failure.Wrap(err)
		}
		d, err := io.ReadAll(r)
		if err != nil {
			return "", nil, failure.Wrap(err)
		}
		docContent = string(d)

		sources = append(sources, extractRelatedSources(docContent, repo)...)
	}

	// Add homepage if available
	if info.Homepage != "" {
		detected := source.DetectSourceTypeFromURL(info.Homepage)
		if detected != source.TypeUnknown {
			// Add as repository if the URL is from GitHub/GitLab
			sources = append(sources, source.RelatedReference{
				Type: detected,
				URL:  cleanupURL(info.Homepage, detected),
				From: "api",
			})
		} else {
			// Add as homepage for other URLs
			sources = append(sources, source.RelatedReference{
				Type: source.TypeHomepage,
				URL:  info.Homepage,
				From: "api",
			})
		}
	}

	return docContent, sources, nil
}

func (c githubContentResponse) GetContent() (io.Reader, error) {
	if c.Encoding != "base64" {
		return nil, failure.New(ErrGHCommandFailed,
			failure.Message("the content is not base64 encoded"),
			failure.Context{
				"encoding": c.Encoding,
			},
		)
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(c.Content))
	return reader, nil
}

// Implementation of GitHub Investigator
type GitHubInvestigator struct{}

func (i *GitHubInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from GitHub
	content, rel, err := fetchGitHub(packagePath)
	if err != nil {
		return source.Data{}, err
	}

	// Generate browser URL
	browserURL, _ := url.Parse(i.GetURL(packagePath))

	return source.Data{
		Contents:       map[string]string{"README.md": content},
		FetchedAt:      time.Now(),
		RelatedSources: rel,
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
