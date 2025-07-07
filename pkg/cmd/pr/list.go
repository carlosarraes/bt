package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
)

type ListCmd struct {
	State      string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Author     string `help:"Filter by pull request author"`
	Reviewer   string `help:"Filter by pull request reviewer"`
	Limit      int    `help:"Maximum number of pull requests to show" default:"30"`
	Sort       string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *ListCmd) Run(ctx context.Context) error {
	prCtx, err := NewPRContext(ctx, cmd.Output, cmd.NoColor)
	if err != nil {
		return err
	}

	if cmd.Workspace != "" {
		prCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		prCtx.Repository = cmd.Repository
	}

	if err := prCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	if cmd.State != "" {
		if err := validateState(cmd.State); err != nil {
			return err
		}
	}

	if cmd.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	if cmd.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	options := &api.PullRequestListOptions{
		PageLen: cmd.Limit,
		Page:    1,
		Sort:    "-updated_on",
	}

	if cmd.State != "" && cmd.State != "all" {
		options.State = strings.ToUpper(cmd.State)
	}

	if cmd.Author != "" {
		options.Author = cmd.Author
	}

	if cmd.Reviewer != "" {
		options.Reviewer = cmd.Reviewer
	}

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

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	pullRequests, err := parsePullRequestResults(result)
	if err != nil {
		return fmt.Errorf("failed to parse pull request results: %w", err)
	}

	return cmd.formatOutput(prCtx, pullRequests)
}

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

func (cmd *ListCmd) formatTable(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	if len(pullRequests) == 0 {
		fmt.Println("No pull requests found")
		return nil
	}

	headers := []string{"ID", "Title", "Branch", "Author", "State", "Updated"}
	rows := make([][]string, len(pullRequests))
	
	for i, pr := range pullRequests {
		title := pr.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		sourceBranch := "-"
		if pr.Source != nil && pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
			if len(sourceBranch) > 20 {
				sourceBranch = sourceBranch[:17] + "..."
			}
		}

		author := "-"
		if pr.Author != nil {
			if pr.Author.DisplayName != "" {
				author = pr.Author.DisplayName
			} else if pr.Author.Username != "" {
				author = pr.Author.Username
			}
			if len(author) > 15 {
				author = author[:12] + "..."
			}
		}

		state := pr.State
		if state == "" {
			state = "UNKNOWN"
		}

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

func (cmd *ListCmd) formatJSON(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	output := map[string]interface{}{
		"total_count":     len(pullRequests),
		"pull_requests":   pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ListCmd) formatYAML(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	output := map[string]interface{}{
		"total_count":     len(pullRequests),
		"pull_requests":   pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

func parsePullRequestResults(result *api.PaginatedResponse) ([]*api.PullRequest, error) {
	var pullRequests []*api.PullRequest

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


func renderCustomTable(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

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

	for i, header := range headers {
		fmt.Printf("%-*s", colWidths[i], header)
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	for i, width := range colWidths {
		fmt.Print(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

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
