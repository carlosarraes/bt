package pr

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePRID_ReadyCmd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "numeric ID",
			input:    "123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "numeric ID with hash",
			input:    "#123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "zero ID",
			input:    "0",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "negative ID",
			input:    "-1",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid format",
			input:    "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePRID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestReadyCmd_isPRDraft(t *testing.T) {
	tests := []struct {
		name     string
		pr       *api.PullRequest
		expected bool
	}{
		{
			name: "draft state",
			pr: &api.PullRequest{
				State: "DRAFT",
				Title: "Normal title",
			},
			expected: true,
		},
		{
			name: "open state",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "Normal title",
			},
			expected: false,
		},
		{
			name: "draft prefix in title",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "Draft: Add new feature",
			},
			expected: true,
		},
		{
			name: "draft bracket in title",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "[Draft] Add new feature",
			},
			expected: true,
		},
		{
			name: "wip prefix in title",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "WIP: Add new feature",
			},
			expected: true,
		},
		{
			name: "wip bracket in title",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "[WIP] Add new feature",
			},
			expected: true,
		},
		{
			name: "normal title and state",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "Add new feature",
			},
			expected: false,
		},
		{
			name: "draft in middle of title",
			pr: &api.PullRequest{
				State: "OPEN",
				Title: "Add draft implementation",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPRDraft(tt.pr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadyCmd_Run_ValidationErrors(t *testing.T) {

	tests := []struct {
		name    string
		prID    string
		wantErr string
	}{
		{
			name:    "invalid PR ID",
			prID:    "invalid",
			wantErr: "invalid pull request ID",
		},
		{
			name:    "zero PR ID",
			prID:    "0",
			wantErr: "must be positive",
		},
		{
			name:    "negative PR ID",
			prID:    "-1",
			wantErr: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePRID(tt.prID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestReadyCmd_formatOutput(t *testing.T) {
	result := &PRReadyResult{
		PullRequest: &api.PullRequest{
			ID:    123,
			Title: "Test PR",
			State: "OPEN",
			Author: &api.User{
				Username: "testuser",
			},
			Source: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "feature",
				},
			},
			Destination: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "main",
				},
			},
		},
		Ready: true,
	}

	assert.NotNil(t, result.PullRequest)
	assert.True(t, result.Ready)
	assert.Equal(t, 123, result.PullRequest.ID)
	assert.Equal(t, "Test PR", result.PullRequest.Title)
	assert.Equal(t, "OPEN", result.PullRequest.State)
}

func TestReadyCmd_formatTable(t *testing.T) {
	cmd := &ReadyCmd{
		Output: "table",
	}

	result := &PRReadyResult{
		PullRequest: &api.PullRequest{
			ID:    123,
			Title: "Test PR",
			State: "OPEN",
			Author: &api.User{
				Username: "testuser",
			},
			Source: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "feature",
				},
			},
			Destination: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "main",
				},
			},
			Links: &api.PullRequestLinks{
				HTML: &api.Link{
					Href: "https://bitbucket.org/workspace/repo/pull-requests/123",
				},
			},
		},
		Comment: &api.PullRequestComment{
			Content: &api.PullRequestCommentContent{
				Raw: "This PR is now ready for review",
			},
		},
		Ready: true,
	}

	err := cmd.formatTable(nil, result)
	assert.NoError(t, err)
}
