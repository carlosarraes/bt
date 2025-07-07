package pr

import (
	"os"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestReviewCmd_ParsePRID(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReviewCmd{
				PRID: tt.prid,
			}

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

func TestReviewCmd_ValidateReviewAction(t *testing.T) {
	tests := []struct {
		name           string
		approve        bool
		requestChanges bool
		comment        bool
		expectedAction reviewAction
		expectErr      bool
		errMsg         string
	}{
		{
			name:           "approve action",
			approve:        true,
			requestChanges: false,
			comment:        false,
			expectedAction: actionApprove,
			expectErr:      false,
		},
		{
			name:           "request changes action",
			approve:        false,
			requestChanges: true,
			comment:        false,
			expectedAction: actionRequestChanges,
			expectErr:      false,
		},
		{
			name:           "comment action",
			approve:        false,
			requestChanges: false,
			comment:        true,
			expectedAction: actionComment,
			expectErr:      false,
		},
		{
			name:           "no action specified",
			approve:        false,
			requestChanges: false,
			comment:        false,
			expectedAction: actionApprove,
			expectErr:      true,
			errMsg:         "must specify one of --approve, --request-changes, or --comment",
		},
		{
			name:           "multiple actions - approve and request changes",
			approve:        true,
			requestChanges: true,
			comment:        false,
			expectedAction: actionApprove,
			expectErr:      true,
			errMsg:         "cannot specify multiple review actions",
		},
		{
			name:           "multiple actions - approve and comment",
			approve:        true,
			requestChanges: false,
			comment:        true,
			expectedAction: actionApprove,
			expectErr:      true,
			errMsg:         "cannot specify multiple review actions",
		},
		{
			name:           "multiple actions - request changes and comment",
			approve:        false,
			requestChanges: true,
			comment:        true,
			expectedAction: actionApprove,
			expectErr:      true,
			errMsg:         "cannot specify multiple review actions",
		},
		{
			name:           "all actions specified",
			approve:        true,
			requestChanges: true,
			comment:        true,
			expectedAction: actionApprove,
			expectErr:      true,
			errMsg:         "cannot specify multiple review actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReviewCmd{
				Approve:        tt.approve,
				RequestChanges: tt.requestChanges,
				Comment:        tt.comment,
			}

			action, err := cmd.validateReviewAction()

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAction, action)
			}
		})
	}
}

func TestReviewCmd_GetCommentBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		bodyFile   string
		action     reviewAction
		force      bool
		fileExists bool
		fileContent string
		expected   string
		expectErr  bool
		errMsg     string
	}{
		{
			name:      "body from command line",
			body:      "This looks good!",
			action:    actionApprove,
			force:     true,
			expected:  "This looks good!",
			expectErr: false,
		},
		{
			name:        "body from file",
			bodyFile:    "test.txt",
			action:      actionComment,
			force:       true,
			fileExists:  true,
			fileContent: "Please fix the tests\nThey are failing",
			expected:    "Please fix the tests\nThey are failing",
			expectErr:   false,
		},
		{
			name:      "both body and file specified",
			body:      "Some comment",
			bodyFile:  "test.txt",
			action:    actionComment,
			force:     true,
			expectErr: true,
			errMsg:    "cannot specify both --body and --body-file",
		},
		{
			name:      "file does not exist",
			bodyFile:  "nonexistent.txt",
			action:    actionComment,
			force:     true,
			expectErr: true,
			errMsg:    "failed to read body file",
		},
		{
			name:      "request changes with empty body (force)",
			action:    actionRequestChanges,
			force:     true,
			expectErr: true,
			errMsg:    "comment is required when requesting changes",
		},
		{
			name:      "approve with empty body (force)",
			action:    actionApprove,
			force:     true,
			expected:  "",
			expectErr: false,
		},
		{
			name:      "comment with empty body (force)",
			action:    actionComment,
			force:     true,
			expected:  "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReviewCmd{
				Body:     tt.body,
				BodyFile: tt.bodyFile,
				Force:    tt.force,
			}

			if tt.bodyFile != "" && tt.fileExists {
				tmpFile := "/tmp/test_review_body.txt"
				err := writeTestFile(tmpFile, tt.fileContent)
				assert.NoError(t, err)
				defer removeTestFile(tmpFile)
				cmd.BodyFile = tmpFile
			}

			result, err := cmd.getCommentBody(tt.action)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestReviewCmd_ConfirmReviewAction(t *testing.T) {
	tests := []struct {
		name        string
		action      reviewAction
		body        string
		prTitle     string
		prID        int
		expectedMsg string
	}{
		{
			name:        "approve action",
			action:      actionApprove,
			body:        "",
			prTitle:     "Fix login bug",
			prID:        123,
			expectedMsg: "approve",
		},
		{
			name:        "request changes action",
			action:      actionRequestChanges,
			body:        "Please add tests",
			prTitle:     "Add new feature",
			prID:        456,
			expectedMsg: "request changes on",
		},
		{
			name:        "comment action",
			action:      actionComment,
			body:        "Looks good but minor suggestions",
			prTitle:     "Refactor utils",
			prID:        789,
			expectedMsg: "comment on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReviewCmd{}
			
			pr := &api.PullRequest{
				ID:    tt.prID,
				Title: tt.prTitle,
			}

			assert.NotNil(t, cmd)
			assert.NotNil(t, pr)
			assert.Equal(t, tt.prID, pr.ID)
			assert.Equal(t, tt.prTitle, pr.Title)
		})
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func removeTestFile(path string) {
	os.Remove(path)
}

func TestReviewActionConstants(t *testing.T) {
	assert.Equal(t, reviewAction(0), actionApprove)
	assert.Equal(t, reviewAction(1), actionRequestChanges)
	assert.Equal(t, reviewAction(2), actionComment)
}

func TestReviewCmd_Structure(t *testing.T) {
	cmd := &ReviewCmd{
		PRID:           "123",
		Approve:        true,
		RequestChanges: false,
		Comment:        false,
		Body:           "test comment",
		BodyFile:       "",
		Force:          false,
		Output:         "table",
		NoColor:        false,
		Workspace:      "test-workspace",
		Repository:     "test-repo",
	}

	assert.Equal(t, "123", cmd.PRID)
	assert.True(t, cmd.Approve)
	assert.False(t, cmd.RequestChanges)
	assert.False(t, cmd.Comment)
	assert.Equal(t, "test comment", cmd.Body)
	assert.Equal(t, "", cmd.BodyFile)
	assert.False(t, cmd.Force)
	assert.Equal(t, "table", cmd.Output)
	assert.False(t, cmd.NoColor)
	assert.Equal(t, "test-workspace", cmd.Workspace)
	assert.Equal(t, "test-repo", cmd.Repository)
}

func BenchmarkReviewCmd_ParsePRID(b *testing.B) {
	cmd := &ReviewCmd{PRID: "12345"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.ParsePRID()
	}
}

func BenchmarkReviewCmd_ValidateReviewAction(b *testing.B) {
	cmd := &ReviewCmd{Approve: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.validateReviewAction()
	}
}
