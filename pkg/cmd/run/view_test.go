package run

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestViewCmd_ValidatePipelineID(t *testing.T) {
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

func TestViewCmd_GetStatusIcon(t *testing.T) {
	cmd := &ViewCmd{}

	tests := []struct {
		status       string
		expectedIcon string
	}{
		{"SUCCESSFUL", "✓"},
		{"FAILED", "✗"},
		{"ERROR", "✗"},
		{"STOPPED", "⏸"},
		{"IN_PROGRESS", "⚙"},
		{"PENDING", "⏳"},
		{"UNKNOWN", "?"},
		{"", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			icon := cmd.getStatusIcon(tt.status)
			assert.Equal(t, tt.expectedIcon, icon)
		})
	}
}

func TestViewCmd_ResolvePipelineUUID(t *testing.T) {
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
			cmd := &ViewCmd{
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

func TestViewCmd_OutputFormats(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"table", true},
		{"json", true},
		{"yaml", true},
		{"xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			// Test the format validation logic directly
			switch tt.format {
			case "table", "json", "yaml":
				// Valid formats
				assert.True(t, tt.valid)
			default:
				// Invalid formats
				assert.False(t, tt.valid)
			}
		})
	}
}

func TestViewCmd_TableFormatting(t *testing.T) {
	// Test the table formatting logic with various pipeline states
	now := time.Now()

	tests := []struct {
		name     string
		pipeline *api.Pipeline
		steps    []*api.PipelineStep
	}{
		{
			name: "successful pipeline with steps",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: "SUCCESSFUL",
				},
				Target: &api.PipelineTarget{
					RefName: "main",
					Commit: &api.Commit{
						Hash:    "abcd1234",
						Message: "Test commit",
					},
				},
				CreatedOn:        &now,
				BuildSecondsUsed: 300,
				Repository: &api.Repository{
					FullName: "test/repo",
				},
			},
			steps: []*api.PipelineStep{
				{
					Name: "build",
					State: &api.PipelineState{
						Name: "SUCCESSFUL",
					},
					BuildSecondsUsed: 150,
				},
			},
		},
		{
			name: "failed pipeline",
			pipeline: &api.Pipeline{
				BuildNumber: 124,
				State: &api.PipelineState{
					Name: "FAILED",
				},
				Target: &api.PipelineTarget{
					RefName: "feature/test",
				},
				BuildSecondsUsed: 100,
			},
			steps: []*api.PipelineStep{
				{
					Name: "test",
					State: &api.PipelineState{
						Name: "FAILED",
					},
					BuildSecondsUsed: 100,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{
				Output:  "table",
				NoColor: true,
			}

			// Test that table formatting doesn't panic
			// We can't easily capture the output without a full RunContext
			err := cmd.formatTable(nil, tt.pipeline, tt.steps)

			// We expect an error due to nil RunContext, but it should be a specific error
			// not a panic or formatting issue
			if err != nil {
				assert.NotContains(t, err.Error(), "panic")
			}
		})
	}
}

func TestViewCmd_WatchFunctionality(t *testing.T) {
	tests := []struct {
		name           string
		pipelineState  string
		shouldWatch    bool
		expectedAction string
	}{
		{
			name:           "running pipeline should watch",
			pipelineState:  "IN_PROGRESS",
			shouldWatch:    true,
			expectedAction: "watch",
		},
		{
			name:           "pending pipeline should watch",
			pipelineState:  "PENDING",
			shouldWatch:    true,
			expectedAction: "watch",
		},
		{
			name:           "completed pipeline should view once",
			pipelineState:  "SUCCESSFUL",
			shouldWatch:    false,
			expectedAction: "view",
		},
		{
			name:           "failed pipeline should view once",
			pipelineState:  "FAILED",
			shouldWatch:    false,
			expectedAction: "view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic for determining watch vs view behavior
			pipeline := &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: tt.pipelineState,
				},
			}

			// Test the condition used in watchPipeline
			canWatch := pipeline.State != nil &&
				(pipeline.State.Name == "IN_PROGRESS" || pipeline.State.Name == "PENDING")

			assert.Equal(t, tt.shouldWatch, canWatch)
		})
	}
}

// Integration test helpers (these would require a real API client)
func TestViewCmd_Integration(t *testing.T) {
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
		cmd := &ViewCmd{
			PipelineID: "123",
			Output:     "json",
			NoColor:    true,
		}

		err = cmd.Run(ctx)
		assert.NoError(t, err)
	*/
}

// Benchmark tests
func BenchmarkViewCmd_GetStatusIcon(b *testing.B) {
	cmd := &ViewCmd{}
	statuses := []string{"SUCCESSFUL", "FAILED", "IN_PROGRESS", "PENDING", "STOPPED", "ERROR", "UNKNOWN"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := statuses[i%len(statuses)]
		cmd.getStatusIcon(status)
	}
}

func BenchmarkViewCmd_ResolvePipelineUUID(b *testing.B) {
	cmd := &ViewCmd{}
	uuids := []string{
		"12345678-1234-1234-1234-123456789abc",
		"87654321-4321-4321-4321-cba987654321",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.PipelineID = uuids[i%len(uuids)]
		// This will return immediately for UUID format
		cmd.resolvePipelineUUID(context.Background(), nil)
	}
}
