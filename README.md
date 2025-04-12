# miru

A command-line tool for viewing package documentation with a man-like interface.

## Features

- View package documentation in terminal with man-like interface
- Support for multiple package registries and documentation sources
- Open documentation in browser
- Search packages and their documentation
- Configurable browser integration

## Installation

```bash
go install github.com/ka2n/miru/cmd/miru@latest
```

## Usage

View package documentation in terminal:

```bash
miru [package]                    # Display documentation in man-like interface
miru [package] -b                 # Open documentation in browser
miru [lang] [package]             # Specify package language explicitly
miru [package] -lang [lang]       # Specify package language with flag
```

Examples:

```bash
# View package documentation
miru github.com/spf13/cobra

# Open documentation in browser
miru golang.org/x/sync -b

# Specify language explicitly
miru go github.com/spf13/cobra

# Specify language with flag
miru github.com/spf13/cobra -lang go
```

## Package Structure

```
github.com/ka2n/miru/
├── api/      # Core implementations for documentation fetching and rendering
├── cli/      # CLI interface implementation
├── mcp/      # Model Context Protocol server implementation
└── cmd/miru/ # Main command implementation
```

## Configuration

Browser integration can be configured through environment variables:

```bash
MIRU_BROWSER=firefox    # Specify browser to use
MIRU_BROWSER_PATH=/path/to/browser  # Specify browser binary path
```

By default, miru uses [go-openbrowser](https://github.com/haya14busa/go-openbrowser) for browser integration.

## Documentation Sources

miru supports fetching documentation from:

- go.pkg.dev
- pkg.go.dev
- GitHub repositories
- Local module documentation

## Development

### Requirements

- Go 1.21 or later

### Setup

1. Clone the repository

```bash
git clone https://github.com/ka2n/miru.git
cd miru
```

2. Build

```bash
go build ./cmd/miru
```

3. Run tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details
