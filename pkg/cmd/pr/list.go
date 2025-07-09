package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/output"
)

type ListCmd struct {
	State      string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Author     string `help:"Filter by pull request author"`
	Reviewer   string `help:"Filter by pull request reviewer"`
	Limit      int    `help:"Maximum number of pull requests to show" default:"30"`
	Sort       string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	All        bool   `help:"Show all pull requests regardless of author"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *ListCmd) Run(ctx context.Context) error {
	prCtx, err := NewPRContext(ctx, cmd.Output, cmd.NoColor)
	if err != nil {
		if cmd.Workspace != "" && cmd.Repository != "" {
			prCtx, err = cmd.createMinimalContext(ctx, cmd.Output, cmd.NoColor)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if cmd.Workspace != "" {
		prCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		prCtx.Repository = cmd.Repository
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Using workspace: %s\n", prCtx.Workspace)
	fmt.Fprintf(os.Stderr, "DEBUG: Using repository: %s\n", prCtx.Repository)

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

	if cmd.Author != "" && !cmd.All {
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

	fmt.Fprintf(os.Stderr, "DEBUG: API request options: %+v\n", options)
	fmt.Fprintf(os.Stderr, "DEBUG: Making API request to /repositories/%s/%s/pullrequests\n", prCtx.Workspace, prCtx.Repository)

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: API error: %v\n", err)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok {
			fmt.Fprintf(os.Stderr, "DEBUG: Error type: %s\n", bitbucketErr.Type)
			fmt.Fprintf(os.Stderr, "DEBUG: Error message: %s\n", bitbucketErr.Message)
			fmt.Fprintf(os.Stderr, "DEBUG: Error detail: %s\n", bitbucketErr.Detail)
			fmt.Fprintf(os.Stderr, "DEBUG: Status code: %d\n", bitbucketErr.StatusCode)
			fmt.Fprintf(os.Stderr, "DEBUG: Raw response: %s\n", bitbucketErr.Raw)
		}
		return handlePullRequestAPIError(err)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: API response - Page: %d, PageLen: %d, Size: %d\n", result.Page, result.PageLen, result.Size)
	fmt.Fprintf(os.Stderr, "DEBUG: API response - Next: %s\n", result.Next)
	fmt.Fprintf(os.Stderr, "DEBUG: API response - Values length: %d\n", len(result.Values))

	pullRequests, err := parsePullRequestResults(result)
	if err != nil {
		return fmt.Errorf("failed to parse pull request results: %w", err)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Parsed %d pull requests\n", len(pullRequests))
	for i, pr := range pullRequests {
		fmt.Fprintf(os.Stderr, "DEBUG: PR %d - ID: %d, Title: %s, State: %s\n", i, pr.ID, pr.Title, pr.State)
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

func (cmd *ListCmd) createMinimalContext(ctx context.Context, outputFormat string, noColor bool) (*PRContext, error) {
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	authManager, err := createAuthManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	clientConfig := api.DefaultClientConfig()
	clientConfig.BaseURL = cfg.API.BaseURL
	clientConfig.Timeout = cfg.API.Timeout

	client, err := api.NewClient(authManager, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	formatterOpts := &output.FormatterOptions{
		NoColor: noColor,
	}
	
	formatter, err := output.NewFormatter(output.Format(outputFormat), formatterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create output formatter: %w", err)
	}

	return &PRContext{
		Client:     client,
		Config:     cfg,
		Workspace:  cmd.Workspace,
		Repository: cmd.Repository,
		Formatter:  formatter,
	}, nil
}
