package picker

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/git"
)

func TestCommitMatcher_FindMatches(t *testing.T) {
	sourceCommits := []git.PickCommit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug in login", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "refactor: improve performance", Date: "2024-01-03"},
	}

	targetCommits := []git.PickCommit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "xyz222", Author: "Other User", Message: "chore: update dependencies", Date: "2024-01-02"},
	}

	matcher := NewCommitMatcher()
	matches := matcher.FindMatches(sourceCommits, targetCommits)

	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 {
		match := matches[0]
		if match.Source.Hash != "abc123" {
			t.Errorf("Expected source hash abc123, got %s", match.Source.Hash)
		}
		if match.Target.Hash != "xyz111" {
			t.Errorf("Expected target hash xyz111, got %s", match.Target.Hash)
		}
	}
}

func TestCommitMatcher_GetUnmatched(t *testing.T) {
	sourceCommits := []git.PickCommit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug in login", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "refactor: improve performance", Date: "2024-01-03"},
	}

	targetCommits := []git.PickCommit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add new feature", Date: "2024-01-01"},
	}

	matcher := NewCommitMatcher()
	unmatched := matcher.GetUnmatched(sourceCommits, targetCommits)

	if len(unmatched) != 2 {
		t.Errorf("Expected 2 unmatched commits, got %d", len(unmatched))
	}

	expectedHashes := map[string]bool{"def456": true, "ghi789": true}
	for _, commit := range unmatched {
		if !expectedHashes[commit.Hash] {
			t.Errorf("Unexpected unmatched commit: %s", commit.Hash)
		}
	}
}

func TestSignatureMatching(t *testing.T) {
	commit1 := git.PickCommit{
		Hash:    "abc123",
		Author:  "Test User",
		Message: "feat: add new feature\n\nDetailed description here",
		Date:    "2024-01-01",
	}

	commit2 := git.PickCommit{
		Hash:    "xyz999",
		Author:  "Test User",
		Message: "feat: add new feature",
		Date:    "2024-01-01",
	}

	commit3 := git.PickCommit{
		Hash:    "def456",
		Author:  "Other User",
		Message: "feat: add new feature",
		Date:    "2024-01-01",
	}

	if commit1.Signature() != commit2.Signature() {
		t.Errorf("Expected commits with same content to have same signature")
		t.Logf("Commit1 signature: %s", commit1.Signature())
		t.Logf("Commit2 signature: %s", commit2.Signature())
	}

	if commit1.Signature() == commit3.Signature() {
		t.Errorf("Expected commits with different authors to have different signatures")
	}
}

func TestCommitMatcher_MatchingStrategies(t *testing.T) {
	tests := []struct {
		name        string
		source      git.PickCommit
		target      git.PickCommit
		shouldMatch bool
	}{
		{
			name: "exact signature match",
			source: git.PickCommit{
				Hash: "abc123", Author: "Test User",
				Message: "feat: add feature", Date: "2024-01-01",
			},
			target: git.PickCommit{
				Hash: "xyz999", Author: "Test User",
				Message: "feat: add feature", Date: "2024-01-01",
			},
			shouldMatch: true,
		},
		{
			name: "message-only match (same author, different date)",
			source: git.PickCommit{
				Hash: "abc123", Author: "Test User",
				Message: "fix: resolve issue", Date: "2024-01-01",
			},
			target: git.PickCommit{
				Hash: "xyz999", Author: "Test User",
				Message: "fix: resolve issue", Date: "2024-01-02",
			},
			shouldMatch: true,
		},
		{
			name: "no match - different author and message",
			source: git.PickCommit{
				Hash: "abc123", Author: "Test User",
				Message: "feat: add feature", Date: "2024-01-01",
			},
			target: git.PickCommit{
				Hash: "xyz999", Author: "Other User",
				Message: "fix: different change", Date: "2024-01-01",
			},
			shouldMatch: false,
		},
	}

	matcher := NewCommitMatcher()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCommits := []git.PickCommit{tt.source}
			targetCommits := []git.PickCommit{tt.target}

			matches := matcher.FindMatches(sourceCommits, targetCommits)

			if tt.shouldMatch {
				if len(matches) != 1 {
					t.Errorf("Expected 1 match, got %d", len(matches))
				}
			} else {
				if len(matches) != 0 {
					t.Errorf("Expected 0 matches, got %d", len(matches))
				}
			}
		})
	}
}

