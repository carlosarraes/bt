package pr

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/carlosarraes/bt/pkg/ai"
	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/git"
)

type EditCmd struct {
	PRID           string   `arg:"" help:"Pull request ID (number)"`
	Title          string   `help:"Edit pull request title"`
	Body           string   `help:"Edit pull request description"`
	BodyFile       string   `short:"F" name:"body-file" help:"Read description from file"`
	AddReviewer    []string `name:"add-reviewer" help:"Add reviewer by username"`
	RemoveReviewer []string `name:"remove-reviewer" help:"Remove reviewer by username"`
	Ready          bool     `help:"Mark pull request as ready for review (if draft)"`
	Draft          bool     `help:"Convert pull request to draft"`
	AI             bool     `help:"Generate PR description using AI analysis"`
	Template       string   `help:"Template language for AI generation (portuguese, english)" enum:"portuguese,english" default:"portuguese"`
	Jira           string   `help:"Path to JIRA context file (markdown format)"`
	Debug          bool     `help:"Print debug information including git diff and AI inputs"`
	Output         string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor        bool
	Workspace      string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository     string   `help:"Repository name (defaults to git remote)"`
}

func (cmd *EditCmd) Run(ctx context.Context) error {
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

	prID, err := ParsePRID(cmd.PRID)
	if err != nil {
		return err
	}

	if cmd.Ready && cmd.Draft {
		return fmt.Errorf("cannot use both --ready and --draft flags together")
	}

	if cmd.isInteractiveMode() {
		return cmd.runInteractiveMode(ctx, prCtx, prID)
	}

	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if cmd.AI {
		if err := cmd.validateAIOptions(); err != nil {
			return err
		}
		
		aiResult, err := cmd.generateAIDescription(ctx, prCtx, pr)
		if err != nil {
			fmt.Printf("⚠️  AI generation failed: %v\n", err)
			fmt.Println("Falling back to current description...")
		} else {
			if cmd.Body == "" {
				cmd.Body = aiResult.Description
			}
			if cmd.Title == "" {
				cmd.Title = aiResult.Title
			}
		}
	}

	updateReq, err := cmd.buildUpdateRequest(pr)
	if err != nil {
		return err
	}

	if cmd.hasChanges() {
		updatedPR, err := prCtx.Client.PullRequests.UpdatePullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID, updateReq)
		if err != nil {
			return handlePullRequestAPIError(err)
		}

		return cmd.formatOutput(prCtx, updatedPR)
	}

	fmt.Println("No changes specified. Use --help to see available options.")
	return nil
}


func (cmd *EditCmd) isInteractiveMode() bool {
	return cmd.Title == "" && cmd.Body == "" && cmd.BodyFile == "" &&
		len(cmd.AddReviewer) == 0 && len(cmd.RemoveReviewer) == 0 &&
		!cmd.Ready && !cmd.Draft && !cmd.AI
}

func (cmd *EditCmd) hasChanges() bool {
	return cmd.Title != "" || cmd.Body != "" || cmd.BodyFile != "" ||
		len(cmd.AddReviewer) > 0 || len(cmd.RemoveReviewer) > 0 ||
		cmd.Ready || cmd.Draft || cmd.AI
}

func (cmd *EditCmd) runInteractiveMode(ctx context.Context, prCtx *PRContext, prID int) error {
	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	tempFile, err := cmd.createTempFile(pr)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	editorCmd := exec.Command(editor, tempFile)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	newTitle, newBody, err := cmd.parseEditedContent(tempFile)
	if err != nil {
		return fmt.Errorf("failed to parse edited content: %w", err)
	}

	if newTitle == pr.Title && newBody == pr.Description {
		fmt.Println("No changes made.")
		return nil
	}

	updateReq := &api.UpdatePullRequestRequest{
		Title:       newTitle,
		Description: newBody,
	}

	updatedPR, err := prCtx.Client.PullRequests.UpdatePullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID, updateReq)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	return cmd.formatOutput(prCtx, updatedPR)
}

func (cmd *EditCmd) createTempFile(pr *api.PullRequest) (string, error) {
	tempFile, err := os.CreateTemp("", "bt-pr-edit-*.txt")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	content := fmt.Sprintf("# Edit pull request #%d\n", pr.ID)
	content += "# Lines starting with # are comments and will be ignored\n"
	content += "# First line will be the title, rest will be the description\n\n"
	content += pr.Title + "\n\n"
	content += pr.Description

	if _, err := tempFile.WriteString(content); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func (cmd *EditCmd) parseEditedContent(filename string) (string, string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(string(content), "\n")
	var contentLines []string

	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			contentLines = append(contentLines, line)
		}
	}

	fullContent := strings.Join(contentLines, "\n")
	fullContent = strings.TrimSpace(fullContent)

	if fullContent == "" {
		return "", "", fmt.Errorf("title cannot be empty")
	}

	parts := strings.SplitN(fullContent, "\n\n", 2)
	title := strings.TrimSpace(parts[0])
	body := ""

	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}

	if title == "" {
		return "", "", fmt.Errorf("title cannot be empty")
	}

	return title, body, nil
}

