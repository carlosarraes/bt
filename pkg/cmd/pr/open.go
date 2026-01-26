package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type OpenCmd struct {
	PRIDs      []string `arg:"" help:"Pull request IDs (numbers)"`
	Show       bool     `help:"Print URLs instead of opening in browser"`
	Workspace  string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string   `help:"Repository name (defaults to git remote)"`
	Debug      bool     `help:"Show debug output"`
	NoColor    bool
}

func (cmd *OpenCmd) Run(ctx context.Context) error {
	if len(cmd.PRIDs) == 0 {
		return fmt.Errorf("at least one pull request ID is required")
	}

	var wg sync.WaitGroup
	for _, prid := range cmd.PRIDs {
		wg.Add(1)
		go func(prid string) {
			defer wg.Done()
			if err := cmd.processPR(ctx, prid); err != nil {
				if cmd.Debug {
					fmt.Fprintf(os.Stderr, "DEBUG: Error processing PR %s: %v\n", prid, err)
				}
				fmt.Fprintf(os.Stderr, "Error processing PR #%s: %v\n", prid, err)
			}
		}(prid)
	}

	wg.Wait()

	return nil
}

func (cmd *OpenCmd) processPR(ctx context.Context, prid string) error {
	prID, err := strconv.Atoi(strings.TrimPrefix(prid, "#"))
	if err != nil {
		return fmt.Errorf("invalid PR ID '%s': %w", prid, err)
	}

	if cmd.Workspace != "" && cmd.Repository != "" {
		url := fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", cmd.Workspace, cmd.Repository, prID)
		return cmd.handleURL(url)
	}

	prCtx, err := shared.NewCommandContext(ctx, "table", cmd.NoColor, cmd.Debug)
	if err != nil {
		prCtx, err = cmd.createMinimalContext(ctx, "table", cmd.NoColor)
		if err != nil {
			return fmt.Errorf("failed to create PR context: %w", err)
		}
	}

	if prCtx.Workspace == "" {
		return fmt.Errorf("could not determine workspace. Use --workspace flag")
	}

	url, err := cmd.findPRURL(ctx, prCtx, prID)
	if err != nil {
		return err
	}

	return cmd.handleURL(url)
}

type PRMatch struct {
	URL        string
	Repository string
	FullName   string
}

func (cmd *OpenCmd) findPRURL(ctx context.Context, prCtx *PRContext, prID int) (string, error) {
	repoOptions := &api.RepositoryListOptions{
		Role:    "member",
		PageLen: 100,
		Page:    1,
	}

	repoResult, err := prCtx.Client.Repositories.ListRepositories(ctx, prCtx.Workspace, repoOptions)
	if err != nil {
		return "", fmt.Errorf("failed to fetch repositories: %w", err)
	}

	var repositories []*api.Repository
	if repoResult.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(repoResult.Values, &values); err != nil {
			return "", fmt.Errorf("failed to unmarshal repository values: %w", err)
		}

		repositories = make([]*api.Repository, len(values))
		for i, rawRepo := range values {
			var repo api.Repository
			if err := json.Unmarshal(rawRepo, &repo); err != nil {
				return "", fmt.Errorf("failed to unmarshal repository %d: %w", i, err)
			}
			repositories[i] = &repo
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var matches []PRMatch

	for _, repo := range repositories {
		wg.Add(1)
		go func(repo *api.Repository) {
			defer wg.Done()

			repoSlug := repo.Name
			if repo.FullName != "" {
				parts := strings.Split(repo.FullName, "/")
				if len(parts) == 2 {
					repoSlug = parts[1]
				}
			}

			pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, repoSlug, prID)
			if err == nil {
				if pr.State != "OPEN" {
					if cmd.Debug {
						fmt.Fprintf(os.Stderr, "DEBUG: PR #%d in repository %s is %s, skipping\n", prID, repo.FullName, pr.State)
					}
					return
				}
				currentUser, userErr := prCtx.Client.GetAuthManager().GetAuthenticatedUser(ctx)
				if userErr != nil {
					if cmd.Debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Could not get current user: %v\n", userErr)
					}
					return
				}

				if cmd.Debug {
					fmt.Fprintf(os.Stderr, "DEBUG: Current user username: '%s', account_id: '%s'\n", currentUser.Username, currentUser.AccountID)
					if pr.Author != nil {
						fmt.Fprintf(os.Stderr, "DEBUG: PR author username: '%s', display_name: '%s', account_id: '%s'\n",
							pr.Author.Username, pr.Author.DisplayName, pr.Author.AccountID)
					} else {
						fmt.Fprintf(os.Stderr, "DEBUG: PR #%d in repository %s has no author field\n", prID, repo.FullName)
					}
				}

				if pr.Author != nil && pr.Author.AccountID == currentUser.AccountID {
					url := fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", prCtx.Workspace, repoSlug, prID)
					if cmd.Debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Found user's PR #%d in repository %s\n", prID, repo.FullName)
					}

					mu.Lock()
					matches = append(matches, PRMatch{
						URL:        url,
						Repository: repoSlug,
						FullName:   repo.FullName,
					})
					mu.Unlock()
				} else {
					if cmd.Debug {
						authorAccountID := "unknown"
						if pr.Author != nil {
							authorAccountID = pr.Author.AccountID
						}
						fmt.Fprintf(os.Stderr, "DEBUG: PR #%d in repository %s belongs to account '%s', not '%s'\n", prID, repo.FullName, authorAccountID, currentUser.AccountID)
					}
				}
				return
			}

			if cmd.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: PR #%d not found in repository %s\n", prID, repo.FullName)
			}
		}(repo)
	}

	wg.Wait()

	if len(matches) == 0 {
		return "", fmt.Errorf("PR #%d not found in any repository in workspace %s", prID, prCtx.Workspace)
	}

	if len(matches) == 1 {
		return matches[0].URL, nil
	}

	fmt.Fprintf(os.Stderr, "Multiple PRs found with ID #%d:\n", prID)
	for i, match := range matches {
		fmt.Fprintf(os.Stderr, "[%d] %s (%s)\n", i+1, match.FullName, match.URL)
	}
	fmt.Fprintf(os.Stderr, "\nPlease be more specific by using: bt pr open --workspace %s --repository <repo_name> %d\n", prCtx.Workspace, prID)

	return "", fmt.Errorf("multiple PRs found - please specify repository")
}

func (cmd *OpenCmd) handleURL(url string) error {
	if cmd.Show {
		fmt.Println(url)
		return nil
	}

	return cmd.openInBrowser(url)
}

func (cmd *OpenCmd) openInBrowser(url string) error {
	var cmdName string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmdName = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmdName = "open"
		args = []string{url}
	case "linux":
		cmdName = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	execCmd := exec.Command(cmdName, args...)
	return execCmd.Start()
}

func (cmd *OpenCmd) createMinimalContext(ctx context.Context, outputFormat string, noColor bool) (*PRContext, error) {
	return shared.NewMinimalContext(ctx, shared.MinimalContextOptions{
		OutputFormat: outputFormat,
		Workspace:    cmd.Workspace,
		NoColor:      noColor,
		Debug:        cmd.Debug,
	})
}
