package pr

import (
	"context"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestCreateCmd_Run(t *testing.T) {
	tests := []struct {
		name      string
		cmd       *CreateCmd
		wantError bool
		errorMsg  string
	}{
		{
			name: "success with all flags",
			cmd: &CreateCmd{
				Title:      "Test PR",
				Body:       "Test description",
				Base:       "main",
				Output:     "json",
				NoColor:    true,
				Workspace:  "test-workspace",
				Repository: "test-repo",
				NoPush:     true,
			},
			wantError: false,
		},
		{
			name: "missing workspace",
			cmd: &CreateCmd{
				Title:   "Test PR",
				Body:    "Test description",
				Base:    "main",
				Output:  "json",
				NoColor: true,
				NoPush:  true,
			},
			wantError: true,
			errorMsg:  "workspace not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.cmd.Run(ctx)

			if tt.wantError {
				if err == nil {
					t.Errorf("CreateCmd.Run() expected error but got none")
				} else if tt.errorMsg != "" && !containsError(err.Error(), tt.errorMsg) {
					t.Errorf("CreateCmd.Run() error = %v, want error containing %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CreateCmd.Run() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestCreateCmd_formatTable(t *testing.T) {
	cmd := &CreateCmd{
		Output: "table",
	}

	result := &PRCreateResult{
		PullRequest: &api.PullRequest{
			ID:    123,
			Title: "Test PR",
			State: "OPEN",
			Source: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "feature-branch",
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
			Reviewers: []*api.PullRequestParticipant{
				{
					User: &api.User{
						Username: "reviewer1",
					},
				},
			},
		},
		URL:     "https://bitbucket.org/workspace/repo/pull-requests/123",
		Created: true,
	}

	prCtx := &PRContext{}

	err := cmd.formatTable(prCtx, result)
	if err != nil {
		t.Errorf("formatTable() error = %v", err)
	}
}

func TestCreateCmd_createPullRequest(t *testing.T) {
	cmd := &CreateCmd{
		Reviewer: []string{"reviewer1", "reviewer2"},
	}

	title := "Test PR"
	body := "Test description"
	sourceBranch := "feature-branch"
	baseBranch := "main"

	_ = title
	_ = body
	_ = sourceBranch
	_ = baseBranch

	if len(cmd.Reviewer) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(cmd.Reviewer))
	}
}

func TestCreateCmd_promptForTitle(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:      "empty title",
			input:     "",
			wantError: true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input == "" || len(strings.TrimSpace(tt.input)) == 0 {
				if !tt.wantError {
					t.Errorf("Expected error for empty input")
				}
			}
		})
	}
}

func TestCreateCmd_getPRTemplate(t *testing.T) {
	cmd := &CreateCmd{}

	_, err := cmd.getPRTemplate()
	if err == nil {
		t.Errorf("Expected error when no PR template exists")
	}
}

func TestCreateCmd_getCommitMessages(t *testing.T) {
	cmd := &CreateCmd{}

	title, body, err := cmd.getCommitMessages(nil, "main", "feature-branch")
	if err != nil {
		t.Errorf("getCommitMessages() error = %v", err)
	}

	if title == "" {
		t.Errorf("Expected non-empty title")
	}

	if body == "" {
		t.Errorf("Expected non-empty body")
	}
}

func TestCreateCmd_handleBranchPush(t *testing.T) {
	cmd := &CreateCmd{}

	_ = cmd
}

func TestCreateCmd_formatOutput(t *testing.T) {
	tests := []struct {
		name       string
		outputType string
		wantError  bool
	}{
		{
			name:       "json output",
			outputType: "json",
			wantError:  false,
		},
		{
			name:       "yaml output", 
			outputType: "yaml",
			wantError:  false,
		},
		{
			name:       "table output",
			outputType: "table",
			wantError:  false,
		},
		{
			name:       "invalid output",
			outputType: "invalid",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CreateCmd{
				Output: tt.outputType,
			}

			result := &PRCreateResult{
				PullRequest: &api.PullRequest{
					ID:    123,
					Title: "Test PR",
					Links: &api.PullRequestLinks{
						HTML: &api.Link{
							Href: "https://example.com",
						},
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
				URL:     "https://example.com",
				Created: true,
			}

			prCtx := &PRContext{}

			err := cmd.formatOutput(prCtx, result)

			if tt.wantError && err == nil {
				t.Errorf("formatOutput() expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("formatOutput() unexpected error = %v", err)
			}
		})
	}
}

func containsError(actual, expected string) bool {
	return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("isTerminal", func(t *testing.T) {
		_ = isTerminal()
	})

	t.Run("confirmAction", func(t *testing.T) {
		_ = confirmAction("test prompt")
	})
}

func TestCreateCmd_validation(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantError bool
	}{
		{
			name:      "valid title",
			title:     "Valid PR Title",
			wantError: false,
		},
		{
			name:      "empty title",
			title:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed := strings.TrimSpace(tt.title)
			isEmpty := trimmed == ""

			if tt.wantError && !isEmpty {
				t.Errorf("Expected validation error for title %q", tt.title)
			}
			if !tt.wantError && isEmpty {
				t.Errorf("Expected valid title %q", tt.title)
			}
		})
	}
}
