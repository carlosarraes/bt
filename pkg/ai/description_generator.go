package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/git"
	"github.com/carlosarraes/bt/pkg/utils"
)

type DescriptionGenerator struct {
	client       *api.Client
	repo         *git.Repository
	workspace    string
	repository   string
	noColor      bool
	openaiClient *OpenAIClient
}

func NewDescriptionGenerator(client *api.Client, repo *git.Repository, workspace, repository string, noColor bool, cfg *config.Config) *DescriptionGenerator {
	openaiClient, _ := NewOpenAIClientWithConfig(cfg)

	return &DescriptionGenerator{
		client:       client,
		repo:         repo,
		workspace:    workspace,
		repository:   repository,
		noColor:      noColor,
		openaiClient: openaiClient,
	}
}

type GenerateOptions struct {
	SourceBranch string
	TargetBranch string
	JiraFile     string
	Verbose      bool
	Debug        bool
}

type PRDescriptionResult struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Stats       *utils.DiffStats       `json:"stats"`
	Metadata    map[string]interface{} `json:"metadata"`
	Generated   time.Time              `json:"generated"`
}

func (g *DescriptionGenerator) GenerateDescription(ctx context.Context, opts *GenerateOptions) (*PRDescriptionResult, error) {
	if opts.Verbose {
		g.logStep("🔍 Analyzing PR context...")
	}

	branchContext, err := g.getBranchContext(opts.SourceBranch, opts.TargetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch context: %w", err)
	}

	if opts.Verbose {
		g.logStep("📊 Analyzing code changes...")
	}

	diffData, err := g.getGitDiff(opts.SourceBranch, opts.TargetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	if opts.Verbose {
		g.logStep(fmt.Sprintf("🏷️  Categorizing changes: %d files changed (+%d -%d lines)",
			diffData.Stats.FilesChanged, diffData.Stats.LinesAdded, diffData.Stats.LinesRemoved))
	}

	var jiraContext string
	if opts.JiraFile != "" {
		if opts.Verbose {
			g.logStep("📋 Reading JIRA context...")
		}

		jiraContext, err = g.readJiraContext(opts.JiraFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read JIRA context: %w", err)
		}
	}

	if g.openaiClient != nil {
		if opts.Verbose {
			g.logStep(fmt.Sprintf("🤖 Generating description with OpenAI %s...", g.openaiClient.GetModel()))
		}

		result, err := g.generateWithOpenAI(ctx, opts, branchContext, diffData, jiraContext)
		if err == nil {
			if opts.Verbose {
				g.logStep("✅ OpenAI description generated successfully!")
			}
			return result, nil
		}

		if opts.Verbose {
			g.logStep(fmt.Sprintf("⚠️  OpenAI generation failed: %v", err))
			g.logStep("🔄 Falling back to local template generation...")
		}
	}

	return g.generateWithLocalTemplates(ctx, opts, branchContext, diffData, jiraContext)
}

func (g *DescriptionGenerator) generateWithOpenAI(ctx context.Context, opts *GenerateOptions, branchContext *BranchContext, diffData *DiffData, jiraContext string) (*PRDescriptionResult, error) {
	input := &PRAnalysisInput{
		SourceBranch:   opts.SourceBranch,
		TargetBranch:   opts.TargetBranch,
		CommitMessages: branchContext.Commits,
		ChangedFiles:   diffData.Files,
		GitDiff:        diffData.Content,
		FilesChanged:   diffData.Stats.FilesChanged,
		LinesAdded:     diffData.Stats.LinesAdded,
		LinesRemoved:   diffData.Stats.LinesRemoved,
		JiraContext:    jiraContext,
	}

	if opts.Debug {
		fmt.Printf("\n=== DEBUG: AI INPUT DATA ===\n")
		fmt.Printf("Source Branch: %s\n", input.SourceBranch)
		fmt.Printf("Target Branch: %s\n", input.TargetBranch)
		fmt.Printf("Files Changed: %d\n", input.FilesChanged)
		fmt.Printf("Lines Added: %d\n", input.LinesAdded)
		fmt.Printf("Lines Removed: %d\n", input.LinesRemoved)
		fmt.Printf("\nCommit Messages:\n")
		for i, commit := range input.CommitMessages {
			fmt.Printf("  %d: %s\n", i+1, commit)
		}
		fmt.Printf("\nChanged Files:\n")
		for i, file := range input.ChangedFiles {
			fmt.Printf("  %d: %s\n", i+1, file)
		}
		fmt.Printf("\nGit Diff (first 1000 chars):\n")
		fmt.Printf("---\n")
		if len(input.GitDiff) > 1000 {
			fmt.Printf("%s\n... [truncated for display]\n", input.GitDiff[:1000])
		} else {
			fmt.Printf("%s\n", input.GitDiff)
		}
		fmt.Printf("---\n")
		fmt.Printf("=== END DEBUG ===\n\n")
	}

	schema, err := g.openaiClient.GeneratePRDescription(ctx, input)
	if err != nil {
		return nil, err
	}

	if opts.Debug {
		fmt.Printf("\n=== DEBUG: OpenAI Schema Response ===\n")
		fmt.Printf("Title: %s\n", schema.Title)
		fmt.Printf("ChangeType: %s\n", schema.ChangeType)
		fmt.Printf("Summary: %s\n", schema.Summary)
		fmt.Printf("JiraTicket: %s\n", schema.JiraTicket)
		fmt.Printf("UIChanges: %s\n", schema.UIChanges)
		fmt.Printf("DBArchitecture: %s\n", schema.DBArchitecture)
		fmt.Printf("Dependencies: %s\n", schema.Dependencies)
		fmt.Printf("Documentation: %s\n", schema.Documentation)
		fmt.Printf("TestCases: %s\n", schema.TestCases)
		fmt.Printf("BugFixDetails: %s\n", schema.BugFixDetails)
		fmt.Printf("Security: %s\n", schema.Security)
		fmt.Printf("RollbackSafety: %s\n", schema.RollbackSafety)
		fmt.Printf("=== END DEBUG ===\n\n")
	}

	templateVars := map[string]interface{}{
		"change_type":           schema.ChangeType,
		"summary":               schema.Summary,
		"jira_ticket":           coalesce(schema.JiraTicket, "[Link]"),
		"design_doc":            "[Link/NA]",
		"ui_changes":            schema.UIChanges,
		"db_architecture":       schema.DBArchitecture,
		"dependencies":          schema.Dependencies,
		"documentation":         schema.Documentation,
		"testing_env":           "[Local / Homolog / N/A]",
		"test_cases":            schema.TestCases,
		"bug_fix_details":       schema.BugFixDetails,
		"feature_flags":         "(List new feature flags added and how to enable them)",
		"security":              schema.Security,
		"monitoring":            "(List Datadog dashboards, new logs, or specific alerts to watch)",
		"rollback_safety":       schema.RollbackSafety,
		"production_validation": "How will you confirm success after deployment?",
		"branch_name":           opts.SourceBranch,
		"target_branch":         opts.TargetBranch,
		"files_changed":         diffData.Stats.FilesChanged,
		"additions":             diffData.Stats.LinesAdded,
		"deletions":             diffData.Stats.LinesRemoved,
	}

	tmpl := NewTemplateEngine()
	description, err := tmpl.Apply(templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template with OpenAI data: %w", err)
	}

	return &PRDescriptionResult{
		Title:       schema.Title,
		Description: description,
		Stats:       diffData.Stats,
		Metadata: map[string]interface{}{
			"branch_name":   opts.SourceBranch,
			"target_branch": opts.TargetBranch,
			"has_jira":      opts.JiraFile != "",
			"openai_used":   true,
			"files_changed": diffData.Stats.FilesChanged,
			"lines_added":   diffData.Stats.LinesAdded,
			"lines_removed": diffData.Stats.LinesRemoved,
		},
		Generated: time.Now(),
	}, nil
}

func (g *DescriptionGenerator) generateWithLocalTemplates(_ context.Context, opts *GenerateOptions, branchContext *BranchContext, diffData *DiffData, jiraContext string) (*PRDescriptionResult, error) {
	if opts.Verbose {
		g.logStep("🧠 Generating description with local templates...")
	}

	analysis, err := g.analyzeDiff(diffData)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze diff: %w", err)
	}

	templateVars := g.buildTemplateVariables(branchContext, analysis, jiraContext, diffData.Stats)

	if opts.Verbose {
		g.logStep("📝 Applying template...")
	}

	tmpl := NewTemplateEngine()
	description, err := tmpl.Apply(templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template: %w", err)
	}

	title := g.generateTitle(branchContext, analysis)

	if opts.Verbose {
		g.logStep("✅ Local template description generated successfully!")
	}

	return &PRDescriptionResult{
		Title:       title,
		Description: description,
		Stats:       diffData.Stats,
		Metadata: map[string]interface{}{
			"branch_name":   opts.SourceBranch,
			"target_branch": opts.TargetBranch,
			"has_jira":      opts.JiraFile != "",
			"change_types":  analysis.ChangeTypes,
			"openai_used":   false,
			"files_changed": diffData.Stats.FilesChanged,
			"lines_added":   diffData.Stats.LinesAdded,
			"lines_removed": diffData.Stats.LinesRemoved,
		},
		Generated: time.Now(),
	}, nil
}

