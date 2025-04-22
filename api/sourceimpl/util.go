package sourceimpl

import (
	"strings"

	"github.com/ka2n/miru/api/source"
)

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
