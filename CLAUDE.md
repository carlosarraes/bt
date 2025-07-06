# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`bt` is a 1:1 replacement for GitHub CLI that works with Bitbucket Cloud. It provides identical command structure and user experience as GitHub CLI (`gh`) but targets Bitbucket's REST API v2. The core value proposition is 5x faster pipeline debugging through intelligent log analysis and error extraction.

## Development Commands

### Essential Commands

```bash
# Build and development
make build                    # Build optimized binary
make build-dev               # Build for development (no optimization)
make install                 # Install to $GOPATH/bin
make dev                     # Build and run with help

# Testing
make test                    # Unit tests only
make test-cover             # Tests with coverage report
make test-integration       # Integration tests (requires Bitbucket auth)
make test-race              # Race condition detection
make test-cli               # CLI command testing

# Code quality
make fmt                    # Format code
make vet                    # Run go vet
make check                  # All quality checks (fmt, vet, test)

# Single test execution
go test -v ./pkg/api -run TestClient
go test -v ./pkg/cmd/pr -run TestViewCmd_ParsePRID
go test -v ./test/integration -run TestPullRequestAPI
```

### Development Environment Setup

```bash
make deps                   # Download dependencies
make clean                  # Clean build artifacts
```

## Architecture Overview

### Core Package Structure

- **`pkg/api/`** - Bitbucket API v2 client with authentication integration

  - `client.go` - HTTP client with rate limiting and error handling
  - `pipelines.go` - Pipeline API methods
  - `pullrequests.go` - Pull request API methods
  - `types.go` - Common API response types
  - Real API integration only - no mocks

- **`pkg/cmd/`** - Command implementations organized by command group

  - `commands.go` - Kong CLI framework command definitions
  - `auth/` - Authentication commands (login, logout, status, refresh)
  - `run/` - Pipeline commands (list, view, logs, watch, cancel)
  - `pr/` - Pull request commands (list, view, diff, review, etc.)
  - `config/` - Configuration management commands

- **`pkg/auth/`** - Multi-method authentication system

  - `manager.go` - AuthManager interface and factory
  - `app_password.go`, `oauth.go`, `access_token.go` - Auth implementations
  - `storage.go` - Encrypted credential storage using OS keyring

- **`pkg/config/`** - Configuration management using Koanf
  - `loader.go` - Configuration loading with environment variables
  - `config.go` - Configuration structure and validation
  - Supports nested keys with dot notation

### Command Flow Architecture

1. **CLI Entry Point** (`cmd/bt/main.go`)

   - Kong CLI framework for command parsing
   - Global flag handling (--llm, --version, --no-color)
   - Context setup with configuration

2. **Command Execution** (`pkg/cmd/commands.go`)

   - Command structs bridge Kong CLI to implementation packages
   - Context passing for global flags and configuration
   - Consistent error handling patterns

3. **Implementation Layer** (`pkg/cmd/{auth,run,pr,config}/`)
   - Actual command logic and business rules
   - API client integration
   - Output formatting and user feedback

### API Integration Architecture

- **Unified Client** (`pkg/api/client.go`)

  - Single HTTP client with authentication injection
  - Rate limiting with exponential backoff
  - Request/response logging and error handling
  - Service composition (Pipelines, PullRequests)

- **Service Pattern**
  - Each API domain has its own service (PipelineService, PullRequestService)
  - Services attached to main client for unified access
  - Real API integration from day one - no mock data

### Authentication Architecture

- **Multi-method Support**: App passwords, OAuth 2.0, Access tokens
- **Environment Priority**: Environment variables override stored credentials
- **Secure Storage**: AES-GCM encryption with PBKDF2 key derivation
- **Token Management**: Automatic refresh for OAuth flows

## Key Implementation Patterns

### Command Structure Pattern

All commands follow this pattern:

```go
type CommandCmd struct {
    // Command-specific flags
    Flag1 string `help:"Description"`
    Flag2 bool   `help:"Description"`

    // Common flags (added by command registration)
    Output     string `short:"o" help:"Output format"`
    Workspace  string `help:"Bitbucket workspace"`
    Repository string `help:"Repository name"`
}

func (c *CommandCmd) Run(ctx context.Context) error {
    // 1. Create context with auth and config
    cmdCtx, err := NewCommandContext(ctx, c.Output, c.NoColor)

    // 2. Validate inputs and context
    if err := cmdCtx.ValidateWorkspaceAndRepo(); err != nil {
        return err
    }

    // 3. Execute business logic
    result, err := cmdCtx.Client.Service.Method(ctx, params)

    // 4. Format and display output
    return c.formatOutput(cmdCtx, result)
}
```

