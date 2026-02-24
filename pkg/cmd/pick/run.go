package pick

import (
	"context"
	"fmt"
	"os"

	"github.com/carlosarraes/bt/pkg/git"
)

type RunCmd struct {
	Reverse     bool
	Latest      bool
	Count       int
	NoFilter    bool
	Today       bool
	Yesterday   bool
	Since       string
	Until       string
	Prefix      string
	SuffixPrd   string
	SuffixHml   string
	Debug       bool
	NoColor     bool
}

func (cmd *RunCmd) Run(ctx context.Context) error {
	pickCfg, err := loadPickConfig(cmd.Prefix, cmd.SuffixPrd, cmd.SuffixHml)
	if err != nil {
		return err
	}

	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	branches, err := resolveBranches(repoDir, pickCfg, cmd.Reverse, cmd.Debug)
	if err != nil {
		return err
	}

	opts := &commitFilterOpts{
		Count:     cmd.Count,
		Latest:    cmd.Latest,
		NoFilter:  cmd.NoFilter,
		Today:     cmd.Today,
		Yesterday: cmd.Yesterday,
		Since:     cmd.Since,
		Until:     cmd.Until,
		Debug:     cmd.Debug,
		ShowMode:  false,
	}

	commits, currentUser, err := getUnpickedCommits(repoDir, branches, opts)
	if err != nil {
		return err
	}
	if commits == nil {
		return nil
	}

	displayCommits(commits, currentUser, cmd.NoColor)

	var commitHashes []string
	for _, commit := range commits {
		commitHashes = append(commitHashes, commit.Hash)
	}

	if err := git.CherryPickCommits(repoDir, commitHashes); err != nil {
		return fmt.Errorf("cherry-pick failed: %w", err)
	}

	return nil
}
