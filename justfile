# Justfile for Bitbucket CLI (bt)

# Variables
binary_name := "bt"
build_dir := "./build"
cmd_dir := "./cmd/bt"
pkg_list := `go list ./... | grep -v /vendor/`
test_packages := `go list ./... | grep -v /vendor/ | grep -v /test/integration`
integration_test_packages := `go list ./test/integration/...`

# Version information
version := `git describe --tags --exact-match 2>/dev/null || echo "0.0.1"`
commit := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
date := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
goversion := `go version | cut -d' ' -f3`

# Build flags
build_flags := "-trimpath"
ldflags := "-s -w -X github.com/carlosarraes/bt/pkg/version.Version=" + version + " -X github.com/carlosarraes/bt/pkg/version.Commit=" + commit + " -X github.com/carlosarraes/bt/pkg/version.Date=" + date + " -X github.com/carlosarraes/bt/pkg/version.GoVersion=" + goversion
dev_ldflags := "-X github.com/carlosarraes/bt/pkg/version.Version=" + version + " -X github.com/carlosarraes/bt/pkg/version.Commit=" + commit + " -X github.com/carlosarraes/bt/pkg/version.Date=" + date + " -X github.com/carlosarraes/bt/pkg/version.GoVersion=" + goversion

# Coverage settings
coverage_dir := "./coverage"
coverage_profile := coverage_dir / "coverage.out"
coverage_html := coverage_dir / "coverage.html"

# Default recipe
default: build

# Build the binary (optimized) and copy to ~/.local/bin
build:
    @echo "Building {{binary_name}}..."
    @mkdir -p {{build_dir}}
    @go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}} {{cmd_dir}}
    @echo "Built {{build_dir}}/{{binary_name}}"
    @mkdir -p ~/.local/bin
    @cp {{build_dir}}/{{binary_name}} ~/.local/bin/
    @echo "Copied to ~/.local/bin/{{binary_name}}"

# Build for development (no optimizations)
build-dev:
    @echo "Building {{binary_name}} for development..."
    @go build -ldflags "{{dev_ldflags}}" -o {{binary_name}} {{cmd_dir}}

# Build static binary (CGO_ENABLED=0)
build-static:
    @echo "Building static {{binary_name}}..."
    @mkdir -p {{build_dir}}
    @CGO_ENABLED=0 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-static {{cmd_dir}}

# Install the binary to $GOPATH/bin
install: build
    @echo "Installing {{binary_name}}..."
    @go install {{build_flags}} -ldflags "{{ldflags}}" {{cmd_dir}}

# Remove the binary from $GOPATH/bin
uninstall:
    @echo "Uninstalling {{binary_name}}..."
    @rm -f $(go env GOPATH)/bin/{{binary_name}}

# Build and run in development mode
dev: build-dev
    @echo "Running {{binary_name}} in development mode..."
    @./{{binary_name}} --help

# Quick build and run with arguments (use: just run "version")
run *ARGS: build-dev
    @./{{binary_name}} {{ARGS}}

# Watch for changes and rebuild (requires entr)
watch:
    @echo "Watching for changes... (press Ctrl+C to stop)"
    @find . -name "*.go" | entr -r just build-dev

# Run unit tests
test:
    @echo "Running unit tests..."
    @go test -v {{test_packages}}

# Run unit tests (short mode)
test-short:
    @echo "Running unit tests (short)..."
    @go test -short -v {{test_packages}}

# Run tests with race detector
test-race:
    @echo "Running tests with race detector..."
    @go test -race -v {{test_packages}}

# Run tests with coverage
test-cover:
    @echo "Running tests with coverage..."
    @mkdir -p {{coverage_dir}}
    @go test -coverprofile={{coverage_profile}} {{test_packages}}
    @go tool cover -html={{coverage_profile}} -o {{coverage_html}}
    @echo "Coverage report generated: {{coverage_html}}"
    @go tool cover -func={{coverage_profile}} | tail -1

# Run all tests with coverage (including integration)
test-cover-all:
    @echo "Running all tests with coverage..."
    @mkdir -p {{coverage_dir}}
    @go test -coverprofile={{coverage_dir}}/unit.out {{test_packages}}
    @go test -coverprofile={{coverage_dir}}/integration.out {{integration_test_packages}}
    @echo "mode: set" > {{coverage_profile}}
    @cat {{coverage_dir}}/unit.out {{coverage_dir}}/integration.out | grep -v mode: >> {{coverage_profile}} || true
    @go tool cover -html={{coverage_profile}} -o {{coverage_html}}
    @go tool cover -func={{coverage_profile}} | tail -1

