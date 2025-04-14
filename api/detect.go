package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/morikuni/failure/v2"
)

// SourceType represents the type of documentation source
type SourceType string

// String returns the string representation of the SourceType
func (s SourceType) String() string {
	return string(s)
}

// SourceTypeFromString creates a SourceType from a string
func SourceTypeFromString(s string) SourceType {
	return SourceType(s)
}

// IsRegistry returns true if the source type is a package registry
func (s SourceType) IsRegistry() bool {
	switch s {
	case SourceTypeGoPkgDev, SourceTypeJSR, SourceTypeNPM, SourceTypeCratesIO, SourceTypeRubyGems:
		return true
	default:
		return false
	}
}

// IsRepository returns true if the source type is a code repository
func (s SourceType) IsRepository() bool {
	switch s {
	case SourceTypeGitHub, SourceTypeGitLab:
		return true
	default:
		return false
	}
}

// DocSource represents a documentation source with related sources and homepage
type DocSource struct {
	// Type represents the documentation source type (e.g., "go.pkg.dev", "npm", "jsr")
	Type SourceType
	// PackagePath represents the processed package path for the documentation source
	PackagePath string
	// RelatedSources contains links to related documentation sources
	RelatedSources []RelatedSource
	// Homepage represents the package's homepage URL
	Homepage string
}

// RelatedSource represents a related documentation source found in content or API responses
type RelatedSource struct {
	// Type represents the source type (e.g., SourceTypeGoPkgDev) or RelatedSourceType*
	Type RelatedSourceType
	// URL represents the complete URL to the documentation
	URL string
	// From indicates how this source was discovered: "api", "document_link", or "document_command"
	From string
}

type RelatedSourceType string

// String returns the string representation of the RelatedSourceType
func (s RelatedSourceType) String() string {
	return string(s)
}

const (
	RelatedSourceTypeDocumentation RelatedSourceType = "documentation"
	RelatedSourceTypeHomepage      RelatedSourceType = "homepage"
)

// RelatedSourceTypeFromString creates a SourceType from a string
func RelatedSourceTypeFromString(s string) RelatedSourceType {
	return RelatedSourceType(s)
}

const (
	// Documentation source types
	SourceTypeGoPkgDev SourceType = "go.pkg.dev"
	SourceTypeJSR      SourceType = "jsr.io"
	SourceTypeNPM      SourceType = "npmjs.com"
	SourceTypeCratesIO SourceType = "crates.io"
	SourceTypeRubyGems SourceType = "rubygems.org"
	SourceTypeGitHub   SourceType = "github.com"
	SourceTypeGitLab   SourceType = "gitlab.com"
	SourceTypeUnknown  SourceType = ""
)

// GetLanguageAliases returns a map of language aliases to their documentation source types
func GetLanguageAliases() map[string]SourceType {
	return languageAliases
}

// languageAliases maps language aliases to their canonical language name
var languageAliases = map[string]SourceType{
	"go":         SourceTypeGoPkgDev,
	"golang":     SourceTypeGoPkgDev,
	"js":         SourceTypeNPM,
	"javascript": SourceTypeNPM,
	"npm":        SourceTypeNPM,
	"node":       SourceTypeNPM,
	"nodejs":     SourceTypeNPM,
	"jsr":        SourceTypeJSR,
	"ts":         SourceTypeNPM,
	"tsx":        SourceTypeNPM,
	"typescript": SourceTypeNPM,
	"rust":       SourceTypeCratesIO,
	"rs":         SourceTypeCratesIO,
	"ruby":       SourceTypeRubyGems,
	"rb":         SourceTypeRubyGems,
	"gem":        SourceTypeRubyGems,
}

