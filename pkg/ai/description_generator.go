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
	client     *api.Client
	repo       *git.Repository
	workspace  string
	repository string
	noColor    bool
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
	Template     string
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

	schema, err := g.openaiClient.GeneratePRDescription(ctx, input, opts.Template)
	if err != nil {
		return nil, err
	}

	if opts.Debug {
		fmt.Printf("\n=== DEBUG: OpenAI Schema Response ===\n")
		fmt.Printf("Contexto: %s\n", schema.Contexto)
		fmt.Printf("Alteracoes: %v\n", schema.Alteracoes)
		fmt.Printf("ChecklistItems: %v\n", schema.ChecklistItems)
		fmt.Printf("EvidencePlaceholders: %v\n", schema.EvidencePlaceholders)
		fmt.Printf("Title: %s\n", schema.Title)
		fmt.Printf("JiraTicket: %s\n", schema.JiraTicket)
		fmt.Printf("ClientSpecific: %s\n", schema.ClientSpecific)
		fmt.Printf("=== END DEBUG ===\n\n")
	}

	checklist := strings.Join(schema.ChecklistItems, "\n\n")
	if strings.TrimSpace(checklist) == "" {
		checklist = "✅ Testado localmente\n\n✅ Código revisado"
	}
	
	evidencePlaceholders := strings.Join(schema.EvidencePlaceholders, "\n\n")
	if strings.TrimSpace(evidencePlaceholders) == "" {
		evidencePlaceholders = "- [ ] Evidências de teste\n\n- [ ] Documentação relevante"
	}

	templateVars := map[string]interface{}{
		"contexto":              schema.Contexto,
		"alteracoes":            strings.Join(schema.Alteracoes, "\n\n"),
		"checklist":             checklist,
		"evidence_placeholders": evidencePlaceholders,
		"branch_name":           opts.SourceBranch,
		"target_branch":         opts.TargetBranch,
		"files_changed":         diffData.Stats.FilesChanged,
		"additions":             diffData.Stats.LinesAdded,
		"deletions":             diffData.Stats.LinesRemoved,
		"jira_ticket":           schema.JiraTicket,
		"client_specific":       schema.ClientSpecific,
	}

	template := NewTemplateEngine(opts.Template)
	description, err := template.Apply(templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template with OpenAI data: %w", err)
	}

	return &PRDescriptionResult{
		Title:       schema.Title,
		Description: description,
		Stats:       diffData.Stats,
		Metadata: map[string]interface{}{
			"branch_name":    opts.SourceBranch,
			"target_branch":  opts.TargetBranch,
			"template":       opts.Template,
			"has_jira":       opts.JiraFile != "",
			"openai_used":    true,
			"files_changed":  diffData.Stats.FilesChanged,
			"lines_added":    diffData.Stats.LinesAdded,
			"lines_removed":  diffData.Stats.LinesRemoved,
		},
		Generated: time.Now(),
	}, nil
}

func (g *DescriptionGenerator) generateWithLocalTemplates(ctx context.Context, opts *GenerateOptions, branchContext *BranchContext, diffData *DiffData, jiraContext string) (*PRDescriptionResult, error) {
	if opts.Verbose {
		g.logStep("🧠 Generating changes summary with local templates...")
	}

	analysis, err := g.analyzeDiff(diffData)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze diff: %w", err)
	}

	templateVars := g.buildTemplateVariables(branchContext, analysis, jiraContext, diffData.Stats)
	
	if opts.Verbose {
		g.logStep("📝 Creating checklist based on change types...")
	}

	checklist := g.generateChecklist(analysis)
	templateVars["checklist"] = checklist

	if opts.Verbose {
		g.logStep(fmt.Sprintf("🎯 Applying %s template...", opts.Template))
	}

	template := NewTemplateEngine(opts.Template)
	description, err := template.Apply(templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template: %w", err)
	}

	title := g.generateTitle(branchContext, analysis)

	if opts.Verbose {
		g.logStep("✅ Local template description generated successfully!")
		g.logStep("")
		g.logStep("📋 Generated Description:")
	}

	result := &PRDescriptionResult{
		Title:       title,
		Description: description,
		Stats:       diffData.Stats,
		Metadata: map[string]interface{}{
			"branch_name":    opts.SourceBranch,
			"target_branch":  opts.TargetBranch,
			"template":       opts.Template,
			"has_jira":       opts.JiraFile != "",
			"change_types":   analysis.ChangeTypes,
			"openai_used":    false,
			"files_changed":  diffData.Stats.FilesChanged,
			"lines_added":    diffData.Stats.LinesAdded,
			"lines_removed":  diffData.Stats.LinesRemoved,
		},
		Generated: time.Now(),
	}

	return result, nil
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
	vars := map[string]interface{}{
		"branch_name":   branchContext.SourceBranch,
		"target_branch": branchContext.TargetBranch,
		"files_changed": stats.FilesChanged,
		"additions":     stats.LinesAdded,
		"deletions":     stats.LinesRemoved,
	}

	if jiraContext != "" {
		vars["contexto"] = g.extractContextFromJira(jiraContext)
		vars["jira_ticket"] = g.extractJiraTicket(jiraContext)
		vars["client_specific"] = g.extractClientSpecific(jiraContext)
	} else {
		vars["contexto"] = g.generateContextFromBranch(branchContext, analysis)
		vars["jira_ticket"] = ""
		vars["client_specific"] = ""
	}

	vars["alteracoes"] = g.generateChanges(analysis)

	vars["evidence_placeholders"] = g.generateEvidencePlaceholders(analysis)

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

