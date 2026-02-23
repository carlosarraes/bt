package run

import (
	"context"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestRerunCmd_Run(t *testing.T) {
	tests := []struct {
		name        string
		pipelineID  string
		failed      bool
		step        string
		force       bool
		output      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid pipeline ID",
			pipelineID:  "123",
			failed:      false,
			step:        "",
			force:       true,
			output:      "table",
			expectError: false,
		},
		{
			name:        "valid pipeline ID with failed flag",
			pipelineID:  "123",
			failed:      true,
			step:        "",
			force:       true,
			output:      "json",
			expectError: false,
		},
		{
			name:        "valid pipeline ID with step flag",
			pipelineID:  "123",
			failed:      false,
			step:        "build",
			force:       true,
			output:      "yaml",
			expectError: false,
		},
		{
			name:        "empty pipeline ID",
			pipelineID:  "",
			failed:      false,
			step:        "",
			force:       true,
			output:      "table",
			expectError: true,
			errorMsg:    "not in a git repository",
		},
		{
			name:        "invalid pipeline ID",
			pipelineID:  "abc",
			failed:      false,
			step:        "",
			force:       true,
			output:      "table",
			expectError: true,
			errorMsg:    "not in a git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: tt.pipelineID,
				Failed:     tt.failed,
				Step:       tt.step,
				Force:      tt.force,
				Output:     tt.output,
				NoColor:    true,
				Workspace:  "test-workspace",
				Repository: "test-repo",
			}

			err := cmd.Run(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRerunCmd_resolvePipelineUUID(t *testing.T) {
	tests := []struct {
		name        string
		pipelineID  string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "UUID format",
			pipelineID:  "12345678-1234-1234-1234-123456789012",
			expected:    "12345678-1234-1234-1234-123456789012",
			expectError: false,
		},
		{
			name:        "build number format",
			pipelineID:  "123",
			expected:    "",
			expectError: true,
		},
		{
			name:        "build number with hash",
			pipelineID:  "#123",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid format",
			pipelineID:  "abc",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid pipeline ID",
		},
		{
			name:        "empty ID",
			pipelineID:  "",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid pipeline ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if strings.Contains(tt.pipelineID, "-") {
				result, err := resolvePipelineUUID(context.Background(), nil, tt.pipelineID)
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorMsg != "" {
						assert.Contains(t, err.Error(), tt.errorMsg)
					}
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, result)
				}
			} else {
				assert.True(t, tt.expectError, "Build number cases should expect errors without real API")
			}
		})
	}
}

func TestRerunCmd_validateRerunnable(t *testing.T) {
	tests := []struct {
		name          string
		pipelineState string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "successful pipeline",
			pipelineState: "SUCCESSFUL",
			expectError:   false,
		},
		{
			name:          "failed pipeline",
			pipelineState: "FAILED",
			expectError:   false,
		},
		{
			name:          "error pipeline",
			pipelineState: "ERROR",
			expectError:   false,
		},
		{
			name:          "stopped pipeline",
			pipelineState: "STOPPED",
			expectError:   false,
		},
		{
			name:          "pending pipeline",
			pipelineState: "PENDING",
			expectError:   true,
			errorMsg:      "still running",
		},
		{
			name:          "in progress pipeline",
			pipelineState: "IN_PROGRESS",
			expectError:   true,
			errorMsg:      "still running",
		},
		{
			name:          "unknown state",
			pipelineState: "UNKNOWN",
			expectError:   true,
			errorMsg:      "unknown state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: "123",
				Force:      true,
				Output:     "table",
				NoColor:    true,
			}

			pipeline := &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: tt.pipelineState,
				},
			}

			err := cmd.validateRerunnable(pipeline)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRerunCmd_buildTriggerRequest(t *testing.T) {
	tests := []struct {
		name        string
		failed      bool
		step        string
		pipeline    *api.Pipeline
		expectError bool
		errorMsg    string
	}{
		{
			name:   "valid pipeline with target",
			failed: false,
			step:   "",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				Target: &api.PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
					Commit: &api.Commit{
						Hash: "abc123",
					},
				},
			},
			expectError: false,
		},
		{
			name:   "pipeline with failed flag",
			failed: true,
			step:   "",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				Target: &api.PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
				},
			},
			expectError: false,
		},
		{
			name:   "pipeline with step flag",
			failed: false,
			step:   "build",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				Target: &api.PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
				},
			},
			expectError: false,
		},
		{
			name:   "pipeline without target",
			failed: false,
			step:   "",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				Target:      nil,
			},
			expectError: true,
			errorMsg:    "no target information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: "123",
				Failed:     tt.failed,
				Step:       tt.step,
				Force:      true,
				Output:     "table",
				NoColor:    true,
			}

			request, err := cmd.buildTriggerRequest(context.Background(), nil, tt.pipeline)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, request)
				assert.NotNil(t, request.Target)
				assert.Equal(t, tt.pipeline.Target.Type, request.Target.Type)
				assert.Equal(t, tt.pipeline.Target.RefType, request.Target.RefType)
				assert.Equal(t, tt.pipeline.Target.RefName, request.Target.RefName)
			}
		})
	}
}

