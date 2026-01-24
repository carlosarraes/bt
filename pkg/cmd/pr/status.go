package pr

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/git"
)

type StatusCmd struct {
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

type PRStatusResult struct {
	CreatedByYou      []*api.PullRequest `json:"created_by_you"`
	NeedingReview     []*api.PullRequest `json:"needing_review"`
	CurrentBranch     *api.PullRequest   `json:"current_branch,omitempty"`
	CurrentBranchName string             `json:"current_branch_name,omitempty"`
}

func (cmd *StatusCmd) Run(ctx context.Context) error {
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

	user, err := prCtx.Client.GetAuthManager().GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		currentBranch = ""
	}

	result := &PRStatusResult{
		CurrentBranchName: currentBranch,
	}

	createdByYou, err := cmd.getPRsCreatedByUser(ctx, prCtx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to get PRs created by you: %w", err)
	}
	result.CreatedByYou = createdByYou

	needingReview, err := cmd.getPRsNeedingReview(ctx, prCtx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to get PRs needing review: %w", err)
	}
	result.NeedingReview = needingReview

	if currentBranch != "" {
		currentBranchPR, err := cmd.findPRForBranch(ctx, prCtx, currentBranch)
		if err == nil && currentBranchPR != nil {
			result.CurrentBranch = currentBranchPR
		}
	}

	return cmd.formatOutput(prCtx, result)
}

func (cmd *StatusCmd) getPRsCreatedByUser(ctx context.Context, prCtx *PRContext, username string) ([]*api.PullRequest, error) {
	options := &api.PullRequestListOptions{
		State:   "OPEN",
		Author:  username,
		Sort:    "-updated_on",
		PageLen: 50,
		Page:    1,
	}

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		return nil, handlePullRequestAPIError(err)
	}

	return parsePullRequestResults(result)
}

func (cmd *StatusCmd) getPRsNeedingReview(ctx context.Context, prCtx *PRContext, username string) ([]*api.PullRequest, error) {
	options := &api.PullRequestListOptions{
		State:    "OPEN",
		Reviewer: username,
		Sort:     "-updated_on",
		PageLen:  50,
		Page:     1,
	}

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		return nil, handlePullRequestAPIError(err)
	}

	return parsePullRequestResults(result)
}

func (cmd *StatusCmd) findPRForBranch(ctx context.Context, prCtx *PRContext, branchName string) (*api.PullRequest, error) {
	options := &api.PullRequestListOptions{
		State:   "OPEN",
		Sort:    "-updated_on",
		PageLen: 50,
		Page:    1,
	}

	result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
	if err != nil {
		return nil, handlePullRequestAPIError(err)
	}

	pullRequests, err := parsePullRequestResults(result)
	if err != nil {
		return nil, err
	}

	for _, pr := range pullRequests {
		if pr.Source != nil && pr.Source.Branch != nil && pr.Source.Branch.Name == branchName {
			return pr, nil
		}
	}

	return nil, nil
}

func getCurrentBranch() (string, error) {
	gitRepo, err := git.NewRepository("")
	if err != nil {
		return "", err
	}

	ctx, err := gitRepo.GetContext()
	if err != nil {
		return "", err
	}

	return ctx.Branch, nil
}

func (cmd *StatusCmd) formatOutput(prCtx *PRContext, result *PRStatusResult) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, result)
	case "json":
		return prCtx.Formatter.Format(result)
	case "yaml":
		return prCtx.Formatter.Format(result)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *StatusCmd) formatTable(prCtx *PRContext, result *PRStatusResult) error {
	hasAny := len(result.CreatedByYou) > 0 || len(result.NeedingReview) > 0 || result.CurrentBranch != nil

	if !hasAny {
		fmt.Println("No relevant pull requests found")
		return nil
	}

	if result.CurrentBranch != nil {
		fmt.Printf("Current branch: %s\n", result.CurrentBranchName)
		cmd.printPRInfo(result.CurrentBranch, "  ")
		fmt.Println()
	}

	if len(result.CreatedByYou) > 0 {
		fmt.Printf("Created by you\n")
		for _, pr := range result.CreatedByYou {
			cmd.printPRInfo(pr, "  ")
		}
		fmt.Println()
	}

	if len(result.NeedingReview) > 0 {
		fmt.Printf("Requesting a code review from you\n")
		for _, pr := range result.NeedingReview {
			cmd.printPRInfo(pr, "  ")
		}
		fmt.Println()
	}

	return nil
}

func (cmd *StatusCmd) printPRInfo(pr *api.PullRequest, indent string) {
	statusIcon := cmd.getStatusIcon(pr)
	
	branchInfo := ""
	if pr.Source != nil && pr.Source.Branch != nil {
		branchInfo = pr.Source.Branch.Name
		if pr.Destination != nil && pr.Destination.Branch != nil {
			branchInfo = fmt.Sprintf("%s → %s", branchInfo, pr.Destination.Branch.Name)
		}
	}

	fmt.Printf("%s%s #%d %s", indent, statusIcon, pr.ID, pr.Title)
	if branchInfo != "" {
		fmt.Printf(" (%s)", branchInfo)
	}
	fmt.Println()
}

func (cmd *StatusCmd) getStatusIcon(pr *api.PullRequest) string {
	switch pr.State {
	case "OPEN":
		return "✓"
	case "MERGED":
		return "✓"
	case "DECLINED":
		return "✗"
	default:
		return "⏳"
	}
}

func getStatusSymbol(state string) string {
	switch state {
	case "OPEN":
		return "✓"
	case "MERGED":
		return "✓"
	case "DECLINED":
		return "✗"
	default:
		return "⏳"
	}
}
