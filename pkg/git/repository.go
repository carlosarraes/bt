package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository represents a Git repository with Bitbucket context
type Repository struct {
	path      string
	repo      *git.Repository
	workspace string
	name      string
	remotes   map[string]*Remote
}

// RepositoryContext contains all Git repository information
type RepositoryContext struct {
	Workspace      string
	Repository     string
	Branch         string
	RemoteBranch   string
	Remote         string
	HasUncommitted bool
	IsGitRepo      bool
	WorkingDir     string
}

// Remote represents a Git remote with parsed Bitbucket information
type Remote struct {
	Name      string
	URL       string
	Workspace string
	RepoName  string
	IsSSH     bool
}

var (
	ErrNotGitRepository = errors.New("not a git repository")
	ErrNoRemotes        = errors.New("no remotes found")
	ErrInvalidRemoteURL = errors.New("invalid remote URL format")
)

// NewRepository creates a new Repository instance from the given path
func NewRepository(path string) (*Repository, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Find the Git repository
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrNotGitRepository
		}
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	r := &Repository{
		path:    workTree.Filesystem.Root(),
		repo:    repo,
		remotes: make(map[string]*Remote),
	}

	// Parse remotes
	if err := r.parseRemotes(); err != nil {
		return nil, fmt.Errorf("failed to parse remotes: %w", err)
	}

	// Set workspace and repository name from preferred remote
	if err := r.setRepositoryInfo(); err != nil {
		return nil, fmt.Errorf("failed to set repository info: %w", err)
	}

	return r, nil
}

// parseRemotes parses all remotes and extracts Bitbucket information
func (r *Repository) parseRemotes() error {
	remotes, err := r.repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	if len(remotes) == 0 {
		return ErrNoRemotes
	}

	for _, remote := range remotes {
		cfg := remote.Config()
		if len(cfg.URLs) == 0 {
			continue
		}

		// Parse the first URL for each remote
		parsedRemote, err := parseRemoteURL(cfg.Name, cfg.URLs[0])
		if err != nil {
			// Skip remotes that can't be parsed (might not be Bitbucket)
			continue
		}

		r.remotes[cfg.Name] = parsedRemote
	}

	if len(r.remotes) == 0 {
		return fmt.Errorf("no valid Bitbucket remotes found")
	}

	return nil
}

// setRepositoryInfo sets workspace and repository name from the preferred remote
func (r *Repository) setRepositoryInfo() error {
	// Prefer origin, then upstream, then any other remote
	var selectedRemote *Remote
	if remote, exists := r.remotes["origin"]; exists {
		selectedRemote = remote
	} else if remote, exists := r.remotes["upstream"]; exists {
		selectedRemote = remote
	} else {
		// Use the first available remote
		for _, remote := range r.remotes {
			selectedRemote = remote
			break
		}
	}

	if selectedRemote == nil {
		return fmt.Errorf("no suitable remote found")
	}

	r.workspace = selectedRemote.Workspace
	r.name = selectedRemote.RepoName

	return nil
}

// GetContext returns the complete repository context
func (r *Repository) GetContext() (*RepositoryContext, error) {
	ctx := &RepositoryContext{
		Workspace:  r.workspace,
		Repository: r.name,
		IsGitRepo:  true,
		WorkingDir: r.path,
	}

	// Get current branch
	head, err := r.repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			// Repository exists but has no commits yet
			ctx.Branch = "main" // Default branch name
		} else {
			return nil, fmt.Errorf("failed to get HEAD: %w", err)
		}
	} else {
		if head.Name().IsBranch() {
			ctx.Branch = head.Name().Short()
		}
	}

	// Get remote tracking information
	if ctx.Branch != "" {
		remoteBranch, remote, err := r.getRemoteTrackingInfo(ctx.Branch)
		if err == nil {
			ctx.RemoteBranch = remoteBranch
			ctx.Remote = remote
		}
	}

	// Check for uncommitted changes
	workTree, err := r.repo.Worktree()
	if err == nil {
		status, err := workTree.Status()
		if err == nil {
			ctx.HasUncommitted = !status.IsClean()
		}
	}

	return ctx, nil
}

// getRemoteTrackingInfo returns the remote tracking branch information
func (r *Repository) getRemoteTrackingInfo(branchName string) (string, string, error) {
	cfg, err := r.repo.Config()
	if err != nil {
		return "", "", err
	}

	// Look for branch configuration
	for _, branch := range cfg.Branches {
		if branch.Name == branchName {
			if branch.Remote != "" && branch.Merge != "" {
				// Extract branch name from merge ref
				remoteBranch := strings.TrimPrefix(string(branch.Merge), "refs/heads/")
				return remoteBranch, branch.Remote, nil
			}
		}
	}

	return "", "", fmt.Errorf("no remote tracking information found")
}

