package pr

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/git"
)

type UpdateBranchCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Force      bool   `short:"f" help:"Force update, overriding safety checks"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

type UpdateBranchResult struct {
	PRID          int                 `json:"pr_id"`
	Title         string              `json:"title"`
	SourceBranch  string              `json:"source_branch"`
	TargetBranch  string              `json:"target_branch"`
	Success       bool                `json:"success"`
	Message       string              `json:"message"`
	FilesUpdated  []string            `json:"files_updated,omitempty"`
	HasConflicts  bool                `json:"has_conflicts"`
	ConflictFiles []string            `json:"conflict_files,omitempty"`
	MergeResult   *git.MergeResult    `json:"merge_result,omitempty"`
}

func (cmd *UpdateBranchCmd) Run(ctx context.Context) error {
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

	prID, err := ParsePRID(cmd.PRID)
	if err != nil {
		return err
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if err := cmd.validatePRState(pr); err != nil {
		return err
	}

	sourceBranch, targetBranch, err := cmd.extractBranchNames(pr)
	if err != nil {
		return err
	}

	gitRepo, err := git.NewRepository("")
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	if err := cmd.validateGitRepository(gitRepo, prCtx); err != nil {
		return err
	}

	result, err := cmd.updateBranch(ctx, gitRepo, prID, pr.Title, sourceBranch, targetBranch)
	if err != nil {
		return err
	}

	return cmd.formatOutput(prCtx, result)
}

func (cmd *UpdateBranchCmd) validatePRState(pr *api.PullRequest) error {
	if pr.State != string(api.PullRequestStateOpen) {
		return fmt.Errorf("pull request #%d is %s and cannot be updated", pr.ID, pr.State)
	}
	
	return nil
}

func (cmd *UpdateBranchCmd) extractBranchNames(pr *api.PullRequest) (string, string, error) {
	if pr.Source == nil || pr.Source.Branch == nil {
		return "", "", fmt.Errorf("pull request source branch information is missing")
	}
	
	if pr.Destination == nil || pr.Destination.Branch == nil {
		return "", "", fmt.Errorf("pull request destination branch information is missing")
	}
	
	sourceBranch := pr.Source.Branch.Name
	targetBranch := pr.Destination.Branch.Name
	
	if sourceBranch == "" {
		return "", "", fmt.Errorf("pull request source branch name is empty")
	}
	
	if targetBranch == "" {
		return "", "", fmt.Errorf("pull request destination branch name is empty")
	}
	
	return sourceBranch, targetBranch, nil
}

func (cmd *UpdateBranchCmd) validateGitRepository(gitRepo *git.Repository, prCtx *PRContext) error {
	gitWorkspace := gitRepo.GetWorkspace()
	gitRepoName := gitRepo.GetName()

	if gitWorkspace != prCtx.Workspace {
		return fmt.Errorf("git repository workspace '%s' doesn't match PR workspace '%s'", gitWorkspace, prCtx.Workspace)
	}

	if gitRepoName != prCtx.Repository {
		return fmt.Errorf("git repository name '%s' doesn't match PR repository '%s'", gitRepoName, prCtx.Repository)
	}

	return nil
}

func (cmd *UpdateBranchCmd) updateBranch(ctx context.Context, gitRepo *git.Repository, prID int, title, sourceBranch, targetBranch string) (*UpdateBranchResult, error) {
	result := &UpdateBranchResult{
		PRID:         prID,
		Title:        title,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
	}

	if !gitRepo.BranchExists(sourceBranch) {
		return nil, fmt.Errorf("source branch '%s' does not exist locally", sourceBranch)
	}

	if !gitRepo.BranchExists(targetBranch) {
		return nil, fmt.Errorf("target branch '%s' does not exist locally", targetBranch)
	}

	currentBranch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	if err := gitRepo.FetchRemote("origin"); err != nil {
		return nil, fmt.Errorf("failed to fetch from remote: %w", err)
	}

	targetStatus, err := gitRepo.GetBranchStatus(targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get target branch status: %w", err)
	}

	if targetStatus.UpToDate {
		result.Success = true
		result.Message = fmt.Sprintf("PR #%d branch '%s' is already up to date with '%s'", prID, sourceBranch, targetBranch)
		return result, nil
	}

	if targetStatus.Behind > 0 {
		_, err := gitRepo.PullBranch(targetBranch, cmd.Force)
		if err != nil {
			return nil, fmt.Errorf("failed to update target branch '%s' from remote: %w", targetBranch, err)
		}
	}

	if currentBranch.ShortName != sourceBranch {
		if err := gitRepo.CheckoutBranch(sourceBranch, false); err != nil {
			return nil, fmt.Errorf("failed to checkout source branch '%s': %w", sourceBranch, err)
		}
	}

	mergeResult, err := gitRepo.MergeBranch(targetBranch, sourceBranch, cmd.Force)
	if err != nil {
		return nil, fmt.Errorf("failed to merge '%s' into '%s': %w", targetBranch, sourceBranch, err)
	}

	result.MergeResult = mergeResult
	result.Success = mergeResult.Success
	result.HasConflicts = mergeResult.HasConflicts
	result.ConflictFiles = mergeResult.ConflictFiles

	if mergeResult.Success {
		result.Message = fmt.Sprintf("Successfully updated PR #%d branch '%s' from '%s'", prID, sourceBranch, targetBranch)
		if mergeResult.CommitHash != "" {
			result.Message += fmt.Sprintf(" (commit: %s)", mergeResult.CommitHash[:8])
		}
	} else {
		result.Message = fmt.Sprintf("Failed to update PR #%d branch '%s' from '%s': %s", prID, sourceBranch, targetBranch, mergeResult.Message)
	}

	if currentBranch.ShortName != sourceBranch && currentBranch.ShortName != "" {
		if restoreErr := gitRepo.CheckoutBranch(currentBranch.ShortName, false); restoreErr != nil {
			result.Message += fmt.Sprintf(" (Warning: failed to restore original branch '%s')", currentBranch.ShortName)
		}
	}

	return result, nil
}

func (cmd *UpdateBranchCmd) formatOutput(prCtx *PRContext, result *UpdateBranchResult) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(result)
	case "json":
		return prCtx.Formatter.Format(result)
	case "yaml":
		return prCtx.Formatter.Format(result)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *UpdateBranchCmd) formatTable(result *UpdateBranchResult) error {
	fmt.Printf("PR #%d: %s\n", result.PRID, result.Title)
	fmt.Printf("Branch update: %s ← %s\n", result.SourceBranch, result.TargetBranch)
	
	if result.Success {
		fmt.Printf("Status: ✅ Success\n")
	} else {
		fmt.Printf("Status: ❌ Failed\n")
	}

	fmt.Printf("Result: %s\n", result.Message)

	if result.HasConflicts {
		fmt.Printf("\n⚠️  Merge Conflicts Detected:\n")
		for _, file := range result.ConflictFiles {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Printf("\nResolve conflicts manually and commit your changes.\n")
	}

	if len(result.FilesUpdated) > 0 {
		fmt.Printf("\nFiles updated:\n")
		for _, file := range result.FilesUpdated {
			fmt.Printf("  • %s\n", file)
		}
	}

	if result.MergeResult != nil && result.MergeResult.CommitHash != "" {
		fmt.Printf("\nCommit: %s\n", result.MergeResult.CommitHash)
	}

	return nil
}
