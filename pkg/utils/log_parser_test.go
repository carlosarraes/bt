package utils

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogParser(t *testing.T) {
	parser := NewLogParser()

	assert.NotNil(t, parser)
	assert.NotEmpty(t, parser.ErrorPatterns)
	assert.Equal(t, 3, parser.ContextLines)
	assert.False(t, parser.CaseSensitive)

	// Verify we have patterns for all major categories
	categories := parser.GetSupportedCategories()
	expectedCategories := []string{"build", "test", "runtime", "dependency", "docker"}

	for _, expected := range expectedCategories {
		assert.Contains(t, categories, expected, "Should have patterns for category: %s", expected)
	}
}

func TestLogParser_AnalyzeLog(t *testing.T) {
	tests := []struct {
		name               string
		logContent         string
		expectedErrors     int
		expectedCategories []string
	}{
		{
			name: "build errors",
			logContent: `
INFO: Starting build process
error: compilation failed in main.go:25
INFO: Cleaning up
build failed with exit code 1
`,
			expectedErrors:     2,
			expectedCategories: []string{"build"},
		},
		{
			name: "test failures",
			logContent: `
Running tests...
Test failed: TestUserLogin
AssertionError: Expected 200, got 401
FAIL: TestDatabase connection timeout
All tests completed
`,
			expectedErrors:     3,
			expectedCategories: []string{"test"},
		},
		{
			name: "mixed errors",
			logContent: `
Starting application...
panic: runtime error: nil pointer dereference
module not found: github.com/missing/pkg
docker: error during container build
fatal: connection refused
`,
			expectedErrors:     4,
			expectedCategories: []string{"runtime", "dependency", "docker"},
		},
		{
			name: "warnings and errors",
			logContent: `
warning: deprecated API usage
error: failed to compile
deprecation warning: function will be removed
fatal error: segmentation violation
`,
			expectedErrors:     2, // Only counting errors, not warnings
			expectedCategories: []string{"build", "runtime"},
		},
		{
			name: "no errors",
			logContent: `
INFO: Starting process
DEBUG: Configuration loaded
INFO: Process completed successfully
`,
			expectedErrors:     0,
			expectedCategories: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewLogParser()
			reader := strings.NewReader(tt.logContent)

			result, err := parser.AnalyzeLog(reader, "test-step")
			require.NoError(t, err)

			// Count only errors (not warnings)
			errorCount := 0
			for _, logError := range result.Errors {
				if logError.Severity == "error" || logError.Severity == "critical" {
					errorCount++
				}
			}

			assert.Equal(t, tt.expectedErrors, errorCount, "Error count mismatch")

			// Check categories are present
			foundCategories := make(map[string]bool)
			for _, logError := range result.Errors {
				if logError.Severity == "error" || logError.Severity == "critical" {
					foundCategories[logError.Category] = true
				}
			}

			for _, expectedCat := range tt.expectedCategories {
				assert.True(t, foundCategories[expectedCat], "Expected category %s not found", expectedCat)
			}
		})
	}
}

