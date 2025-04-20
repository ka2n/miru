package source

import (
	"net/url"
	"time"
)

// Data is a structure that represents data retrieved from a specific source
type Data struct {
	Source Reference

	Contents map[string]string

	// Metadata is metadata retrieved from the source
	Metadata map[string]any

	// BrowserURL is URL that accesible with web browser
	BrowserURL *url.URL

	FetchError error

	// FetchedAt is the time when the data was retrieved
	FetchedAt time.Time

	// RelatedSources are sources related to this data
	RelatedSources []RelatedReference
}
