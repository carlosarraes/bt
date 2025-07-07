package pr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestChecksCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name     string
		prid     string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid integer",
			prid:     "123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "valid integer with hash prefix",
			prid:     "#123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "empty string",
			prid:     "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid format",
			prid:     "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "zero value",
			prid:     "0",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "negative value",
			prid:     "-1",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ChecksCmd{PRID: tt.prid}
			result, err := cmd.ParsePRID()

			if tt.wantErr && err == nil {
				t.Errorf("ParsePRID() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParsePRID() unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ParsePRID() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_getStatusIndicator(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *api.Pipeline
		expected string
	}{
		{
			name:     "nil state",
			pipeline: &api.Pipeline{State: nil},
			expected: "○ ",
		},
		{
			name: "successful state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "SUCCESSFUL"},
			},
			expected: "✓ ",
		},
		{
			name: "failed state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "FAILED"},
			},
			expected: "✗ ",
		},
		{
			name: "error state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "ERROR"},
			},
			expected: "✗ ",
		},
		{
			name: "in progress state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "IN_PROGRESS"},
			},
			expected: "● ",
		},
		{
			name: "pending state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "PENDING"},
			},
			expected: "○ ",
		},
		{
			name: "stopped state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "STOPPED"},
			},
			expected: "◐ ",
		},
		{
			name: "unknown state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "UNKNOWN"},
			},
			expected: "○ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ChecksCmd{}
			result := cmd.getStatusIndicator(tt.pipeline)
			if result != tt.expected {
				t.Errorf("getStatusIndicator() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_getPipelineName(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *api.Pipeline
		expected string
	}{
		{
			name: "pipeline with ref name",
			pipeline: &api.Pipeline{
				BuildNumber: 123,
				Target: &api.PipelineTarget{
					RefName: "feature/add-tests",
				},
			},
			expected: "Pipeline #123 (feature/add-tests)",
		},
		{
			name: "pipeline without ref name",
			pipeline: &api.Pipeline{
				BuildNumber: 456,
				Target: &api.PipelineTarget{
					RefName: "",
				},
			},
			expected: "Pipeline #456",
		},
		{
			name: "pipeline without target",
			pipeline: &api.Pipeline{
				BuildNumber: 789,
				Target:      nil,
			},
			expected: "Pipeline #789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ChecksCmd{}
			result := cmd.getPipelineName(tt.pipeline)
			if result != tt.expected {
				t.Errorf("getPipelineName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_formatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours and minutes",
			duration: 1*time.Hour + 30*time.Minute,
			expected: "1h 30m",
		},
		{
			name:     "exact minute",
			duration: 3 * time.Minute,
			expected: "3m 0s",
		},
		{
			name:     "exact hour",
			duration: 2 * time.Hour,
			expected: "2h 0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_getPipelinePriority(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *api.Pipeline
		expected int
	}{
		{
			name:     "nil state",
			pipeline: &api.Pipeline{State: nil},
			expected: 4,
		},
		{
			name: "failed state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "FAILED"},
			},
			expected: 0,
		},
		{
			name: "error state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "ERROR"},
			},
			expected: 0,
		},
		{
			name: "in progress state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "IN_PROGRESS"},
			},
			expected: 1,
		},
		{
			name: "pending state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "PENDING"},
			},
			expected: 2,
		},
		{
			name: "successful state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "SUCCESSFUL"},
			},
			expected: 3,
		},
		{
			name: "stopped state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "STOPPED"},
			},
			expected: 4,
		},
		{
			name: "unknown state",
			pipeline: &api.Pipeline{
				State: &api.PipelineState{Name: "UNKNOWN"},
			},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ChecksCmd{}
			result := cmd.getPipelinePriority(tt.pipeline)
			if result != tt.expected {
				t.Errorf("getPipelinePriority() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_sortChecksByPriority(t *testing.T) {
	cmd := &ChecksCmd{}
	
	pipelines := []*api.Pipeline{
		{
			UUID:        "successful-1",
			BuildNumber: 1,
			State:       &api.PipelineState{Name: "SUCCESSFUL"},
		},
		{
			UUID:        "failed-1",
			BuildNumber: 2,
			State:       &api.PipelineState{Name: "FAILED"},
		},
		{
			UUID:        "running-1",
			BuildNumber: 3,
			State:       &api.PipelineState{Name: "IN_PROGRESS"},
		},
		{
			UUID:        "pending-1",
			BuildNumber: 4,
			State:       &api.PipelineState{Name: "PENDING"},
		},
	}

	sorted := cmd.sortChecksByPriority(pipelines)

	expectedOrder := []string{"failed-1", "running-1", "pending-1", "successful-1"}

	if len(sorted) != len(expectedOrder) {
		t.Fatalf("sortChecksByPriority() returned %d items, expected %d", len(sorted), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if sorted[i].UUID != expected {
			t.Errorf("sortChecksByPriority()[%d] = %s, expected %s", i, sorted[i].UUID, expected)
		}
	}
}

func TestChecksCmd_getChecksSummary(t *testing.T) {
	cmd := &ChecksCmd{}

	tests := []struct {
		name      string
		pipelines []*api.Pipeline
		expected  string
	}{
		{
			name:      "no checks",
			pipelines: []*api.Pipeline{},
			expected:  "no checks",
		},
		{
			name: "all successful",
			pipelines: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
			},
			expected: "2 successful",
		},
		{
			name: "mixed statuses",
			pipelines: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: &api.PipelineState{Name: "FAILED"}},
				{State: &api.PipelineState{Name: "IN_PROGRESS"}},
				{State: &api.PipelineState{Name: "PENDING"}},
			},
			expected: "1 successful, 1 failed, 1 running, 1 pending",
		},
		{
			name: "with nil states",
			pipelines: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: nil},
				{State: &api.PipelineState{Name: "FAILED"}},
			},
			expected: "1 successful, 1 failed, 1 pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.getChecksSummary(tt.pipelines)
			if result != tt.expected {
				t.Errorf("getChecksSummary() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_countByStatus(t *testing.T) {
	cmd := &ChecksCmd{}

	pipelines := []*api.Pipeline{
		{State: &api.PipelineState{Name: "SUCCESSFUL"}},
		{State: &api.PipelineState{Name: "SUCCESSFUL"}},
		{State: &api.PipelineState{Name: "FAILED"}},
		{State: &api.PipelineState{Name: "IN_PROGRESS"}},
		{State: nil},
	}

	tests := []struct {
		name     string
		status   string
		expected int
	}{
		{
			name:     "count successful",
			status:   "SUCCESSFUL",
			expected: 2,
		},
		{
			name:     "count failed",
			status:   "FAILED",
			expected: 1,
		},
		{
			name:     "count running",
			status:   "IN_PROGRESS",
			expected: 1,
		},
		{
			name:     "count non-existent status",
			status:   "UNKNOWN",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.countByStatus(pipelines, tt.status)
			if result != tt.expected {
				t.Errorf("countByStatus() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_hasStatusChanged(t *testing.T) {
	cmd := &ChecksCmd{}

	tests := []struct {
		name      string
		oldChecks []*api.Pipeline
		newChecks []*api.Pipeline
		expected  bool
	}{
		{
			name:      "different lengths",
			oldChecks: []*api.Pipeline{{UUID: "1"}},
			newChecks: []*api.Pipeline{{UUID: "1"}, {UUID: "2"}},
			expected:  true,
		},
		{
			name: "same status",
			oldChecks: []*api.Pipeline{
				{UUID: "1", State: &api.PipelineState{Name: "SUCCESSFUL"}},
			},
			newChecks: []*api.Pipeline{
				{UUID: "1", State: &api.PipelineState{Name: "SUCCESSFUL"}},
			},
			expected: false,
		},
		{
			name: "changed status",
			oldChecks: []*api.Pipeline{
				{UUID: "1", State: &api.PipelineState{Name: "IN_PROGRESS"}},
			},
			newChecks: []*api.Pipeline{
				{UUID: "1", State: &api.PipelineState{Name: "SUCCESSFUL"}},
			},
			expected: true,
		},
		{
			name: "nil to non-nil state",
			oldChecks: []*api.Pipeline{
				{UUID: "1", State: nil},
			},
			newChecks: []*api.Pipeline{
				{UUID: "1", State: &api.PipelineState{Name: "IN_PROGRESS"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.hasStatusChanged(tt.oldChecks, tt.newChecks)
			if result != tt.expected {
				t.Errorf("hasStatusChanged() = %t, expected %t", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_allChecksCompleted(t *testing.T) {
	cmd := &ChecksCmd{}

	tests := []struct {
		name     string
		checks   []*api.Pipeline
		expected bool
	}{
		{
			name:     "empty checks",
			checks:   []*api.Pipeline{},
			expected: true,
		},
		{
			name: "all completed",
			checks: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: &api.PipelineState{Name: "FAILED"}},
				{State: &api.PipelineState{Name: "ERROR"}},
				{State: &api.PipelineState{Name: "STOPPED"}},
			},
			expected: true,
		},
		{
			name: "has pending",
			checks: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: &api.PipelineState{Name: "PENDING"}},
			},
			expected: false,
		},
		{
			name: "has in progress",
			checks: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: &api.PipelineState{Name: "IN_PROGRESS"}},
			},
			expected: false,
		},
		{
			name: "nil state",
			checks: []*api.Pipeline{
				{State: &api.PipelineState{Name: "SUCCESSFUL"}},
				{State: nil},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.allChecksCompleted(tt.checks)
			if result != tt.expected {
				t.Errorf("allChecksCompleted() = %t, expected %t", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_getPipelineDuration(t *testing.T) {
	cmd := &ChecksCmd{}

	now := time.Now()
	createdTime := now.Add(-5 * time.Minute)
	completedTime := now.Add(-1 * time.Minute)

	tests := []struct {
		name     string
		pipeline *api.Pipeline
		expected string
	}{
		{
			name: "completed pipeline",
			pipeline: &api.Pipeline{
				CreatedOn:   &createdTime,
				CompletedOn: &completedTime,
			},
			expected: "4m 0s",
		},
		{
			name: "running pipeline",
			pipeline: &api.Pipeline{
				CreatedOn: &createdTime,
				State:     &api.PipelineState{Name: "IN_PROGRESS"},
			},
			expected: "5m 0s",
		},
		{
			name: "pipeline without created time",
			pipeline: &api.Pipeline{
				CreatedOn: nil,
			},
			expected: "",
		},
		{
			name: "pending pipeline",
			pipeline: &api.Pipeline{
				CreatedOn: &createdTime,
				State:     &api.PipelineState{Name: "PENDING"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.getPipelineDuration(tt.pipeline)
			
			if tt.pipeline.State != nil && tt.pipeline.State.Name == "IN_PROGRESS" {
				if result == "" {
					t.Errorf("getPipelineDuration() returned empty string for running pipeline")
				}
			} else if result != tt.expected {
				t.Errorf("getPipelineDuration() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestChecksCmd_formatOutput_UnsupportedFormat(t *testing.T) {
	cmd := &ChecksCmd{Output: "xml"}
	
	err := cmd.formatOutput(nil, []*api.Pipeline{})
	if err == nil {
		t.Error("formatOutput() expected error for unsupported format but got none")
	}
	
	expectedMsg := "unsupported output format: xml"
	if err.Error() != expectedMsg {
		t.Errorf("formatOutput() error = %q, expected %q", err.Error(), expectedMsg)
	}
}

func TestChecksCmd_Run_InvalidPRID(t *testing.T) {
	cmd := &ChecksCmd{
		PRID: "invalid",
	}
	
	err := cmd.Run(context.Background())
	if err == nil {
		t.Error("Run() expected error for invalid PR ID but got none")
	}
	
	if err.Error() == "" {
		t.Error("Run() returned empty error message")
	}
}

func BenchmarkChecksCmd_sortChecksByPriority(b *testing.B) {
	cmd := &ChecksCmd{}
	
	pipelines := make([]*api.Pipeline, 100)
	states := []string{"SUCCESSFUL", "FAILED", "IN_PROGRESS", "PENDING", "STOPPED"}
	
	for i := 0; i < 100; i++ {
		pipelines[i] = &api.Pipeline{
			UUID:        fmt.Sprintf("pipeline-%d", i),
			BuildNumber: i,
			State:       &api.PipelineState{Name: states[i%len(states)]},
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.sortChecksByPriority(pipelines)
	}
}

func BenchmarkChecksCmd_getChecksSummary(b *testing.B) {
	cmd := &ChecksCmd{}
	
	pipelines := make([]*api.Pipeline, 50)
	states := []string{"SUCCESSFUL", "FAILED", "IN_PROGRESS", "PENDING", "STOPPED"}
	
	for i := 0; i < 50; i++ {
		pipelines[i] = &api.Pipeline{
			State: &api.PipelineState{Name: states[i%len(states)]},
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.getChecksSummary(pipelines)
	}
}
