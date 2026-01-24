package ai

import (
	"fmt"
	"path/filepath"
	"strings"
)

type DiffAnalyzer struct{}

func NewDiffAnalyzer() *DiffAnalyzer {
	return &DiffAnalyzer{}
}

type DiffAnalysis struct {
	ChangeTypes   []string                 `json:"change_types"`
	FileChanges   map[string]*FileAnalysis `json:"file_changes"`
	Summary       string                   `json:"summary"`
	Complexity    string                   `json:"complexity"`
	Impact        string                   `json:"impact"`
	TestsIncluded bool                     `json:"tests_included"`
	DocsIncluded  bool                     `json:"docs_included"`
	ConfigChanges bool                     `json:"config_changes"`
}

type FileAnalysis struct {
	Path         string   `json:"path"`
	Type         string   `json:"type"`
	Category     string   `json:"category"`
	Language     string   `json:"language"`
	LinesAdded   int      `json:"lines_added"`
	LinesRemoved int      `json:"lines_removed"`
	Patterns     []string `json:"patterns"`
}

func (a *DiffAnalyzer) Analyze(diffData *DiffData) (*DiffAnalysis, error) {
	analysis := &DiffAnalysis{
		ChangeTypes: []string{},
		FileChanges: make(map[string]*FileAnalysis),
	}

	for _, filePath := range diffData.Files {
		fileAnalysis := a.analyzeFile(filePath, diffData.Content)
		analysis.FileChanges[filePath] = fileAnalysis

		if !contains(analysis.ChangeTypes, fileAnalysis.Category) {
			analysis.ChangeTypes = append(analysis.ChangeTypes, fileAnalysis.Category)
		}
	}

	a.analyzePatterns(diffData.Content, analysis)

	analysis.Summary = a.generateSummary(analysis)
	analysis.Complexity = a.determineComplexity(analysis)
	analysis.Impact = a.determineImpact(analysis)

	return analysis, nil
}

func (a *DiffAnalyzer) analyzeFile(filePath, diffContent string) *FileAnalysis {
	analysis := &FileAnalysis{
		Path:     filePath,
		Language: a.detectLanguage(filePath),
		Patterns: []string{},
	}

	analysis.Category = a.categorizeFile(filePath)
	analysis.Type = a.determineFileType(filePath)

	analysis.Patterns = a.detectPatterns(filePath, diffContent)

	analysis.LinesAdded, analysis.LinesRemoved = a.countFileChanges(filePath, diffContent)

	return analysis
}

func (a *DiffAnalyzer) categorizeFile(filePath string) string {
	path := strings.ToLower(filePath)
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(path))

	if strings.Contains(base, "readme") || strings.Contains(base, "changelog") ||
		strings.Contains(base, "license") || ext == ".md" || ext == ".rst" ||
		strings.Contains(dir, "docs") || strings.Contains(dir, "documentation") {
		return "documentation"
	}

	if strings.Contains(path, "test") || strings.Contains(path, "spec") ||
		strings.Contains(base, "_test") || strings.Contains(base, ".test") ||
		strings.Contains(base, "_spec") || strings.Contains(base, ".spec") {
		return "tests"
	}

	if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" ||
		ext == ".ini" || ext == ".conf" || ext == ".config" ||
		base == "dockerfile" || base == "makefile" || base == ".gitignore" ||
		strings.Contains(base, "docker") || strings.Contains(base, "compose") {
		return "configuration"
	}

	if strings.Contains(path, "migration") || strings.Contains(path, "schema") ||
		strings.Contains(path, "database") || strings.Contains(path, "db") ||
		ext == ".sql" || strings.Contains(dir, "migrations") {
		return "database"
	}

	if ext == ".html" || ext == ".css" || ext == ".scss" || ext == ".sass" ||
		ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" ||
		ext == ".vue" || ext == ".svelte" || ext == ".angular" ||
		strings.Contains(dir, "frontend") || strings.Contains(dir, "web") ||
		strings.Contains(dir, "client") || strings.Contains(dir, "ui") ||
		strings.Contains(dir, "assets") || strings.Contains(dir, "public") {
		return "frontend"
	}

	if strings.Contains(path, "api") || strings.Contains(path, "endpoint") ||
		strings.Contains(path, "controller") || strings.Contains(path, "handler") ||
		strings.Contains(path, "route") || strings.Contains(path, "server") {
		return "api"
	}

	if ext == ".go" || ext == ".py" || ext == ".java" || ext == ".php" ||
		ext == ".rb" || ext == ".cs" || ext == ".cpp" || ext == ".c" ||
		ext == ".rs" || ext == ".kt" || ext == ".scala" {
		return "backend"
	}

	return "other"
}

func (a *DiffAnalyzer) determineFileType(filePath string) string {
	base := strings.ToLower(filepath.Base(filePath))

	if strings.Contains(base, "model") {
		return "model"
	} else if strings.Contains(base, "controller") {
		return "controller"
	} else if strings.Contains(base, "service") {
		return "service"
	} else if strings.Contains(base, "component") {
		return "component"
	} else if strings.Contains(base, "util") || strings.Contains(base, "helper") {
		return "utility"
	} else if strings.Contains(base, "config") {
		return "configuration"
	} else if strings.Contains(base, "test") {
		return "test"
	}

	return "source"
}