type DiffData struct {
	Content string
	Stats   *utils.DiffStats
	Files   []string
}

func (g *DescriptionGenerator) getGitDiff(sourceBranch, targetBranch string) (*DiffData, error) {
	diffContent, err := g.repo.GetDiff(targetBranch, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	changedFiles, err := g.repo.GetChangedFiles(targetBranch, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	stats, err := g.calculateDiffStats(diffContent)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate diff stats: %w", err)
	}

	return &DiffData{
		Content: diffContent,
		Stats:   stats,
		Files:   changedFiles,
	}, nil
}

func (g *DescriptionGenerator) getBranchContext(sourceBranch, targetBranch string) (*BranchContext, error) {
	commits, err := g.getCommitMessages(sourceBranch, targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}

	return &BranchContext{
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Commits:      commits,
	}, nil
}

type BranchContext struct {
	SourceBranch string
	TargetBranch string
	Commits      []string
}

func (g *DescriptionGenerator) getCommitMessages(sourceBranch, targetBranch string) ([]string, error) {
	commits, err := g.repo.GetCommitMessages(targetBranch, sourceBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}

	if len(commits) == 0 {
		return []string{"No commits found"}, nil
	}

	return commits, nil
}

func (g *DescriptionGenerator) calculateDiffStats(diffContent string) (*utils.DiffStats, error) {
	lines := strings.Split(diffContent, "\n")

	var filesChanged int
	var linesAdded int
	var linesRemoved int

	currentFileChanges := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			filesChanged++
			currentFileChanges = true
			continue
		}

		if !currentFileChanges {
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			linesAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			linesRemoved++
		}
	}

	return &utils.DiffStats{
		FilesChanged: filesChanged,
		LinesAdded:   linesAdded,
		LinesRemoved: linesRemoved,
	}, nil
}

