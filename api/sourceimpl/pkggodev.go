package sourceimpl

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ka2n/miru/api/source"
	"github.com/morikuni/failure/v2"
	"golang.org/x/net/html"
)

const (
	ErrPkgGoDevREADMENotFound ErrCode = "ErrPkgGoDevREADMENotFound"
)

// fetchPkgGoDev fetches the README file from pkg.go.dev or the source repository
func fetchPkgGoDev(pkgPath string) (string, []source.RelatedReference, error) {
	// https://pkg.go.dev/cmd/go#hdr-Remote_import_paths
	if strings.Contains(pkgPath, "github.com/") {
		return fetchGitHub(pkgPath)
	} else if strings.Contains(pkgPath, "gitlab.com/") {
		return fetchGitlab(pkgPath)
	}

	repo, home, err := detectGoMetadata(pkgPath, nil)
	if repo == nil {
		return "", nil, err
	}

	// Get Readme content from the repository
	var sourceRepoURL *url.URL // URL of the source repository not git URL
	if repo.Hostname() == "github.com" || repo.Hostname() == "gitlab.com" {
		sourceRepoURL = repo
	} else if home != nil && (home.Hostname() == "github.com" || home.Hostname() == "gitlab.com") {
		sourceRepoURL = home
	}
	if sourceRepoURL != nil {
		var content string
		var sources []source.RelatedReference
		var err error

		if strings.Contains(sourceRepoURL.String(), "github.com") {
			content, sources, err = fetchGitHub(sourceRepoURL.String())
		} else if strings.Contains(sourceRepoURL.String(), "gitlab.com") {
			content, sources, err = fetchGitlab(sourceRepoURL.String())
		} else {
			panic("Unsupported source repository URL: " + sourceRepoURL.String())
		}

		if err != nil {
			return "", nil, err
		}

		if home != nil {
			sources = append(sources, source.RelatedReference{
				Type: source.TypeHomepage,
				URL:  home.String(),
				From: "api",
			})
		}

		sources = append(sources, source.RelatedReference{
			Type: source.DetectSourceTypeFromURL(sourceRepoURL.String()),
			URL:  sourceRepoURL.String(),
			From: "api",
		})

		return content, sources, nil
	}

	return "", nil, failure.New(ErrPkgGoDevREADMENotFound,
		failure.Message("Package not found"),
		failure.Context{
			"pkg": pkgPath,
		},
	)
}

var (
	// ErrRepositoryNotFound represents errors when repository information cannot be found
	ErrRepositoryNotFound ErrCode = "RepositoryNotFound"
	// ErrInvalidMetaTag represents errors when meta tag is invalid or missing
	ErrInvalidMetaTag ErrCode = "InvalidMetaTag"
)

// GoMetadata contains metadata extracted from go-import and go-source meta tags
type GoMetadata struct {
	Repository *url.URL // Repository URL from go-import meta tag
	Homepage   *url.URL // Homepage URL from go-source meta tag
}

// detectGoMetadata attempts to detect repository and homepage URLs from go-import and go-source meta tags
// by making an HTTP request to the package path with ?go-get=1 parameter.
// It returns repository URL, homepage URL if found, or an error if the request fails or required meta tags are not present.
func detectGoMetadata(pkgPath string, client *http.Client) (*url.URL, *url.URL, error) {
	if client == nil {
		client = http.DefaultClient
	}
	// Ensure package path starts with https://
	if !strings.HasPrefix(pkgPath, "https://") {
		pkgPath = "https://" + pkgPath
	}

	// Parse and add go-get=1 parameter
	u, err := url.Parse(pkgPath)
	if err != nil {
		return nil, nil, failure.Wrap(err, failure.WithCode(ErrRepositoryNotFound),
			failure.Message("Failed to parse package path"),
			failure.Context{"path": pkgPath})
	}
	q := u.Query()
	q.Set("go-get", "1")
	u.RawQuery = q.Encode()

	// Make HTTP request
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, nil, failure.Wrap(err, failure.WithCode(ErrRepositoryNotFound),
			failure.Message("Failed to fetch go-import meta tag"),
			failure.Context{"url": u.String()})
	}
	defer resp.Body.Close()

	// Parse HTML and find meta tag
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, nil, failure.Wrap(err, failure.WithCode(ErrInvalidMetaTag),
			failure.Message("Failed to parse HTML response"),
			failure.Context{"url": u.String()})
	}

	// Find go-import and go-source meta tags
	var importContent, sourceContent string
	var findMeta func(*html.Node)
	findMeta = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if name == "go-import" && content != "" {
				importContent = content
			}
			if name == "go-source" && content != "" {
				sourceContent = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findMeta(c)
		}
	}
	findMeta(doc)

	if importContent == "" {
		return nil, nil, failure.New(ErrInvalidMetaTag,
			failure.Message("No go-import meta tag found"),
			failure.Context{"url": u.String()})
	}

	// Parse go-import content (format: "prefix vcs repo")
	importParts := strings.Fields(importContent)
	if len(importParts) != 3 {
		return nil, nil, failure.New(ErrInvalidMetaTag,
			failure.Message("Invalid go-import meta tag format"),
			failure.Context{
				"url":     u.String(),
				"content": importContent,
			})
	}

	var repoURL *url.URL
	var homepageURL *url.URL

	// Parse repository URL
	repoURL, err = url.Parse(importParts[2])
	if err != nil {
		return nil, nil, failure.Wrap(err, failure.WithCode(ErrInvalidMetaTag),
			failure.Message("Invalid repository URL in meta tag"),
			failure.Context{
				"url":     u.String(),
				"content": importContent,
			})
	}

	// Parse go-source content if available (format: "prefix homepage dir file")
	if sourceContent != "" {
		sourceParts := strings.Fields(sourceContent)
		if len(sourceParts) >= 2 {
			homepageURL, err = url.Parse(sourceParts[1])
		}
	}

	if err != nil {
		return repoURL, homepageURL, failure.Wrap(err, failure.WithCode(ErrInvalidMetaTag),
			failure.Message("Invalid homepage URL in meta tag"),
			failure.Context{
				"url":     u.String(),
				"content": sourceContent,
			})
	}

	return repoURL, homepageURL, nil
}

// Implementation of GoPkgDev Investigator
type GoPkgDevInvestigator struct{}

func (i *GoPkgDevInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Process to retrieve data from pkg.go.dev
	content, RelatedSources, err := fetchPkgGoDev(packagePath)
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

func (i *GoPkgDevInvestigator) GetURL(packagePath string) string {
	return fmt.Sprintf("https://pkg.go.dev/%s", packagePath)
}

func (i *GoPkgDevInvestigator) GetSourceType() source.Type {
	return source.TypeGoPkgDev
}

func (i *GoPkgDevInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from pkg.go.dev URL
	// Example: https://pkg.go.dev/github.com/username/repo -> github.com/username/repo
	prefix := "https://pkg.go.dev/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", failure.New(ErrInvalidPackagePath,
				failure.Message("Invalid Go package path"),
				failure.Context{"url": url},
			)
		}
		return packagePath, nil
	}
	return url, nil
}
