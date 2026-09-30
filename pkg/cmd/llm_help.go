package cmd

import (
	"context"
	"fmt"
)

// LLMHelp provides structured guidance for LLM agents using the bt CLI
type LLMHelp struct {
	Command string // The specific command context (empty for global)
}

// Run executes the LLM help
func (l *LLMHelp) Run(ctx context.Context) error {
	if l.Command == "" {
		showGlobalLLMHelp()
	} else {
		showCommandLLMHelp(l.Command)
	}
	return nil
}

// Global LLM help covering the entire bt CLI
func showGlobalLLMHelp() {
	help := `# Bitbucket CLI (bt) - LLM Guide

## Overview
bt is a GitHub CLI-inspired Bitbucket Cloud CLI.
- **Purpose**: Familiar gh-style commands for Bitbucket workflows
- **Key Strength**: Fast pipeline debugging + SonarCloud coverage/issues reports
- **AI PRs**: Optional AI-assisted PR descriptions with templates
- **LLM-Friendly**: Most commands support structured JSON/YAML output (run, pr, config)

## GitHub CLI Mapping
bt commands map directly to GitHub CLI equivalents:
` + "```" + `
gh auth login    → bt auth login
gh pr list       → bt pr list
gh run list      → bt run list      # Main differentiator: enhanced pipeline debugging
gh run view      → bt run view      # Enhanced with log analysis
gh pr checks     → bt pr checks     # Pipelines for a PR (number, branch, or current branch)
gh config        → bt config        # Advanced configuration management
(no gh equivalent) → bt run report    # SonarCloud coverage/issues for a pipeline (also bt pr report)
(no gh equivalent) → bt pick          # Rebase-safe cherry-pick between PRD/HML branches
(no gh equivalent) → bt skill         # Install the bt agent skill (Claude, Cursor, Codex, Pi)
` + "```" + `

## Pipeline Debugging Workflow (Killer Feature)
The primary advantage of bt over standard CLI tools:

### Quick Error Discovery
` + "```bash" + `
# 1. Find failed pipelines
bt run list --status failed
bt pr checks                     # Or: pipelines for the current branch's PR

# 2. Get instant failure summary (last 100 lines)
bt run view 3808 --log-failed

# 3. Get complete failure context if needed
bt run view 3808 --log-failed --full-output

# 4. Analyze test failures specifically
bt run view 3808 --tests

# 5. Debug specific step
bt run view 3808 --step "Run Tests"
` + "```" + `

### Automation-Friendly JSON Output
` + "```bash" + `
# Get structured pipeline data for analysis
bt run list --status failed --output json

# Get detailed pipeline information with logs
bt run view 3808 --log-failed --output json

# Example JSON structure for failed pipeline:
{
  "id": "3808",
  "state": "FAILED", 
  "steps": [
    {
      "name": "Run Tests",
      "state": "FAILED",
      "logs": "FAILED (failures=4, skipped=138)\nAssertionError: None != '001'"
    }
  ]
}
` + "```" + `

### SonarCloud Coverage & Issues (bt run report)
` + "```bash" + `
# Coverage summary for a pipeline
bt run report 3808 --coverage

# Issues only (code quality)
bt run report 3808 --issues

# Focus on new code only, JSON output
bt run report 3808 --new-code-only --output json

# Open or print the SonarCloud dashboard URL
bt run report 3808 --web
bt run report 3808 --url
` + "```" + `

Requires ` + "`SONARCLOUD_TOKEN`" + ` in the environment. ` + "`bt pr report <pr-id>`" + ` takes the same flags for a PR.

## Common Use Cases

### Authentication Setup
` + "```bash" + `
bt auth login                    # Interactive setup (API token recommended)
bt auth status                   # Check current authentication
bt auth login --with-token <token>  # Non-interactive token login
export BITBUCKET_EMAIL="user@company.com"      # Environment variable auth
export BITBUCKET_API_TOKEN="your_token"        # Recommended method
export BITBUCKET_SESSION_TOKEN="..."           # Only for --image: bitbucket.org cookie cloud.session.token
export BITBUCKET_CSRF_TOKEN="..."              # Only for --image: bitbucket.org cookie csrftoken
` + "```" + `

### Pull Request Management with AI
` + "```bash" + `
bt pr list                       # List pull requests
bt pr create --ai                # AI-generated description (needs OPENAI_API_KEY)
bt pr create --ai --jira context.md   # Include JIRA context
bt pr create --title "Fix" --body "Description"  # Traditional creation
bt pr create --fill --image shot.png  # Attach screenshot (needs web session env vars)
bt pr view 42                    # PR details
bt pr review 42 --approve        # Approve PR
bt pr comment 42 -b "LGTM!"     # Add comment
bt pr comments 42                # Read all comments
bt pr merge 42                   # Merge PR
bt pr checkout 42                # Switch to PR branch
bt pr status                     # Your PR dashboard
` + "```" + `

### Pipeline Monitoring & Debugging
` + "```bash" + `
bt run list                      # Recent pipeline runs
bt run list --status failed     # Failed runs only
bt run list --branch main       # Specific branch
bt run view <id>                 # Pipeline overview
bt run view <id> --log-failed   # Quick error analysis (⚡ FASTEST)
bt run view <id> --log          # All step logs
bt run view <id> --tests        # Test results focus
bt run view <id> --step "name"  # Specific step logs
bt run logs <id> --errors-only  # Extracted errors only
bt run watch <id>               # Real-time monitoring
bt run cancel <id>              # Cancel running pipeline
bt run rerun <id> --failed      # Rerun only failed steps
` + "```" + `

## Output Formats
Most commands support multiple output formats via -o/--output:
- **table** (default): Human-readable terminal output
- **json**: Structured data for automation and LLM analysis
- **yaml**: Alternative structured format

` + "```bash" + `
bt run list --output json       # JSON for automation
bt pr list --output yaml        # YAML for configuration
bt run view 123 --output table  # Formatted terminal output (default)
` + "```" + `
Exceptions: ` + "`bt run logs`" + ` (text/json/yaml, default text), ` + "`bt pr diff`" + ` (diff/json/yaml, default diff), ` + "`bt run watch`" + ` (table/json).

## Environment Variables for Automation
` + "```bash" + `
# Authentication (recommended)
BITBUCKET_EMAIL="user@company.com"
BITBUCKET_API_TOKEN="your_api_token"

# Fallback names (used when EMAIL/API_TOKEN are unset)
BITBUCKET_USERNAME="username"
BITBUCKET_PASSWORD="app_password"

# Image upload (pr create/edit --image) - bitbucket.org browser cookies
BITBUCKET_SESSION_TOKEN="..."   # cookie cloud.session.token
BITBUCKET_CSRF_TOKEN="..."      # cookie csrftoken
# (alternatively stored in ~/.config/bt/bb-session)

# Configuration (BT_<SECTION>_<KEY>)
BT_DEFAULTS_OUTPUT_FORMAT="json"   # Default output format
BT_AUTH_DEFAULT_WORKSPACE="ws"     # Default workspace
BT_API_TIMEOUT="60s"               # API timeout
BT_CONFIG_PATH="..."               # Alternate config file

# AI PR descriptions (pr create/edit --ai)
OPENAI_API_KEY="sk-..."         # Required for --ai
BT_LLM_MODEL="gpt-5.4-mini"     # Optional model override (or OPENAI_MODEL)

# SonarCloud reports
SONARCLOUD_TOKEN="your_token"   # Required for bt run report / bt pr report
` + "```" + `

## Error Analysis Capabilities
bt includes advanced error detection for common scenarios:
- Build failures (compilation errors, dependency issues)
- Test failures (assertion errors, timeout failures)
- Docker errors (image pull failures, build context issues)
- Runtime errors (segfaults, out of memory)
- Network errors (connection timeouts, DNS resolution)

Error patterns are automatically highlighted and extracted for faster diagnosis.

## Best Practices for LLM Integration
1. **Use JSON output** for structured data analysis
2. **Focus on pipeline debugging workflow** for maximum time savings
3. **Leverage environment variables** for seamless automation
4. **Start with failed pipelines** using --status failed filter
5. **Use --log-failed flag** for fastest error identification

## Cherry Pick Workflow (bt pick)
Rebase-safe cherry-picking between PRD and HML branches using commit signatures (author+date+message) instead of hashes.
` + "```bash" + `
bt pick show                     # Preview unpicked commits
bt pick show -l                  # Show my latest commits
bt pick show --today             # Today's commits only
bt pick run                      # Execute cherry-pick
bt pick run -l                   # Pick my latest commits
bt pick run -r --count 3         # Pick 3 from HML to PRD
bt pick continue                 # Resume after conflict
` + "```" + `
Use ` + "`bt pick --llm`" + ` for detailed LLM guidance on pick commands.

## Global Flags
` + "```" + `
-v, --verbose        Verbose output
--no-color           Disable colored output
--config-file PATH   Config file (default ~/.config/bt/config.yml)
--version            Show version
--llm                This guide; ` + "`bt <command> --llm`" + ` for run, pr, auth, config, pick, skill, repo
` + "```" + `

## Command Categories by Priority
1. **Critical**: run (pipeline view + SonarCloud report)
2. **Important**: pr, auth (standard Git operations)
3. **Useful**: pick (cherry-pick between PRD/HML branches)
4. **Utility**: config (configuration management), skill (agent skill install)
5. **Not implemented**: repo

bt excels at pipeline debugging and provides 5x faster error diagnosis compared to web UI navigation.
`

	fmt.Print(help)
}

