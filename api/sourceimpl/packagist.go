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
	// ErrPackagistREADMENotFound represents an error when README is not found
	ErrPackagistREADMENotFound ErrorCode = "PackagistREADMENotFound"
)

// packagistPackageInfo represents the Packagist package information from registry
type packagistPackageInfo struct {
	Package struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Repository  string `json:"repository"`
		Homepage    string `json:"homepage"`
		Versions    map[string]struct {
			Description string `json:"description"`
			Homepage    string `json:"homepage"`
			Source      struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"versions"`
	} `json:"package"`
}

// fetchPackagist fetches the README content from Packagist registry
// Returns the content, related sources, and any error
func fetchPackagist(pkgPath string) (string, []source.RelatedReference, error) {
	// Get package information from Packagist API
	url := fmt.Sprintf("https://packagist.org/packages/%s.json", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	// Parse JSON response
	var info packagistPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Packagist does not have a README file, but it has a description
	if info.Package.Description == "" {
		// Check if there are versions available
		for _, version := range info.Package.Versions {
			if version.Description != "" {
				info.Package.Description = version.Description
				break
			}
		}

		// If still no description, return an error
		if info.Package.Description == "" {
			return "", nil, failure.New(ErrPackagistREADMENotFound,
				failure.Message("README not found in package"),
				failure.Context{
					"pkg": pkgPath,
				},
			)
		}
	}

	// Extract related sources
	var sources []source.RelatedReference

	// Add homepage if available
	if info.Package.Homepage != "" {
		detected := source.DetectSourceTypeFromURL(info.Package.Homepage)
		if detected != source.TypeUnknown {
			// Add as repository if the URL is from GitHub/GitLab
			sources = append(sources, source.RelatedReference{
				Type: detected,
				URL:  cleanupURL(info.Package.Homepage, source.TypeUnknown),
				From: "api",
			})
		} else {
			// Add as homepage for other URLs
			sources = append(sources, source.RelatedReference{
				Type: source.TypeHomepage,
				URL:  info.Package.Homepage,
				From: "api",
			})
		}
	}

	// Add repository if available
	var repoURL string
	if info.Package.Repository != "" {
		repoURL = info.Package.Repository
	} else {
		// Check versions for repository URL
		for _, version := range info.Package.Versions {
			if version.Source.URL != "" {
				repoURL = version.Source.URL
				break
			}
		}
	}

	if repoURL != "" {
		repoType := source.DetectSourceTypeFromURL(repoURL)
		sources = append(sources, source.RelatedReference{
			Type: repoType,
			URL:  cleanupURL(repoURL, source.TypeUnknown),
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := extractRelatedSources(info.Package.Description, pkgPath)
	sources = append(sources, docSources...)

	return info.Package.Description, sources, nil
}

// Implementation of Packagist Investigator
type PackagistInvestigator struct{}

func (i *PackagistInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from packagist.org
	content, RelatedSources, err := fetchPackagist(packagePath)
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

func (i *PackagistInvestigator) GetURL(packagePath string) string {
	return fmt.Sprintf("https://packagist.org/packages/%s", packagePath)
}

func (i *PackagistInvestigator) GetSourceType() source.Type {
	return source.TypePackagist
}

func (i *PackagistInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from Packagist URL
	// Example: https://packagist.org/packages/vendor/package -> vendor/package
	prefix := "https://packagist.org/packages/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid Packagist package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
