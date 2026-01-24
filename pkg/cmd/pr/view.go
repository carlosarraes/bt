package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
)

// ViewCmd handles the pr view command
type ViewCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Web        bool   `help:"Open pull request in browser"`
	Comments   bool   `help:"Show comments with the pull request"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the pr view command
func (cmd *ViewCmd) Run(ctx context.Context) error {
	// Create PR context with authentication and configuration
	prCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor)
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

	// Parse PR ID
	prID, err := cmd.ParsePRID()
	if err != nil {
		return err
	}

	// Handle web flag first - open in browser and exit
	if cmd.Web {
		return cmd.openInBrowser(prCtx, prID)
	}

	// Fetch PR details
	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	// Fetch additional data if needed
	var files *api.PullRequestDiffStat
	var comments *api.PaginatedResponse

	// Always fetch file statistics for comprehensive view
	files, err = prCtx.Client.PullRequests.GetPullRequestFiles(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		// Don't fail the command if file stats aren't available
		files = nil
	}

	// Fetch comments if requested or for table output
	if cmd.Comments || cmd.Output == "table" {
		comments, err = prCtx.Client.PullRequests.GetComments(ctx, prCtx.Workspace, prCtx.Repository, prID)
		if err != nil {
			// Don't fail the command if comments aren't available
			comments = nil
		}
	}

	// Format and display output
	return cmd.formatOutput(prCtx, pr, files, comments)
}

// parsePRID parses the PR ID argument
func (cmd *ViewCmd) ParsePRID() (int, error) {
	if cmd.PRID == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

	// Remove # prefix if present (GitHub CLI compatibility)
	prIDStr := strings.TrimPrefix(cmd.PRID, "#")

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s': must be a positive integer", cmd.PRID)
	}

	if prID <= 0 {
		return 0, fmt.Errorf("pull request ID must be positive, got %d", prID)
	}

	return prID, nil
}

// openInBrowser opens the PR in the default browser
func (cmd *ViewCmd) openInBrowser(prCtx *PRContext, prID int) error {
	// Construct Bitbucket web URL
	url := fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d",
		prCtx.Workspace, prCtx.Repository, prID)

	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform for opening browser")
	}

	if err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	fmt.Printf("Opening %s in your browser.\n", url)
	return nil
}

// formatOutput formats and displays the PR details
func (cmd *ViewCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest, files *api.PullRequestDiffStat, comments *api.PaginatedResponse) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, pr, files, comments)
	case "json":
		return cmd.formatJSON(prCtx, pr, files, comments)
	case "yaml":
		return cmd.formatYAML(prCtx, pr, files, comments)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

// formatTable formats PR details as a human-readable table
func (cmd *ViewCmd) formatTable(prCtx *PRContext, pr *api.PullRequest, files *api.PullRequestDiffStat, comments *api.PaginatedResponse) error {
	// PR Header
	fmt.Printf("#%d • %s\n", pr.ID, pr.Title)
	fmt.Printf("State: %s\n", pr.State)

	// Author information
	if pr.Author != nil {
		authorName := pr.Author.DisplayName
		if authorName == "" {
			authorName = pr.Author.Username
		}
		fmt.Printf("Author: %s\n", authorName)
	}

	// Branch information
	if pr.Source != nil && pr.Destination != nil {
		sourceBranch := "unknown"
		destBranch := "unknown"

		if pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
		}
		if pr.Destination.Branch != nil {
			destBranch = pr.Destination.Branch.Name
		}

		fmt.Printf("Branches: %s → %s\n", sourceBranch, destBranch)
	}

	// Timestamps
	if pr.CreatedOn != nil {
		fmt.Printf("Created: %s\n", output.FormatRelativeTime(pr.CreatedOn))
	}
	if pr.UpdatedOn != nil {
		fmt.Printf("Updated: %s\n", output.FormatRelativeTime(pr.UpdatedOn))
	}

	// Description
	if pr.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", pr.Description)
	}

	// Reviewers
	if len(pr.Reviewers) > 0 {
		fmt.Printf("\nReviewers:\n")
		for _, reviewer := range pr.Reviewers {
			if reviewer.User != nil {
				name := reviewer.User.DisplayName
				if name == "" {
					name = reviewer.User.Username
				}

				status := "pending"
				if reviewer.Approved {
					status = "approved"
				} else if reviewer.State == "changes_requested" {
					status = "changes requested"
				}

				fmt.Printf("  • %s (%s)\n", name, status)
			}
		}
	}

	// File changes
	if files != nil {
		fmt.Printf("\nFiles changed: %d\n", files.FilesChanged)
		if files.LinesAdded > 0 || files.LinesRemoved > 0 {
			fmt.Printf("Lines: +%d -%d\n", files.LinesAdded, files.LinesRemoved)
		}
	}

	// Comments count
	commentCount := pr.CommentCount
	if commentCount > 0 {
		fmt.Printf("\nComments: %d\n", commentCount)
	}

	// Show comments if requested
	if cmd.Comments && comments != nil {
		if err := cmd.displayComments(comments); err != nil {
			return fmt.Errorf("failed to display comments: %w", err)
		}
	}

	return nil
}

// displayComments displays the comments for the PR
func (cmd *ViewCmd) displayComments(comments *api.PaginatedResponse) error {
	if comments == nil || comments.Values == nil {
		return nil
	}

	var commentsData []json.RawMessage
	if err := json.Unmarshal(comments.Values, &commentsData); err != nil {
		return fmt.Errorf("failed to unmarshal comments: %w", err)
	}

	if len(commentsData) > 0 {
		fmt.Printf("\nComments:\n")
		for i, rawComment := range commentsData {
			var comment api.PullRequestComment
			if err := json.Unmarshal(rawComment, &comment); err != nil {
				continue // Skip malformed comments
			}

			// Comment header
			authorName := "Unknown"
			if comment.User != nil {
				if comment.User.DisplayName != "" {
					authorName = comment.User.DisplayName
				} else if comment.User.Username != "" {
					authorName = comment.User.Username
				}
			}

			timeStr := ""
			if comment.CreatedOn != nil {
				timeStr = output.FormatRelativeTime(comment.CreatedOn)
			}

			fmt.Printf("\n%s (%s):\n", authorName, timeStr)

			// Comment content
			if comment.Content != nil && comment.Content.Raw != "" {
				// Indent comment content
				lines := strings.Split(comment.Content.Raw, "\n")
				for _, line := range lines {
					fmt.Printf("  %s\n", line)
				}
			}

			// Inline comment info
			if comment.Inline != nil {
				fmt.Printf("  [Inline comment on %s:%d]\n", comment.Inline.Path, comment.Inline.To)
			}

			// Add separator between comments (except for the last one)
			if i < len(commentsData)-1 {
				fmt.Println("  ---")
			}
		}
	}

	return nil
}

// formatJSON formats PR details as JSON
func (cmd *ViewCmd) formatJSON(prCtx *PRContext, pr *api.PullRequest, files *api.PullRequestDiffStat, comments *api.PaginatedResponse) error {
	// Create a comprehensive output structure
	output := map[string]interface{}{
		"pull_request": pr,
	}

	if files != nil {
		output["files"] = files
	}

	if comments != nil {
		// Parse comments for JSON output
		var commentsData []json.RawMessage
		if comments.Values != nil {
			if err := json.Unmarshal(comments.Values, &commentsData); err == nil {
				parsedComments := make([]api.PullRequestComment, 0, len(commentsData))
				for _, rawComment := range commentsData {
					var comment api.PullRequestComment
					if err := json.Unmarshal(rawComment, &comment); err == nil {
						parsedComments = append(parsedComments, comment)
					}
				}
				output["comments"] = parsedComments
			}
		}
	}

	return prCtx.Formatter.Format(output)
}

// formatYAML formats PR details as YAML
func (cmd *ViewCmd) formatYAML(prCtx *PRContext, pr *api.PullRequest, files *api.PullRequestDiffStat, comments *api.PaginatedResponse) error {
	// Create a comprehensive output structure (same as JSON)
	output := map[string]interface{}{
		"pull_request": pr,
	}

	if files != nil {
		output["files"] = files
	}

	if comments != nil {
		// Parse comments for YAML output
		var commentsData []json.RawMessage
		if comments.Values != nil {
			if err := json.Unmarshal(comments.Values, &commentsData); err == nil {
				parsedComments := make([]api.PullRequestComment, 0, len(commentsData))
				for _, rawComment := range commentsData {
					var comment api.PullRequestComment
					if err := json.Unmarshal(rawComment, &comment); err == nil {
						parsedComments = append(parsedComments, comment)
					}
				}
				output["comments"] = parsedComments
			}
		}
	}

	return prCtx.Formatter.Format(output)
}
