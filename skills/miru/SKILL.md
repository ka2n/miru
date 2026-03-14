---
name: miru
description: "Look up library/package documentation using the miru CLI. Use this skill whenever you need to find a package's README content, documentation URL, repository URL, registry URL, or homepage URL. Triggers on: looking up package docs, checking library README, finding repository URL, getting package info, researching a library. Also use when the user mentions miru, asks about a package's documentation source, or needs to quickly understand what a library does by reading its README."
allowed-tools:
  - Bash(miru:*)
---

# miru - Package Documentation Lookup

miru is a CLI tool that fetches package documentation, README content, and related URLs (repository, registry, homepage, docs) from multiple language registries.

## Supported Languages

| Language | Aliases | Registry |
|----------|---------|----------|
| Go | go, golang | pkg.go.dev |
| JavaScript/TypeScript | npm, js, ts, typescript, node, nodejs | npmjs.com |
| Rust | rust, rs, crates | crates.io |
| Ruby | ruby, rb, gem | rubygems.org |
| Python | python, py, pip, pypi | pypi.org |
| PHP | php, composer, packagist | packagist.org |
| Deno/JSR | jsr | jsr.io |
| (fallback) | — | github.com, gitlab.com |

## Usage Patterns

### Get README content (markdown)

Pipe miru output to get the README rendered as markdown. When stdout is not a terminal, miru outputs the README text directly.

```bash
miru [package]              # Auto-detect language from package path
miru [lang] [package]       # Specify language explicitly
miru [package] --lang [lang]  # Specify language with flag
```

Examples:
```bash
miru github.com/spf13/cobra     # Go package (auto-detected from path)
miru npm express                 # npm package
miru python requests             # Python package
miru rust serde                  # Rust crate
miru ruby rails                  # Ruby gem
miru php laravel/framework       # PHP package
```

### Get package metadata as JSON

Use `-o json` to get structured metadata including all related URLs:

```bash
miru [package] -o json
```

The JSON output contains:
- `type`: Registry type (e.g., "npmjs.com", "pkg.go.dev")
- `url`: Primary documentation URL
- `homepage`: Package homepage URL (if available)
- `repository`: Source code repository URL (if available)
- `registry`: Package registry page URL (if available)
- `document`: Documentation page URL (if available)
- `urls`: Array of all discovered URLs with their types

Example output:
```json
{
  "type": "npmjs.com",
  "url": "https://www.npmjs.com/package/express",
  "homepage": "https://expressjs.com/",
  "repository": "https://github.com/expressjs/express",
  "registry": "https://www.npmjs.com/package/express",
  "urls": [
    {"Type": "npmjs.com", "URL": "https://www.npmjs.com/package/express"},
    {"Type": "homepage", "URL": "https://expressjs.com/"},
    {"Type": "github.com", "URL": "https://github.com/expressjs/express"}
  ]
}
```

### Open documentation in browser

```bash
miru [package] -b                # Open default documentation page
miru [package] -b=registry       # Open registry page (alias: r)
miru [package] -b=repository     # Open repository page (alias: g)
miru [package] -b=homepage       # Open homepage (alias: h)
```

## How to Use This Skill

1. **To understand what a library does**: Run `miru [lang] [package]` to read the README content in markdown.
2. **To get URLs for a package**: Run `miru [package] -o json` and parse the JSON to extract the specific URL you need.
3. **To find the source code**: Look at the `repository` field from the JSON output.
4. **To find official docs**: Look at the `document` or `homepage` field from the JSON output.

When the user asks about a library and you need to quickly understand it, use miru to fetch the README — it's faster than cloning the repo or fetching web pages manually.
