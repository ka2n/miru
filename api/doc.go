package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ka2n/miru/api/cache"
	"github.com/morikuni/failure/v2"
)

// ErrorCode defines error types for API operations
type ErrorCode string

const (
	// ErrDocumentationFetch represents errors that occur during documentation fetching
	ErrDocumentationFetch ErrorCode = "DocumentationFetchError"
)

func (c ErrorCode) ErrorCode() string {
	return string(c)
}

// GetDocumentationURL returns the URL for viewing the package documentation in a browser.
// For unsupported sources, it returns the GitHub URL as a fallback.
func GetDocumentationURL(docSource DocSource) (*url.URL, error) {
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
				"source": docSource.Type,
				"pkg":    docSource.PackagePath,
			},
		)
	}
	return u, nil
}

// FetchDocumentation fetches documentation text for the given package from the specified source
func FetchDocumentation(docSource DocSource) (string, error) {
	key := fmt.Sprintf("%s:%s", docSource.Type, docSource.PackagePath)
	return cache.GetOrSet(key, func() (string, error) {
		// For GitHub/GitLab repositories or unknown sources, try to fetch README
		if docSource.Type == SourceTypeGitHub {
			return FetchGitHubReadme(docSource.PackagePath)
		}
		if docSource.Type == SourceTypeGitLab {
			return FetchGitLabReadme(docSource.PackagePath)
		}
		if docSource.Type == SourceTypeUnknown {
			// Try GitHub first, then GitLab if GitHub fails
			if strings.HasPrefix(docSource.PackagePath, "github.com/") {
				return FetchGitHubReadme(docSource.PackagePath)
			}
			if strings.HasPrefix(docSource.PackagePath, "gitlab.com/") {
				return FetchGitLabReadme(docSource.PackagePath)
			}
			return "", failure.New(ErrDocumentationFetch,
				failure.Message("Unknown documentation source"),
				failure.Context{
					"source": docSource.Type,
					"pkg":    docSource.PackagePath,
				},
			)
		}

		// For other sources, return placeholder message for now
		u, err := GetDocumentationURL(docSource)
		if err != nil {
			return "", failure.Wrap(err)
		}

		switch docSource.Type {
		case SourceTypeGoPkgDev:
			return fmt.Sprintf("Go package documentation for %s\nSource: go.pkg.dev", u.String()), nil
		case SourceTypeJSR:
			return fmt.Sprintf("JavaScript package documentation for %s\nSource: jsr.io", u.String()), nil
		case SourceTypeNPM:
			return FetchNPMReadme(docSource.PackagePath)
		case SourceTypeCratesIO:
			return FetchCratesReadme(docSource.PackagePath)
		case SourceTypeRubyGems:
			return FetchRubyGemsReadme(docSource.PackagePath)
		default:
			return "", failure.New(ErrDocumentationFetch,
				failure.Message("Unsupported documentation source"),
				failure.Context{
					"source": docSource.Type,
					"pkg":    docSource.PackagePath,
				},
			)
		}
	})
}
