# Makefile for Bitbucket CLI (bt)

# Variables
BINARY_NAME=bt
BUILD_DIR=./build
CMD_DIR=./cmd/bt
PKG_LIST=$(shell go list ./... | grep -v /vendor/)
TEST_PACKAGES=$(shell go list ./... | grep -v /vendor/ | grep -v /test/integration)
INTEGRATION_TEST_PACKAGES=$(shell go list ./test/integration/...)

# Version information
VERSION=$(shell git describe --tags --exact-match 2>/dev/null || echo "0.0.1")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION=$(shell go version | cut -d' ' -f3)

# Build optimization flags
BUILD_FLAGS=-trimpath
LDFLAGS=-ldflags "-s -w -X github.com/carlosarraes/bt/pkg/version.Version=${VERSION} \
                  -X github.com/carlosarraes/bt/pkg/version.Commit=${COMMIT} \
                  -X github.com/carlosarraes/bt/pkg/version.Date=${DATE} \
                  -X github.com/carlosarraes/bt/pkg/version.GoVersion=${GOVERSION}"

# Development flags (no optimization)
DEV_LDFLAGS=-ldflags "-X github.com/carlosarraes/bt/pkg/version.Version=${VERSION} \
                      -X github.com/carlosarraes/bt/pkg/version.Commit=${COMMIT} \
                      -X github.com/carlosarraes/bt/pkg/version.Date=${DATE} \
                      -X github.com/carlosarraes/bt/pkg/version.GoVersion=${GOVERSION}"

# Coverage settings
COVERAGE_DIR=./coverage
COVERAGE_PROFILE=${COVERAGE_DIR}/coverage.out
COVERAGE_HTML=${COVERAGE_DIR}/coverage.html

.PHONY: help build clean test test-race test-cover test-integration lint fmt vet deps install uninstall run dev watch release security vulnerability-check

# Default target
all: build

# Help target
help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary (optimized)
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${CMD_DIR}
	@echo "Built ${BUILD_DIR}/${BINARY_NAME}"

build-dev: ## Build for development (no optimizations)
	@echo "Building ${BINARY_NAME} for development..."
	@go build ${DEV_LDFLAGS} -o ${BINARY_NAME} ${CMD_DIR}

build-static: ## Build static binary (CGO_ENABLED=0)
	@echo "Building static ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@CGO_ENABLED=0 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-static ${CMD_DIR}

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing ${BINARY_NAME}..."
	@go install ${BUILD_FLAGS} ${LDFLAGS} ${CMD_DIR}

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
test: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v ${TEST_PACKAGES}

test-short: ## Run unit tests (short mode)
	@echo "Running unit tests (short)..."
	@go test -short -v ${TEST_PACKAGES}

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -race -v ${TEST_PACKAGES}

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p ${COVERAGE_DIR}
	@go test -coverprofile=${COVERAGE_PROFILE} ${TEST_PACKAGES}
	@go tool cover -html=${COVERAGE_PROFILE} -o ${COVERAGE_HTML}
	@echo "Coverage report generated: ${COVERAGE_HTML}"
	@go tool cover -func=${COVERAGE_PROFILE} | tail -1

test-cover-all: ## Run all tests with coverage (including integration)
	@echo "Running all tests with coverage..."
	@mkdir -p ${COVERAGE_DIR}
	@go test -coverprofile=${COVERAGE_DIR}/unit.out ${TEST_PACKAGES}
	@go test -coverprofile=${COVERAGE_DIR}/integration.out ${INTEGRATION_TEST_PACKAGES}
	@echo "mode: set" > ${COVERAGE_PROFILE}
	@cat ${COVERAGE_DIR}/unit.out ${COVERAGE_DIR}/integration.out | grep -v mode: >> ${COVERAGE_PROFILE} || true
	@go tool cover -html=${COVERAGE_PROFILE} -o ${COVERAGE_HTML}
	@go tool cover -func=${COVERAGE_PROFILE} | tail -1

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ${INTEGRATION_TEST_PACKAGES}

test-cli: ## Run CLI tests using testscript
	@echo "Running CLI tests..."
	@go test -v ./test/cli/...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ${PKG_LIST}

bench-compare: ## Run benchmarks and save for comparison
	@echo "Running benchmarks (comparison mode)..."
	@go test -bench=. -benchmem ${PKG_LIST} | tee bench.out

# Code quality targets
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ${PKG_LIST}

fmt-check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@unformatted=$$(go fmt ${PKG_LIST}); \
	if [ -n "$$unformatted" ]; then \
		echo "Code not formatted. Run 'make fmt'"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ${PKG_LIST}

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run 'make setup-dev' to install"; \
		exit 1; \
	fi

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@golangci-lint run --fix

# Security targets
security: ## Run security checks
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Run 'make setup-dev' to install"; \
	fi

vulnerability-check: ## Check for known vulnerabilities
	@echo "Checking for vulnerabilities..."
	@go list -json -deps ./... | nancy sleuth

# Quality checks
check: fmt-check vet lint test ## Run all checks (format, vet, lint, test)
check-all: fmt-check vet lint security test test-integration ## Run all checks including integration tests

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

deps-audit: ## Audit dependencies for vulnerabilities
	@echo "Auditing dependencies..."
	@go list -json -deps ./... | audit

deps-clean: ## Clean unused dependencies
	@echo "Cleaning unused dependencies..."
	@go mod tidy

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -f ${BINARY_NAME}
	@rm -rf ${COVERAGE_DIR}
	@rm -f *.out *.html
	@rm -f bench.out

