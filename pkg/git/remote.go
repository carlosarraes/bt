package git

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

// Bitbucket URL patterns
var (
	// SSH format: git@bitbucket.org:workspace/repo.git
	sshPattern = regexp.MustCompile(`^git@bitbucket\.org:([^/]+)/([^/]+?)(?:\.git)?/?$`)
	
	// HTTPS formats:
	// https://bitbucket.org/workspace/repo.git
	// https://user@bitbucket.org/workspace/repo.git
	httpsPattern = regexp.MustCompile(`^https://(?:[^@]+@)?bitbucket\.org/([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

// parseRemoteURL parses a Git remote URL and extracts Bitbucket workspace and repository information
func parseRemoteURL(remoteName, remoteURL string) (*Remote, error) {
	if remoteURL == "" {
		return nil, fmt.Errorf("empty remote URL")
	}

	remote := &Remote{
		Name: remoteName,
		URL:  remoteURL,
	}

	// Try SSH format first
	if matches := sshPattern.FindStringSubmatch(remoteURL); matches != nil {
		remote.IsSSH = true
		remote.Workspace = matches[1]
		remote.RepoName = matches[2]
		return remote, nil
	}

	// Try HTTPS format
	if matches := httpsPattern.FindStringSubmatch(remoteURL); matches != nil {
		remote.IsSSH = false
		remote.Workspace = matches[1]
		remote.RepoName = matches[2]
		return remote, nil
	}

	// Try parsing as generic URL to provide better error messages
	if strings.Contains(remoteURL, "bitbucket.org") {
		return nil, fmt.Errorf("invalid Bitbucket URL format: %s", remoteURL)
	}

	return nil, fmt.Errorf("not a Bitbucket URL: %s", remoteURL)
}

// ParseBitbucketURL parses various Bitbucket URL formats and returns workspace and repository
func ParseBitbucketURL(urlStr string) (workspace, repo string, err error) {
	if urlStr == "" {
		return "", "", fmt.Errorf("empty URL")
	}

	// Clean up the URL
	urlStr = strings.TrimSpace(urlStr)

	// Handle SSH format
	if strings.HasPrefix(urlStr, "git@") {
		matches := sshPattern.FindStringSubmatch(urlStr)
		if matches == nil {
			return "", "", fmt.Errorf("invalid SSH URL format: %s", urlStr)
		}
		return matches[1], matches[2], nil
	}

	// Handle HTTPS format
	if strings.HasPrefix(urlStr, "https://") {
		matches := httpsPattern.FindStringSubmatch(urlStr)
		if matches == nil {
			return "", "", fmt.Errorf("invalid HTTPS URL format: %s", urlStr)
		}
		return matches[1], matches[2], nil
	}

	// Handle workspace/repo format (shorthand)
	if strings.Contains(urlStr, "/") && !strings.Contains(urlStr, "://") {
		parts := strings.Split(urlStr, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}

	return "", "", fmt.Errorf("unrecognized URL format: %s", urlStr)
}

// BuildBitbucketURL builds various Bitbucket URLs from workspace and repository
func BuildBitbucketURL(workspace, repo, urlType string) string {
	switch strings.ToLower(urlType) {
	case "ssh":
		return fmt.Sprintf("git@bitbucket.org:%s/%s.git", workspace, repo)
	case "https", "http":
		return fmt.Sprintf("https://bitbucket.org/%s/%s.git", workspace, repo)
	case "web", "browser":
		return fmt.Sprintf("https://bitbucket.org/%s/%s", workspace, repo)
	case "api":
		return fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s", workspace, repo)
	default:
		// Default to HTTPS clone URL
		return fmt.Sprintf("https://bitbucket.org/%s/%s.git", workspace, repo)
	}
}

// ValidateRemoteURL validates if a URL is a valid Bitbucket remote URL
func ValidateRemoteURL(remoteURL string) error {
	_, err := parseRemoteURL("test", remoteURL)
	return err
}

// ExtractRepositoryInfo extracts repository information from various URL formats
func ExtractRepositoryInfo(input string) (workspace, repo string, err error) {
	if input == "" {
		return "", "", fmt.Errorf("empty input")
	}

	input = strings.TrimSpace(input)

	// Try to parse as URL first
	workspace, repo, err = ParseBitbucketURL(input)
	if err == nil {
		return workspace, repo, nil
	}

	// If it looks like a URL but failed to parse, return the error
	if strings.Contains(input, "://") || strings.Contains(input, "@") {
		return "", "", err
	}

	// Try as workspace/repo format
	parts := strings.Split(input, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid format: expected 'workspace/repo' or valid Bitbucket URL")
}

// NormalizeRemoteURL normalizes a remote URL to a standard format
func NormalizeRemoteURL(remoteURL string, preferSSH bool) (string, error) {
	remote, err := parseRemoteURL("temp", remoteURL)
	if err != nil {
		return "", err
	}

	if preferSSH {
		return BuildBitbucketURL(remote.Workspace, remote.RepoName, "ssh"), nil
	}
	return BuildBitbucketURL(remote.Workspace, remote.RepoName, "https"), nil
}

// GetRemoteInfo returns detailed information about a remote URL
func GetRemoteInfo(remoteURL string) (*RemoteInfo, error) {
	remote, err := parseRemoteURL("", remoteURL)
	if err != nil {
		return nil, err
	}

	info := &RemoteInfo{
		URL:       remoteURL,
		Workspace: remote.Workspace,
		RepoName:  remote.RepoName,
		IsSSH:     remote.IsSSH,
		WebURL:    BuildBitbucketURL(remote.Workspace, remote.RepoName, "web"),
		CloneSSH:  BuildBitbucketURL(remote.Workspace, remote.RepoName, "ssh"),
		CloneHTTPS: BuildBitbucketURL(remote.Workspace, remote.RepoName, "https"),
		APIURL:    BuildBitbucketURL(remote.Workspace, remote.RepoName, "api"),
	}

	return info, nil
}

// RemoteInfo contains detailed information about a Bitbucket remote
type RemoteInfo struct {
	URL        string `json:"url"`
	Workspace  string `json:"workspace"`
	RepoName   string `json:"repo_name"`
	IsSSH      bool   `json:"is_ssh"`
	WebURL     string `json:"web_url"`
	CloneSSH   string `json:"clone_ssh"`
	CloneHTTPS string `json:"clone_https"`
	APIURL     string `json:"api_url"`
}

// IsValidBitbucketURL checks if a URL is a valid Bitbucket URL format
func IsValidBitbucketURL(urlStr string) bool {
	return sshPattern.MatchString(urlStr) || httpsPattern.MatchString(urlStr)
}

// ConvertToSSH converts an HTTPS Bitbucket URL to SSH format
func ConvertToSSH(httpsURL string) (string, error) {
	workspace, repo, err := ParseBitbucketURL(httpsURL)
	if err != nil {
		return "", err
	}
	return BuildBitbucketURL(workspace, repo, "ssh"), nil
}

// ConvertToHTTPS converts an SSH Bitbucket URL to HTTPS format
func ConvertToHTTPS(sshURL string) (string, error) {
	workspace, repo, err := ParseBitbucketURL(sshURL)
	if err != nil {
		return "", err
	}
	return BuildBitbucketURL(workspace, repo, "https"), nil
}

// ParseCloneURL parses a clone URL and returns normalized repository information
func ParseCloneURL(cloneURL string) (workspace, repo, protocol string, err error) {
	remote, err := parseRemoteURL("", cloneURL)
	if err != nil {
		return "", "", "", err
	}

	protocol = "https"
	if remote.IsSSH {
		protocol = "ssh"
	}

	return remote.Workspace, remote.RepoName, protocol, nil
}

func (r *Repository) AddRemote(name, url string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}
	if url == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	if _, exists := r.remotes[name]; exists {
		return fmt.Errorf("remote '%s' already exists", name)
	}

	remoteConfig := &config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	}

	_, err := r.repo.CreateRemote(remoteConfig)
	if err != nil {
		return fmt.Errorf("failed to add remote '%s': %w", name, err)
	}

	parsedRemote, err := parseRemoteURL(name, url)
	if err != nil {
		return nil
	}

	r.remotes[name] = parsedRemote
	return nil
}

func (r *Repository) RemoteExists(name string) bool {
	_, exists := r.remotes[name]
	return exists
}

func (r *Repository) GetRemote(name string) (*Remote, bool) {
	remote, exists := r.remotes[name]
	return remote, exists
}


func (r *Repository) FetchBranch(remoteName, branchName string) error {
	remote, err := r.repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote '%s': %w", remoteName, err)
	}

	refSpec := fmt.Sprintf("refs/heads/%s:refs/remotes/%s/%s", branchName, remoteName, branchName)
	err = remote.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{config.RefSpec(refSpec)},
	})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return fmt.Errorf("failed to fetch branch '%s' from remote '%s': %w", branchName, remoteName, err)
	}

	return nil
}
