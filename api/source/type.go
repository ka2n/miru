package source

// Error codes for repository detection
type ErrorCode string

// Type represents the type of documentation source
type Type string

// String returns the string representation of the SourceType
func (s Type) String() string {
	return string(s)
}

// SourceTypeFromString creates a SourceType from a string
func SourceTypeFromString(s string) Type {
	return Type(s)
}

// IsRegistry returns true if the source type is a package registry
func (s Type) IsRegistry() bool {
	switch s {
	case TypeGoPkgDev, TypeJSR, TypeNPM, TypeCratesIO, TypeRubyGems, TypePyPI, TypePackagist:
		return true
	default:
		return false
	}
}

// IsRepository returns true if the source type is a code repository
func (s Type) IsRepository() bool {
	switch s {
	case TypeGitHub, TypeGitLab:
		return true
	default:
		return false
	}
}

func (s Type) IsDocumentation() bool {
	switch s {
	case TypeGoPkgDev, TypeJSR:
		return true
	default:
		return false
	}
}

func (s Type) ContainRepositoryURL() bool {
	switch s {
	case TypeGitHub, TypeGitLab, TypeGoPkgDev:
		return true
	default:
		return false
	}
}

const (
	// Documentation source types
	TypeGoPkgDev      Type = "pkg.go.dev"
	TypeJSR           Type = "jsr.io"
	TypeNPM           Type = "npmjs.com"
	TypeCratesIO      Type = "crates.io"
	TypeRubyGems      Type = "rubygems.org"
	TypePyPI          Type = "pypi.org"
	TypePackagist     Type = "packagist.org"
	TypeGitHub        Type = "github.com"
	TypeGitLab        Type = "gitlab.com"
	TypeDocumentation Type = "documentation"
	TypeHomepage      Type = "homepage"
	TypeUnknown       Type = ""
)
