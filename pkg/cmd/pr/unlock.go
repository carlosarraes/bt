package pr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type UnlockCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *UnlockCmd) Run(ctx context.Context) error {
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

	prID, err := cmd.parsePRID()
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

	if !cmd.Force {
		if err := cmd.confirmUnlock(pr); err != nil {
			return err
		}
	}

	unlockedPR, err := prCtx.Client.PullRequests.UnlockPullRequestConversation(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatOutput(prCtx, unlockedPR)
}

func (cmd *UnlockCmd) parsePRID() (int, error) {
	if cmd.PRID == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

	prIDStr := strings.TrimPrefix(cmd.PRID, "#")

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s': must be a positive integer", cmd.PRID)
	}

	if prID <= 0 {
		return 0, fmt.Errorf("pull request ID must be positive, got %d", prID)
	}

	return prID, nil
}

func (cmd *UnlockCmd) validatePRState(pr *api.PullRequest) error {
	switch pr.State {
	case "OPEN", "MERGED", "DECLINED", "SUPERSEDED":
		return nil
	default:
		return fmt.Errorf("pull request #%d is in an unknown state '%s'", pr.ID, pr.State)
	}
}

func (cmd *UnlockCmd) confirmUnlock(pr *api.PullRequest) error {
	fmt.Printf("Are you sure you want to unlock conversation for pull request #%d (%s)? [y/N] ", pr.ID, pr.Title)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("operation cancelled")
	}
	
	return nil
}

func (cmd *UnlockCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(pr)
	case "json":
		return prCtx.Formatter.Format(pr)
	case "yaml":
		return prCtx.Formatter.Format(pr)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *UnlockCmd) formatTable(pr *api.PullRequest) error {
	fmt.Printf("✓ Unlocked conversation for pull request #%d\n", pr.ID)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("State: %s\n", pr.State)
	
	if pr.Author != nil {
		authorName := pr.Author.DisplayName
		if authorName == "" {
			authorName = pr.Author.Username
		}
		fmt.Printf("Author: %s\n", authorName)
	}

	if pr.Source != nil && pr.Destination != nil {
		sourceBranch := "unknown"
		destBranch := "unknown"
		
		if pr.Source.Branch != nil {
			sourceBranch = pr.Source.Branch.Name
		}
		if pr.Destination.Branch != nil {
			destBranch = pr.Destination.Branch.Name
		}
		
		fmt.Printf("Branches: %s → %s\n", sourceBranch, destBranch)
	}

	fmt.Printf("🔓 Comments on this pull request have been re-enabled\n")

	return nil
}

func (cmd *UnlockCmd) ParsePRID() (int, error) {
	return ParsePRID(cmd.PRID)
}
