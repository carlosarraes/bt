# Bitbucket CLI (bt)

> A 1:1 replacement for GitHub CLI that works with Bitbucket Cloud

Work seamlessly with Bitbucket from the command line. `bt` provides the same command structure and user experience as GitHub CLI (`gh`), but for Bitbucket repositories, pull requests, and pipelines. Now with **AI-powered PR descriptions** for intelligent development workflows.

## Features

- **🔄 Drop-in replacement** for GitHub CLI - same commands, same patterns
- **🔐 Secure API token authentication** - Uses Atlassian API tokens for secure access
- **🤖 AI-powered PR descriptions** - OpenAI o4-mini integration with structured output and 24-hour caching
- **⚙️ Advanced configuration** - Comprehensive CLI config management with validation
- **📊 Complete pull request workflow** - Create, review, merge, edit, comment, and manage PRs
- **🚀 Pipeline debugging** - 5x faster error diagnosis with smart log analysis
- **🌐 Repository operations** - Clone, create, fork, and manage repositories
- **📱 Cross-platform** - Works on macOS, Linux, and Windows
- **🤖 LLM-friendly** - Structured output perfect for AI agents and automation

## Installation

### Quick Install Script (Recommended)
```bash
curl -sSf https://raw.githubusercontent.com/carlosarraes/bt/main/install.sh | sh
```

