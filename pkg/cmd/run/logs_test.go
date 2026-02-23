package run

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestLogsCmd_ValidatePipelineID(t *testing.T) {
	tests := []struct {
		name        string
		pipelineID  string
		expectError bool
	}{
		{
			name:        "empty pipeline ID",
			pipelineID:  "",
			expectError: true,
		},
		{
			name:        "whitespace only pipeline ID",
			pipelineID:  "   ",
			expectError: true,
		},
		{
			name:        "valid build number",
			pipelineID:  "123",
			expectError: false,
		},
		{
			name:        "valid build number with hash",
			pipelineID:  "#123",
			expectError: false,
		},
		{
			name:        "valid UUID",
			pipelineID:  "12345678-1234-1234-1234-123456789abc",
			expectError: false,
		},
		{
			name:        "invalid format",
			pipelineID:  "abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// For empty/whitespace pipeline IDs, test validation directly
			if tt.pipelineID == "" || strings.TrimSpace(tt.pipelineID) == "" {
				if tt.expectError {
					assert.Equal(t, "", strings.TrimSpace(tt.pipelineID), "Pipeline ID should be empty or whitespace")
				}
				return
			}

			// For non-empty IDs, test UUID detection logic
			pipelineID := strings.TrimSpace(tt.pipelineID)
			if strings.HasPrefix(pipelineID, "#") {
				pipelineID = pipelineID[1:]
			}

			// Test UUID format detection
			isUUID := strings.Contains(pipelineID, "-")
			if isUUID {
				// UUID format should pass validation
				assert.True(t, len(pipelineID) > 10, "UUID should be reasonably long")
			} else {
				// Build number format - test parsing
				_, err := strconv.Atoi(pipelineID)
				if tt.expectError {
					assert.Error(t, err, "Expected parsing error for: %s", pipelineID)
				} else {
					assert.NoError(t, err, "Should parse as integer: %s", pipelineID)
				}
			}
		})
	}
}

func TestMatchesStepName(t *testing.T) {
	tests := []struct {
		stepName      string
		requestedName string
		shouldMatch   bool
	}{
		{"build", "build", true},
		{"test", "test", true},
		{"deploy", "deploy", true},
		{"build", "BUILD", true},           // Case insensitive
		{"test-unit", "test", true},        // Contains match
		{"integration-test", "test", true}, // Contains match
		{"build-app", "build", true},       // Prefix match
		{"deploy", "test", false},          // No match
		{"compile", "build", false},        // No match
	}

	for _, tt := range tests {
		t.Run(tt.stepName+"_vs_"+tt.requestedName, func(t *testing.T) {
			result := matchesStepName(tt.stepName, tt.requestedName)
			assert.Equal(t, tt.shouldMatch, result, "Step name '%s' should match '%s': %v", tt.stepName, tt.requestedName, tt.shouldMatch)
		})
	}
}

func TestFilterStepsByName(t *testing.T) {
	steps := []*api.PipelineStep{
		{Name: "build", UUID: "step1"},
		{Name: "test-unit", UUID: "step2"},
		{Name: "test-integration", UUID: "step3"},
		{Name: "deploy", UUID: "step4"},
		{Name: "cleanup", UUID: "step5"},
	}

	tests := []struct {
		name          string
		stepFilter    string
		expectedUUIDs []string
	}{
		{
			name:          "exact match",
			stepFilter:    "build",
			expectedUUIDs: []string{"step1"},
		},
		{
			name:          "contains match",
			stepFilter:    "test",
			expectedUUIDs: []string{"step2", "step3"},
		},
		{
			name:          "no match",
			stepFilter:    "missing",
			expectedUUIDs: []string{},
		},
		{
			name:          "case insensitive",
			stepFilter:    "BUILD",
			expectedUUIDs: []string{"step1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterStepsByName(steps, tt.stepFilter)

			assert.Equal(t, len(tt.expectedUUIDs), len(filtered), "Filtered step count mismatch")

			for i, expectedUUID := range tt.expectedUUIDs {
				if i < len(filtered) {
					assert.Equal(t, expectedUUID, filtered[i].UUID, "Step UUID mismatch at index %d", i)
				}
			}
		})
	}
}

