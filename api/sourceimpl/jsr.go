package sourceimpl

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ka2n/miru/api/source"
)

// Implementation of JSR Investigator
type JSRInvestigator struct{}

func (i *JSRInvestigator) Fetch(packagePath string) (source.Data, error) {
	// Data retrieval from JSR is currently not implemented
	// Providing a simple implementation as a placeholder
	u := fmt.Sprintf("https://jsr.io/%s", packagePath)
	content := fmt.Sprintf("JavaScript package documentation for %s\nSource: jsr.io", u)

	// Generate browser URL
	browserURL, _ := url.Parse(i.GetURL(packagePath))

	return source.Data{
		Contents: map[string]string{
			"README.md": content,
		},
		FetchedAt:  time.Now(),
		BrowserURL: browserURL,
	}, nil
}

func (i *JSRInvestigator) GetURL(packagePath string) string {
	return fmt.Sprintf("https://jsr.io/%s", packagePath)
}

func (i *JSRInvestigator) GetSourceType() source.Type {
	return source.TypeJSR
}

func (i *JSRInvestigator) PackageFromURL(url string) (string, error) {
	// Extract package path from JSR URL
	// Example: https://jsr.io/package-name -> package-name
	prefix := "https://jsr.io/"
	if strings.HasPrefix(url, prefix) {
		packagePath := url[len(prefix):]
		if packagePath == "" {
			return "", fmt.Errorf("invalid JSR package path: %s", url)
		}
		return packagePath, nil
	}
	return url, nil
}
