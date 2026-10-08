package pr

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

// ResolveCmd resolves (or, with Unresolve, reopens) comment threads.
type ResolveCmd struct {
	PRID       string
	CommentIDs []string
	AllFrom    string
	Unresolve  bool
	NoColor    bool
	Workspace  string
	Repository string
}

type commentResolver interface {
	ResolveComment(ctx context.Context, workspace, repoSlug string, prID, commentID int) (*api.CommentResolution, error)
	UnresolveComment(ctx context.Context, workspace, repoSlug string, prID, commentID int) error
}

func (cmd *ResolveCmd) Run(ctx context.Context) error {
	if len(cmd.CommentIDs) == 0 && cmd.AllFrom == "" {
		return fmt.Errorf("specify comment IDs or --all-from <author>")
	}
	if len(cmd.CommentIDs) > 0 && cmd.AllFrom != "" {
		return fmt.Errorf("cannot combine comment IDs with --all-from")
	}

	commentIDs, err := parseCommentIDs(cmd.CommentIDs)
	if err != nil {
		return err
	}

	prCtx, err := shared.NewCommandContext(ctx, "table", cmd.NoColor)
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

	prID, err := ParsePRID(cmd.PRID)
	if err != nil {
		return err
	}

	if cmd.AllFrom != "" {
		author, err := ResolveAuthor(ctx, prCtx.Client, cmd.AllFrom)
		if err != nil {
			return err
		}
		comments, err := prCtx.Client.PullRequests.GetAllComments(ctx, prCtx.Workspace, prCtx.Repository, prID)
		if err != nil {
			return handlePullRequestAPIError(err)
		}
		threads := unresolvedThreadsFrom(filterDeletedComments(comments), author)
		if len(threads) == 0 {
			fmt.Printf("No unresolved threads from %q on pull request #%d\n", cmd.AllFrom, prID)
			return nil
		}
		authors := map[string]bool{}
		for _, c := range threads {
			commentIDs = append(commentIDs, c.ID)
			authors[commentAuthor(c)] = true
		}
		names := make([]string, 0, len(authors))
		for name := range authors {
			names = append(names, name)
		}
		fmt.Printf("Resolving %d thread(s) started by: %s\n", len(threads), strings.Join(names, ", "))
	}

	return applyResolution(ctx, prCtx.Client.PullRequests, prCtx.Workspace, prCtx.Repository, prID, commentIDs, cmd.Unresolve, os.Stdout)
}

func parseCommentIDs(raw []string) ([]int, error) {
	out := make([]int, 0, len(raw))
	for _, r := range raw {
		id, err := strconv.Atoi(strings.TrimPrefix(r, "#"))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid comment ID %q: must be a positive integer", r)
		}
		out = append(out, id)
	}
	return out, nil
}

// unresolvedThreadsFrom returns the root comments of unresolved threads the author started.
func unresolvedThreadsFrom(comments []api.PullRequestComment, author string) []api.PullRequestComment {
	var roots []api.PullRequestComment
	for _, c := range comments {
		if c.Parent == nil && c.Resolution == nil && c.User != nil && userMatches(c.User, author) {
			roots = append(roots, c)
		}
	}
	return roots
}

// applyResolution keeps going past individual failures and reports them all at the end.
func applyResolution(ctx context.Context, r commentResolver, workspace, repo string, prID int, commentIDs []int, unresolve bool, w io.Writer) error {
	failed := 0
	for _, cid := range commentIDs {
		var err error
		if unresolve {
			err = r.UnresolveComment(ctx, workspace, repo, prID, cid)
		} else {
			_, err = r.ResolveComment(ctx, workspace, repo, prID, cid)
		}
		switch {
		case err != nil:
			failed++
			fmt.Fprintf(w, "✗ #%d: %v\n", cid, err)
		case unresolve:
			fmt.Fprintf(w, "✓ Reopened #%d\n", cid)
		default:
			fmt.Fprintf(w, "✓ Resolved #%d\n", cid)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d comment(s) failed", failed, len(commentIDs))
	}
	return nil
}
