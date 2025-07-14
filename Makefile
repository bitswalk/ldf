# Linux Distribution Factory Makefile

# Variables
BINARY_DIR := bin
CMD_DIR := cmd
COVERAGE_DIR := coverage

# Binary names
CLI_BINARY := ldf
API_BINARY := ldf-api
TUI_BINARY := ldf-tui

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Build flags
LDFLAGS := -ldflags "-s -w"

# Platforms
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: all build clean test coverage fmt lint install help

# Default target
all: clean lint test build

# Help target
help:
	@echo "Linux Distribution Factory - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Run clean, lint, test, and build"
	@echo "  build        - Build all binaries"
	@echo "  build-cli    - Build CLI binary only"
	@echo "  build-api    - Build API binary only"
	@echo "  build-tui    - Build TUI binary only"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  install      - Install binaries to /usr/local/bin"
	@echo "  deps         - Download dependencies"
	@echo "  cross        - Cross-compile for multiple platforms"
	@echo "  docker       - Build Docker image"
	@echo "  run-api      - Run API server"
	@echo "  run-tui      - Run TUI"
	@echo "  help         - Show this help message"

# Build all binaries
build: build-cli build-api build-tui

# Build individual binaries
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(CLI_BINARY) $(CMD_DIR)/ldf/main.go

build-api:
	@echo "Building API..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(API_BINARY) $(CMD_DIR)/ldf-api/main.go

build-tui:
	@echo "Building TUI..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(TUI_BINARY) $(CMD_DIR)/ldf-tui/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -rf build/

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

# Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not installed. Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
		exit 1; \
	fi

# Install binaries
install: build
	@echo "Installing binaries..."
	@sudo cp $(BINARY_DIR)/$(CLI_BINARY) /usr/local/bin/
	@sudo cp $(BINARY_DIR)/$(API_BINARY) /usr/local/bin/
	@sudo cp $(BINARY_DIR)/$(TUI_BINARY) /usr/local/bin/
	@echo "Installation complete!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Cross-compile for multiple platforms
cross:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d/ -f1) \
		GOARCH=$$(echo $$platform | cut -d/ -f2) \
		output_name=$(BINARY_DIR)/$(CLI_BINARY)-$$(echo $$platform | tr / -); \
		echo "Building $$output_name..."; \
		GOOS=$$(echo $$platform | cut -d/ -f1) GOARCH=$$(echo $$platform | cut -d/ -f2) \
		$(GOBUILD) $(LDFLAGS) -o $$output_name $(CMD_DIR)/ldf/main.go; \
	done

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t linux-distribution-factory:latest .

# Run API server
run-api: build-api
	@echo "Starting API server..."
	./$(BINARY_DIR)/$(API_BINARY)

# Run TUI
run-tui: build-tui
	@echo "Starting TUI..."
	./$(BINARY_DIR)/$(TUI_BINARY)

# Generate OpenAPI documentation
openapi:
	@echo "Generating OpenAPI documentation..."
	@# Add your OpenAPI generation command here

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	$(GOMOD) download
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development setup complete!"
