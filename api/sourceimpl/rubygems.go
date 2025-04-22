package sourceimpl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ka2n/miru/api/source"
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

// fetchRubyGemsReadme fetches the package information from RubyGems API
// Returns the formatted documentation and related sources
func fetchRubyGemsReadme(pkgPath string) (string, []source.RelatedReference, error) {
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
	var sources []source.RelatedReference
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
	if info.Documentation != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.TypeDocumentation,
			URL:  info.Documentation,
			From: "api",
		})
	}
	if info.Source != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.DetectSourceTypeFromURL(info.Source),
			URL:  cleanupURL(info.Source, source.TypeUnknown),
			From: "api",
		})
	}

	// Extract additional sources from documentation
	docSources := extractRelatedSources(doc, pkgPath)
	sources = append(sources, docSources...)

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueSources []source.RelatedReference
	for _, s := range sources {
		if !seen[s.URL] {
			uniqueSources = append(uniqueSources, s)
			seen[s.URL] = true
		}
	}

	return doc, uniqueSources, nil
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

// Implementation of RubyGems Investigator
type RubyGemsInvestigator struct{}

func (i *RubyGemsInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from rubygems.org
	content, RelatedSources, err := fetchRubyGemsReadme(packagePath)
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

func (i *RubyGemsInvestigator) GetURL(packagePath string) string {
	// For RubyGems, use only the package name without organization
	pkgName := packagePath
	if idx := strings.LastIndex(packagePath, "/"); idx != -1 {
		pkgName = packagePath[idx+1:]
	}
	return fmt.Sprintf("https://rubygems.org/gems/%s", pkgName)
}

func (i *RubyGemsInvestigator) GetSourceType() source.Type {
	return source.TypeRubyGems
}

func (i *RubyGemsInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from RubyGems URL
	// Example: https://rubygems.org/gems/package-name -> package-name
	prefix := "https://rubygems.org/gems/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid RubyGems package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
