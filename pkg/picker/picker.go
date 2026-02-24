package picker

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosarraes/bt/pkg/git"
)

type CommitMatch struct {
	Source git.PickCommit
	Target git.PickCommit
	Score  int
}

type CommitMatcher struct{}

func NewCommitMatcher() *CommitMatcher {
	return &CommitMatcher{}
}

func (cm *CommitMatcher) FindMatches(sourceCommits, targetCommits []git.PickCommit) []CommitMatch {
	var matches []CommitMatch

	for _, sourceCommit := range sourceCommits {
		for _, targetCommit := range targetCommits {
			if match, score := cm.matchCommits(sourceCommit, targetCommit); match {
				matches = append(matches, CommitMatch{
					Source: sourceCommit,
					Target: targetCommit,
					Score:  score,
				})
				break
			}
		}
	}

	return matches
}

func (cm *CommitMatcher) GetUnmatched(sourceCommits, targetCommits []git.PickCommit) []git.PickCommit {
	matches := cm.FindMatches(sourceCommits, targetCommits)
	matchedHashes := make(map[string]bool)

	for _, match := range matches {
		matchedHashes[match.Source.Hash] = true
	}

	var unmatched []git.PickCommit
	for _, commit := range sourceCommits {
		if !matchedHashes[commit.Hash] {
			unmatched = append(unmatched, commit)
		}
	}

	return unmatched
}

func (cm *CommitMatcher) matchCommits(source, target git.PickCommit) (bool, int) {
	if source.Signature() == target.Signature() {
		return true, 100
	}

	if cm.sameAuthorAndMessage(source, target) {
		return true, 80
	}

	return false, 0
}

func (cm *CommitMatcher) sameAuthorAndMessage(c1, c2 git.PickCommit) bool {
	if c1.Author != c2.Author {
		return false
	}

	msg1 := strings.Split(c1.Message, "\n")[0]
	msg2 := strings.Split(c2.Message, "\n")[0]

	return strings.TrimSpace(msg1) == strings.TrimSpace(msg2)
}

func FilterUnpickedCommits(sourceCommits, targetCommits []git.PickCommit, debug bool) []git.PickCommit {
	if len(targetCommits) == 0 {
		return sourceCommits
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Filtering %d source commits against %d target commits\n", len(sourceCommits), len(targetCommits))
	}

	matcher := NewCommitMatcher()
	matches := matcher.FindMatches(sourceCommits, targetCommits)

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: Found %d matches:\n", len(matches))
		for _, match := range matches {
			fmt.Fprintf(os.Stderr, "  Source: %s (%s) -> Target: %s (%s) [score: %d]\n",
				match.Source.Hash, match.Source.Message,
				match.Target.Hash, match.Target.Message,
				match.Score)
		}
	}

	unmatched := matcher.GetUnmatched(sourceCommits, targetCommits)

	if debug {
		fmt.Fprintf(os.Stderr, "Debug: %d unmatched commits remain\n", len(unmatched))
	}

	return unmatched
}

type CommitGroup struct {
	Title   string
	Commits []git.PickCommit
}

func GroupCommitsByMessage(commits []git.PickCommit) []CommitGroup {
	groups := make(map[string][]git.PickCommit)
	var order []string

	for _, commit := range commits {
		prefix := extractMessagePrefix(commit.Message)
		if _, exists := groups[prefix]; !exists {
			order = append(order, prefix)
		}
		groups[prefix] = append(groups[prefix], commit)
	}

	var result []CommitGroup
	for _, prefix := range order {
		result = append(result, CommitGroup{
			Title:   prefix,
			Commits: groups[prefix],
		})
	}

	return result
}

func extractMessagePrefix(message string) string {
	firstLine := strings.Split(message, "\n")[0]
	if strings.Contains(firstLine, ":") {
		parts := strings.SplitN(firstLine, ":", 2)
		return strings.TrimSpace(parts[0]) + ":"
	}
	return "other:"
}

type CommitSummary struct {
	Total    int
	ByAuthor map[string]int
	ByType   map[string]int
}

func SummarizeCommits(commits []git.PickCommit) CommitSummary {
	summary := CommitSummary{
		Total:    len(commits),
		ByAuthor: make(map[string]int),
		ByType:   make(map[string]int),
	}

	for _, commit := range commits {
		summary.ByAuthor[commit.Author]++
		prefix := extractMessagePrefix(commit.Message)
		summary.ByType[prefix]++
	}

	return summary
}
