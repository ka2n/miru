package sourceimpl

import (
	"regexp"
	"strings"

	"github.com/ka2n/miru/api/source"
	"github.com/samber/lo"
)

// sourcePattern defines patterns for detecting package references in content
type sourcePattern struct {
	Type           source.Type    // Source type identifier
	URLPattern     *regexp.Regexp // Pattern for matching URLs
	CommandPattern *regexp.Regexp // Pattern for matching installation commands
	Description    string         // Description of the source type
}

// Known patterns for package references in documentation
var sourcePatterns = []sourcePattern{
	{
		Type:           source.TypeJSR,
		URLPattern:     regexp.MustCompile(`https?://jsr\.io/(@[^/]+/([^/\s]+))`),
		CommandPattern: regexp.MustCompile(`jsr add (@[^\s]+)`),
		Description:    "JSR package reference",
	},
	{
		Type:           source.TypeJSR,
		CommandPattern: regexp.MustCompile(`deno add jsr:(@[^\s]+)`),
		Description:    "JSR package reference for Deno",
	},
	{
		Type:           source.TypeNPM,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?npmjs\.com/package/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`(?:npm|yarn|pnpm) (?:add|install|create) ([^@\s]+)`),
		Description:    "NPM package reference",
	},
	{
		Type:           source.TypeGoPkgDev,
		URLPattern:     regexp.MustCompile(`https?://pkg\.go\.dev/(?:badge/)?([^\s\.]+)(?:\.svg)?`),
		CommandPattern: regexp.MustCompile(`go (?:get|install|test)(?:\s-u)? ([^@\s]+)`),
		Description:    "Go package reference",
	},
	{
		Type:           source.TypeCratesIO,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?crates\.io/crates/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`cargo add ([^@\s]+)`),
		Description:    "Cargo package reference",
	},
	{
		Type:           source.TypeRubyGems,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?rubygems\.org/gems/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`gem install ([^@\s]+)`),
		Description:    "RubyGems package reference",
	},
	{
		Type:           source.TypePyPI,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?pypi\.org/project/([^/\s]+)`),
		CommandPattern: regexp.MustCompile(`pip install ([^@=\s]+)`),
		Description:    "Python package reference",
	},
	{
		Type:           source.TypePackagist,
		URLPattern:     regexp.MustCompile(`https?://(?:www\.)?packagist\.org/packages/([^/\s]+/[^/\s]+)`),
		CommandPattern: regexp.MustCompile(`composer (?:require|install) ([^@\s]+)`),
		Description:    "PHP package reference",
	},
}

// extractSourcesFromURLs extracts source.RelatedSource entries from URLs.
func extractSourcesFromURLs(urls []string) []source.RelatedReference {
	var sources []source.RelatedReference

	for _, url := range urls {
		for _, pattern := range sourcePatterns {
			if pattern.URLPattern == nil {
				continue
			}
			if matches := pattern.URLPattern.FindStringSubmatch(url); len(matches) > 1 {
				pkgName := matches[1]
				sources = append(sources, source.RelatedReference{
					Type: pattern.Type,
					Path: pkgName,
					From: "document",
				})
				break
			}
		}
	}

	return sources
}

// extractSourcesFromCommands extracts source.RelatedSource entries from installation commands.
func extractSourcesFromCommands(content string) []source.RelatedReference {
	var sources []source.RelatedReference

	for _, pattern := range sourcePatterns {
		if pattern.CommandPattern == nil {
			continue
		}

		matches := pattern.CommandPattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				pkgName := match[1]
				sources = append(sources, source.RelatedReference{
					Type: pattern.Type,
					Path: pkgName,
					From: "document",
				})
			}
		}
	}

	return sources
}

// filterAndDeduplicate filters and deduplicates source.RelatedSource entries.
func filterAndDeduplicate(sources []source.RelatedReference, currentPackage string) []source.RelatedReference {
	var filtered []source.RelatedReference
	seen := make(map[string]bool)

	for _, source := range sources {
		key := lo.Ternary(source.URL != "", source.URL, source.Path)
		if seen[key] {
			continue
		}

		if strings.Contains(key, currentPackage) {
			filtered = append(filtered, source)
			seen[key] = true
		}
	}

	return filtered
}

// extractRelatedSources finds related documentation sources in the given content
// by matching URLs and package installation commands.
// It returns a deduplicated list of source.RelatedSource entries that match the current package.
func extractRelatedSources(content, currentPackage string) []source.RelatedReference {
	// Extract sources from URLs and commands
	sources := extractSourcesFromURLs(extractURLs(content))
	sources = append(sources, extractSourcesFromCommands(content)...)

	// Filter and deduplicate sources
	sources = filterAndDeduplicate(sources, currentPackage)
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