// DetectDocSource attempts to detect the documentation source from a package path
// If explicitLang is provided, it will be used as an explicit language hint
func DetectDocSource(pkgPath string, explicitLang string) DocSource {
	var result DocSource

	// If explicit language is provided, try to resolve it
	if explicitLang != "" {
		if source, ok := languageAliases[explicitLang]; ok {
			result.Type = source
		}
	}

	// Check for JavaScript package prefixes
	if result.Type == SourceTypeJSR {
		return DocSource{
			Type:        SourceTypeJSR,
			PackagePath: pkgPath,
		}
	}
	if result.Type == SourceTypeNPM {
		return DocSource{
			Type:        SourceTypeNPM,
			PackagePath: pkgPath,
		}
	}

	// Check for Rust package names (from crates.io)
	if result.Type == SourceTypeCratesIO {
		return DocSource{
			Type:        SourceTypeCratesIO,
			PackagePath: pkgPath,
		}
	}

	// Check for Ruby package names (from rubygems.org)
	if result.Type == SourceTypeRubyGems {
		return DocSource{
			Type:        SourceTypeRubyGems,
			PackagePath: pkgPath,
		}
	}

	// Check for known Go package domains
	if result.Type == SourceTypeGoPkgDev || strings.HasPrefix(pkgPath, "golang.org/") ||
		strings.HasPrefix(pkgPath, "go.dev/") ||
		strings.HasPrefix(pkgPath, "pkg.go.dev/") ||
		strings.HasPrefix(pkgPath, "go.pkg.dev/") {
		return DocSource{
			Type:        SourceTypeGoPkgDev,
			PackagePath: pkgPath,
		}
	}

	// For GitHub/GitLab repositories, try to detect from the path
	if result.Type == SourceTypeGitHub || strings.HasPrefix(pkgPath, "github.com/") ||
		result.Type == SourceTypeGitLab || strings.HasPrefix(pkgPath, "gitlab.com/") {
		parts := strings.Split(pkgPath, "/")
		if len(parts) >= 3 {
			// Check if the repository name contains language hints
			repoName := strings.ToLower(parts[2])
			trimmedPath := pkgPath
			if strings.HasPrefix(pkgPath, "github.com/") {
				trimmedPath = strings.TrimPrefix(pkgPath, "github.com/")
			} else if strings.HasPrefix(pkgPath, "gitlab.com/") {
				trimmedPath = strings.TrimPrefix(pkgPath, "gitlab.com/")
			}

			if strings.HasPrefix(repoName, "go-") ||
				strings.HasSuffix(repoName, "-go") ||
				strings.Contains(repoName, ".go") {
				return DocSource{
					Type:        SourceTypeGoPkgDev,
					PackagePath: trimmedPath,
				}
			}
			if strings.HasPrefix(repoName, "rust-") ||
				strings.HasSuffix(repoName, "-rust") ||
				strings.HasPrefix(repoName, "rs-") ||
				strings.HasSuffix(repoName, "-rs") {
				return DocSource{
					Type:        SourceTypeCratesIO,
					PackagePath: trimmedPath,
				}
			}
			if strings.HasPrefix(repoName, "ruby-") ||
				strings.HasSuffix(repoName, "-ruby") ||
				strings.HasPrefix(repoName, "rb-") ||
				strings.HasSuffix(repoName, "-rb") {
				return DocSource{
					Type:        SourceTypeRubyGems,
					PackagePath: trimmedPath,
				}
			}
		}
	}

	// Default to GitHub/GitLab for unknown languages or repository paths
	if strings.HasPrefix(pkgPath, "github.com/") {
		return DocSource{
			Type:        SourceTypeGitHub,
			PackagePath: strings.TrimPrefix(pkgPath, "github.com/"),
		}
	}
	if strings.HasPrefix(pkgPath, "gitlab.com/") {
		return DocSource{
			Type:        SourceTypeGitLab,
			PackagePath: strings.TrimPrefix(pkgPath, "gitlab.com/"),
		}
	}

	return DocSource{
		Type:        SourceTypeUnknown,
		PackagePath: pkgPath,
	}
}

// GetHomepage returns the homepage URL for the package.
// It first checks RelatedSources for a homepage URL, then falls back to the Homepage field,
// and finally uses the registry URL as a last resort.
func (docSource DocSource) GetHomepage() (*url.URL, error) {
	// Use Homepage field if available
	if docSource.Homepage != "" {
		u, err := url.Parse(docSource.Homepage)
		if err == nil {
			return u, nil
		}
	}

	// Check RelatedSources
	for _, source := range docSource.RelatedSources {
		if source.Type == RelatedSourceTypeHomepage {
			u, err := url.Parse(source.URL)
			if err == nil {
				return u, nil
			}
		}
	}

	return nil, nil
}

func (docSource DocSource) GetDocument() (*url.URL, error) {
	for _, source := range docSource.RelatedSources {
		if source.Type == RelatedSourceTypeDocumentation {
			u, err := url.Parse(source.URL)
			if err == nil {
				return u, nil
			}
		}
	}

	return nil, nil
}

// GetRepository returns the repository URL for the package.
// It first checks RelatedSources for a repository URL, then generates one from the package path.
func (docSource DocSource) GetRepository() (*url.URL, error) {
	// Generate repository URL from package path
	var rawURL string
	if strings.HasPrefix(docSource.PackagePath, "gitlab.com/") {
		rawURL = fmt.Sprintf("https://gitlab.com/%s", docSource.PackagePath)
	} else if strings.HasPrefix(docSource.PackagePath, "github.com/") {
		rawURL = fmt.Sprintf("https://github.com/%s", docSource.PackagePath)
	}

	if rawURL != "" {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, failure.Wrap(err,
				failure.Context{
					"source": docSource.Type.String(),
					"pkg":    docSource.PackagePath,
				},
			)
		}
		return u, nil
	}

	// Check RelatedSources
	for _, source := range docSource.RelatedSources {
		t := SourceTypeFromString(source.Type.String())
		if t == SourceTypeGitHub || t == SourceTypeGitLab {
			cleanURL := cleanupRepositoryURL(source.URL)
			u, err := url.Parse(cleanURL)
			if err == nil {
				return u, nil
			}
		}
	}

	return nil, nil
}

