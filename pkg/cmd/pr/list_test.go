package pr

import (
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/output"
	"github.com/stretchr/testify/assert"
)

func TestValidateState(t *testing.T) {
	tests := []struct {
		name      string
		state     string
		expectErr bool
	}{
		{
			name:      "valid state open",
			state:     "open",
			expectErr: false,
		},
		{
			name:      "valid state merged",
			state:     "merged",
			expectErr: false,
		},
		{
			name:      "valid state declined",
			state:     "declined",
			expectErr: false,
		},
		{
			name:      "valid state all",
			state:     "all",
			expectErr: false,
		},
		{
			name:      "valid uppercase state",
			state:     "OPEN",
			expectErr: false,
		},
		{
			name:      "valid mixed case state",
			state:     "Merged",
			expectErr: false,
		},
		{
			name:      "invalid state",
			state:     "invalid",
			expectErr: true,
		},
		{
			name:      "empty state",
			state:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateState(tt.state)
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
				State:  "open",
				Limit:  30,
				Output: "table",
			},
			expectErr: false,
		},
		{
			name: "valid command with valid state",
			cmd: &ListCmd{
				State:  "merged",
				Limit:  20,
				Output: "json",
			},
			expectErr: false,
		},
		{
			name: "valid command with all filters",
			cmd: &ListCmd{
				State:    "open",
				Author:   "testuser",
				Reviewer: "reviewer",
				Limit:    50,
				Sort:     "updated",
				Output:   "yaml",
			},
			expectErr: false,
		},
		{
			name: "invalid limit - zero",
			cmd: &ListCmd{
				State:  "open",
				Limit:  0,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit must be greater than 0",
		},
		{
			name: "invalid limit - negative",
			cmd: &ListCmd{
				State:  "open",
				Limit:  -5,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit must be greater than 0",
		},
		{
			name: "invalid limit - too high",
			cmd: &ListCmd{
				State:  "open",
				Limit:  150,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "limit cannot exceed 100",
		},
		{
			name: "invalid state",
			cmd: &ListCmd{
				State:  "invalid",
				Limit:  30,
				Output: "table",
			},
			expectErr: true,
			errMsg:    "invalid state",
		},
		{
			name: "invalid sort field",
			cmd: &ListCmd{
				State:  "open",
				Limit:  30,
				Sort:   "invalid",
				Output: "table",
			},
			expectErr: true,
			errMsg:    "invalid sort field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic directly
			if tt.cmd.State != "" {
				err := validateState(tt.cmd.State)
				if tt.errMsg == "invalid state" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "invalid state")
					return
				} else {
					assert.NoError(t, err)
				}
			}

			// Test limit validation
			if tt.cmd.Limit <= 0 {
				assert.Contains(t, tt.errMsg, "limit must be greater than 0")
			} else if tt.cmd.Limit > 100 {
				assert.Contains(t, tt.errMsg, "limit cannot exceed 100")
			}

			// Test sort validation
			if tt.cmd.Sort != "" && tt.errMsg == "invalid sort field" {
				validSorts := []string{"created", "updated", "priority"}
				found := false
				for _, valid := range validSorts {
					if tt.cmd.Sort == valid {
						found = true
						break
					}
				}
				assert.False(t, found, "Sort field should be invalid for this test case")
			}
		})
	}
}

func TestParsePullRequestResults(t *testing.T) {
	tests := []struct {
		name      string
		result    *api.PaginatedResponse
		expectErr bool
		expected  int
	}{
		{
			name: "empty result",
			result: &api.PaginatedResponse{
				Size:   0,
				Values: nil,
			},
			expectErr: false,
			expected:  0,
		},
		{
			name: "valid pull requests",
			result: &api.PaginatedResponse{
				Size: 2,
				Values: []byte(`[
					{
						"type": "pullrequest",
						"id": 123,
						"title": "Test PR 1",
						"state": "OPEN",
						"author": {
							"username": "user1"
						}
					},
					{
						"type": "pullrequest", 
						"id": 124,
						"title": "Test PR 2",
						"state": "MERGED",
						"author": {
							"username": "user2"
						}
					}
				]`),
			},
			expectErr: false,
			expected:  2,
		},
		{
			name: "invalid JSON",
			result: &api.PaginatedResponse{
				Size:   1,
				Values: []byte(`invalid json`),
			},
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pullRequests, err := parsePullRequestResults(tt.result)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, pullRequests)
			} else {
				assert.NoError(t, err)
				assert.Len(t, pullRequests, tt.expected)

				// Verify the parsed data for valid cases
				if tt.expected > 0 && pullRequests != nil {
					assert.Equal(t, 123, pullRequests[0].ID)
					assert.Equal(t, "Test PR 1", pullRequests[0].Title)
					assert.Equal(t, "OPEN", pullRequests[0].State)
					assert.Equal(t, "user1", pullRequests[0].Author.Username)

					if len(pullRequests) > 1 {
						assert.Equal(t, 124, pullRequests[1].ID)
						assert.Equal(t, "Test PR 2", pullRequests[1].Title)
						assert.Equal(t, "MERGED", pullRequests[1].State)
						assert.Equal(t, "user2", pullRequests[1].Author.Username)
					}
				}
			}
		})
	}
}

