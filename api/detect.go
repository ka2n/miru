package api

import (
	"strings"

	"github.com/ka2n/miru/api/source"
	"github.com/morikuni/failure/v2"
)

// detectInitialQuery attempts to detect the documentation source from a package path
// If explicitLang is provided, it will be used as an explicit language hint
// If error occurs, it indicates that the package path format is invalid for the explicitLang.
func detectInitialQuery(pkgPath string, explicitLang string) (InitialQuery, error) {
	var sourceType source.Type

	// If explicit language is provided, try to resolve it
	if explicitLang != "" {
		if source, ok := languageAliases[explicitLang]; ok {
			sourceType = source
		}
	}

	// Check for JavaScript package prefixes
	if sourceType == source.TypeJSR {
		// Append the "@" prefix if not present
		if !strings.HasPrefix(pkgPath, "@") {
			pkgPath = "@" + pkgPath
		}

		// Check if the package path is formatted as "@<org>/<name>"
		if strings.Count(pkgPath, "/") != 1 {
			return InitialQuery{}, failure.New(
				ErrInvalidPackagePath,
				failure.Message("JSR package path must be formatted as '@<org>/<name>'"),
				failure.Field(failure.Context{
					"explicitLang": explicitLang,
					"pkgPath":      pkgPath,
				}))
		}

		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeJSR,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}
	if sourceType == source.TypeNPM {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeNPM,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// Check for Rust package names (from crates.io)
	if sourceType == source.TypeCratesIO {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeCratesIO,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// Check for Ruby package names (from rubygems.org)
	if sourceType == source.TypeRubyGems {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeRubyGems,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// Check for Python package names (from pypi.org)
	if sourceType == source.TypePyPI {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypePyPI,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// Check for PHP package names (from packagist.org)
	if sourceType == source.TypePackagist {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypePackagist,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// Check for known Go package domains
	if sourceType == source.TypeGoPkgDev ||
		strings.HasPrefix(pkgPath, "pkg.go.dev/") {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeGoPkgDev,
				Path: pkgPath,
			},
			ForceUpdate: false,
		}, nil
	}

	// For GitHub/GitLab repositories, try to detect from the path
	if sourceType == source.TypeGitHub || strings.HasPrefix(pkgPath, "github.com/") ||
		sourceType == source.TypeGitLab || strings.HasPrefix(pkgPath, "gitlab.com/") {
		parts := strings.Split(pkgPath, "/")
		if len(parts) >= 3 {
			// Check if the repository name contains language hints
			repoName := strings.ToLower(parts[2])
			if strings.HasPrefix(repoName, "go-") ||
				strings.HasSuffix(repoName, "-go") ||
				strings.Contains(repoName, ".go") {
				return InitialQuery{
					SourceRef: source.Reference{
						Type: source.TypeGoPkgDev,
						Path: pkgPath,
					},
					ForceUpdate: false,
				}, nil
			}
		}
	}

	// Default to GitHub/GitLab for unknown languages or repository paths
	if strings.HasPrefix(pkgPath, "github.com/") {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeGitHub,
				Path: strings.TrimPrefix(pkgPath, "github.com/"),
			},
			ForceUpdate: false,
		}, nil
	}
	if strings.HasPrefix(pkgPath, "gitlab.com/") {
		return InitialQuery{
			SourceRef: source.Reference{
				Type: source.TypeGitLab,
				Path: strings.TrimPrefix(pkgPath, "gitlab.com/"),
			},
			ForceUpdate: false,
		}, nil
	}

	return InitialQuery{
		SourceRef: source.Reference{
			Type: source.TypeUnknown,
			Path: pkgPath,
		},
		ForceUpdate: false,
	}, nil
}

// GetLanguageAliases returns a map of language aliases to their documentation source types
func GetLanguageAliases() map[string]source.Type {
	return languageAliases
}

// languageAliases maps language aliases to their canonical language name
var languageAliases = map[string]source.Type{
	// go
	"go":     source.TypeGoPkgDev,
	"golang": source.TypeGoPkgDev,

	// JavaScript, TypeScript
	"js":         source.TypeNPM,
	"javascript": source.TypeNPM,
	"npm":        source.TypeNPM,
	"node":       source.TypeNPM,
	"nodejs":     source.TypeNPM,
	"ts":         source.TypeNPM,
	"tsx":        source.TypeNPM,
	"typescript": source.TypeNPM,
	"jsr":        source.TypeJSR,

	// rust
	"rust":   source.TypeCratesIO,
	"rs":     source.TypeCratesIO,
	"crates": source.TypeCratesIO,

	// ruby
	"ruby": source.TypeRubyGems,
	"rb":   source.TypeRubyGems,
	"gem":  source.TypeRubyGems,

	// python
	"python": source.TypePyPI,
	"py":     source.TypePyPI,
	"pypi":   source.TypePyPI,
	"pip":    source.TypePyPI,

	// php
	"php":       source.TypePackagist,
	"packagist": source.TypePackagist,
	"composer":  source.TypePackagist,
}
