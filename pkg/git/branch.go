package git

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// BranchInfo contains information about a Git branch
type BranchInfo struct {
	Name         string `json:"name"`
	ShortName    string `json:"short_name"`
	IsHead       bool   `json:"is_head"`
	Remote       string `json:"remote,omitempty"`
	RemoteBranch string `json:"remote_branch,omitempty"`
	Hash         string `json:"hash"`
	IsTracking   bool   `json:"is_tracking"`
}

// GetCurrentBranch returns information about the current branch
func (r *Repository) GetCurrentBranch() (*BranchInfo, error) {
	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return nil, fmt.Errorf("HEAD is not pointing to a branch (detached HEAD)")
	}

	branchName := head.Name().Short()

	info := &BranchInfo{
		Name:      head.Name().String(),
		ShortName: branchName,
		IsHead:    true,
		Hash:      head.Hash().String(),
	}

	// Get remote tracking information
	remote, remoteBranch, err := r.getRemoteTrackingInfo(branchName)
	if err == nil {
		info.Remote = remote
		info.RemoteBranch = remoteBranch
		info.IsTracking = true
	}

	return info, nil
}

// GetAllBranches returns information about all local branches
func (r *Repository) GetAllBranches() ([]*BranchInfo, error) {
	branches, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var branchInfos []*BranchInfo
	var currentBranch string

	// Get current branch name
	if head, err := r.repo.Head(); err == nil && head.Name().IsBranch() {
		currentBranch = head.Name().Short()
	}

	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsBranch() {
			return nil
		}

		branchName := ref.Name().Short()
		isHead := branchName == currentBranch

		info := &BranchInfo{
			Name:      ref.Name().String(),
			ShortName: branchName,
			IsHead:    isHead,
			Hash:      ref.Hash().String(),
		}

		// Get remote tracking information
		remote, remoteBranch, err := r.getRemoteTrackingInfo(branchName)
		if err == nil {
			info.Remote = remote
			info.RemoteBranch = remoteBranch
			info.IsTracking = true
		}

		branchInfos = append(branchInfos, info)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branchInfos, nil
}

// GetRemoteBranches returns information about remote branches
func (r *Repository) GetRemoteBranches() ([]*BranchInfo, error) {
	refs, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var branchInfos []*BranchInfo

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsRemote() {
			return nil
		}

		// Parse remote branch name: refs/remotes/origin/branch-name
		parts := strings.Split(ref.Name().String(), "/")
		if len(parts) < 4 {
			return nil
		}

		remoteName := parts[2]
		branchName := strings.Join(parts[3:], "/")

		info := &BranchInfo{
			Name:         ref.Name().String(),
			ShortName:    branchName,
			Remote:       remoteName,
			RemoteBranch: branchName,
			Hash:         ref.Hash().String(),
			IsTracking:   false,
		}

		branchInfos = append(branchInfos, info)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate remote branches: %w", err)
	}

	return branchInfos, nil
}

// BranchExists checks if a local branch exists
func (r *Repository) BranchExists(branchName string) bool {
	ref := plumbing.NewBranchReferenceName(branchName)
	_, err := r.repo.Reference(ref, true)
	return err == nil
}

// RemoteBranchExists checks if a remote branch exists
func (r *Repository) RemoteBranchExists(remote, branchName string) bool {
	ref := plumbing.NewRemoteReferenceName(remote, branchName)
	_, err := r.repo.Reference(ref, true)
	return err == nil
}