// Command-specific LLM help
func showCommandLLMHelp(command string) {
	switch command {
	case "run":
		showRunLLMHelp()
	case "auth":
		showAuthLLMHelp()
	case "pr":
		showPRLLMHelp()
	case "repo":
		showRepoLLMHelp()
	case "config":
		showConfigLLMHelp()
	case "pick":
		showPickLLMHelp()
	case "skill":
		showSkillLLMHelp()
	default:
		fmt.Printf("No specific LLM guidance available for command: %s\n", command)
		fmt.Println("Use 'bt --llm' for general guidance.")
	}
}

func showRunLLMHelp() {
	fmt.Print(runLLMHelpText())
}

func runLLMHelpText() string {
	help := `# bt run - Pipeline Debugging (LLM Guide)

## Primary Use Case
bt run commands provide fast pipeline debugging and SonarCloud coverage/issues reporting.

## Quick Debugging Workflow
` + "```bash" + `
# Step 1: Find problems
bt run list --status failed

# Step 2: Quick diagnosis (FASTEST - last 100 lines of failures)
bt run view <pipeline-id> --log-failed

# Step 3: Deep dive if needed
bt run view <pipeline-id> --log-failed --full-output

# Step 4: Specific analysis
bt run view <pipeline-id> --tests        # Test focus
bt run view <pipeline-id> --step "name"  # Specific step

# Step 5: SonarCloud coverage/issues (if enabled)
bt run report <pipeline-id> --coverage
` + "```" + `

## Command Details

### bt run list
Find pipelines to analyze:
` + "```bash" + `
bt run list                      # Recent runs (last 10)
bt run list --status failed     # Failed runs only (most common)
bt run list --status in_progress # Currently running (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED; case-insensitive)
bt run list --branch main       # Specific branch (matches PR-triggered runs too)
bt run list --event pull_request # Only PR-triggered runs
bt run list --event push        # Only branch/tag runs
bt run list --commit a1b2c3d    # Runs for a specific commit
bt run list --creator "Jane Doe" # Filter by creator display name
bt run list --limit 50          # More results
bt run list --output json       # Structured data
` + "```" + `

The Ref column shows which PR or branch each run belongs to, e.g.
"PR #312 feat/auth→main" for a pull request run or "main" for a branch run.

To go the other way - from a pull request to its runs - use ` + "`bt pr checks`" + `:
` + "```bash" + `
bt pr checks                     # Current branch's PR
bt pr checks 312                 # By PR number
bt pr checks feat/auth           # By branch name
` + "```" + `

### bt run view (KILLER FEATURE)
Pipeline analysis with integrated log viewing:
` + "```bash" + `
bt run view <id>                 # Pipeline overview + step status
bt run view <id> --log-failed    # Show failures (last 100 lines) ⚡ FASTEST
bt run view <id> --log-failed --full-output  # Complete failure logs
bt run view <id> --log           # All step logs (verbose)
bt run view <id> --tests         # Focus on test results
bt run view <id> --step "Run Tests"  # Specific step only
bt run view <id> --output json   # Structured data for analysis
bt run watch <id>                # Real-time monitoring (dedicated command)
bt run view <id> --watch         # Live updates (alternative method)
bt run view <id> --web           # Open in browser (--web --url prints the URL)
` + "```" + `

### bt run report (SonarCloud Coverage & Issues)
Generate a SonarCloud report tied to a pipeline (coverage + code quality):
` + "```bash" + `
# Coverage summary
bt run report <id> --coverage

# Issues only (code quality)
bt run report <id> --issues

# Focus on new code only
bt run report <id> --new-code-only

# Filter coverage findings
bt run report <id> --coverage-threshold 80
bt run report <id> --min-uncovered-lines 5
bt run report <id> --max-uncovered-lines 10
bt run report <id> --file "src/**"
bt run report <id> --new-lines-only --show-all-lines   # Only new uncovered lines, no per-file cap
bt run report <id> --issues --severity BLOCKER --severity CRITICAL
bt run report <id> --limit 20 --lines-per-file 10 --no-line-details

# Output for automation
bt run report <id> --output json

# Open or print SonarCloud dashboard
bt run report <id> --web
bt run report <id> --url
` + "```" + `

Requires ` + "`SONARCLOUD_TOKEN`" + ` in the environment.

### bt run watch (NEW - Real-time Monitoring)
Dedicated command for monitoring running pipelines:
` + "```bash" + `
bt run watch <id>                # Monitor pipeline in real-time
bt run watch <id> --output json  # JSON output for automation
bt run watch 123                 # Watch pipeline by build number
bt run watch {uuid}              # Watch pipeline by UUID
` + "```" + `

Streams step progress until completion; Ctrl+C to exit. Works only with running/pending pipelines.

### bt run logs
Fetch step logs directly, without the pipeline overview:
` + "```bash" + `
bt run logs <id>                 # All step logs
bt run logs <id> --errors-only   # Only lines matching error patterns
bt run logs <id> --step "name"   # A single step
bt run logs <id> --errors-only --context 10  # More context around errors (default 3)
bt run logs <id> --follow        # Follow live logs of a running pipeline
bt run logs <id> --output json   # text (default), json, yaml
` + "```" + `

### bt run cancel
` + "```bash" + `
bt run cancel <id>               # Stop a running pipeline
bt run cancel <id> --force       # Skip confirmation
` + "```" + `

### bt run rerun
` + "```bash" + `
bt run rerun <id>                # Rerun a pipeline
bt run rerun <id> --failed       # Rerun only failed steps
bt run rerun <id> --step "name"  # Rerun a specific step
bt run rerun <id> --force        # Skip confirmation
` + "```" + `
Reruns preserve the original target, so rerunning a PR-triggered pipeline
re-runs it against the same pull request.

## JSON Output Structure
Perfect for LLM analysis:
` + "```json" + `
{
  "id": "3808",
  "build_number": 123,
  "state": "FAILED",
  "result": "FAILED", 
  "target": {
    "branch": "main",
    "commit": "abc123"
  },
  "steps": [
    {
      "name": "Run Tests",
      "state": "FAILED",
      "duration": 120,
      "logs": "FAILED (failures=4)\nAssertionError: None != '001'"
    }
  ]
}
` + "```" + `

## Common Error Patterns Detected
- Test failures: "FAILED (failures=N)", "AssertionError", "Test failed"
- Build errors: "compilation terminated", "build failed", "error:"
- Docker issues: "image pull failed", "build context"
- Runtime errors: "segmentation fault", "out of memory"
- Network issues: "connection timeout", "DNS resolution failed"

## Performance Benefits
- **Web UI**: Navigate → Pipelines → Click run → Find failed step → Click logs → Scroll
- **bt CLI**: ` + "`bt run view <id> --log-failed`" + ` (1 command, instant results)

## Best Practices
1. Start with ` + "`bt run list --status failed`" + ` to find issues
2. Use ` + "`--log-failed`" + ` for quickest error identification  
3. Add ` + "`--full-output`" + ` only when you need complete context
4. Use ` + "`--output json`" + ` for automated analysis
5. Specify ` + "`--step`" + ` when you know which step failed
`

	return help
}