### API Client Pattern

```go
// Service composition in main client
type Client struct {
    httpClient   *http.Client
    authManager  auth.AuthManager

    // Services
    Pipelines    *PipelineService
    PullRequests *PullRequestService
}

// Service methods use consistent patterns
func (s *Service) Method(ctx context.Context, workspace, repo string, opts *Options) (*Result, error) {
    // 1. Input validation
    // 2. Build API endpoint
    // 3. Make authenticated request
    // 4. Handle errors and parse response
    // 5. Return typed result
}
```

### Output Formatting Pattern

```go
// Multi-format output support
func (c *Command) formatOutput(ctx *Context, data interface{}) error {
    switch c.Output {
    case "table":
        return c.formatTable(ctx, data)
    case "json":
        return ctx.Formatter.Format(data)  // Uses pkg/output
    case "yaml":
        return ctx.Formatter.Format(data)
    }
}
```

## Testing Strategy

### Real API Integration Focus

- **No Mocks**: All tests use real Bitbucket API v2
- **Integration Tests**: Located in `test/integration/`
- **Auth Required**: Set environment variables for integration tests
- **Performance Targets**: <100ms startup, <500ms API response

### Test Categories

- **Unit Tests**: Alongside source code (`*_test.go`)
- **Integration Tests**: `test/integration/` (requires auth)
- **CLI Tests**: `test/cli/` (command execution)
- **Performance Tests**: `test/performance/` (benchmarks)

### Test Execution

```bash
# Unit tests only (fast)
make test

# Integration tests (requires Bitbucket auth)
export BT_INTEGRATION_TESTS=1
export BT_TEST_USERNAME="username"
export BT_TEST_APP_PASSWORD="password"
export BT_TEST_WORKSPACE="workspace"
make test-integration

# Single test file
go test -v ./pkg/cmd/pr -run TestViewCmd
```

## GitHub CLI Compatibility

The project maintains strict 1:1 compatibility with GitHub CLI:

- **Identical command structure**: `gh pr list` → `bt pr list`
- **Same flags and options**: `--state`, `--author`, `--output`
- **Matching output formats**: Table, JSON, YAML
- **Error message patterns**: Similar tone and structure

### Command Mapping Examples

```bash
# Authentication
gh auth login       → bt auth login
gh auth status      → bt auth status

# Pull requests
gh pr list          → bt pr list
gh pr view 123      → bt pr view 123
gh pr diff 123      → bt pr diff 123

# Repositories
gh repo list        → bt repo list
gh repo clone       → bt repo clone

# API access
gh api /user        → bt api /user
```

## Configuration System

### Configuration Structure

```yaml
# ~/.config/bt/config.yml
version: 1
auth:
  method: app_password
  default_workspace: myworkspace
api:
  base_url: https://api.bitbucket.org/2.0
  timeout: 30s
defaults:
  output_format: table
```

### Environment Variables

```bash
# Authentication (recommended)
BITBUCKET_EMAIL="email@domain.com"
BITBUCKET_API_TOKEN="token"

# Configuration overrides
BT_CONFIG_DIR="~/.config/bt"
BT_OUTPUT_FORMAT="json"
BT_NO_COLOR="1"
```

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use Kong CLI framework tags for command definitions
- Implement interfaces for testability
- Handle errors with context information
- Use structured logging (avoid fmt.Println in libraries)

### Adding New Commands

1. Add command struct to `pkg/cmd/commands.go`
2. Implement command logic in appropriate `pkg/cmd/{group}/` package
3. Add comprehensive tests (unit + integration)
4. Update help text and documentation
5. Ensure GitHub CLI compatibility

### API Integration

- Always use real Bitbucket API v2 endpoints
- Implement proper error handling with Bitbucket error format
- Add rate limiting and retry logic
- Include comprehensive test coverage
- Document API endpoints and response formats

### Performance Requirements

- Command startup: <100ms
- API response time: <500ms average
- Memory usage: <50MB during normal operation
- Binary size: <20MB compressed

This codebase emphasizes real API integration, GitHub CLI compatibility, and performance-focused pipeline debugging as key differentiators.