// GetBranchCommitCount returns the number of commits on a branch
func (r *Repository) GetBranchCommitCount(branchName string) (int, error) {
	ref := plumbing.NewBranchReferenceName(branchName)
	branchRef, err := r.repo.Reference(ref, true)
	if err != nil {
		return 0, fmt.Errorf("branch not found: %s", branchName)
	}

	commitIter, err := r.repo.Log(&git.LogOptions{
		From: branchRef.Hash(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to count commits: %w", err)
	}

	return count, nil
}

// GetBranchStatus returns status information comparing local branch with remote
func (r *Repository) GetBranchStatus(branchName string) (*BranchStatus, error) {
	// Get local branch reference
	localRef := plumbing.NewBranchReferenceName(branchName)
	localBranchRef, err := r.repo.Reference(localRef, true)
	if err != nil {
		return nil, fmt.Errorf("local branch not found: %s", branchName)
	}

	status := &BranchStatus{
		Branch:    branchName,
		LocalHash: localBranchRef.Hash().String(),
		HasLocal:  true,
	}

	// Get remote tracking information
	remote, remoteBranch, err := r.getRemoteTrackingInfo(branchName)
	if err != nil {
		// No remote tracking, return local-only status
		return status, nil
	}

	status.Remote = remote
	status.RemoteBranch = remoteBranch

	// Get remote branch reference
	remoteRef := plumbing.NewRemoteReferenceName(remote, remoteBranch)
	remoteBranchRef, err := r.repo.Reference(remoteRef, true)
	if err != nil {
		// Remote branch doesn't exist locally (needs fetch)
		status.NeedsFetch = true
		return status, nil
	}

	status.RemoteHash = remoteBranchRef.Hash().String()
	status.HasRemote = true

	// Compare hashes
	if status.LocalHash == status.RemoteHash {
		status.UpToDate = true
	} else {
		// Count commits ahead/behind
		ahead, behind, err := r.countCommitsDifference(localBranchRef.Hash(), remoteBranchRef.Hash())
		if err == nil {
			status.Ahead = ahead
			status.Behind = behind
		}
	}

	return status, nil
}

// BranchStatus contains status information about a branch relative to its remote
type BranchStatus struct {
	Branch       string `json:"branch"`
	Remote       string `json:"remote,omitempty"`
	RemoteBranch string `json:"remote_branch,omitempty"`
	LocalHash    string `json:"local_hash"`
	RemoteHash   string `json:"remote_hash,omitempty"`
	HasLocal     bool   `json:"has_local"`
	HasRemote    bool   `json:"has_remote"`
	UpToDate     bool   `json:"up_to_date"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	NeedsFetch   bool   `json:"needs_fetch"`
}

// countCommitsDifference calculates how many commits ahead and behind localHash is from remoteHash
func (r *Repository) countCommitsDifference(localHash, remoteHash plumbing.Hash) (ahead, behind int, err error) {
	if localHash == remoteHash {
		return 0, 0, nil
	}

	// Get merge base to determine the common ancestor
	mergeBase, err := r.findMergeBase(localHash, remoteHash)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Count commits from merge base to local (ahead)
	ahead, err = r.countCommitsBetween(mergeBase, localHash)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count ahead commits: %w", err)
	}

	// Count commits from merge base to remote (behind)
	behind, err = r.countCommitsBetween(mergeBase, remoteHash)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count behind commits: %w", err)
	}

	return ahead, behind, nil
}

// findMergeBase finds the merge base between two commits
func (r *Repository) findMergeBase(hash1, hash2 plumbing.Hash) (plumbing.Hash, error) {
	// Simple implementation: if one is ancestor of the other, use it as merge base
	// For more complex cases, this would need a proper merge base algorithm

	// Check if hash1 is ancestor of hash2
	if isAncestor, _ := r.isAncestor(hash1, hash2); isAncestor {
		return hash1, nil
	}

	// Check if hash2 is ancestor of hash1
	if isAncestor, _ := r.isAncestor(hash2, hash1); isAncestor {
		return hash2, nil
	}

	// For now, return hash1 as a fallback
	// In a real implementation, you'd walk the commit history to find the actual merge base
	return hash1, nil
}

// isAncestor checks if ancestorHash is an ancestor of commitHash
func (r *Repository) isAncestor(ancestorHash, commitHash plumbing.Hash) (bool, error) {
	if ancestorHash == commitHash {
		return true, nil
	}

	commitIter, err := r.repo.Log(&git.LogOptions{
		From: commitHash,
	})
	if err != nil {
		return false, err
	}
	defer commitIter.Close()

	found := false
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Hash == ancestorHash {
			found = true
			return storer.ErrStop
		}
		return nil
	})

	return found, err
}

// countCommitsBetween counts commits between two hashes
func (r *Repository) countCommitsBetween(fromHash, toHash plumbing.Hash) (int, error) {
	if fromHash == toHash {
		return 0, nil
	}

	commitIter, err := r.repo.Log(&git.LogOptions{
		From: toHash,
	})
	if err != nil {
		return 0, err
	}
	defer commitIter.Close()

	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Hash == fromHash {
			return storer.ErrStop
		}
		count++
		return nil
	})

	if err != nil && err != storer.ErrStop {
		return 0, err
	}

	return count, nil
}

// GetDefaultBranch attempts to determine the default branch of the repository
func (r *Repository) GetDefaultBranch() (string, error) {
	// Try to get the default branch from remote HEAD
	for remoteName := range r.remotes {
		remoteHeadRef := plumbing.NewSymbolicReference(
			plumbing.NewRemoteHEADReferenceName(remoteName),
			plumbing.NewRemoteReferenceName(remoteName, "main"),
		)

		if ref, err := r.repo.Reference(remoteHeadRef.Name(), true); err == nil {
			if ref.Type() == plumbing.SymbolicReference {
				// Extract branch name from the target
				target := ref.Target()
				if target.IsRemote() {
					parts := strings.Split(target.String(), "/")
					if len(parts) >= 4 {
						return strings.Join(parts[3:], "/"), nil
					}
				}
			}
		}
	}

	// Fallback: check for common default branch names
	defaultBranches := []string{"main", "master", "develop", "dev"}
	for _, branchName := range defaultBranches {
		if r.BranchExists(branchName) {
			return branchName, nil
		}
	}

	// If no default branch found, return the first available branch
	branches, err := r.GetAllBranches()
	if err != nil {
		return "", err
	}

	if len(branches) > 0 {
		return branches[0].ShortName, nil
	}

	return "", fmt.Errorf("no branches found in repository")
}

// ValidateBranchName validates if a branch name is valid according to Git rules
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Basic Git branch name validation
	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "..", "@{", "//"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("branch name contains invalid character: %s", char)
		}
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("branch name cannot start with '-' or end with '.'")
	}

	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot start or end with '/'")
	}

	return nil
}

func (r *Repository) CheckoutBranch(branchName string, detached bool) error {
	if err := ValidateBranchName(branchName); err != nil {
		return err
	}

	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if detached {
		ref := plumbing.NewBranchReferenceName(branchName)
		branchRef, err := r.repo.Reference(ref, true)
		if err != nil {
			return fmt.Errorf("branch not found: %s", branchName)
		}

		err = workTree.Checkout(&git.CheckoutOptions{
			Hash: branchRef.Hash(),
		})
		if err != nil {
			return fmt.Errorf("failed to checkout branch in detached mode: %w", err)
		}
	} else {
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: !r.BranchExists(branchName),
		})
		if err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	return nil
}

func (r *Repository) CreateTrackingBranch(localBranch, remoteName, remoteBranch string) error {
	if err := ValidateBranchName(localBranch); err != nil {
		return err
	}

	if !r.RemoteBranchExists(remoteName, remoteBranch) {
		return fmt.Errorf("remote branch %s/%s does not exist", remoteName, remoteBranch)
	}

	remoteRef := plumbing.NewRemoteReferenceName(remoteName, remoteBranch)
	remoteBranchRef, err := r.repo.Reference(remoteRef, true)
	if err != nil {
		return fmt.Errorf("failed to get remote branch reference: %w", err)
	}

	localRef := plumbing.NewBranchReferenceName(localBranch)
	newRef := plumbing.NewHashReference(localRef, remoteBranchRef.Hash())

	err = r.repo.Storer.SetReference(newRef)
	if err != nil {
		return fmt.Errorf("failed to create local branch: %w", err)
	}

	cfg, err := r.repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repository config: %w", err)
	}

	cfg.Branches[localBranch] = &config.Branch{
		Name:   localBranch,
		Remote: remoteName,
		Merge:  plumbing.NewBranchReferenceName(remoteBranch),
	}

	r.repo.Storer.SetConfig(cfg)

	return nil
}

func (r *Repository) ForceCheckoutBranch(branchName string, detached bool) error {
	if err := ValidateBranchName(branchName); err != nil {
		return err
	}

	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if detached {
		ref := plumbing.NewBranchReferenceName(branchName)
		branchRef, err := r.repo.Reference(ref, true)
		if err != nil {
			return fmt.Errorf("branch not found: %s", branchName)
		}

		err = workTree.Checkout(&git.CheckoutOptions{
			Hash:  branchRef.Hash(),
			Force: true,
		})
		if err != nil {
			return fmt.Errorf("failed to force checkout branch in detached mode: %w", err)
		}
	} else {
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branchName),
			Create: !r.BranchExists(branchName),
			Force:  true,
		})
		if err != nil {
			return fmt.Errorf("failed to force checkout branch: %w", err)
		}
	}

	return nil
}

func (r *Repository) HasUncommittedChanges() (bool, error) {
	workTree, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := workTree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return !status.IsClean(), nil
}

type MergeResult struct {
	Success       bool     `json:"success"`
	HasConflicts  bool     `json:"has_conflicts"`
	ConflictFiles []string `json:"conflict_files,omitempty"`
	Message       string   `json:"message"`
	CommitHash    string   `json:"commit_hash,omitempty"`
}

func (r *Repository) MergeBranch(sourceBranch, targetBranch string, force bool) (*MergeResult, error) {
	if err := ValidateBranchName(sourceBranch); err != nil {
		return nil, fmt.Errorf("invalid source branch name: %w", err)
	}
	if err := ValidateBranchName(targetBranch); err != nil {
		return nil, fmt.Errorf("invalid target branch name: %w", err)
	}

	if !r.BranchExists(sourceBranch) {
		return nil, fmt.Errorf("source branch '%s' does not exist", sourceBranch)
	}
	if !r.BranchExists(targetBranch) {
		return nil, fmt.Errorf("target branch '%s' does not exist", targetBranch)
	}

	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	if !force {
		hasChanges, err := r.HasUncommittedChanges()
		if err != nil {
			return nil, fmt.Errorf("failed to check for uncommitted changes: %w", err)
		}
		if hasChanges {
			return nil, fmt.Errorf("uncommitted changes detected. Use --force to override or commit your changes first")
		}
	}

	if currentBranch.ShortName != targetBranch {
		if err := r.CheckoutBranch(targetBranch, false); err != nil {
			return nil, fmt.Errorf("failed to checkout target branch '%s': %w", targetBranch, err)
		}
	}

	sourceBranchRef := plumbing.NewBranchReferenceName(sourceBranch)
	sourceRef, err := r.repo.Reference(sourceBranchRef, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get source branch reference: %w", err)
	}

	sourceCommit, err := r.repo.CommitObject(sourceRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get source commit: %w", err)
	}

	result := &MergeResult{
		Success: true,
		Message: fmt.Sprintf("Would merge '%s' into '%s' (simplified implementation)", sourceBranch, targetBranch),
	}

	if sourceCommit != nil {
		result.Message += fmt.Sprintf(" (commit: %s)", sourceCommit.Hash.String()[:8])
	}

	return result, nil
}

func (r *Repository) FetchRemote(remoteName string) error {
	if remoteName == "" {
		remoteName = "origin"
	}

	remote, err := r.repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote '%s': %w", remoteName, err)
	}

	err = remote.Fetch(&git.FetchOptions{})
	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil
		}
		return fmt.Errorf("failed to fetch from remote '%s': %w", remoteName, err)
	}

	return nil
}

func (r *Repository) PullBranch(branchName string, force bool) (*MergeResult, error) {
	if err := ValidateBranchName(branchName); err != nil {
		return nil, fmt.Errorf("invalid branch name: %w", err)
	}

	remote, remoteBranch, err := r.getRemoteTrackingInfo(branchName)
	if err != nil {
		return nil, fmt.Errorf("branch '%s' has no remote tracking information: %w", branchName, err)
	}

	if err := r.FetchRemote(remote); err != nil {
		return nil, fmt.Errorf("failed to fetch from remote: %w", err)
	}

	if !r.RemoteBranchExists(remote, remoteBranch) {
		return nil, fmt.Errorf("remote branch '%s/%s' does not exist", remote, remoteBranch)
	}

	currentBranch, err := r.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	if currentBranch.ShortName != branchName {
		if err := r.CheckoutBranch(branchName, false); err != nil {
			return nil, fmt.Errorf("failed to checkout branch '%s': %w", branchName, err)
		}
	}

	remoteBranchName := fmt.Sprintf("%s/%s", remote, remoteBranch)

	return r.MergeBranch(remoteBranchName, branchName, force)
}
