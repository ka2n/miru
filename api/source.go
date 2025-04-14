package api

import (
	"strings"
)

// detectSourceTypeFromURL detects the source type from a URL
func detectSourceTypeFromURL(url string) SourceType {
	switch {
	case strings.Contains(url, "github.com"):
		return SourceTypeGitHub
	case strings.Contains(url, "gitlab.com"):
		return SourceTypeGitLab
	case strings.Contains(url, "rubygems.org"):
		return SourceTypeRubyGems
	case strings.Contains(url, "npmjs.com"):
		return SourceTypeNPM
	case strings.Contains(url, "jsr.io"):
		return SourceTypeJSR
	case strings.Contains(url, "pkg.go.dev"):
		return SourceTypeGoPkgDev
	case strings.Contains(url, "crates.io"):
		return SourceTypeCratesIO
	default:
		return SourceTypeUnknown
	}
}

// cleanupRepositoryURL converts a git-clonable URL to a browser-viewable URL
func cleanupRepositoryURL(url string) string {
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
	sourceType := detectSourceTypeFromURL(url)
	switch sourceType {
	case SourceTypeGitHub:
		// GitHub URLs are already in the correct format
		return url
	case SourceTypeGitLab:
		// GitLab URLs might need normalization
		if strings.Contains(url, "/-/") {
			// Remove any /-/ in the path as it's not needed for viewing
			url = strings.Replace(url, "/-/", "/", -1)
		}
		return url
	default:
		// For other services, return the normalized URL
		return url
	}
}
