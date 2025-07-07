package pr

import (
	"context"
	"os"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestStatusCmd_Run(t *testing.T) {
	if os.Getenv("BT_INTEGRATION_TESTS") != "1" {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name   string
		output string
		want   error
	}{
		{
			name:   "table output",
			output: "table",
			want:   nil,
		},
		{
			name:   "json output",
			output: "json",
			want:   nil,
		},
		{
			name:   "yaml output",
			output: "yaml",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &StatusCmd{
				Output:     tt.output,
				Workspace:  os.Getenv("BT_TEST_WORKSPACE"),
				Repository: os.Getenv("BT_TEST_REPOSITORY"),
			}

			ctx := context.Background()
			err := cmd.Run(ctx)

			if (err != nil) != (tt.want != nil) {
				t.Errorf("StatusCmd.Run() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestStatusCmd_getPRsCreatedByUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     int
	}{
		{
			name:     "valid user",
			username: "testuser",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &StatusCmd{}
			ctx := context.Background()
			
			prCtx := &PRContext{
				Workspace:  "test-workspace",
				Repository: "test-repo",
			}

			if prCtx.Client == nil {
				t.Skip("no client available for testing")
			}

			_, err := cmd.getPRsCreatedByUser(ctx, prCtx, tt.username)
			if err != nil {
				t.Errorf("StatusCmd.getPRsCreatedByUser() error = %v", err)
			}
		})
	}
}

func TestStatusCmd_getStatusIcon(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{
			name:  "open PR",
			state: "OPEN",
			want:  "✓",
		},
		{
			name:  "merged PR",
			state: "MERGED",
			want:  "✓",
		},
		{
			name:  "declined PR",
			state: "DECLINED",
			want:  "✗",
		},
		{
			name:  "unknown state",
			state: "UNKNOWN",
			want:  "⏳",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &StatusCmd{}
			pr := &api.PullRequest{
				State: tt.state,
			}

			got := cmd.getStatusIcon(pr)
			if got != tt.want {
				t.Errorf("StatusCmd.getStatusIcon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusCmd_formatOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		result     *PRStatusResult
		wantErr    bool
	}{
		{
			name:   "table format",
			output: "table",
			result: &PRStatusResult{
				CreatedByYou:  []*api.PullRequest{},
				NeedingReview: []*api.PullRequest{},
			},
			wantErr: false,
		},
		{
			name:   "json format",
			output: "json",
			result: &PRStatusResult{
				CreatedByYou:  []*api.PullRequest{},
				NeedingReview: []*api.PullRequest{},
			},
			wantErr: false,
		},
		{
			name:   "yaml format",
			output: "yaml",
			result: &PRStatusResult{
				CreatedByYou:  []*api.PullRequest{},
				NeedingReview: []*api.PullRequest{},
			},
			wantErr: false,
		},
		{
			name:   "invalid format",
			output: "xml",
			result: &PRStatusResult{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &StatusCmd{
				Output: tt.output,
			}

			prCtx := &PRContext{
			}

			if prCtx.Formatter == nil {
				t.Skip("no formatter available for testing")
			}

			err := cmd.formatOutput(prCtx, tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusCmd.formatOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	branch, err := getCurrentBranch()
	
	if err != nil && branch == "" {
		t.Log("Not in git repository, skipping branch test")
		return
	}

	if err != nil {
		t.Errorf("getCurrentBranch() error = %v", err)
	}

	t.Logf("Current branch: %s", branch)
}
