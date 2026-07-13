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
	Author     string `help:"Only show comments by this author (username, nickname, display name, account_id, or @me)"`
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

	if cmd.Author != "" {
		author, err := ResolveAuthor(ctx, prCtx.Client, cmd.Author)
		if err != nil {
			return err
		}
		comments = FilterCommentsByAuthor(comments, author)
	}

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

// ResolveAuthor turns an author selector into a concrete match string. "@me"
// (or "me") resolves to the authenticated user's account_id via the auth
// manager; anything else is returned unchanged for case-insensitive matching
// against a comment author's username, nickname, display name, or account_id.
func ResolveAuthor(ctx context.Context, client *api.Client, selector string) (string, error) {
	if selector == "@me" || selector == "me" {
		user, err := client.GetAuthManager().GetAuthenticatedUser(ctx)
		if err != nil {
			return "", fmt.Errorf("could not resolve @me to the authenticated user: %w", err)
		}
		if user.AccountID != "" {
			return user.AccountID, nil
		}
		if user.Username != "" {
			return user.Username, nil
		}
		return user.DisplayName, nil
	}
	return selector, nil
}

// FilterCommentsByAuthor keeps only comments whose author matches (case-
// insensitively) the given selector against any of the identity fields
// Bitbucket may populate — username, nickname, display name, or account_id.
func FilterCommentsByAuthor(comments []api.PullRequestComment, author string) []api.PullRequestComment {
	filtered := make([]api.PullRequestComment, 0, len(comments))
	for _, comment := range comments {
		if comment.User != nil && userMatches(comment.User, author) {
			filtered = append(filtered, comment)
		}
	}
	return filtered
}

// userMatches reports whether a user's identity fields match the selector.
func userMatches(u *api.User, selector string) bool {
	s := strings.ToLower(selector)
	for _, field := range []string{u.Username, u.Nickname, u.DisplayName, u.AccountID, u.UUID} {
		if field != "" && strings.ToLower(field) == s {
			return true
		}
	}
	return false
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
