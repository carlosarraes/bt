package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type PickCommit struct {
	Hash    string
	Author  string
	Message string
	Date    string
}

func (c PickCommit) Signature() string {
	return fmt.Sprintf("%s:%s:%s", c.Author, c.Date, strings.Split(c.Message, "\n")[0])
}

type DateFilterType int

const (
	DateFilterToday DateFilterType = iota
	DateFilterYesterday
	DateFilterSince
	DateFilterUntil
	DateFilterRange
)

type DateFilter struct {
	Type  DateFilterType
	Since time.Time
	Until time.Time
}

func localMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func parseLocalDate(dateStr string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", dateStr, time.Local)
}

func NewTodayFilter() *DateFilter {
	today := localMidnight(time.Now())
	return &DateFilter{
		Type:  DateFilterToday,
		Since: today,
		Until: today.AddDate(0, 0, 1),
	}
}

func NewYesterdayFilter() *DateFilter {
	today := localMidnight(time.Now())
	return &DateFilter{
		Type:  DateFilterYesterday,
		Since: today.AddDate(0, 0, -1),
		Until: today,
	}
}

func GetCurrentBranchExec(repoDir string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetCurrentUserExec(repoDir string) (string, error) {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user name: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func BranchExistsExec(repoDir, branch string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoDir
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func BranchOrRemoteExists(repoDir, branch string, debug bool) (bool, error) {
	if exists, err := BranchExistsExec(repoDir, branch); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}

	fetchErr := FetchBranchesExec(repoDir, debug)

	remoteRef := fmt.Sprintf("origin/%s", branch)
	if exists, err := BranchExistsExec(repoDir, remoteRef); err != nil {
		return false, err
	} else if exists {
		if debug {
			fmt.Fprintf(os.Stderr, "Debug: Branch '%s' found on remote as '%s'\n", branch, remoteRef)
		}
		return true, nil
	}

	if fetchErr != nil {
		return false, fmt.Errorf("branch '%s' not found locally and fetch failed: %w", branch, fetchErr)
	}
	return false, nil
}

func FetchBranchesExec(repoDir string, debug bool, branches ...string) error {
	checkCmd := exec.Command("git", "remote", "get-url", "origin")
	checkCmd.Dir = repoDir
	if err := checkCmd.Run(); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "Debug: No origin remote found, skipping fetch\n")
		}
		return nil
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Fetching from origin...\n")
	}
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from origin: %w", err)
	}
	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Fetch completed successfully\n")
	}
	return nil
}

func GetPickCommits(repoDir, targetBranch, sourceBranch string, limit int, debug bool) ([]PickCommit, error) {
	sourceRef := resolveBranchRef(repoDir, sourceBranch, debug, "source")
	targetRef := resolveBranchRef(repoDir, targetBranch, debug, "target")

	args := []string{"log",
		fmt.Sprintf("^%s", targetRef),
		sourceRef,
		"--format=%h|%an|%s|%ad",
		"--date=short",
	}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Running git command: git %s\n", strings.Join(args, " "))
		fmt.Fprintf(os.Stderr, "Debug: sourceRef=%s, targetRef=%s\n", sourceRef, targetRef)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		if debug {
			if exitError, ok := err.(*exec.ExitError); ok {
				fmt.Fprintf(os.Stderr, "Debug: Git command failed with stderr: %s\n", string(exitError.Stderr))
			}
		}
		return nil, fmt.Errorf("failed to get commits with command 'git %s': %w", strings.Join(args, " "), err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Git command succeeded, output length: %d bytes\n", len(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []PickCommit{}, nil
	}

	commits := make([]PickCommit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		commits = append(commits, PickCommit{
			Hash:    parts[0],
			Author:  parts[1],
			Message: parts[2],
			Date:    parts[3],
		})
	}

	return commits, nil
}

func resolveBranchRef(repoDir, branch string, debug bool, label string) string {
	if localExists, _ := BranchExistsExec(repoDir, branch); localExists {
		if debug {
			fmt.Fprintf(os.Stderr, "Debug: Using local %s branch: %s\n", label, branch)
		}
		return branch
	}

	_ = FetchBranchesExec(repoDir, debug)

	remoteRef := fmt.Sprintf("origin/%s", branch)
	if remoteExists, _ := BranchExistsExec(repoDir, remoteRef); remoteExists {
		if debug {
			fmt.Fprintf(os.Stderr, "Debug: Local %s branch not found, using remote: %s\n", label, remoteRef)
		}
		return remoteRef
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Neither local nor remote %s branch found, using: %s\n", label, branch)
	}
	return branch
}

