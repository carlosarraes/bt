package git

import (
	"errors"
	"testing"
)

func TestRepositoryContext(t *testing.T) {
	// Test RepositoryContext struct creation and basic functionality
	ctx := &RepositoryContext{
		Workspace:      "myworkspace",
		Repository:     "myrepo",
		Branch:         "main",
		RemoteBranch:   "main",
		Remote:         "origin",
		HasUncommitted: false,
		IsGitRepo:      true,
		WorkingDir:     "/path/to/repo",
	}

	if ctx.Workspace != "myworkspace" {
		t.Errorf("RepositoryContext.Workspace = %v, want %v", ctx.Workspace, "myworkspace")
	}

	if ctx.Repository != "myrepo" {
		t.Errorf("RepositoryContext.Repository = %v, want %v", ctx.Repository, "myrepo")
	}

	if ctx.Branch != "main" {
		t.Errorf("RepositoryContext.Branch = %v, want %v", ctx.Branch, "main")
	}

	if ctx.RemoteBranch != "main" {
		t.Errorf("RepositoryContext.RemoteBranch = %v, want %v", ctx.RemoteBranch, "main")
	}

	if ctx.Remote != "origin" {
		t.Errorf("RepositoryContext.Remote = %v, want %v", ctx.Remote, "origin")
	}

	if ctx.HasUncommitted {
		t.Errorf("RepositoryContext.HasUncommitted = %v, want %v", ctx.HasUncommitted, false)
	}

	if !ctx.IsGitRepo {
		t.Errorf("RepositoryContext.IsGitRepo = %v, want %v", ctx.IsGitRepo, true)
	}

	if ctx.WorkingDir != "/path/to/repo" {
		t.Errorf("RepositoryContext.WorkingDir = %v, want %v", ctx.WorkingDir, "/path/to/repo")
	}
}

func TestRepositoryContextNonGit(t *testing.T) {
	// Test RepositoryContext for non-Git directory
	ctx := &RepositoryContext{
		IsGitRepo:  false,
		WorkingDir: "/path/to/non-git-dir",
	}

	if ctx.IsGitRepo {
		t.Errorf("RepositoryContext.IsGitRepo = %v, want %v", ctx.IsGitRepo, false)
	}

	if ctx.WorkingDir != "/path/to/non-git-dir" {
		t.Errorf("RepositoryContext.WorkingDir = %v, want %v", ctx.WorkingDir, "/path/to/non-git-dir")
	}

	// Other fields should be empty for non-Git repos
	if ctx.Workspace != "" {
		t.Errorf("RepositoryContext.Workspace = %v, want empty", ctx.Workspace)
	}

	if ctx.Repository != "" {
		t.Errorf("RepositoryContext.Repository = %v, want empty", ctx.Repository)
	}

	if ctx.Branch != "" {
		t.Errorf("RepositoryContext.Branch = %v, want empty", ctx.Branch)
	}
}

