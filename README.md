# Bitbucket CLI (bt)

> A 1:1 replacement for GitHub CLI that works with Bitbucket Cloud

Work seamlessly with Bitbucket from the command line. `bt` provides the same command structure and user experience as GitHub CLI (`gh`), but for Bitbucket repositories, pull requests, and pipelines.

## Features

- **🔄 Drop-in replacement** for GitHub CLI - same commands, same patterns
- **🔐 Multiple authentication methods** - App passwords, OAuth 2.0, Access tokens
- **⚙️ Advanced configuration** - Comprehensive CLI config management with validation
- **📊 Pull request management** - Create, review, merge, and manage PRs
- **🚀 Pipeline integration** - View runs, logs, and trigger builds
- **🌐 Repository operations** - Clone, create, fork, and manage repositories
- **📱 Cross-platform** - Works on macOS, Linux, and Windows
- **🤖 LLM-friendly** - Structured output perfect for AI agents and automation

## Installation

### Go Install
```bash
go install github.com/carraes/bt/cmd/bt@latest
```

*Binary releases and package managers will be available once we have a working implementation.*

## Quick Start

### 1. Authentication
```bash
# Authenticate with Bitbucket (choose your preferred method)
bt auth login

# Check authentication status
bt auth status
```

### 2. Clone a repository
```bash
# Clone using workspace/repo format
bt repo clone myworkspace/myproject

# Clone using full URL
bt repo clone https://bitbucket.org/myworkspace/myproject
```

### 3. Work with pull requests
```bash
# Create a pull request
bt pr create --title "Add new feature" --body "Description of changes"

# List pull requests
bt pr list

# View a specific pull request
bt pr view 123
```

### 4. Monitor pipelines
```bash
# List recent pipeline runs
bt run list

# View pipeline details and logs
bt run view 1234567890abcdef

# Watch a running pipeline
bt run watch 1234567890abcdef
```

## Authentication

### API Token (Recommended - New Method)

**⚠️ App passwords are being deprecated by Bitbucket. Use API tokens instead.**

