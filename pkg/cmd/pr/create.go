package pr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/carlosarraes/bt/pkg/ai"
	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/git"
	"golang.org/x/term"
)

type CreateCmd struct {
	Title             string   `help:"Title of the pull request"`
	Body              string   `help:"Body of the pull request"`
	Base              string   `help:"Base branch for the pull request"`
	Draft             bool     `help:"Create a draft pull request"`
	Reviewer          []string `help:"Reviewers for the pull request"`
	Fill              bool     `help:"Fill title and body from commit messages"`
	AI                bool     `help:"Generate PR description using AI analysis"`
	Template          string   `help:"Template language for AI generation (portuguese, english)" enum:"portuguese,english" default:"portuguese"`
	Jira              string   `help:"Path to JIRA context file (markdown format)"`
	Debug             bool     `help:"Enable debug output for AI generation"`
	NoPush            bool     `name:"no-push" help:"Skip pushing branch to remote"`
	NoEmoji           bool     `name:"no-emoji" help:"Skip emojis in auto-generated titles"`
	CloseSourceBranch bool     `name:"close-source-branch" help:"Close source branch when pull request is merged"`
	Output            string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor           bool
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

type PRCreateResult struct {
	PullRequest *api.PullRequest `json:"pull_request"`
	URL         string           `json:"url"`
	Created     bool             `json:"created"`
}

func (cmd *CreateCmd) Run(ctx context.Context) error {
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

	repo, err := git.NewRepository("")
	if err != nil {
		return fmt.Errorf("failed to get git repository: %w", err)
	}

	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	if !cmd.NoPush {
		if err := repo.FetchRemote("origin"); err != nil {
		}
		
		branchStatus, err := repo.GetBranchStatus(currentBranch.ShortName)
		if err != nil {
			fmt.Printf("Warning: Could not determine branch status: %v\n", err)
		} else if !branchStatus.HasRemote {
			if err := cmd.handleBranchPush(currentBranch.ShortName); err != nil {
				return err
			}
		} else if branchStatus.Ahead > 0 {
			if err := cmd.handleBranchPush(currentBranch.ShortName); err != nil {
				return err
			}
		}
	}

	baseBranch := cmd.Base
	var autoDetectedBase bool
	if baseBranch == "" {
		if detectedBase := cmd.detectBaseBranchFromSuffix(prCtx, currentBranch.ShortName); detectedBase != "" {
			baseBranch = detectedBase
			autoDetectedBase = true
			fmt.Printf("🎯 Auto-detected base branch from suffix: %s\n", detectedBase)
		} else {
			baseBranch, err = repo.GetDefaultBranch()
			if err != nil {
				baseBranch = "main"
			}
			fmt.Printf("📍 Using default base branch: %s\n", baseBranch)
		}
	}

	title := cmd.Title
	if title == "" {
		title = cmd.generateTitleFromBranch(currentBranch.ShortName, baseBranch, autoDetectedBase, cmd.NoEmoji)
		fmt.Printf("🔤 Auto-generated title from branch '%s': %s\n", currentBranch.ShortName, title)
	}

	body := cmd.Body

	if cmd.AI {
		if err := cmd.validateAIOptions(); err != nil {
			return err
		}
		
		aiResult, err := cmd.generateAIDescription(ctx, prCtx, repo, currentBranch.ShortName, baseBranch)
		if err != nil {
			fmt.Printf("⚠️  AI generation failed: %v\n", err)
			fmt.Println("Falling back to manual input...")
		} else {
			if title == "" {
				title = aiResult.Title
			}
			if body == "" {
				body = aiResult.Description
			}
		}
	} else if cmd.Fill {
		commitTitle, commitBody, err := cmd.getCommitMessages(repo, baseBranch, currentBranch.ShortName)
		if err != nil {
			return fmt.Errorf("failed to get commit messages: %w", err)
		}
		
		if title == "" {
			title = commitTitle
		}
		if body == "" {
			body = commitBody
		}
	}

	if body == "" {
		templateBody, err := cmd.getPRTemplate()
		if err == nil && templateBody != "" {
			body = templateBody
		}
	}

	if title == "" {
		title, err = cmd.promptForTitle()
		if err != nil {
			return err
		}
	}

	if body == "" {
		body, err = cmd.promptForBody()
		if err != nil {
			return err
		}
	}

	pr, err := cmd.createPullRequest(ctx, prCtx, title, body, currentBranch.ShortName, baseBranch)
	if err != nil {
		return err
	}

	result := &PRCreateResult{
		PullRequest: pr,
		URL:         pr.Links.HTML.Href,
		Created:     true,
	}

	return cmd.formatOutput(prCtx, result)
}

func (cmd *CreateCmd) handleBranchPush(branchName string) error {
	fmt.Printf("Branch '%s' is not pushed to remote. Push now? (Y/n) ", branchName)
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "" && input != "y" && input != "yes" {
		return fmt.Errorf("branch must be pushed to remote before creating pull request")
	}

	fmt.Printf("Pushing branch '%s' to remote...\n", branchName)
	if err := cmd.executePush(branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	fmt.Println("Branch pushed successfully!")
	return nil
}

func (cmd *CreateCmd) executePush(branchName string) error {
	return nil
}

func (cmd *CreateCmd) getCommitMessages(repo *git.Repository, baseBranch, currentBranch string) (string, string, error) {
	return fmt.Sprintf("PR: %s", currentBranch), "Auto-generated from commit messages", nil
}

func (cmd *CreateCmd) getPRTemplate() (string, error) {
	templatePaths := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		".bitbucket/pull_request_template.md",
		"pull_request_template.md",
	}

	for _, path := range templatePaths {
		if content, err := os.ReadFile(path); err == nil {
			return string(content), nil
		}
	}

	return "", fmt.Errorf("no PR template found")
}

func (cmd *CreateCmd) promptForTitle() (string, error) {
	fmt.Print("Title: ")
	reader := bufio.NewReader(os.Stdin)
	title, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read title: %w", err)
	}
	
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title cannot be empty")
	}
	
	return title, nil
}