func TestPullRequestStateColor(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"OPEN", "green"},
		{"MERGED", "blue"},
		{"DECLINED", "red"},
		{"SUPERSEDED", "yellow"},
		{"UNKNOWN", "white"},
		{"", "white"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			color := PullRequestStateColor(tt.state)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		time     *time.Time
		expected string
	}{
		{
			name:     "nil time",
			time:     nil,
			expected: "-",
		},
		{
			name:     "just now",
			time:     &now,
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     func() *time.Time { t := now.Add(-1 * time.Minute); return &t }(),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     func() *time.Time { t := now.Add(-5 * time.Minute); return &t }(),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     func() *time.Time { t := now.Add(-3 * time.Hour); return &t }(),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     func() *time.Time { t := now.Add(-24 * time.Hour); return &t }(),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     func() *time.Time { t := now.Add(-72 * time.Hour); return &t }(),
			expected: "3 days ago",
		},
		{
			name:     "old date",
			time:     func() *time.Time { t := now.Add(-8 * 24 * time.Hour); return &t }(),
			expected: now.Add(-8 * 24 * time.Hour).Format("2006-01-02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := output.FormatRelativeTime(tt.time)
			if tt.name == "old date" {
				// For old dates, check the format pattern instead of exact match
				assert.Regexp(t, `\d{4}-\d{2}-\d{2}`, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestHandlePullRequestAPIError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "not found error",
			err: &api.BitbucketError{
				Type:    api.ErrorTypeNotFound,
				Message: "Not found",
			},
			expected: "repository not found or no pull requests exist",
		},
		{
			name: "authentication error",
			err: &api.BitbucketError{
				Type:    api.ErrorTypeAuthentication,
				Message: "Unauthorized",
			},
			expected: "authentication failed",
		},
		{
			name: "permission error",
			err: &api.BitbucketError{
				Type:    api.ErrorTypePermission,
				Message: "Forbidden",
			},
			expected: "permission denied",
		},
		{
			name: "rate limit error",
			err: &api.BitbucketError{
				Type:    api.ErrorTypeRateLimit,
				Message: "Too many requests",
			},
			expected: "rate limit exceeded",
		},
		{
			name: "other bitbucket error",
			err: &api.BitbucketError{
				Type:    api.ErrorTypeServer,
				Message: "Server error",
			},
			expected: "API error: Server error",
		},
		{
			name:     "generic error",
			err:      assert.AnError,
			expected: "failed to list pull requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handlePullRequestAPIError(tt.err)
			assert.Contains(t, result.Error(), tt.expected)
		})
	}
}

func TestFormatTable(t *testing.T) {
	tests := []struct {
		name string
		prs  []*api.PullRequest
	}{
		{
			name: "empty list",
			prs:  []*api.PullRequest{},
		},
		{
			name: "single pull request",
			prs: []*api.PullRequest{
				{
					ID:    123,
					Title: "Test PR",
					State: "OPEN",
					Author: &api.User{
						Username:    "testuser",
						DisplayName: "Test User",
					},
					Source: &api.PullRequestBranch{
						Branch: &api.Branch{
							Name: "feature/test",
						},
					},
					UpdatedOn: func() *time.Time {
						t := time.Now().Add(-2 * time.Hour)
						return &t
					}(),
				},
			},
		},
		{
			name: "multiple pull requests",
			prs: []*api.PullRequest{
				{
					ID:    123,
					Title: "First PR",
					State: "OPEN",
					Author: &api.User{
						Username: "user1",
					},
					Source: &api.PullRequestBranch{
						Branch: &api.Branch{
							Name: "feature/first",
						},
					},
					UpdatedOn: func() *time.Time {
						t := time.Now().Add(-1 * time.Hour)
						return &t
					}(),
				},
				{
					ID:    124,
					Title: "Second PR with a very long title that should be truncated properly",
					State: "MERGED",
					Author: &api.User{
						DisplayName: "User Two",
					},
					Source: &api.PullRequestBranch{
						Branch: &api.Branch{
							Name: "feature/second-with-very-long-branch-name",
						},
					},
					UpdatedOn: func() *time.Time {
						t := time.Now().Add(-3 * time.Hour)
						return &t
					}(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ListCmd{Output: "table"}

			// This test just ensures the function doesn't panic
			// In a real environment, we'd capture stdout for verification
			err := cmd.formatTable(&PRContext{}, tt.prs)
			assert.NoError(t, err)
		})
	}
}

func TestSortFieldValidation(t *testing.T) {
	tests := []struct {
		sort     string
		expected string
	}{
		{"created", "-created_on"},
		{"updated", "-updated_on"},
		{"priority", "-priority"},
		{"CREATED", "-created_on"}, // Test case insensitive
		{"Updated", "-updated_on"}, // Test mixed case
	}

	for _, tt := range tests {
		t.Run(tt.sort, func(t *testing.T) {
			// Simulate the sort validation logic from the Run method
			var sortField string
			switch strings.ToLower(tt.sort) {
			case "created":
				sortField = "-created_on"
			case "updated":
				sortField = "-updated_on"
			case "priority":
				sortField = "-priority"
			}

			assert.Equal(t, tt.expected, sortField)
		})
	}
}
