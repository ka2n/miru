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

// npmPackageInfo represents the npm package information from registry
type npmPackageInfo struct {
	Readme     string `json:"readme"`
	Homepage   string `json:"homepage"`
	Repository struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"repository"`
}

// fetchNPM fetches the README content from npm registry
// Returns the content, related sources, and any error
func fetchNPM(pkgPath string) (string, []source.RelatedReference, error) {
	// Get package information from npm registry
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, failure.New(ErrRepositoryNotFound,
			failure.Message("Failed to fetch package information from npm registry"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	// Parse JSON response
	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Extract related sources from content and API response
	var sources []source.RelatedReference

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

	// Add repository if available
	if info.Repository.URL != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.DetectSourceTypeFromURL(info.Repository.URL),
			URL:  cleanupURL(info.Repository.URL, source.TypeUnknown),
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := extractRelatedSources(info.Readme, pkgPath)
	sources = append(sources, docSources...)

	return info.Readme, sources, nil
}

// Implementation of NPM Investigator
type NPMInvestigator struct{}

func (i *NPMInvestigator) Fetch(packagePath string) (source.Data, error) {

	// Process to retrieve data from NPM
	content, RelatedSources, err := fetchNPM(packagePath)
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

func (i *NPMInvestigator) GetURL(packagePath string) string {
	return fmt.Sprintf("https://www.npmjs.com/package/%s", packagePath)
}

func (i *NPMInvestigator) GetSourceType() source.Type {
	return source.TypeNPM
}

func (i *NPMInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from NPM URL
	// Example: https://www.npmjs.com/package/package-name -> package-name
	// Example: https://www.npmjs.com/package/org/package-name -> org/package-name
	prefix := "https://www.npmjs.com/package/"
	if strings.HasPrefix(url, prefix) {
		// Convert hyphens to slashes (if necessary)
		pkgName := url[len(prefix):]
		if pkgName == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid NPM package path"),
				failure.Context{"url": url},
			)
		}
		return pkgName, nil
	}
	return url, nil
}
