package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
)

type CommentsCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *CommentsCmd) Run(ctx context.Context) error {
	prCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor)
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

	prID, err := cmd.ParsePRID()
	if err != nil {
		return err
	}

	comments, err := prCtx.Client.PullRequests.GetAllComments(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	comments = filterDeletedComments(comments)

	switch cmd.Output {
	case "json", "yaml":
		return prCtx.Formatter.Format(comments)
	default:
		return cmd.displayComments(comments, prID)
	}
}

func (cmd *CommentsCmd) ParsePRID() (int, error) {
	if cmd.PRID == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

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

// filterDeletedComments removes tombstones Bitbucket keeps for deleted comments
func filterDeletedComments(comments []api.PullRequestComment) []api.PullRequestComment {
	filtered := make([]api.PullRequestComment, 0, len(comments))
	for _, comment := range comments {
		if !comment.Deleted {
			filtered = append(filtered, comment)
		}
	}
	return filtered
}

func (cmd *CommentsCmd) displayComments(comments []api.PullRequestComment, prID int) error {
	if len(comments) == 0 {
		fmt.Printf("No comments on pull request #%d\n", prID)
		return nil
	}

	fmt.Printf("Comments on pull request #%d:\n", prID)
	for i, comment := range comments {
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

		fmt.Printf("\n#%d %s (%s):\n", comment.ID, authorName, timeStr)

		if comment.Parent != nil {
			fmt.Printf("  [Reply to comment #%d]\n", comment.Parent.ID)
		}

		if comment.Inline != nil {
			fmt.Printf("  [Inline comment on %s:%d]\n", comment.Inline.Path, comment.Inline.To)
		}

		if comment.Content != nil && comment.Content.Raw != "" {
			lines := strings.Split(comment.Content.Raw, "\n")
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
		}

		if i < len(comments)-1 {
			fmt.Println("  ---")
		}
	}

	return nil
}