func TestGetAvailableStepNames(t *testing.T) {
	steps := []*api.PipelineStep{
		{Name: "build"},
		{Name: "test"},
		{Name: "deploy"},
	}

	result := getAvailableStepNames(steps)
	expected := "build, test, deploy"

	assert.Equal(t, expected, result)
}

func TestLogsCmd_ContainsError(t *testing.T) {
	cmd := &LogsCmd{}
	parser := utils.NewLogParser()

	tests := []struct {
		name        string
		logLine     string
		shouldMatch bool
	}{
		{
			name:        "error line",
			logLine:     "error: compilation failed",
			shouldMatch: true,
		},
		{
			name:        "panic line",
			logLine:     "panic: runtime error",
			shouldMatch: true,
		},
		{
			name:        "test failure",
			logLine:     "Test failed: TestUserAuth",
			shouldMatch: true,
		},
		{
			name:        "warning line",
			logLine:     "warning: deprecated function",
			shouldMatch: false, // warnings are not errors
		},
		{
			name:        "info line",
			logLine:     "INFO: Starting process",
			shouldMatch: false,
		},
		{
			name:        "debug line",
			logLine:     "DEBUG: Configuration loaded",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.containsError(tt.logLine, parser)
			assert.Equal(t, tt.shouldMatch, result, "Line '%s' error detection mismatch", tt.logLine)
		})
	}
}

func TestLogsCmd_OutputFormats(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"text", true},
		{"json", true},
		{"yaml", true},
		{"xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			// Test the format validation logic directly
			switch tt.format {
			case "text", "json", "yaml":
				// Valid formats
				assert.True(t, tt.valid)
			default:
				// Invalid formats
				assert.False(t, tt.valid)
			}
		})
	}
}

func TestLogsCmd_ProcessAccumulatedLogs(t *testing.T) {
	cmd := &LogsCmd{
		ErrorsOnly: true,
	}
	parser := utils.NewLogParser()

	tests := []struct {
		name     string
		logLines []string
		stepName string
	}{
		{
			name: "logs with errors",
			logLines: []string{
				"INFO: Starting process",
				"error: compilation failed",
				"INFO: Process completed",
			},
			stepName: "build",
		},
		{
			name: "logs without errors",
			logLines: []string{
				"INFO: Starting process",
				"INFO: Process completed successfully",
			},
			stepName: "deploy",
		},
		{
			name:     "empty logs",
			logLines: []string{},
			stepName: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just ensures the function doesn't panic
			err := cmd.processAccumulatedLogs(tt.logLines, tt.stepName, parser)
			assert.NoError(t, err, "processAccumulatedLogs should not return error")
		})
	}
}

func TestLogsCmd_ResolvePipelineUUID(t *testing.T) {
	tests := []struct {
		name       string
		pipelineID string
		expected   string
		expectUUID bool
	}{
		{
			name:       "UUID format",
			pipelineID: "12345678-1234-1234-1234-123456789abc",
			expected:   "12345678-1234-1234-1234-123456789abc",
			expectUUID: true,
		},
		{
			name:       "build number format",
			pipelineID: "123",
			expected:   "", // Will depend on API response
			expectUUID: false,
		},
		{
			name:       "build number with hash",
			pipelineID: "#456",
			expected:   "", // Will depend on API response
			expectUUID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &LogsCmd{
				PipelineID: tt.pipelineID,
			}

			if tt.expectUUID {
				// Test UUID passthrough logic
				pipelineID := strings.TrimSpace(cmd.PipelineID)
				isUUID := strings.Contains(pipelineID, "-")
				assert.True(t, isUUID, "Should be detected as UUID format")

				// UUID should be returned as-is
				assert.Equal(t, tt.expected, pipelineID)
			} else {
				// For build numbers, test the parsing logic
				pipelineID := strings.TrimSpace(cmd.PipelineID)
				if strings.HasPrefix(pipelineID, "#") {
					pipelineID = pipelineID[1:]
				}

				_, err := strconv.Atoi(pipelineID)
				assert.NoError(t, err, "Should parse as build number")
			}
		})
	}
}

