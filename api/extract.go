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
	Description    string         // Description of the source type
}

// Known patterns for package references in documentation
var sourcePatterns = []sourcePattern{
	{
		Type:           SourceTypeJSR,
		URLPattern:     regexp.MustCompile(`https?://jsr\.io/(@[^/]+/([^/\s]+))`),
		CommandPattern: regexp.MustCompile(`jsr add (@[^\s]+)`),
		Description:    "JSR package reference",
	},
	{
		Type:           SourceTypeJSR,
		CommandPattern: regexp.MustCompile(`deno add jsr:(@[^\s]+)`),
		Description:    "JSR package reference for Deno",
	},
	{
		Type:           SourceTypeNPM,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?npmjs\.com/package/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`(?:npm|yarn|pnpm) (?:add|install|create) ([^@\s]+)`),
		Description:    "NPM package reference",
	},
	{
		Type:           SourceTypeGoPkgDev,
		URLPattern:     regexp.MustCompile(`https?://pkg\.go\.dev/(?:badge/)?([^\s\.]+)(?:\.svg)?`),
		CommandPattern: regexp.MustCompile(`go (?:get|install|test)(?:\s-u)? ([^@\s]+)`),
		Description:    "Go package reference",
	},
	{
		Type:           SourceTypeCratesIO,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?crates\.io/crates/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`cargo add ([^@\s]+)`),
		Description:    "Cargo package reference",
	},
	{
		Type:           SourceTypeRubyGems,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?rubygems\.org/gems/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`gem install ([^@\s]+)`),
		Description:    "RubyGems package reference",
	},
	{
		Type:           SourceTypePyPI,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?pypi\.org/project/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`pip install ([^@=\s]+)`),
		Description:    "Python package reference",
	},
	{
		Type:           SourceTypePackagist,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?packagist\.org/packages/([^/\s]+/[^/\s]+)`),
		CommandPattern: regexp.MustCompile(`composer (?:require|install) ([^@\s]+)`),
		Description:    "PHP package reference",
	},
}

// extractSourcesFromURLs extracts RelatedSource entries from URLs.
func extractSourcesFromURLs(urls []string) []RelatedSource {
	var sources []RelatedSource

	for _, url := range urls {
		for _, pattern := range sourcePatterns {
			if pattern.URLPattern == nil {
				continue
			}
			if matches := pattern.URLPattern.FindStringSubmatch(url); len(matches) > 1 {
				pkgName := matches[1]
				pkgUrl := generatePackageURL(pattern.Type, pkgName)
				sources = append(sources, RelatedSource{
					Type: RelatedSourceTypeFromString(pattern.Type.String()),
					URL:  pkgUrl,
					From: "document",
				})
				break
			}
		}
	}

	return sources
}

// extractSourcesFromCommands extracts RelatedSource entries from installation commands.
func extractSourcesFromCommands(content string) []RelatedSource {
	var sources []RelatedSource

	for _, pattern := range sourcePatterns {
		if pattern.CommandPattern == nil {
			continue
		}

		matches := pattern.CommandPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pkgName := match[1]
				url := generatePackageURL(pattern.Type, pkgName)
				if url != "" {
					sources = append(sources, RelatedSource{
						Type: RelatedSourceTypeFromString(pattern.Type.String()),
						URL:  url,
						From: "document",
					})
				}
			}
		}
	}

	return sources
}

// filterAndDeduplicate filters and deduplicates RelatedSource entries.
func filterAndDeduplicate(sources []RelatedSource, currentPackage string) []RelatedSource {
	var filtered []RelatedSource
	seen := make(map[string]bool)

	for _, source := range sources {
		if seen[source.URL] {
			continue
		}

		if strings.Contains(source.URL, currentPackage) {
			filtered = append(filtered, source)
			seen[source.URL] = true
		}
	}

	return filtered
}

// ExtractRelatedSources finds related documentation sources in the given content
// by matching URLs and package installation commands.
// It returns a deduplicated list of RelatedSource entries that match the current package.
func ExtractRelatedSources(content, currentPackage string) []RelatedSource {
	// Extract sources from URLs and commands
	sources := extractSourcesFromURLs(extractURLs(content))
	sources = append(sources, extractSourcesFromCommands(content)...)

	// Filter and deduplicate sources
	return filterAndDeduplicate(sources, currentPackage)
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
				if len(match) > 2 {
					u := match[2]
					// strip hash fragments
					u = strings.Split(u, "#")[0]

					if seen[u] {
						continue
					}
					urls = append(urls, u)
					seen[u] = true
				}
			}
		} else {
			// Extract raw URLs from non-Markdown-link lines
			matches := urlPattern.FindAllString(line, -1)
			for _, url := range matches {
				// strip hash fragments
				url = strings.Split(url, "#")[0]
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
	case SourceTypeJSR:
		return fmt.Sprintf("https://jsr.io/%s", pkgName)
	case SourceTypePyPI:
		return fmt.Sprintf("https://pypi.org/project/%s", pkgName)
	case SourceTypePackagist:
		return fmt.Sprintf("https://packagist.org/packages/%s", pkgName)
	default:
		panic("Unsupported source type: " + sourceType)
	}
}