func TestLogParser_ErrorPatterns(t *testing.T) {
	parser := NewLogParser()

	tests := []struct {
		name             string
		logLine          string
		shouldMatch      bool
		expectedPattern  string
		expectedCategory string
		expectedSeverity string
	}{
		// Build errors
		{
			name:             "generic error",
			logLine:          "error: something generic went wrong",
			shouldMatch:      true,
			expectedPattern:  "generic_error",
			expectedCategory: "build",
			expectedSeverity: "error",
		},
		{
			name:             "compilation failure",
			logLine:          "compilation failed in main.go",
			shouldMatch:      true,
			expectedPattern:  "compilation_failure",
			expectedCategory: "build",
			expectedSeverity: "error",
		},
		{
			name:             "build failed",
			logLine:          "build failed with exit code 1",
			shouldMatch:      true,
			expectedPattern:  "build_failed",
			expectedCategory: "build",
			expectedSeverity: "error",
		},

		// Test errors
		{
			name:             "test failure",
			logLine:          "Test failed: TestUserAuth",
			shouldMatch:      true,
			expectedPattern:  "test_failure",
			expectedCategory: "test",
			expectedSeverity: "error",
		},
		{
			name:             "assertion error",
			logLine:          "AssertionError: Expected 200, got 404",
			shouldMatch:      true,
			expectedPattern:  "test_assertion",
			expectedCategory: "test",
			expectedSeverity: "error",
		},

		// Runtime errors
		{
			name:             "panic",
			logLine:          "panic: runtime error: nil pointer dereference",
			shouldMatch:      true,
			expectedPattern:  "panic",
			expectedCategory: "runtime",
			expectedSeverity: "critical",
		},
		{
			name:             "fatal error",
			logLine:          "fatal: could not connect to database",
			shouldMatch:      true,
			expectedPattern:  "fatal_error",
			expectedCategory: "runtime",
			expectedSeverity: "critical",
		},
		{
			name:             "segmentation fault",
			logLine:          "segmentation fault (core dumped)",
			shouldMatch:      true,
			expectedPattern:  "segmentation_fault",
			expectedCategory: "runtime",
			expectedSeverity: "critical",
		},

		// Dependency errors
		{
			name:             "module not found",
			logLine:          "module not found: github.com/missing/pkg",
			shouldMatch:      true,
			expectedPattern:  "module_not_found",
			expectedCategory: "dependency",
			expectedSeverity: "error",
		},
		{
			name:             "npm error",
			logLine:          "npm error: package not found",
			shouldMatch:      true,
			expectedPattern:  "npm_error",
			expectedCategory: "dependency",
			expectedSeverity: "error",
		},

		// Docker errors
		{
			name:             "docker error",
			logLine:          "docker: error during build",
			shouldMatch:      true,
			expectedPattern:  "docker_error",
			expectedCategory: "docker",
			expectedSeverity: "error",
		},
		{
			name:             "image not found",
			logLine:          "image not found: alpine:latest",
			shouldMatch:      true,
			expectedPattern:  "image_not_found",
			expectedCategory: "docker",
			expectedSeverity: "error",
		},

		// Warnings
		{
			name:             "deprecation warning",
			logLine:          "deprecation warning: function is deprecated",
			shouldMatch:      true,
			expectedPattern:  "deprecation_warning",
			expectedCategory: "build",
			expectedSeverity: "warning",
		},
		{
			name:             "generic warning",
			logLine:          "warning: unused variable 'x'",
			shouldMatch:      true,
			expectedPattern:  "generic_warning",
			expectedCategory: "build",
			expectedSeverity: "warning",
		},

		// Non-matching lines
		{
			name:        "info message",
			logLine:     "INFO: Starting application",
			shouldMatch: false,
		},
		{
			name:        "debug message",
			logLine:     "DEBUG: Configuration loaded",
			shouldMatch: false,
		},
		{
			name:        "success message",
			logLine:     "SUCCESS: Build completed",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.logLine)
			result, err := parser.AnalyzeLog(reader, "test-step")
			require.NoError(t, err)

			if tt.shouldMatch {
				assert.Greater(t, len(result.Errors), 0, "Expected to find at least one error")

				if len(result.Errors) > 0 {
					error := result.Errors[0]
					assert.Equal(t, tt.expectedPattern, error.Pattern, "Pattern mismatch")
					assert.Equal(t, tt.expectedCategory, error.Category, "Category mismatch")
					assert.Equal(t, tt.expectedSeverity, error.Severity, "Severity mismatch")
					assert.Equal(t, "test-step", error.StepName, "Step name mismatch")
					assert.Contains(t, error.Content, strings.TrimSpace(tt.logLine), "Content should contain original line")
				}
			} else {
				assert.Equal(t, 0, len(result.Errors), "Expected no errors for non-matching line")
			}
		})
	}
}

func TestLogParser_ContextExtraction(t *testing.T) {
	logContent := `
Line 1: Starting process
Line 2: Loading configuration
Line 3: error: failed to connect
Line 4: Retrying connection
Line 5: Process completed
`

	parser := NewLogParser()
	parser.SetContextLines(2)

	reader := strings.NewReader(logContent)
	result, err := parser.AnalyzeLog(reader, "test-step")
	require.NoError(t, err)

	assert.Greater(t, len(result.Errors), 0, "Should find at least one error")

	if len(result.Errors) > 0 {
		error := result.Errors[0]
		assert.Equal(t, 5, len(error.Context), "Should have 5 context lines (2 before + 1 error + 2 after)")

		// Check that the error line is marked with →
		errorLineFound := false
		for _, contextLine := range error.Context {
			if strings.HasPrefix(contextLine, "→ ") {
				assert.Contains(t, contextLine, "error: failed to connect")
				errorLineFound = true
				break
			}
		}
		assert.True(t, errorLineFound, "Error line should be marked with → in context")
	}
}

func TestLogParser_FilterErrorsOnly(t *testing.T) {
	logContent := `
warning: deprecated function used
error: compilation failed
deprecation warning: API will be removed
fatal: connection refused
info: process started
`

	parser := NewLogParser()
	reader := strings.NewReader(logContent)
	result, err := parser.AnalyzeLog(reader, "test-step")
	require.NoError(t, err)

	// Original result should have both errors and warnings
	assert.Greater(t, len(result.Errors), 0, "Should find errors and warnings")

	// Filter to errors only
	filtered := parser.FilterErrorsOnly(result)

	// Filtered result should have only errors
	errorCount := 0
	for _, logError := range filtered.Errors {
		assert.NotEqual(t, "warning", logError.Severity, "Filtered result should not contain warnings")
		if logError.Severity == "error" || logError.Severity == "critical" {
			errorCount++
		}
	}

	assert.Equal(t, 2, errorCount, "Should have exactly 2 errors after filtering")
	assert.Equal(t, 2, filtered.ErrorCount, "ErrorCount should match")
}

