package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/git"
)

type PRContext = shared.CommandContext

// maxBranchLookupPages bounds the scan for a branch's pull request. The search
// exits on the first match, so this only matters for branches with no open PR.
const maxBranchLookupPages = 10

func PullRequestStateColor(state string) string {
	switch state {
	case "OPEN":
		return "green"
	case "MERGED":
		return "blue"
	case "DECLINED":
		return "red"
	case "SUPERSEDED":
		return "yellow"
	default:
		return "white"
	}
}

func handlePullRequestAPIError(err error) error {
	return shared.HandleAPIError(err, shared.DomainPullRequest)
}

func ParsePRID(prIDStr string) (int, error) {
	if prIDStr == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

	prIDStr = strings.TrimPrefix(prIDStr, "#")

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s': must be a number", prIDStr)
	}

	if prID <= 0 {
		return 0, fmt.Errorf("invalid pull request ID '%d': must be positive", prID)
	}

	return prID, nil
}

// CurrentBranch returns the branch checked out in the working directory.
func CurrentBranch() (string, error) {
	gitRepo, err := git.NewRepository("")
	if err != nil {
		return "", err
	}

	repoCtx, err := gitRepo.GetContext()
	if err != nil {
		return "", err
	}

	return repoCtx.Branch, nil
}

// FindPRForBranch returns the open pull request whose source branch is branchName,
// or nil when the branch has none.
func FindPRForBranch(ctx context.Context, prCtx *PRContext, branchName string) (*api.PullRequest, error) {
	options := &api.PullRequestListOptions{
		State:   "OPEN",
		Sort:    "-updated_on",
		PageLen: 50,
		Page:    1,
	}

	for {
		result, err := prCtx.Client.PullRequests.ListPullRequests(ctx, prCtx.Workspace, prCtx.Repository, options)
		if err != nil {
			return nil, handlePullRequestAPIError(err)
		}

		pullRequests, err := parsePullRequestResults(result)
		if err != nil {
			return nil, err
		}

		for _, pr := range pullRequests {
			if pr.Source != nil && pr.Source.Branch != nil && pr.Source.Branch.Name == branchName {
				return pr, nil
			}
		}

		if result.Next == "" || options.Page >= maxBranchLookupPages {
			return nil, nil
		}
		options.Page++
	}
}

// ResolvePRArg resolves a gh-style pull request argument: a number, a branch name,
// or empty meaning the pull request belonging to the current branch.
func ResolvePRArg(ctx context.Context, prCtx *PRContext, arg string) (int, error) {
	arg = strings.TrimSpace(arg)

	if arg != "" {
		if prID, err := ParsePRID(arg); err == nil {
			return prID, nil
		}
	}

	branch := arg
	if branch == "" {
		var err error
		if branch, err = CurrentBranch(); err != nil {
			return 0, fmt.Errorf("no pull request specified and could not determine the current branch: %w", err)
		}
	}

	pr, err := FindPRForBranch(ctx, prCtx, branch)
	if err != nil {
		return 0, err
	}
	if pr == nil {
		return 0, fmt.Errorf("no open pull request found for branch %q", branch)
	}

	return pr.ID, nil
}