func showAuthLLMHelp() {
	help := `# bt auth - Authentication (LLM Guide)

## Overview
bt supports multiple Bitbucket authentication methods with seamless CLI integration.

## Recommended Method: API Tokens
` + "```bash" + `
# Interactive setup (recommended)
bt auth login

# Environment variables (automation)
export BITBUCKET_EMAIL="user@company.com"
export BITBUCKET_API_TOKEN="your_api_token"
` + "```" + `

## Commands
` + "```bash" + `
bt auth login                    # Interactive authentication setup
bt auth login --with-token <token>  # Non-interactive token login
bt auth logout                   # Clear stored credentials
bt auth status                   # Show current authentication
bt auth refresh                  # Refresh expired tokens
` + "```" + `

## Authentication Method
bt authenticates with an Atlassian API token (email + token). BITBUCKET_USERNAME /
BITBUCKET_PASSWORD are accepted as fallback names for the same credentials.

## Environment Variables
` + "```bash" + `
# API Token (recommended)
BITBUCKET_EMAIL="user@company.com"
BITBUCKET_API_TOKEN="your_token"

# Fallback names (used when EMAIL/API_TOKEN are unset)
BITBUCKET_USERNAME="username"
BITBUCKET_PASSWORD="app_password"

# Web session - only needed for pr create/edit --image (bitbucket.org cookies)
BITBUCKET_SESSION_TOKEN="..."   # cookie cloud.session.token
BITBUCKET_CSRF_TOKEN="..."      # cookie csrftoken
# Or store both in ~/.config/bt/bb-session instead of env vars
` + "```" + `

## Troubleshooting
` + "```bash" + `
bt auth status                   # Check authentication state
bt auth refresh                  # Fix expired tokens
bt auth logout && bt auth login  # Reset authentication
` + "```" + `
`

	fmt.Print(help)
}

