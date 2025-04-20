package sourceimpl

import (
	"fmt"
	"net/url"
	"time"

	"github.com/ka2n/miru/api/source"
)

// Implementation of Website Investigator
type WebsiteInvestigator struct {
	Type source.Type
}

func (i *WebsiteInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Data retrieval from Website is currently not implemented

	// Generate browser URL
	browserURL, _ := url.Parse(i.GetURL(packagePath))

	return source.Data{
		Contents:   map[string]string{},
		FetchedAt:  time.Now(),
		BrowserURL: browserURL,
	}, nil
}

func (i *WebsiteInvestigator) GetURL(packagePath string) string {
	return packagePath
}

func (i *WebsiteInvestigator) GetSourceType() source.Type {
	return i.Type
}

func (i *WebsiteInvestigator) PackageFromURL(url string) (string, error) {
	// Website URL is treated as the package path directly
	if url == "" {
		return "", fmt.Errorf("invalid website URL")
	}
	return url, nil
}
