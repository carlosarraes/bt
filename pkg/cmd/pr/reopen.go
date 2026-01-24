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

type ReopenCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Comment    string `short:"c" help:"Comment to add when reopening the PR"`
	Force      bool   `short:"f" help:"Skip confirmation prompt"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *ReopenCmd) Run(ctx context.Context) error {
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
		if err := cmd.confirmReopen(pr); err != nil {
			return err
		}
	}

	reopenedPR, err := prCtx.Client.PullRequests.ReopenPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID, cmd.Comment)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatOutput(prCtx, reopenedPR)
}

func (cmd *ReopenCmd) parsePRID() (int, error) {
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

func (cmd *ReopenCmd) validatePRState(pr *api.PullRequest) error {
	switch pr.State {
	case "DECLINED":
		return nil
	case "OPEN":
		return fmt.Errorf("pull request #%d is already open", pr.ID)
	case "MERGED":
		return fmt.Errorf("pull request #%d is already merged and cannot be reopened", pr.ID)
	case "SUPERSEDED":
		return fmt.Errorf("pull request #%d is superseded and cannot be reopened", pr.ID)
	default:
		return fmt.Errorf("pull request #%d is in an unknown state '%s'", pr.ID, pr.State)
	}
}

func (cmd *ReopenCmd) confirmReopen(pr *api.PullRequest) error {
	fmt.Printf("Are you sure you want to reopen pull request #%d (%s)? [y/N] ", pr.ID, pr.Title)

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

func (cmd *ReopenCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest) error {
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

func (cmd *ReopenCmd) formatTable(pr *api.PullRequest) error {
	fmt.Printf("✓ Reopened pull request #%d\n", pr.ID)
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

	return nil
}