func (a *DiffAnalyzer) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go":
		return "Go"
	case ".py":
		return "Python"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".jsx", ".tsx":
		return "React"
	case ".vue":
		return "Vue"
	case ".java":
		return "Java"
	case ".php":
		return "PHP"
	case ".rb":
		return "Ruby"
	case ".cs":
		return "C#"
	case ".cpp", ".cc", ".cxx":
		return "C++"
	case ".c":
		return "C"
	case ".rs":
		return "Rust"
	case ".kt":
		return "Kotlin"
	case ".scala":
		return "Scala"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".scss":
		return "SCSS"
	case ".sql":
		return "SQL"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".xml":
		return "XML"
	case ".md":
		return "Markdown"
	default:
		return "Unknown"
	}
}

func (a *DiffAnalyzer) detectPatterns(filePath, diffContent string) []string {
	var patterns []string

	fileSection := a.extractFileSection(filePath, diffContent)
	if fileSection == "" {
		return patterns
	}

	section := strings.ToLower(fileSection)

	if strings.Contains(section, "func ") || strings.Contains(section, "def ") ||
		strings.Contains(section, "function ") {
		patterns = append(patterns, "function_definition")
	}

	if strings.Contains(section, "http") || strings.Contains(section, "rest") ||
		strings.Contains(section, "endpoint") || strings.Contains(section, "route") {
		patterns = append(patterns, "api_endpoint")
	}

	if strings.Contains(section, "select") || strings.Contains(section, "insert") ||
		strings.Contains(section, "update") || strings.Contains(section, "delete") ||
		strings.Contains(section, "create table") || strings.Contains(section, "alter table") {
		patterns = append(patterns, "database_query")
	}

	if strings.Contains(section, "error") || strings.Contains(section, "exception") ||
		strings.Contains(section, "try") || strings.Contains(section, "catch") {
		patterns = append(patterns, "error_handling")
	}

	if strings.Contains(section, "test") || strings.Contains(section, "assert") ||
		strings.Contains(section, "expect") || strings.Contains(section, "mock") {
		patterns = append(patterns, "testing")
	}

	if strings.Contains(section, "auth") || strings.Contains(section, "password") ||
		strings.Contains(section, "token") || strings.Contains(section, "security") {
		patterns = append(patterns, "security")
	}

	if strings.Contains(section, "cache") || strings.Contains(section, "optimize") ||
		strings.Contains(section, "performance") || strings.Contains(section, "async") {
		patterns = append(patterns, "performance")
	}

	return patterns
}

func (a *DiffAnalyzer) extractFileSection(filePath, diffContent string) string {
	lines := strings.Split(diffContent, "\n")
	var inFile bool
	var section []string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if strings.Contains(line, filePath) {
				inFile = true
				section = []string{line}
				continue
			} else {
				inFile = false
			}
		}

		if inFile {
			section = append(section, line)
			if strings.HasPrefix(line, "diff --git") && !strings.Contains(line, filePath) {
				break
			}
		}
	}

	return strings.Join(section, "\n")
}

func (a *DiffAnalyzer) countFileChanges(filePath, diffContent string) (int, int) {
	fileSection := a.extractFileSection(filePath, diffContent)
	lines := strings.Split(fileSection, "\n")

	var added, removed int
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed++
		}
	}

	return added, removed
}

func (a *DiffAnalyzer) analyzePatterns(diffContent string, analysis *DiffAnalysis) {
	content := strings.ToLower(diffContent)

	analysis.TestsIncluded = strings.Contains(content, "test") ||
		strings.Contains(content, "spec") || strings.Contains(content, "assert")

	analysis.DocsIncluded = strings.Contains(content, "readme") ||
		strings.Contains(content, ".md") || strings.Contains(content, "doc")

	analysis.ConfigChanges = strings.Contains(content, ".json") ||
		strings.Contains(content, ".yaml") || strings.Contains(content, ".yml") ||
		strings.Contains(content, "config") || strings.Contains(content, "dockerfile")
}

func (a *DiffAnalyzer) generateSummary(analysis *DiffAnalysis) string {
	var parts []string

	fileCount := len(analysis.FileChanges)
	parts = append(parts, fmt.Sprintf("%d file(s) modified", fileCount))

	if len(analysis.ChangeTypes) > 0 {
		parts = append(parts, fmt.Sprintf("affecting %s", strings.Join(analysis.ChangeTypes, ", ")))
	}

	if analysis.TestsIncluded {
		parts = append(parts, "includes tests")
	}

	if analysis.DocsIncluded {
		parts = append(parts, "includes documentation")
	}

	return strings.Join(parts, ", ")
}

func (a *DiffAnalyzer) determineComplexity(analysis *DiffAnalysis) string {
	score := 0

	score += len(analysis.FileChanges)
	score += len(analysis.ChangeTypes) * 2

	for _, fileAnalysis := range analysis.FileChanges {
		for _, pattern := range fileAnalysis.Patterns {
			switch pattern {
			case "database_query", "api_endpoint":
				score += 3
			case "security", "performance":
				score += 2
			case "error_handling", "testing":
				score += 1
			}
		}
	}

	if score <= 3 {
		return "low"
	} else if score <= 10 {
		return "medium"
	} else {
		return "high"
	}
}

func (a *DiffAnalyzer) determineImpact(analysis *DiffAnalysis) string {
	hasHighImpact := false
	hasMediumImpact := false

	for _, changeType := range analysis.ChangeTypes {
		switch changeType {
		case "database", "api":
			hasHighImpact = true
		case "backend", "frontend":
			hasMediumImpact = true
		}
	}

	for _, fileAnalysis := range analysis.FileChanges {
		for _, pattern := range fileAnalysis.Patterns {
			if pattern == "database_query" || pattern == "security" || pattern == "api_endpoint" {
				hasHighImpact = true
			}
		}
	}

	if hasHighImpact {
		return "high"
	} else if hasMediumImpact {
		return "medium"
	} else {
		return "low"
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