func (g *DescriptionGenerator) analyzeDiff(diffData *DiffData) (*DiffAnalysis, error) {
	analyzer := NewDiffAnalyzer()
	return analyzer.Analyze(diffData)
}

func (g *DescriptionGenerator) readJiraContext(jiraFile string) (string, error) {
	if _, err := os.Stat(jiraFile); os.IsNotExist(err) {
		return "", fmt.Errorf("JIRA context file not found: %s", jiraFile)
	}

	content, err := os.ReadFile(jiraFile)
	if err != nil {
		return "", fmt.Errorf("failed to read JIRA context file: %w", err)
	}

	return string(content), nil
}

func (g *DescriptionGenerator) buildTemplateVariables(branchContext *BranchContext, analysis *DiffAnalysis, jiraContext string, stats *utils.DiffStats) map[string]interface{} {
	changeType := g.detectChangeType(branchContext)

	vars := map[string]interface{}{
		"branch_name":           branchContext.SourceBranch,
		"target_branch":         branchContext.TargetBranch,
		"files_changed":         stats.FilesChanged,
		"additions":             stats.LinesAdded,
		"deletions":             stats.LinesRemoved,
		"change_type":           changeType,
		"summary":               g.generateSummaryFromBranch(branchContext, analysis),
		"jira_ticket":           "[Link]",
		"design_doc":            "[Link/NA]",
		"ui_changes":            g.detectUIChanges(analysis),
		"db_architecture":       g.detectDBChanges(analysis),
		"dependencies":          g.detectDependencyChanges(analysis),
		"documentation":         g.detectDocChanges(analysis),
		"testing_env":           "[Local / Homolog / N/A]",
		"test_cases":            g.generateTestCases(analysis),
		"bug_fix_details":       "",
		"feature_flags":         "(List new feature flags added and how to enable them)",
		"security":              g.detectSecurityImpact(analysis),
		"monitoring":            "(List Datadog dashboards, new logs, or specific alerts to watch)",
		"rollback_safety":       g.assessRollbackSafety(analysis),
		"production_validation": "How will you confirm success after deployment?",
	}

	if jiraContext != "" {
		vars["jira_ticket"] = g.extractJiraTicket(jiraContext)
	}
	if changeType == "Bug Fix" {
		vars["bug_fix_details"] = "**Severity:** [1-5] | **PR that introduced the bug:** [Link] | **Time in Production:** [Duration]"
	}

	return vars
}

