package pr

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/git"
)

type CheckoutCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Detach     bool   `help:"Checkout in detached HEAD mode"`
	Force      bool   `short:"f" help:"Force checkout, discarding local changes"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string
	Repository string
}

func (c *CheckoutCmd) Run(ctx context.Context) error {
	prCtx, err := NewPRContext(ctx, c.Output, c.NoColor)
	if err != nil {
		return err
	}

	if c.Workspace != "" {
		prCtx.Workspace = c.Workspace
	}
	if c.Repository != "" {
		prCtx.Repository = c.Repository
	}

	if err := prCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	prID, err := ParsePRID(c.PRID)
	if err != nil {
		return fmt.Errorf("invalid pull request ID: %s", c.PRID)
	}

	gitRepo, err := git.NewRepository("")
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	if pr.Source == nil || pr.Source.Branch == nil {
		return fmt.Errorf("pull request source branch information not available")
	}

	sourceBranch := pr.Source.Branch.Name
	sourceRepo := pr.Source.Repository
	
	if sourceRepo == nil {
		return fmt.Errorf("pull request source repository information not available")
	}

	isFork := sourceRepo.FullName != fmt.Sprintf("%s/%s", prCtx.Workspace, prCtx.Repository)
	
	var remoteName string
	var remoteURL string
	
	if isFork {
		remoteName = fmt.Sprintf("pr-%d", prID)
		
		parts := strings.Split(sourceRepo.FullName, "/")
		if len(parts) == 2 {
			remoteURL = fmt.Sprintf("https://bitbucket.org/%s/%s.git", parts[0], parts[1])
		} else {
			return fmt.Errorf("unable to determine fork repository URL from: %s", sourceRepo.FullName)
		}
	} else {
		remoteName = "origin"
	}

	if !c.Force {
		hasChanges, err := gitRepo.HasUncommittedChanges()
		if err != nil {
			return fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}
		
		if hasChanges {
			return fmt.Errorf("you have uncommitted changes. Use --force to discard them or commit/stash your changes")
		}
	}

	if isFork {
		if !gitRepo.RemoteExists(remoteName) {
			fmt.Printf("Adding remote for fork: %s -> %s\n", remoteName, remoteURL)
			if err := gitRepo.AddRemote(remoteName, remoteURL); err != nil {
				return fmt.Errorf("failed to add remote for fork: %w", err)
			}
		}
	}

	fmt.Printf("Fetching PR branch: %s/%s\n", remoteName, sourceBranch)
	if err := gitRepo.FetchBranch(remoteName, sourceBranch); err != nil {
		return fmt.Errorf("failed to fetch PR branch: %w", err)
	}

	localBranch := sourceBranch
	
	if gitRepo.BranchExists(localBranch) && !isSameBranch(gitRepo, localBranch, remoteName, sourceBranch) {
		localBranch = fmt.Sprintf("pr-%d", prID)
		if gitRepo.BranchExists(localBranch) {
			fmt.Printf("Updating existing branch: %s\n", localBranch)
		} else {
			fmt.Printf("Creating local branch: %s\n", localBranch)
		}
	}

	if !gitRepo.BranchExists(localBranch) {
		if err := gitRepo.CreateTrackingBranch(localBranch, remoteName, sourceBranch); err != nil {
			return fmt.Errorf("failed to create tracking branch: %w", err)
		}
	}

	fmt.Printf("Switching to branch: %s\n", localBranch)
	if c.Force {
		if err := gitRepo.ForceCheckoutBranch(localBranch, c.Detach); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	} else {
		if err := gitRepo.CheckoutBranch(localBranch, c.Detach); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	if c.Detach {
		fmt.Printf("Checked out PR #%d in detached HEAD mode\n", prID)
	} else {
		fmt.Printf("Checked out PR #%d to branch '%s'\n", prID, localBranch)
		if isFork {
			fmt.Printf("Tracking %s/%s from fork\n", remoteName, sourceBranch)
		} else {
			fmt.Printf("Tracking %s/%s\n", remoteName, sourceBranch)
		}
	}

	return nil
}


func isSameBranch(repo *git.Repository, localBranch, remoteName, remoteBranch string) bool {
	if currentBranch, err := repo.GetCurrentBranch(); err == nil {
		if currentBranch.ShortName == localBranch && 
		   currentBranch.Remote == remoteName && 
		   currentBranch.RemoteBranch == remoteBranch {
			return true
		}
	}
	
	return false
}
