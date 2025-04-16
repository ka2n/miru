package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrPyPIREADMENotFound represents an error when README is not found
	ErrPyPIREADMENotFound ErrorCode = "PyPIREADMENotFound"
)

// pypiPackageInfo represents the PyPI package information from registry
type pypiPackageInfo struct {
	Info struct {
		ProjectURLs map[string]string `json:"project_urls"`
		Description string            `json:"description"`
		Homepage    string            `json:"home_page"`
	} `json:"info"`
}

// FetchPyPIReadme fetches the README content from PyPI registry
// Returns the content, DocSource with related sources, and any error
func FetchPyPIReadme(pkgPath string) (string, *DocSource, error) {
	// Extract only the package name (remove organization name if present)
	pkgName := pkgPath
	if idx := strings.LastIndex(pkgPath, "/"); idx != -1 {
		pkgName = pkgPath[idx+1:]
	}

	// Get package information from PyPI API
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", pkgName)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	// Parse JSON response
	var info pypiPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Description is used as README
	if info.Info.Description == "" {
		return "", nil, failure.New(ErrPyPIREADMENotFound,
			failure.Message("README not found in package"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	// Extract related sources
	var sources []RelatedSource

	// Add homepage if available
	if info.Info.Homepage != "" {
		sources = append(sources, RelatedSource{
			Type: RelatedSourceTypeHomepage,
			URL:  info.Info.Homepage,
			From: "api",
		})
	}

	// Add related sources from Project URLs
	for name, url := range info.Info.ProjectURLs {
		// Detect source type from URL
		detectedType := detectSourceTypeFromURL(url)
		var sourceType RelatedSourceType

		// Set related source type based on detected source type
		if detectedType.IsRepository() {
			// Use as is if it's a repository
			sourceType = RelatedSourceTypeFromString(detectedType.String())
		} else {
			// Determine based on name
			switch strings.ToLower(name) {
			case "homepage", "home":
				sourceType = RelatedSourceTypeHomepage
			case "repository", "source", "source code", "code":
				// Detect source type from repository URL
				repoType := detectSourceTypeFromURL(url)
				sourceType = RelatedSourceTypeFromString(repoType.String())
			default:
				// Default is documentation
				sourceType = RelatedSourceTypeDocumentation
			}
		}

		sources = append(sources, RelatedSource{
			Type: sourceType,
			URL:  url,
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := ExtractRelatedSources(info.Info.Description, pkgPath)
	sources = append(sources, docSources...)

	// Create DocSource
	result := &DocSource{
		Type:           SourceTypePyPI,
		PackagePath:    pkgPath,
		RelatedSources: sources,
		Homepage:       info.Info.Homepage,
	}

	return info.Info.Description, result, nil
}
