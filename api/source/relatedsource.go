package source

// RelatedReference represents a related documentation source found in content or API responses
type RelatedReference struct {
	// Type represents the source type (e.g., SourceTypeGoPkgDev) or SourceType*
	Type Type

	Path string

	// URL represents the complete URL to the documentation
	URL string

	// From indicates how this source was discovered: "api", or "document"
	From string
}

func (s RelatedReference) ToSourceReference() Reference {
	sourceType := SourceTypeFromString((string(s.Type)))
	var path string
	if s.Path != "" {
		path = s.Path
	} else {
		path = s.URL
	}

	return Reference{
		Type: sourceType,
		Path: path,
	}
}
