# Bitbucket CLI (bt)

> A 1:1 replacement for GitHub CLI that works with Bitbucket Cloud

Work seamlessly with Bitbucket from the command line. `bt` provides the same command structure and user experience as GitHub CLI (`gh`), but for Bitbucket repositories, pull requests, and pipelines.

## Features

- **🔄 Drop-in replacement** for GitHub CLI - same commands, same patterns
- **🔐 Multiple authentication methods** - App passwords, OAuth 2.0, Access tokens
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

### App Password (Recommended for personal use)

1. Go to Bitbucket Settings → Personal Settings → App Passwords
2. Create a new app password with required permissions:
   - **Repositories**: Read, Write
   - **Pull requests**: Read, Write
   - **Pipelines**: Read
3. Run `bt auth login` and choose "App Password"

```bash
bt auth login
? How would you like to authenticate? App Password
? Bitbucket username: yourusername
? App password: [hidden]
✓ Authentication successful
```

### OAuth 2.0 (Recommended for team use)

1. Create an OAuth consumer in your Bitbucket workspace
2. Run `bt auth login` and choose "OAuth 2.0"
3. Complete the browser authentication flow

```bash
bt auth login
? How would you like to authenticate? OAuth 2.0
✓ Opening browser for authentication...
✓ Authentication successful
```

### Access Token

For repository, project, or workspace-scoped access tokens:

```bash
# Set via environment variable
export BITBUCKET_TOKEN=your_access_token_here

# Or authenticate interactively
bt auth login
? How would you like to authenticate? Access Token
? Access token: [paste token]
✓ Authentication successful
```

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

```bash
# List pipeline runs
bt run list                         # Recent runs
bt run list --status failed        # Filter by status
bt run list --branch main          # Filter by branch

# View pipeline details
bt run view abc123def456           # Pipeline overview
bt run view abc123def456 --steps   # Include step details

# Real-time monitoring
bt run watch abc123def456          # Watch live execution
bt run logs abc123def456           # View logs
bt run logs abc123def456 --step "Deploy to staging"

# Pipeline management
bt run cancel abc123def456         # Cancel running pipeline
bt run rerun abc123def456          # Restart pipeline
bt run download abc123def456       # Download artifacts

# Trigger pipelines
bt workflow run --branch feature-branch
bt workflow run --branch main --variable "ENV=production"
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

# Set default workspace
bt config set default_workspace mycompany

# Set output format preference
bt config set output.format json

# Set up aliases
bt alias set prs "pr list"
bt alias set co "pr checkout"
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
# Authentication
export BITBUCKET_TOKEN=your_token_here
export BITBUCKET_USERNAME=your_username  # For app passwords

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
bt run logs failed-pipeline-id --format json --errors-only > errors.json

# Stream logs with error highlighting
bt run logs running-pipeline-id --follow --highlight-errors

# Get failed steps with context
bt run view failed-pipeline-id --failed-steps-only --include-logs
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

### Aliases and Shortcuts

```bash
# Create custom aliases
bt alias set prs "pr list --state open"
bt alias set myprs "pr list --author @me"
bt alias set failing "run list --status failed"

# Use aliases
bt prs
bt myprs
bt failing
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

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o bt cmd/bt/main.go

# Install locally
go install ./cmd/bt
```

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