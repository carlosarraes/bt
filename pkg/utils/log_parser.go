package utils

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// LogParser provides advanced log analysis and error extraction capabilities
type LogParser struct {
	ErrorPatterns []ErrorPattern
	ContextLines  int
	CaseSensitive bool
}

// ErrorPattern defines a pattern for detecting specific types of errors in logs
type ErrorPattern struct {
	Name        string         // Human-readable name for the pattern
	Regex       *regexp.Regexp // Compiled regular expression
	Category    string         // Error category: "build", "test", "runtime", "dependency", "docker"
	Severity    string         // Severity level: "error", "warning", "critical", "info"
	Description string         // Description of what this pattern detects
	Examples    []string       // Example log lines that match this pattern
}

// ExtractedError represents an error found in log output
type ExtractedError struct {
	Line        int       `json:"line"`        // Line number where error was found
	Content     string    `json:"content"`     // The actual error line content
	Pattern     string    `json:"pattern"`     // Name of the pattern that matched
	Category    string    `json:"category"`    // Error category
	Severity    string    `json:"severity"`    // Error severity
	Context     []string  `json:"context"`     // Surrounding lines for context
	Timestamp   time.Time `json:"timestamp"`   // When the error was extracted
	StepName    string    `json:"step_name"`   // Pipeline step where error occurred
}

// LogAnalysisResult contains the complete analysis of a log stream
type LogAnalysisResult struct {
	TotalLines   int              `json:"total_lines"`
	ErrorCount   int              `json:"error_count"`
	WarningCount int              `json:"warning_count"`
	Errors       []ExtractedError `json:"errors"`
	Summary      map[string]int   `json:"summary"` // Error count by category
	ProcessedAt  time.Time        `json:"processed_at"`
}

// NewLogParser creates a new log parser with default error patterns
func NewLogParser() *LogParser {
	return &LogParser{
		ErrorPatterns: GetDefaultErrorPatterns(),
		ContextLines:  3,
		CaseSensitive: false,
	}
}

