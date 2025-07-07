package pr

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
)


func TestEditCmd_isInteractiveMode(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *EditCmd
		expected bool
	}{
		{
			name: "Interactive mode - no flags",
			cmd: &EditCmd{
				PRID: "123",
			},
			expected: true,
		},
		{
			name: "Non-interactive - title provided",
			cmd: &EditCmd{
				PRID:  "123",
				Title: "New title",
			},
			expected: false,
		},
		{
			name: "Non-interactive - body provided",
			cmd: &EditCmd{
				PRID: "123",
				Body: "New body",
			},
			expected: false,
		},
		{
			name: "Non-interactive - body file provided",
			cmd: &EditCmd{
				PRID:     "123",
				BodyFile: "body.txt",
			},
			expected: false,
		},
		{
			name: "Non-interactive - add reviewer",
			cmd: &EditCmd{
				PRID:        "123",
				AddReviewer: []string{"user1"},
			},
			expected: false,
		},
		{
			name: "Non-interactive - remove reviewer",
			cmd: &EditCmd{
				PRID:           "123",
				RemoveReviewer: []string{"user1"},
			},
			expected: false,
		},
		{
			name: "Non-interactive - ready flag",
			cmd: &EditCmd{
				PRID:  "123",
				Ready: true,
			},
			expected: false,
		},
		{
			name: "Non-interactive - draft flag",
			cmd: &EditCmd{
				PRID:  "123",
				Draft: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.isInteractiveMode()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEditCmd_hasChanges(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *EditCmd
		expected bool
	}{
		{
			name: "No changes",
			cmd: &EditCmd{
				PRID: "123",
			},
			expected: false,
		},
		{
			name: "Title change",
			cmd: &EditCmd{
				PRID:  "123",
				Title: "New title",
			},
			expected: true,
		},
		{
			name: "Body change",
			cmd: &EditCmd{
				PRID: "123",
				Body: "New body",
			},
			expected: true,
		},
		{
			name: "Body file change",
			cmd: &EditCmd{
				PRID:     "123",
				BodyFile: "body.txt",
			},
			expected: true,
		},
		{
			name: "Add reviewer",
			cmd: &EditCmd{
				PRID:        "123",
				AddReviewer: []string{"user1"},
			},
			expected: true,
		},
		{
			name: "Remove reviewer",
			cmd: &EditCmd{
				PRID:           "123",
				RemoveReviewer: []string{"user1"},
			},
			expected: true,
		},
		{
			name: "Ready flag",
			cmd: &EditCmd{
				PRID:  "123",
				Ready: true,
			},
			expected: true,
		},
		{
			name: "Draft flag",
			cmd: &EditCmd{
				PRID:  "123",
				Draft: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.hasChanges()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEditCmd_buildUpdateRequest(t *testing.T) {
	now := time.Now()
	mockPR := &api.PullRequest{
		ID:          123,
		Title:       "Original Title",
		Description: "Original Description",
		State:       "OPEN",
		CreatedOn:   &now,
		UpdatedOn:   &now,
		Reviewers: []*api.PullRequestParticipant{
			{
				Type: "reviewer",
				User: &api.User{
					Username:    "existing_user",
					DisplayName: "Existing User",
				},
				Role: "REVIEWER",
			},
		},
	}

	tests := []struct {
		name     string
		cmd      *EditCmd
		expected *api.UpdatePullRequestRequest
	}{
		{
			name: "Title update",
			cmd: &EditCmd{
				Title: "New Title",
			},
			expected: &api.UpdatePullRequestRequest{
				Title: "New Title",
			},
		},
		{
			name: "Body update",
			cmd: &EditCmd{
				Body: "New Body",
			},
			expected: &api.UpdatePullRequestRequest{
				Description: "New Body",
			},
		},
		{
			name: "Ready state change from DRAFT",
			cmd: &EditCmd{
				Ready: true,
			},
			expected: &api.UpdatePullRequestRequest{
				State: "OPEN",
			},
		},
		{
			name: "Draft state change from OPEN",
			cmd: &EditCmd{
				Draft: true,
			},
			expected: &api.UpdatePullRequestRequest{
				State: "DRAFT",
			},
		},
		{
			name: "Add reviewer",
			cmd: &EditCmd{
				AddReviewer: []string{"new_user"},
			},
			expected: &api.UpdatePullRequestRequest{
				Reviewers: []*api.PullRequestParticipant{
					{
						Type: "reviewer",
						User: &api.User{
							Username:    "existing_user",
							DisplayName: "Existing User",
						},
						Role: "REVIEWER",
					},
					{
						Type: "reviewer",
						User: &api.User{
							Username: "new_user",
						},
						Role: "REVIEWER",
					},
				},
			},
		},
		{
			name: "Remove reviewer",
			cmd: &EditCmd{
				RemoveReviewer: []string{"existing_user"},
			},
			expected: &api.UpdatePullRequestRequest{
				Reviewers: []*api.PullRequestParticipant{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPR := *mockPR
			if tt.name == "Ready state change from DRAFT" {
				testPR.State = "DRAFT"
			}

			result, err := tt.cmd.buildUpdateRequest(&testPR)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected.Title != "" && result.Title != tt.expected.Title {
				t.Errorf("Expected title '%s', got '%s'", tt.expected.Title, result.Title)
			}

			if tt.expected.Description != "" && result.Description != tt.expected.Description {
				t.Errorf("Expected description '%s', got '%s'", tt.expected.Description, result.Description)
			}

			if tt.expected.State != "" && result.State != tt.expected.State {
				t.Errorf("Expected state '%s', got '%s'", tt.expected.State, result.State)
			}

			if tt.expected.Reviewers != nil {
				if len(result.Reviewers) != len(tt.expected.Reviewers) {
					t.Errorf("Expected %d reviewers, got %d", len(tt.expected.Reviewers), len(result.Reviewers))
					return
				}

				expectedMap := make(map[string]bool)
				for _, r := range tt.expected.Reviewers {
					if r.User != nil {
						expectedMap[r.User.Username] = true
					}
				}

				resultMap := make(map[string]bool)
				for _, r := range result.Reviewers {
					if r.User != nil {
						resultMap[r.User.Username] = true
					}
				}

				for username := range expectedMap {
					if !resultMap[username] {
						t.Errorf("Expected reviewer '%s' not found in result", username)
					}
				}

				for username := range resultMap {
					if !expectedMap[username] {
						t.Errorf("Unexpected reviewer '%s' found in result", username)
					}
				}
			}
		})
	}
}

func TestEditCmd_buildReviewersList(t *testing.T) {
	mockPR := &api.PullRequest{
		Reviewers: []*api.PullRequestParticipant{
			{
				Type: "reviewer",
				User: &api.User{
					Username:    "user1",
					DisplayName: "User One",
				},
				Role: "REVIEWER",
			},
			{
				Type: "reviewer",
				User: &api.User{
					Username:    "user2",
					DisplayName: "User Two",
				},
				Role: "REVIEWER",
			},
		},
	}

	tests := []struct {
		name           string
		addReviewers   []string
		removeReviewers []string
		expectedUsers  []string
	}{
		{
			name:          "No changes",
			expectedUsers: []string{"user1", "user2"},
		},
		{
			name:          "Add new reviewer",
			addReviewers:  []string{"user3"},
			expectedUsers: []string{"user1", "user2", "user3"},
		},
		{
			name:            "Remove existing reviewer",
			removeReviewers: []string{"user1"},
			expectedUsers:   []string{"user2"},
		},
		{
			name:            "Add and remove reviewers",
			addReviewers:    []string{"user3"},
			removeReviewers: []string{"user1"},
			expectedUsers:   []string{"user2", "user3"},
		},
		{
			name:            "Remove all reviewers",
			removeReviewers: []string{"user1", "user2"},
			expectedUsers:   []string{},
		},
		{
			name:          "Add duplicate reviewer",
			addReviewers:  []string{"user1"},
			expectedUsers: []string{"user1", "user2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &EditCmd{
				AddReviewer:    tt.addReviewers,
				RemoveReviewer: tt.removeReviewers,
			}

			result, err := cmd.buildReviewersList(mockPR)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expectedUsers) {
				t.Errorf("Expected %d reviewers, got %d", len(tt.expectedUsers), len(result))
				return
			}

			expectedMap := make(map[string]bool)
			for _, username := range tt.expectedUsers {
				expectedMap[username] = true
			}

			resultMap := make(map[string]bool)
			for _, reviewer := range result {
				if reviewer.User != nil {
					resultMap[reviewer.User.Username] = true
				}
			}

			for username := range expectedMap {
				if !resultMap[username] {
					t.Errorf("Expected reviewer '%s' not found in result", username)
				}
			}

			for username := range resultMap {
				if !expectedMap[username] {
					t.Errorf("Unexpected reviewer '%s' found in result", username)
				}
			}
		})
	}
}

func TestEditCmd_parseEditedContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedTitle string
		expectedBody  string
		expectError   bool
	}{
		{
			name:          "Title only",
			content:       "New Title",
			expectedTitle: "New Title",
			expectedBody:  "",
			expectError:   false,
		},
		{
			name:          "Title and body",
			content:       "New Title\n\nNew body content",
			expectedTitle: "New Title",
			expectedBody:  "New body content",
			expectError:   false,
		},
		{
			name:        "Empty content",
			content:     "",
			expectError: true,
		},
		{
			name:        "Only comments",
			content:     "# This is a comment\n# Another comment",
			expectError: true,
		},
		{
			name:          "Mixed content with comments",
			content:       "# Edit PR\nNew Title\n\n# This is a comment\nNew body",
			expectedTitle: "New Title",
			expectedBody:  "New body",
			expectError:   false,
		},
		{
			name:          "Multiline body",
			content:       "New Title\n\nLine 1\nLine 2\nLine 3",
			expectedTitle: "New Title",
			expectedBody:  "Line 1\nLine 2\nLine 3",
			expectError:   false,
		},
		{
			name:          "Title with extra spacing",
			content:       "   New Title   \n\n   New body   ",
			expectedTitle: "New Title",
			expectedBody:  "New body",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "edit-test-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			cmd := &EditCmd{}
			title, body, err := cmd.parseEditedContent(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if title != tt.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.expectedTitle, title)
			}

			if body != tt.expectedBody {
				t.Errorf("Expected body '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}

func TestEditCmd_readBodyFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "Read from file",
			content:     "This is the body content",
			expectError: false,
		},
		{
			name:        "Read multiline content",
			content:     "Line 1\nLine 2\nLine 3",
			expectError: false,
		},
		{
			name:        "Read empty file",
			content:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "body-test-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			cmd := &EditCmd{BodyFile: tmpFile.Name()}
			result, err := cmd.readBodyFile()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if string(result) != tt.content {
				t.Errorf("Expected content '%s', got '%s'", tt.content, string(result))
			}
		})
	}
}

func TestEditCmd_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *EditCmd
		expectError bool
		errorMsg    string
	}{
		{
			name: "Both ready and draft flags",
			cmd: &EditCmd{
				PRID:  "123",
				Ready: true,
				Draft: true,
			},
			expectError: true,
			errorMsg:    "cannot use both --ready and --draft flags together",
		},
		{
			name: "Invalid PR ID",
			cmd: &EditCmd{
				PRID: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid pull request ID",
		},
		{
			name: "Empty PR ID",
			cmd: &EditCmd{
				PRID: "",
			},
			expectError: true,
			errorMsg:    "pull request ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.cmd.Run(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
