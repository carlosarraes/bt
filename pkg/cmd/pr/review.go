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

type ReviewCmd struct {
	PRID           string `arg:"" help:"Pull request ID (number)"`
	Approve        bool   `help:"Approve the pull request"`
	RequestChanges bool   `name:"request-changes" help:"Request changes on the pull request"`
	Comment        bool   `help:"Add a comment to the pull request"`
	Body           string `short:"b" help:"Comment body text"`
	BodyFile       string `short:"F" name:"body-file" help:"Read comment body from file"`
	Force          bool   `short:"f" help:"Skip confirmation prompts"`
	Output         string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor        bool
	Workspace      string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository     string `help:"Repository name (defaults to git remote)"`
}

type reviewAction int

const (
	actionApprove reviewAction = iota
	actionRequestChanges
	actionComment
)

func (cmd *ReviewCmd) Run(ctx context.Context) error {
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
		return err
	}

	action, err := cmd.validateReviewAction()
	if err != nil {
		return err
	}

	body, err := cmd.getCommentBody(action)
	if err != nil {
		return err
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if !cmd.Force {
		if err := cmd.confirmReviewAction(action, pr, body); err != nil {
			return err
		}
	}

	return cmd.executeReviewAction(ctx, prCtx, action, prID, body, pr)
}

func (cmd *ReviewCmd) ParsePRID() (int, error) {
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

func (cmd *ReviewCmd) validateReviewAction() (reviewAction, error) {
	actionCount := 0
	var action reviewAction

	if cmd.Approve {
		actionCount++
		action = actionApprove
	}
	if cmd.RequestChanges {
		actionCount++
		action = actionRequestChanges
	}
	if cmd.Comment {
		actionCount++
		action = actionComment
	}

	if actionCount == 0 {
		return action, fmt.Errorf("must specify one of --approve, --request-changes, or --comment")
	}
	if actionCount > 1 {
		return action, fmt.Errorf("cannot specify multiple review actions (--approve, --request-changes, --comment)")
	}

	return action, nil
}

func (cmd *ReviewCmd) getCommentBody(action reviewAction) (string, error) {
	var body string

	if cmd.Body != "" {
		body = cmd.Body
	}

	if cmd.BodyFile != "" {
		if cmd.Body != "" {
			return "", fmt.Errorf("cannot specify both --body and --body-file")
		}

		fileContent, err := os.ReadFile(cmd.BodyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read body file '%s': %w", cmd.BodyFile, err)
		}
		body = string(fileContent)
	}

	if action == actionRequestChanges && strings.TrimSpace(body) == "" {
		if cmd.Force {
			return "", fmt.Errorf("comment is required when requesting changes")
		}

		fmt.Print("Comment (required for requesting changes): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			body = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read comment: %w", err)
		}

		if strings.TrimSpace(body) == "" {
			return "", fmt.Errorf("comment is required when requesting changes")
		}
	}

	if strings.TrimSpace(body) == "" && !cmd.Force && (action == actionApprove || action == actionComment) {
		prompt := "Comment (optional): "
		if action == actionComment {
			prompt = "Comment: "
		}

		fmt.Print(prompt)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			body = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read comment: %w", err)
		}
	}

	return strings.TrimSpace(body), nil
}