// GetDefaultErrorPatterns returns built-in error patterns for common scenarios
// Note: Order matters! More specific patterns should come before generic ones
func GetDefaultErrorPatterns() []ErrorPattern {
	patterns := []ErrorPattern{
		// Runtime Errors (Critical - should come first)
		{
			Name:        "panic",
			Regex:       regexp.MustCompile(`(?i)(panic\s*:|runtime panic|panic.*occurred)`),
			Category:    "runtime",
			Severity:    "critical",
			Description: "Runtime panics and crashes",
			Examples:    []string{"panic: runtime error", "panic occurred: nil pointer"},
		},
		{
			Name:        "fatal_error",
			Regex:       regexp.MustCompile(`(?i)(fatal\s*:|fatal error|fatal.*occurred)`),
			Category:    "runtime",
			Severity:    "critical",
			Description: "Fatal runtime errors",
			Examples:    []string{"fatal: could not connect", "fatal error: segmentation violation"},
		},
		{
			Name:        "segmentation_fault",
			Regex:       regexp.MustCompile(`(?i)(segmentation fault|segfault|sigsegv|signal.*11)`),
			Category:    "runtime",
			Severity:    "critical",
			Description: "Memory segmentation violations",
			Examples:    []string{"segmentation fault", "SIGSEGV: segmentation violation"},
		},

		// Build Errors (Specific patterns first)
		{
			Name:        "compilation_failure",
			Regex:       regexp.MustCompile(`(?i)(compilation failed|compile error|failed to compile|compilation error)`),
			Category:    "build",
			Severity:    "error",
			Description: "Code compilation failures",
			Examples:    []string{"compilation failed", "compile error in main.go"},
		},
		{
			Name:        "build_failed",
			Regex:       regexp.MustCompile(`(?i)(build failed|build error|failed to build)`),
			Category:    "build",
			Severity:    "error",
			Description: "General build process failures",
			Examples:    []string{"build failed", "Build error: missing dependency"},
		},

		// Test Errors (Specific patterns first)
		{
			Name:        "test_assertion",
			Regex:       regexp.MustCompile(`(?i)(assertion\s*error|assert.*failed|expected.*got|should.*but)`),
			Category:    "test",
			Severity:    "error",
			Description: "Test assertion failures",
			Examples:    []string{"AssertionError: Expected 200, got 404", "should be true but was false"},
		},
		{
			Name:        "test_failure",
			Regex:       regexp.MustCompile(`(?i)(test failed|fail:|failed:|test.*failed|expect.*failed)`),
			Category:    "test",
			Severity:    "error",
			Description: "Test execution failures",
			Examples:    []string{"Test failed: TestLogin", "FAIL: TestUserAuth"},
		},
		{
			Name:        "test_timeout",
			Regex:       regexp.MustCompile(`(?i)(test.*timeout|timeout.*test|test.*timed out)`),
			Category:    "test",
			Severity:    "error",
			Description: "Test execution timeouts",
			Examples:    []string{"Test timeout after 30s", "test timed out"},
		},

		// Dependency Errors (Specific patterns first)
		{
			Name:        "npm_error",
			Regex:       regexp.MustCompile(`(?i)(npm error|npm.*failed|yarn error)`),
			Category:    "dependency",
			Severity:    "error",
			Description: "Node.js package manager errors",
			Examples:    []string{"npm error: ENOENT", "yarn error: package not found"},
		},
		{
			Name:        "module_not_found",
			Regex:       regexp.MustCompile(`(?i)(module not found|package not found|cannot find module|no module named)`),
			Category:    "dependency",
			Severity:    "error",
			Description: "Missing modules or packages",
			Examples:    []string{"module not found: github.com/missing/pkg", "No module named 'requests'"},
		},
		{
			Name:        "dependency_error",
			Regex:       regexp.MustCompile(`(?i)(dependency.*error|failed to.*dependency|missing dependency)`),
			Category:    "dependency",
			Severity:    "error",
			Description: "General dependency resolution failures",
			Examples:    []string{"dependency error: version conflict", "failed to resolve dependency"},
		},

		// Docker Errors (Specific patterns first)
		{
			Name:        "image_not_found",
			Regex:       regexp.MustCompile(`(?i)(image not found|no such image|pull.*failed|image.*does not exist)`),
			Category:    "docker",
			Severity:    "error",
			Description: "Docker image resolution failures",
			Examples:    []string{"image not found: alpine:latest", "pull failed: no such image"},
		},
		{
			Name:        "dockerfile_error",
			Regex:       regexp.MustCompile(`(?i)(dockerfile.*error|failed.*dockerfile|invalid.*dockerfile)`),
			Category:    "docker",
			Severity:    "error",
			Description: "Dockerfile syntax or execution errors",
			Examples:    []string{"Dockerfile error: COPY failed", "invalid Dockerfile instruction"},
		},
		{
			Name:        "docker_error",
			Regex:       regexp.MustCompile(`(?i)(docker\s*:\s*error|docker.*failed|container.*failed)`),
			Category:    "docker",
			Severity:    "error",
			Description: "Docker container and image errors",
			Examples:    []string{"docker: error during build", "container failed to start"},
		},

		// Additional Runtime Errors
		{
			Name:        "exit_code",
			Regex:       regexp.MustCompile(`(?i)(exit code\s*\d+|exited with.*\d+|process.*exit.*[1-9]\d*)`),
			Category:    "runtime",
			Severity:    "error",
			Description: "Non-zero exit codes",
			Examples:    []string{"exit code 1", "process exited with code 2"},
		},

		// Language-Specific Errors
		{
			Name:        "go_error",
			Regex:       regexp.MustCompile(`(?i)(go\s*:\s*.*\.go:\d+:\d+|cannot find package.*go|go build.*failed)`),
			Category:    "build",
			Severity:    "error",
			Description: "Go language specific errors",
			Examples:    []string{"go: main.go:25:10: undefined variable", "go build failed"},
		},
		{
			Name:        "python_error",
			Regex:       regexp.MustCompile(`(?i)(python.*error|traceback.*recent|syntaxerror|importerror|modulenotfounderror)`),
			Category:    "runtime",
			Severity:    "error",
			Description: "Python language specific errors",
			Examples:    []string{"Python error: SyntaxError", "Traceback (most recent call last)"},
		},
		{
			Name:        "java_error",
			Regex:       regexp.MustCompile(`(?i)(java.*exception|exception in thread|java.*error|compilation error.*java)`),
			Category:    "runtime",
			Severity:    "error",
			Description: "Java language specific errors",
			Examples:    []string{"java.lang.Exception", "Exception in thread main"},
		},

		// Network and API Errors
		{
			Name:        "connection_error",
			Regex:       regexp.MustCompile(`(?i)(connection.*failed|failed to connect|connection.*refused|timeout.*connect)`),
			Category:    "runtime",
			Severity:    "error",
			Description: "Network connection failures",
			Examples:    []string{"connection failed: timeout", "failed to connect to database"},
		},
		{
			Name:        "http_error",
			Regex:       regexp.MustCompile(`(?i)(http.*error|status.*[4-5]\d\d|request.*failed|api.*error)`),
			Category:    "runtime",
			Severity:    "error",
			Description: "HTTP request and API errors",
			Examples:    []string{"HTTP error 404", "API request failed: 500 Internal Server Error"},
		},

		// Warning Patterns (Specific first)
		{
			Name:        "deprecation_warning",
			Regex:       regexp.MustCompile(`(?i)(deprecat.*warning|deprecated|deprecation)`),
			Category:    "build",
			Severity:    "warning",
			Description: "Deprecation warnings",
			Examples:    []string{"deprecation warning: function is deprecated", "deprecated API usage"},
		},
		{
			Name:        "generic_warning",
			Regex:       regexp.MustCompile(`(?i)(warning\s*:|warn\s*:|\bwarning\b)`),
			Category:    "build",
			Severity:    "warning",
			Description: "Generic warning messages",
			Examples:    []string{"warning: unused variable", "warn: configuration not found"},
		},

		// Generic Error Pattern (MUST BE LAST - catches any remaining "error:" patterns)
		{
			Name:        "generic_error",
			Regex:       regexp.MustCompile(`(?i)error\s*:\s*(.+)`),
			Category:    "build",
			Severity:    "error",
			Description: "Generic error messages",
			Examples:    []string{"error: compilation failed", "Error: build process terminated"},
		},
	}

	return patterns
}