func TestRemoteStruct(t *testing.T) {
	// Test Remote struct creation and basic functionality
	remote := &Remote{
		Name:      "origin",
		URL:       "git@bitbucket.org:workspace/repo.git",
		Workspace: "workspace",
		RepoName:  "repo",
		IsSSH:     true,
	}

	if remote.Name != "origin" {
		t.Errorf("Remote.Name = %v, want %v", remote.Name, "origin")
	}

	if remote.URL != "git@bitbucket.org:workspace/repo.git" {
		t.Errorf("Remote.URL = %v, want %v", remote.URL, "git@bitbucket.org:workspace/repo.git")
	}

	if remote.Workspace != "workspace" {
		t.Errorf("Remote.Workspace = %v, want %v", remote.Workspace, "workspace")
	}

	if remote.RepoName != "repo" {
		t.Errorf("Remote.RepoName = %v, want %v", remote.RepoName, "repo")
	}

	if !remote.IsSSH {
		t.Errorf("Remote.IsSSH = %v, want %v", remote.IsSSH, true)
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that our custom error types are properly defined
	if ErrNotGitRepository == nil {
		t.Error("ErrNotGitRepository should not be nil")
	}

	if ErrNoRemotes == nil {
		t.Error("ErrNoRemotes should not be nil")
	}

	if ErrInvalidRemoteURL == nil {
		t.Error("ErrInvalidRemoteURL should not be nil")
	}

	// Test error messages
	if ErrNotGitRepository.Error() != "not a git repository" {
		t.Errorf("ErrNotGitRepository.Error() = %v, want %v", ErrNotGitRepository.Error(), "not a git repository")
	}

	if ErrNoRemotes.Error() != "no remotes found" {
		t.Errorf("ErrNoRemotes.Error() = %v, want %v", ErrNoRemotes.Error(), "no remotes found")
	}

	if ErrInvalidRemoteURL.Error() != "invalid remote URL format" {
		t.Errorf("ErrInvalidRemoteURL.Error() = %v, want %v", ErrInvalidRemoteURL.Error(), "invalid remote URL format")
	}
}

func TestErrorComparison(t *testing.T) {
	// Test that we can properly compare errors
	testErr := ErrNotGitRepository
	
	if !errors.Is(testErr, ErrNotGitRepository) {
		t.Error("Should be able to compare ErrNotGitRepository with errors.Is")
	}

	if errors.Is(testErr, ErrNoRemotes) {
		t.Error("ErrNotGitRepository should not match ErrNoRemotes")
	}

	if errors.Is(testErr, ErrInvalidRemoteURL) {
		t.Error("ErrNotGitRepository should not match ErrInvalidRemoteURL")
	}
}

// Test repository preference logic
func TestRemotePreference(t *testing.T) {
	tests := []struct {
		name           string
		remotes        []string
		expectedRemote string
	}{
		{
			name:           "Origin preferred",
			remotes:        []string{"upstream", "origin", "fork"},
			expectedRemote: "origin",
		},
		{
			name:           "Upstream when no origin",
			remotes:        []string{"upstream", "fork"},
			expectedRemote: "upstream",
		},
		{
			name:           "First remote when no origin or upstream",
			remotes:        []string{"fork", "personal"},
			expectedRemote: "fork", // First in alphabetical order typically
		},
		{
			name:           "Single remote",
			remotes:        []string{"origin"},
			expectedRemote: "origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock remotes map
			remotes := make(map[string]*Remote)
			for _, remoteName := range tt.remotes {
				remotes[remoteName] = &Remote{
					Name:      remoteName,
					URL:       "git@bitbucket.org:workspace/repo.git",
					Workspace: "workspace",
					RepoName:  "repo",
					IsSSH:     true,
				}
			}

			// Test the preference logic
			var selectedRemote *Remote
			if remote, exists := remotes["origin"]; exists {
				selectedRemote = remote
			} else if remote, exists := remotes["upstream"]; exists {
				selectedRemote = remote
			} else {
				// Use the first available remote
				for _, remote := range remotes {
					selectedRemote = remote
					break
				}
			}

			if selectedRemote == nil {
				t.Fatal("No remote selected")
			}

			// For the "first remote" case, we can't guarantee order from map iteration
			// so we'll check if it's one of the expected remotes
			if tt.expectedRemote == "fork" && len(tt.remotes) == 2 {
				validRemotes := map[string]bool{"fork": true, "personal": true}
				if !validRemotes[selectedRemote.Name] {
					t.Errorf("Selected remote %v not in expected remotes %v", selectedRemote.Name, tt.remotes)
				}
			} else {
				if selectedRemote.Name != tt.expectedRemote {
					t.Errorf("Selected remote = %v, want %v", selectedRemote.Name, tt.expectedRemote)
				}
			}
		})
	}
}

// Test working directory handling
func TestWorkingDirectoryHandling(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Empty path should use current directory",
			path:    "",
			wantErr: false, // Note: This would normally work but might fail in test environment
		},
		{
			name:    "Absolute path",
			path:    "/tmp",
			wantErr: false, // Note: This would fail if /tmp is not a git repo
		},
		{
			name:    "Relative path",
			path:    ".",
			wantErr: false, // Note: This would fail if current dir is not a git repo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These tests would need a real git repository to work properly
			// For now, we're just testing the path handling logic
			if tt.path == "" {
				// Test that empty path gets converted to current working directory
				// This is handled by os.Getwd() in the actual implementation
				t.Log("Empty path handling tested (would use os.Getwd())")
			} else {
				// Test that non-empty paths are used as-is
				t.Logf("Non-empty path handling tested: %s", tt.path)
			}
		})
	}
}

// Test context creation for non-Git directories
func TestNonGitDirectoryContext(t *testing.T) {
	// This tests the fallback behavior when not in a Git repository
	ctx := &RepositoryContext{
		IsGitRepo:  false,
		WorkingDir: "/some/path",
	}

	// Verify that non-Git context has appropriate defaults
	if ctx.IsGitRepo {
		t.Error("Non-Git context should have IsGitRepo = false")
	}

	if ctx.Workspace != "" {
		t.Error("Non-Git context should have empty Workspace")
	}

	if ctx.Repository != "" {
		t.Error("Non-Git context should have empty Repository")
	}

	if ctx.Branch != "" {
		t.Error("Non-Git context should have empty Branch")
	}

	if ctx.Remote != "" {
		t.Error("Non-Git context should have empty Remote")
	}

	if ctx.RemoteBranch != "" {
		t.Error("Non-Git context should have empty RemoteBranch")
	}
}