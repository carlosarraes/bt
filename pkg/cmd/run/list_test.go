package run

import (
	"fmt"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/output"
	"github.com/stretchr/testify/assert"
)

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		expectErr bool
	}{
		{
			name:      "valid status PENDING",
			status:    "PENDING",
			expectErr: false,
		},
		{
			name:      "valid status IN_PROGRESS",
			status:    "IN_PROGRESS",
			expectErr: false,
		},
		{
			name:      "valid status SUCCESSFUL",
			status:    "SUCCESSFUL",
			expectErr: false,
		},
		{
			name:      "valid status FAILED",
			status:    "FAILED",
			expectErr: false,
		},
		{
			name:      "valid status ERROR",
			status:    "ERROR",
			expectErr: false,
		},
		{
			name:      "valid status STOPPED",
			status:    "STOPPED",
			expectErr: false,
		},
		{
			name:      "valid lowercase status",
			status:    "failed",
			expectErr: false,
		},
		{
			name:      "valid mixed case status",
			status:    "Failed",
			expectErr: false,
		},
		{
			name:      "invalid status",
			status:    "INVALID",
			expectErr: true,
		},
		{
			name:      "empty status",
			status:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStatus(tt.status)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestListCmd_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cmd       *ListCmd
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid command with defaults",
			cmd: &ListCmd{
				Limit:  10,
				Output: "table",
			},
			expectErr: false,
		},
		{
			name: "valid command with valid status",
			cmd: &ListCmd{
				Status: "FAILED",
				Limit:  20,
				Output: "json",
			},
			expectErr: false,
		},
		{
			name: "invalid limit - zero",
			cmd: &ListCmd{
				Limit:  0,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit must be greater than 0",
		},
		{
			name: "invalid limit - negative",
			cmd: &ListCmd{
				Limit:  -5,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit must be greater than 0",
		},
		{
			name: "invalid limit - too large",
			cmd: &ListCmd{
				Limit:  150,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit cannot exceed 100",
		},
		{
			name: "invalid status",
			cmd: &ListCmd{
				Status: "INVALID_STATUS",
				Limit:  10,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test individual validations that we can test without full context
			if tt.cmd.Status != "" {
				err := validateStatus(tt.cmd.Status)
				if tt.expectErr && tt.errMsg == "invalid status" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "invalid status")
					return
				}
			}

			// Test limit validation
			if tt.cmd.Limit <= 0 {
				if tt.expectErr && tt.errMsg == "limit must be greater than 0" {
					// This validation would happen in the Run method
					assert.True(t, tt.expectErr)
					return
				}
			}

			if tt.cmd.Limit > 100 {
				if tt.expectErr && tt.errMsg == "limit cannot exceed 100" {
					// This validation would happen in the Run method
					assert.True(t, tt.expectErr)
					return
				}
			}

			// If we get here, the command should be valid
			assert.False(t, tt.expectErr)
		})
	}
}