clean-deps: ## Clean module cache
	@echo "Cleaning module cache..."
	@go clean -modcache

clean-test: ## Clean test cache
	@echo "Cleaning test cache..."
	@go clean -testcache

clean-all: clean clean-test ## Clean everything

# Release targets
release-build: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p ${BUILD_DIR}
	@GOOS=linux GOARCH=amd64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${CMD_DIR}
	@GOOS=linux GOARCH=arm64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${CMD_DIR}
	@GOOS=darwin GOARCH=amd64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${CMD_DIR}
	@GOOS=darwin GOARCH=arm64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${CMD_DIR}
	@GOOS=windows GOARCH=amd64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${CMD_DIR}
	@GOOS=windows GOARCH=arm64 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-arm64.exe ${CMD_DIR}
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

release-checksums: release-package ## Generate checksums for release files
	@echo "Generating checksums..."
	@cd ${BUILD_DIR} && \
	sha256sum *.tar.gz *.zip > checksums.txt
	@echo "Checksums generated in ${BUILD_DIR}/checksums.txt"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t bt:${VERSION} .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -it bt:${VERSION}

docker-release: ## Build multi-arch Docker image for release
	@echo "Building multi-arch Docker image..."
	@docker buildx build --platform linux/amd64,linux/arm64 -t bt:${VERSION} .

# Git targets
git-tag: ## Create a git tag with current version
	@echo "Creating git tag ${VERSION}..."
	@git tag -a ${VERSION} -m "Release ${VERSION}"
	@echo "Tag created. Push with: git push origin ${VERSION}"

# Info targets
info: ## Show build information
	@echo "Build Information:"
	@echo "  Binary Name:     ${BINARY_NAME}"
	@echo "  Version:         ${VERSION}"
	@echo "  Commit:          ${COMMIT}"
	@echo "  Date:            ${DATE}"
	@echo "  Go Version:      ${GOVERSION}"
	@echo "  Build Dir:       ${BUILD_DIR}"
	@echo "  Test Packages:   $(shell echo ${TEST_PACKAGES} | wc -w)"
	@echo "  Source Files:    $(shell find . -name "*.go" | grep -v vendor | wc -l)"

# Quick commands
quick-test: ## Quick test and build
	@make test && make build

quick-check: ## Quick format and test
	@make fmt && make test

# Development environment setup
setup-dev: ## Setup development environment
	@echo "Setting up development environment..."
	@go mod download
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest
	@go install github.com/sonatypeoss/nancy@latest
	@echo "Setup complete!"

setup-hooks: ## Setup git hooks for development
	@echo "Setting up git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/sh\nmake fmt-check && make vet && make lint' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed!"

# Performance and profiling
profile-cpu: ## Run CPU profiling
	@echo "Running CPU profiling..."
	@go test -cpuprofile=cpu.prof -bench=. ${PKG_LIST}
	@echo "CPU profile saved to cpu.prof. View with: go tool pprof cpu.prof"

profile-mem: ## Run memory profiling
	@echo "Running memory profiling..."
	@go test -memprofile=mem.prof -bench=. ${PKG_LIST}
	@echo "Memory profile saved to mem.prof. View with: go tool pprof mem.prof"

# Cache management
cache-clear: ## Clear build and test caches
	@echo "Clearing build and test caches..."
	@go clean -cache -testcache -modcache

# Show current status
status: ## Show project status
	@echo "Project Status:"
	@echo "  Go version:      $(shell go version)"
	@echo "  Module:          $(shell head -1 go.mod)"
	@echo "  Dependencies:    $(shell go list -m all | wc -l) modules"
	@echo "  Source files:    $(shell find . -name "*.go" | grep -v vendor | wc -l) files"
	@echo "  Test files:      $(shell find . -name "*_test.go" | wc -l) files"
	@echo "  Git status:      $(shell git status --porcelain | wc -l) changes"
	@echo "  Latest commit:   $(shell git log -1 --oneline)"

# CI/CD helpers
ci-test: ## Run tests in CI environment
	@echo "Running CI tests..."
	@go test -v -coverprofile=coverage.out ${TEST_PACKAGES}
	@go tool cover -func=coverage.out

ci-build: ## Build in CI environment
	@echo "Building in CI environment..."
	@CGO_ENABLED=0 go build ${BUILD_FLAGS} ${LDFLAGS} -o ${BINARY_NAME} ${CMD_DIR}

ci-lint: ## Run linting in CI environment
	@echo "Running CI linting..."
	@golangci-lint run --out-format=github-actions

# Database/Config related targets for new packages
config-validate: ## Validate configuration files
	@echo "Validating configuration..."
	@go run ${CMD_DIR} config validate

config-example: ## Generate example configuration
	@echo "Generating example configuration..."
	@go run ${CMD_DIR} config example > config.example.yaml

# API testing targets
test-api: ## Run API integration tests
	@echo "Running API tests..."
	@go test -v -tags=api ./pkg/api/...

test-auth: ## Run authentication tests
	@echo "Running authentication tests..."
	@go test -v -tags=auth ./pkg/auth/...

# Performance testing
perf-test: ## Run performance tests
	@echo "Running performance tests..."
	@go test -v -tags=perf -timeout=30m ./test/performance/...

load-test: ## Run load tests
	@echo "Running load tests..."
	@go test -v -tags=load -timeout=60m ./test/load/...