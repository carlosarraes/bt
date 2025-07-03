package cmd

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/cmd/auth"
	"github.com/carlosarraes/bt/pkg/cmd/run"
	"github.com/carlosarraes/bt/pkg/version"
)

// VersionCmd handles the version command
type VersionCmd struct{}

// Run executes the version command
func (v *VersionCmd) Run(ctx context.Context) error {
	buildInfo := version.GetBuildInfo()
	fmt.Println(buildInfo.String())
	return nil
}

// AuthCmd represents the auth command group
type AuthCmd struct {
	Login   AuthLoginCmd   `cmd:""`
	Logout  AuthLogoutCmd  `cmd:""`
	Status  AuthStatusCmd  `cmd:""`
	Refresh AuthRefreshCmd `cmd:""`
}

// AuthLoginCmd handles auth login
type AuthLoginCmd struct {
	WithToken string `help:"Authenticate with a token instead of interactive flow"`
}

func (a *AuthLoginCmd) Run(ctx context.Context) error {
	cmd := &auth.LoginCmd{
		WithToken: a.WithToken,
	}
	return cmd.Run(ctx)
}

// AuthLogoutCmd handles auth logout
type AuthLogoutCmd struct {
	Force bool `short:"f" help:"Force logout without confirmation"`
}

func (a *AuthLogoutCmd) Run(ctx context.Context) error {
	cmd := &auth.LogoutCmd{
		Force: a.Force,
	}
	return cmd.Run(ctx)
}

// AuthStatusCmd handles auth status
type AuthStatusCmd struct{}

func (a *AuthStatusCmd) Run(ctx context.Context) error {
	cmd := &auth.StatusCmd{}
	return cmd.Run(ctx)
}

// AuthRefreshCmd handles auth refresh
type AuthRefreshCmd struct{}

func (a *AuthRefreshCmd) Run(ctx context.Context) error {
	cmd := &auth.RefreshCmd{}
	return cmd.Run(ctx)
}

// RunCmd represents the run command group
type RunCmd struct {
	List   RunListCmd   `cmd:""`
	View   RunViewCmd   `cmd:""`
	Watch  RunWatchCmd  `cmd:""`
	Logs   RunLogsCmd   `cmd:""`
	Cancel RunCancelCmd `cmd:""`
}

// RunListCmd handles run list
type RunListCmd struct {
	Status     string `help:"Filter by status (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED)"`
	Branch     string `help:"Filter by branch name"`
	Limit      int    `help:"Maximum number of runs to show" default:"10"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunListCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
	cmd := &run.ListCmd{
		Status:     r.Status,
		Branch:     r.Branch,
		Limit:      r.Limit,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

// RunViewCmd handles run view
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
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunViewCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
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
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

// RunWatchCmd handles run watch
type RunWatchCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output     string `short:"o" help:"Output format (table, json)" enum:"table,json" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunWatchCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
	cmd := &run.WatchCmd{
		PipelineID: r.PipelineID,
		Output:     r.Output,
		NoColor:    noColor,
		Workspace:  r.Workspace,
		Repository: r.Repository,
	}
	return cmd.Run(ctx)
}

// RunLogsCmd handles run logs
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
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
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

// RunCancelCmd handles run cancel
type RunCancelCmd struct{}

func (r *RunCancelCmd) Run(ctx context.Context) error {
	return fmt.Errorf("run cancel not yet implemented")
}

// RepoCmd represents the repo command group
type RepoCmd struct{}

func (r *RepoCmd) Run(ctx context.Context) error {
	return fmt.Errorf("repo commands not yet implemented")
}

// PRCmd represents the pr command group
type PRCmd struct{}

func (p *PRCmd) Run(ctx context.Context) error {
	return fmt.Errorf("pr commands not yet implemented")
}

// APICmd represents the api command
type APICmd struct{}

func (a *APICmd) Run(ctx context.Context) error {
	return fmt.Errorf("api command not yet implemented")
}

// BrowseCmd represents the browse command
type BrowseCmd struct{}

func (b *BrowseCmd) Run(ctx context.Context) error {
	return fmt.Errorf("browse command not yet implemented")
}

// ConfigCmd represents the config command
type ConfigCmd struct{}

func (c *ConfigCmd) Run(ctx context.Context) error {
	return fmt.Errorf("config command not yet implemented")
}

// StatusCmd represents the status command
type StatusCmd struct{}

func (s *StatusCmd) Run(ctx context.Context) error {
	return fmt.Errorf("status command not yet implemented")
}
