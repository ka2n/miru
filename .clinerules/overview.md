# miru Project Overview

## Project Purpose

Provide a CLI tool for viewing package documentation with a man-like interface.

## Key Features

- Display documentation with man-style interface
- Package search functionality
- Browser-based documentation viewing
- Support for multiple documentation sources (pkg.go.dev, GitHub, etc.)

## Technology Stack

- Implementation Language: Go
- Browser Integration: github.com/pkg/browser
- Configuration: Environment variables (MIRU_BROWSER, MIRU_BROWSER_PATH)
- CLI Framework: github.com/spf13/cobra

## Package Structure

```
github.com/ka2n/miru/
├── api/      # Core implementations for documentation fetching and rendering
├── cli/      # CLI interface implementation
├── mcp/      # MCP server implementation
└── cmd/miru/ # Main command implementation
```

## Design Principles

- Minimize package separation (maintain simple structure)
- Clear separation of package responsibilities
- Extensible MCP server implementation
- Use English for all error messages and code comments
  - Error messages should be clear and concise
  - Comments should follow Go's documentation conventions

## Command Structure

```bash
miru [package]                    # Display documentation in man format
miru [package] -b                 # Display documentation in browser
miru [lang] [package]             # Specify package language explicitly
miru [package] --lang [lang]      # Specify package language with flag
miru [package] -o json           # Output metadata in JSON format
miru version                      # Display version information
```

Output formats:

- Default: Display documentation in man-style pager
- Browser (-b): Open documentation in browser
- JSON (-o json): Output package metadata without content for testing purposes

Language detection:

- Explicit language specification through command-line arguments
- Automatic language detection from package path and repository structure
- Fallback to GitHub documentation when language-specific source is unavailable

## Next Steps

1. Migrate to Cobra CLI framework
   - Implement root command for documentation display
   - Add version subcommand
   - Integrate existing flag handling
2. Implement core package functionality
3. Implement documentation fetcher
4. Implement man-style rendering
5. Implement MCP server functionality
6. Create tests

## License

MIT License
