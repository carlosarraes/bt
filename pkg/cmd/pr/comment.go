package pr

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
)

type CommentCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Body       string `short:"b" help:"Comment body text"`
	BodyFile   string `short:"F" name:"body-file" help:"Read comment body from file"`
	ReplyTo    string `name:"reply-to" help:"Reply to comment ID"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *CommentCmd) Run(ctx context.Context) error {
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

	body, err := cmd.getCommentBody()
	if err != nil {
		return err
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	var parentComment *api.PullRequestComment
	if cmd.ReplyTo != "" {
		parentComment, err = cmd.resolveReplyTo(ctx, prCtx, prID)
		if err != nil {
			return err
		}
	}

	comment, err := cmd.addComment(ctx, prCtx, prID, body, parentComment)
	if err != nil {
		return err
	}

	return cmd.displayResult(prCtx, pr, comment)
}

func (cmd *CommentCmd) ParsePRID() (int, error) {
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

func (cmd *CommentCmd) getCommentBody() (string, error) {
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

	if strings.TrimSpace(body) == "" {
		prompt := "Comment: "
		fmt.Print(prompt)
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			body = scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read comment: %w", err)
		}

		if strings.TrimSpace(body) == "" {
			return "", fmt.Errorf("comment cannot be empty")
		}
	}

	return strings.TrimSpace(body), nil
}

func (cmd *CommentCmd) resolveReplyTo(ctx context.Context, prCtx *PRContext, prID int) (*api.PullRequestComment, error) {
	replyToID, err := strconv.Atoi(cmd.ReplyTo)
	if err != nil {
		return nil, fmt.Errorf("invalid reply-to comment ID '%s': must be a positive integer", cmd.ReplyTo)
	}

	if replyToID <= 0 {
		return nil, fmt.Errorf("reply-to comment ID must be positive, got %d", replyToID)
	}

	commentsResp, err := prCtx.Client.PullRequests.GetComments(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve comments for reply-to validation: %w", err)
	}

	var commentsData []json.RawMessage
	if err := json.Unmarshal(commentsResp.Values, &commentsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal comments: %w", err)
	}

	for _, rawComment := range commentsData {
		var comment api.PullRequestComment
		if err := json.Unmarshal(rawComment, &comment); err != nil {
			continue
		}
		
		if comment.ID == replyToID {
			return &comment, nil
		}
	}

	return nil, fmt.Errorf("comment with ID %d not found in pull request #%d", replyToID, prID)
}

func (cmd *CommentCmd) addComment(ctx context.Context, prCtx *PRContext, prID int, body string, parentComment *api.PullRequestComment) (*api.PullRequestComment, error) {
	comment, err := prCtx.Client.PullRequests.AddComment(ctx, prCtx.Workspace, prCtx.Repository, prID, body, nil)
	if err != nil {
		if bitbucketErr, ok := err.(*api.BitbucketError); ok {
			switch bitbucketErr.Type {
			case api.ErrorTypeNotFound:
				return nil, fmt.Errorf("pull request #%d not found", prID)
			case api.ErrorTypeAuthentication:
				return nil, fmt.Errorf("authentication failed. Please run 'bt auth login' to authenticate")
			case api.ErrorTypePermission:
				return nil, fmt.Errorf("permission denied. You may not have access to comment on this pull request")
			case api.ErrorTypeRateLimit:
				return nil, fmt.Errorf("rate limit exceeded. Please wait before making more requests")
			default:
				return nil, fmt.Errorf("API error: %s", bitbucketErr.Message)
			}
		}
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	return comment, nil
}

func (cmd *CommentCmd) displayResult(prCtx *PRContext, pr *api.PullRequest, comment *api.PullRequestComment) error {
	switch cmd.Output {
	case "json":
		return prCtx.Formatter.Format(comment)
	case "yaml":
		return prCtx.Formatter.Format(comment)
	default:
		fmt.Printf("✓ Comment added to pull request #%d\n", pr.ID)
		fmt.Printf("  Title: %s\n", pr.Title)
		fmt.Printf("  Comment ID: %d\n", comment.ID)
		
		if comment.Links != nil && comment.Links.HTML != nil {
			fmt.Printf("  URL: %s\n", comment.Links.HTML.Href)
		}
		
		if comment.CreatedOn != nil {
			fmt.Printf("  Created: %s\n", output.FormatRelativeTime(comment.CreatedOn))
		}
		
		return nil
	}
}
