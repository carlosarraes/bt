package run

import (
	"context"
	"fmt"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/git"
	"github.com/carlosarraes/bt/pkg/output"
)

// RunContext holds the common context needed for run commands
type RunContext struct {
	Client     *api.Client
	Config     *config.Config
	Workspace  string
	Repository string
	Formatter  output.Formatter
}

// NewRunContext creates a new run context with authentication and configuration
func NewRunContext(ctx context.Context, outputFormat string, noColor bool) (*RunContext, error) {
	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get repository context from git
	gitRepo, err := git.NewRepository("")
	var workspace, repository string
	
	if err != nil {
		// If not in a git repository, use configuration defaults
		if cfg.Auth.DefaultWorkspace == "" {
			return nil, fmt.Errorf("not in a git repository and no default workspace configured. Run 'bt auth login' or set default_workspace in config")
		}
		workspace = cfg.Auth.DefaultWorkspace
		// Repository will need to be specified via flags or context
	} else {
		// Extract workspace and repository from git
		workspace = gitRepo.GetWorkspace()
		repository = gitRepo.GetName()
		
		if workspace == "" || repository == "" {
			return nil, fmt.Errorf("unable to detect Bitbucket workspace and repository from git remotes")
		}
	}

	// Create authenticated API client
	authManager, err := createAuthManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	// Configure API client
	clientConfig := api.DefaultClientConfig()
	clientConfig.BaseURL = cfg.API.BaseURL
	clientConfig.Timeout = cfg.API.Timeout

	client, err := api.NewClient(authManager, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create output formatter
	formatterOpts := &output.FormatterOptions{
		NoColor: noColor,
	}
	
	formatter, err := output.NewFormatter(output.Format(outputFormat), formatterOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create output formatter: %w", err)
	}

	return &RunContext{
		Client:     client,
		Config:     cfg,
		Workspace:  workspace,
		Repository: repository,
		Formatter:  formatter,
	}, nil
}

// createAuthManager creates an auth manager using stored credentials
func createAuthManager() (auth.AuthManager, error) {
	// Create file-based credential storage
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		return nil, err
	}

	// Check if credentials exist
	if !storage.Exists("auth") {
		return nil, fmt.Errorf("no stored credentials found. Please run 'bt auth login' first")
	}

	// Load existing credentials to determine auth method
	var credentials auth.StoredCredentials
	if err := storage.Retrieve("auth", &credentials); err != nil {
		return nil, fmt.Errorf("failed to load stored credentials: %w", err)
	}

	// Create config with the appropriate method
	config := auth.DefaultConfig()
	config.Method = credentials.Method

	// Create and return the auth manager
	return auth.NewAuthManager(config, storage)
}

// PipelineStateColor returns the appropriate color for a pipeline state
func PipelineStateColor(state string) string {
	switch state {
	case "SUCCESSFUL":
		return "green"
	case "FAILED":
		return "red"
	case "ERROR":
		return "red"
	case "STOPPED":
		return "yellow"
	case "IN_PROGRESS":
		return "blue"
	case "PENDING":
		return "cyan"
	default:
		return "white"
	}
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// FormatRelativeTime formats a time as relative to now (e.g., "3 hours ago")
func FormatRelativeTime(t *time.Time) string {
	if t == nil {
		return "-"
	}

	now := time.Now()
	diff := now.Sub(*t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("2006-01-02")
	}
}

// ValidateWorkspaceAndRepo ensures workspace and repository are available
func (rc *RunContext) ValidateWorkspaceAndRepo() error {
	if rc.Workspace == "" {
		return fmt.Errorf("workspace not specified. Either run from a git repository with Bitbucket remote or configure default_workspace")
	}
	if rc.Repository == "" {
		return fmt.Errorf("repository not specified. Either run from a git repository or use --repo flag")
	}
	return nil
}