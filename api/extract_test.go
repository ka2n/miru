package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// readTestFile reads a test file from the testdata directory
func readTestFile(t *testing.T, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", filename, err)
	}
	return string(content)
}

func TestExtractRelatedSources(t *testing.T) {
	content := `# Express.js

[Express](https://www.npmjs.com/package/express) is a web framework for Node.js.

Install using:
$ npm install express
`
	want := []RelatedSource{
		{
			Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
			URL:  "https://www.npmjs.com/package/express",
			From: "document",
		},
	}

	got := ExtractRelatedSources(content, "express")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ExtractRelatedSources() mismatch (-want +got):\n%s", diff)
	}
}

func TestURLExtraction(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []string
	}{
		{
			name:     "Markdown links",
			filename: "url_markdown.md",
			want: []string{
				"https://example.com",
				"https://another.com",
			},
		},
		{
			name:     "Raw URLs",
			filename: "url_raw.md",
			want: []string{
				"https://example.com",
				"https://another.com",
			},
		},
		{
			name:     "Mixed links",
			filename: "url_mixed.md",
			want: []string{
				"https://example.com",
				"https://another.com",
			},
		},
		{
			name:     "Duplicate URLs",
			filename: "url_duplicate.md",
			want: []string{
				"https://example.com",
			},
		},
		{
			name:     "No URLs",
			filename: "url_empty.md",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readTestFile(t, tt.filename)
			got := extractURLs(content)

			// Check if all expected URLs are in the result
			if len(got) != len(tt.want) {
				t.Errorf("extractURLs() returned %d URLs, want %d", len(got), len(tt.want))
			}

			// Create a map for easier lookup
			gotMap := make(map[string]bool)
			for _, url := range got {
				gotMap[url] = true
			}

			// Check if all expected URLs are in the result
			for _, url := range tt.want {
				if !gotMap[url] {
					t.Errorf("extractURLs() missing URL: %s", url)
				}
			}
		})
	}
}

func TestFilterAndDeduplicate(t *testing.T) {
	tests := []struct {
		name           string
		sources        []RelatedSource
		currentPackage string
		want           []RelatedSource
	}{
		{
			name: "Duplicate URLs",
			sources: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
			currentPackage: "express",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
		},
		{
			name: "Matching package",
			sources: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
			currentPackage: "express",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
		},
		{
			name: "Non-matching package",
			sources: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
			currentPackage: "react",
			want:           nil,
		},
		{
			name:           "Empty sources",
			sources:        []RelatedSource{},
			currentPackage: "express",
			want:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAndDeduplicate(tt.sources, tt.currentPackage)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("filterAndDeduplicate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandExtraction(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     []RelatedSource
	}{
		{
			name:     "NPM commands",
			filename: "command_npm.md",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
			},
		},
		{
			name:     "JSR commands",
			filename: "command_jsr.md",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeJSR.String()),
					URL:  "https://jsr.io/@hono/hono",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeJSR.String()),
					URL:  "https://jsr.io/@hono/hono",
					From: "document",
				},
			},
		},
		{
			name:     "Cargo commands",
			filename: "command_cargo.md",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeCratesIO.String()),
					URL:  "https://crates.io/crates/tokio",
					From: "document",
				},
			},
		},
		{
			name:     "Gem commands",
			filename: "command_gem.md",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeRubyGems.String()),
					URL:  "https://rubygems.org/gems/rails",
					From: "document",
				},
			},
		},
		{
			name:     "Mixed commands",
			filename: "command_mixed.md",
			want: []RelatedSource{
				{
					Type: RelatedSourceTypeFromString(SourceTypeNPM.String()),
					URL:  "https://www.npmjs.com/package/express",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeCratesIO.String()),
					URL:  "https://crates.io/crates/tokio",
					From: "document",
				},
				{
					Type: RelatedSourceTypeFromString(SourceTypeRubyGems.String()),
					URL:  "https://rubygems.org/gems/rails",
					From: "document",
				},
			},
		},
		{
			name:     "No commands",
			filename: "command_empty.md",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readTestFile(t, tt.filename)
			got := extractSourcesFromCommands(content)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("extractSourcesFromCommands() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