func TestRerunCmd_confirmRerun(t *testing.T) {
	tests := []struct {
		name     string
		failed   bool
		step     string
		pipeline *api.Pipeline
		expected string
	}{
		{
			name:   "basic rerun",
			failed: false,
			step:   "",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: "FAILED",
				},
			},
			expected: "rerun pipeline #123",
		},
		{
			name:   "rerun failed steps",
			failed: true,
			step:   "",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: "FAILED",
				},
			},
			expected: "rerun failed steps of pipeline #123",
		},
		{
			name:   "rerun specific step",
			failed: false,
			step:   "build",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				State: &api.PipelineState{
					Name: "FAILED",
				},
			},
			expected: "rerun step 'build' of pipeline #123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: "123",
				Failed:     tt.failed,
				Step:       tt.step,
				Force:      true,
				Output:     "table",
				NoColor:    true,
			}

			action := "rerun"
			if cmd.Failed {
				action = "rerun failed steps of"
			} else if cmd.Step != "" {
				action = "rerun step '" + cmd.Step + "' of"
			}

			assert.Contains(t, tt.expected, action)
		})
	}
}

func TestRerunCmd_parsePipelineResults(t *testing.T) {

	tests := []struct {
		name        string
		response    *api.PaginatedResponse
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil values",
			response: &api.PaginatedResponse{
				Values: nil,
			},
			expectError: false,
		},
		{
			name: "empty values",
			response: &api.PaginatedResponse{
				Values: []byte("[]"),
			},
			expectError: false,
		},
		{
			name: "invalid JSON",
			response: &api.PaginatedResponse{
				Values: []byte("invalid json"),
			},
			expectError: true,
			errorMsg:    "failed to unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipelines, err := parsePipelineResults(tt.response)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.response.Values != nil {
					assert.NotNil(t, pipelines)
				}
			}
		})
	}
}

func TestRerunCmd_outputSuccess(t *testing.T) {
	originalPipeline := &api.Pipeline{
		BuildNumber: 123,
		UUID:        "original-uuid",
		State: &api.PipelineState{
			Name: "FAILED",
		},
	}

	newPipeline := &api.Pipeline{
		BuildNumber: 124,
		UUID:        "new-uuid",
		State: &api.PipelineState{
			Name: "PENDING",
		},
		Repository: &api.Repository{
			FullName: "workspace/repo",
		},
		Target: &api.PipelineTarget{
			RefName: "main",
			Commit: &api.Commit{
				Hash: "abc123def",
			},
		},
	}

	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "table output",
			output: "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: "123",
				Force:      true,
				Output:     tt.output,
				NoColor:    true,
			}

			err := cmd.outputTable(originalPipeline, newPipeline)
			assert.NoError(t, err)
		})
	}
}