func (g *DescriptionGenerator) generateTitle(branchContext *BranchContext, analysis *DiffAnalysis) string {
	branchName := branchContext.SourceBranch

	title := strings.TrimPrefix(branchName, "feature/")
	title = strings.TrimPrefix(title, "fix/")
	title = strings.TrimPrefix(title, "hotfix/")
	title = strings.TrimPrefix(title, "bugfix/")
	title = strings.TrimPrefix(title, "feat/")

	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}

	if len(branchContext.Commits) > 0 {
		commit := branchContext.Commits[0]
		if strings.Contains(commit, ":") {
			parts := strings.SplitN(commit, ":", 2)
			if len(parts) > 1 {
				title = strings.TrimSpace(parts[1])
				if len(title) > 0 {
					title = strings.ToUpper(title[:1]) + title[1:]
				}
			}
		}
	}

	return title
}

func (g *DescriptionGenerator) detectChangeType(branchContext *BranchContext) string {
	branch := strings.ToLower(branchContext.SourceBranch)

	if strings.HasPrefix(branch, "fix/") || strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "bugfix/") {
		return "Bug Fix"
	}
	if strings.HasPrefix(branch, "feature/") || strings.HasPrefix(branch, "feat/") {
		return "Feature"
	}
	if strings.HasPrefix(branch, "refactor/") {
		return "Refactor"
	}
	if strings.HasPrefix(branch, "chore/") || strings.HasPrefix(branch, "docs/") || strings.HasPrefix(branch, "style/") {
		return "Chore"
	}

	return "Feature"
}

func (g *DescriptionGenerator) detectUIChanges(analysis *DiffAnalysis) string {
	if contains(analysis.ChangeTypes, "frontend") {
		return "(Attach screenshots or screen recordings here)"
	}
	return "None"
}

func (g *DescriptionGenerator) detectDBChanges(analysis *DiffAnalysis) string {
	if contains(analysis.ChangeTypes, "database") {
		return "Performance/Locking impact? `[Yes / No]` — (Attach database impact screenshots)"
	}
	return "None"
}

