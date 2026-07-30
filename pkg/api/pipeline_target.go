package api

import (
	"fmt"
	"strconv"
	"strings"
)

// Pipeline target type discriminators returned by Bitbucket.
const (
	TargetTypePullRequest = "pipeline_pullrequest_target"
	TargetTypeBranch      = "pipeline_branch_target"
	TargetTypeRef         = "pipeline_ref_target"
)

// FlexibleID is a pipeline target's pull request ID. Bitbucket has returned this
// field as both a JSON number and a quoted string depending on the endpoint, so it
// accepts either. An unparseable value decodes to 0 rather than failing, because a
// malformed ID must not take down the decode of an otherwise usable pipeline list.
type FlexibleID int

func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(strings.Trim(strings.TrimSpace(string(data)), `"`))
	if raw == "" || raw == "null" {
		return nil
	}

	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	*f = FlexibleID(n)
	return nil
}

// IsPullRequest reports whether the pipeline was triggered by a pull request.
func (t *PipelineTarget) IsPullRequest() bool {
	if t == nil {
		return false
	}
	if t.Type == TargetTypePullRequest {
		return true
	}

	_, ok := t.PRNumber()
	return ok
}

// PRNumber returns the pull request number this pipeline belongs to.
func (t *PipelineTarget) PRNumber() (int, bool) {
	if t == nil {
		return 0, false
	}
	if t.PullRequest != nil && t.PullRequest.ID > 0 {
		return int(t.PullRequest.ID), true
	}
	if t.PullRequestId != nil && *t.PullRequestId > 0 {
		return *t.PullRequestId, true
	}
	return 0, false
}

// BranchName returns the branch the pipeline ran against, whichever shape the
// target uses: ref_name for branch/tag builds, source for pull request builds.
func (t *PipelineTarget) BranchName() string {
	if t == nil {
		return ""
	}
	if t.RefName != "" {
		return t.RefName
	}
	return t.Source
}

// DestinationBranch returns the PR's target branch, or "" for non-PR pipelines.
func (t *PipelineTarget) DestinationBranch() string {
	if t == nil {
		return ""
	}
	return t.Destination
}

// CommitHash returns the commit the pipeline built, or "" if unknown.
func (t *PipelineTarget) CommitHash() string {
	if t == nil || t.Commit == nil {
		return ""
	}
	return t.Commit.Hash
}

// DisplayRef renders the target for humans, e.g. "PR #312 feat/auth→main" or "main".
// Degrades gracefully as fields go missing rather than showing a bare placeholder.
func (t *PipelineTarget) DisplayRef() string {
	if t == nil {
		return "-"
	}

	branch := t.BranchName()
	if t.IsPullRequest() {
		label := "PR"
		if n, ok := t.PRNumber(); ok {
			label = fmt.Sprintf("PR #%d", n)
		}

		switch {
		case branch != "" && t.Destination != "":
			return fmt.Sprintf("%s %s→%s", label, branch, t.Destination)
		case branch != "":
			return fmt.Sprintf("%s %s", label, branch)
		default:
			return label
		}
	}

	if branch != "" {
		return branch
	}
	if t.Type == TargetTypeBranch {
		return "branch"
	}
	return "-"
}
