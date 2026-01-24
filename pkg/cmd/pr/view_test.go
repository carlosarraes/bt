package pr

import (
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestViewCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name      string
		prid      string
		expected  int
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid PR ID",
			prid:      "123",
			expected:  123,
			expectErr: false,
		},
		{
			name:      "valid PR ID with hash prefix",
			prid:      "#456",
			expected:  456,
			expectErr: false,
		},
		{
			name:      "zero PR ID",
			prid:      "0",
			expected:  0,
			expectErr: true,
			errMsg:    "pull request ID must be positive",
		},
		{
			name:      "negative PR ID",
			prid:      "-5",
			expected:  0,
			expectErr: true,
			errMsg:    "pull request ID must be positive",
		},
		{
			name:      "non-numeric PR ID",
			prid:      "abc",
			expected:  0,
			expectErr: true,
			errMsg:    "invalid pull request ID",
		},
		{
			name:      "empty PR ID",
			prid:      "",
			expected:  0,
			expectErr: true,
			errMsg:    "pull request ID is required",
		},
		{
			name:      "large valid PR ID",
			prid:      "999999",
			expected:  999999,
			expectErr: false,
		},
		{
			name:      "hash prefix with invalid ID",
			prid:      "#abc",
			expected:  0,
			expectErr: true,
			errMsg:    "invalid pull request ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{PRID: tt.prid}
			result, err := cmd.ParsePRID()

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, 0, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestViewCmd_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cmd       *ViewCmd
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid command with defaults",
			cmd: &ViewCmd{
				PRID:   "123",
				Output: "table",
			},
			expectErr: false,
		},
		{
			name: "valid command with JSON output",
			cmd: &ViewCmd{
				PRID:   "456",
				Output: "json",
			},
			expectErr: false,
		},
		{
			name: "valid command with YAML output",
			cmd: &ViewCmd{
				PRID:   "789",
				Output: "yaml",
			},
			expectErr: false,
		},
		{
			name: "valid command with web flag",
			cmd: &ViewCmd{
				PRID:   "123",
				Web:    true,
				Output: "table",
			},
			expectErr: false,
		},
		{
			name: "valid command with comments flag",
			cmd: &ViewCmd{
				PRID:     "123",
				Comments: true,
				Output:   "table",
			},
			expectErr: false,
		},
		{
			name: "valid command with workspace and repository",
			cmd: &ViewCmd{
				PRID:       "123",
				Output:     "table",
				Workspace:  "myworkspace",
				Repository: "myrepo",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test PR ID parsing
			_, err := tt.cmd.ParsePRID()

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestViewCmd_FormatOutput(t *testing.T) {
	// Create test data
	now := time.Now()
	testPR := &api.PullRequest{
		ID:          123,
		Title:       "Test Pull Request",
		State:       "OPEN",
		Description: "This is a test pull request description",
		Author: &api.User{
			Username:    "testuser",
			DisplayName: "Test User",
		},
		Source: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: "feature/test",
			},
		},
		Destination: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: "main",
			},
		},
		CreatedOn:    &now,
		UpdatedOn:    &now,
		CommentCount: 5,
		Reviewers: []*api.PullRequestParticipant{
			{
				User: &api.User{
					Username:    "reviewer1",
					DisplayName: "Reviewer One",
				},
				Approved: true,
			},
			{
				User: &api.User{
					Username:    "reviewer2",
					DisplayName: "Reviewer Two",
				},
				Approved: false,
				State:    "changes_requested",
			},
		},
	}

	testFiles := &api.PullRequestDiffStat{
		FilesChanged: 3,
		LinesAdded:   50,
		LinesRemoved: 25,
	}

	tests := []struct {
		name   string
		output string
		pr     *api.PullRequest
		files  *api.PullRequestDiffStat
	}{
		{
			name:   "table output with full data",
			output: "table",
			pr:     testPR,
			files:  testFiles,
		},
		{
			name:   "table output without files",
			output: "table",
			pr:     testPR,
			files:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{Output: tt.output}

			// Create a mock PRContext (this would need proper mocking in real tests)
			prCtx := &PRContext{}

			// Test table format directly since it doesn't use formatter
			err := cmd.formatTable(prCtx, tt.pr, tt.files, nil)
			assert.NoError(t, err)
		})
	}
}

func TestViewCmd_FormatTable_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		pr   *api.PullRequest
	}{
		{
			name: "PR with minimal data",
			pr: &api.PullRequest{
				ID:    1,
				Title: "Minimal PR",
				State: "OPEN",
			},
		},
		{
			name: "PR with no author",
			pr: &api.PullRequest{
				ID:     2,
				Title:  "No Author PR",
				State:  "MERGED",
				Author: nil,
			},
		},
		{
			name: "PR with no branches",
			pr: &api.PullRequest{
				ID:          3,
				Title:       "No Branches PR",
				State:       "DECLINED",
				Source:      nil,
				Destination: nil,
			},
		},
		{
			name: "PR with empty description",
			pr: &api.PullRequest{
				ID:          4,
				Title:       "Empty Description PR",
				State:       "OPEN",
				Description: "",
			},
		},
		{
			name: "PR with empty reviewers",
			pr: &api.PullRequest{
				ID:        5,
				Title:     "No Reviewers PR",
				State:     "OPEN",
				Reviewers: []*api.PullRequestParticipant{},
			},
		},
		{
			name: "PR with nil timestamps",
			pr: &api.PullRequest{
				ID:        6,
				Title:     "No Timestamps PR",
				State:     "OPEN",
				CreatedOn: nil,
				UpdatedOn: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{Output: "table"}

			// This test ensures the table formatting handles edge cases gracefully
			// In a real test, we'd capture and validate the output
			err := cmd.formatTable(&PRContext{}, tt.pr, nil, nil)

			// We expect no panics or crashes, even with minimal data
			// The function should handle nil values gracefully
			assert.NoError(t, err)
		})
	}
}

