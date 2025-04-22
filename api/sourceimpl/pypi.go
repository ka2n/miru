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

// fetchPyPI fetches the README content from PyPI registry
// Returns the content, related sources, and any error
func fetchPyPI(pkgPath string) (string, []source.RelatedReference, error) {
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

	if resp.StatusCode != http.StatusOK {
		return "", nil, failure.New(ErrRepositoryNotFound,
			failure.Message("Failed to fetch package information from pypi.org"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

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
	var sources []source.RelatedReference

	// Add homepage if available
	if info.Info.Homepage != "" {
		detected := source.DetectSourceTypeFromURL(info.Info.Homepage)
		if detected != source.TypeUnknown {
			// Add as repository if the URL is from GitHub/GitLab
			sources = append(sources, source.RelatedReference{
				Type: detected,
				URL:  cleanupURL(info.Info.Homepage, source.TypeUnknown),
				From: "api",
			})
		} else {
			// Add as homepage for other URLs
			sources = append(sources, source.RelatedReference{
				Type: source.TypeHomepage,
				URL:  info.Info.Homepage,
				From: "api",
			})
		}
	}

	// Add related sources from Project URLs
	for name, url := range info.Info.ProjectURLs {
		// Detect source type from URL
		detectedType := source.DetectSourceTypeFromURL(url)
		var sourceType source.Type

		// Set related source type based on detected source type
		if detectedType.IsRepository() {
			// Use as is if it's a repository
			sourceType = detectedType
		} else {
			// Determine based on name
			switch strings.ToLower(name) {
			case "homepage", "home":
				sourceType = source.TypeHomepage
			case "repository", "source", "source code", "code":
				// Detect source type from repository URL
				repoType := source.DetectSourceTypeFromURL(url)
				sourceType = repoType
			default:
				// Default is documentation
				sourceType = source.TypeDocumentation
			}
		}

		sources = append(sources, source.RelatedReference{
			Type: sourceType,
			URL:  url,
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := extractRelatedSources(info.Info.Description, pkgPath)
	sources = append(sources, docSources...)

	return info.Info.Description, sources, nil
}

// Implementation of PyPI Investigator
type PyPIInvestigator struct{}

func (i *PyPIInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from pypi.org
	content, RelatedSources, err := fetchPyPI(packagePath)
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

func (i *PyPIInvestigator) GetURL(packagePath string) string {
	// For PyPI, use only the package name without organization
	pkgName := packagePath
	if idx := strings.LastIndex(packagePath, "/"); idx != -1 {
		pkgName = packagePath[idx+1:]
	}
	return fmt.Sprintf("https://pypi.org/project/%s", pkgName)
}

func (i *PyPIInvestigator) GetSourceType() source.Type {
	return source.TypePyPI
}

func (i *PyPIInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from PyPI URL
	// Example: https://pypi.org/project/package-name -> package-name
	prefix := "https://pypi.org/project/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid PyPI package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