# Run integration tests
test-integration:
    @echo "Running integration tests..."
    @go test -v -tags=integration {{integration_test_packages}}

# Run CLI tests using testscript
test-cli:
    @echo "Running CLI tests..."
    @go test -v ./test/cli/...

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    @go test -bench=. -benchmem {{pkg_list}}

# Run benchmarks and save for comparison
bench-compare:
    @echo "Running benchmarks (comparison mode)..."
    @go test -bench=. -benchmem {{pkg_list}} | tee bench.out

# Format Go code
fmt:
    @echo "Formatting code..."
    @go fmt {{pkg_list}}

# Check if code is formatted
fmt-check:
    #!/usr/bin/env bash
    echo "Checking code formatting..."
    unformatted=$(go fmt {{pkg_list}})
    if [ -n "$unformatted" ]; then
        echo "Code not formatted. Run 'just fmt' to fix."
        echo "$unformatted"
        exit 1
    fi

# Run go vet
vet:
    @echo "Running go vet..."
    @go vet {{pkg_list}}

# Run linter (go vet)
lint:
    @echo "Running go vet..."
    @go vet {{pkg_list}}

# Run security checks
security:
    @echo "Running security checks..."
    @echo "Security tools not available. Use go vet for basic checks."
    @go vet {{pkg_list}}

# Check for known vulnerabilities
vulnerability-check:
    @echo "Checking for vulnerabilities..."
    @go list -json -deps ./... | nancy sleuth

# Run all checks (format, vet, lint, test)
check: fmt-check vet lint test

# Run all checks including integration tests
check-all: fmt-check vet lint test test-integration

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    @go mod download

# Update dependencies
deps-update:
    @echo "Updating dependencies..."
    @go get -u ./...
    @go mod tidy

# Verify dependencies
deps-verify:
    @echo "Verifying dependencies..."
    @go mod verify

# Audit dependencies for vulnerabilities
deps-audit:
    @echo "Auditing dependencies..."
    @go list -json -deps ./... | audit

# Clean unused dependencies
deps-clean:
    @echo "Cleaning unused dependencies..."
    @go mod tidy

# Clean build artifacts
clean:
    @echo "Cleaning..."
    @rm -rf {{build_dir}}
    @rm -f {{binary_name}}
    @rm -rf {{coverage_dir}}
    @rm -f *.out *.html
    @rm -f bench.out

# Clean module cache
clean-deps:
    @echo "Cleaning module cache..."
    @go clean -modcache

# Clean test cache
clean-test:
    @echo "Cleaning test cache..."
    @go clean -testcache

# Clean everything
clean-all: clean clean-test

# Build for all platforms
release-build:
    @echo "Building for all platforms..."
    @mkdir -p {{build_dir}}
    @GOOS=linux GOARCH=amd64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-amd64 {{cmd_dir}}
    @GOOS=linux GOARCH=arm64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-linux-arm64 {{cmd_dir}}
    @GOOS=darwin GOARCH=amd64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-darwin-amd64 {{cmd_dir}}
    @GOOS=darwin GOARCH=arm64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-darwin-arm64 {{cmd_dir}}
    @GOOS=windows GOARCH=amd64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-windows-amd64.exe {{cmd_dir}}
    @GOOS=windows GOARCH=arm64 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{build_dir}}/{{binary_name}}-windows-arm64.exe {{cmd_dir}}
    @echo "Release binaries built in {{build_dir}}/"

# Package release binaries
release-package: release-build
    #!/usr/bin/env bash
    echo "Packaging release binaries..."
    cd {{build_dir}}
    for binary in {{binary_name}}-*; do
        if [[ $binary == *.exe ]]; then
            zip $binary.zip $binary
        else
            tar -czf $binary.tar.gz $binary
        fi
    done
    echo "Release packages created in {{build_dir}}/"

# Generate checksums for release files
release-checksums: release-package
    @echo "Generating checksums..."
    @cd {{build_dir}} && sha256sum *.tar.gz *.zip > checksums.txt
    @echo "Checksums generated in {{build_dir}}/checksums.txt"