// AnalyzeLog processes a log stream and extracts errors according to configured patterns
func (lp *LogParser) AnalyzeLog(reader io.Reader, stepName string) (*LogAnalysisResult, error) {
	result := &LogAnalysisResult{
		Errors:      make([]ExtractedError, 0),
		Summary:     make(map[string]int),
		ProcessedAt: time.Now(),
	}

	scanner := bufio.NewScanner(reader)
	lines := make([]string, 0)
	lineNumber := 0

	// Read all lines first to enable context extraction
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result.TotalLines = len(lines)

	// Process each line for error patterns
	for i, line := range lines {
		lineNumber = i + 1
		
		for _, pattern := range lp.ErrorPatterns {
			if lp.matchesPattern(line, pattern.Regex) {
				// Extract context lines around the error
				context := lp.extractContext(lines, i, lp.ContextLines)
				
				error := ExtractedError{
					Line:        lineNumber,
					Content:     strings.TrimSpace(line),
					Pattern:     pattern.Name,
					Category:    pattern.Category,
					Severity:    pattern.Severity,
					Context:     context,
					Timestamp:   time.Now(),
					StepName:    stepName,
				}

				result.Errors = append(result.Errors, error)
				result.Summary[pattern.Category]++

				// Count by severity
				if pattern.Severity == "error" || pattern.Severity == "critical" {
					result.ErrorCount++
				} else if pattern.Severity == "warning" {
					result.WarningCount++
				}

				break // Only match first pattern per line
			}
		}
	}

	return result, nil
}

// FilterErrorsOnly returns only errors from the analysis result, excluding warnings
func (lp *LogParser) FilterErrorsOnly(result *LogAnalysisResult) *LogAnalysisResult {
	filtered := &LogAnalysisResult{
		TotalLines:  result.TotalLines,
		Summary:     make(map[string]int),
		ProcessedAt: result.ProcessedAt,
	}

	for _, err := range result.Errors {
		if err.Severity == "error" || err.Severity == "critical" {
			filtered.Errors = append(filtered.Errors, err)
			filtered.ErrorCount++
			filtered.Summary[err.Category]++
		}
	}

	return filtered
}

// matchesPattern checks if a line matches the given pattern
func (lp *LogParser) matchesPattern(line string, pattern *regexp.Regexp) bool {
	if !lp.CaseSensitive {
		return pattern.MatchString(line)
	}
	return pattern.MatchString(line)
}

// extractContext extracts surrounding lines around the target line for context
func (lp *LogParser) extractContext(lines []string, targetIndex, contextLines int) []string {
	start := targetIndex - contextLines
	end := targetIndex + contextLines + 1

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	context := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == targetIndex {
			prefix = "→ " // Mark the actual error line
		}
		context = append(context, prefix+lines[i])
	}

	return context
}

// AddCustomPattern allows adding custom error patterns at runtime
// Custom patterns are inserted at the beginning to give them priority over built-in patterns
func (lp *LogParser) AddCustomPattern(pattern ErrorPattern) error {
	// Validate the pattern
	if pattern.Name == "" {
		return fmt.Errorf("pattern name cannot be empty")
	}
	if pattern.Regex == nil {
		return fmt.Errorf("pattern regex cannot be nil")
	}
	if pattern.Category == "" {
		pattern.Category = "custom"
	}
	if pattern.Severity == "" {
		pattern.Severity = "error"
	}

	// Insert at the beginning to give custom patterns priority
	lp.ErrorPatterns = append([]ErrorPattern{pattern}, lp.ErrorPatterns...)
	return nil
}

// SetContextLines configures how many lines of context to include around errors
func (lp *LogParser) SetContextLines(lines int) {
	if lines < 0 {
		lines = 0
	}
	if lines > 10 {
		lines = 10 // Reasonable maximum
	}
	lp.ContextLines = lines
}

// GetPatternsByCategory returns all patterns for a specific category
func (lp *LogParser) GetPatternsByCategory(category string) []ErrorPattern {
	var patterns []ErrorPattern
	for _, pattern := range lp.ErrorPatterns {
		if pattern.Category == category {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

// GetSupportedCategories returns all supported error categories
func (lp *LogParser) GetSupportedCategories() []string {
	categories := make(map[string]bool)
	for _, pattern := range lp.ErrorPatterns {
		categories[pattern.Category] = true
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}
	return result
}