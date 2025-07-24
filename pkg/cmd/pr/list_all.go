package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/config"
)

type ListAllCmd struct {
	State     string `help:"Filter by state (open, merged, declined, all)" default:"open"`
	Limit     int    `help:"Maximum number of pull requests per repository" default:"10"`
	Sort      string `help:"Sort by field (created, updated, priority)" default:"updated"`
	Output    string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	URL       bool   `help:"Output URLs in format: <repo:source-branch> <target-branch> <url>"`
	Debug     bool   `help:"Show debug output"`
	NoColor   bool
	Workspace string `help:"Bitbucket workspace (defaults to git remote or config)"`
}

type PRWithRepo struct {
	*api.PullRequest
	Repository *api.Repository `json:"repository"`
	Workspace  string          `json:"workspace"`
}

func (cmd *ListAllCmd) Run(ctx context.Context) error {
	prCtx, err := NewPRContext(ctx, cmd.Output, cmd.NoColor, cmd.Debug)
	if err != nil {
		prCtx, err = cmd.createMinimalContext(ctx, cmd.Output, cmd.NoColor)
		if err != nil {
			return err
		}
	}

	workspace := cmd.Workspace
	if workspace == "" {
		workspace = prCtx.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace is required. Provide it via --workspace flag or configure it")
	}

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Using workspace: %s\n", workspace)
	}

	if cmd.State != "" {
		if err := validateState(cmd.State); err != nil {
			return err
		}
	}

	if cmd.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	if cmd.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	var currentUser *auth.User
	if prCtx.Client != nil {
		user, err := prCtx.Client.GetAuthManager().GetAuthenticatedUser(ctx)
		if err != nil {
			if cmd.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: Could not get current user: %v\n", err)
			}
			return fmt.Errorf("could not get current user for filtering: %w", err)
		}
		currentUser = user
		if cmd.Debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Current user: %s\n", currentUser.Username)
		}
	}

	repoOptions := &api.RepositoryListOptions{
		Role:    "member",
		PageLen: 100,
		Page:    1,
	}

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Fetching repositories in workspace %s\n", workspace)
	}

	repoResult, err := prCtx.Client.Repositories.ListRepositories(ctx, workspace, repoOptions)
	if err != nil {
		return fmt.Errorf("failed to fetch repositories: %w", err)
	}

	var repositories []*api.Repository
	if repoResult.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(repoResult.Values, &values); err != nil {
			return fmt.Errorf("failed to unmarshal repository values: %w", err)
		}

		repositories = make([]*api.Repository, len(values))
		for i, rawRepo := range values {
			var repo api.Repository
			if err := json.Unmarshal(rawRepo, &repo); err != nil {
				return fmt.Errorf("failed to unmarshal repository %d: %w", i, err)
			}
			repositories[i] = &repo
		}
	}

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Found %d repositories\n", len(repositories))
	}

	var allPRs []*PRWithRepo
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(repositories))

	for _, repo := range repositories {
		wg.Add(1)
		go func(repo *api.Repository) {
			defer wg.Done()

			options := &api.PullRequestListOptions{
				PageLen: cmd.Limit,
				Page:    1,
				Sort:    "-updated_on",
				Author:  currentUser.Username,
			}

			if cmd.State != "" && cmd.State != "all" {
				options.State = strings.ToUpper(cmd.State)
			}

			if cmd.Sort != "" {
				switch strings.ToLower(cmd.Sort) {
				case "created":
					options.Sort = "-created_on"
				case "updated":
					options.Sort = "-updated_on"
				case "priority":
					options.Sort = "-priority"
				}
			}

			if cmd.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: Fetching PRs from repository %s\n", repo.FullName)
			}

			repoSlug := repo.Name
			if repo.FullName != "" {
				parts := strings.Split(repo.FullName, "/")
				if len(parts) == 2 {
					repoSlug = parts[1]
				}
			}
			
			result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, workspace, repoSlug, options)
			if err != nil {
				if cmd.Debug {
					fmt.Fprintf(os.Stderr, "DEBUG: Error fetching PRs from %s: %v\n", repo.FullName, err)
				}
				errChan <- fmt.Errorf("failed to fetch PRs from %s: %w", repo.FullName, err)
				return
			}

			pullRequests, err := parsePullRequestResults(result)
			if err != nil {
				errChan <- fmt.Errorf("failed to parse PR results from %s: %w", repo.FullName, err)
				return
			}

			mu.Lock()
			for _, pr := range pullRequests {
				allPRs = append(allPRs, &PRWithRepo{
					PullRequest: pr,
					Repository:  repo,
					Workspace:   workspace,
				})
			}
			mu.Unlock()

			if cmd.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: Found %d PRs in repository %s\n", len(pullRequests), repo.FullName)
			}
		}(repo)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 && cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: %d repositories had errors:\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "DEBUG: - %v\n", err)
		}
	}

	sort.Slice(allPRs, func(i, j int) bool {
		prI := allPRs[i].PullRequest
		prJ := allPRs[j].PullRequest
		
		targetI := ""
		if prI.Destination != nil && prI.Destination.Branch != nil {
			targetI = prI.Destination.Branch.Name
		}
		targetJ := ""
		if prJ.Destination != nil && prJ.Destination.Branch != nil {
			targetJ = prJ.Destination.Branch.Name
		}
		
		isHomologI := strings.Contains(strings.ToLower(targetI), "homolog")
		isHomologJ := strings.Contains(strings.ToLower(targetJ), "homolog")
		
		if isHomologI && !isHomologJ {
			return true
		}
		if !isHomologI && isHomologJ {
			return false
		}
		
		if prI.UpdatedOn == nil && prJ.UpdatedOn == nil {
			return false
		}
		if prI.UpdatedOn == nil {
			return false
		}
		if prJ.UpdatedOn == nil {
			return true
		}
		
		return prI.UpdatedOn.After(*prJ.UpdatedOn)
	})

	if cmd.Debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Total PRs found across all repositories: %d\n", len(allPRs))
	}

	return cmd.formatOutput(prCtx, allPRs)
}

