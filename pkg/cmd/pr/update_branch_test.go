package pr

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestUpdateBranchCmd_ParsePRID(t *testing.T) {
	cmd := &UpdateBranchCmd{}

	tests := []struct {
		name      string
		prid      string
		expected  int
		expectErr bool
	}{
		{
			name:     "valid number",
			prid:     "123",
			expected: 123,
		},
		{
			name:     "valid number with hash prefix",
			prid:     "#456",
			expected: 456,
		},
		{
			name:      "empty string",
			prid:      "",
			expectErr: true,
		},
		{
			name:      "invalid number",
			prid:      "abc",
			expectErr: true,
		},
		{
			name:      "negative number",
			prid:      "-1",
			expectErr: true,
		},
		{
			name:      "zero",
			prid:      "0",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd.PRID = tt.prid
			result, err := ParsePRID(cmd.PRID)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestUpdateBranchCmd_ValidatePRState(t *testing.T) {
	cmd := &UpdateBranchCmd{}

	tests := []struct {
		name      string
		pr        *api.PullRequest
		expectErr bool
	}{
		{
			name: "open PR",
			pr: &api.PullRequest{
				ID:    123,
				State: string(api.PullRequestStateOpen),
			},
			expectErr: false,
		},
		{
			name: "merged PR",
			pr: &api.PullRequest{
				ID:    123,
				State: string(api.PullRequestStateMerged),
			},
			expectErr: true,
		},
		{
			name: "declined PR",
			pr: &api.PullRequest{
				ID:    123,
				State: string(api.PullRequestStateDeclined),
			},
			expectErr: true,
		},
		{
			name: "superseded PR",
			pr: &api.PullRequest{
				ID:    123,
				State: string(api.PullRequestStateSuperseded),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.validatePRState(tt.pr)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateBranchCmd_ExtractBranchNames(t *testing.T) {
	cmd := &UpdateBranchCmd{}

	tests := []struct {
		name           string
		pr             *api.PullRequest
		expectedSource string
		expectedTarget string
		expectErr      bool
	}{
		{
			name: "valid branches",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "feature/new-feature",
					},
				},
				Destination: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "main",
					},
				},
			},
			expectedSource: "feature/new-feature",
			expectedTarget: "main",
			expectErr:      false,
		},
		{
			name: "missing source branch",
			pr: &api.PullRequest{
				Source: nil,
				Destination: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "main",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "missing destination branch",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "feature/new-feature",
					},
				},
				Destination: nil,
			},
			expectErr: true,
		},
		{
			name: "empty source branch name",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "",
					},
				},
				Destination: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "main",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty destination branch name",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "feature/new-feature",
					},
				},
				Destination: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "nil source branch object",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: nil,
				},
				Destination: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "main",
					},
				},
			},
			expectErr: true,
		},
		{
			name: "nil destination branch object",
			pr: &api.PullRequest{
				Source: &api.PullRequestBranch{
					Branch: &api.Branch{
						Name: "feature/new-feature",
					},
				},
				Destination: &api.PullRequestBranch{
					Branch: nil,
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, target, err := cmd.extractBranchNames(tt.pr)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if source != tt.expectedSource {
				t.Errorf("expected source branch %q, got %q", tt.expectedSource, source)
			}

			if target != tt.expectedTarget {
				t.Errorf("expected target branch %q, got %q", tt.expectedTarget, target)
			}
		})
	}
}

func TestUpdateBranchCmd_FormatTable(t *testing.T) {
	cmd := &UpdateBranchCmd{}

	tests := []struct {
		name   string
		result *UpdateBranchResult
	}{
		{
			name: "successful update",
			result: &UpdateBranchResult{
				PRID:         123,
				Title:        "Add new feature",
				SourceBranch: "feature/new-feature",
				TargetBranch: "main",
				Success:      true,
				Message:      "Successfully updated PR #123 branch 'feature/new-feature' from 'main'",
			},
		},
		{
			name: "failed update with conflicts",
			result: &UpdateBranchResult{
				PRID:         456,
				Title:        "Fix bug",
				SourceBranch: "bugfix/issue-789",
				TargetBranch: "develop",
				Success:      false,
				HasConflicts: true,
				ConflictFiles: []string{
					"src/main.go",
					"README.md",
				},
				Message: "Failed to update PR #456 branch 'bugfix/issue-789' from 'develop': merge conflicts detected",
			},
		},
		{
			name: "successful update with files",
			result: &UpdateBranchResult{
				PRID:         789,
				Title:        "Update documentation",
				SourceBranch: "docs/update",
				TargetBranch: "main",
				Success:      true,
				FilesUpdated: []string{
					"docs/README.md",
					"docs/CHANGELOG.md",
				},
				Message: "Successfully updated PR #789 branch 'docs/update' from 'main'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.formatTable(tt.result)
			if err != nil {
				t.Errorf("formatTable failed: %v", err)
			}
		})
	}
}

func TestUpdateBranchResult_JSON(t *testing.T) {
	result := &UpdateBranchResult{
		PRID:         123,
		Title:        "Add new feature",
		SourceBranch: "feature/new-feature",
		TargetBranch: "main",
		Success:      true,
		Message:      "Successfully updated",
		HasConflicts: false,
	}

	_ = result
}

func TestUpdateBranchCmd_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Skip("integration tests not implemented yet")
}

func BenchmarkUpdateBranchCmd_ParsePRID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParsePRID("123")
	}
}

func BenchmarkUpdateBranchCmd_ExtractBranchNames(b *testing.B) {
	cmd := &UpdateBranchCmd{}
	pr := &api.PullRequest{
		Source: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: "feature/benchmark-test",
			},
		},
		Destination: &api.PullRequestBranch{
			Branch: &api.Branch{
				Name: "main",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = cmd.extractBranchNames(pr)
	}
}
