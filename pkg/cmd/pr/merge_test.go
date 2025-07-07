package pr

import (
	"context"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)


func TestMergeCmd_validateMergeability(t *testing.T) {
	cmd := &MergeCmd{}

	tests := []struct {
		name    string
		pr      *api.PullRequest
		wantErr bool
	}{
		{
			name: "open PR",
			pr: &api.PullRequest{
				ID:    123,
				State: "OPEN",
			},
			wantErr: false,
		},
		{
			name: "merged PR",
			pr: &api.PullRequest{
				ID:    123,
				State: "MERGED",
			},
			wantErr: true,
		},
		{
			name: "declined PR",
			pr: &api.PullRequest{
				ID:    123,
				State: "DECLINED",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.validateMergeability(tt.pr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMergeability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeCmd_getUserDisplayName(t *testing.T) {
	tests := []struct {
		name string
		user *api.User
		want string
	}{
		{
			name: "nil user",
			user: nil,
			want: "Unknown",
		},
		{
			name: "user with display name",
			user: &api.User{
				Username:    "johndoe",
				DisplayName: "John Doe",
			},
			want: "John Doe",
		},
		{
			name: "user with only username",
			user: &api.User{
				Username: "johndoe",
			},
			want: "johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getUserDisplayName(tt.user)
			if got != tt.want {
				t.Errorf("getUserDisplayName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeCmd_getBranchName(t *testing.T) {
	tests := []struct {
		name   string
		branch *api.PullRequestBranch
		want   string
	}{
		{
			name:   "nil branch",
			branch: nil,
			want:   "Unknown",
		},
		{
			name: "branch with nil branch field",
			branch: &api.PullRequestBranch{
				Branch: nil,
			},
			want: "Unknown",
		},
		{
			name: "valid branch",
			branch: &api.PullRequestBranch{
				Branch: &api.Branch{
					Name: "feature-branch",
				},
			},
			want: "feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBranchName(tt.branch)
			if got != tt.want {
				t.Errorf("getBranchName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeCmd_Run_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		cmd  *MergeCmd
		want string
	}{
		{
			name: "invalid PR ID",
			cmd: &MergeCmd{
				PRID: "invalid",
			},
			want: "invalid pull request ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Run(context.Background())
			if err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.want, err.Error())
			}
		})
	}
}

func TestMergeCmd_handleMergeAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "404 error",
			err: &api.BitbucketError{
				StatusCode: 404,
				Message:    "Not found",
			},
			want: "pull request not found or repository not accessible",
		},
		{
			name: "409 error",
			err: &api.BitbucketError{
				StatusCode: 409,
				Message:    "Conflict",
			},
			want: "pull request cannot be merged (conflicts or checks failed)",
		},
		{
			name: "422 error",
			err: &api.BitbucketError{
				StatusCode: 422,
				Message:    "Unprocessable Entity",
			},
			want: "pull request is not in a mergeable state",
		},
		{
			name: "generic API error",
			err: &api.BitbucketError{
				StatusCode: 500,
				Message:    "Internal Server Error",
			},
			want: "merge failed: Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleMergeAPIError(tt.err)
			if got.Error() != tt.want {
				t.Errorf("handleMergeAPIError() = %v, want %v", got.Error(), tt.want)
			}
		})
	}
}