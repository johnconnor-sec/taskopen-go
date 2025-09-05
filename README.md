# Taskopen - Go Edition

A powerful task annotation opener for Taskwarrior, rewritten in Go for better performance and maintainability.

[![CI](https://github.com/johnconnor-sec/taskopen-go/actions/workflows/ci.yml/badge.svg)](https://github.com/johnconnor-sec/taskopen-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/johnconnor-sec/taskopen-go)](https://goreportcard.com/report/github.com/johnconnor-sec/taskopen-go)
[![Coverage](https://codecov.io/gh/johnconnor-sec/taskopen-go/branch/main/graph/badge.svg)](https://codecov.io/gh/johnconnor-sec/taskopen-go)
[![License: GPL v2](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)

## Features

- 🚀 **Fast**: Startup time < 100ms, action matching < 10ms for 1000+ actions
- 🔍 **Smart**: Fuzzy search and interactive menus for quick action discovery  
- 🛡️ **Secure**: Process execution sandboxing and input validation
- 🎨 **Beautiful**: Rich terminal output with accessibility support
- 🔧 **Configurable**: YAML configuration with schema validation and IDE support
- 🧪 **Tested**: >90% test coverage with comprehensive integration tests

## Quick Start

### Installation

#### From Source (Recommended for Development)

```bash
git clone https://github.com/johnconnor-sec/taskopen-go.git
cd taskopen-go/
make dev-setup
make build
sudo make install
```

#### Using Go Install

```bash
go install github.com/johnconnor-sec/taskopen-go/cmd/taskopen@latest
```

#### Binary Releases

Download pre-built binaries from [Releases](https://github.com/johnconnor-sec/taskopen-go/releases).

### Basic Usage

```bash
# Open annotations from selected tasks
taskopen

# Run diagnostics to verify setup
taskopen diagnostics

# Initialize configuration interactively
taskopen config init

# Show version and build info
taskopen version
```

## Development

### Prerequisites

- **Go 1.21+**: [Install Go](https://golang.org/doc/install)
- **Taskwarrior**: Required for integration tests

  ```bash
  # Ubuntu/Debian
  sudo apt-get install taskwarrior
  
  # macOS
  brew install task
  ```

- **Development Tools** (automatically installed with `make dev-setup`):
  - golangci-lint
  - gosec

### Development Setup

```bash
# Clone and setup
git clone https://github.com/johnconnor-sec/taskopen-go.git
cd taskopen-go

# Setup development environment
make dev-setup

# Run all quality checks
make quality

# Build and run
make run
```

### Project Structure

```
taskopen-go/
├── cmd/taskopen/           # CLI entry point
├── internal/               # Private packages
│   ├── types/             # Core types with validation
│   ├── config/            # Configuration handling  
│   ├── exec/              # Process execution
│   ├── output/            # Terminal output
│   └── core/              # Business logic
├── pkg/                   # Public APIs (future)
├── test/                  # Integration tests
├── .github/workflows/     # CI/CD configuration
└── docs/                  # Documentation
```

### Common Development Tasks

```bash
# Build binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests (requires taskwarrior)
make test-integration

# Format code
make fmt

# Run linter
make lint

# Run security checks  
make security

# Watch for changes and rebuild (requires entr)
make watch

# Prepare for release
make prepare-release
```

### Code Quality Standards

We maintain high code quality through:

- **Testing**: >90% coverage, unit + integration tests
- **Linting**: golangci-lint with strict configuration  
- **Security**: gosec security scanning
- **Formatting**: gofmt + goimports
- **Documentation**: Comprehensive godoc comments

### Performance Targets

- Startup time: < 100ms (cold start)
- Action matching: < 10ms for 1000+ actions
- Memory usage: < 50MB for typical workflows
- Taskwarrior query time: < 2x current Nim implementation

## Migration from Nim Version

The Go version is designed for seamless migration:

1. **Configuration**: Automatic INI → YAML migration with `taskopen config migrate`
2. **Actions**: 100% compatibility with existing action definitions
3. **Performance**: Significant improvements in startup and execution time
4. **Features**: All Nim features plus new interactive capabilities

### Migration Guide

```bash
# Backup existing configuration
cp ~/.taskopenrc ~/.taskopenrc.backup

# Install Go version
make install

# Migrate configuration
taskopen config migrate

# Verify migration
taskopen diagnostics

# Test with existing workflows
taskopen
```

## Configuration

### YAML Configuration (Recommended)

```yaml
# ~/.config/taskopen/config.yml
general:
  editor: "vim"
  browser: "firefox"

actions:
  - name: "edit"
    target: "annotation"
    regex: ".*"
    command: "$EDITOR"
    modes: ["batch", "interactive"]
    
  - name: "open-url"
    target: "annotation" 
    regex: "https?://.*"
    command: "$BROWSER"
```

### INI Configuration (Legacy Support)

```ini
# ~/.taskopenrc (automatically migrated)
[general]
editor = vim
browser = firefox

[ACTION edit]
target = annotation
regex = .*
command = $EDITOR
```

## Testing

### Unit Tests

```bash
make test
```

### Integration Tests

```bash
make test-integration
```

### Performance Tests

```bash
go test -bench=. ./internal/...
```

## Contributing

Contributions are welcome! Please see the [Contributing Guide](CONTRIBUTING.md) (*TODO*) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes following our code standards
4. Run quality checks: `make quality`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to your fork: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `make fmt` for consistent formatting
- Write comprehensive tests for new features
- Document public APIs with godoc comments

## Architecture

The Go version uses a modern, modular architecture:

- **Types**: Strongly typed with validation
- **Configuration**: Schema-driven with helpful error messages  
- **Execution**: Context-aware with cancellation support
- **Output**: Rich terminal UI with accessibility support
- **Testing**: Comprehensive unit and integration test coverage

## License

This project is licensed under the GNU General Public License v2.0 - see the [LICENSE](LICENSE) (TODO) file for details.

## Acknowledgments

- Original Nim implementation by [Johannes Schlatow](https://github.com/jschlatow)
- Taskwarrior project for the excellent task management foundation
- Go community for excellent tooling and libraries
