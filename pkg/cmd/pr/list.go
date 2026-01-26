package pr

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
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
	Debug      bool   `help:"Show debug output"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *ListCmd) Run(ctx context.Context) error {
	prCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor, cmd.Debug)
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

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Using workspace: %s\n", prCtx.Workspace)
		fmt.Fprintf(os.Stderr, "DEBUG: Using repository: %s\n", prCtx.Repository)
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

	if !cmd.All {
		if cmd.Author != "" {
			options.Author = cmd.Author
		} else {
			if prCtx.Client != nil {
				currentUser, err := prCtx.Client.GetAuthManager().GetAuthenticatedUser(ctx)
				if err == nil && currentUser != nil {
					options.Author = currentUser.Username
					if cmd.Debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Filtering by current user: %s\n", currentUser.Username)
						fmt.Fprintf(os.Stderr, "DEBUG: Current user details: %+v\n", currentUser)
					}
				} else {
					if cmd.Debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Could not get current user, showing all PRs: %v\n", err)
					}
				}
			}
		}
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

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: API request options: %+v\n", options)
		fmt.Fprintf(os.Stderr, "DEBUG: Making API request to /repositories/%s/%s/pullrequests\n", prCtx.Workspace, prCtx.Repository)
	}

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		if cmd.Debug {
			fmt.Fprintf(os.Stderr, "DEBUG: API error: %v\n", err)
			if bitbucketErr, ok := err.(*api.BitbucketError); ok {
				fmt.Fprintf(os.Stderr, "DEBUG: Error type: %s\n", bitbucketErr.Type)
				fmt.Fprintf(os.Stderr, "DEBUG: Error message: %s\n", bitbucketErr.Message)
				fmt.Fprintf(os.Stderr, "DEBUG: Error detail: %s\n", bitbucketErr.Detail)
				fmt.Fprintf(os.Stderr, "DEBUG: Status code: %d\n", bitbucketErr.StatusCode)
				fmt.Fprintf(os.Stderr, "DEBUG: Raw response: %s\n", bitbucketErr.Raw)
			}
		}
		return handlePullRequestAPIError(err)
	}

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: API response - Page: %d, PageLen: %d, Size: %d\n", result.Page, result.PageLen, result.Size)
		fmt.Fprintf(os.Stderr, "DEBUG: API response - Next: %s\n", result.Next)
		fmt.Fprintf(os.Stderr, "DEBUG: API response - Values length: %d\n", len(result.Values))
	}

	pullRequests, err := parsePullRequestResults(result)
	if err != nil {
		return fmt.Errorf("failed to parse pull request results: %w", err)
	}

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Parsed %d pull requests\n", len(pullRequests))
		for i, pr := range pullRequests {
			fmt.Fprintf(os.Stderr, "DEBUG: PR %d - ID: %d, Title: %s, State: %s\n", i, pr.ID, pr.Title, pr.State)
		}
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

	headers := []string{"ID", "Title", "Branch", "Author", "State", "Approved", "Mergeable", "Updated"}
	rows := make([][]string, len(pullRequests))

	mergeableResults := cmd.checkMergeableStatusConcurrently(prCtx, pullRequests)

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

		updatedTime := output.FormatRelativeTime(pr.UpdatedOn)

		approved := cmd.isPRApproved(pr)
		approvedStatus := "✗"
		if approved {
			approvedStatus = "✓"
		}

		mergeable := mergeableResults[i]
		mergeableStatus := "✓"
		if !mergeable {
			mergeableStatus = "✗"
		}

		rows[i] = []string{
			fmt.Sprintf("#%d", pr.ID),
			title,
			sourceBranch,
			author,
			state,
			approvedStatus,
			mergeableStatus,
			updatedTime,
		}
	}

	return output.RenderSimpleTable(headers, rows)
}

func (cmd *ListCmd) formatJSON(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	output := map[string]interface{}{
		"total_count":   len(pullRequests),
		"pull_requests": pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ListCmd) formatYAML(prCtx *PRContext, pullRequests []*api.PullRequest) error {
	output := map[string]interface{}{
		"total_count":   len(pullRequests),
		"pull_requests": pullRequests,
	}

	return prCtx.Formatter.Format(output)
}

func parsePullRequestResults(result *api.PaginatedResponse) ([]*api.PullRequest, error) {
	return shared.ParsePaginatedResults[api.PullRequest](result)
}

func validateState(state string) error {
	return shared.ValidateAllowedValue(state, shared.AllowedPRStates, "state")
}

func (cmd *ListCmd) createMinimalContext(ctx context.Context, outputFormat string, noColor bool) (*PRContext, error) {
	return shared.NewMinimalContext(ctx, shared.MinimalContextOptions{
		OutputFormat: outputFormat,
		Workspace:    cmd.Workspace,
		Repository:   cmd.Repository,
		NoColor:      noColor,
		Debug:        cmd.Debug,
	})
}

func (cmd *ListCmd) checkMergeableStatusConcurrently(prCtx *PRContext, pullRequests []*api.PullRequest) []bool {
	results := make([]bool, len(pullRequests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	semaphore := make(chan struct{}, 10)

	for i, pr := range pullRequests {
		wg.Add(1)
		go func(index int, pullRequest *api.PullRequest) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			mergeable := cmd.isPRMergeable(prCtx, pullRequest)

			mu.Lock()
			results[index] = mergeable
			mu.Unlock()
		}(i, pr)
	}

	wg.Wait()
	return results
}

func (cmd *ListCmd) isPRApproved(pr *api.PullRequest) bool {
	if pr.Reviewers != nil {
		for _, reviewer := range pr.Reviewers {
			if reviewer.Approved {
				return true
			}
		}
	}

	if pr.Participants != nil {
		for _, participant := range pr.Participants {
			if participant.Approved {
				return true
			}
		}
	}

	return false
}

func (cmd *ListCmd) isPRMergeable(prCtx *PRContext, pr *api.PullRequest) bool {
	if pr.State != "OPEN" {
		return true
	}

	diffstat, err := prCtx.Client.PullRequests.GetDiffstat(context.Background(), prCtx.Workspace, prCtx.Repository, pr.ID)
	if err != nil {
		if cmd.Debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to get diffstat for PR #%d: %v\n", pr.ID, err)
		}
		return true
	}

	if strings.Contains(strings.ToLower(diffstat.Status), "conflict") {
		return false
	}

	if diffstat.Files != nil {
		for _, file := range diffstat.Files {
			if strings.Contains(strings.ToLower(file.Status), "conflict") {
				return false
			}
		}
	}

	return true
}