func showPRLLMHelp() {
	fmt.Print(prLLMHelpText())
}

func prLLMHelpText() string {
	help := `# bt pr - Pull Requests (LLM Guide)

## Overview
Complete pull request workflow with AI-powered descriptions and GitHub CLI compatibility.

## AI-Powered PR Creation (OpenAI)
` + "```bash" + `
bt pr create --ai                          # AI-generated title/description
bt pr create --ai --jira project.md       # Include JIRA context from a markdown file
bt pr create --ai --debug                 # Show AI inputs/debug output
bt pr edit 42 --ai                         # Regenerate an existing PR's description

# - Requires OPENAI_API_KEY; model via BT_LLM_MODEL / OPENAI_MODEL (default gpt-5.4-mini)
# - Structured JSON output rendered into a fixed English markdown template
#   (Context & Description, Technical Impact & UI, Testing & Quality, Safety & Risk)
# - 24-hour cache for identical requests; falls back to local templates if OpenAI fails
` + "```" + `

## PR Creation Flags
` + "```bash" + `
bt pr create --title "Fix" --body "Desc"   # Explicit title/body
bt pr create --fill                        # Title/body from commit messages
bt pr create --base develop                # Target branch
bt pr create --draft                       # Draft PR
bt pr create --reviewer alice --reviewer bob
bt pr create --close-source-branch         # Delete source branch on merge
bt pr create --no-push                     # Don't push the branch first
bt pr create --no-emoji                    # No emojis in generated titles
` + "```" + `

## Images in PR Descriptions (--image, repeatable)
` + "```bash" + `
bt pr create --fill --image shot.png --image after.png
bt pr edit 42 --image a.png --image b.png
` + "```" + `
Each file is uploaded to Bitbucket and ` + "`![name](url)`" + ` is appended to the description.
Uploads use a bitbucket.org web session, not the API token: set
` + "`BITBUCKET_SESSION_TOKEN`" + ` (cookie cloud.session.token) and ` + "`BITBUCKET_CSRF_TOKEN`" + ` (cookie csrftoken),
or save the browser request headers (` + "`Cookie:`" + ` and ` + "`X-CSRFToken:`" + ` lines) to ` + "`~/.config/bt/bb-session`" + `.
Sessions expire; bt warns when expiry is near.

## Complete PR Workflow
` + "```bash" + `
# Creation and setup
bt pr list                                 # List pull requests
bt pr list --state open                   # Filter by state
bt pr list                                # Your PRs (default); --all for everyone
bt pr list --reviewer alice --state all   # Reviewer filter; state: open, merged, declined, all
bt pr list --all --limit 50 --sort created
bt pr list-all                            # Open PRs across every repo in the workspace
bt pr list-all --approved --url           # Only approved, as "<repo:branch> <target> <url>"
bt pr open 42 43                          # Open PRs in the browser (--show prints URLs)
bt pr create --ai                         # AI-generated description
bt pr create --title "Fix" --body "Desc" # Traditional creation

# Review and collaboration
bt pr view 42                             # PR details
bt pr view 42 --comments                  # PR details plus comment bodies
bt pr view 42 --web                       # Open in browser
bt pr diff 42                             # Show changes (test files excluded by default)
bt pr diff 42 --name-only                 # Changed file names
bt pr diff 42 --include-tests --file src/auth.go  # Include tests / single file
bt pr diff 42 --patch > pr.patch          # git-apply-able patch
bt pr files 42 --filter '*.go'            # List changed files
bt pr review 42 --approve                 # Approve PR
bt pr review 42 --request-changes -b "See comments"
bt pr review 42 --comment -F review.md    # Review comment from file
bt pr checkout 42                         # Switch to PR branch

# Reading and writing comments
bt pr comments 42                         # Read all comment bodies, threaded, with IDs
bt pr comments 42 --author @me            # Only your comments
bt pr comments 42 -o json                 # Full comment objects (id, parent, inline)
bt pr comment 42 -b "Great work!"         # Add a top-level comment
bt pr comment 42 --reply-to 12344 -b "Done"        # Reply in-thread to comment 12344
bt pr comment 42 --file src/auth.go --line 15 -b "Extract this"  # Inline comment
bt pr comment 42 --file src/auth.go --line 9 --line-type old -b "Why removed?"  # Old side of diff
bt pr review-history                      # Your (@me) comments across merged PRs in the repo
bt pr review-history --author alice --state all --concurrency 16

# Management and status
bt pr status                              # Your PR dashboard
bt pr checks                              # CI/build status for the current branch's PR
bt pr checks 42                           # CI/build status for PR 42
bt pr checks feat/auth                    # CI/build status by branch name
bt pr checks 42 --watch -i 15             # Poll every 15s
bt pr checks 42 --watch --fail-fast       # Stop watching on first failure
bt pr edit 42 --title "New title"        # Edit metadata
bt pr edit 42 -F body.md                  # Description from file
bt pr edit 42 --add-reviewer alice --remove-reviewer bob
bt pr edit 42 --draft                     # Convert to draft (--ready to undo)
bt pr edit 42 --image a.png --image b.png # Upload images and append to description
bt pr ready 42                            # Mark draft as ready

# Lifecycle
bt pr merge 42                            # Merge PR
bt pr merge 42 --squash --delete-branch  # Squash merge with cleanup
bt pr merge 42 --auto -m "msg" -f         # Merge when checks pass, custom message, no prompt
bt pr close 42 -c "Superseded" --delete-branch  # Decline PR
bt pr reopen 42                           # Reopen PR

# Advanced operations
bt pr update-branch 42                    # Sync with target branch
bt pr lock 42 --reason spam               # Lock conversation
bt pr unlock 42                           # Unlock conversation

# SonarCloud (same flags as bt run report; needs SONARCLOUD_TOKEN)
bt pr report 42 --coverage --new-lines-only    # New uncovered lines in this PR
bt pr report 42 --coverage --context 3         # Show code context around uncovered lines
bt pr report 42 --issues --all-issues          # Include accepted/pre-existing issues
` + "```" + `

## GitHub CLI Mapping (Complete Parity)
` + "```bash" + `
gh pr list     → bt pr list
gh pr create   → bt pr create     # Enhanced with AI
gh pr view     → bt pr view
gh pr diff     → bt pr diff
gh pr review   → bt pr review
gh pr comment  → bt pr comment     # Plus --reply-to for threaded replies
gh pr checkout → bt pr checkout
gh pr checks   → bt pr checks      # Optional arg: number, branch, or current branch
gh pr merge    → bt pr merge
gh pr close    → bt pr close
gh pr edit     → bt pr edit
gh pr status   → bt pr status

# bt-only (no gh equivalent)
bt pr comments         # gh reads comments via 'gh pr view --comments' only
bt pr review-history   # Mine one author's comments across every PR
bt pr report           # SonarCloud coverage/issues for a PR
` + "```" + `

## AI Analysis Capabilities
- **File categorization**: backend, frontend, database, documentation, configuration
- **Change type detection**: 20+ programming languages supported
- **Smart checklists**: Auto-generated based on detected change types
- **JIRA integration**: Context extraction from markdown files
- **Template compliance**: Fixed markdown structure

Most pr commands accept -o/--output (table, json, yaml), --workspace, --repository; destructive ones accept -f/--force to skip prompts.
`

	return help
}

