# miru - Package Documentation Lookup

miru is a CLI tool that fetches package documentation, README content, and related URLs (repository, registry, homepage, docs) from multiple language registries.

## Supported Languages

{{SUPPORTED_LANGUAGES_TABLE}}

## Usage Patterns

### Get README content (markdown)

Pipe miru output to get the README rendered as markdown. When stdout is not a terminal, miru outputs the README text directly, preceded by a YAML frontmatter block with URL and metadata fields.

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
- `type`: Registry type of the initial query (e.g., "npmjs.com", "pkg.go.dev")
- `url`: Primary documentation URL for the initial query
- `homepage`: Package homepage URL (omitted if not available)
- `repository`: Source code repository URL (omitted if not available)
- `registry`: Package registry page URL (omitted if not available)
- `document`: Documentation page URL (omitted if not available)
- `urls`: Array of all discovered URLs, each with `Type` and `URL`
- `metadata`: Map of metadata collected from all sources (omitted if empty). For GitHub-backed repositories this can include `stars`, `forks`, `open_issues`, `archived` (only present when true), `description`, `language`, `pushed_at`, `created_at`, `default_branch`, `license`, `topics`. Registry sources add their own fields, such as `version`, `license`, `keywords`, or `authors`.

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
  ],
  "metadata": {
    "description": "Fast, unopinionated, minimalist web framework",
    "version": "4.21.2",
    "license": "MIT",
    "stars": 65000,
    "forks": 11000
  }
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