func TestLogParser_AddCustomPattern(t *testing.T) {
	parser := NewLogParser()
	initialPatternCount := len(parser.ErrorPatterns)

	// Add valid custom pattern (must be very specific to avoid conflicts)
	customPattern := ErrorPattern{
		Name:        "custom_test_error",
		Regex:       regexp.MustCompile(`CUSTOM_SPECIFIC_ERROR:\s*(.+)`),
		Category:    "custom",
		Severity:    "error",
		Description: "Custom test error pattern",
	}

	err := parser.AddCustomPattern(customPattern)
	assert.NoError(t, err)
	assert.Equal(t, initialPatternCount+1, len(parser.ErrorPatterns))

	// Test the custom pattern works
	reader := strings.NewReader("CUSTOM_SPECIFIC_ERROR: something went wrong")
	result, analyzeErr := parser.AnalyzeLog(reader, "test-step")
	require.NoError(t, analyzeErr)

	assert.Equal(t, 1, len(result.Errors), "Should find exactly one error")
	assert.Equal(t, "custom_test_error", result.Errors[0].Pattern)
	assert.Equal(t, "custom", result.Errors[0].Category)

	// Test invalid pattern
	invalidPattern := ErrorPattern{
		Name:  "", // Empty name should fail
		Regex: regexp.MustCompile("test"),
	}

	err = parser.AddCustomPattern(invalidPattern)
	assert.Error(t, err)
}

func TestLogParser_SetContextLines(t *testing.T) {
	parser := NewLogParser()

	// Test valid values
	parser.SetContextLines(5)
	assert.Equal(t, 5, parser.ContextLines)

	// Test negative value (should be set to 0)
	parser.SetContextLines(-1)
	assert.Equal(t, 0, parser.ContextLines)

	// Test too large value (should be capped at 10)
	parser.SetContextLines(15)
	assert.Equal(t, 10, parser.ContextLines)
}

func TestLogParser_GetPatternsByCategory(t *testing.T) {
	parser := NewLogParser()

	buildPatterns := parser.GetPatternsByCategory("build")
	assert.NotEmpty(t, buildPatterns)

	for _, pattern := range buildPatterns {
		assert.Equal(t, "build", pattern.Category)
	}

	// Test non-existent category
	nonExistentPatterns := parser.GetPatternsByCategory("nonexistent")
	assert.Empty(t, nonExistentPatterns)
}

func TestLogParser_GetSupportedCategories(t *testing.T) {
	parser := NewLogParser()
	categories := parser.GetSupportedCategories()

	expectedCategories := []string{"build", "test", "runtime", "dependency", "docker"}

	for _, expected := range expectedCategories {
		assert.Contains(t, categories, expected)
	}
}

func TestLogParser_LargeLogPerformance(t *testing.T) {
	// Create a large log with exactly 1000 lines
	var logBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		if i%100 == 0 {
			logBuilder.WriteString("error: test error on line " + string(rune(48+i/100))) // Use ASCII numbers
		} else {
			logBuilder.WriteString("INFO: regular log line " + string(rune(48+i%10))) // Use ASCII numbers
		}
		if i < 999 { // Add newline except for the very last line
			logBuilder.WriteString("\n")
		}
	}

	parser := NewLogParser()
	reader := strings.NewReader(logBuilder.String())

	result, err := parser.AnalyzeLog(reader, "performance-test")
	require.NoError(t, err)

	assert.Equal(t, 1000, result.TotalLines)
	assert.Greater(t, len(result.Errors), 5, "Should find multiple errors in large log")

	// Test should complete quickly (no explicit timing, but shouldn't hang)
}

// Benchmark tests
func BenchmarkLogParser_AnalyzeLog(b *testing.B) {
	logContent := `
INFO: Starting process
error: compilation failed
warning: deprecated function
Test failed: TestAuth
panic: runtime error
module not found: missing
docker: error during build
INFO: Process completed
`

	parser := NewLogParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(logContent)
		_, err := parser.AnalyzeLog(reader, "benchmark-step")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLogParser_PatternMatching(b *testing.B) {
	parser := NewLogParser()
	testLine := "error: compilation failed in main.go:25"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pattern := range parser.ErrorPatterns {
			parser.matchesPattern(testLine, pattern.Regex)
		}
	}
}