// GetWorkspace returns the workspace name
func (r *Repository) GetWorkspace() string {
	return r.workspace
}

// GetName returns the repository name
func (r *Repository) GetName() string {
	return r.name
}

// GetRemotes returns all parsed remotes
func (r *Repository) GetRemotes() map[string]*Remote {
	return r.remotes
}

// GetPath returns the repository path
func (r *Repository) GetPath() string {
	return r.path
}

// DetectRepository detects and returns Git repository context from the current directory
func DetectRepository() (*RepositoryContext, error) {
	repo, err := NewRepository("")
	if err != nil {
		if errors.Is(err, ErrNotGitRepository) {
			// Return context indicating this is not a Git repository
			wd, _ := os.Getwd()
			return &RepositoryContext{
				IsGitRepo:  false,
				WorkingDir: wd,
			}, nil
		}
		return nil, err
	}

	return repo.GetContext()
}

// IsGitRepository checks if the current directory is a Git repository
func IsGitRepository() bool {
	_, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	return err == nil
}

// FindRepositoryRoot finds the root directory of the Git repository
func FindRepositoryRoot(startPath string) (string, error) {
	if startPath == "" {
		var err error
		startPath, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	path := startPath
	for {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil {
			if info.IsDir() {
				return path, nil
			}
			// Handle .git file (worktree)
			if !info.IsDir() {
				return path, nil
			}
		}

		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}

	return "", ErrNotGitRepository
}

func (r *Repository) GetDiff(baseBranch, targetBranch string) (string, error) {
	baseHash, err := r.repo.ResolveRevision(plumbing.Revision(baseBranch))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base branch %s: %w", baseBranch, err)
	}

	targetHash, err := r.repo.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return "", fmt.Errorf("failed to resolve target branch %s: %w", targetBranch, err)
	}

	baseCommit, err := r.repo.CommitObject(*baseHash)
	if err != nil {
		return "", fmt.Errorf("failed to get base commit: %w", err)
	}

	targetCommit, err := r.repo.CommitObject(*targetHash)
	if err != nil {
		return "", fmt.Errorf("failed to get target commit: %w", err)
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get base tree: %w", err)
	}

	targetTree, err := targetCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get target tree: %w", err)
	}

	patch, err := targetTree.Patch(baseTree)
	if err != nil {
		return "", fmt.Errorf("failed to generate patch: %w", err)
	}

	return patch.String(), nil
}

func (r *Repository) GetChangedFiles(baseBranch, targetBranch string) ([]string, error) {
	baseHash, err := r.repo.ResolveRevision(plumbing.Revision(baseBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base branch %s: %w", baseBranch, err)
	}

	targetHash, err := r.repo.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target branch %s: %w", targetBranch, err)
	}

	baseCommit, err := r.repo.CommitObject(*baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}

	targetCommit, err := r.repo.CommitObject(*targetHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get target commit: %w", err)
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get base tree: %w", err)
	}

	targetTree, err := targetCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get target tree: %w", err)
	}

	patch, err := targetTree.Patch(baseTree)
	if err != nil {
		return nil, fmt.Errorf("failed to generate patch: %w", err)
	}

	var files []string
	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()
		if from != nil {
			files = append(files, from.Path())
		} else if to != nil {
			files = append(files, to.Path())
		}
	}

	return files, nil
}

func (r *Repository) GetCommitMessages(baseBranch, targetBranch string) ([]string, error) {
	baseHash, err := r.repo.ResolveRevision(plumbing.Revision(baseBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base branch %s: %w", baseBranch, err)
	}

	targetHash, err := r.repo.ResolveRevision(plumbing.Revision(targetBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target branch %s: %w", targetBranch, err)
	}

	commitIter, err := r.repo.Log(&git.LogOptions{
		From: *targetHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	var messages []string
	baseCommit, _ := r.repo.CommitObject(*baseHash)
	
	err = commitIter.ForEach(func(commit *object.Commit) error {
		if baseCommit != nil && commit.Hash == baseCommit.Hash {
			return nil
		}
		
		message := strings.Split(strings.TrimSpace(commit.Message), "\n")[0]
		messages = append(messages, message)
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return messages, nil
}
