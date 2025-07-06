package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
)

// ListCmd handles the pr list command
type ListCmd struct {
	State      string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Author     string `help:"Filter by pull request author"`
	Reviewer   string `help:"Filter by pull request reviewer"`
	Limit      int    `help:"Maximum number of pull requests to show" default:"30"`
	Sort       string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the pr list command
func (cmd *ListCmd) Run(ctx context.Context) error {
	// Create PR context with authentication and configuration
	prCtx, err := NewPRContext(ctx, cmd.Output, cmd.NoColor)
	if err != nil {
		return err
	}

	// Override workspace and repository if provided via flags
	if cmd.Workspace != "" {
		prCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		prCtx.Repository = cmd.Repository
	}

	// Validate workspace and repository are available
	if err := prCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	// Validate state filter if provided
	if cmd.State != "" {
		if err := validateState(cmd.State); err != nil {
			return err
		}
	}

	// Validate limit
	if cmd.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	if cmd.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	// Prepare pull request list options
	options := &api.PullRequestListOptions{
		PageLen: cmd.Limit,
		Page:    1,
		Sort:    "-updated_on", // Most recently updated first by default
	}

	// Add state filter if specified (GitHub CLI compatible)
	if cmd.State != "" && cmd.State != "all" {
		options.State = strings.ToUpper(cmd.State)
	}

	// Add author filter if specified
	if cmd.Author != "" {
		options.Author = cmd.Author
	}

	// Add reviewer filter if specified
	if cmd.Reviewer != "" {
		options.Reviewer = cmd.Reviewer
	}

	// Add sort order if specified
	if cmd.Sort != "" {
		switch strings.ToLower(cmd.Sort) {
		case "created":
			options.Sort = "-created_on"
		case "updated":
			options.Sort = "-updated_on"
		case "priority":
			options.Sort = "-priority"
		default:
			return fmt.Errorf("invalid sort field '%s'. Valid fields are: created, updated, priority", cmd.Sort)
		}
	}

	// List pull requests
	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	// Parse the pull request results
	pullRequests, err := parsePullRequestResults(result)
	if err != nil {
		return fmt.Errorf("failed to parse pull request results: %w", err)
	}

	// Format and display output
	return cmd.formatOutput(prCtx, pullRequests)
}

// formatOutput formats and displays the pull request results
func (cmd *ListCmd) formatOutput(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, pullRequests)
	case "json":
		return cmd.formatJSON(prCtx, pullRequests)
	case "yaml":
		return cmd.formatYAML(prCtx, pullRequests)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

// formatTable formats pull requests as a table
func (cmd *ListCmd) formatTable(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	if len(pullRequests) == 0 {
		fmt.Println("No pull requests found")
		return nil
	}

	// Custom table rendering for better control (GitHub CLI compatible layout)
	headers := []string{"ID", "Title", "Branch", "Author", "State", "Updated"}
	rows := make([][]string, len(pullRequests))
	
	for i, pr := range pullRequests {
		// Format PR title with length limit
		title := pr.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		// Format source branch
		sourceBranch := "-"
		if pr.Source != nil && pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
			if len(sourceBranch) > 20 {
				sourceBranch = sourceBranch[:17] + "..."
			}
		}

		// Format author
		author := "-"
		if pr.Author != nil {
			if pr.Author.DisplayName != "" {
				author = pr.Author.DisplayName
			} else if pr.Author.Username != "" {
				author = pr.Author.Username
			}
			// Truncate long names
			if len(author) > 15 {
				author = author[:12] + "..."
			}
		}

		// Format state
		state := pr.State
		if state == "" {
			state = "UNKNOWN"
		}

		// Format updated time
		updatedTime := FormatRelativeTime(pr.UpdatedOn)

		rows[i] = []string{
			fmt.Sprintf("#%d", pr.ID),
			title,
			sourceBranch,
			author,
			state,
			updatedTime,
		}
	}

	return renderCustomTable(headers, rows)
}

// formatJSON formats pull requests as JSON
func (cmd *ListCmd) formatJSON(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	// Create a simplified structure for JSON output
	output := map[string]interface{}{
		"total_count":     len(pullRequests),
		"pull_requests":   pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

// formatYAML formats pull requests as YAML
func (cmd *ListCmd) formatYAML(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	// Create a simplified structure for YAML output
	output := map[string]interface{}{
		"total_count":     len(pullRequests),
		"pull_requests":   pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

// parsePullRequestResults parses the paginated response into PullRequest structs
func parsePullRequestResults(result *api.PaginatedResponse) ([]*api.PullRequest, error) {
	var pullRequests []*api.PullRequest

	// Parse the Values field (raw JSON) into PullRequest structs
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pull request values: %w", err)
		}

		pullRequests = make([]*api.PullRequest, len(values))
		for i, rawPR := range values {
			var pr api.PullRequest
			if err := json.Unmarshal(rawPR, &pr); err != nil {
				return nil, fmt.Errorf("failed to unmarshal pull request %d: %w", i, err)
			}
			pullRequests[i] = &pr
		}
	}

	return pullRequests, nil
}

// validateState validates the state filter (GitHub CLI compatible)
func validateState(state string) error {
	validStates := []string{
		"open", "merged", "declined", "all",
	}

	stateLower := strings.ToLower(state)
	for _, validState := range validStates {
		if stateLower == validState {
			return nil
		}
	}

	return fmt.Errorf("invalid state '%s'. Valid states are: %s", 
		state, strings.Join(validStates, ", "))
}

// handlePullRequestAPIError provides user-friendly error messages for API errors
func handlePullRequestAPIError(err error) error {
	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		switch bitbucketErr.Type {
		case api.ErrorTypeNotFound:
			return fmt.Errorf("repository not found or no pull requests exist. Verify the repository exists and you have access")
		case api.ErrorTypeAuthentication:
			return fmt.Errorf("authentication failed. Please run 'bt auth login' to authenticate")
		case api.ErrorTypePermission:
			return fmt.Errorf("permission denied. You may not have access to this repository")
		case api.ErrorTypeRateLimit:
			return fmt.Errorf("rate limit exceeded. Please wait before making more requests")
		default:
			return fmt.Errorf("API error: %s", bitbucketErr.Message)
		}
	}

	return fmt.Errorf("failed to list pull requests: %w", err)
}

// renderCustomTable renders a table with proper alignment
func renderCustomTable(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Render header
	for i, header := range headers {
		fmt.Printf("%-*s", colWidths[i], header)
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	// Render separator
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	// Render rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Printf("%-*s", colWidths[i], cell)
				if i < len(row)-1 {
					fmt.Print("  ")
				}
			}
		}
		fmt.Println()
	}

	return nil
}