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
bt is a 1:1 replacement for GitHub CLI that works with Bitbucket Cloud.
- **Purpose**: Provide identical command structure and user experience as GitHub CLI (gh) for Bitbucket
- **Key Strength**: 5x faster pipeline debugging compared to Bitbucket web UI
- **LLM-Friendly**: All commands support structured JSON output for automation

## GitHub CLI Mapping
bt commands map directly to GitHub CLI equivalents:
` + "```" + `
gh auth login    → bt auth login
gh repo clone    → bt repo clone  
gh pr list       → bt pr list
gh run list      → bt run list      # Main differentiator: enhanced pipeline debugging
gh run view      → bt run view      # Enhanced with log analysis
gh api           → bt api
` + "```" + `

## Pipeline Debugging Workflow (Killer Feature)
The primary advantage of bt over standard CLI tools:

### Quick Error Discovery
` + "```bash" + `
# 1. Find failed pipelines
bt run list --status failed

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

## Common Use Cases

### Authentication Setup
` + "```bash" + `
bt auth login                    # Interactive setup (API token recommended)
bt auth status                   # Check current authentication
export BITBUCKET_EMAIL="user@company.com"      # Environment variable auth
export BITBUCKET_API_TOKEN="your_token"        # Recommended method
` + "```" + `

### Repository Operations
` + "```bash" + `
bt repo clone workspace/repo     # Clone repository
bt repo list                     # List accessible repositories
bt repo view workspace/repo      # Repository details
` + "```" + `

### Pull Request Management
` + "```bash" + `
bt pr list                       # List pull requests
bt pr create --title "Fix" --body "Description"  # Create PR
bt pr view 42                    # PR details
bt pr merge 42                   # Merge PR
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
bt run watch <id>               # Real-time monitoring ✅ AVAILABLE
bt run cancel <id>              # Cancel running pipeline (coming soon)
` + "```" + `

## Output Formats
All commands support multiple output formats for different use cases:
- **table** (default): Human-readable terminal output
- **json**: Structured data for automation and LLM analysis
- **yaml**: Alternative structured format

` + "```bash" + `
bt run list --output json       # JSON for automation
bt pr list --output yaml        # YAML for configuration
bt run view 123 --output table  # Formatted terminal output (default)
` + "```" + `

## Environment Variables for Automation
` + "```bash" + `
# Authentication (recommended)
BITBUCKET_EMAIL="user@company.com"
BITBUCKET_API_TOKEN="your_api_token"

# Legacy authentication (still supported)
BITBUCKET_USERNAME="username"
BITBUCKET_PASSWORD="app_password"
BITBUCKET_TOKEN="access_token"

# Configuration
BT_OUTPUT_FORMAT="json"         # Default output format
BT_NO_COLOR="1"                 # Disable colors for automation
BT_VERBOSE="1"                  # Enable verbose output
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

## Command Categories by Priority
1. **Critical**: auth, run (pipeline debugging)
2. **Important**: repo, pr (standard Git operations)  
3. **Utility**: api, config, browse (advanced features)

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
	default:
		fmt.Printf("No specific LLM guidance available for command: %s\n", command)
		fmt.Println("Use 'bt --llm' for general guidance.")
	}
}

