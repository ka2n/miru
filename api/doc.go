package api

import (
	"fmt"
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

// DocumentationResult represents the result of a documentation fetch operation
type DocumentationResult struct {
	Content string
	Source  *DocSource
}

// FetchDocumentation fetches documentation text for the given package from the specified source
func FetchDocumentation(docSource *DocSource, forceUpdate bool) (string, error) {
	key := fmt.Sprintf("%s:%s", docSource.Type, docSource.PackagePath)
	c := cache.New[DocumentationResult]("docs")

	result, err := c.GetOrSet(key, func() (DocumentationResult, error) {
		var content string
		var source *DocSource
		var err error

		// For GitHub/GitLab repositories or unknown sources, try to fetch README
		switch docSource.Type {
		case SourceTypeGitHub:
			content, source, err = FetchGitHubReadme(docSource.PackagePath)
		case SourceTypeGitLab:
			content, source, err = FetchGitLabReadme(docSource.PackagePath)
		case SourceTypeUnknown:
			// Try GitHub first, then GitLab if GitHub fails
			if strings.HasPrefix(docSource.PackagePath, "github.com/") {
				content, source, err = FetchGitHubReadme(docSource.PackagePath)
			} else if strings.HasPrefix(docSource.PackagePath, "gitlab.com/") {
				content, source, err = FetchGitLabReadme(docSource.PackagePath)
			} else {
				return DocumentationResult{}, failure.New(ErrDocumentationFetch,
					failure.Message("Unknown documentation source"),
					failure.Context{
						"source": docSource.Type.String(),
						"pkg":    docSource.PackagePath,
					},
				)
			}
		default:
			// For other sources, return placeholder message for now
			u := docSource.GetURL()

			switch docSource.Type {
			case SourceTypeGoPkgDev:
				content = fmt.Sprintf("Go package documentation for %s\nSource: go.pkg.dev", u.String())
				source = docSource
			case SourceTypeJSR:
				content = fmt.Sprintf("JavaScript package documentation for %s\nSource: jsr.io", u.String())
				source = docSource
			case SourceTypeNPM:
				content, source, err = FetchNPMReadme(docSource.PackagePath)
			case SourceTypeCratesIO:
				content, source, err = FetchCratesReadme(docSource.PackagePath)
			case SourceTypeRubyGems:
				content, source, err = FetchRubyGemsReadme(docSource.PackagePath)
			default:
				return DocumentationResult{}, failure.New(ErrDocumentationFetch,
					failure.Message("Unsupported documentation source"),
					failure.Context{
						"source": docSource.Type.String(),
						"pkg":    docSource.PackagePath,
					},
				)
			}
		}

		if err != nil {
			return DocumentationResult{}, err
		}

		return DocumentationResult{
			Content: content,
			Source:  source,
		}, nil
	}, forceUpdate)

	if err != nil {
		return "", err
	}

	// Update the docSource with related sources if available
	if result.Source != nil {
		docSource.RelatedSources = result.Source.RelatedSources
		docSource.Homepage = result.Source.Homepage
	}

	return result.Content, nil
}