func (cmd *ReviewCmd) confirmReviewAction(action reviewAction, pr *api.PullRequest, body string) error {
	actionStr := ""
	switch action {
	case actionApprove:
		actionStr = "approve"
	case actionRequestChanges:
		actionStr = "request changes on"
	case actionComment:
		actionStr = "comment on"
	}

	fmt.Printf("Review #%d (%s):\n", pr.ID, pr.Title)
	fmt.Printf("Action: %s\n", actionStr)
	if body != "" {
		fmt.Printf("Comment: %s\n", body)
	}
	fmt.Print("\nProceed? [y/N] ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "y" && response != "yes" {
			return fmt.Errorf("review cancelled")
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	return nil
}

func (cmd *ReviewCmd) executeReviewAction(ctx context.Context, prCtx *PRContext, action reviewAction, prID int, body string, pr *api.PullRequest) error {
	switch action {
	case actionApprove:
		return cmd.executeApproval(ctx, prCtx, prID, body, pr)
	case actionRequestChanges:
		return cmd.executeRequestChanges(ctx, prCtx, prID, body, pr)
	case actionComment:
		return cmd.executeComment(ctx, prCtx, prID, body, pr)
	default:
		return fmt.Errorf("invalid review action")
	}
}

func (cmd *ReviewCmd) executeApproval(ctx context.Context, prCtx *PRContext, prID int, body string, pr *api.PullRequest) error {
	approval, err := prCtx.Client.PullRequests.ApprovePullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if body != "" {
		_, err = prCtx.Client.PullRequests.AddComment(ctx, prCtx.Workspace, prCtx.Repository, prID, body, nil)
		if err != nil {
			fmt.Printf("Warning: Failed to add comment: %v\n", err)
		}
	}

	return cmd.formatApprovalOutput(prCtx, pr, approval, body)
}

func (cmd *ReviewCmd) executeRequestChanges(ctx context.Context, prCtx *PRContext, prID int, body string, pr *api.PullRequest) error {
	comment, err := prCtx.Client.PullRequests.RequestChanges(ctx, prCtx.Workspace, prCtx.Repository, prID, body)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatRequestChangesOutput(prCtx, pr, comment)
}

func (cmd *ReviewCmd) executeComment(ctx context.Context, prCtx *PRContext, prID int, body string, pr *api.PullRequest) error {
	if body == "" {
		return fmt.Errorf("comment body is required")
	}

	comment, err := prCtx.Client.PullRequests.AddComment(ctx, prCtx.Workspace, prCtx.Repository, prID, body, nil)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatCommentOutput(prCtx, pr, comment)
}

func (cmd *ReviewCmd) formatApprovalOutput(prCtx *PRContext, pr *api.PullRequest, approval *api.PullRequestApproval, body string) error {
	switch cmd.Output {
	case "table":
		fmt.Printf("✓ Approved pull request #%d (%s)\n", pr.ID, pr.Title)
		if body != "" {
			fmt.Printf("Comment: %s\n", body)
		}
		return nil
	case "json", "yaml":
		output := map[string]interface{}{
			"action":       "approved",
			"pull_request": pr,
			"approval":     approval,
		}
		if body != "" {
			output["comment"] = body
		}
		return prCtx.Formatter.Format(output)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ReviewCmd) formatRequestChangesOutput(prCtx *PRContext, pr *api.PullRequest, comment *api.PullRequestComment) error {
	switch cmd.Output {
	case "table":
		fmt.Printf("✓ Requested changes on pull request #%d (%s)\n", pr.ID, pr.Title)
		if comment.Content != nil && comment.Content.Raw != "" {
			fmt.Printf("Comment: %s\n", comment.Content.Raw)
		}
		return nil
	case "json", "yaml":
		output := map[string]interface{}{
			"action":       "requested_changes",
			"pull_request": pr,
			"comment":      comment,
		}
		return prCtx.Formatter.Format(output)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ReviewCmd) formatCommentOutput(prCtx *PRContext, pr *api.PullRequest, comment *api.PullRequestComment) error {
	switch cmd.Output {
	case "table":
		fmt.Printf("✓ Added comment to pull request #%d (%s)\n", pr.ID, pr.Title)
		if comment.Content != nil && comment.Content.Raw != "" {
			fmt.Printf("Comment: %s\n", comment.Content.Raw)
		}
		return nil
	case "json", "yaml":
		output := map[string]interface{}{
			"action":       "commented",
			"pull_request": pr,
			"comment":      comment,
		}
		return prCtx.Formatter.Format(output)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}
