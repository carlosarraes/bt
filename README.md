# bt

A CLI for Bitbucket Cloud with the same command structure as GitHub CLI (`gh`).

## Installation

```bash
# Quick install
curl -sSf https://raw.githubusercontent.com/carlosarraes/bt/main/install.sh | sh

# Or with Go
go install github.com/carlosarraes/bt/cmd/bt@latest
```

Manual download available on the [releases page](https://github.com/carlosarraes/bt/releases).

## Authentication

```bash
bt auth login
```

Create an API token at [Atlassian Account Security](https://id.atlassian.com/manage-profile/security/api-tokens).

For automation, use environment variables:

| Variable | Description |
|----------|-------------|
| `BITBUCKET_EMAIL` | Your Atlassian account email |
| `BITBUCKET_API_TOKEN` | API token from Atlassian |

## Quick Start

```bash
# Pull requests
bt pr list                    # List PRs in current repo
bt pr list-all                # List all your PRs across workspace
bt pr create --ai             # Create PR with AI-generated description
bt pr view 123                # View PR details
bt pr review 123 --approve    # Approve a PR
bt pr merge 123               # Merge a PR

# Pipelines
bt run list                   # List recent runs
bt run view 1234 --log-failed # View failed step logs
bt run watch 1234             # Watch running pipeline
```

## Commands

### Auth

| Command | Description |
|---------|-------------|
| `auth login` | Authenticate with Bitbucket |
| `auth logout` | Log out |
| `auth status` | Check authentication status |

### Pull Requests

| Command | Description |
|---------|-------------|
| `pr list` | List PRs in repository |
| `pr list-all` | List all your PRs across workspace |
| `pr create` | Create a PR (`--ai` for AI description) |
| `pr view <id>` | View PR details |
| `pr diff <id>` | Show PR diff |
| `pr review <id>` | Review PR (`--approve`, `--request-changes`, `--comment`) |
| `pr merge <id>` | Merge PR (`--squash`, `--delete-branch`) |
| `pr checkout <id>` | Check out PR branch locally |
| `pr edit <id>` | Edit PR title/description |
| `pr comment <id>` | Add comment to PR |
| `pr close <id>` | Close PR |
| `pr reopen <id>` | Reopen closed PR |
| `pr status` | Show your PR activity |
| `pr checks <id>` | View CI status |
| `pr open <id>` | Open PR in browser |
| `pr files <id>` | List changed files |
| `pr report <id>` | SonarCloud quality report |

### Pipelines

| Command | Description |
|---------|-------------|
| `run list` | List pipeline runs |
| `run view <id>` | View run details (`--log-failed`, `--tests`) |
| `run watch <id>` | Watch running pipeline |
| `run cancel <id>` | Cancel running pipeline |
| `run report <id>` | SonarCloud quality report |

### Configuration

| Command | Description |
|---------|-------------|
| `config list` | View all settings |
| `config get <key>` | Get specific setting |
| `config set <key> <value>` | Set a value |
| `config unset <key>` | Remove a value |

## Configuration

Config file: `~/.config/bt/config.yml`

```yaml
auth:
  default_workspace: myworkspace
defaults:
  output_format: table  # table, json, yaml
pr:
  branch_suffix_mapping:
    hml: homolog   # -hml branches target homolog
    prd: main      # -prd branches target main
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BITBUCKET_EMAIL` | Atlassian account email |
| `BITBUCKET_API_TOKEN` | API token |
| `SONARCLOUD_TOKEN` | SonarCloud token (for reports) |
| `BT_OUTPUT_FORMAT` | Default output format |
| `BT_NO_COLOR` | Disable colors |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Repository not found" | Check workspace/repo name, verify access |
| "Pipeline not found" | Ensure pipelines are enabled, check `bitbucket-pipelines.yml` exists |
| Auth issues | Run `bt auth logout` then `bt auth login` |

## License

MIT - see [LICENSE](LICENSE)