1. Go to [Atlassian Account Security](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Create a new API token with a descriptive label
3. Run `bt auth login` - it will guide you through the API token setup

```bash
bt auth login
🚀 Welcome to Bitbucket CLI Authentication
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔑 Authentication uses API tokens (email + token)
📋 Create an API token at: https://id.atlassian.com/manage-profile/security/api-tokens

📧 Atlassian account email: your.email@company.com
🔑 API token (hidden): [paste token]
✅ Authentication successful!
👤 Logged in as: Your Name (your.username)
```

### Environment Variables (Recommended for automation)

```bash
# Set these environment variables for seamless authentication
export BITBUCKET_EMAIL="your.email@company.com"
export BITBUCKET_API_TOKEN="your_api_token_here"

# Now commands work without additional setup
bt run list
bt run view 1234 --log-failed
```

### Legacy Methods (Still Supported)

<details>
<summary>App Password (Being deprecated by Bitbucket)</summary>

```bash
export BITBUCKET_USERNAME=your_username
export BITBUCKET_PASSWORD=your_app_password
```

</details>

<details>
<summary>Access Token (For specific scopes)</summary>

```bash
export BITBUCKET_TOKEN=your_access_token_here
```

</details>

## Usage

### Repository Commands

```bash
# List repositories in your workspace
bt repo list

# List repositories in a specific workspace
bt repo list myworkspace

# Create a new repository
bt repo create myworkspace/new-repo --private

# Fork a repository
bt repo fork myworkspace/original-repo

# View repository details
bt repo view myworkspace/my-repo

# Clone with SSH/HTTPS preference
bt repo clone myworkspace/my-repo --ssh
```

### Pull Request Commands

```bash
# List pull requests
bt pr list                          # Current repository
bt pr list --state merged          # Filter by state
bt pr list --author @me             # Your PRs
bt pr list myworkspace/other-repo   # Different repository

# Create a pull request
bt pr create                        # Interactive mode
bt pr create --title "Fix bug" --body "Description" --draft
bt pr create --base develop --head feature-branch

# View and interact with PRs
bt pr view 42                       # View PR details
bt pr diff 42                       # View PR diff
bt pr checkout 42                   # Checkout PR branch
bt pr checks 42                     # View pipeline status

# Review and merge
bt pr review 42 --approve
bt pr review 42 --comment "Looks good!"
bt pr merge 42 --squash
bt pr close 42

# Comments and updates
bt pr comment 42 --body "Great work!"
bt pr edit 42 --title "New title"
```

### Pipeline Commands (The Killer Feature 🚀)

**✨ New: Smart log viewing with instant failed step detection**

```bash
# List pipeline runs
bt run list                         # Recent runs
bt run list --status failed        # Filter by status
bt run list --branch main          # Filter by branch

# View pipeline details and logs
bt run view 1234                   # Pipeline overview with step details
bt run view 1234 --log-failed      # Show last 100 lines of failed steps (⚡ FAST)
bt run view 1234 --log-failed --full-output  # Show complete failed step logs
bt run view 1234 --log             # Show logs for all steps
bt run view 1234 --tests           # Show test results and failures
bt run view 1234 --step "Run Tests" # Show logs for specific step

# Real-time monitoring
bt run watch 1234                  # Watch live execution with real-time updates

# Pipeline management  
bt run cancel 1234                 # Cancel running pipeline with confirmation
```

**🎯 Smart Log Analysis - Get to the error instantly:**
```bash
# Quick debugging workflow
bt run list --status failed        # Find failed pipelines
bt run view 3808 --log-failed      # See failure immediately (last 100 lines)
bt run view 3808 --log-failed --full-output  # Get full context if needed

# Example output:
# ✅ Successfully retrieved logs for step: Run Tests
# Showing last 100 lines (use --full-output for complete logs)
# ================================================================================
# FAILED (failures=4, skipped=138)
# AssertionError: None != '001'
# ================================================================================
```

### API Access

Make direct API calls to Bitbucket:

```bash
# Get repository information
bt api repositories/myworkspace/myrepo

# List pull requests with custom filtering
bt api repositories/myworkspace/myrepo/pullrequests --query "state=\"OPEN\""

# Trigger a pipeline with variables
bt api repositories/myworkspace/myrepo/pipelines \
  --method POST \
  --field target.ref_name=main \
  --field variables.ENV=staging
```

### Configuration

```bash
# View current configuration
bt config list

# View specific configuration value
bt config get auth.default_workspace

# Set default workspace
bt config set auth.default_workspace mycompany

# Set output format preference
bt config set defaults.output_format json

# Set API timeout
bt config set api.timeout 45s

# Remove configuration value
bt config unset auth.default_workspace

# JSON output for automation
bt config list --output json
```

### Browser Integration

```bash
# Open repository in browser
bt browse

# Open specific PR
bt browse pr/123

# Open pipeline run
bt browse run/abc123def456

# Open any path
bt browse /projects/PROJ/repos/myrepo/pull-requests
```

## Environment Variables

```bash
# Authentication (Recommended)
export BITBUCKET_EMAIL="your.email@company.com"      # Your Atlassian account email
export BITBUCKET_API_TOKEN="your_api_token_here"     # API token from Atlassian

# Legacy authentication (still supported)
export BITBUCKET_USERNAME=your_username               # For app passwords
export BITBUCKET_PASSWORD=your_app_password          # App password
export BITBUCKET_TOKEN=your_access_token             # Access tokens

# Configuration
export BITBUCKET_HOST=https://bitbucket.org
export BT_CONFIG_DIR=~/.config/bt
export BT_PAGER=less
export BT_EDITOR=vim

# Output preferences
export BT_OUTPUT_FORMAT=table  # table, json, yaml
export BT_NO_COLOR=1          # Disable colors
export BT_VERBOSE=1           # Enable verbose output
```

## Output Formats

### Table (Default)
```bash
bt pr list
# ┌────────┬─────────────────────────┬────────┬────────────┐
# │ NUMBER │ TITLE                   │ BRANCH │ STATUS     │
# ├────────┼─────────────────────────┼────────┼────────────┤
# │ 42     │ Add new authentication  │ auth   │ OPEN       │
# │ 41     │ Fix pipeline bug        │ fix    │ MERGED     │
# └────────┴─────────────────────────┴────────┴────────────┘
```

### JSON (Great for scripting and LLMs)
```bash
bt pr list --output json
# [
#   {
#     "id": 42,
#     "title": "Add new authentication",
#     "source": {"branch": {"name": "auth"}},
#     "state": "OPEN",
#     "created_on": "2024-01-15T10:30:00Z"
#   }
# ]

# Perfect for AI agents and automation
bt run logs 123 --output json | jq '.steps[] | select(.state == "FAILED") | .name'
```

### Custom Templates
```bash
# Use Go templates for custom formatting
bt pr list --template "{{range .}}{{.id}}: {{.title}} ({{.state}}){{end}}"
```

## Advanced Features

### Pipeline Log Analysis (🤖 LLM Integration)

```bash
# Get structured error output for AI analysis
bt run view failed-pipeline-id --log-failed --output json > errors.json

# Quick debugging workflow
bt run view 3808 --log-failed           # Last 100 lines (instant debugging)
bt run view 3808 --log-failed --full-output  # Complete logs when needed

# Test-specific analysis
bt run view 3808 --tests                # Show test results and failures

# Step-specific debugging
bt run view 3808 --step "Run Tests" --log  # Logs for specific step
```

### Workspace Management

```bash
# Switch between workspaces
bt config set default_workspace personal-workspace
bt config set default_workspace company-workspace

# Operations on different workspaces
bt repo list company-workspace
bt pr list --workspace company-workspace
```

### Advanced Configuration

```bash
# Advanced configuration management
bt config list                           # View all configuration settings
bt config get auth.method               # Get specific setting
bt config set auth.default_workspace myorg  # Set workspace preference  
bt config set api.timeout 60s           # Increase API timeout
bt config unset auth.default_workspace  # Reset to default

# Configuration supports nested keys:
# - auth.method, auth.default_workspace
# - api.base_url, api.timeout  
# - defaults.output_format
# - version

# Export configuration for backup
bt config list --output yaml > bt-config-backup.yml
```

## GitHub CLI Migration

If you're coming from GitHub CLI, `bt` commands work identically:

```bash
# These commands work the same way
gh pr list        →  bt pr list
gh repo clone     →  bt repo clone
gh run list       →  bt run list
gh api           →  bt api

# Automatic command translation (planned feature)
gh pr create --title "Fix bug"
# → Did you mean: bt pr create --title "Fix bug"?
```

## Troubleshooting

### Authentication Issues

```bash
# Check authentication status
bt auth status

# Refresh expired tokens
bt auth refresh

# Re-authenticate
bt auth logout
bt auth login
```

### API Rate Limiting

```bash
# Check rate limit status
bt api user --include-rate-limit

# Use personal access token for higher limits
export BITBUCKET_TOKEN=your_personal_access_token
```

### Common Issues

**"Repository not found"**
- Ensure you have access to the repository
- Check if the workspace/repository name is correct
- Verify authentication scope includes repository access

**"Pipeline not found"**
- Pipelines must be enabled for the repository
- Check if `bitbucket-pipelines.yml` exists
- Verify authentication scope includes pipeline access

**"Command not recognized"**
- Check if you're using the correct command syntax
- Use `bt --help` to see available commands
- Ensure you're in a git repository for repo-specific commands

## Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/carraes/bt.git
cd bt

# Automated development environment setup
make setup-dev
# This will:
# - Install all required development tools (golangci-lint, gosec, etc.)
# - Download dependencies and verify them
# - Set up Git hooks for code quality
# - Configure IDE settings (VS Code)

# Alternative manual setup
go mod download          # Install dependencies
make deps-verify         # Verify dependencies
make test               # Run tests
make lint               # Run linter
make build              # Build binary

# Development workflow commands
make help               # Show all available commands
make dev                # Build and run in development mode
make watch              # Watch for changes and rebuild
make test-cover         # Run tests with coverage
make security           # Run security scans

# Code quality checks
make fmt                # Format code
make check              # Run all quality checks (fmt, vet, lint, test)
make clean              # Clean build artifacts
```

### Build System Features

The enhanced build system provides comprehensive development support:

- **50+ Make targets** for build, test, security, and performance
- **Multi-platform builds** (Linux, macOS, Windows on AMD64/ARM64)
- **Security scanning** with gosec and vulnerability checking
- **Performance profiling** and benchmark comparison
- **Automated CI/CD** with GitHub Actions
- **Code quality** with golangci-lint (40+ enabled linters)
- **Development tools** installation and Git hooks setup

### Architecture

See [SPEC.md](SPEC.md) for detailed technical specifications and architecture decisions.

## Comparison with GitHub CLI

| Feature | GitHub CLI | Bitbucket CLI | Notes |
|---------|------------|---------------|-------|
| Repository management | ✅ | ✅ | Full parity |
| Pull requests | ✅ | ✅ | Full parity |
| CI/CD (Actions/Pipelines) | ✅ | ✅ | Pipeline logs are enhanced |
| Issues | ✅ | ⚠️ | Depends on Jira integration |
| Releases | ✅ | ➡️ | Maps to tags |
| Gists | ✅ | ➡️ | Maps to snippets |
| Organizations | ✅ | ➡️ | Maps to workspaces |
| Authentication | ✅ | ✅ | Multiple methods supported |
| API Access | ✅ | ✅ | Full Bitbucket API access |
| Browser integration | ✅ | ✅ | Full parity |

## FAQ

**Q: Why create another CLI when there's already a Bitbucket CLI?**  
A: The official Atlassian CLI doesn't provide the same developer experience as GitHub CLI. `bt` maintains identical command structure and patterns, making it a true drop-in replacement.

**Q: Does this work with Bitbucket Server/Data Center?**  
A: Currently, `bt` is designed for Bitbucket Cloud. Bitbucket Server support may be added in the future.

**Q: Can I use this with AI coding assistants?**  
A: Absolutely! `bt` is designed to work seamlessly with AI agents. The structured JSON output and identical command patterns make it perfect for LLM-based automation.

**Q: Is this an official Atlassian project?**  
A: No, this is an independent open-source project. It uses Bitbucket's public APIs.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Inspired by the excellent [GitHub CLI](https://github.com/cli/cli)
- Built with [Kong](https://github.com/alecthomas/kong) CLI framework
- Configuration powered by [Koanf](https://github.com/knadh/koanf)

---

**Made with ❤️ for developers who want the same great CLI experience on Bitbucket**

*Star this project if you find it useful! ⭐*