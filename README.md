# bt

A CLI for Bitbucket Cloud inspired by GitHub CLI (`gh`).

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
| `BITBUCKET_SESSION_TOKEN` | `cloud.session.token` cookie from bitbucket.org (only for `--image`) |
| `BITBUCKET_CSRF_TOKEN` | `csrftoken` cookie from bitbucket.org (only for `--image`) |

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
bt run report 1234 --coverage # SonarCloud coverage summary

# Cherry pick (PRD → HML)
bt pick show                  # Preview unpicked commits
bt pick show -l               # Show my latest commits
bt pick run                   # Execute cherry-pick
bt pick continue              # Resume after conflict
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
| `run logs <id>` | Show logs (supports `--errors-only`) |
| `run cancel <id>` | Cancel running pipeline |
| `run rerun <id>` | Rerun a pipeline (`--failed`, `--step`) (in progress) |
| `run report <id>` | SonarCloud quality report |

### Cherry Pick

| Command | Description |
|---------|-------------|
| `pick show` | Preview unpicked commits (dry run) |
| `pick run` | Execute cherry-pick of unpicked commits |
| `pick continue` | Resume after conflict resolution |

Common flags: `-r` reverse, `-l` latest, `-c N` count, `--today`, `--yesterday`, `--since`, `--until`

#### Images in PR descriptions

`bt pr create --image shot.png` and `bt pr edit 42 --image a.png --image b.png` upload images and append them to the description. Bitbucket has no public API for this, so `bt` uses your browser session:

1. In a logged-in bitbucket.org tab, open DevTools → Application → Cookies → `https://bitbucket.org`.
2. Export `cloud.session.token` as `BITBUCKET_SESSION_TOKEN` and `csrftoken` as `BITBUCKET_CSRF_TOKEN`.
   Alternatively, save a `Cookie:` line and an `X-CSRFToken:` line to `~/.config/bt/bb-session` (chmod 600).

The session lasts about 30 days; `bt` warns 3 days before it expires. A 401/403 means the session must be re-exported. Uploaded images are visible only to users with access to the repository.

## Configuration

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
pick:
  prefix: ZUP-       # Branch prefix (e.g. ZUP-123-prd)
  suffix_prd: -prd   # Production branch suffix
  suffix_hml: -hml   # Homologation branch suffix
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BITBUCKET_EMAIL` | Atlassian account email |
| `BITBUCKET_API_TOKEN` | API token |
| `BITBUCKET_SESSION_TOKEN` | `cloud.session.token` cookie from bitbucket.org (only for `--image`) |
| `BITBUCKET_CSRF_TOKEN` | `csrftoken` cookie from bitbucket.org (only for `--image`) |
| `SONARCLOUD_TOKEN` | SonarCloud token (required for reports) |
| `BT_OUTPUT_FORMAT` | Default output format |
| `BT_NO_COLOR` | Disable colors |
| `BT_PICK_PREFIX` | Override pick branch prefix |
| `BT_PICK_SUFFIX_PRD` | Override pick PRD suffix |
| `BT_PICK_SUFFIX_HML` | Override pick HML suffix |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Repository not found" | Check workspace/repo name, verify access |
| "Pipeline not found" | Ensure pipelines are enabled, check `bitbucket-pipelines.yml` exists |
| Auth issues | Run `bt auth logout` then `bt auth login` |

## License

MIT - see [LICENSE](LICENSE)