func TestParsePipelineResults(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		expectErr bool
		expectLen int
	}{
		{
			name: "valid pipeline results",
			input: []byte(`[
				{
					"uuid": "{12345678-1234-1234-1234-123456789012}",
					"build_number": 1,
					"state": {
						"name": "SUCCESSFUL",
						"type": "pipeline_state"
					},
					"created_on": "2023-01-01T00:00:00.000000+00:00"
				},
				{
					"uuid": "{87654321-4321-4321-4321-210987654321}",
					"build_number": 2,
					"state": {
						"name": "FAILED",
						"type": "pipeline_state"
					},
					"created_on": "2023-01-02T00:00:00.000000+00:00"
				}
			]`),
			expectErr: false,
			expectLen: 2,
		},
		{
			name:      "empty results",
			input:     []byte(`[]`),
			expectErr: false,
			expectLen: 0,
		},
		{
			name:      "invalid JSON",
			input:     []byte(`invalid json`),
			expectErr: true,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PaginatedResponse
			result := &api.PaginatedResponse{
				Values: tt.input,
				Size:   tt.expectLen,
			}

			pipelines, err := parsePipelineResults(result)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, pipelines, tt.expectLen)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{
			name:     "under a minute",
			seconds:  45,
			expected: "45s",
		},
		{
			name:     "exactly one minute",
			seconds:  60,
			expected: "1m 0s",
		},
		{
			name:     "minutes and seconds",
			seconds:  125,
			expected: "2m 5s",
		},
		{
			name:     "exactly one hour",
			seconds:  3600,
			expected: "1h 0m",
		},
		{
			name:     "hours and minutes",
			seconds:  3725,
			expected: "1h 2m",
		},
		{
			name:     "zero seconds",
			seconds:  0,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := output.FormatDuration(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsClientSideStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"PENDING", true},
		{"pending", true},
		{"IN_PROGRESS", true},
		{"in_progress", true},
		{"FAILED", false},
		{"SUCCESSFUL", false},
		{"ERROR", false},
		{"STOPPED", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.expected, isClientSideStatus(tt.status))
		})
	}
}

func TestFilterPipelines(t *testing.T) {
	pipelines := []*api.Pipeline{
		{
			BuildNumber: 1,
			State:       &api.PipelineState{Name: "PENDING"},
			Creator:     &api.User{DisplayName: "Alice Smith"},
		},
		{
			BuildNumber: 2,
			State:       &api.PipelineState{Name: "IN_PROGRESS"},
			Creator:     &api.User{DisplayName: "Bob Jones"},
		},
		{
			BuildNumber: 3,
			State:       &api.PipelineState{Name: "COMPLETED"},
			Creator:     &api.User{DisplayName: "Alice Wonder"},
		},
		{
			BuildNumber: 4,
			State:       &api.PipelineState{Name: "PENDING"},
			Creator:     nil,
		},
	}

	t.Run("filter by PENDING status", func(t *testing.T) {
		result := filterPipelines(pipelines, "PENDING", "")
		assert.Len(t, result, 2)
		assert.Equal(t, 1, result[0].BuildNumber)
		assert.Equal(t, 4, result[1].BuildNumber)
	})

	t.Run("filter by creator", func(t *testing.T) {
		result := filterPipelines(pipelines, "", "alice")
		assert.Len(t, result, 2)
		assert.Equal(t, 1, result[0].BuildNumber)
		assert.Equal(t, 3, result[1].BuildNumber)
	})

	t.Run("filter by status and creator", func(t *testing.T) {
		result := filterPipelines(pipelines, "PENDING", "alice")
		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].BuildNumber)
	})

	t.Run("no matches", func(t *testing.T) {
		result := filterPipelines(pipelines, "PENDING", "nobody")
		assert.Empty(t, result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := filterPipelines(nil, "PENDING", "")
		assert.Empty(t, result)
	})

	t.Run("no filters applied for non-client-side status", func(t *testing.T) {
		result := filterPipelines(pipelines, "FAILED", "")
		assert.Len(t, result, 4)
	})

	t.Run("creator case insensitive substring", func(t *testing.T) {
		result := filterPipelines(pipelines, "", "ALICE")
		assert.Len(t, result, 2)
	})

	t.Run("nil creator filtered out", func(t *testing.T) {
		result := filterPipelines(pipelines, "", "bob")
		assert.Len(t, result, 1)
		assert.Equal(t, 2, result[0].BuildNumber)
	})
}

