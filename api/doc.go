package api

import (
	"fmt"
	"net/url"
	"strings"

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
	default:
		// Return GitHub URL as fallback for unsupported sources
		rawURL = fmt.Sprintf("https://github.com/%s", docSource.PackagePath)
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
	// For GitHub repositories or unknown sources, try to fetch README
	if docSource.Type == SourceTypeGitHub || docSource.Type == SourceTypeUnknown {
		return FetchGitHubReadme(docSource.PackagePath)
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
		return fmt.Sprintf("JavaScript package documentation for %s\nSource: npmjs.com", u.String()), nil
	case SourceTypeCratesIO:
		return fmt.Sprintf("Rust package documentation for %s\nSource: crates.io", u.String()), nil
	default:
		return "", failure.New(ErrDocumentationFetch,
			failure.Message("Unsupported documentation source"),
			failure.Context{
				"source": docSource.Type,
				"pkg":    docSource.PackagePath,
			},
		)
	}
}
