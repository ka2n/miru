package api

// UserInput is a structure that represents input from the user
type UserInput struct {
	// PackagePath is the path of the package
	PackagePath string
	// Language is the explicitly specified language
	Language string
	// ForceUpdate determines whether to forcibly update by ignoring the cache
	ForceUpdate bool
	// Other user-specified options
}
