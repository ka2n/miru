package api

import "github.com/ka2n/miru/api/source"

// InitialQuery is a structure that represents the initial query created from user input
type InitialQuery struct {
	SourceRef source.Reference

	// ForceUpdate determines whether to forcibly update by ignoring the cache
	ForceUpdate bool
}

// NewInitialQuery creates an initial query from user input
func NewInitialQuery(input UserInput) (InitialQuery, error) {
	// Create initial query using DetectDocSource
	initialQuery, err := detectInitialQuery(input.PackagePath, input.Language)
	if err != nil {
		return InitialQuery{}, err
	}

	// Set ForceUpdate flag
	initialQuery.ForceUpdate = input.ForceUpdate

	return initialQuery, nil
}
