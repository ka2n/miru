package api

import (
	"strings"
)

// DocSource represents a documentation source
type DocSource struct {
	// Type represents the documentation source type (e.g., "go.pkg.dev", "npm", "jsr")
	Type string
	// PackagePath represents the processed package path for the documentation source
	PackagePath string
}

const (
	// Documentation source types
	SourceTypeGoPkgDev = "go.pkg.dev"
	SourceTypeJSR      = "jsr.io"
	SourceTypeNPM      = "npmjs.com"
	SourceTypeCratesIO = "crates.io"
	SourceTypeRubyGems = "rubygems.org"
	SourceTypeGitHub   = "github.com"
	SourceTypeGitLab   = "gitlab.com"
	SourceTypeUnknown  = ""
)

// GetLanguageAliases returns a map of language aliases to their documentation source types
func GetLanguageAliases() map[string]string {
	return languageAliases
}

// languageAliases maps language aliases to their canonical language name
var languageAliases = map[string]string{
	"go":         SourceTypeGoPkgDev,
	"golang":     SourceTypeGoPkgDev,
	"js":         SourceTypeNPM,
	"javascript": SourceTypeNPM,
	"npm":        SourceTypeNPM,
	"node":       SourceTypeNPM,
	"nodejs":     SourceTypeNPM,
	"jsr":        SourceTypeJSR,
	"ts":         SourceTypeNPM,
	"tsx":        SourceTypeNPM,
	"typescript": SourceTypeNPM,
	"rust":       SourceTypeCratesIO,
	"rs":         SourceTypeCratesIO,
	"ruby":       SourceTypeRubyGems,
	"rb":         SourceTypeRubyGems,
	"gem":        SourceTypeRubyGems,
}

// DetectDocSource attempts to detect the documentation source from a package path
// If explicitLang is provided, it will be used as an explicit language hint
func DetectDocSource(pkgPath string, explicitLang string) DocSource {
	var result DocSource

	// If explicit language is provided, try to resolve it
	if explicitLang != "" {
		if source, ok := languageAliases[explicitLang]; ok {
			result.Type = source
		}
	}

	// Check for JavaScript package prefixes
	if result.Type == SourceTypeJSR {
		return DocSource{
			Type:        SourceTypeJSR,
			PackagePath: pkgPath,
		}
	}
	if result.Type == SourceTypeNPM {
		return DocSource{
			Type:        SourceTypeNPM,
			PackagePath: pkgPath,
		}
	}

	// Check for Rust package names (from crates.io)
	if result.Type == SourceTypeCratesIO {
		return DocSource{
			Type:        SourceTypeCratesIO,
			PackagePath: pkgPath,
		}
	}

	// Check for Ruby package names (from rubygems.org)
	if result.Type == SourceTypeRubyGems {
		return DocSource{
			Type:        SourceTypeRubyGems,
			PackagePath: pkgPath,
		}
	}

	// Check for known Go package domains
	if result.Type == SourceTypeGoPkgDev || strings.HasPrefix(pkgPath, "golang.org/") ||
		strings.HasPrefix(pkgPath, "go.dev/") ||
		strings.HasPrefix(pkgPath, "pkg.go.dev/") ||
		strings.HasPrefix(pkgPath, "go.pkg.dev/") {
		return DocSource{
			Type:        SourceTypeGoPkgDev,
			PackagePath: pkgPath,
		}
	}

	// For GitHub/GitLab repositories, try to detect from the path
	if result.Type == SourceTypeGitHub || strings.HasPrefix(pkgPath, "github.com/") ||
		result.Type == SourceTypeGitLab || strings.HasPrefix(pkgPath, "gitlab.com/") {
		parts := strings.Split(pkgPath, "/")
		if len(parts) >= 3 {
			// Check if the repository name contains language hints
			repoName := strings.ToLower(parts[2])
			if strings.HasPrefix(repoName, "go-") ||
				strings.HasSuffix(repoName, "-go") ||
				strings.Contains(repoName, ".go") {
				return DocSource{
					Type:        SourceTypeGoPkgDev,
					PackagePath: pkgPath,
				}
			}
			if strings.HasPrefix(repoName, "rust-") ||
				strings.HasSuffix(repoName, "-rust") ||
				strings.HasPrefix(repoName, "rs-") ||
				strings.HasSuffix(repoName, "-rs") {
				return DocSource{
					Type:        SourceTypeCratesIO,
					PackagePath: pkgPath,
				}
			}
			if strings.HasPrefix(repoName, "ruby-") ||
				strings.HasSuffix(repoName, "-ruby") ||
				strings.HasPrefix(repoName, "rb-") ||
				strings.HasSuffix(repoName, "-rb") {
				return DocSource{
					Type:        SourceTypeRubyGems,
					PackagePath: pkgPath,
				}
			}
		}
	}

	// Default to GitHub/GitLab for unknown languages or repository paths
	if strings.HasPrefix(pkgPath, "github.com/") {
		return DocSource{
			Type:        SourceTypeGitHub,
			PackagePath: pkgPath,
		}
	}
	if strings.HasPrefix(pkgPath, "gitlab.com/") {
		return DocSource{
			Type:        SourceTypeGitLab,
			PackagePath: pkgPath,
		}
	}

	return DocSource{
		Type:        SourceTypeUnknown,
		PackagePath: pkgPath,
	}
}
