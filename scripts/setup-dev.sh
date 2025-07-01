#!/bin/bash

# Development Environment Setup Script for bt (Bitbucket CLI)
# This script sets up the development environment with all necessary tools and dependencies

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Go version
check_go_version() {
    local required_version="1.24"
    local current_version

    if ! command_exists go; then
        log_error "Go is not installed. Please install Go $required_version or later."
        echo "Visit: https://golang.org/doc/install"
        exit 1
    fi

    current_version=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
    
    if [[ "$(printf '%s\n' "$required_version" "$current_version" | sort -V | head -n1)" != "$required_version" ]]; then
        log_error "Go version $current_version is too old. Please upgrade to Go $required_version or later."
        exit 1
    fi

    log_success "Go version $current_version is compatible"
}

# Install development tools
install_dev_tools() {
    log_info "Installing development tools..."

    # Array of tools to install
    declare -A tools=(
        ["github.com/golangci/golangci-lint/cmd/golangci-lint@latest"]="golangci-lint"
        ["github.com/securecodewarrior/sast-scan/cmd/gosec@latest"]="gosec"
        ["golang.org/x/vuln/cmd/govulncheck@latest"]="govulncheck"
        ["golang.org/x/perf/cmd/benchstat@latest"]="benchstat"
        ["github.com/sonatypeoss/nancy@latest"]="nancy"
    )

    for tool_path in "${!tools[@]}"; do
        tool_name=${tools[$tool_path]}
        
        if ! command_exists "$tool_name"; then
            log_info "Installing $tool_name..."
            if go install "$tool_path"; then
                log_success "$tool_name installed successfully"
            else
                log_warning "Failed to install $tool_name, continuing..."
            fi
        else
            log_info "$tool_name is already installed"
        fi
    done
}

# Setup Git hooks
setup_git_hooks() {
    log_info "Setting up Git hooks..."

    if [ ! -d ".git" ]; then
        log_warning "Not in a Git repository, skipping Git hooks setup"
        return
    fi

    # Create hooks directory if it doesn't exist
    mkdir -p .git/hooks

    # Pre-commit hook
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/sh
# Pre-commit hook for bt project

set -e

echo "Running pre-commit checks..."

# Check formatting
echo "Checking code formatting..."
if ! make fmt-check; then
    echo "Code is not properly formatted. Run 'make fmt' to fix."
    exit 1
fi

# Run vet
echo "Running go vet..."
if ! make vet; then
    echo "go vet found issues. Please fix them."
    exit 1
fi

# Run linter
echo "Running linter..."
if ! make lint; then
    echo "Linter found issues. Please fix them."
    exit 1
fi

# Run tests
echo "Running tests..."
if ! make test-short; then
    echo "Tests failed. Please fix them."
    exit 1
fi

echo "All pre-commit checks passed!"
EOF

    # Make hooks executable
    chmod +x .git/hooks/pre-commit

    log_success "Git hooks installed successfully"
}

# Verify dependencies
verify_dependencies() {
    log_info "Verifying Go module dependencies..."

    # Download dependencies
    go mod download

    # Verify dependencies
    if go mod verify; then
        log_success "All dependencies verified successfully"
    else
        log_error "Dependency verification failed"
        exit 1
    fi

    # Check for security vulnerabilities
    if command_exists govulncheck; then
        log_info "Checking for security vulnerabilities..."
        if govulncheck ./...; then
            log_success "No known vulnerabilities found"
        else
            log_warning "Security vulnerabilities detected. Please review and update dependencies."
        fi
    fi
}

# Setup IDE/Editor configurations
setup_ide_config() {
    log_info "Setting up IDE/Editor configurations..."

    # VS Code settings
    if [ -d ".vscode" ] || command_exists code; then
        mkdir -p .vscode
        
        # Settings
        cat > .vscode/settings.json << 'EOF'
{
    "go.testFlags": ["-v"],
    "go.testTimeout": "30s",
    "go.coverOnSave": false,
    "go.coverOnSingleTest": true,
    "go.lintOnSave": "package",
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.formatTool": "goimports",
    "go.useLanguageServer": true,
    "[go]": {
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    },
    "go.toolsManagement.checkForUpdates": "local",
    "gopls": {
        "ui.completion.usePlaceholders": true,
        "ui.diagnostic.staticcheck": true
    }
}
EOF

        # Extensions recommendations
        cat > .vscode/extensions.json << 'EOF'
{
    "recommendations": [
        "golang.Go",
        "ms-vscode.vscode-json",
        "redhat.vscode-yaml",
        "ms-vscode.makefile-tools",
        "streetsidesoftware.code-spell-checker"
    ]
}
EOF

        log_success "VS Code configuration created"
    fi
}