func showRepoLLMHelp() {
	help := `# bt repo - Repository Operations (LLM Guide)

## Status
Repository commands are not yet implemented in this build.
`

	fmt.Print(help)
}

func showConfigLLMHelp() {
	help := `# bt config - Configuration Management (LLM Guide)

## Overview
Advanced configuration management for bt CLI with nested key support and type validation.

## Key Features
- **Nested keys**: Use dot notation (auth.default_workspace, api.timeout)
- **Type validation**: Automatic validation for durations, URLs, and enum values
- **Multi-format output**: Support for table, JSON, and YAML formats
- **Safe operations**: Atomic file updates with validation

## Common Commands
` + "```bash" + `
# View all configuration
bt config list
bt config list --output json        # JSON for automation

# Get specific values
bt config get auth.method            # Get authentication method
bt config get auth.default_workspace # Get default workspace
bt config get api.timeout           # Get API timeout

# Set configuration values with validation
bt config set auth.default_workspace mycompany  # Set workspace
bt config set api.timeout 60s                   # Set timeout (validates duration)
bt config set defaults.output_format json      # Set default output format

# Remove configuration (reset to default)
bt config unset auth.default_workspace
bt config unset api.timeout
` + "```" + `

## Available Configuration Keys
` + "```" + `
auth.method              # Authentication method (app_password, oauth, access_token)
auth.default_workspace   # Default workspace for operations
api.base_url            # Bitbucket API base URL
api.timeout             # API request timeout (duration format: 30s, 1m, etc.)
defaults.output_format  # Default output format (table, json, yaml)
llm.model               # Model for --ai PR descriptions (env BT_LLM_MODEL)
pick.prefix             # bt pick branch prefix
pick.suffix_prd         # bt pick production branch suffix
pick.suffix_hml         # bt pick homologation branch suffix
version                 # Configuration schema version
` + "```" + `

## Automation Examples
` + "```bash" + `
# Export all configuration for backup
bt config list --output yaml > bt-config-backup.yml

# Get specific config value for scripting
WORKSPACE=$(bt config get auth.default_workspace --output json | jq -r .value)

# Batch configuration setup
bt config set auth.default_workspace $MY_WORKSPACE
bt config set api.timeout 45s
bt config set defaults.output_format json
` + "```" + `

## Type Validation
The config system validates values based on their expected types:
- **Duration fields** (api.timeout): Must be valid Go duration (30s, 1m, 1h30m)
- **Enum fields** (auth.method): Must be one of valid options
- **URL fields** (api.base_url): Must be valid HTTP/HTTPS URLs

## Error Handling
` + "```bash" + `
# Invalid duration
bt config set api.timeout invalid-time
# Error: invalid duration format

# Invalid auth method  
bt config set auth.method invalid-method
# Error: invalid configuration: unknown auth method

# Nonexistent key
bt config get nonexistent.key
# Error: configuration key not found
` + "```" + `

## Best Practices for LLM Integration
1. **Use JSON output** for structured data extraction
2. **Validate before setting** complex values like durations
3. **Use get commands** to check current state before modifications
4. **Handle errors gracefully** with proper validation feedback

Note: Configuration is automatically saved to ~/.config/bt/config.yml with secure atomic operations.
`

	fmt.Print(help)
}