// GetRegistry returns the package registry URL.
// It first checks RelatedSources for a registry URL, then generates one based on the source type.
func (docSource DocSource) GetRegistry() (*url.URL, error) {
	// Generate registry URL based on source type
	var rawURL string
	switch docSource.Type {
	case SourceTypeGoPkgDev:
		rawURL = fmt.Sprintf("https://go.pkg.dev/%s", docSource.PackagePath)
	case SourceTypeJSR:
		rawURL = fmt.Sprintf("https://jsr.io/%s", docSource.PackagePath)
	case SourceTypeNPM:
		pkgName := strings.ReplaceAll(docSource.PackagePath, "/", "-")
		rawURL = fmt.Sprintf("https://www.npmjs.com/package/%s", pkgName)
	case SourceTypeCratesIO:
		pkgName := docSource.PackagePath
		if idx := strings.LastIndex(docSource.PackagePath, "/"); idx != -1 {
			pkgName = docSource.PackagePath[idx+1:]
		}
		rawURL = fmt.Sprintf("https://crates.io/crates/%s", pkgName)
	case SourceTypeRubyGems:
		pkgName := docSource.PackagePath
		if idx := strings.LastIndex(docSource.PackagePath, "/"); idx != -1 {
			pkgName = docSource.PackagePath[idx+1:]
		}
		rawURL = fmt.Sprintf("https://rubygems.org/gems/%s", pkgName)
	}

	if rawURL != "" {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, failure.Wrap(err,
				failure.Context{
					"source": docSource.Type.String(),
					"pkg":    docSource.PackagePath,
				},
			)
		}
		return u, nil
	}

	// Check RelatedSources
	for _, source := range docSource.RelatedSources {
		t := SourceTypeFromString(source.Type.String())
		switch t {
		case SourceTypeNPM, SourceTypeCratesIO, SourceTypeRubyGems, SourceTypeGoPkgDev, SourceTypeJSR:
			u, err := url.Parse(source.URL)
			if err == nil {
				return u, nil
			}
		}
	}

	return nil, nil
}

// OtherLinks returns a list of related URLs that are not homepage, repository, or registry URLs.
func (docSource DocSource) OtherLinks() ([]RelatedSource, error) {
	var links []RelatedSource
	seen := make(map[string]bool)

	// Get main URLs to exclude them from other links
	homepage, _ := docSource.GetHomepage()
	repository, _ := docSource.GetRepository()
	registry, _ := docSource.GetRegistry()
	docs, _ := docSource.GetDocument()

	// Add URLs to seen map
	if homepage != nil {
		seen[homepage.String()] = true
	}
	if repository != nil {
		seen[repository.String()] = true
	}
	if registry != nil {
		seen[registry.String()] = true
	}
	if docs != nil {
		seen[docs.String()] = true
	}

	// Process RelatedSources
	for _, source := range docSource.RelatedSources {
		if !seen[source.URL] {
			links = append(links, source)
			seen[source.URL] = true
		}
	}

	return links, nil
}

// GetURL returns the URL for viewing the package documentation in a browser.
// For unsupported sources, it returns the GitHub URL as a fallback.
func (docSource DocSource) GetURL() (*url.URL, error) {
	var rawURL string

	switch docSource.Type {
	case SourceTypeGoPkgDev:
		rawURL = fmt.Sprintf("https://go.pkg.dev/%s", docSource.PackagePath)
	case SourceTypeJSR:
		rawURL = fmt.Sprintf("https://jsr.io/%s", docSource.PackagePath)
	case SourceTypeNPM:
		// For npm packages, convert path separators to package name format
		pkgName := strings.ReplaceAll(docSource.PackagePath, "/", "-")
		rawURL = fmt.Sprintf("https://www.npmjs.com/package/%s", pkgName)
	case SourceTypeCratesIO:
		// For crates.io, use only the package name without organization
		pkgName := docSource.PackagePath
		if idx := strings.LastIndex(docSource.PackagePath, "/"); idx != -1 {
			pkgName = docSource.PackagePath[idx+1:]
		}
		rawURL = fmt.Sprintf("https://crates.io/crates/%s", pkgName)
	case SourceTypeRubyGems:
		// For RubyGems, use only the package name without organization
		pkgName := docSource.PackagePath
		if idx := strings.LastIndex(docSource.PackagePath, "/"); idx != -1 {
			pkgName = docSource.PackagePath[idx+1:]
		}
		rawURL = fmt.Sprintf("https://rubygems.org/gems/%s", pkgName)
	case SourceTypeGitLab:
		rawURL = fmt.Sprintf("https://gitlab.com/%s", docSource.PackagePath)
	default:
		// Return GitHub/GitLab URL based on path prefix
		if strings.HasPrefix(docSource.PackagePath, "gitlab.com/") {
			rawURL = fmt.Sprintf("https://gitlab.com/%s", docSource.PackagePath)
		} else {
			rawURL = fmt.Sprintf("https://github.com/%s", docSource.PackagePath)
		}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, failure.Wrap(err,
			failure.Context{
				"source": docSource.Type.String(),
				"pkg":    docSource.PackagePath,
			},
		)
	}
	return u, nil
}
