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

// commentNode is one comment plus the replies threaded beneath it.
type commentNode struct {
	comment  api.PullRequestComment
	children []*commentNode
}

// buildCommentThreads groups comments into reply trees, preserving the order they
// arrived in. A reply whose parent is missing - deleted, or filtered out by
// --author - is promoted to a root so that it is still shown rather than dropped.
func buildCommentThreads(comments []api.PullRequestComment) []*commentNode {
	nodes := make(map[int]*commentNode, len(comments))
	ordered := make([]*commentNode, 0, len(comments))

	for _, comment := range comments {
		node := &commentNode{comment: comment}
		nodes[comment.ID] = node
		ordered = append(ordered, node)
	}

	var roots []*commentNode
	for _, node := range ordered {
		parentRef := node.comment.Parent
		if parentRef != nil && parentRef.ID != node.comment.ID {
			if parent, ok := nodes[parentRef.ID]; ok {
				parent.children = append(parent.children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	// Defensive: a parent cycle would leave every node attached and nothing to
	// render. Falling back to a flat list is better than silently showing nothing.
	if len(roots) == 0 && len(ordered) > 0 {
		for _, node := range ordered {
			node.children = nil
		}
		return ordered
	}

	return roots
}

func commentAuthor(comment api.PullRequestComment) string {
	if comment.User != nil {
		if comment.User.DisplayName != "" {
			return comment.User.DisplayName
		}
		if comment.User.Username != "" {
			return comment.User.Username
		}
	}
	return "Unknown"
}

// inlineAnchor renders an inline comment's file location, using whichever diff
// side the comment is anchored to.
func inlineAnchor(inline *api.PullRequestCommentInline) string {
	line := inline.To
	if line == 0 {
		line = inline.From
	}
	if line == 0 {
		return inline.Path
	}
	return fmt.Sprintf("%s:%d", inline.Path, line)
}

func (cmd *CommentsCmd) displayComments(comments []api.PullRequestComment, prID int) error {
	if len(comments) == 0 {
		fmt.Printf("No comments on pull request #%d\n", prID)
		return nil
	}

	fmt.Printf("Comments on pull request #%d:\n", prID)

	roots := buildCommentThreads(comments)
	for i, root := range roots {
		fmt.Println()
		renderCommentNode(root, 0)

		if i < len(roots)-1 {
			fmt.Println("  ---")
		}
	}

	return nil
}

func renderCommentNode(node *commentNode, depth int) {
	comment := node.comment

	timeStr := ""
	if comment.CreatedOn != nil {
		timeStr = output.FormatRelativeTime(comment.CreatedOn)
	}

	indent := strings.Repeat("  ", depth)
	header := fmt.Sprintf("#%d %s (%s):", comment.ID, commentAuthor(comment), timeStr)

	bodyIndent := indent + "  "
	if depth == 0 {
		fmt.Printf("%s\n", header)
	} else {
		fmt.Printf("%s└─ %s\n", indent, header)
		bodyIndent = indent + "     "
	}

	// Only annotate an unresolved parent; a nested reply already shows the
	// relationship through its indentation.
	if depth == 0 && comment.Parent != nil {
		fmt.Printf("%s[Reply to comment #%d]\n", bodyIndent, comment.Parent.ID)
	}

	if comment.Inline != nil {
		fmt.Printf("%s[Inline comment on %s]\n", bodyIndent, inlineAnchor(comment.Inline))
	}

	if comment.Content != nil && comment.Content.Raw != "" {
		for _, line := range strings.Split(comment.Content.Raw, "\n") {
			fmt.Printf("%s%s\n", bodyIndent, line)
		}
	}

	for _, child := range node.children {
		renderCommentNode(child, depth+1)
	}
}
