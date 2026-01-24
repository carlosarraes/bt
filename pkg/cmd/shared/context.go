package shared

import (
	"context"
	"fmt"
	"os"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/git"
	"github.com/carlosarraes/bt/pkg/output"
)

type CommandContext struct {
	Client     *api.Client
	Config     *config.Config
	Workspace  string
	Repository string
	Formatter  output.Formatter
	Debug      bool
}

func NewCommandContext(ctx context.Context, outputFormat string, noColor bool, debug ...bool) (*CommandContext, error) {
	debugEnabled := false
	if len(debug) > 0 {
		debugEnabled = debug[0]
	}

	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	gitRepo, err := git.NewRepository("")
	var workspace, repository string

	if err != nil {
		if debugEnabled {
			fmt.Fprintf(os.Stderr, "DEBUG: Not in git repository, error: %v\n", err)
		}
		if cfg.Auth.DefaultWorkspace == "" {
			return nil, fmt.Errorf("not in a git repository and no default workspace configured. Run 'bt auth login' or set default_workspace in config")
		}
		workspace = cfg.Auth.DefaultWorkspace
		if debugEnabled {
			fmt.Fprintf(os.Stderr, "DEBUG: Using default workspace from config: %s\n", workspace)
		}
	} else {
		workspace = gitRepo.GetWorkspace()
		repository = gitRepo.GetName()

		if debugEnabled {
			fmt.Fprintf(os.Stderr, "DEBUG: Git extracted workspace: %s\n", workspace)
			fmt.Fprintf(os.Stderr, "DEBUG: Git extracted repository: %s\n", repository)

			remotes := gitRepo.GetRemotes()
			fmt.Fprintf(os.Stderr, "DEBUG: Git remotes found: %d\n", len(remotes))
			for name, remote := range remotes {
				fmt.Fprintf(os.Stderr, "DEBUG: Remote %s: %s (workspace: %s, repo: %s)\n", name, remote.URL, remote.Workspace, remote.RepoName)
			}
		}

		if workspace == "" || repository == "" {
			return nil, fmt.Errorf("unable to detect Bitbucket workspace and repository from git remotes")
		}
	}

	authManager, err := CreateAuthManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	clientConfig := api.DefaultClientConfig()
	clientConfig.BaseURL = cfg.API.BaseURL
	clientConfig.Timeout = cfg.API.Timeout

	client, err := api.NewClient(authManager, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	formatterOpts := &output.FormatterOptions{
		NoColor: noColor,
	}

	formatter, err := output.NewFormatter(output.Format(outputFormat), formatterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create output formatter: %w", err)
	}

	return &CommandContext{
		Client:     client,
		Config:     cfg,
		Workspace:  workspace,
		Repository: repository,
		Formatter:  formatter,
		Debug:      debugEnabled,
	}, nil
}

func (c *CommandContext) ValidateWorkspaceAndRepo() error {
	if c.Workspace == "" {
		return fmt.Errorf("workspace not specified. Either run from a git repository with Bitbucket remote or configure default_workspace")
	}
	if c.Repository == "" {
		return fmt.Errorf("repository not specified. Either run from a git repository or use --repo flag")
	}
	return nil
}
