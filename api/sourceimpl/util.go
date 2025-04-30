package sourceimpl

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/ka2n/miru/api/cache"
	"github.com/ka2n/miru/api/source"
	"github.com/ka2n/miru/log"
	"github.com/mackee/go-readability"
)

// execCmdJSON executes a command and unmarshals the JSON output into the provided struct
func execCmdJSON(cmdStr string, args []string, out interface{}) error {
	logger := log.Logger.With("cmd", cmdStr, "args", args)

	logger.Debug("Executing command")
	cmd := exec.Command(cmdStr, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	decoder := json.NewDecoder(stdoutPipe)
	err = decoder.Decode(out)
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			logger.Error("Command failed", "error", exitError.Error())
		} else {
			logger.Error("Command error", "error", err.Error())
		}
		return err
	}
	logger.Debug("Command completed successfully")
	return nil
}

func fetchHTML(url *url.URL, forceUpdate bool) (string, error) {
	// Generate cache key
	cacheKey := url.String()

	// Create cache instance for string type
	htmlCache := cache.New[string]("html")

	// Get HTML from cache or fetch it
	html, err := htmlCache.GetOrSet(cacheKey, func() (string, error) {
		// Create HTTP client
		client := &http.Client{}

		// Create request
		req, err := http.NewRequest("GET", url.String(), nil)
		if err != nil {
			return "", err
		}

		// Set user agent to avoid being blocked
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(body), nil
	}, forceUpdate)

	return html, err
}

// FetchHTML fetches HTML content from a URL with cache support
// It uses the cache.GetOrSet function to retrieve HTML from cache or fetch it if not available
// The cache key is generated from the URL
// The forceUpdate parameter can be used to ignore the cache and fetch fresh HTML
func FetchHTML(url *url.URL, forceUpdate bool) (string, error) {
	content, err := fetchHTML(url, forceUpdate)
	if err != nil {
		return "", err
	}

	md, err := markdown(url, content)
	if err != nil {
		return content, nil
	}

	return md, nil
}

func markdown(url *url.URL, body string) (string, error) {
	// Convert HTML to Markdown using readability first
	article, err := readability.Extract(body, readability.DefaultOptions())
	if err != nil {
		return "", err
	}

	if article.Root != nil {
		return readability.ToMarkdown(article.Root), nil
	}

	// If readability fails, use html2md as a fallback
	converter := html2md.NewConverter(url.Host, true, &html2md.Options{})
	md, err := converter.ConvertString(body)
	if err != nil {
		return "", err
	}
	return md, nil
}

// cleanupURL converts a git-clonable URL or other url to a browser-viewable URL
func cleanupURL(url string, t source.Type) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle git+https:// prefix
	url = strings.TrimPrefix(url, "git+")

	// Handle git:// protocol
	url = strings.TrimPrefix(url, "git://")

	// Handle SSH format (git@host:path)
	if strings.HasPrefix(url, "git@") {
		// Convert git@host:path to https://host/path
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		url = "https://" + url
	}

	// Handle ssh:// protocol
	if strings.HasPrefix(url, "ssh://") {
		// Convert ssh://git@host/path to https://host/path
		url = strings.TrimPrefix(url, "ssh://")
		url = strings.TrimPrefix(url, "git@")
		url = "https://" + url
	}

	// Ensure https:// prefix if not present
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// Handle specific hosting services
	if t == source.TypeUnknown {
		t = source.DetectSourceTypeFromURL(url)
	}

	if t.IsRepository() {
		// remove fragment part if present
		url = strings.Split(url, "#")[0]
	}

	switch t {
	case source.TypeGitHub:
		// GitHub URLs are already in the correct format
		return url
	case source.TypeGitLab:
		// GitLab URLs might need normalization
		if strings.Contains(url, "/-/") {
			// Remove any /-/ in the path as it's not needed for viewing
			url = strings.ReplaceAll(url, "/-/", "/")
		}
		return url
	default:
		// For other services, return the normalized URL
		return url
	}
}