func (cmd *ListAllCmd) formatOutput(prCtx *PRContext, prs []*PRWithRepo) error {
	if cmd.URL {
		return cmd.formatURL(prCtx, prs)
	}
	
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, prs)
	case "json":
		return cmd.formatJSON(prCtx, prs)
	case "yaml":
		return cmd.formatYAML(prCtx, prs)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ListAllCmd) formatTable(prCtx *PRContext, prs []*PRWithRepo) error {
	if len(prs) == 0 {
		fmt.Println("No pull requests found across all repositories")
		return nil
	}

	headers := []string{"Repository", "ID", "Title", "Source", "Target", "State", "Updated"}
	rows := make([][]string, len(prs))
	
	for i, prWithRepo := range prs {
		pr := prWithRepo.PullRequest
		repo := prWithRepo.Repository
		
		repoName := repo.Name
		if len(repoName) > 15 {
			repoName = repoName[:12] + "..."
		}

		title := pr.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		sourceBranch := "-"
		if pr.Source != nil && pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
			if len(sourceBranch) > 15 {
				sourceBranch = sourceBranch[:12] + "..."
			}
		}

		targetBranch := "-"
		if pr.Destination != nil && pr.Destination.Branch != nil {
			targetBranch = pr.Destination.Branch.Name
			if len(targetBranch) > 15 {
				targetBranch = targetBranch[:12] + "..."
			}
		}

		state := pr.State
		if state == "" {
			state = "UNKNOWN"
		}

		updatedTime := FormatRelativeTime(pr.UpdatedOn)

		rows[i] = []string{
			repoName,
			fmt.Sprintf("#%d", pr.ID),
			title,
			sourceBranch,
			targetBranch,
			state,
			updatedTime,
		}
	}

	return renderCustomTable(headers, rows)
}

func (cmd *ListAllCmd) formatJSON(prCtx *PRContext, prs []*PRWithRepo) error {
	output := map[string]interface{}{
		"total_count":     len(prs),
		"pull_requests":   prs,
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ListAllCmd) formatYAML(prCtx *PRContext, prs []*PRWithRepo) error {
	output := map[string]interface{}{
		"total_count":     len(prs),
		"pull_requests":   prs,
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ListAllCmd) formatURL(prCtx *PRContext, prs []*PRWithRepo) error {
	if len(prs) == 0 {
		return nil
	}

	for _, prWithRepo := range prs {
		pr := prWithRepo.PullRequest
		repo := prWithRepo.Repository

		repoName := repo.Name
		if repo.FullName != "" {
			parts := strings.Split(repo.FullName, "/")
			if len(parts) == 2 {
				repoName = parts[1]
			}
		}

		sourceBranch := "-"
		if pr.Source != nil && pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
		}

		targetBranch := "-"
		if pr.Destination != nil && pr.Destination.Branch != nil {
			targetBranch = pr.Destination.Branch.Name
		}

		url := fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", prWithRepo.Workspace, repoName, pr.ID)

		fmt.Printf("%s:%s %s %s\n", repoName, sourceBranch, targetBranch, url)
	}

	return nil
}

func (cmd *ListAllCmd) createMinimalContext(ctx context.Context, outputFormat string, noColor bool) (*PRContext, error) {
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	authManager, err := createAuthManager()
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

	return &PRContext{
		Client:    client,
		Config:    cfg,
		Workspace: cmd.Workspace,
		Debug:     cmd.Debug,
	}, nil
}
