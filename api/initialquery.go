package api

import "github.com/ka2n/miru/api/source"

// InitialQuery is a structure that represents the initial query created from user input
type InitialQuery struct {
	SourceRef source.Reference

	// ForceUpdate determines whether to forcibly update by ignoring the cache
	ForceUpdate bool
}

// NewInitialQuery creates an initial query from user input
func NewInitialQuery(input UserInput) InitialQuery {
	// Create initial query using DetectDocSource
	initialQuery := DetectInitialQuery(input.PackagePath, input.Language)

	// Set ForceUpdate flag
	initialQuery.ForceUpdate = input.ForceUpdate

	return initialQuery
}
