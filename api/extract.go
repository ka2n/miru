package api

import (
	"fmt"
	"regexp"
	"strings"
)

// sourcePattern defines patterns for detecting package references in content
type sourcePattern struct {
	Type           SourceType     // Source type identifier
	URLPattern     *regexp.Regexp // Pattern for matching URLs
	CommandPattern *regexp.Regexp // Pattern for matching installation commands
}

// Known patterns for package references in documentation
var sourcePatterns = []sourcePattern{
	{
		Type:       SourceTypeJSR,
		URLPattern: regexp.MustCompile(`https?://jsr\.io/@[^/]+/([^/\s]+)`),
	},
	{
		Type:           SourceTypeNPM,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?npmjs\.com/package/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`(?:npm|yarn|pnpm) (?:add|install) ([^@\s]+)`),
	},
	{
		Type:       SourceTypeGoPkgDev,
		URLPattern: regexp.MustCompile(`https?://(?:pkg\.)?go\.dev/([^/\s]+)`),
	},
	{
		Type:           SourceTypeCratesIO,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?crates\.io/crates/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`cargo add ([^@\s]+)`),
	},
	{
		Type:           SourceTypeRubyGems,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?rubygems\.org/gems/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`gem install ([^@\s]+)`),
	},
}

// ExtractRelatedSources finds related documentation sources in the given content
// by matching URLs and package installation commands.
// It returns a deduplicated list of RelatedSource entries that match the current package.
func ExtractRelatedSources(content, currentPackage string) []RelatedSource {
	var sources []RelatedSource
	seen := make(map[string]bool) // For deduplication

	// Extract URLs from content
	urls := extractURLs(content)
	for _, url := range urls {
		if seen[url] {
			continue
		}

		for _, pattern := range sourcePatterns {
			if matches := pattern.URLPattern.FindStringSubmatch(url); len(matches) > 1 {
				pkgName := matches[1]
				if pkgName == currentPackage {
					sources = append(sources, RelatedSource{
						Type: RelatedSourceTypeFromString(pattern.Type.String()),
						URL:  url,
						From: "document_link",
					})
					seen[url] = true
					break
				}
			}
		}
	}

	// Extract package references from installation commands
	for _, pattern := range sourcePatterns {
		if pattern.CommandPattern == nil {
			continue
		}

		matches := pattern.CommandPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] == currentPackage {
				url := generatePackageURL(pattern.Type, currentPackage)
				if url != "" && !seen[url] {
					sources = append(sources, RelatedSource{
						Type: RelatedSourceTypeFromString(pattern.Type.String()),
						URL:  url,
						From: "document_command",
					})
					seen[url] = true
				}
			}
		}
	}

	return sources
}

// extractURLs finds URLs in content, handling both Markdown links and raw URLs.
// It returns a slice of unique URLs found in the content.
func extractURLs(content string) []string {
	// Markdown link pattern: [text](url)
	mdPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Raw URL pattern: match URLs
	urlPattern := regexp.MustCompile(`https?://[^\s<>"]+`)

	var urls []string
	seen := make(map[string]bool)

	// Extract URLs from content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Skip URLs in Markdown links
		if strings.Contains(line, "](") {
			// Extract URLs from Markdown links
			matches := mdPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 2 && !seen[match[2]] {
					urls = append(urls, match[2])
					seen[match[2]] = true
				}
			}
		} else {
			// Extract raw URLs from non-Markdown-link lines
			matches := urlPattern.FindAllString(line, -1)
			for _, url := range matches {
				if !seen[url] {
					urls = append(urls, url)
					seen[url] = true
				}
			}
		}
	}

	return urls
}

// generatePackageURL creates a URL for a package based on its source type.
// Returns an empty string if the source type is not supported.
func generatePackageURL(sourceType SourceType, pkgName string) string {
	switch sourceType {
	case SourceTypeNPM:
		return fmt.Sprintf("https://www.npmjs.com/package/%s", pkgName)
	case SourceTypeGoPkgDev:
		return fmt.Sprintf("https://pkg.go.dev/%s", pkgName)
	case SourceTypeCratesIO:
		return fmt.Sprintf("https://crates.io/crates/%s", pkgName)
	case SourceTypeRubyGems:
		return fmt.Sprintf("https://rubygems.org/gems/%s", pkgName)
	}
	return ""
}
