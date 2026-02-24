package cmd

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/cmd/auth"
	"github.com/carlosarraes/bt/pkg/cmd/config"
	"github.com/carlosarraes/bt/pkg/cmd/pick"
	"github.com/carlosarraes/bt/pkg/cmd/pr"
	"github.com/carlosarraes/bt/pkg/cmd/run"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/cmd/skill"
	"github.com/carlosarraes/bt/pkg/version"
)

type VersionCmd struct{}

func (v *VersionCmd) Run(ctx context.Context) error {
	buildInfo := version.GetBuildInfo()
	fmt.Println(buildInfo.String())
	return nil
}

type AuthCmd struct {
	Login   AuthLoginCmd   `cmd:""`
	Logout  AuthLogoutCmd  `cmd:""`
	Status  AuthStatusCmd  `cmd:""`
	Refresh AuthRefreshCmd `cmd:""`
}

type AuthLoginCmd struct {
	WithToken string `help:"Authenticate with a token instead of interactive flow"`
}

func (a *AuthLoginCmd) Run(ctx context.Context) error {
	cmd := &auth.LoginCmd{
		WithToken: a.WithToken,
	}
	return cmd.Run(ctx)
}

type AuthLogoutCmd struct {
	Force bool `short:"f" help:"Force logout without confirmation"`
}

func (a *AuthLogoutCmd) Run(ctx context.Context) error {
	cmd := &auth.LogoutCmd{
		Force: a.Force,
	}
	return cmd.Run(ctx)
}

type AuthStatusCmd struct{}

func (a *AuthStatusCmd) Run(ctx context.Context) error {
	cmd := &auth.StatusCmd{}
	return cmd.Run(ctx)
}

type AuthRefreshCmd struct{}

func (a *AuthRefreshCmd) Run(ctx context.Context) error {
	cmd := &auth.RefreshCmd{}
	return cmd.Run(ctx)
}

type RunCmd struct {
	List   RunListCmd   `cmd:""`
	View   RunViewCmd   `cmd:""`
	Watch  RunWatchCmd  `cmd:""`
	Logs   RunLogsCmd   `cmd:""`
	Cancel RunCancelCmd `cmd:""`
	Rerun  RunRerunCmd  `cmd:""`
	Report RunReportCmd `cmd:""`
}