func showPickLLMHelp() {
	help := `# bt pick - Cherry Pick (LLM Guide)

## Overview
Rebase-safe cherry-picking between PRD and HML branches. Matches commits by signature (author + date + message) instead of hashes, so it works correctly after rebases.

## Workflow
` + "```bash" + `
# 1. Preview what will be picked (dry run)
bt pick show

# 2. Execute the cherry-pick
bt pick run

# 3. If conflicts occur, resolve them, then continue
bt pick continue
` + "```" + `

## Commands

### bt pick show (preview)
` + "```bash" + `
bt pick show                     # Preview unpicked PRD → HML commits
bt pick show -l                  # Show current user's latest (up to 100)
bt pick show -r                  # Reverse: HML → PRD
bt pick show --today             # Today's commits only
bt pick show --yesterday         # Yesterday's commits only
bt pick show --since 2024-01-01  # Since date
bt pick show --until 2024-01-31  # Until date
bt pick show --prefix ZUP- --suffix-prd -prd --suffix-hml -hml  # Override branch convention
bt pick show --count 10          # Limit to 10
bt pick show --no-filter         # Skip smart deduplication
bt pick show --debug             # Debug output
` + "```" + `

### bt pick run (execute)
Same flags as show. Displays commits then cherry-picks them.
` + "```bash" + `
bt pick run                      # Pick all unpicked commits
bt pick run -l                   # Pick my latest
bt pick run -r --count 3         # Pick 3 from HML to PRD
bt pick run --today              # Pick today's commits
` + "```" + `

### bt pick continue
Resume cherry-picking after resolving conflicts:
` + "```bash" + `
# 1. Resolve conflicts in files
# 2. git add <resolved-files>
# 3. bt pick continue
` + "```" + `

## Branch Convention
Requires branches following: ` + "`{prefix}{identifier}{suffix}`" + `
Default: ` + "`ZUP-123-prd`" + ` and ` + "`ZUP-123-hml`" + `

You must be on one of the pair branches (PRD or HML) when running pick.

## Configuration
` + "```bash" + `
bt config set pick.prefix ZUP-
bt config set pick.suffix_prd -prd
bt config set pick.suffix_hml -hml
` + "```" + `

## Environment Variables
` + "```bash" + `
BT_PICK_PREFIX=ZUP-
BT_PICK_SUFFIX_PRD=-prd
BT_PICK_SUFFIX_HML=-hml
` + "```" + `

## How Deduplication Works
Commits are matched by signature (author + date + first line of message), not by hash. This means:
- Rebased branches still match correctly
- Already-picked commits are automatically excluded
- Use ` + "`--no-filter`" + ` to skip deduplication if needed
`

	fmt.Print(help)
}