func (g *DescriptionGenerator) generateChecklist(analysis *DiffAnalysis) []string {
	var checklist []string

	for _, changeType := range analysis.ChangeTypes {
		switch changeType {
		case "backend":
			checklist = append(checklist, "✅ Testado localmente")
			checklist = append(checklist, "✅ Testes unitários executados")
			checklist = append(checklist, "✅ Documentação atualizada")
		case "frontend":
			checklist = append(checklist, "✅ Testado em diferentes navegadores")
			checklist = append(checklist, "✅ Responsividade verificada")
			checklist = append(checklist, "✅ Acessibilidade verificada")
		case "database":
			checklist = append(checklist, "✅ Migration testada")
			checklist = append(checklist, "✅ Backup realizado")
			checklist = append(checklist, "✅ Rollback testado")
		case "api":
			checklist = append(checklist, "✅ Documentação da API atualizada")
			checklist = append(checklist, "✅ Testes de integração executados")
			checklist = append(checklist, "✅ Versionamento da API considerado")
		case "configuration":
			checklist = append(checklist, "✅ Configurações validadas")
			checklist = append(checklist, "✅ Variáveis de ambiente documentadas")
		case "documentation":
			checklist = append(checklist, "✅ Documentação revisada")
			checklist = append(checklist, "✅ Links verificados")
		}
	}

	if len(checklist) == 0 {
		checklist = append(checklist, "✅ Testado localmente")
		checklist = append(checklist, "✅ Código revisado")
	}

	return checklist
}

func (g *DescriptionGenerator) extractContextFromJira(jiraContext string) string {
	lines := strings.Split(jiraContext, "\n")
	for _, line := range lines {
		if strings.Contains(line, "## Contexto") || strings.Contains(line, "## Context") {
			context := strings.TrimSpace(line)
			if context != "" {
				return context
			}
		}
	}
	return "Contexto extraído do JIRA"
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
	return ""
}

func (g *DescriptionGenerator) extractClientSpecific(jiraContext string) string {
	if strings.Contains(strings.ToLower(jiraContext), "client") || strings.Contains(strings.ToLower(jiraContext), "cliente") {
		return "Cliente específico"
	}
	return ""
}

func (g *DescriptionGenerator) generateContextFromBranch(branchContext *BranchContext, analysis *DiffAnalysis) string {
	branchName := branchContext.SourceBranch
	
	if strings.Contains(branchName, "feature") {
		return "Implementação de nova funcionalidade"
	} else if strings.Contains(branchName, "fix") || strings.Contains(branchName, "bug") {
		return "Correção de bug identificado"
	} else if strings.Contains(branchName, "hotfix") {
		return "Correção crítica em produção"
	} else if strings.Contains(branchName, "refactor") {
		return "Refatoração de código existente"
	}
	
	return "Desenvolvimento de melhorias no sistema"
}

func (g *DescriptionGenerator) generateChanges(analysis *DiffAnalysis) string {
	var changes []string
	
	for _, changeType := range analysis.ChangeTypes {
		switch changeType {
		case "backend":
			changes = append(changes, "• Alterações no backend")
		case "frontend":
			changes = append(changes, "• Modificações na interface do usuário")
		case "database":
			changes = append(changes, "• Alterações no banco de dados")
		case "api":
			changes = append(changes, "• Modificações na API")
		case "configuration":
			changes = append(changes, "• Atualizações de configuração")
		case "documentation":
			changes = append(changes, "• Atualizações na documentação")
		case "tests":
			changes = append(changes, "• Adição/atualização de testes")
		}
	}
	
	if len(changes) == 0 {
		changes = append(changes, "• Implementação de melhorias no código")
	}
	
	return strings.Join(changes, "\n")
}

func (g *DescriptionGenerator) generateEvidencePlaceholders(analysis *DiffAnalysis) string {
	var placeholders []string
	
	for _, changeType := range analysis.ChangeTypes {
		switch changeType {
		case "frontend":
			placeholders = append(placeholders, "- [ ] Screenshots da interface")
			placeholders = append(placeholders, "- [ ] Testes de responsividade")
		case "backend":
			placeholders = append(placeholders, "- [ ] Logs de teste")
			placeholders = append(placeholders, "- [ ] Resultados de testes unitários")
		case "database":
			placeholders = append(placeholders, "- [ ] Scripts de migration")
			placeholders = append(placeholders, "- [ ] Testes de rollback")
		case "api":
			placeholders = append(placeholders, "- [ ] Documentação da API")
			placeholders = append(placeholders, "- [ ] Testes de integração")
		}
	}
	
	if len(placeholders) == 0 {
		placeholders = append(placeholders, "- [ ] Evidências de teste")
		placeholders = append(placeholders, "- [ ] Documentação relevante")
	}
	
	return strings.Join(placeholders, "\n")
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
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			fmt.Println(cleaned)
		}
	}
}