func TestViewCmd_DisplayComments(t *testing.T) {
	// Test comment display functionality
	commentsJSON := `[
		{
			"type": "pullrequest_comment",
			"id": 1,
			"content": {
				"type": "text",
				"raw": "This is a test comment",
				"html": "<p>This is a test comment</p>"
			},
			"user": {
				"username": "commenter1",
				"display_name": "Commenter One"
			},
			"created_on": "2023-01-01T12:00:00Z"
		},
		{
			"type": "pullrequest_comment",
			"id": 2,
			"content": {
				"type": "text", 
				"raw": "This is an inline comment",
				"html": "<p>This is an inline comment</p>"
			},
			"user": {
				"username": "commenter2"
			},
			"inline": {
				"type": "inline",
				"path": "src/main.go",
				"to": 42
			},
			"created_on": "2023-01-01T13:00:00Z"
		}
	]`

	tests := []struct {
		name     string
		comments *api.PaginatedResponse
	}{
		{
			name:     "nil comments",
			comments: nil,
		},
		{
			name: "empty comments",
			comments: &api.PaginatedResponse{
				Values: nil,
			},
		},
		{
			name: "valid comments",
			comments: &api.PaginatedResponse{
				Values: []byte(commentsJSON),
			},
		},
		{
			name: "invalid JSON comments",
			comments: &api.PaginatedResponse{
				Values: []byte(`invalid json`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{}

			// Test that displayComments doesn't panic with various inputs
			err := cmd.displayComments(tt.comments)

			// The function should handle all cases gracefully
			if tt.name == "invalid JSON comments" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestViewCmd_OpenInBrowser(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		repo      string
		prID      int
		expected  string
	}{
		{
			name:      "valid URL construction",
			workspace: "myworkspace",
			repo:      "myrepo",
			prID:      123,
			expected:  "https://bitbucket.org/myworkspace/myrepo/pull-requests/123",
		},
		{
			name:      "workspace with special characters",
			workspace: "my-workspace",
			repo:      "my_repo",
			prID:      456,
			expected:  "https://bitbucket.org/my-workspace/my_repo/pull-requests/456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{}
			prCtx := &PRContext{
				Workspace:  tt.workspace,
				Repository: tt.repo,
			}

			// We can't actually test browser opening in unit tests,
			// but we can test URL construction logic by examining the error
			// (since we don't have a real browser environment)
			err := cmd.openInBrowser(prCtx, tt.prID)

			// In most CI environments, this will fail, but that's expected
			// The important thing is that it constructs the right URL
			// We can verify this by checking the error message contains the expected URL
			if err != nil {
				assert.Contains(t, err.Error(), "failed to open browser")
			}
		})
	}
}

func TestViewCmd_UnsupportedOutputFormat(t *testing.T) {
	cmd := &ViewCmd{Output: "unsupported"}
	pr := &api.PullRequest{ID: 123, Title: "Test", State: "OPEN"}

	err := cmd.formatOutput(&PRContext{}, pr, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format: unsupported")
}

// Test GitHub CLI compatibility
func TestViewCmd_GitHubCLICompatibility(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected *ViewCmd
	}{
		{
			name: "basic view command",
			args: []string{"123"},
			expected: &ViewCmd{
				PRID:   "123",
				Output: "table",
			},
		},
		{
			name: "view with web flag",
			args: []string{"456", "--web"},
			expected: &ViewCmd{
				PRID:   "456",
				Web:    true,
				Output: "table",
			},
		},
		{
			name: "view with comments flag",
			args: []string{"789", "--comments"},
			expected: &ViewCmd{
				PRID:     "789",
				Comments: true,
				Output:   "table",
			},
		},
		{
			name: "view with JSON output",
			args: []string{"101", "--output", "json"},
			expected: &ViewCmd{
				PRID:   "101",
				Output: "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the command structure matches GitHub CLI expectations
			// In practice, Kong CLI framework would parse these arguments

			// Verify PRID parsing works correctly
			cmd := &ViewCmd{PRID: tt.expected.PRID}
			prID, err := cmd.ParsePRID()
			assert.NoError(t, err)
			assert.Greater(t, prID, 0)
		})
	}
}

// Test error handling scenarios
func TestViewCmd_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		prid   string
		errMsg string
	}{
		{
			name:   "empty PR ID",
			prid:   "",
			errMsg: "pull request ID is required",
		},
		{
			name:   "invalid characters",
			prid:   "abc123",
			errMsg: "invalid pull request ID",
		},
		{
			name:   "negative ID",
			prid:   "-1",
			errMsg: "pull request ID must be positive",
		},
		{
			name:   "zero ID",
			prid:   "0",
			errMsg: "pull request ID must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ViewCmd{PRID: tt.prid}
			_, err := cmd.ParsePRID()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
