package api

import (
	"encoding/json"
	"fmt"
	"net/http"

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

// FetchPackagistReadme fetches the README content from Packagist registry
// Returns the content, DocSource with related sources, and any error
func FetchPackagistReadme(pkgPath string) (string, *DocSource, error) {
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

	// Packagegist does not have a README file, but it has a description
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
	var sources []RelatedSource

	// Add homepage if available
	if info.Package.Homepage != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeHomepage,
			URL:  info.Package.Homepage,
			From: "api",
		})
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
		repoType := detectSourceTypeFromURL(repoURL)
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeFromString(repoType.String()),
			URL:  cleanupRepositoryURL(repoURL),
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := ExtractRelatedSources(info.Package.Description, pkgPath)
	sources = append(sources, docSources...)

	// Create DocSource
	result := &DocSource{
		Type:           SourceTypePackagist,
		PackagePath:    pkgPath,
		RelatedSources: sources,
		Homepage:       info.Package.Homepage,
	}

	return info.Package.Description, result, nil
}
