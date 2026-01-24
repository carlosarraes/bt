package pr

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type MergeCmd struct {
	PRID         string `arg:"" help:"Pull request ID (number)"`
	Squash       bool   `help:"Squash commits when merging"`
	DeleteBranch bool   `help:"Delete source branch after merge"`
	Auto         bool   `help:"Automatically merge when checks pass"`
	Force        bool   `short:"f" help:"Skip confirmation prompt"`
	Message      string `short:"m" help:"Custom merge commit message"`
	Output       string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor      bool
	Workspace    string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository   string `help:"Repository name (defaults to git remote)"`
}

func (cmd *MergeCmd) Run(ctx context.Context) error {
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

	prID, err := cmd.ParsePRID()
	if err != nil {
		return fmt.Errorf("invalid pull request ID '%s': %w", cmd.PRID, err)
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if err := cmd.validateMergeability(pr); err != nil {
		return err
	}

	if !cmd.Force {
		if err := cmd.showConfirmationPrompt(pr); err != nil {
			return err
		}
	}

	mergeRequest := &api.PullRequestMerge{
		Type:              "pullrequest_merge",
		CloseSourceBranch: cmd.DeleteBranch,
	}

	if cmd.Squash {
		mergeRequest.MergeStrategy = "squash"
	}

	if cmd.Message != "" {
		mergeRequest.Message = cmd.Message
	}

	mergedPR, err := prCtx.Client.PullRequests.MergePullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID, mergeRequest)
	if err != nil {
		return handleMergeAPIError(err)
	}

	if cmd.DeleteBranch && !mergedPR.CloseSourceBranch {
		if err := cmd.deleteBranch(ctx, prCtx, pr); err != nil {
			fmt.Printf("Warning: Failed to delete source branch: %v\n", err)
		}
	}

	return cmd.formatOutput(prCtx, mergedPR)
}

func (cmd *MergeCmd) validateMergeability(pr *api.PullRequest) error {
	if pr.State != "OPEN" {
		return fmt.Errorf("pull request #%d is %s and cannot be merged", pr.ID, strings.ToLower(pr.State))
	}

	// TODO: Add more validation checks once we have access to PR status/checks

	return nil
}

func (cmd *MergeCmd) showConfirmationPrompt(pr *api.PullRequest) error {
	fmt.Printf("Are you sure you want to merge pull request #%d?\n", pr.ID)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("Author: %s\n", getUserDisplayName(pr.Author))
	fmt.Printf("Source: %s\n", getBranchName(pr.Source))
	fmt.Printf("Target: %s\n", getBranchName(pr.Destination))

	if cmd.Squash {
		fmt.Printf("Merge strategy: squash\n")
	} else {
		fmt.Printf("Merge strategy: merge commit\n")
	}

	if cmd.DeleteBranch {
		fmt.Printf("Source branch will be deleted after merge\n")
	}

	fmt.Print("\nContinue? (y/N): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("merge cancelled by user")
	}

	return nil
}

func (cmd *MergeCmd) deleteBranch(ctx context.Context, prCtx *PRContext, pr *api.PullRequest) error {
	if pr.Source == nil || pr.Source.Branch == nil {
		return fmt.Errorf("source branch information not available")
	}

	branchName := pr.Source.Branch.Name
	endpoint := fmt.Sprintf("repositories/%s/%s/refs/branches/%s", prCtx.Workspace, prCtx.Repository, branchName)

	resp, err := prCtx.Client.Delete(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", branchName, err)
	}
	defer resp.Body.Close()

	return nil
}

func (cmd *MergeCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, pr)
	case "json":
		return prCtx.Formatter.Format(pr)
	case "yaml":
		return prCtx.Formatter.Format(pr)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *MergeCmd) formatTable(prCtx *PRContext, pr *api.PullRequest) error {
	fmt.Printf("✓ Merged pull request #%d\n", pr.ID)
	fmt.Printf("Title: %s\n", pr.Title)

	if pr.MergeCommit != nil {
		fmt.Printf("Merge commit: %s\n", pr.MergeCommit.Hash)
	}

	if cmd.DeleteBranch {
		fmt.Printf("Source branch deleted: %s\n", getBranchName(pr.Source))
	}

	return nil
}

func handleMergeAPIError(err error) error {
	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		switch bitbucketErr.StatusCode {
		case 404:
			return fmt.Errorf("pull request not found or repository not accessible")
		case 409:
			return fmt.Errorf("pull request cannot be merged (conflicts or checks failed)")
		case 422:
			return fmt.Errorf("pull request is not in a mergeable state")
		default:
			return fmt.Errorf("merge failed: %s", bitbucketErr.Message)
		}
	}
	return fmt.Errorf("failed to merge pull request: %w", err)
}

func getUserDisplayName(user *api.User) string {
	if user == nil {
		return "Unknown"
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.Username
}

func getBranchName(branch *api.PullRequestBranch) string {
	if branch == nil || branch.Branch == nil {
		return "Unknown"
	}
	return branch.Branch.Name
}

func (cmd *MergeCmd) ParsePRID() (int, error) {
	return ParsePRID(cmd.PRID)
}