type RunListCmd struct {
	Status     string `help:"Filter by status (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED)"`
	Branch     string `help:"Filter by branch name"`
	Creator    string `help:"Filter by pipeline creator (display name)"`
	Limit      int    `help:"Maximum number of runs to show" default:"10"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunListCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.ListCmd{
		Status:     r.Status,
		Branch:     r.Branch,
		Creator:    r.Creator,
		Limit:      r.Limit,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunViewCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Watch      bool   `short:"w" help:"Watch for live updates (running pipelines only)"`
	Log        bool   `help:"View full logs for all steps"`
	LogFailed  bool   `name:"log-failed" help:"View logs only for failed steps (last 100 lines)"`
	FullOutput bool   `name:"full-output" help:"Show complete logs (use with --log-failed for full failure logs)"`
	Tests      bool   `short:"t" help:"Show test results and failures"`
	Step       string `help:"View specific step only"`
	Web        bool   `help:"Open pipeline in browser"`
	URL        bool   `help:"Print pipeline URL instead of opening in browser (use with --web)"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunViewCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.ViewCmd{
		PipelineID: r.PipelineID,
		Output:     r.Output,
		NoColor:    noColor,
		Watch:      r.Watch,
		Log:        r.Log,
		LogFailed:  r.LogFailed,
		FullOutput: r.FullOutput,
		Tests:      r.Tests,
		Step:       r.Step,
		Web:        r.Web,
		URL:        r.URL,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunWatchCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output     string `short:"o" help:"Output format (table, json)" enum:"table,json" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunWatchCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.WatchCmd{
		PipelineID: r.PipelineID,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunLogsCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Step       string `help:"Show logs for specific step only"`
	ErrorsOnly bool   `help:"Extract and show errors only"`
	Follow     bool   `short:"f" help:"Follow live logs for running pipelines"`
	Output     string `short:"o" help:"Output format (text, json, yaml)" enum:"text,json,yaml" default:"text"`
	Context    int    `help:"Number of context lines around errors" default:"3"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunLogsCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.LogsCmd{
		PipelineID: r.PipelineID,
		Step:       r.Step,
		ErrorsOnly: r.ErrorsOnly,
		Follow:     r.Follow,
		Output:     r.Output,
		NoColor:    noColor,
		Context:    r.Context,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunCancelCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Force      bool   `short:"f" help:"Force cancellation without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunCancelCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.CancelCmd{
		PipelineID: r.PipelineID,
		Force:      r.Force,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunRerunCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Failed     bool   `help:"Rerun only failed steps"`
	Step       string `help:"Rerun specific step"`
	Force      bool   `short:"f" help:"Force rerun without confirmation"`
	Debug      bool   `help:"Show debug information"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunRerunCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.RerunCmd{
		PipelineID: r.PipelineID,
		Failed:     r.Failed,
		Step:       r.Step,
		Force:      r.Force,
		Debug:      r.Debug,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

type RunReportCmd struct {
	PipelineID        string   `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output            string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Coverage          bool     `help:"Show only coverage-related information"`
	Issues            bool     `help:"Show only code quality issues"`
	Web               bool     `help:"Open SonarCloud dashboard in browser"`
	URL               bool     `help:"Print SonarCloud URL instead of opening browser"`
	CoverageThreshold int      `name:"coverage-threshold" help:"Show only files below N% coverage"`
	Limit             int      `help:"Limit number of files/issues shown" default:"10"`
	NewCodeOnly       bool     `name:"new-code-only" help:"Focus on new code analysis"`
	Severity          []string `help:"Filter issues by severity level (BLOCKER,CRITICAL,MAJOR,MINOR,INFO)"`
	ShowAllLines      bool     `name:"show-all-lines" help:"Show all uncovered lines (not just top 5 per file)"`
	LinesPerFile      int      `name:"lines-per-file" help:"Max lines to show per file" default:"5"`
	NewLinesOnly      bool     `name:"new-lines-only" help:"Only show NEW uncovered lines from this PR"`
	MinUncoveredLines int      `name:"min-uncovered-lines" help:"Only show files with N+ uncovered lines"`
	MaxUncoveredLines int      `name:"max-uncovered-lines" help:"Only show files with ≤N uncovered lines (quick wins)"`
	FilePattern       string   `name:"file" help:"Filter to specific files (glob pattern)"`
	NoLineDetails     bool     `name:"no-line-details" help:"Skip line-by-line breakdown (performance)"`
	TruncateLines     int      `name:"truncate-lines" help:"Truncate code lines after N characters" default:"80"`
	Debug             bool     `help:"Enable debug output for troubleshooting"`
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

func (r *RunReportCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &run.ReportCmd{
		PipelineID:        r.PipelineID,
		Output:            r.Output,
		NoColor:           noColor,
		Coverage:          r.Coverage,
		Issues:            r.Issues,
		Web:               r.Web,
		URL:               r.URL,
		CoverageThreshold: r.CoverageThreshold,
		Limit:             r.Limit,
		NewCodeOnly:       r.NewCodeOnly,
		Severity:          r.Severity,
		ShowAllLines:      r.ShowAllLines,
		LinesPerFile:      r.LinesPerFile,
		NewLinesOnly:      r.NewLinesOnly,
		MinUncoveredLines: r.MinUncoveredLines,
		MaxUncoveredLines: r.MaxUncoveredLines,
		FilePattern:       r.FilePattern,
		NoLineDetails:     r.NoLineDetails,
		TruncateLines:     r.TruncateLines,
		Debug:             r.Debug,
		Workspace:         r.Workspace,
		Repository:        r.Repository,
	}
	return cmd.Run(ctx)
}

type RepoCmd struct{}

func (r *RepoCmd) Run(ctx context.Context) error {
	return fmt.Errorf("repo commands not yet implemented")
}

type PRCmd struct {
	Create       PRCreateCmd       `cmd:""`
	List         PRListCmd         `cmd:""`
	ListAll      PRListAllCmd      `cmd:"list-all"`
	View         PRViewCmd         `cmd:""`
	Open         PROpenCmd         `cmd:""`
	Edit         PREditCmd         `cmd:""`
	Diff         PRDiffCmd         `cmd:""`
	Review       PRReviewCmd       `cmd:""`
	Files        PRFilesCmd        `cmd:""`
	Comment      PRCommentCmd      `cmd:""`
	Merge        PRMergeCmd        `cmd:""`
	Checkout     PRCheckoutCmd     `cmd:""`
	Ready        PRReadyCmd        `cmd:""`
	Checks       PRChecksCmd       `cmd:""`
	Close        PRCloseCmd        `cmd:""`
	Reopen       PRReopenCmd       `cmd:""`
	Status       PRStatusCmd       `cmd:""`
	UpdateBranch PRUpdateBranchCmd `cmd:"update-branch"`
	Lock         PRLockCmd         `cmd:""`
	Unlock       PRUnlockCmd       `cmd:""`
	Report       PRReportCmd       `cmd:""`
}

type PRCreateCmd struct {
	Title             string   `help:"Title of the pull request"`
	Body              string   `help:"Body of the pull request"`
	Base              string   `help:"Base branch for the pull request"`
	Draft             bool     `help:"Create a draft pull request"`
	Reviewer          []string `help:"Reviewers for the pull request"`
	Fill              bool     `help:"Fill title and body from commit messages"`
	AI                bool     `help:"Generate PR description using AI analysis"`
	Jira              string   `help:"Path to JIRA context file (markdown format)"`
	Debug             bool     `help:"Enable debug output for AI generation"`
	NoPush            bool     `name:"no-push" help:"Skip pushing branch to remote"`
	NoEmoji           bool     `name:"no-emoji" help:"Skip emojis in auto-generated titles"`
	CloseSourceBranch bool     `name:"close-source-branch" help:"Close source branch when pull request is merged"`
	Output            string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

func (p *PRCreateCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.CreateCmd{
		Title:             p.Title,
		Body:              p.Body,
		Base:              p.Base,
		Draft:             p.Draft,
		Reviewer:          p.Reviewer,
		Fill:              p.Fill,
		AI:                p.AI,
		Jira:              p.Jira,
		Debug:             p.Debug,
		NoPush:            p.NoPush,
		NoEmoji:           p.NoEmoji,
		CloseSourceBranch: p.CloseSourceBranch,
		Output:            p.Output,
		NoColor:           noColor,
		Workspace:         p.Workspace,
		Repository:        p.Repository,
	}
	return cmd.Run(ctx)
}

type PRListCmd struct {
	State      string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Author     string `help:"Filter by pull request author"`
	Reviewer   string `help:"Filter by pull request reviewer"`
	Limit      int    `help:"Maximum number of pull requests to show" default:"30"`
	Sort       string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	All        bool   `help:"Show all pull requests regardless of author"`
	Debug      bool   `help:"Show debug output"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRListCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ListCmd{
		State:      p.State,
		Author:     p.Author,
		Reviewer:   p.Reviewer,
		Limit:      p.Limit,
		Sort:       p.Sort,
		Output:     p.Output,
		All:        p.All,
		Debug:      p.Debug,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRListAllCmd struct {
	State     string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Limit     int    `help:"Maximum number of pull requests per repository" default:"10"`
	Sort      string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output    string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	URL       bool   `help:"Output URLs in format: <repo:source-branch> <target-branch> <url>"`
	Approved  bool   `help:"Filter to show only approved PRs"`
	Debug     bool   `help:"Show debug output"`
	Workspace string `help:"Bitbucket workspace (defaults to git remote or config)"`
}

func (p *PRListAllCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ListAllCmd{
		State:     p.State,
		Limit:     p.Limit,
		Sort:      p.Sort,
		Output:    p.Output,
		URL:       p.URL,
		Approved:  p.Approved,
		Debug:     p.Debug,
		NoColor:   noColor,
		Workspace: p.Workspace,
	}
	return cmd.Run(ctx)
}

type PRViewCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Web        bool   `help:"Open pull request in browser"`
	Comments   bool   `help:"Show comments with the pull request"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRViewCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ViewCmd{
		PRID:       p.PRID,
		Web:        p.Web,
		Comments:   p.Comments,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PROpenCmd struct {
	PRIDs      []string `arg:"" help:"Pull request IDs (numbers)"`
	Show       bool     `help:"Print URLs instead of opening in browser"`
	Workspace  string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string   `help:"Repository name (defaults to git remote)"`
	Debug      bool     `help:"Show debug output"`
}

func (p *PROpenCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.OpenCmd{
		PRIDs:      p.PRIDs,
		Show:       p.Show,
		Workspace:  p.Workspace,
		Repository: p.Repository,
		Debug:      p.Debug,
		NoColor:    noColor,
	}
	return cmd.Run(ctx)
}

type PREditCmd struct {
	PRID           string   `arg:"" help:"Pull request ID (number)"`
	Title          string   `help:"Edit pull request title"`
	Body           string   `help:"Edit pull request description"`
	BodyFile       string   `short:"F" name:"body-file" help:"Read description from file"`
	AddReviewer    []string `name:"add-reviewer" help:"Add reviewer by username"`
	RemoveReviewer []string `name:"remove-reviewer" help:"Remove reviewer by username"`
	Ready          bool     `help:"Mark pull request as ready for review (if draft)"`
	Draft          bool     `help:"Convert pull request to draft"`
	AI             bool     `help:"Generate PR description using AI analysis"`
	Jira           string   `help:"Path to JIRA context file (markdown format)"`
	Debug          bool     `help:"Print debug information including git diff and AI inputs"`
	Output         string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace      string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository     string   `help:"Repository name (defaults to git remote)"`
}

func (p *PREditCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.EditCmd{
		PRID:           p.PRID,
		Title:          p.Title,
		Body:           p.Body,
		BodyFile:       p.BodyFile,
		AddReviewer:    p.AddReviewer,
		RemoveReviewer: p.RemoveReviewer,
		Ready:          p.Ready,
		Draft:          p.Draft,
		AI:             p.AI,
		Jira:           p.Jira,
		Debug:          p.Debug,
		Output:         p.Output,
		NoColor:        noColor,
		Workspace:      p.Workspace,
		Repository:     p.Repository,
	}
	return cmd.Run(ctx)
}

type PRDiffCmd struct {
	PRID         string `arg:"" help:"Pull request ID (number)"`
	NameOnly     bool   `name:"name-only" help:"Show only names of changed files"`
	Patch        bool   `help:"Output in patch format suitable for git apply"`
	File         string `help:"Show diff for specific file only"`
	Color        string `help:"When to use color (always, never, auto)" enum:"always,never,auto" default:"auto"`
	Output       string `short:"o" help:"Output format (diff, json, yaml)" enum:"diff,json,yaml" default:"diff"`
	Page         bool   `help:"Page output through diff-so-fancy and less for enhanced viewing"`
	IncludeTests bool   `name:"include-tests" help:"Include test files in diff (excluded by default)"`
	Workspace    string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository   string `help:"Repository name (defaults to git remote)"`
}

func (p *PRDiffCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.DiffCmd{
		PRID:         p.PRID,
		NameOnly:     p.NameOnly,
		Patch:        p.Patch,
		File:         p.File,
		Color:        p.Color,
		Output:       p.Output,
		Page:         p.Page,
		IncludeTests: p.IncludeTests,
		NoColor:      noColor,
		Workspace:    p.Workspace,
		Repository:   p.Repository,
	}
	return cmd.Run(ctx)
}

type PRReviewCmd struct {
	PRID           string `arg:"" help:"Pull request ID (number)"`
	Approve        bool   `help:"Approve the pull request"`
	RequestChanges bool   `name:"request-changes" help:"Request changes on the pull request"`
	Comment        bool   `help:"Add a comment to the pull request"`
	Body           string `short:"b" help:"Comment body text"`
	BodyFile       string `short:"F" name:"body-file" help:"Read comment body from file"`
	Force          bool   `short:"f" help:"Skip confirmation prompts"`
	Output         string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace      string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository     string `help:"Repository name (defaults to git remote)"`
}

func (p *PRReviewCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ReviewCmd{
		PRID:           p.PRID,
		Approve:        p.Approve,
		RequestChanges: p.RequestChanges,
		Comment:        p.Comment,
		Body:           p.Body,
		BodyFile:       p.BodyFile,
		Force:          p.Force,
		Output:         p.Output,
		NoColor:        noColor,
		Workspace:      p.Workspace,
		Repository:     p.Repository,
	}
	return cmd.Run(ctx)
}

type PRFilesCmd struct {
	PRID       string `arg:"" name:"pr-id" help:"Pull request ID or number (e.g., 123 or #123)"`
	NameOnly   bool   `help:"Show only file names"`
	Filter     string `help:"Filter files by pattern (e.g., '*.go', 'src/**/*.js')"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRFilesCmd) Run(ctx context.Context) error {
	cmd := &pr.FilesCmd{
		PRID:       p.PRID,
		NameOnly:   p.NameOnly,
		Filter:     p.Filter,
		Output:     p.Output,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRCommentCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Body       string `short:"b" help:"Comment body text"`
	BodyFile   string `short:"F" name:"body-file" help:"Read comment body from file"`
	ReplyTo    string `name:"reply-to" help:"Reply to comment ID"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRCommentCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.CommentCmd{
		PRID:       p.PRID,
		Body:       p.Body,
		BodyFile:   p.BodyFile,
		ReplyTo:    p.ReplyTo,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRMergeCmd struct {
	PRID         string `arg:"" help:"Pull request ID (number)"`
	Squash       bool   `help:"Squash commits when merging"`
	DeleteBranch bool   `help:"Delete source branch after merge"`
	Auto         bool   `help:"Automatically merge when checks pass"`
	Force        bool   `short:"f" help:"Skip confirmation prompt"`
	Message      string `short:"m" help:"Custom merge commit message"`
	Output       string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace    string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository   string `help:"Repository name (defaults to git remote)"`
}

func (p *PRMergeCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.MergeCmd{
		PRID:         p.PRID,
		Squash:       p.Squash,
		DeleteBranch: p.DeleteBranch,
		Auto:         p.Auto,
		Force:        p.Force,
		Message:      p.Message,
		Output:       p.Output,
		NoColor:      noColor,
		Workspace:    p.Workspace,
		Repository:   p.Repository,
	}
	return cmd.Run(ctx)
}

type PRCheckoutCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Detach     bool   `help:"Checkout in detached HEAD mode"`
	Force      bool   `short:"f" help:"Force checkout, discarding local changes"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRCheckoutCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.CheckoutCmd{
		PRID:       p.PRID,
		Detach:     p.Detach,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRReadyCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Comment    string `help:"Add a comment when marking as ready"`
	Force      bool   `short:"f" help:"Force mark as ready without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRReadyCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ReadyCmd{
		PRID:       p.PRID,
		Comment:    p.Comment,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRChecksCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Watch      bool   `short:"w" help:"Watch for live updates"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRChecksCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ChecksCmd{
		PRID:       p.PRID,
		Watch:      p.Watch,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRCloseCmd struct {
	PRID         string `arg:"" help:"Pull request ID (number)"`
	Comment      string `short:"c" help:"Comment to add when closing the PR"`
	DeleteBranch bool   `name:"delete-branch" help:"Delete the source branch after closing"`
	Force        bool   `short:"f" help:"Skip confirmation prompt"`
	Output       string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace    string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository   string `help:"Repository name (defaults to git remote)"`
}

func (p *PRCloseCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.CloseCmd{
		PRID:         p.PRID,
		Comment:      p.Comment,
		DeleteBranch: p.DeleteBranch,
		Force:        p.Force,
		Output:       p.Output,
		NoColor:      noColor,
		Workspace:    p.Workspace,
		Repository:   p.Repository,
	}
	return cmd.Run(ctx)
}

type PRReopenCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Comment    string `short:"c" help:"Comment to add when reopening the PR"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRReopenCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.ReopenCmd{
		PRID:       p.PRID,
		Comment:    p.Comment,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRStatusCmd struct {
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRStatusCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.StatusCmd{
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRUpdateBranchCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Force      bool   `short:"f" help:"Force update, overriding safety checks"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRUpdateBranchCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.UpdateBranchCmd{
		PRID:       p.PRID,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRLockCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Reason     string `help:"Reason for locking conversation (off_topic, resolved, spam, too_heated)"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRLockCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.LockCmd{
		PRID:       p.PRID,
		Reason:     p.Reason,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRUnlockCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRUnlockCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &pr.UnlockCmd{
		PRID:       p.PRID,
		Force:      p.Force,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

type PRReportCmd struct {
	PRID              string   `arg:"" help:"Pull request ID (number)"`
	Output            string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Coverage          bool     `help:"Show only coverage-related information"`
	Issues            bool     `help:"Show only code quality issues"`
	Web               bool     `help:"Open SonarCloud dashboard in browser"`
	URL               bool     `help:"Print SonarCloud URL instead of opening browser"`
	CoverageThreshold int      `name:"coverage-threshold" help:"Show only files below N% coverage"`
	Limit             int      `help:"Limit number of files/issues shown" default:"10"`
	NewCodeOnly       bool     `name:"new-code-only" help:"Focus on new code analysis"`
	Severity          []string `help:"Filter issues by severity level (BLOCKER,CRITICAL,MAJOR,MINOR,INFO)"`
	ShowAllLines      bool     `name:"show-all-lines" help:"Show all uncovered lines (not just top 5 per file)"`
	LinesPerFile      int      `name:"lines-per-file" help:"Max lines to show per file" default:"5"`
	NewLinesOnly      bool     `name:"new-lines-only" help:"Only show NEW uncovered lines from this PR"`
	MinUncoveredLines int      `name:"min-uncovered-lines" help:"Only show files with N+ uncovered lines"`
	MaxUncoveredLines int      `name:"max-uncovered-lines" help:"Only show files with ≤N uncovered lines (quick wins)"`
	FilePattern       string   `name:"file" help:"Filter to specific files (glob pattern)"`
	NoLineDetails     bool     `name:"no-line-details" help:"Skip line-by-line breakdown (performance)"`
	TruncateLines     int      `name:"truncate-lines" help:"Truncate code lines after N characters" default:"80"`
	Context           int      `name:"context" help:"Show N lines of context around each uncovered line"`
	Debug             bool     `help:"Enable debug output for troubleshooting"`
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

func (p *PRReportCmd) Run(ctx context.Context) error {
	cmd := &pr.ReportCmd{
		PRID:              p.PRID,
		Output:            p.Output,
		Coverage:          p.Coverage,
		Issues:            p.Issues,
		Web:               p.Web,
		URL:               p.URL,
		CoverageThreshold: p.CoverageThreshold,
		Limit:             p.Limit,
		NewCodeOnly:       p.NewCodeOnly,
		Severity:          p.Severity,
		ShowAllLines:      p.ShowAllLines,
		LinesPerFile:      p.LinesPerFile,
		NewLinesOnly:      p.NewLinesOnly,
		MinUncoveredLines: p.MinUncoveredLines,
		MaxUncoveredLines: p.MaxUncoveredLines,
		FilePattern:       p.FilePattern,
		NoLineDetails:     p.NoLineDetails,
		TruncateLines:     p.TruncateLines,
		Context:           p.Context,
		Debug:             p.Debug,
		Workspace:         p.Workspace,
		Repository:        p.Repository,
	}
	return cmd.Run(ctx)
}

type APICmd struct{}

func (a *APICmd) Run(ctx context.Context) error {
	return fmt.Errorf("api command not yet implemented")
}

type BrowseCmd struct{}

func (b *BrowseCmd) Run(ctx context.Context) error {
	return fmt.Errorf("browse command not yet implemented")
}

type ConfigCmd struct {
	Get   ConfigGetCmd   `cmd:""`
	Set   ConfigSetCmd   `cmd:""`
	List  ConfigListCmd  `cmd:""`
	Unset ConfigUnsetCmd `cmd:""`
}

type ConfigGetCmd struct {
	Key    string `arg:"" help:"Configuration key to retrieve (e.g., auth.default_workspace)"`
	Output string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
}

func (c *ConfigGetCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &config.GetCmd{
		Key:     c.Key,
		Output:  c.Output,
		NoColor: noColor,
	}
	return cmd.Run(ctx)
}

type ConfigSetCmd struct {
	Key   string `arg:"" help:"Configuration key to set (e.g., auth.default_workspace)"`
	Value string `arg:"" help:"Configuration value to set"`
}

func (c *ConfigSetCmd) Run(ctx context.Context) error {
	cmd := &config.SetCmd{
		Key:   c.Key,
		Value: c.Value,
	}
	return cmd.Run(ctx)
}

type ConfigListCmd struct {
	Output string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
}

func (c *ConfigListCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)

	cmd := &config.ListCmd{
		Output:  c.Output,
		NoColor: noColor,
	}
	return cmd.Run(ctx)
}

type ConfigUnsetCmd struct {
	Key string `arg:"" help:"Configuration key to remove (e.g., auth.default_workspace)"`
}

func (c *ConfigUnsetCmd) Run(ctx context.Context) error {
	cmd := &config.UnsetCmd{
		Key: c.Key,
	}
	return cmd.Run(ctx)
}

type StatusCmd struct{}

func (s *StatusCmd) Run(ctx context.Context) error {
	return fmt.Errorf("status command not yet implemented")
}

type SkillCmd struct {
	Add    SkillAddCmd    `cmd:""`
	Update SkillUpdateCmd `cmd:""`
	Remove SkillRemoveCmd `cmd:""`
	Status SkillStatusCmd `cmd:""`
}

type SkillAddCmd struct {
	Force bool `short:"f" help:"Force overwrite existing non-symlink skill directories"`
}

func (s *SkillAddCmd) Run(ctx context.Context) error {
	cmd := &skill.AddCmd{
		Force: s.Force,
	}
	return cmd.Run(ctx)
}

type SkillUpdateCmd struct{}

func (s *SkillUpdateCmd) Run(ctx context.Context) error {
	cmd := &skill.UpdateCmd{}
	return cmd.Run(ctx)
}

type SkillRemoveCmd struct{}

func (s *SkillRemoveCmd) Run(ctx context.Context) error {
	cmd := &skill.RemoveCmd{}
	return cmd.Run(ctx)
}

type SkillStatusCmd struct{}

func (s *SkillStatusCmd) Run(ctx context.Context) error {
	cmd := &skill.StatusCmd{}
	return cmd.Run(ctx)
}

type PickCmd struct {
	Show     PickShowCmd     `cmd:"" help:"Preview unpicked commits (dry run)"`
	Run      PickRunCmd      `cmd:"" help:"Execute cherry-pick of unpicked commits"`
	Continue PickContinueCmd `cmd:"" help:"Resume cherry-picking after conflict resolution"`
}

type PickShowCmd struct {
	Reverse   bool   `short:"r" help:"Reverse direction: pick from HML to PRD"`
	Latest    bool   `short:"l" help:"Show current user's latest commits (up to 100)"`
	Count     int    `short:"c" help:"Limit number of commits" default:"5"`
	NoFilter  bool   `name:"no-filter" help:"Skip smart deduplication"`
	Today     bool   `help:"Show commits from today only"`
	Yesterday bool   `help:"Show commits from yesterday only"`
	Since     string `help:"Show commits since date (YYYY-MM-DD)"`
	Until     string `help:"Show commits until date (YYYY-MM-DD)"`
	Prefix    string `help:"Override branch prefix"`
	SuffixPrd string `name:"suffix-prd" help:"Override production branch suffix"`
	SuffixHml string `name:"suffix-hml" help:"Override homologation branch suffix"`
	Debug     bool   `help:"Show debug output"`
}

func (p *PickShowCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)
	cmd := &pick.ShowCmd{
		Reverse:   p.Reverse,
		Latest:    p.Latest,
		Count:     p.Count,
		NoFilter:  p.NoFilter,
		Today:     p.Today,
		Yesterday: p.Yesterday,
		Since:     p.Since,
		Until:     p.Until,
		Prefix:    p.Prefix,
		SuffixPrd: p.SuffixPrd,
		SuffixHml: p.SuffixHml,
		Debug:     p.Debug,
		NoColor:   noColor,
	}
	return cmd.Run(ctx)
}

type PickRunCmd struct {
	Reverse     bool   `short:"r" help:"Reverse direction: pick from HML to PRD"`
	Latest      bool   `short:"l" help:"Pick current user's latest commits (up to 100)"`
	Count       int    `short:"c" help:"Limit number of commits" default:"5"`
	NoFilter    bool   `name:"no-filter" help:"Skip smart deduplication"`
	Today       bool   `help:"Pick commits from today only"`
	Yesterday   bool   `help:"Pick commits from yesterday only"`
	Since       string `help:"Pick commits since date (YYYY-MM-DD)"`
	Until       string `help:"Pick commits until date (YYYY-MM-DD)"`
	Prefix      string `help:"Override branch prefix"`
	SuffixPrd   string `name:"suffix-prd" help:"Override production branch suffix"`
	SuffixHml   string `name:"suffix-hml" help:"Override homologation branch suffix"`
	Debug       bool   `help:"Show debug output"`
}

func (p *PickRunCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)
	cmd := &pick.RunCmd{
		Reverse:     p.Reverse,
		Latest:      p.Latest,
		Count:       p.Count,
		NoFilter:    p.NoFilter,
		Today:       p.Today,
		Yesterday:   p.Yesterday,
		Since:       p.Since,
		Until:       p.Until,
		Prefix:      p.Prefix,
		SuffixPrd:   p.SuffixPrd,
		SuffixHml:   p.SuffixHml,
		Debug:       p.Debug,
		NoColor:     noColor,
	}
	return cmd.Run(ctx)
}

type PickContinueCmd struct{}

func (p *PickContinueCmd) Run(ctx context.Context) error {
	noColor := shared.GetNoColor(ctx)
	cmd := &pick.ContinueCmd{NoColor: noColor}
	return cmd.Run(ctx)
}
