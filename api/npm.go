package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrNPMREADMENotFound represents an error when README is not found
	ErrNPMREADMENotFound ErrorCode = "NPMREADMENotFound"
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

// FetchNPMReadme fetches the README content from npm registry
// Returns the content, DocSource with related sources, and any error
func FetchNPMReadme(pkgPath string) (string, *DocSource, error) {
	// Get package information from npm registry
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	// Parse JSON response
	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, failure.Wrap(err)
	}

	if info.Readme == "" {
		return "", nil, failure.New(ErrNPMREADMENotFound,
			failure.Message("README not found in package"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	// Extract related sources from content and API response
	var sources []RelatedSource

	// Add homepage if available
	if info.Homepage != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeHomepage,
			URL:  info.Homepage,
			From: "api",
		})
	}

	// Add repository if available
	if info.Repository.URL != "" {
		sources = append(sources, RelatedSource{
			Type: detectSourceTypeFromURL(info.Repository.URL).String(),
			URL:  cleanupRepositoryURL(info.Repository.URL),
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := ExtractRelatedSources(info.Readme, pkgPath)
	sources = append(sources, docSources...)

	// Create DocSource with related sources
	result := &DocSource{
		Type:           SourceTypeNPM,
		PackagePath:    pkgPath,
		RelatedSources: sources,
		Homepage:       info.Homepage,
	}

	return info.Readme, result, nil
}