func FilterCommitsByAuthor(commits []PickCommit, author string) []PickCommit {
	filtered := make([]PickCommit, 0)
	for _, commit := range commits {
		if commit.Author == author {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

func FilterCommitsByDate(commits []PickCommit, filter *DateFilter) []PickCommit {
	if filter == nil {
		return commits
	}

	filtered := make([]PickCommit, 0)
	for _, commit := range commits {
		commitDate, err := parseLocalDate(commit.Date)
		if err != nil {
			continue
		}

		switch filter.Type {
		case DateFilterToday, DateFilterYesterday, DateFilterRange:
			if !commitDate.Before(filter.Since) && commitDate.Before(filter.Until) {
				filtered = append(filtered, commit)
			}
		case DateFilterSince:
			if !commitDate.Before(filter.Since) {
				filtered = append(filtered, commit)
			}
		case DateFilterUntil:
			if !commitDate.After(filter.Until) {
				filtered = append(filtered, commit)
			}
		}
	}

	return filtered
}

func CherryPickCommits(repoDir string, commitHashes []string) error {
	if len(commitHashes) == 0 {
		return nil
	}

	fmt.Printf("Cherry-picking %d commits...\n", len(commitHashes))

	oldestCommit := commitHashes[len(commitHashes)-1]
	newestCommit := commitHashes[0]
	commitRange := fmt.Sprintf("%s^..%s", oldestCommit, newestCommit)

	revListCmd := exec.Command("git", "rev-list", "--reverse", commitRange)
	revListCmd.Dir = repoDir
	revListOutput, err := revListCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit range: %v", err)
	}

	cherryPickCmd := exec.Command("git", "cherry-pick", "--stdin")
	cherryPickCmd.Dir = repoDir
	cherryPickCmd.Stdin = strings.NewReader(string(revListOutput))

	if err := cherryPickCmd.Run(); err != nil {
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = repoDir
		statusOutput, _ := statusCmd.Output()

		fmt.Println("\nConflicts found - needs to be resolved.")

		if len(statusOutput) > 0 {
			fmt.Printf("\nFiles with conflicts:\n%s", string(statusOutput))
		}

		fmt.Println("\nWhat to do:")
		fmt.Println("1. Resolve the conflicts in the files listed above")
		fmt.Println("2. Add the resolved files: git add <file>")
		fmt.Println("3. Continue: bt pick continue")
		fmt.Println("4. Or abort: git cherry-pick --abort")

		return nil
	}

	fmt.Println("Successfully cherry-picked all commits!")
	return nil
}

func IsCherryPickInProgress(repoDir string) bool {
	cherryPickHeadPath := fmt.Sprintf("%s/.git/CHERRY_PICK_HEAD", repoDir)
	_, err := os.Stat(cherryPickHeadPath)
	return !os.IsNotExist(err)
}

func CherryPickContinue(repoDir string) error {
	if !IsCherryPickInProgress(repoDir) {
		fmt.Println("No cherry-pick in progress. Nothing to continue.")
		return nil
	}

	fmt.Println("Continuing cherry-pick...")

	cmd := exec.Command("git", "cherry-pick", "--continue")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = repoDir
		statusOutput, _ := statusCmd.Output()

		if len(statusOutput) > 0 {
			fmt.Println("Still have conflicts to resolve:")
			fmt.Printf("%s", string(statusOutput))
			fmt.Println("\nResolve conflicts and run 'bt pick continue' again.")
		}

		return fmt.Errorf("cherry-pick continue failed: %v", err)
	}

	fmt.Println("Cherry-pick completed successfully!")
	return nil
}

func ParseBranchName(branchName, prefix, suffix string) (string, error) {
	if !strings.HasPrefix(branchName, prefix) {
		return "", fmt.Errorf("branch '%s' doesn't start with prefix '%s'", branchName, prefix)
	}

	if !strings.HasSuffix(branchName, suffix) {
		return "", fmt.Errorf("branch '%s' doesn't end with suffix '%s'", branchName, suffix)
	}

	branchIdentifier := strings.TrimPrefix(branchName, prefix)
	branchIdentifier = strings.TrimSuffix(branchIdentifier, suffix)

	if branchIdentifier == "" {
		return "", fmt.Errorf("empty branch identifier in branch name '%s'", branchName)
	}

	return branchIdentifier, nil
}

func FindBranchByPattern(repoDir, pattern string) (string, error) {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", fmt.Sprintf("refs/heads/%s", pattern))
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to search for branches matching pattern '%s': %w", pattern, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var matches []string
	for _, line := range lines {
		if line != "" {
			matches = append(matches, line)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no branches found matching pattern '%s'", pattern)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple branches found matching pattern '%s': %v", pattern, matches)
	}

	return matches[0], nil
}
