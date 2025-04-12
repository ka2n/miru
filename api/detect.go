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
	SourceTypeGitHub   = "github.com"
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

	// For GitHub repositories, try to detect from the path
	if result.Type == SourceTypeGitHub || strings.HasPrefix(pkgPath, "github.com/") {
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
		}
	}

	// Default to GitHub for unknown languages or GitHub paths
	if strings.HasPrefix(pkgPath, "github.com/") {
		return DocSource{
			Type:        SourceTypeGitHub,
			PackagePath: pkgPath,
		}
	}

	return DocSource{
		Type:        SourceTypeUnknown,
		PackagePath: pkgPath,
	}
}
