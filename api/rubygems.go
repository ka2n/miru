package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrRubyGemsREADMENotFound represents an error when README is not found
	ErrRubyGemsREADMENotFound ErrorCode = "RubyGemsREADMENotFound"
)

// rubyGemsPackageInfo represents the RubyGems package information from API
type rubyGemsPackageInfo struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Info          string   `json:"info"`
	Homepage      string   `json:"homepage_uri"`
	Source        string   `json:"source_code_uri"`
	Documentation string   `json:"documentation_uri"`
	Version       string   `json:"version"`
	Platform      string   `json:"platform"`
	DownloadCount int      `json:"downloads"`
	Authors       string   `json:"authors"`
	Licenses      []string `json:"licenses"`
}

// FetchRubyGemsReadme fetches the package information from RubyGems API
// Returns the formatted documentation and DocSource containing related sources
func FetchRubyGemsReadme(pkgPath string) (string, *DocSource, error) {
	// Get package information from RubyGems API
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil, failure.New(ErrRubyGemsREADMENotFound,
			failure.Message("Package not found"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	// Parse JSON response
	var info rubyGemsPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Format the documentation text
	doc := formatRubyGemsDoc(info)

	// Extract related sources from API response
	var sources []RelatedSource
	if info.Homepage != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeHomepage,
			URL:  info.Homepage,
			From: "api",
		})
	}
	if info.Documentation != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeDocumentation,
			URL:  info.Documentation,
			From: "api",
		})
	}
	if info.Source != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeFromString(detectSourceTypeFromURL(info.Source).String()),
			URL:  cleanupRepositoryURL(info.Source),
			From: "api",
		})
	}

	// Extract additional sources from documentation
	docSources := ExtractRelatedSources(doc, pkgPath)
	sources = append(sources, docSources...)

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueSources []RelatedSource
	for _, s := range sources {
		if !seen[s.URL] {
			uniqueSources = append(uniqueSources, s)
			seen[s.URL] = true
		}
	}

	// Create result
	result := DocSource{
		Type:           SourceTypeRubyGems,
		PackagePath:    pkgPath,
		RelatedSources: uniqueSources,
		Homepage:       info.Homepage,
	}

	return doc, &result, nil
}

// formatRubyGemsDoc formats the RubyGems package information into a markdown document
func formatRubyGemsDoc(info rubyGemsPackageInfo) string {
	var sections []string

	// Title and version
	sections = append(sections, fmt.Sprintf("# %s v%s", info.Name, info.Version))

	// Description
	if info.Description != "" {
		sections = append(sections, info.Description)
	} else if info.Info != "" {
		sections = append(sections, info.Info)
	}

	// Metadata
	var metadata []string
	if info.Authors != "" {
		metadata = append(metadata, fmt.Sprintf("**Authors:** %s", info.Authors))
	}
	if len(info.Licenses) > 0 {
		metadata = append(metadata, fmt.Sprintf("**License:** %s", strings.Join(info.Licenses, ", ")))
	}
	if info.Platform != "" {
		metadata = append(metadata, fmt.Sprintf("**Platform:** %s", info.Platform))
	}
	metadata = append(metadata, fmt.Sprintf("**Downloads:** %d", info.DownloadCount))
	if len(metadata) > 0 {
		sections = append(sections, strings.Join(metadata, " • "))
	}

	// Links
	var links []string
	if info.Homepage != "" {
		links = append(links, fmt.Sprintf("**Homepage:** %s", info.Homepage))
	}
	if info.Documentation != "" {
		links = append(links, fmt.Sprintf("**Documentation:** %s", info.Documentation))
	}
	if info.Source != "" {
		links = append(links, fmt.Sprintf("**Source:** %s", info.Source))
	}
	if len(links) > 0 {
		sections = append(sections, strings.Join(links, "\n"))
	}

	// Join all sections with double newlines
	return strings.Join(sections, "\n\n")
}