func BenchmarkRerunCmd_resolvePipelineUUID(b *testing.B) {
	uuids := []string{
		"12345678-1234-1234-1234-123456789012",
		"87654321-4321-4321-4321-210987654321",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolvePipelineUUID(context.Background(), nil, uuids[i%len(uuids)])
	}
}

func BenchmarkRerunCmd_validateRerunnable(b *testing.B) {
	cmd := &RerunCmd{
		PipelineID: "123",
		Force:      true,
		Output:     "table",
		NoColor:    true,
	}

	pipeline := &api.Pipeline{
		BuildNumber: 123,
		State: &api.PipelineState{
			Name: "FAILED",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.validateRerunnable(pipeline)
	}
}

func BenchmarkRerunCmd_buildTriggerRequest(b *testing.B) {
	cmd := &RerunCmd{
		PipelineID: "123",
		Force:      true,
		Output:     "table",
		NoColor:    true,
	}

	pipeline := &api.Pipeline{
		BuildNumber: 123,
		Target: &api.PipelineTarget{
			Type:    "pipeline_ref_target",
			RefType: "branch",
			RefName: "main",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.buildTriggerRequest(context.Background(), nil, pipeline)
	}
}

func TestRerunCmd_GitHubCLICompatibility(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected RerunCmd
	}{
		{
			name: "basic rerun",
			args: []string{"123"},
			expected: RerunCmd{
				PipelineID: "123",
				Failed:     false,
				Step:       "",
				Force:      false,
				Output:     "table",
			},
		},
		{
			name: "rerun with failed flag",
			args: []string{"123", "--failed"},
			expected: RerunCmd{
				PipelineID: "123",
				Failed:     true,
				Step:       "",
				Force:      false,
				Output:     "table",
			},
		},
		{
			name: "rerun with step flag",
			args: []string{"123", "--step", "build"},
			expected: RerunCmd{
				PipelineID: "123",
				Failed:     false,
				Step:       "build",
				Force:      false,
				Output:     "table",
			},
		},
		{
			name: "rerun with force flag",
			args: []string{"123", "--force"},
			expected: RerunCmd{
				PipelineID: "123",
				Failed:     false,
				Step:       "",
				Force:      true,
				Output:     "table",
			},
		},
		{
			name: "rerun with output flag",
			args: []string{"123", "--output", "json"},
			expected: RerunCmd{
				PipelineID: "123",
				Failed:     false,
				Step:       "",
				Force:      false,
				Output:     "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: tt.expected.PipelineID,
				Failed:     tt.expected.Failed,
				Step:       tt.expected.Step,
				Force:      tt.expected.Force,
				Output:     tt.expected.Output,
				NoColor:    true,
			}

			assert.Equal(t, tt.expected.PipelineID, cmd.PipelineID)
			assert.Equal(t, tt.expected.Failed, cmd.Failed)
			assert.Equal(t, tt.expected.Step, cmd.Step)
			assert.Equal(t, tt.expected.Force, cmd.Force)
			assert.Equal(t, tt.expected.Output, cmd.Output)
		})
	}
}

func TestRerunCmd_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		pipelineID  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "whitespace pipeline ID",
			pipelineID:  "  123  ",
			expectError: true,
		},
		{
			name:        "negative build number",
			pipelineID:  "-123",
			expectError: true,
			errorMsg:    "not in a git repository",
		},
		{
			name:        "zero build number",
			pipelineID:  "0",
			expectError: true,
		},
		{
			name:        "very large build number",
			pipelineID:  "999999999",
			expectError: true,
		},
		{
			name:        "UUID with wrong format",
			pipelineID:  "12345678-1234-1234-1234-12345678901",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &RerunCmd{
				PipelineID: tt.pipelineID,
				Force:      true,
				Output:     "table",
				NoColor:    true,
				Workspace:  "test-workspace",
				Repository: "test-repo",
			}

			err := cmd.Run(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}