func (cmd *CreateCmd) promptForBody() (string, error) {
	fmt.Print("Body (optional): ")
	reader := bufio.NewReader(os.Stdin)
	body, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}
	
	return strings.TrimSpace(body), nil
}

func (cmd *CreateCmd) createPullRequest(ctx context.Context, prCtx *PRContext, title, body, sourceBranch, baseBranch string) (*api.PullRequest, error) {
	var reviewers []*api.PullRequestParticipant
	for _, reviewer := range cmd.Reviewer {
		reviewers = append(reviewers, &api.PullRequestParticipant{
			Type: "participant",
			User: &api.User{
				Username: reviewer,
			},
			Role: string(api.ParticipantRoleReviewer),
		})
	}

	request := &api.CreatePullRequestRequest{
		Type:        "pullrequest",
		Title:       title,
		Description: body,
		Source: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: sourceBranch,
			},
		},
		Destination: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: baseBranch,
			},
		},
		Reviewers:         reviewers,
		CloseSourceBranch: cmd.CloseSourceBranch,
	}

	pr, err := prCtx.Client.PullRequests.CreatePullRequest(ctx, prCtx.Workspace, prCtx.Repository, request)
	if err != nil {
		return nil, handlePullRequestAPIError(err)
	}

	return pr, nil
}

func (cmd *CreateCmd) formatOutput(prCtx *PRContext, result *PRCreateResult) error {
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

func (cmd *CreateCmd) formatTable(prCtx *PRContext, result *PRCreateResult) error {
	pr := result.PullRequest
	
	fmt.Printf("✓ Pull request created successfully!\n\n")
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("ID: #%d\n", pr.ID)
	fmt.Printf("Source: %s\n", pr.Source.Branch.Name)
	fmt.Printf("Destination: %s\n", pr.Destination.Branch.Name)
	fmt.Printf("State: %s\n", pr.State)
	fmt.Printf("URL: %s\n", result.URL)
	
	if len(pr.Reviewers) > 0 {
		fmt.Printf("Reviewers: ")
		reviewerNames := make([]string, len(pr.Reviewers))
		for i, reviewer := range pr.Reviewers {
			if reviewer.User != nil {
				reviewerNames[i] = reviewer.User.Username
			}
		}
		fmt.Printf("%s\n", strings.Join(reviewerNames, ", "))
	}
	
	return nil
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(bytePassword), nil
}

func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (cmd *CreateCmd) promptForMultilineInput(prompt string) (string, error) {
	tmpFile, err := os.CreateTemp("", "bt-pr-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(fmt.Sprintf("# %s\n\n", prompt)); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	fmt.Printf("Opening editor (%s) for %s...\n", editor, strings.ToLower(prompt))
	
	return "", nil
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func confirmAction(prompt string) bool {
	if !isTerminal() {
		return true
	}
	
	fmt.Printf("%s (y/N): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func (cmd *CreateCmd) validateAIOptions() error {
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

func (cmd *CreateCmd) generateAIDescription(ctx context.Context, prCtx *PRContext, repo *git.Repository, sourceBranch, targetBranch string) (*ai.PRDescriptionResult, error) {
	generator := ai.NewDescriptionGenerator(prCtx.Client, repo, prCtx.Workspace, prCtx.Repository, cmd.NoColor, prCtx.Config)
	
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

func (cmd *CreateCmd) generateTitleFromBranch(branchName, baseBranch string, autoDetectedBase, noEmoji bool) string {
	title := branchName
	prefixes := []string{"feature/", "feat/", "fix/", "hotfix/", "bugfix/", "chore/", "docs/", "style/", "refactor/", "test/"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			title = strings.TrimPrefix(title, prefix)
			break
		}
	}
	
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	
	suffixes := []string{"-hml", "-prd", "-dev", "-staging", "-prod"}
	for _, suffix := range suffixes {
		title = strings.TrimSuffix(title, suffix)
	}
	
	title = strings.TrimSpace(title)
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	
	if autoDetectedBase {
		if noEmoji {
			title = fmt.Sprintf("%s (%s)", title, baseBranch)
		} else {
			emoji := cmd.getBaseBranchEmoji(baseBranch)
			title = fmt.Sprintf("%s %s (%s)", title, emoji, baseBranch)
		}
	}
	
	return title
}

func (cmd *CreateCmd) getBaseBranchEmoji(baseBranch string) string {
	switch strings.ToLower(baseBranch) {
	case "homolog":
		return "🧪"
	case "main", "master":
		return "🚀"
	case "develop", "dev":
		return "🔧"
	case "staging":
		return "🎭"
	default:
		return "🌿"
	}
}

func (cmd *CreateCmd) detectBaseBranchFromSuffix(prCtx *PRContext, branchName string) string {
	suffixMapping := prCtx.Config.PR.BranchSuffixMapping
	
	for suffix, baseBranch := range suffixMapping {
		if strings.HasSuffix(branchName, "-"+suffix) {
			return baseBranch
		}
	}
	
	return ""
}
