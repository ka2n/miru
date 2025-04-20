package source

import "strings"

// detectSourceTypeFromURL detects the source type from a URL
func DetectSourceTypeFromURL(url string) Type {
	switch {
	case strings.Contains(url, "github.com"):
		return TypeGitHub
	case strings.Contains(url, "gitlab.com"):
		return TypeGitLab
	case strings.Contains(url, "rubygems.org"):
		return TypeRubyGems
	case strings.Contains(url, "npmjs.com"):
		return TypeNPM
	case strings.Contains(url, "jsr.io"):
		return TypeJSR
	case strings.Contains(url, "pkg.go.dev"):
		return TypeGoPkgDev
	case strings.Contains(url, "crates.io"):
		return TypeCratesIO
	case strings.Contains(url, "packagist.org"):
		return TypePackagist
	default:
		return TypeUnknown
	}
}
