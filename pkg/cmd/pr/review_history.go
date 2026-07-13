package pr

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

// ReviewHistoryCmd enumerates every pull request in the repository (all
// authors) and returns the comments a given author left across them. It is the
// mining primitive for "show me everything I reviewed here" — the review voice
// lives in comments on other people's PRs, which a single-PR command can't
// surface and Bitbucket's reviewer/participant query filters can't reliably
// enumerate. Filtering is by comment authorship, so it also catches PRs where
// the author commented without being a formally-requested reviewer.
type ReviewHistoryCmd struct {
	Author      string `help:"Author whose comments to collect (username, nickname, display name, account_id, or @me)" default:"@me"`
	State       string `help:"PR state to scan (open, merged, declined, all)" default:"merged"`
	Output      string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Concurrency int    `help:"Parallel PRs to fetch comments for" default:"8"`
	NoColor     bool
	Workspace   string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository  string `help:"Repository name (defaults to git remote)"`
}

// ReviewHistoryComment is one mined comment, flattened for downstream tooling.
type ReviewHistoryComment struct {
	PR     int                           `json:"pr"`
	ID     int                           `json:"id"`
	Kind   string                        `json:"kind"` // "inline" or "review"
	Path   string                        `json:"path,omitempty"`
	Line   int                           `json:"line,omitempty"`
	Body   string                        `json:"body"`
	Date   string                        `json:"date,omitempty"`
	Inline *api.PullRequestCommentInline `json:"inline,omitempty"`
}

func (cmd *ReviewHistoryCmd) Run(ctx context.Context) error {
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

	author, err := ResolveAuthor(ctx, prCtx.Client, cmd.Author)
	if err != nil {
		return err
	}

	state := ""
	if cmd.State != "" && cmd.State != "all" {
		if err := validateState(cmd.State); err != nil {
			return err
		}
		state = cmd.State
	}

	prs, err := prCtx.Client.PullRequests.GetAllPullRequests(ctx, prCtx.Workspace, prCtx.Repository, state)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	concurrency := cmd.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		mu    sync.Mutex
		mined []ReviewHistoryComment
		wg    sync.WaitGroup
		sem   = make(chan struct{}, concurrency)
	)

	for _, pr := range prs {
		wg.Add(1)
		sem <- struct{}{}
		go func(prID int) {
			defer wg.Done()
			defer func() { <-sem }()

			comments, err := prCtx.Client.PullRequests.GetAllComments(ctx, prCtx.Workspace, prCtx.Repository, prID)
			if err != nil {
				return // skip PRs we can't read; a partial history beats a hard failure
			}
			comments = filterDeletedComments(comments)
			comments = FilterCommentsByAuthor(comments, author)

			local := make([]ReviewHistoryComment, 0, len(comments))
			for _, c := range comments {
				rc := ReviewHistoryComment{PR: prID, ID: c.ID, Kind: "review"}
				if c.Content != nil {
					rc.Body = c.Content.Raw
				}
				if c.CreatedOn != nil {
					rc.Date = c.CreatedOn.Format("2006-01-02T15:04:05Z07:00")
				}
				if c.Inline != nil {
					rc.Kind = "inline"
					rc.Path = c.Inline.Path
					rc.Line = c.Inline.To
					if rc.Line == 0 {
						rc.Line = c.Inline.From
					}
					rc.Inline = c.Inline
				}
				local = append(local, rc)
			}

			if len(local) > 0 {
				mu.Lock()
				mined = append(mined, local...)
				mu.Unlock()
			}
		}(pr.ID)
	}
	wg.Wait()

	sort.Slice(mined, func(i, j int) bool {
		if mined[i].PR != mined[j].PR {
			return mined[i].PR > mined[j].PR
		}
		return mined[i].ID < mined[j].ID
	})

	switch cmd.Output {
	case "json", "yaml":
		return prCtx.Formatter.Format(mined)
	default:
		return cmd.display(mined, author, len(prs))
	}
}

func (cmd *ReviewHistoryCmd) display(mined []ReviewHistoryComment, author string, prCount int) error {
	if len(mined) == 0 {
		fmt.Printf("No comments by %q across %d %s PR(s)\n", author, prCount, cmd.State)
		return nil
	}
	fmt.Printf("%d comment(s) by %q across %d %s PR(s):\n", len(mined), author, prCount, cmd.State)
	lastPR := -1
	for _, c := range mined {
		if c.PR != lastPR {
			fmt.Printf("\nPR #%d:\n", c.PR)
			lastPR = c.PR
		}
		loc := ""
		if c.Kind == "inline" {
			loc = fmt.Sprintf(" [%s:%d]", c.Path, c.Line)
		}
		fmt.Printf("  - %s%s\n", firstLine(c.Body), loc)
	}
	return nil
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
