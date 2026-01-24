package git

import (
	"testing"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "Valid branch name",
			branchName: "feature/new-feature",
			wantErr:    false,
		},
		{
			name:       "Valid branch name with numbers",
			branchName: "hotfix-123",
			wantErr:    false,
		},
		{
			name:       "Valid branch name with underscores",
			branchName: "feature_branch",
			wantErr:    false,
		},
		{
			name:       "Valid branch name with dashes",
			branchName: "feature-branch",
			wantErr:    false,
		},
		{
			name:        "Empty branch name",
			branchName:  "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "Branch name with space",
			branchName:  "feature branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with tilde",
			branchName:  "feature~branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with caret",
			branchName:  "feature^branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with colon",
			branchName:  "feature:branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with question mark",
			branchName:  "feature?branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with asterisk",
			branchName:  "feature*branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with square bracket",
			branchName:  "feature[branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with backslash",
			branchName:  "feature\\branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with double dot",
			branchName:  "feature..branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with @{",
			branchName:  "feature@{branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name with double slash",
			branchName:  "feature//branch",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "Branch name starting with dash",
			branchName:  "-feature",
			wantErr:     true,
			errContains: "cannot start with '-'",
		},
		{
			name:        "Branch name ending with dot",
			branchName:  "feature.",
			wantErr:     true,
			errContains: "end with '.'",
		},
		{
			name:        "Branch name starting with slash",
			branchName:  "/feature",
			wantErr:     true,
			errContains: "start or end with '/'",
		},
		{
			name:        "Branch name ending with slash",
			branchName:  "feature/",
			wantErr:     true,
			errContains: "start or end with '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branchName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateBranchName() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateBranchName() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateBranchName() unexpected error = %v", err)
			}
		})
	}
}

func TestBranchInfo(t *testing.T) {
	// Test BranchInfo struct creation and basic functionality
	info := &BranchInfo{
		Name:         "refs/heads/feature/test",
		ShortName:    "feature/test",
		IsHead:       true,
		Remote:       "origin",
		RemoteBranch: "feature/test",
		Hash:         "abc123",
		IsTracking:   true,
	}

	if info.Name != "refs/heads/feature/test" {
		t.Errorf("BranchInfo.Name = %v, want %v", info.Name, "refs/heads/feature/test")
	}

	if info.ShortName != "feature/test" {
		t.Errorf("BranchInfo.ShortName = %v, want %v", info.ShortName, "feature/test")
	}

	if !info.IsHead {
		t.Errorf("BranchInfo.IsHead = %v, want %v", info.IsHead, true)
	}

	if info.Remote != "origin" {
		t.Errorf("BranchInfo.Remote = %v, want %v", info.Remote, "origin")
	}

	if info.RemoteBranch != "feature/test" {
		t.Errorf("BranchInfo.RemoteBranch = %v, want %v", info.RemoteBranch, "feature/test")
	}

	if info.Hash != "abc123" {
		t.Errorf("BranchInfo.Hash = %v, want %v", info.Hash, "abc123")
	}

	if !info.IsTracking {
		t.Errorf("BranchInfo.IsTracking = %v, want %v", info.IsTracking, true)
	}
}

func TestBranchStatus(t *testing.T) {
	// Test BranchStatus struct creation and basic functionality
	status := &BranchStatus{
		Branch:       "feature/test",
		Remote:       "origin",
		RemoteBranch: "feature/test",
		LocalHash:    "abc123",
		RemoteHash:   "def456",
		HasLocal:     true,
		HasRemote:    true,
		UpToDate:     false,
		Ahead:        2,
		Behind:       1,
		NeedsFetch:   false,
	}

	if status.Branch != "feature/test" {
		t.Errorf("BranchStatus.Branch = %v, want %v", status.Branch, "feature/test")
	}

	if status.Remote != "origin" {
		t.Errorf("BranchStatus.Remote = %v, want %v", status.Remote, "origin")
	}

	if status.RemoteBranch != "feature/test" {
		t.Errorf("BranchStatus.RemoteBranch = %v, want %v", status.RemoteBranch, "feature/test")
	}

	if status.LocalHash != "abc123" {
		t.Errorf("BranchStatus.LocalHash = %v, want %v", status.LocalHash, "abc123")
	}

	if status.RemoteHash != "def456" {
		t.Errorf("BranchStatus.RemoteHash = %v, want %v", status.RemoteHash, "def456")
	}

	if !status.HasLocal {
		t.Errorf("BranchStatus.HasLocal = %v, want %v", status.HasLocal, true)
	}

	if !status.HasRemote {
		t.Errorf("BranchStatus.HasRemote = %v, want %v", status.HasRemote, true)
	}

	if status.UpToDate {
		t.Errorf("BranchStatus.UpToDate = %v, want %v", status.UpToDate, false)
	}

	if status.Ahead != 2 {
		t.Errorf("BranchStatus.Ahead = %v, want %v", status.Ahead, 2)
	}

	if status.Behind != 1 {
		t.Errorf("BranchStatus.Behind = %v, want %v", status.Behind, 1)
	}

	if status.NeedsFetch {
		t.Errorf("BranchStatus.NeedsFetch = %v, want %v", status.NeedsFetch, false)
	}
}

// Test default branch detection logic
func TestDefaultBranchNames(t *testing.T) {
	defaultBranches := []string{"main", "master", "develop", "dev"}

	// Test that all expected default branch names are present
	expectedBranches := map[string]bool{
		"main":    true,
		"master":  true,
		"develop": true,
		"dev":     true,
	}

	for _, branch := range defaultBranches {
		if !expectedBranches[branch] {
			t.Errorf("Unexpected default branch name: %s", branch)
		}
		delete(expectedBranches, branch)
	}

	if len(expectedBranches) > 0 {
		t.Errorf("Missing expected default branch names: %v", expectedBranches)
	}
}

// Test branch name validation edge cases
func TestBranchNameEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{
			name:       "Single character",
			branchName: "a",
			wantErr:    false,
		},
		{
			name:       "Numbers only",
			branchName: "123",
			wantErr:    false,
		},
		{
			name:       "Mixed case",
			branchName: "Feature-Branch",
			wantErr:    false,
		},
		{
			name:       "With slashes (valid)",
			branchName: "feature/sub/branch",
			wantErr:    false,
		},
		{
			name:       "With dots (valid)",
			branchName: "v1.0.0",
			wantErr:    false,
		},
		{
			name:       "Long branch name",
			branchName: "very-long-branch-name-that-should-still-be-valid-according-to-git-rules",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branchName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBranchName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
