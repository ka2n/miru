package sourceimpl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/ka2n/miru/api/source"
	"github.com/morikuni/failure/v2"
)

const (
	// ErrCratesREADMENotFound represents an error when README is not found
	ErrCratesREADMENotFound ErrorCode = "CratesREADMENotFound"
	// ErrCratesPackageNotFound represents an error when package is not found
	ErrCratesPackageNotFound ErrorCode = "CratesPackageNotFound"
)

// cratesPackageInfo represents the Crates.io package metadata
type cratesPackageInfo struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	DefaultVersion string   `json:"default_version"`
	Homepage       string   `json:"homepage"`
	Repository     string   `json:"repository"`
	Documentation  string   `json:"documentation"`
	Categories     []string `json:"categories"`
	Keywords       []string `json:"keywords"`
}

type cratesVersionInfo struct {
	Num        string `json:"num"`
	ReadmePath string `json:"readme_path"`
	License    string `json:"license"`
}

// fetchCratesIO fetches the README content from crates.io
// Returns the content, related sources, and any error
func fetchCratesIO(pkgPath string) (string, []source.RelatedReference, error) {
	// Get package information from crates.io API
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s?include=default_version", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil, failure.New(ErrCratesPackageNotFound,
			failure.Message("Package not found"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	// Parse JSON response
	var response struct {
		Crate    cratesPackageInfo   `json:"crate"`
		Versions []cratesVersionInfo `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", nil, failure.Wrap(err)
	}

	info := response.Crate

	// Find the default version
	var defaultVersion *cratesVersionInfo
	for _, v := range response.Versions {
		if v.Num == info.DefaultVersion {
			defaultVersion = &v
			break
		}
	}

	if defaultVersion == nil || defaultVersion.ReadmePath == "" {
		return "", nil, failure.New(ErrCratesREADMENotFound,
			failure.Message("README not found in package"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	readmeURL := fmt.Sprintf("https://crates.io%s", defaultVersion.ReadmePath)
	readmeResp, err := http.Get(readmeURL)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}
	defer readmeResp.Body.Close()

	if readmeResp.StatusCode == http.StatusNotFound {
		return "", nil, failure.New(ErrCratesREADMENotFound,
			failure.Message("README not found"),
			failure.Context{
				"pkg": pkgPath,
				"url": readmeURL,
			},
		)
	}

	// Read HTML content
	htmlContent, err := io.ReadAll(readmeResp.Body)
	if err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Convert HTML to Markdown
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(string(htmlContent))
	if err != nil {
		return "", nil, failure.Wrap(err)
	}

	// Format the documentation text
	var sections []string

	// Title and version
	sections = append(sections, fmt.Sprintf("# %s v%s", info.Name, info.DefaultVersion))

	// Description
	if info.Description != "" {
		sections = append(sections, info.Description)
	}

	// Metadata
	var metadata []string
	if defaultVersion.License != "" {
		metadata = append(metadata, fmt.Sprintf("**License:** %s", defaultVersion.License))
	}
	if len(info.Categories) > 0 {
		metadata = append(metadata, fmt.Sprintf("**Categories:** %s", strings.Join(info.Categories, ", ")))
	}
	if len(info.Keywords) > 0 {
		metadata = append(metadata, fmt.Sprintf("**Keywords:** %s", strings.Join(info.Keywords, ", ")))
	}
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
	if info.Repository != "" {
		links = append(links, fmt.Sprintf("**Repository:** %s", info.Repository))
	}
	if len(links) > 0 {
		sections = append(sections, strings.Join(links, "\n"))
	}

	// README content
	sections = append(sections, markdown)

	// Join all sections with double newlines
	doc := strings.Join(sections, "\n\n")

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

	// Add documentation if available
	if info.Documentation != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.TypeDocumentation,
			Path: info.Documentation,
			URL:  info.Documentation,
			From: "api",
		})
	}

	// Add repository if available
	if info.Repository != "" {
		sources = append(sources, source.RelatedReference{
			Type: source.DetectSourceTypeFromURL(info.Repository),
			Path: info.Repository,
			URL:  cleanupURL(info.Repository, source.TypeUnknown),
			From: "api",
		})
	}

	// Extract additional sources from README content
	docSources := extractRelatedSources(doc, pkgPath)
	sources = append(sources, docSources...)

	return doc, sources, nil
}

// Implementation of CratesIO Investigator
type CratesIOInvestigator struct{}

func (i *CratesIOInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from crates.io
	content, relatedSources, err := fetchCratesIO(packagePath)
	if err != nil {
		return source.Data{}, err
	}

	// Generate browser URL
	browserURL, _ := url.Parse(i.GetURL(packagePath))

	return source.Data{
		Contents:       map[string]string{"README.md": content},
		FetchedAt:      time.Now(),
		RelatedSources: relatedSources,
		BrowserURL:     browserURL,
	}, nil
}

func (i *CratesIOInvestigator) GetURL(packagePath string) string {
	// For crates.io, use only the package name without organization
	pkgName := packagePath
	if idx := strings.LastIndex(packagePath, "/"); idx != -1 {
		pkgName = packagePath[idx+1:]
	}
	return fmt.Sprintf("https://crates.io/crates/%s", pkgName)
}

func (i *CratesIOInvestigator) GetSourceType() source.Type {
	return source.TypeCratesIO
}

func (i *CratesIOInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from crates.io URL
	// Example: https://crates.io/crates/package-name -> package-name
	prefix := "https://crates.io/crates/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid Crates.io package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