func showSkillLLMHelp() {
	help := `# bt skill - Agent Skill Management (LLM Guide)

## Overview
Installs the bt agent skill and symlinks it into each supported agent's skills directory:
~/.claude/skills, ~/.cursor/skills, ~/.codex/skills, ~/.pi/agent/skills.

## Commands
` + "```bash" + `
bt skill add                     # Install and link the skill
bt skill add --force             # Overwrite existing non-symlink skill directories
bt skill update                  # Update to the latest published version
bt skill status                  # Installed version, per-agent link status, update check
bt skill remove                  # Remove the skill and its links
` + "```" + `
`

	fmt.Print(help)
}

// GetLLMHelpContent returns structured help content for programmatic access
func GetLLMHelpContent() map[string]interface{} {
	return map[string]interface{}{
		"overview":     "bt is a GitHub CLI-inspired Bitbucket Cloud CLI",
		"key_strength": "fast pipeline debugging plus SonarCloud coverage/issues reporting",
		"primary_workflow": []string{
			"bt run list --status failed",
			"bt run view <id> --log-failed",
			"bt run view <id> --log-failed --full-output",
			"bt run view <id> --tests",
			"bt run view <id> --step 'Step Name'",
		},
		"command_mapping": map[string]string{
			"gh auth login": "bt auth login",
			"gh pr list":    "bt pr list",
			"gh run list":   "bt run list",
			"gh run view":   "bt run view",
			"gh pr create":  "bt pr create",
			"gh pr view":    "bt pr view",
			"gh pr checks":  "bt pr checks",
			"gh config":     "bt config",
		},
		"output_formats": []string{"table", "json", "yaml"},
		"auth_env_vars": []string{
			"BITBUCKET_EMAIL",
			"BITBUCKET_API_TOKEN",
			"BITBUCKET_USERNAME",
			"BITBUCKET_PASSWORD",
			"BITBUCKET_SESSION_TOKEN",
			"BITBUCKET_CSRF_TOKEN",
		},
	}
}
