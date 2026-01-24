package pr

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestLockCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name     string
		prid     string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid PR ID",
			prid:     "123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "valid PR ID with hash prefix",
			prid:     "#456",
			expected: 456,
			wantErr:  false,
		},
		{
			name:     "empty PR ID",
			prid:     "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid PR ID - not a number",
			prid:     "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid PR ID - negative number",
			prid:     "-1",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid PR ID - zero",
			prid:     "0",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &LockCmd{
				PRID: tt.prid,
			}
			result, err := cmd.ParsePRID()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePRID() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ParsePRID() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ParsePRID() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestLockCmd_ValidatePRState(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{
			name:    "valid state - OPEN",
			state:   "OPEN",
			wantErr: false,
		},
		{
			name:    "valid state - MERGED",
			state:   "MERGED",
			wantErr: false,
		},
		{
			name:    "valid state - DECLINED",
			state:   "DECLINED",
			wantErr: false,
		},
		{
			name:    "valid state - SUPERSEDED",
			state:   "SUPERSEDED",
			wantErr: false,
		},
		{
			name:    "invalid state",
			state:   "INVALID",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &LockCmd{}
			pr := &api.PullRequest{
				ID:    123,
				State: tt.state,
			}

			err := cmd.validatePRState(pr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePRState() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validatePRState() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLockCmd_FormatTable(t *testing.T) {
	cmd := &LockCmd{
		Reason: "spam",
	}

	pr := &api.PullRequest{
		ID:    123,
		Title: "Test PR",
		State: "OPEN",
		Author: &api.User{
			DisplayName: "Test User",
			Username:    "testuser",
		},
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
	}

	err := cmd.formatTable(pr)
	if err != nil {
		t.Errorf("formatTable() unexpected error: %v", err)
	}
}