func (cmd *EditCmd) buildUpdateRequest(pr *api.PullRequest) (*api.UpdatePullRequestRequest, error) {
	updateReq := &api.UpdatePullRequestRequest{}

	if cmd.Title != "" {
		updateReq.Title = cmd.Title
	}

	if cmd.BodyFile != "" {
		bodyBytes, err := cmd.readBodyFile()
		if err != nil {
			return nil, err
		}
		updateReq.Description = string(bodyBytes)
	} else if cmd.Body != "" {
		updateReq.Description = cmd.Body
	}

	if cmd.Ready && pr.State == "DRAFT" {
		updateReq.State = "OPEN"
	} else if cmd.Draft && pr.State == "OPEN" {
		updateReq.State = "DRAFT"
	}

	if len(cmd.AddReviewer) > 0 || len(cmd.RemoveReviewer) > 0 {
		reviewers, err := cmd.buildReviewersList(pr)
		if err != nil {
			return nil, err
		}
		updateReq.Reviewers = reviewers
	}

	return updateReq, nil
}

func (cmd *EditCmd) buildReviewersList(pr *api.PullRequest) ([]*api.PullRequestParticipant, error) {
	existingReviewers := make(map[string]*api.PullRequestParticipant)
	
	for _, reviewer := range pr.Reviewers {
		if reviewer.User != nil {
			existingReviewers[reviewer.User.Username] = reviewer
		}
	}

	for _, username := range cmd.RemoveReviewer {
		delete(existingReviewers, username)
	}

	for _, username := range cmd.AddReviewer {
		if _, exists := existingReviewers[username]; !exists {
			existingReviewers[username] = &api.PullRequestParticipant{
				Type: "reviewer",
				User: &api.User{
					Username: username,
				},
				Role: "REVIEWER",
			}
		}
	}

	reviewers := make([]*api.PullRequestParticipant, 0, len(existingReviewers))
	for _, reviewer := range existingReviewers {
		reviewers = append(reviewers, reviewer)
	}

	return reviewers, nil
}

func (cmd *EditCmd) readBodyFile() ([]byte, error) {
	if cmd.BodyFile == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(cmd.BodyFile)
}

func (cmd *EditCmd) formatOutput(prCtx *PRContext, pr *api.PullRequest) error {
	fmt.Printf("✓ Pull request #%d has been updated\n\n", pr.ID)

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

func (cmd *EditCmd) formatTable(prCtx *PRContext, pr *api.PullRequest) error {
	fmt.Printf("#%d • %s\n", pr.ID, pr.Title)
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

	if pr.CreatedOn != nil {
		fmt.Printf("Created: %s\n", FormatRelativeTime(pr.CreatedOn))
	}
	if pr.UpdatedOn != nil {
		fmt.Printf("Updated: %s\n", FormatRelativeTime(pr.UpdatedOn))
	}

	if pr.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", pr.Description)
	}

	if len(pr.Reviewers) > 0 {
		fmt.Printf("\nReviewers:\n")
		for _, reviewer := range pr.Reviewers {
			if reviewer.User != nil {
				name := reviewer.User.DisplayName
				if name == "" {
					name = reviewer.User.Username
				}
				
				status := "pending"
				if reviewer.Approved {
					status = "approved"
				} else if reviewer.State == "changes_requested" {
					status = "changes requested"
				}
				
				fmt.Printf("  • %s (%s)\n", name, status)
			}
		}
	}

	return nil
}

func (cmd *EditCmd) validateAIOptions() error {
	if err := ai.ValidateLanguage(cmd.Template); err != nil {
		return err
	}
	
	if cmd.Jira != "" {
		if _, err := os.Stat(cmd.Jira); os.IsNotExist(err) {
			return fmt.Errorf("JIRA context file not found: %s", cmd.Jira)
		}
	}
	
	return nil
}

func (cmd *EditCmd) generateAIDescription(ctx context.Context, prCtx *PRContext, pr *api.PullRequest) (*ai.PRDescriptionResult, error) {
	repo, err := git.NewRepository("")
	if err != nil {
		return nil, fmt.Errorf("failed to get git repository: %w", err)
	}

	generator := ai.NewDescriptionGenerator(prCtx.Client, repo, prCtx.Workspace, prCtx.Repository, cmd.NoColor, prCtx.Config)
	
	sourceBranch := "unknown"
	targetBranch := "unknown"
	
	if pr.Source != nil && pr.Source.Branch != nil {
		sourceBranch = pr.Source.Branch.Name
	}
	if pr.Destination != nil && pr.Destination.Branch != nil {
		targetBranch = pr.Destination.Branch.Name
	}
	
	opts := &ai.GenerateOptions{
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Template:     cmd.Template,
		JiraFile:     cmd.Jira,
		Verbose:      true,
		Debug:        cmd.Debug,
	}
	
	result, err := generator.GenerateDescription(ctx, opts)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}