### Manual Download
Download the latest binary from the [releases page](https://github.com/carlosarraes/bt/releases) and add it to your PATH.

### Go Install
```bash
go install github.com/carlosarraes/bt/cmd/bt@latest
```

## Quick Start

### 1. Authentication
```bash
# Authenticate with Bitbucket (choose your preferred method)
bt auth login

# Check authentication status
bt auth status
```

### 2. Work with pull requests
```bash
# List pull requests
bt pr list

# Create PR with AI-generated description
bt pr create --ai --template portuguese

# View a specific pull request  
bt pr view 123

# Review and approve PRs
bt pr review 123 --approve
```

### 3. Monitor pipelines
```bash
# List recent pipeline runs
bt run list

# View pipeline details and logs
bt run view 1234567890abcdef

# Watch a running pipeline
bt run watch 1234567890abcdef
```

## Authentication

### API Token Authentication

`bt` uses secure Atlassian API tokens for authentication:

1. Go to [Atlassian Account Security](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Create a new API token with a descriptive label
3. Run `bt auth login` - it will guide you through the setup

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

### Alternative Environment Variables (Backward compatibility)

```bash
# Legacy format (still supported)
export BITBUCKET_USERNAME="your.email@company.com"
export BITBUCKET_PASSWORD="your_api_token_here"
```

## Usage

### Repository Commands

```bash
# Repository commands are not yet implemented
# Coming soon: clone, create, fork, list, view
```

### Pull Request Commands

**✨ Complete PR workflow with AI-powered descriptions**

```bash
# Create pull requests with AI assistance (OpenAI o4-mini)
bt pr create --ai                          # AI-generated description (Portuguese)
bt pr create --ai --template english      # English AI description
bt pr create --ai --jira context.md       # Include JIRA context for better descriptions
bt pr create --title "Fix bug" --body "Manual description"  # Traditional creation

# List and filter pull requests
bt pr list                                 # Current repository
bt pr list --state merged                 # Filter by state
bt pr list --author @me                   # Your PRs
bt pr list myworkspace/other-repo          # Different repository

# View and inspect pull requests
bt pr view 42                             # View PR details
bt pr view 42 --web                      # Open PR in browser
bt pr view 42 --comments                 # Show PR comments
bt pr files 42                           # List changed files
bt pr diff 42                            # Show diff
bt pr diff 42 --name-only                # Show changed files only
bt pr diff 42 | delta                    # Enhanced viewing with delta

# Review and collaborate
bt pr review 42 --approve                  # Approve a PR
bt pr review 42 --request-changes -b "Please fix tests"  # Request changes
bt pr review 42 --comment -b "LGTM!"      # Add a comment
bt pr comment 42 -b "Great work!"         # Add general comment

# Development workflow
bt pr checkout 42                          # Switch to PR branch locally
bt pr edit 42 --title "New title"         # Edit PR metadata
bt pr status                              # Your PR activity dashboard
bt pr checks 42                           # View CI/build status

# Lifecycle management
bt pr merge 42                            # Merge PR
bt pr merge 42 --squash --delete-branch   # Squash merge with cleanup
bt pr close 42                            # Close PR
bt pr reopen 42                           # Reopen closed PR
bt pr ready 42                            # Mark draft as ready

# Advanced operations
bt pr update-branch 42                    # Sync with target branch
bt pr lock 42 --reason spam               # Lock conversation (admin)
bt pr unlock 42                           # Unlock conversation
```

### Pipeline Commands

**✨ New: Smart log viewing with instant failed step detection + Real-time monitoring**

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

# Real-time monitoring with streaming logs
bt run watch 1234                  # Watch live execution with continuous log streaming  
                                   # ✨ Streams new log lines as they appear (3s updates)
                                   # ✨ Shows rolling buffer of last 10 lines
                                   # ✨ Clean display without timestamp conflicts
                                   # ✨ Live progress indicators and status
                                   # ✨ Automatic completion detection

# Pipeline management  
bt run cancel 1234                 # Cancel running pipeline with confirmation
bt run rerun 1234                  # ⚠️ Rerun pipeline (currently has issues with PR pipelines - WIP)
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


## Environment Variables

```bash
# Authentication (Recommended)
export BITBUCKET_EMAIL="your.email@company.com"      # Your Atlassian account email
export BITBUCKET_API_TOKEN="your_api_token_here"     # API token from Atlassian

# Alternative format (still supported)
export BITBUCKET_USERNAME="your.email@company.com"   # Your Atlassian account email
export BITBUCKET_PASSWORD="your_api_token_here"      # API token from Atlassian

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
# Set default workspace
bt config set auth.default_workspace personal-workspace
bt config set auth.default_workspace company-workspace

# Operations on different workspaces (use --workspace flag)
bt pr list --workspace company-workspace
bt run list --workspace company-workspace
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
# Complete 1:1 compatibility for implemented commands
gh auth login        →  bt auth login
gh auth status       →  bt auth status
gh pr list           →  bt pr list
gh pr create         →  bt pr create      # ✨ Enhanced with AI descriptions
gh pr view           →  bt pr view
gh pr diff           →  bt pr diff
gh pr review         →  bt pr review
gh pr comment        →  bt pr comment
gh pr checkout       →  bt pr checkout
gh pr merge          →  bt pr merge
gh pr close          →  bt pr close
gh pr edit           →  bt pr edit
gh pr status         →  bt pr status
gh run list          →  bt run list
gh run view          →  bt run view
gh run watch         →  bt run watch
gh run cancel        →  bt run cancel

# Coming soon: repo clone, api, browse
```

## Troubleshooting

### Authentication Issues

```bash
# Check authentication status
bt auth status

# Re-authenticate with new token
bt auth logout
bt auth login
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


## Comparison with GitHub CLI

| Feature | GitHub CLI | Bitbucket CLI | Notes |
|---------|------------|---------------|-------|
| Repository management | ✅ | 🚧 | Coming soon |
| Pull requests | ✅ | ✅ | **Complete workflow implemented** + AI descriptions |
| CI/CD (Actions/Pipelines) | ✅ | ✅ | **Enhanced** - 5x faster debugging |
| AI-powered descriptions | ❌ | ✅ | **bt CLI innovation** - intelligent PR descriptions |
| Issues | ✅ | ❌ | Would depend on Jira integration |
| Releases | ✅ | 🚧 | Will map to tags |
| Gists | ✅ | 🚧 | Will map to snippets |
| Organizations | ✅ | ➡️ | Maps to workspaces |
| Authentication | ✅ | ✅ | API tokens supported |
| API Access | ✅ | 🚧 | Coming soon |
| Browser integration | ✅ | 🚧 | Coming soon |

## FAQ

**Q: Why create another CLI when there's already a Bitbucket CLI?**  
A: The official Atlassian CLI doesn't provide the same developer experience as GitHub CLI. `bt` maintains identical command structure and patterns, making it a true drop-in replacement.

**Q: Does this work with Bitbucket Server/Data Center?**  
A: Currently, `bt` is designed for Bitbucket Cloud. Bitbucket Server support may be added in the future.

**Q: Can I use this with AI coding assistants?**  
A: Absolutely! `bt` is designed to work seamlessly with AI agents. The structured JSON output, identical command patterns, and built-in AI-powered PR descriptions make it perfect for LLM-based automation and intelligent development workflows.

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