func (g *DescriptionGenerator) detectDependencyChanges(analysis *DiffAnalysis) string {
	if analysis.ConfigChanges {
		return "(List any new libraries or required config changes)"
	}
	return "None"
}

func (g *DescriptionGenerator) detectDocChanges(analysis *DiffAnalysis) string {
	if analysis.DocsIncluded {
		return "Documentation updated"
	}
	return "None"
}

func (g *DescriptionGenerator) detectSecurityImpact(analysis *DiffAnalysis) string {
	for _, fa := range analysis.FileChanges {
		for _, pattern := range fa.Patterns {
			if pattern == "security" {
				return "Auth/sensitive data/permissions changes detected — review required"
			}
		}
	}
	return "No security impact detected"
}

func (g *DescriptionGenerator) assessRollbackSafety(analysis *DiffAnalysis) string {
	if contains(analysis.ChangeTypes, "database") {
		return "Caution — includes DB changes, verify rollback safety"
	}
	return "Safe to revert — no database migrations"
}

func (g *DescriptionGenerator) generateTestCases(analysis *DiffAnalysis) string {
	var cases []string

	for _, changeType := range analysis.ChangeTypes {
		switch changeType {
		case "backend":
			cases = append(cases, "Unit tests for modified backend logic")
		case "frontend":
			cases = append(cases, "UI testing across browsers and screen sizes")
		case "database":
			cases = append(cases, "Migration up/down tested, rollback verified")
		case "api":
			cases = append(cases, "API integration tests for modified endpoints")
		}
	}

	if analysis.TestsIncluded {
		cases = append(cases, "Automated tests included in this PR")
	}

	if len(cases) == 0 {
		return "(Add a list with scenarios, edge cases, and failure cases tested)"
	}

	return strings.Join(cases, "\n    - ")
}

func (g *DescriptionGenerator) generateSummaryFromBranch(branchContext *BranchContext, analysis *DiffAnalysis) string {
	branchName := branchContext.SourceBranch

	if strings.Contains(branchName, "feature") || strings.Contains(branchName, "feat") {
		return "Implementation of new functionality"
	} else if strings.Contains(branchName, "fix") || strings.Contains(branchName, "bug") {
		return "Bug fix for identified issue"
	} else if strings.Contains(branchName, "hotfix") {
		return "Critical production fix"
	} else if strings.Contains(branchName, "refactor") {
		return "Code refactoring"
	}

	return "System improvements"
}

func (g *DescriptionGenerator) extractJiraTicket(jiraContext string) string {
	lines := strings.Split(jiraContext, "\n")
	for _, line := range lines {
		if strings.Contains(line, "-") && len(strings.Fields(line)) > 0 {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, "-") && len(part) > 3 {
					return part
				}
			}
		}
	}
	return "[Link]"
}

func (g *DescriptionGenerator) logStep(message string) {
	if !g.noColor {
		fmt.Println(message)
	} else {
		cleaned := strings.ReplaceAll(message, "🔍", "")
		cleaned = strings.ReplaceAll(cleaned, "📊", "")
		cleaned = strings.ReplaceAll(cleaned, "🏷️", "")
		cleaned = strings.ReplaceAll(cleaned, "📋", "")
		cleaned = strings.ReplaceAll(cleaned, "🧠", "")
		cleaned = strings.ReplaceAll(cleaned, "📝", "")
		cleaned = strings.ReplaceAll(cleaned, "🎯", "")
		cleaned = strings.ReplaceAll(cleaned, "✅", "")
		cleaned = strings.ReplaceAll(cleaned, "🤖", "")
		cleaned = strings.ReplaceAll(cleaned, "⚠️", "")
		cleaned = strings.ReplaceAll(cleaned, "🔄", "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			fmt.Println(cleaned)
		}
	}
}

func coalesce(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