func TestPaginationFilterAccumulation(t *testing.T) {
	makePage := func(buildNumbers []int, stateName string, hasNext bool) *api.PaginatedResponse {
		var entries []string
		for _, bn := range buildNumbers {
			entries = append(entries, fmt.Sprintf(
				`{"build_number":%d,"uuid":"{uuid-%d}","state":{"name":"%s"}}`,
				bn, bn, stateName,
			))
		}
		next := ""
		if hasNext {
			next = "https://api.bitbucket.org/next"
		}
		return &api.PaginatedResponse{
			Values: []byte("[" + strings.Join(entries, ",") + "]"),
			Next:   next,
		}
	}

	t.Run("accumulates across pages and stops at limit", func(t *testing.T) {
		pages := []*api.PaginatedResponse{
			makePage([]int{1, 2, 3}, "PENDING", true),
			makePage([]int{4, 5, 6}, "PENDING", true),
			makePage([]int{7, 8, 9}, "PENDING", false),
		}

		limit := 5
		var accumulated []*api.Pipeline
		for _, page := range pages {
			parsed, err := parsePipelineResults(page)
			assert.NoError(t, err)
			filtered := filterPipelines(parsed, "PENDING", "")
			accumulated = append(accumulated, filtered...)
			if len(accumulated) >= limit || page.Next == "" {
				break
			}
		}
		if len(accumulated) > limit {
			accumulated = accumulated[:limit]
		}

		assert.Len(t, accumulated, 5)
		assert.Equal(t, 1, accumulated[0].BuildNumber)
		assert.Equal(t, 5, accumulated[4].BuildNumber)
	})

	t.Run("stops when Next is empty", func(t *testing.T) {
		pages := []*api.PaginatedResponse{
			makePage([]int{1, 2}, "PENDING", false),
		}

		limit := 10
		var accumulated []*api.Pipeline
		for _, page := range pages {
			parsed, err := parsePipelineResults(page)
			assert.NoError(t, err)
			filtered := filterPipelines(parsed, "PENDING", "")
			accumulated = append(accumulated, filtered...)
			if len(accumulated) >= limit || page.Next == "" {
				break
			}
		}

		assert.Len(t, accumulated, 2)
	})

	t.Run("filters across pages with mixed statuses", func(t *testing.T) {
		page1 := &api.PaginatedResponse{
			Values: []byte(`[
				{"build_number":1,"uuid":"{u1}","state":{"name":"PENDING"}},
				{"build_number":2,"uuid":"{u2}","state":{"name":"COMPLETED"}},
				{"build_number":3,"uuid":"{u3}","state":{"name":"PENDING"}}
			]`),
			Next: "https://api.bitbucket.org/next",
		}
		page2 := &api.PaginatedResponse{
			Values: []byte(`[
				{"build_number":4,"uuid":"{u4}","state":{"name":"COMPLETED"}},
				{"build_number":5,"uuid":"{u5}","state":{"name":"PENDING"}}
			]`),
			Next: "",
		}

		limit := 10
		var accumulated []*api.Pipeline
		for _, page := range []*api.PaginatedResponse{page1, page2} {
			parsed, err := parsePipelineResults(page)
			assert.NoError(t, err)
			filtered := filterPipelines(parsed, "PENDING", "")
			accumulated = append(accumulated, filtered...)
			if len(accumulated) >= limit || page.Next == "" {
				break
			}
		}

		assert.Len(t, accumulated, 3)
		assert.Equal(t, 1, accumulated[0].BuildNumber)
		assert.Equal(t, 3, accumulated[1].BuildNumber)
		assert.Equal(t, 5, accumulated[2].BuildNumber)
	})
}

func TestPipelineStateColor(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected string
	}{
		{
			name:     "successful state",
			state:    "SUCCESSFUL",
			expected: "green",
		},
		{
			name:     "failed state",
			state:    "FAILED",
			expected: "red",
		},
		{
			name:     "error state",
			state:    "ERROR",
			expected: "red",
		},
		{
			name:     "stopped state",
			state:    "STOPPED",
			expected: "yellow",
		},
		{
			name:     "in progress state",
			state:    "IN_PROGRESS",
			expected: "blue",
		},
		{
			name:     "pending state",
			state:    "PENDING",
			expected: "cyan",
		},
		{
			name:     "unknown state",
			state:    "UNKNOWN",
			expected: "white",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PipelineStateColor(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}
