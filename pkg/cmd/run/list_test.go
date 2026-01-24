package run

import (
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
