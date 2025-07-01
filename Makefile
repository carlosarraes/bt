# Makefile for Bitbucket CLI (bt)

# Variables
BINARY_NAME=bt
BUILD_DIR=./build
CMD_DIR=./cmd/bt
PKG_LIST=$(shell go list ./... | grep -v /vendor/)

# Version information
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION=$(shell go version | cut -d' ' -f3)

# Linker flags
LDFLAGS=-ldflags "-X github.com/carlosarraes/bt/pkg/version.Version=${VERSION} \
                  -X github.com/carlosarraes/bt/pkg/version.Commit=${COMMIT} \
                  -X github.com/carlosarraes/bt/pkg/version.Date=${DATE} \
                  -X github.com/carlosarraes/bt/pkg/version.GoVersion=${GOVERSION}"

.PHONY: help build clean test test-race test-cover lint fmt vet deps install uninstall run dev watch release

# Default target
all: build

# Help target
help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${CMD_DIR}
	@echo "Built ${BUILD_DIR}/${BINARY_NAME}"

build-dev: ## Build for development (no optimizations)
	@echo "Building ${BINARY_NAME} for development..."
	@go build -o ${BINARY_NAME} ${CMD_DIR}

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing ${BINARY_NAME}..."
	@go install ${LDFLAGS} ${CMD_DIR}

uninstall: ## Remove the binary from $GOPATH/bin
	@echo "Uninstalling ${BINARY_NAME}..."
	@rm -f $(shell go env GOPATH)/bin/${BINARY_NAME}

# Development targets
dev: build-dev ## Build and run in development mode
	@echo "Running ${BINARY_NAME} in development mode..."
	@./${BINARY_NAME} --help

run: build-dev ## Quick build and run with arguments (use: make run ARGS="version")
	@./${BINARY_NAME} $(ARGS)

watch: ## Watch for changes and rebuild (requires entr: brew install entr)
	@echo "Watching for changes... (press Ctrl+C to stop)"
	@find . -name "*.go" | entr -r make build-dev

# Testing targets
test: ## Run tests
	@echo "Running tests..."
	@go test -v ${PKG_LIST}

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -race -v ${PKG_LIST}

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ${PKG_LIST}
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ${PKG_LIST}

# Code quality targets
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ${PKG_LIST}

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ${PKG_LIST}

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@go mod verify

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -f ${BINARY_NAME}
	@rm -f coverage.out coverage.html

clean-deps: ## Clean module cache
	@echo "Cleaning module cache..."
	@go clean -modcache

# Release targets
release-build: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p ${BUILD_DIR}
	@GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${CMD_DIR}
	@GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${CMD_DIR}
	@GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${CMD_DIR}
	@GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${CMD_DIR}
	@GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${CMD_DIR}
	@echo "Release binaries built in ${BUILD_DIR}/"

release-package: release-build ## Package release binaries
	@echo "Packaging release binaries..."
	@cd ${BUILD_DIR} && \
	for binary in ${BINARY_NAME}-*; do \
		if [[ $$binary == *.exe ]]; then \
			zip $$binary.zip $$binary; \
		else \
			tar -czf $$binary.tar.gz $$binary; \
		fi; \
	done
	@echo "Release packages created in ${BUILD_DIR}/"

# Docker targets (for future use)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t bt:${VERSION} .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -it bt:${VERSION}

# Git targets
git-tag: ## Create a git tag with current version
	@echo "Creating git tag ${VERSION}..."
	@git tag -a ${VERSION} -m "Release ${VERSION}"
	@echo "Tag created. Push with: git push origin ${VERSION}"

# Info targets
info: ## Show build information
	@echo "Build Information:"
	@echo "  Binary Name: ${BINARY_NAME}"
	@echo "  Version:     ${VERSION}"
	@echo "  Commit:      ${COMMIT}"
	@echo "  Date:        ${DATE}"
	@echo "  Go Version:  ${GOVERSION}"
	@echo "  Build Dir:   ${BUILD_DIR}"

# Quick commands
quick-test: ## Quick test and build
	@make test && make build

quick-check: ## Quick format and test
	@make fmt && make test

# Development workflow
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@go mod download
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Setup complete!"

# Show current status
status: ## Show project status
	@echo "Project Status:"
	@echo "  Go version: $(shell go version)"
	@echo "  Module: $(shell head -1 go.mod)"
	@echo "  Dependencies: $(shell go list -m all | wc -l) modules"
	@echo "  Source files: $(shell find . -name "*.go" | grep -v vendor | wc -l) files"
	@echo "  Test files: $(shell find . -name "*_test.go" | wc -l) files"