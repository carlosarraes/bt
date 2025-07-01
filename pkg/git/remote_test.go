package git

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		remoteName  string
		remoteURL   string
		wantRemote  *Remote
		wantErr     bool
		errContains string
	}{
		{
			name:       "SSH format basic",
			remoteName: "origin",
			remoteURL:  "git@bitbucket.org:myworkspace/myrepo.git",
			wantRemote: &Remote{
				Name:      "origin",
				URL:       "git@bitbucket.org:myworkspace/myrepo.git",
				Workspace: "myworkspace",
				RepoName:  "myrepo",
				IsSSH:     true,
			},
		},
		{
			name:       "SSH format without .git",
			remoteName: "origin",
			remoteURL:  "git@bitbucket.org:workspace/repo",
			wantRemote: &Remote{
				Name:      "origin",
				URL:       "git@bitbucket.org:workspace/repo",
				Workspace: "workspace",
				RepoName:  "repo",
				IsSSH:     true,
			},
		},
		{
			name:       "HTTPS format basic",
			remoteName: "origin",
			remoteURL:  "https://bitbucket.org/myworkspace/myrepo.git",
			wantRemote: &Remote{
				Name:      "origin",
				URL:       "https://bitbucket.org/myworkspace/myrepo.git",
				Workspace: "myworkspace",
				RepoName:  "myrepo",
				IsSSH:     false,
			},
		},
		{
			name:       "HTTPS format with user",
			remoteName: "origin",
			remoteURL:  "https://user@bitbucket.org/workspace/repo.git",
			wantRemote: &Remote{
				Name:      "origin",
				URL:       "https://user@bitbucket.org/workspace/repo.git",
				Workspace: "workspace",
				RepoName:  "repo",
				IsSSH:     false,
			},
		},
		{
			name:       "HTTPS format without .git",
			remoteName: "upstream",
			remoteURL:  "https://bitbucket.org/workspace/repo",
			wantRemote: &Remote{
				Name:      "upstream",
				URL:       "https://bitbucket.org/workspace/repo",
				Workspace: "workspace",
				RepoName:  "repo",
				IsSSH:     false,
			},
		},
		{
			name:        "Empty URL",
			remoteName:  "origin",
			remoteURL:   "",
			wantErr:     true,
			errContains: "empty remote URL",
		},
		{
			name:        "Non-Bitbucket URL",
			remoteName:  "origin",
			remoteURL:   "https://github.com/user/repo.git",
			wantErr:     true,
			errContains: "not a Bitbucket URL",
		},
		{
			name:        "Invalid SSH format",
			remoteName:  "origin",
			remoteURL:   "git@github.com:user/repo.git",
			wantErr:     true,
			errContains: "not a Bitbucket URL",
		},
		{
			name:        "Invalid HTTPS format",
			remoteName:  "origin",
			remoteURL:   "https://bitbucket.org/workspace",
			wantErr:     true,
			errContains: "invalid Bitbucket URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRemoteURL(tt.remoteName, tt.remoteURL)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRemoteURL() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("parseRemoteURL() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseRemoteURL() unexpected error = %v", err)
				return
			}

			if !remoteEqual(got, tt.wantRemote) {
				t.Errorf("parseRemoteURL() = %+v, want %+v", got, tt.wantRemote)
			}
		})
	}
}

func TestParseBitbucketURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantWorkspace string
		wantRepo      string
		wantErr       bool
	}{
		{
			name:          "SSH format",
			url:           "git@bitbucket.org:workspace/repo.git",
			wantWorkspace: "workspace",
			wantRepo:      "repo",
		},
		{
			name:          "HTTPS format",
			url:           "https://bitbucket.org/workspace/repo.git",
			wantWorkspace: "workspace",
			wantRepo:      "repo",
		},
		{
			name:          "HTTPS with user",
			url:           "https://user@bitbucket.org/workspace/repo.git",
			wantWorkspace: "workspace",
			wantRepo:      "repo",
		},
		{
			name:          "Shorthand format",
			url:           "workspace/repo",
			wantWorkspace: "workspace",
			wantRepo:      "repo",
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			url:     "invalid-url",
			wantErr: true,
		},
		{
			name:    "GitHub URL",
			url:     "https://github.com/user/repo.git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo, err := ParseBitbucketURL(tt.url)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseBitbucketURL() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseBitbucketURL() unexpected error = %v", err)
				return
			}

			if workspace != tt.wantWorkspace {
				t.Errorf("ParseBitbucketURL() workspace = %v, want %v", workspace, tt.wantWorkspace)
			}

			if repo != tt.wantRepo {
				t.Errorf("ParseBitbucketURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestBuildBitbucketURL(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
		repo      string
		urlType   string
		want      string
	}{
		{
			name:      "SSH URL",
			workspace: "workspace",
			repo:      "repo",
			urlType:   "ssh",
			want:      "git@bitbucket.org:workspace/repo.git",
		},
		{
			name:      "HTTPS URL",
			workspace: "workspace",
			repo:      "repo",
			urlType:   "https",
			want:      "https://bitbucket.org/workspace/repo.git",
		},
		{
			name:      "Web URL",
			workspace: "workspace",
			repo:      "repo",
			urlType:   "web",
			want:      "https://bitbucket.org/workspace/repo",
		},
		{
			name:      "API URL",
			workspace: "workspace",
			repo:      "repo",
			urlType:   "api",
			want:      "https://api.bitbucket.org/2.0/repositories/workspace/repo",
		},
		{
			name:      "Default to HTTPS",
			workspace: "workspace",
			repo:      "repo",
			urlType:   "unknown",
			want:      "https://bitbucket.org/workspace/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBitbucketURL(tt.workspace, tt.repo, tt.urlType)
			if got != tt.want {
				t.Errorf("BuildBitbucketURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantErr   bool
	}{
		{
			name:      "Valid SSH URL",
			remoteURL: "git@bitbucket.org:workspace/repo.git",
			wantErr:   false,
		},
		{
			name:      "Valid HTTPS URL",
			remoteURL: "https://bitbucket.org/workspace/repo.git",
			wantErr:   false,
		},
		{
			name:      "Invalid URL",
			remoteURL: "not-a-url",
			wantErr:   true,
		},
		{
			name:      "GitHub URL",
			remoteURL: "https://github.com/user/repo.git",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemoteURL(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRemoteURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConvertToSSH(t *testing.T) {
	tests := []struct {
		name     string
		httpsURL string
		want     string
		wantErr  bool
	}{
		{
			name:     "Valid HTTPS URL",
			httpsURL: "https://bitbucket.org/workspace/repo.git",
			want:     "git@bitbucket.org:workspace/repo.git",
			wantErr:  false,
		},
		{
			name:     "HTTPS URL with user",
			httpsURL: "https://user@bitbucket.org/workspace/repo.git",
			want:     "git@bitbucket.org:workspace/repo.git",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			httpsURL: "not-a-url",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToSSH(tt.httpsURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToSSH() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConvertToSSH() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		sshURL  string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid SSH URL",
			sshURL:  "git@bitbucket.org:workspace/repo.git",
			want:    "https://bitbucket.org/workspace/repo.git",
			wantErr: false,
		},
		{
			name:    "SSH URL without .git",
			sshURL:  "git@bitbucket.org:workspace/repo",
			want:    "https://bitbucket.org/workspace/repo.git",
			wantErr: false,
		},
		{
			name:    "Invalid URL",
			sshURL:  "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToHTTPS(tt.sshURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToHTTPS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConvertToHTTPS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidBitbucketURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "Valid SSH URL",
			url:  "git@bitbucket.org:workspace/repo.git",
			want: true,
		},
		{
			name: "Valid HTTPS URL",
			url:  "https://bitbucket.org/workspace/repo.git",
			want: true,
		},
		{
			name: "Invalid URL",
			url:  "not-a-url",
			want: false,
		},
		{
			name: "GitHub URL",
			url:  "https://github.com/user/repo.git",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidBitbucketURL(tt.url)
			if got != tt.want {
				t.Errorf("IsValidBitbucketURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRemoteInfo(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		want        *RemoteInfo
		wantErr     bool
	}{
		{
			name:      "Valid SSH URL",
			remoteURL: "git@bitbucket.org:workspace/repo.git",
			want: &RemoteInfo{
				URL:        "git@bitbucket.org:workspace/repo.git",
				Workspace:  "workspace",
				RepoName:   "repo",
				IsSSH:      true,
				WebURL:     "https://bitbucket.org/workspace/repo",
				CloneSSH:   "git@bitbucket.org:workspace/repo.git",
				CloneHTTPS: "https://bitbucket.org/workspace/repo.git",
				APIURL:     "https://api.bitbucket.org/2.0/repositories/workspace/repo",
			},
			wantErr: false,
		},
		{
			name:      "Valid HTTPS URL",
			remoteURL: "https://bitbucket.org/workspace/repo.git",
			want: &RemoteInfo{
				URL:        "https://bitbucket.org/workspace/repo.git",
				Workspace:  "workspace",
				RepoName:   "repo",
				IsSSH:      false,
				WebURL:     "https://bitbucket.org/workspace/repo",
				CloneSSH:   "git@bitbucket.org:workspace/repo.git",
				CloneHTTPS: "https://bitbucket.org/workspace/repo.git",
				APIURL:     "https://api.bitbucket.org/2.0/repositories/workspace/repo",
			},
			wantErr: false,
		},
		{
			name:      "Invalid URL",
			remoteURL: "not-a-url",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRemoteInfo(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !remoteInfoEqual(got, tt.want) {
				t.Errorf("GetRemoteInfo() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Helper functions for testing
func remoteEqual(a, b *Remote) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name &&
		a.URL == b.URL &&
		a.Workspace == b.Workspace &&
		a.RepoName == b.RepoName &&
		a.IsSSH == b.IsSSH
}

func remoteInfoEqual(a, b *RemoteInfo) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.URL == b.URL &&
		a.Workspace == b.Workspace &&
		a.RepoName == b.RepoName &&
		a.IsSSH == b.IsSSH &&
		a.WebURL == b.WebURL &&
		a.CloneSSH == b.CloneSSH &&
		a.CloneHTTPS == b.CloneHTTPS &&
		a.APIURL == b.APIURL
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 indexOfString(s, substr) >= 0)))
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}