# Build Docker image
docker-build:
    @echo "Building Docker image..."
    @docker build -t bt:{{version}} .

# Run Docker container
docker-run:
    @echo "Running Docker container..."
    @docker run --rm -it bt:{{version}}

# Build multi-arch Docker image for release
docker-release:
    @echo "Building multi-arch Docker image..."
    @docker buildx build --platform linux/amd64,linux/arm64 -t bt:{{version}} .

# Create a git tag with current version
git-tag:
    @echo "Creating git tag {{version}}..."
    @git tag -a {{version}} -m "Release {{version}}"
    @echo "Tag created. Push with: git push origin {{version}}"

# Show build information
info:
    @echo "Build Information:"
    @echo "  Binary Name:     {{binary_name}}"
    @echo "  Version:         {{version}}"
    @echo "  Commit:          {{commit}}"
    @echo "  Date:            {{date}}"
    @echo "  Go Version:      {{goversion}}"
    @echo "  Build Dir:       {{build_dir}}"
    @echo "  Test Packages:   $(echo {{test_packages}} | wc -w)"
    @echo "  Source Files:    $(find . -name '*.go' | grep -v vendor | wc -l)"

# Quick test and build
quick-test: test build

# Quick format and test
quick-check: fmt test

# Show project status
status:
    @echo "Project Status:"
    @echo "  Go version:      $(go version)"
    @echo "  Module:          $(head -1 go.mod)"
    @echo "  Dependencies:    $(go list -m all | wc -l) modules"
    @echo "  Source files:    $(find . -name '*.go' | grep -v vendor | wc -l) files"
    @echo "  Test files:      $(find . -name '*_test.go' | wc -l) files"
    @echo "  Git status:      $(git status --porcelain | wc -l) changes"
    @echo "  Latest commit:   $(git log -1 --oneline)"

# Setup development environment
setup-dev:
    @echo "Setting up development environment..."
    @go mod download
    @echo "Installing development tools..."
    @go install github.com/sonatypeoss/nancy@latest
    @echo "Setup complete!"

# Setup git hooks for development
setup-hooks:
    @echo "Setting up git hooks..."
    @mkdir -p .git/hooks
    @echo '#!/bin/sh\njust fmt-check && just vet' > .git/hooks/pre-commit
    @chmod +x .git/hooks/pre-commit
    @echo "Git hooks installed!"

# Run CPU profiling
profile-cpu:
    @echo "Running CPU profiling..."
    @go test -cpuprofile=cpu.prof -bench=. {{pkg_list}}
    @echo "CPU profile saved to cpu.prof. View with: go tool pprof cpu.prof"

# Run memory profiling
profile-mem:
    @echo "Running memory profiling..."
    @go test -memprofile=mem.prof -bench=. {{pkg_list}}
    @echo "Memory profile saved to mem.prof. View with: go tool pprof mem.prof"

# Clear build and test caches
cache-clear:
    @echo "Clearing build and test caches..."
    @go clean -cache -testcache -modcache

# Run tests in CI environment
ci-test:
    @echo "Running CI tests..."
    @go test -v -coverprofile=coverage.out {{test_packages}}
    @go tool cover -func=coverage.out

# Build in CI environment
ci-build:
    @echo "Building in CI environment..."
    @CGO_ENABLED=0 go build {{build_flags}} -ldflags "{{ldflags}}" -o {{binary_name}} {{cmd_dir}}

# Run linting in CI environment
ci-lint:
    @echo "Running CI linting..."
    @go vet {{pkg_list}}

# Validate configuration files
config-validate:
    @echo "Validating configuration..."
    @go run {{cmd_dir}} config validate

# Generate example configuration
config-example:
    @echo "Generating example configuration..."
    @go run {{cmd_dir}} config example > config.example.yaml

# Run API integration tests
test-api:
    @echo "Running API tests..."
    @go test -v -tags=api ./pkg/api/...

# Run authentication tests
test-auth:
    @echo "Running authentication tests..."
    @go test -v -tags=auth ./pkg/auth/...

# Run performance tests
perf-test:
    @echo "Running performance tests..."
    @go test -v -tags=perf -timeout=30m ./test/performance/...

# Run load tests
load-test:
    @echo "Running load tests..."
    @go test -v -tags=load -timeout=60m ./test/load/...
