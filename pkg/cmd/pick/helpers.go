package pick

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/git"
	"github.com/carlosarraes/bt/pkg/picker"
)

type commitFilterOpts struct {
	Count     int
	Latest    bool
	NoFilter  bool
	Today     bool
	Yesterday bool
	Since     string
	Until     string
	Debug     bool
	ShowMode  bool
}

type branchResult struct {
	Source        string
	Target        string
	CurrentBranch string
}

func loadPickConfig(flagPrefix, flagSuffixPrd, flagSuffixHml string) (*config.PickConfig, error) {
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	pickCfg := &cfg.Pick
	if flagPrefix != "" {
		pickCfg.Prefix = flagPrefix
	}
	if flagSuffixPrd != "" {
		pickCfg.SuffixPrd = flagSuffixPrd
	}
	if flagSuffixHml != "" {
		pickCfg.SuffixHml = flagSuffixHml
	}

	return pickCfg, nil
}

func resolveBranches(repoDir string, cfg *config.PickConfig, reverse, debug bool) (*branchResult, error) {
	currentBranch, err := git.GetCurrentBranchExec(repoDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	var currentSuffix string
	if strings.HasSuffix(currentBranch, cfg.SuffixPrd) {
		currentSuffix = cfg.SuffixPrd
	} else if strings.HasSuffix(currentBranch, cfg.SuffixHml) {
		currentSuffix = cfg.SuffixHml
	} else {
		return nil, fmt.Errorf("current branch '%s' doesn't end with PRD suffix '%s' or HML suffix '%s'",
			currentBranch, cfg.SuffixPrd, cfg.SuffixHml)
	}

	branchIdentifier, err := git.ParseBranchName(currentBranch, cfg.Prefix, currentSuffix)
	if err != nil {
		return nil, fmt.Errorf("failed to parse branch name: %w", err)
	}

	prdBranch := cfg.Prefix + branchIdentifier + cfg.SuffixPrd
	hmlBranch := cfg.Prefix + branchIdentifier + cfg.SuffixHml

	if exists, err := git.BranchOrRemoteExists(repoDir, prdBranch, debug); err != nil {
		return nil, fmt.Errorf("failed to check PRD branch: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("PRD branch '%s' does not exist (checked local and origin)", prdBranch)
	}

	if exists, err := git.BranchOrRemoteExists(repoDir, hmlBranch, debug); err != nil {
		return nil, fmt.Errorf("failed to check HML branch: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("HML branch '%s' does not exist (checked local and origin)", hmlBranch)
	}

	result := &branchResult{CurrentBranch: currentBranch}
	if reverse {
		result.Source = hmlBranch
		result.Target = prdBranch
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Printf("Source (HML) branch: %s\n", result.Source)
		fmt.Printf("Target (PRD) branch: %s\n", result.Target)
	} else {
		result.Source = prdBranch
		result.Target = hmlBranch
		fmt.Printf("Current branch: %s\n", currentBranch)
		fmt.Printf("Source (PRD) branch: %s\n", result.Source)
		fmt.Printf("Target (HML) branch: %s\n", result.Target)
	}

	return result, nil
}

func getUnpickedCommits(repoDir string, branches *branchResult, opts *commitFilterOpts) ([]git.PickCommit, string, error) {
	commitLimit := 100
	if opts.Latest {
		commitLimit = 100
	} else if opts.ShowMode {
		commitLimit = 0
	}

	sourceCommits, err := git.GetPickCommits(repoDir, branches.Target, branches.Source, commitLimit, opts.Debug)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get source commits: %w", err)
	}

	if len(sourceCommits) == 0 {
		fmt.Printf("No new commits found in %s branch.\n", branches.Source)
		return nil, "", nil
	}

	currentUser, err := git.GetCurrentUserExec(repoDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current user: %w", err)
	}

	userCommits := git.FilterCommitsByAuthor(sourceCommits, currentUser)

	if opts.Debug {
		fmt.Fprintf(os.Stderr, "Debug: Found %d commits from user %s in %s:\n", len(userCommits), currentUser, branches.Source)
		for i, commit := range userCommits {
			fmt.Fprintf(os.Stderr, "  %d. %s - %s - %s\n", i+1, commit.Hash, commit.Date, commit.Message)
		}
	}

	filteredCommits, err := applyDateFilter(userCommits, opts)
	if err != nil {
		return nil, "", err
	}

	if len(filteredCommits) == 0 {
		fmt.Println("No commits found for the current user with the specified filters.")
		return nil, "", nil
	}

	var unpickedCommits []git.PickCommit
	if opts.NoFilter {
		unpickedCommits = filteredCommits
		if opts.Debug {
			fmt.Fprintf(os.Stderr, "Debug: Using --no-filter, skipping smart deduplication\n")
		}
	} else {
		targetCommits, err := git.GetPickCommits(repoDir, "main", branches.Target, 100, opts.Debug)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get target commits: %w", err)
		}

		unpickedCommits = picker.FilterUnpickedCommits(filteredCommits, targetCommits, opts.Debug)
	}

	if len(unpickedCommits) == 0 {
		fmt.Printf("All commits have already been picked to %s branch.\n", branches.Target)
		return nil, "", nil
	}

	if !opts.Latest && opts.Count > 0 && len(unpickedCommits) > opts.Count {
		unpickedCommits = unpickedCommits[:opts.Count]
	}

	return unpickedCommits, currentUser, nil
}

func applyDateFilter(commits []git.PickCommit, opts *commitFilterOpts) ([]git.PickCommit, error) {
	if opts.Today {
		return git.FilterCommitsByDate(commits, git.NewTodayFilter()), nil
	}
	if opts.Yesterday {
		return git.FilterCommitsByDate(commits, git.NewYesterdayFilter()), nil
	}
	if opts.Since != "" {
		if err := validateDate(opts.Since); err != nil {
			return nil, fmt.Errorf("invalid since date: %w", err)
		}
		since, _ := time.Parse("2006-01-02", opts.Since)
		filter := &git.DateFilter{Type: git.DateFilterSince, Since: since}
		return git.FilterCommitsByDate(commits, filter), nil
	}
	if opts.Until != "" {
		if err := validateDate(opts.Until); err != nil {
			return nil, fmt.Errorf("invalid until date: %w", err)
		}
		until, _ := time.Parse("2006-01-02", opts.Until)
		filter := &git.DateFilter{Type: git.DateFilterUntil, Until: until}
		return git.FilterCommitsByDate(commits, filter), nil
	}
	return commits, nil
}

func validateDate(dateStr string) error {
	_, err := time.Parse("2006-01-02", dateStr)
	return err
}

var (
	indexStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	hashStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	ownAuthor    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	otherAuthor  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dateStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	msgDefault   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	msgFeat      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	msgFix       = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	msgDocs      = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	msgRefactor  = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func displayCommits(commits []git.PickCommit, currentUser string, noColor bool) {
	fmt.Printf("\nFound %d unpicked commits:\n", len(commits))
	for i, commit := range commits {
		displayCommit(i+1, commit, currentUser, noColor)
	}
}

func displayCommit(index int, commit git.PickCommit, currentUser string, noColor bool) {
	if noColor {
		fmt.Printf("%d. %s | %s | %s | %s\n", index, commit.Hash, commit.Author, commit.Date, commit.Message)
		return
	}

	authorStyle := ownAuthor
	if commit.Author != currentUser {
		authorStyle = otherAuthor
	}

	msgStyle := msgDefault
	switch {
	case strings.HasPrefix(commit.Message, "feat:"):
		msgStyle = msgFeat
	case strings.HasPrefix(commit.Message, "fix:"):
		msgStyle = msgFix
	case strings.HasPrefix(commit.Message, "docs:"):
		msgStyle = msgDocs
	case strings.HasPrefix(commit.Message, "refactor:"):
		msgStyle = msgRefactor
	}

	fmt.Printf("%s %s | %s | %s | %s\n",
		indexStyle.Render(fmt.Sprintf("%d.", index)),
		hashStyle.Render(commit.Hash),
		authorStyle.Render(commit.Author),
		dateStyle.Render(commit.Date),
		msgStyle.Render(commit.Message),
	)
}