func TestFilterUnpickedCommits(t *testing.T) {
	prdCommits := []git.PickCommit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug", Date: "2024-01-02"},
		{Hash: "ghi789", Author: "Test User", Message: "docs: update readme", Date: "2024-01-03"},
	}

	hmlCommits := []git.PickCommit{
		{Hash: "xyz111", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"},
		{Hash: "xyz222", Author: "Other User", Message: "chore: unrelated", Date: "2024-01-04"},
	}

	unpicked := FilterUnpickedCommits(prdCommits, hmlCommits, false)

	if len(unpicked) != 2 {
		t.Errorf("Expected 2 unpicked commits, got %d", len(unpicked))
	}

	expectedMessages := map[string]bool{
		"fix: resolve bug":    true,
		"docs: update readme": true,
	}

	for _, commit := range unpicked {
		if !expectedMessages[commit.Message] {
			t.Errorf("Unexpected unpicked commit: %s", commit.Message)
		}
	}
}

func TestFilterUnpickedCommits_EmptyTarget(t *testing.T) {
	prdCommits := []git.PickCommit{
		{Hash: "abc123", Author: "Test User", Message: "feat: add feature", Date: "2024-01-01"},
		{Hash: "def456", Author: "Test User", Message: "fix: resolve bug", Date: "2024-01-02"},
	}

	var hmlCommits []git.PickCommit

	unpicked := FilterUnpickedCommits(prdCommits, hmlCommits, false)

	if len(unpicked) != 2 {
		t.Errorf("Expected 2 unpicked commits, got %d", len(unpicked))
	}
}

func TestGroupCommitsByMessage(t *testing.T) {
	commits := []git.PickCommit{
		{Hash: "a", Author: "User", Message: "feat: feature 1", Date: "2024-01-01"},
		{Hash: "b", Author: "User", Message: "fix: bug fix", Date: "2024-01-02"},
		{Hash: "c", Author: "User", Message: "feat: feature 2", Date: "2024-01-03"},
		{Hash: "d", Author: "User", Message: "docs: update docs", Date: "2024-01-04"},
		{Hash: "e", Author: "User", Message: "no prefix here", Date: "2024-01-05"},
	}

	groups := GroupCommitsByMessage(commits)

	if len(groups) != 4 {
		t.Errorf("Expected 4 groups, got %d", len(groups))
	}

	if groups[0].Title != "feat:" {
		t.Errorf("Expected first group title 'feat:', got %q", groups[0].Title)
	}
	if len(groups[0].Commits) != 2 {
		t.Errorf("Expected 2 commits in feat: group, got %d", len(groups[0].Commits))
	}
}

func TestSummarizeCommits(t *testing.T) {
	commits := []git.PickCommit{
		{Hash: "a", Author: "User A", Message: "feat: feature 1", Date: "2024-01-01"},
		{Hash: "b", Author: "User B", Message: "fix: bug fix", Date: "2024-01-02"},
		{Hash: "c", Author: "User A", Message: "feat: feature 2", Date: "2024-01-03"},
	}

	summary := SummarizeCommits(commits)

	if summary.Total != 3 {
		t.Errorf("Expected total 3, got %d", summary.Total)
	}

	if summary.ByAuthor["User A"] != 2 {
		t.Errorf("Expected 2 commits by User A, got %d", summary.ByAuthor["User A"])
	}

	if summary.ByType["feat:"] != 2 {
		t.Errorf("Expected 2 feat: commits, got %d", summary.ByType["feat:"])
	}
}