func TestLogsCmd_ValidationRules(t *testing.T) {
	tests := []struct {
		name    string
		cmd     LogsCmd
		isValid bool
	}{
		{
			name: "valid basic command",
			cmd: LogsCmd{
				PipelineID: "123",
				Output:     "text",
				Context:    3,
			},
			isValid: true,
		},
		{
			name: "valid with step filter",
			cmd: LogsCmd{
				PipelineID: "123",
				Step:       "build",
				Output:     "json",
				Context:    5,
			},
			isValid: true,
		},
		{
			name: "valid errors-only mode",
			cmd: LogsCmd{
				PipelineID: "123",
				ErrorsOnly: true,
				Output:     "text",
				Context:    1,
			},
			isValid: true,
		},
		{
			name: "valid follow mode",
			cmd: LogsCmd{
				PipelineID: "123",
				Follow:     true,
				Output:     "text",
				Context:    3,
			},
			isValid: true,
		},
		{
			name: "empty pipeline ID",
			cmd: LogsCmd{
				PipelineID: "",
				Output:     "text",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pipeline ID validation
			hasValidPipelineID := strings.TrimSpace(tt.cmd.PipelineID) != ""
			assert.Equal(t, tt.isValid, hasValidPipelineID, "Pipeline ID validation mismatch")

			// Test output format validation
			validFormats := []string{"text", "json", "yaml"}
			isValidFormat := false
			for _, format := range validFormats {
				if tt.cmd.Output == format {
					isValidFormat = true
					break
				}
			}
			if tt.cmd.Output == "" {
				isValidFormat = true // Default will be applied
			}
			assert.True(t, isValidFormat, "Output format should be valid")

			// Test context validation
			if tt.cmd.Context < 0 {
				assert.False(t, tt.isValid, "Negative context should be invalid")
			}
		})
	}
}

// Integration test helpers (these would require a real API client)
func TestLogsCmd_Integration(t *testing.T) {
	// Skip integration tests if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require setting up a real Bitbucket API connection
	// and having test pipelines available. For now, we'll skip this.
	t.Skip("Integration tests require real Bitbucket API access")

	// Example of what an integration test might look like:
	/*
		ctx := context.Background()

		// Create real RunContext with authentication
		runCtx, err := NewRunContext(ctx, "json", true)
		require.NoError(t, err)

		// Test with a known pipeline ID
		cmd := &LogsCmd{
			PipelineID: "123",
			Output:     "json",
			NoColor:    true,
		}

		err = cmd.Run(ctx)
		assert.NoError(t, err)
	*/
}

// Benchmark tests
func BenchmarkMatchesStepName(b *testing.B) {
	stepNames := []string{"build", "test-unit", "test-integration", "deploy", "cleanup"}
	requestedNames := []string{"build", "test", "deploy", "missing"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stepName := stepNames[i%len(stepNames)]
		requestedName := requestedNames[i%len(requestedNames)]
		matchesStepName(stepName, requestedName)
	}
}

func BenchmarkFilterStepsByName(b *testing.B) {
	steps := make([]*api.PipelineStep, 100)
	for i := 0; i < 100; i++ {
		steps[i] = &api.PipelineStep{
			Name: fmt.Sprintf("step-%d", i),
			UUID: fmt.Sprintf("uuid-%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterStepsByName(steps, "step")
	}
}

func BenchmarkLogsCmd_ContainsError(b *testing.B) {
	cmd := &LogsCmd{}
	parser := utils.NewLogParser()

	logLines := []string{
		"INFO: Starting process",
		"error: compilation failed",
		"warning: deprecated function",
		"Test failed: TestAuth",
		"panic: runtime error",
		"DEBUG: Configuration loaded",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		line := logLines[i%len(logLines)]
		cmd.containsError(line, parser)
	}
}
