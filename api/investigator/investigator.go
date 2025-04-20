package investigator

import "github.com/ka2n/miru/api/source"

// SourceInvestigator is an interface for retrieving data from sources
type SourceInvestigator interface {
	// Fetch retrieves data from the source
	Fetch(packagePath string) (source.Data, error)

	// GetURL generates a URL for the source
	GetURL(packagePath string) string

	// PackageFromURL extracts a package path from a URL
	PackageFromURL(url string) (string, error)

	// GetSourceType returns the source type
	GetSourceType() source.Type
}