# Create example configuration files
create_example_configs() {
    log_info "Creating example configuration files..."

    # Example environment file
    if [ ! -f ".env.example" ]; then
        cat > .env.example << 'EOF'
# Bitbucket API Configuration
BITBUCKET_USERNAME=your-username
BITBUCKET_APP_PASSWORD=your-app-password
BITBUCKET_WORKSPACE=your-workspace

# OAuth Configuration (optional)
BITBUCKET_CLIENT_ID=your-client-id
BITBUCKET_CLIENT_SECRET=your-client-secret

# Development Settings
BT_LOG_LEVEL=debug
BT_CONFIG_FILE=~/.bt/config.yaml
EOF
        log_success "Created .env.example file"
    fi

    # Example bt configuration
    mkdir -p examples
    if [ ! -f "examples/config.yaml" ]; then
        cat > examples/config.yaml << 'EOF'
# bt (Bitbucket CLI) Configuration Example
api:
  base_url: "https://api.bitbucket.org/2.0"
  timeout: 30s
  retry_attempts: 3

auth:
  method: "app_password"  # or "oauth"
  username: ""
  app_password: ""

defaults:
  workspace: ""
  output_format: "table"  # table, json, yaml
  paging: true
  per_page: 50

display:
  colors: true
  unicode: true
  timestamps: "relative"  # relative, absolute, none

behavior:
  confirm_destructive: true
  auto_update_check: true
  telemetry: false
EOF
        log_success "Created examples/config.yaml file"
    fi
}

# Run tests to verify setup
run_setup_tests() {
    log_info "Running setup verification tests..."

    # Build the project
    if make build; then
        log_success "Project builds successfully"
    else
        log_error "Project build failed"
        exit 1
    fi

    # Run tests
    if make test-short; then
        log_success "Tests pass successfully"
    else
        log_error "Tests failed"
        exit 1
    fi

    # Run linter
    if make lint; then
        log_success "Code passes linting"
    else
        log_warning "Linting issues found. Run 'make lint' to see details."
    fi
}

# Display setup summary
display_summary() {
    echo
    log_success "Development environment setup completed!"
    echo
    log_info "Summary of installed tools:"
    
    declare -a tools=("go" "golangci-lint" "gosec" "govulncheck" "benchstat")
    for tool in "${tools[@]}"; do
        if command_exists "$tool"; then
            version=$($tool version 2>/dev/null | head -1 || echo "unknown")
            echo "  ✓ $tool: $version"
        else
            echo "  ✗ $tool: not found"
        fi
    done

    echo
    log_info "Available Make targets:"
    echo "  make help          - Show all available targets"
    echo "  make build         - Build the binary"
    echo "  make test          - Run tests"
    echo "  make lint          - Run linter"
    echo "  make fmt           - Format code"
    echo "  make setup-dev     - Run this setup again"
    echo

    log_info "Next steps:"
    echo "  1. Copy .env.example to .env and fill in your credentials"
    echo "  2. Run 'make build' to build the project"
    echo "  3. Run 'make test' to verify everything works"
    echo "  4. Start developing! 🚀"
}

# Main function
main() {
    echo "🔧 Setting up development environment for bt (Bitbucket CLI)"
    echo

    check_go_version
    install_dev_tools
    setup_git_hooks
    verify_dependencies
    setup_ide_config
    create_example_configs
    run_setup_tests
    display_summary
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --no-hooks     Skip Git hooks setup"
        echo "  --no-tests     Skip running tests"
        echo "  --minimal      Minimal setup (tools only)"
        exit 0
        ;;
    --no-hooks)
        setup_git_hooks() { log_info "Skipping Git hooks setup"; }
        ;;
    --no-tests)
        run_setup_tests() { log_info "Skipping setup tests"; }
        ;;
    --minimal)
        setup_git_hooks() { log_info "Skipping Git hooks setup"; }
        setup_ide_config() { log_info "Skipping IDE config setup"; }
        create_example_configs() { log_info "Skipping example configs creation"; }
        run_setup_tests() { log_info "Skipping setup tests"; }
        ;;
esac

# Run main function
main