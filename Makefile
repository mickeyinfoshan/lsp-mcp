# MCP-LSP Bridge Service Makefile

# Project information
PROJECT_NAME := lsp-mcp
VERSION := 1.1.0
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go configuration
GO := go
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)' -X main.gitCommit=$(GIT_COMMIT)"
BINARY_NAME := $(PROJECT_NAME)
BUILD_DIR := ./bin
CMD_DIR := ./cmd/server

# Default target
.PHONY: all
all: clean build

# Build
.PHONY: build
build:
	@echo "Building $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building all platform versions..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "All platform builds complete"

# Run
.PHONY: run
run: build
	@echo "Starting $(PROJECT_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode
.PHONY: dev
dev:
	@echo "Starting $(PROJECT_NAME) in development mode..."
	$(GO) run $(CMD_DIR) -config ./config/config.yaml

# Test
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Test coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests and generating coverage report..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code formatting
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Code check
.PHONY: vet
vet:
	@echo "Checking code..."
	$(GO) vet ./...

# Dependency management
.PHONY: mod-tidy
mod-tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

.PHONY: mod-download
mod-download:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Clean
.PHONY: clean
clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install
.PHONY: install
install: build
	@echo "Installing $(PROJECT_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete: /usr/local/bin/$(BINARY_NAME)"

# Uninstall
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(PROJECT_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation complete"

# Create configuration directory and sample config
.PHONY: setup-config
setup-config:
	@echo "Setting up configuration file..."
	@mkdir -p ./config
	@if [ ! -f ./config/config.yaml ]; then \
		echo "Configuration file already exists, skipping creation"; \
	else \
		echo "Configuration file does not exist, please create it manually"; \
	fi

# Create log directory
.PHONY: setup-logs
setup-logs:
	@echo "Creating log directory..."
	@mkdir -p ./logs

# Full setup
.PHONY: setup
setup: setup-config setup-logs mod-download
	@echo "Project setup complete"

# Code quality check
.PHONY: lint
lint: fmt vet
	@echo "Code quality check complete"

# Full CI pipeline
.PHONY: ci
ci: mod-tidy lint test build
	@echo "CI pipeline complete"

# Show help
.PHONY: help
help:
	@echo "MCP-LSP Bridge Service Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build project"
	@echo "  build-all      - Build all platform versions"
	@echo "  run            - Build and run project"
	@echo "  dev            - Run in development mode"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests and generate coverage report"
	@echo "  fmt            - Format code"
	@echo "  vet            - Check code"
	@echo "  lint           - Code quality check (fmt + vet)"
	@echo "  mod-tidy       - Tidy dependencies"
	@echo "  mod-download   - Download dependencies"
	@echo "  clean          - Clean build files"
	@echo "  install        - Install to system"
	@echo "  uninstall      - Uninstall from system"
	@echo "  setup          - Initial project setup"
	@echo "  setup-config   - Set up configuration file"
	@echo "  setup-logs     - Create log directory"
	@echo "  ci             - Full CI pipeline"
	@echo "  help           - Show this help information"
	@echo ""
	@echo "Examples:"
	@echo "  make build     # Build project"
	@echo "  make dev       # Run in development mode"
	@echo "  make test      # Run tests"
	@echo "  make ci        # Run full CI pipeline"

# Version information
.PHONY: version
version:
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Build release package
.PHONY: release
release: build
	@echo "Preparing release directory..."
	@rm -rf release/lsp-mcp-release
	@mkdir -p release/lsp-mcp-release
	@cp ./bin/$(BINARY_NAME) release/lsp-mcp-release/lsp-mcp
	@cp release/config.yaml release/lsp-mcp-release/config.yaml
	@cd release && zip -y -r lsp-mcp-release-$(VERSION).zip lsp-mcp-release
	@echo "Release package generated: release/lsp-mcp-release-$(VERSION).zip"