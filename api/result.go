package api

import (
	"net/url"

	"github.com/ka2n/miru/api/source"
)

// Result is a structure that represents the investigation result
type Result struct {
	README string

	InitialQueryURL  *url.URL
	InitialQueryType source.Type
	Links            []Link
}

type Link struct {
	Type source.Type
	URL  *url.URL
}

// CreateResult generates a result from investigation data
func CreateResult(inv *Investigation) Result {
	var result Result
	result.Links = make([]Link, 0, len(inv.CollectedData))

	// Check if README content is available in the collected data
	for _, data := range inv.CollectedData {
		// Pickup most longest README content
		readme := data.Contents["README.md"]
		if len(readme) > len(result.README) {
			result.README = readme
		}

		result.Links = append(result.Links, Link{
			Type: data.Source.Type,
			URL:  data.BrowserURL,
		})
	}

	// Get data from the source type of the initial query
	if data, ok := inv.CollectedData[inv.Query.SourceRef.Type]; ok {
		result.InitialQueryURL = data.BrowserURL
		result.InitialQueryType = inv.Query.SourceRef.Type
	}

	return result
}

func (r Result) GetHomepage() *url.URL {
	for _, link := range r.Links {
		if link.Type == source.TypeHomepage {
			return link.URL
		}
	}
	return nil
}

func (r Result) GetDocumentation() *url.URL {
	for _, link := range r.Links {
		if link.Type.IsDocumentation() {
			return link.URL
		}
	}
	return nil
}

func (r Result) GetRegistry() *url.URL {
	for _, link := range r.Links {
		if link.Type.IsRegistry() {
			return link.URL
		}
	}
	return nil
}

func (r Result) GetRepository() *url.URL {
	for _, link := range r.Links {
		if link.Type.IsRepository() {
			return link.URL
		}
	}
	return nil
}
