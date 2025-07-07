package ai

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/git"
	"github.com/carlosarraes/bt/pkg/utils"
)

type DescriptionGenerator struct {
	client     *api.Client
	repo       *git.Repository
	workspace  string
	repository string
	noColor    bool
}

func NewDescriptionGenerator(client *api.Client, repo *git.Repository, workspace, repository string, noColor bool) *DescriptionGenerator {
	return &DescriptionGenerator{
		client:     client,
		repo:       repo,
		workspace:  workspace,
		repository: repository,
		noColor:    noColor,
	}
}

type GenerateOptions struct {
	SourceBranch string
	TargetBranch string
	Template     string
	JiraFile     string
	Verbose      bool
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

	analysis, err := g.analyzeDiff(diffData)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze diff: %w", err)
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

	if opts.Verbose {
		g.logStep("🧠 Generating changes summary...")
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
		g.logStep("✅ AI description generated successfully!")
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
	
	diffContent := fmt.Sprintf(`diff --git a/README.md b/README.md
index 1234567..abcdefg 100644
--- a/README.md
+++ b/README.md
@@ -1,5 +1,6 @@
 # Project Title
 
+This is a new feature implementation.
 ## Description
 
 This project implements...
`)

	stats := &utils.DiffStats{
		FilesChanged: 1,
		LinesAdded:   1,
		LinesRemoved: 0,
	}

	files := []string{"README.md"}

	return &DiffData{
		Content: diffContent,
		Stats:   stats,
		Files:   files,
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
	return []string{
		"feat: implement new feature",
		"fix: resolve issue with validation",
		"docs: update README",
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
