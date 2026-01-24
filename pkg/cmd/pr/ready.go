package pr

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type ReadyCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Comment    string `help:"Add a comment when marking as ready"`
	Force      bool   `short:"f" help:"Force mark as ready without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

type PRReadyResult struct {
	PullRequest *api.PullRequest        `json:"pull_request"`
	Comment     *api.PullRequestComment `json:"comment,omitempty"`
	Ready       bool                    `json:"ready"`
}

func (cmd *ReadyCmd) Run(ctx context.Context) error {
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
		return fmt.Errorf("invalid pull request ID: %w", err)
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if !isPRDraft(pr) {
		return fmt.Errorf("pull request #%d is already ready for review (current state: %s)", prID, pr.State)
	}

	if !cmd.Force {
		if !cmd.confirmReady(pr) {
			return fmt.Errorf("aborted")
		}
	}

	updateRequest := &api.UpdatePullRequestRequest{
		Type:  "pullrequest",
		State: "OPEN",
	}

	updatedPR, err := prCtx.Client.PullRequests.UpdatePullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID, updateRequest)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	result := &PRReadyResult{
		PullRequest: updatedPR,
		Ready:       true,
	}

	if cmd.Comment != "" {
		comment, err := prCtx.Client.PullRequests.AddComment(ctx, prCtx.Workspace, prCtx.Repository, prID, cmd.Comment, nil)
		if err != nil {
			fmt.Printf("Warning: Failed to add comment: %v\n", err)
		} else {
			result.Comment = comment
		}
	}

	return cmd.formatOutput(prCtx, result)
}

func isPRDraft(pr *api.PullRequest) bool {
	if strings.EqualFold(pr.State, "DRAFT") {
		return true
	}

	title := strings.ToLower(pr.Title)
	if strings.HasPrefix(title, "draft:") || strings.HasPrefix(title, "[draft]") || strings.HasPrefix(title, "wip:") || strings.HasPrefix(title, "[wip]") {
		return true
	}

	return false
}

func (cmd *ReadyCmd) confirmReady(pr *api.PullRequest) bool {
	fmt.Printf("Mark pull request #%d as ready for review?\n", pr.ID)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("Author: %s\n", pr.Author.Username)
	fmt.Printf("Source: %s -> %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
	fmt.Printf("\nThis will:\n")
	fmt.Printf("- Mark the PR as ready for review\n")
	fmt.Printf("- Notify reviewers\n")
	fmt.Printf("- Allow the PR to be merged\n")

	if cmd.Comment != "" {
		fmt.Printf("- Add comment: %s\n", cmd.Comment)
	}

	fmt.Printf("\nContinue? (y/N): ")
	var response string
	fmt.Scanln(&response)

	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func (cmd *ReadyCmd) formatOutput(prCtx *PRContext, result *PRReadyResult) error {
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

func (cmd *ReadyCmd) formatTable(prCtx *PRContext, result *PRReadyResult) error {
	pr := result.PullRequest

	fmt.Printf("✓ Pull request #%d is now ready for review!\n\n", pr.ID)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("Author: %s\n", pr.Author.Username)
	fmt.Printf("Source: %s -> %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
	fmt.Printf("State: %s\n", pr.State)

	if pr.Links != nil && pr.Links.HTML != nil {
		fmt.Printf("URL: %s\n", pr.Links.HTML.Href)
	}

	if result.Comment != nil {
		fmt.Printf("\n💬 Comment added: %s\n", result.Comment.Content.Raw)
	}

	fmt.Printf("\nReviewers will be notified that this PR is ready for review.\n")

	return nil
}
