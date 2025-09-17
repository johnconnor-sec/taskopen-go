# Taskopen Go Version - Makefile
# Development workflow automation

.PHONY: help run build test clean install lint fmt deps security coverage dev-setup

# Build configuration
BINARY_NAME=taskopen
GO_VERSION=$(shell go version | cut -d' ' -f3)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

# Linker flags for version info
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION} -X main.commit=${GIT_COMMIT} -X main.date=${BUILD_DATE}"

# Default target
help: ## Show this help message
	@echo "Taskopen Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Build info:"
	@echo "  Go version: $(GO_VERSION)"
	@echo "  Git commit: $(GIT_COMMIT)"
	@echo "  Version:    $(VERSION)"

# Development setup
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "✓ Development environment ready"

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	go mod tidy
	@echo "✓ Dependencies updated"

# Building
build: ## Build the taskopen binary
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/taskopen
	@echo "✓ Build complete: ./$(BINARY_NAME)"

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/taskopen
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/taskopen
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/taskopen
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/taskopen
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/taskopen
	@echo "✓ All platform builds complete in dist/"

# Testing
test: ## Run all tests
	@echo "Running tests..."
	CGO_ENABLED=1 go test -v -race ./... 
	@echo "✓ All tests passed"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

test-integration: ## Run integration tests (requires taskwarrior)
	@echo "Running integration tests..."
	@which task >/dev/null 2>&1 || (echo "❌ taskwarrior not found. Install with: apt-get install taskwarrior" && exit 1)
	go test -v -tags=integration ./test/...
	@echo "✓ Integration tests complete"

# Code quality
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .
	@echo "✓ Code formatted"

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run
	@echo "✓ Linting complete"

security: ## Run security checks
	@echo "Running security scan..."
	gosec ./...
	@echo "✓ Security scan complete"

# Quality gates (used in CI)
quality: fmt lint security test ## Run all quality checks

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html
	go clean -cache -testcache
	@echo "✓ Cleanup complete"

# Installation
install: build ## Install taskopen to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/taskopen
	@echo "✓ $(BINARY_NAME) installed to $(shell go env GOPATH)/bin"

# Development helpers
run: build ## Build and run taskopen with diagnostics
	./$(BINARY_NAME) diagnostics

# Release preparation
prepare-release: ## Prepare for release (run quality checks and build all platforms)
	@echo "Preparing release..."
	make quality
	make build-all
	@echo "✓ Release preparation complete"

# Documentation
docs-serve: ## Serve documentation locally (requires godoc)
	@echo "Starting documentation server..."
	@which godoc >/dev/null 2>&1 || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Documentation available at http://localhost:6060/pkg/github.com/taskopen/taskopen/"
	godoc -http=:6060
