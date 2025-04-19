package api

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/samber/lo"
)

// Error codes for repository detection
type ErrCode string

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
	case SourceTypeGoPkgDev, SourceTypeJSR, SourceTypeNPM, SourceTypeCratesIO, SourceTypeRubyGems, SourceTypePyPI, SourceTypePackagist:
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

func (s SourceType) IsDocumentation() bool {
	switch s {
	case SourceTypeGoPkgDev, SourceTypeJSR:
		return true
	default:
		return false
	}
}

func (s SourceType) ContainRepositoryURL() bool {
	switch s {
	case SourceTypeGitHub, SourceTypeGitLab, SourceTypeGoPkgDev:
		return true
	default:
		return false
	}
}

// DocSource represents a documentation source with related sources and homepage
type DocSource struct {
	// Type represents the documentation source type (e.g., "pkg.go.dev", "npm", "jsr")
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
	// From indicates how this source was discovered: "api", or "document"
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
	SourceTypeGoPkgDev  SourceType = "pkg.go.dev"
	SourceTypeJSR       SourceType = "jsr.io"
	SourceTypeNPM       SourceType = "npmjs.com"
	SourceTypeCratesIO  SourceType = "crates.io"
	SourceTypeRubyGems  SourceType = "rubygems.org"
	SourceTypePyPI      SourceType = "pypi.org"
	SourceTypePackagist SourceType = "packagist.org"
	SourceTypeGitHub    SourceType = "github.com"
	SourceTypeGitLab    SourceType = "gitlab.com"
	SourceTypeUnknown   SourceType = ""
)

// GetLanguageAliases returns a map of language aliases to their documentation source types
func GetLanguageAliases() map[string]SourceType {
	return languageAliases
}

// languageAliases maps language aliases to their canonical language name
var languageAliases = map[string]SourceType{
	// go
	"go":     SourceTypeGoPkgDev,
	"golang": SourceTypeGoPkgDev,

	// JavaScript, TypeScript
	"js":         SourceTypeNPM,
	"javascript": SourceTypeNPM,
	"npm":        SourceTypeNPM,
	"node":       SourceTypeNPM,
	"nodejs":     SourceTypeNPM,
	"ts":         SourceTypeNPM,
	"tsx":        SourceTypeNPM,
	"typescript": SourceTypeNPM,
	"jsr":        SourceTypeJSR,

	// rust
	"rust":   SourceTypeCratesIO,
	"rs":     SourceTypeCratesIO,
	"crates": SourceTypeCratesIO,

	// ruby
	"ruby": SourceTypeRubyGems,
	"rb":   SourceTypeRubyGems,
	"gem":  SourceTypeRubyGems,

	// python
	"python": SourceTypePyPI,
	"py":     SourceTypePyPI,
	"pypi":   SourceTypePyPI,
	"pip":    SourceTypePyPI,

	// php
	"php":       SourceTypePackagist,
	"packagist": SourceTypePackagist,
	"composer":  SourceTypePackagist,
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

	// Check for Python package names (from pypi.org)
	if result.Type == SourceTypePyPI {
		return DocSource{
			Type:        SourceTypePyPI,
			PackagePath: pkgPath,
		}
	}

	// Check for PHP package names (from packagist.org)
	if result.Type == SourceTypePackagist {
		return DocSource{
			Type:        SourceTypePackagist,
			PackagePath: pkgPath,
		}
	}

	// Check for known Go package domains
	if result.Type == SourceTypeGoPkgDev ||
		strings.HasPrefix(pkgPath, "pkg.go.dev/") {
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
			if strings.HasPrefix(repoName, "go-") ||
				strings.HasSuffix(repoName, "-go") ||
				strings.Contains(repoName, ".go") {
				return DocSource{
					Type:        SourceTypeGoPkgDev,
					PackagePath: pkgPath,
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
	var candidate *url.URL

	sort.Slice(docSource.RelatedSources, func(i, j int) bool {
		return len(docSource.RelatedSources[i].URL) < len(docSource.RelatedSources[j].URL)
	})

	for _, source := range docSource.RelatedSources {
		t := SourceTypeFromString(source.Type.String())
		if source.Type == RelatedSourceTypeDocumentation || t.IsDocumentation() {
			u, err := url.Parse(source.URL)
			if err == nil {
				candidate = u
				break
			}
		}
	}

	if docSource.Type.IsDocumentation() {
		u := docSource.GetURL()
		if candidate == nil {
			candidate = u
		} else if len(u.String()) < len(candidate.String()) {
			candidate = u
		}
	}

	if candidate == nil {
		return nil, nil
	}
	return candidate, nil
}

// GetRepository returns the repository URL for the package.
// It first checks RelatedSources for a repository URL, then generates one from the package path.
func (docSource DocSource) GetRepository() (*url.URL, error) {
	if docSource.Type.IsRepository() {
		return docSource.GetURL(), nil
	}

	// Check RelatedSources
	for _, source := range docSource.RelatedSources {
		t := SourceTypeFromString(source.Type.String())
		if t.IsRepository() {
			u, err := url.Parse(source.URL)
			if err == nil {
				return u, nil
			}
		}
	}

	if docSource.Type.ContainRepositoryURL() {
		u, err := url.Parse("https://" + docSource.PackagePath)
		if err == nil {
			return u, nil
		}
	}

	return nil, nil
}

// GetRegistry returns the package registry URL.
// It first checks RelatedSources for a registry URL, then generates one based on the source type.
func (docSource DocSource) GetRegistry() (*url.URL, error) {
	if docSource.Type.IsRegistry() {
		return docSource.GetURL(), nil
	}

	// Check RelatedSources
	for _, source := range docSource.RelatedSources {
		t := SourceTypeFromString(source.Type.String())
		if t.IsRegistry() {
			return url.Parse(source.URL)
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
func (docSource DocSource) GetURL() *url.URL {
	var rawURL string

	switch docSource.Type {
	case SourceTypeGoPkgDev:
		rawURL = fmt.Sprintf("https://pkg.go.dev/%s", docSource.PackagePath)
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
	case SourceTypePyPI:
		// For PyPI, use only the package name without organization
		pkgName := docSource.PackagePath
		if idx := strings.LastIndex(docSource.PackagePath, "/"); idx != -1 {
			pkgName = docSource.PackagePath[idx+1:]
		}
		rawURL = fmt.Sprintf("https://pypi.org/project/%s", pkgName)
	case SourceTypePackagist:
		// For Packagist, use the full package path (vendor/package)
		rawURL = fmt.Sprintf("https://packagist.org/packages/%s", docSource.PackagePath)
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

	u := lo.Must(url.Parse(rawURL))
	return u
}
