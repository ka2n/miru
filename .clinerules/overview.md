# miru Project Overview

## Project Purpose

Provide a CLI tool for viewing package documentation with a man-like interface.

## Key Features

- Display documentation with man-style interface
- Package search functionality
- Browser-based documentation viewing
- Support for multiple documentation sources (go.pkg.dev, pkg.go.dev, GitHub, etc.)

## Technology Stack

- Implementation Language: Go
- Browser Integration: github.com/haya14busa/go-openbrowser
- Configuration: Environment variables (MIRU_BROWSER, MIRU_BROWSER_PATH)

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

## Command Structure

```bash
miru [package]           # Display documentation in man format
miru [package] -b        # Display documentation in browser
```

## Next Steps

1. Implement core package functionality
2. Implement documentation fetcher
3. Implement man-style rendering
4. Implement MCP server functionality
5. Create tests

## License

MIT License
