package cmd

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/cmd/auth"
	"github.com/carlosarraes/bt/pkg/cmd/config"
	"github.com/carlosarraes/bt/pkg/cmd/pr"
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
type RunCancelCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Force      bool   `short:"f" help:"Force cancellation without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (r *RunCancelCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
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

// RepoCmd represents the repo command group
type RepoCmd struct{}

func (r *RepoCmd) Run(ctx context.Context) error {
	return fmt.Errorf("repo commands not yet implemented")
}

// PRCmd represents the pr command group
type PRCmd struct {
	List PRListCmd `cmd:""`
	View PRViewCmd `cmd:""`
}

// PRListCmd handles pr list
type PRListCmd struct {
	State      string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Author     string `help:"Filter by pull request author"`
	Reviewer   string `help:"Filter by pull request reviewer"`
	Limit      int    `help:"Maximum number of pull requests to show" default:"30"`
	Sort       string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRListCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
	cmd := &pr.ListCmd{
		State:      p.State,
		Author:     p.Author,
		Reviewer:   p.Reviewer,
		Limit:      p.Limit,
		Sort:       p.Sort,
		Output:     p.Output,
		NoColor:    noColor,
		Workspace:  p.Workspace,
		Repository: p.Repository,
	}
	return cmd.Run(ctx)
}

// PRViewCmd handles pr view
type PRViewCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Web        bool   `help:"Open pull request in browser"`
	Comments   bool   `help:"Show comments with the pull request"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (p *PRViewCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
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

// ConfigCmd represents the config command group
type ConfigCmd struct {
	Get   ConfigGetCmd   `cmd:""`
	Set   ConfigSetCmd   `cmd:""`
	List  ConfigListCmd  `cmd:""`
	Unset ConfigUnsetCmd `cmd:""`
}

// ConfigGetCmd handles config get
type ConfigGetCmd struct {
	Key    string `arg:"" help:"Configuration key to retrieve (e.g., auth.default_workspace)"`
	Output string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
}

func (c *ConfigGetCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
	cmd := &config.GetCmd{
		Key:     c.Key,
		Output:  c.Output,
		NoColor: noColor,
	}
	return cmd.Run(ctx)
}

// ConfigSetCmd handles config set
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

// ConfigListCmd handles config list
type ConfigListCmd struct {
	Output string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
}

func (c *ConfigListCmd) Run(ctx context.Context) error {
	// Get global NoColor from context
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}
	
	cmd := &config.ListCmd{
		Output:  c.Output,
		NoColor: noColor,
	}
	return cmd.Run(ctx)
}

// ConfigUnsetCmd handles config unset
type ConfigUnsetCmd struct {
	Key string `arg:"" help:"Configuration key to remove (e.g., auth.default_workspace)"`
}

func (c *ConfigUnsetCmd) Run(ctx context.Context) error {
	cmd := &config.UnsetCmd{
		Key: c.Key,
	}
	return cmd.Run(ctx)
}

// StatusCmd represents the status command
type StatusCmd struct{}

func (s *StatusCmd) Run(ctx context.Context) error {
	return fmt.Errorf("status command not yet implemented")
}
