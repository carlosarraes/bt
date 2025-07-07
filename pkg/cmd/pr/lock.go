package pr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
)

type LockCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Reason     string `help:"Reason for locking conversation (off_topic, resolved, spam, too_heated)"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *LockCmd) Run(ctx context.Context) error {
	prCtx, err := NewPRContext(ctx, cmd.Output, cmd.NoColor)
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
		if err := cmd.confirmLock(pr); err != nil {
			return err
		}
	}

	lockedPR, err := prCtx.Client.PullRequests.LockPullRequestConversation(ctx, prCtx.Workspace, prCtx.Repository, prID, cmd.Reason)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatOutput(prCtx, lockedPR)
}

func (cmd *LockCmd) parsePRID() (int, error) {
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

func (cmd *LockCmd) validatePRState(pr *api.PullRequest) error {
	switch pr.State {
	case "OPEN", "MERGED", "DECLINED", "SUPERSEDED":
		return nil
	default:
		return fmt.Errorf("pull request #%d is in an unknown state '%s'", pr.ID, pr.State)
	}
}

func (cmd *LockCmd) confirmLock(pr *api.PullRequest) error {
	reasonMsg := ""
	if cmd.Reason != "" {
		reasonMsg = fmt.Sprintf(" (reason: %s)", cmd.Reason)
	}
	
	fmt.Printf("Are you sure you want to lock conversation for pull request #%d (%s)%s? [y/N] ", pr.ID, pr.Title, reasonMsg)
	
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

func (cmd *LockCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest) error {
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

func (cmd *LockCmd) formatTable(pr *api.PullRequest) error {
	fmt.Printf("✓ Locked conversation for pull request #%d\n", pr.ID)
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

	if cmd.Reason != "" {
		fmt.Printf("Lock reason: %s\n", cmd.Reason)
	}
	
	fmt.Printf("🔒 Further comments on this pull request have been disabled\n")

	return nil
}

func (cmd *LockCmd) ParsePRID() (int, error) {
	return ParsePRID(cmd.PRID)
}