func showRunLLMHelp() {
	help := `# bt run - Pipeline Debugging (LLM Guide)

## Primary Use Case
bt run commands provide 5x faster pipeline debugging compared to Bitbucket web UI.

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
` + "```" + `

## Command Details

### bt run list
Find pipelines to analyze:
` + "```bash" + `
bt run list                      # Recent runs (last 10)
bt run list --status failed     # Failed runs only (most common)
bt run list --status in_progress # Currently running
bt run list --branch main       # Specific branch
bt run list --limit 50          # More results
bt run list --output json       # Structured data
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
` + "```" + `

### bt run watch (NEW - Real-time Monitoring)
Dedicated command for monitoring running pipelines:
` + "```bash" + `
bt run watch <id>                # Monitor pipeline in real-time
bt run watch <id> --output json  # JSON output for automation
bt run watch 123                 # Watch pipeline by build number
bt run watch {uuid}              # Watch pipeline by UUID
` + "```" + `

**Key Features:**
- ✅ Live updates every 5 seconds
- ✅ Step completion notifications
- ✅ Graceful Ctrl+C exit
- ✅ Progress indicators and status icons
- ✅ Automatic completion detection
- ✅ Works only with running/pending pipelines

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

	fmt.Print(help)
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
bt auth login --with-token      # Direct token input
bt auth logout                   # Clear stored credentials
bt auth status                   # Show current authentication
bt auth refresh                  # Refresh expired tokens
` + "```" + `

## Authentication Methods
1. **API Token** (recommended): Email + API token from Atlassian
2. **App Password** (legacy): Username + app password  
3. **OAuth 2.0**: Browser-based flow with automatic refresh
4. **Access Token**: Repository/workspace scoped tokens

## Environment Variables
` + "```bash" + `
# API Token (recommended)
BITBUCKET_EMAIL="user@company.com"
BITBUCKET_API_TOKEN="your_token"

# App Password (legacy)
BITBUCKET_USERNAME="username"  
BITBUCKET_PASSWORD="app_password"

# Access Token
BITBUCKET_TOKEN="access_token"
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
	help := `# bt pr - Pull Requests (LLM Guide)

## Overview
Manage Bitbucket pull requests with identical syntax to GitHub CLI.

## Common Commands
` + "```bash" + `
bt pr list                       # List pull requests
bt pr list --state open         # Filter by state
bt pr list --author @me         # Your PRs only
bt pr create                     # Interactive PR creation
bt pr create --title "Fix" --body "Description"  # Direct creation
bt pr view 42                    # PR details
bt pr merge 42                   # Merge PR
bt pr close 42                   # Close PR
` + "```" + `

## GitHub CLI Mapping
` + "```bash" + `
gh pr list    → bt pr list
gh pr create  → bt pr create
gh pr view    → bt pr view
gh pr merge   → bt pr merge
gh pr close   → bt pr close
` + "```" + `

Note: PR commands follow GitHub CLI patterns exactly for seamless migration.
`

	fmt.Print(help)
}

func showRepoLLMHelp() {
	help := `# bt repo - Repository Operations (LLM Guide)

## Overview
Repository management with GitHub CLI compatibility.

## Common Commands
` + "```bash" + `
bt repo clone workspace/repo     # Clone repository
bt repo list                     # List repositories
bt repo list workspace          # List workspace repositories
bt repo create workspace/name   # Create repository
bt repo view workspace/repo     # Repository details
` + "```" + `

## GitHub CLI Mapping
` + "```bash" + `
gh repo clone    → bt repo clone
gh repo list     → bt repo list  
gh repo create   → bt repo create
gh repo view     → bt repo view
` + "```" + `

Note: Repository commands maintain GitHub CLI compatibility for easy migration.
`

	fmt.Print(help)
}

// GetLLMHelpContent returns structured help content for programmatic access
func GetLLMHelpContent() map[string]interface{} {
	return map[string]interface{}{
		"overview": "bt is a 1:1 replacement for GitHub CLI that works with Bitbucket Cloud",
		"key_strength": "5x faster pipeline debugging compared to Bitbucket web UI",
		"primary_workflow": []string{
			"bt run list --status failed",
			"bt run view <id> --log-failed",
			"bt run view <id> --log-failed --full-output",
			"bt run view <id> --tests",
			"bt run view <id> --step 'Step Name'",
		},
		"command_mapping": map[string]string{
			"gh auth login": "bt auth login",
			"gh repo clone": "bt repo clone",
			"gh pr list":    "bt pr list",
			"gh run list":   "bt run list",
			"gh run view":   "bt run view",
			"gh api":        "bt api",
		},
		"output_formats": []string{"table", "json", "yaml"},
		"auth_env_vars": []string{
			"BITBUCKET_EMAIL",
			"BITBUCKET_API_TOKEN",
		},
	}
}