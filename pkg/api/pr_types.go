package api

import (
	"time"
)

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	Type                string                     `json:"type"`
	ID                  int                        `json:"id"`
	Title               string                     `json:"title"`
	Description         string                     `json:"description"`
	State               string                     `json:"state"`
	Author              *User                      `json:"author,omitempty"`
	Source              *PullRequestBranch         `json:"source,omitempty"`
	Destination         *PullRequestBranch         `json:"destination,omitempty"`
	MergeCommit         *Commit                    `json:"merge_commit,omitempty"`
	CommentCount        int                        `json:"comment_count"`
	TaskCount           int                        `json:"task_count"`
	CloseSourceBranch   bool                       `json:"close_source_branch"`
	ClosedBy            *User                      `json:"closed_by,omitempty"`
	Reason              string                     `json:"reason,omitempty"`
	CreatedOn           *time.Time                 `json:"created_on,omitempty"`
	UpdatedOn           *time.Time                 `json:"updated_on,omitempty"`
	Reviewers           []*PullRequestParticipant  `json:"reviewers,omitempty"`
	Participants        []*PullRequestParticipant  `json:"participants,omitempty"`
	Links               *PullRequestLinks          `json:"links,omitempty"`
	Summary             *PullRequestSummary        `json:"summary,omitempty"`
}

// PullRequestBranch represents a branch in a pull request (source or destination)
type PullRequestBranch struct {
	Branch     *Branch     `json:"branch,omitempty"`
	Commit     *Commit     `json:"commit,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
}

// Branch represents a git branch
type Branch struct {
	Type  string         `json:"type"`
	Name  string         `json:"name"`
	Links *Links         `json:"links,omitempty"`
}

// PullRequestParticipant represents a participant in a pull request
type PullRequestParticipant struct {
	Type             string     `json:"type"`
	User             *User      `json:"user,omitempty"`
	Role             string     `json:"role,omitempty"`
	Approved         bool       `json:"approved"`
	State            string     `json:"state,omitempty"`
	ParticipatedOn   *time.Time `json:"participated_on,omitempty"`
}

// PullRequestLinks represents links related to a pull request
type PullRequestLinks struct {
	Self         *Link `json:"self,omitempty"`
	HTML         *Link `json:"html,omitempty"`
	Diff         *Link `json:"diff,omitempty"`
	DiffStat     *Link `json:"diffstat,omitempty"`
	Comments     *Link `json:"comments,omitempty"`
	Activity     *Link `json:"activity,omitempty"`
	Merge        *Link `json:"merge,omitempty"`
	Decline      *Link `json:"decline,omitempty"`
	Approve      *Link `json:"approve,omitempty"`
	RequestChanges *Link `json:"request-changes,omitempty"`
}

// PullRequestSummary represents a summary of the pull request
type PullRequestSummary struct {
	Type   string `json:"type"`
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

// PullRequestComment represents a comment on a pull request
type PullRequestComment struct {
	Type      string                      `json:"type"`
	ID        int                         `json:"id"`
	Parent    *PullRequestComment         `json:"parent,omitempty"`
	Content   *PullRequestCommentContent  `json:"content,omitempty"`
	Inline    *PullRequestCommentInline   `json:"inline,omitempty"`
	User      *User                       `json:"user,omitempty"`
	CreatedOn *time.Time                  `json:"created_on,omitempty"`
	UpdatedOn *time.Time                  `json:"updated_on,omitempty"`
	Links     *PullRequestCommentLinks    `json:"links,omitempty"`
}

// PullRequestCommentContent represents the content of a comment
type PullRequestCommentContent struct {
	Type   string `json:"type"`
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

// PullRequestCommentInline represents inline comment metadata
type PullRequestCommentInline struct {
	Type string `json:"type"`
	Path string `json:"path"`
	From int    `json:"from,omitempty"`
	To   int    `json:"to,omitempty"`
}

// PullRequestCommentLinks represents links for a comment
type PullRequestCommentLinks struct {
	Self *Link `json:"self,omitempty"`
	HTML *Link `json:"html,omitempty"`
	Code *Link `json:"code,omitempty"`
}

// PullRequestDiff represents a diff for a pull request
type PullRequestDiff struct {
	Type    string `json:"type"`
	Raw     string `json:"raw"`
	Patch   string `json:"patch"`
	Unified string `json:"unified"`
}

// PullRequestFile represents a file changed in a pull request
type PullRequestFile struct {
	Type            string `json:"type"`
	Status          string `json:"status"`
	OldPath         string `json:"old_path,omitempty"`
	NewPath         string `json:"new_path,omitempty"`
	LinesAdded      int    `json:"lines_added"`
	LinesRemoved    int    `json:"lines_removed"`
	Binary          bool   `json:"binary"`
	Links           *Links `json:"links,omitempty"`
}

// PullRequestDiffStat represents diffstat information for a pull request
type PullRequestDiffStat struct {
	Type           string              `json:"type"`
	Status         string              `json:"status"`
	LinesAdded     int                 `json:"lines_added"`
	LinesRemoved   int                 `json:"lines_removed"`
	FilesChanged   int                 `json:"files_changed"`
	Files          []*PullRequestFile  `json:"files,omitempty"`
	Links          *Links              `json:"links,omitempty"`
}

// PullRequestApproval represents an approval on a pull request
type PullRequestApproval struct {
	Type        string     `json:"type"`
	User        *User      `json:"user,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
	PullRequest *PullRequest `json:"pullrequest,omitempty"`
}

// PullRequestMerge represents a merge operation for a pull request
type PullRequestMerge struct {
	Type                string `json:"type"`
	MergeStrategy       string `json:"merge_strategy,omitempty"`
	CloseSourceBranch   bool   `json:"close_source_branch"`
	Message             string `json:"message,omitempty"`
}

// PullRequestStateType represents the possible pull request states
type PullRequestStateType string

const (
	PullRequestStateOpen        PullRequestStateType = "OPEN"
	PullRequestStateMerged      PullRequestStateType = "MERGED"
	PullRequestStateDeclined    PullRequestStateType = "DECLINED"
	PullRequestStateSuperseded  PullRequestStateType = "SUPERSEDED"
)

// String returns the string representation of PullRequestStateType
func (p PullRequestStateType) String() string {
	return string(p)
}

// PullRequestParticipantRole represents the role of a participant
type PullRequestParticipantRole string

const (
	ParticipantRoleReviewer      PullRequestParticipantRole = "REVIEWER"
	ParticipantRoleParticipant   PullRequestParticipantRole = "PARTICIPANT"
)

// String returns the string representation of PullRequestParticipantRole
func (p PullRequestParticipantRole) String() string {
	return string(p)
}

// PullRequestParticipantState represents the state of a participant
type PullRequestParticipantState string

const (
	ParticipantStateApproved      PullRequestParticipantState = "approved"
	ParticipantStateChangesRequested PullRequestParticipantState = "changes_requested"
)

// String returns the string representation of PullRequestParticipantState
func (p PullRequestParticipantState) String() string {
	return string(p)
}

// PullRequestListOptions represents options for listing pull requests
type PullRequestListOptions struct {
	State    string `json:"state,omitempty"`    // OPEN, MERGED, DECLINED, SUPERSEDED
	Author   string `json:"author,omitempty"`   // Filter by author username
	Reviewer string `json:"reviewer,omitempty"` // Filter by reviewer username
	Sort     string `json:"sort,omitempty"`     // Sort field (created_on, updated_on, priority, title)
	Page     int    `json:"page,omitempty"`     // Page number
	PageLen  int    `json:"pagelen,omitempty"`  // Items per page
}

// CreatePullRequestRequest represents a request to create a pull request
type CreatePullRequestRequest struct {
	Type              string                     `json:"type"`
	Title             string                     `json:"title"`
	Description       string                     `json:"description,omitempty"`
	Source            *PullRequestBranch         `json:"source"`
	Destination       *PullRequestBranch         `json:"destination"`
	Reviewers         []*PullRequestParticipant  `json:"reviewers,omitempty"`
	CloseSourceBranch bool                       `json:"close_source_branch,omitempty"`
}

// UpdatePullRequestRequest represents a request to update a pull request
type UpdatePullRequestRequest struct {
	Type              string                     `json:"type,omitempty"`
	Title             string                     `json:"title,omitempty"`
	Description       string                     `json:"description,omitempty"`
	State             string                     `json:"state,omitempty"`
	Reviewers         []*PullRequestParticipant  `json:"reviewers,omitempty"`
	CloseSourceBranch *bool                      `json:"close_source_branch,omitempty"`
}

// AddCommentRequest represents a request to add a comment to a pull request
type AddCommentRequest struct {
	Type    string                      `json:"type"`
	Content *PullRequestCommentContent  `json:"content"`
	Inline  *PullRequestCommentInline   `json:"inline,omitempty"`
	Parent  *PullRequestComment         `json:"parent,omitempty"`
}

// RequestChangesRequest represents a request to request changes on a pull request
type RequestChangesRequest struct {
	Type    string                      `json:"type"`
	Content *PullRequestCommentContent  `json:"content"`
}

// PullRequestActivity represents activity on a pull request
type PullRequestActivity struct {
	Type        string                  `json:"type"`
	PullRequest *PullRequest            `json:"pull_request,omitempty"`
	Update      *PullRequestUpdate      `json:"update,omitempty"`
	Approval    *PullRequestApproval    `json:"approval,omitempty"`
	Comment     *PullRequestComment     `json:"comment,omitempty"`
	User        *User                   `json:"user,omitempty"`
	Date        *time.Time              `json:"date,omitempty"`
}

// PullRequestUpdate represents an update to a pull request
type PullRequestUpdate struct {
	Type        string     `json:"type"`
	Description string     `json:"description,omitempty"`
	Title       string     `json:"title,omitempty"`
	State       string     `json:"state,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	Author      *User      `json:"author,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
}

// PullRequestCheck represents a check (CI/build) on a pull request
type PullRequestCheck struct {
	Type        string     `json:"type"`
	UUID        string     `json:"uuid"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	URL         string     `json:"url,omitempty"`
	State       string     `json:"state"`
	Description string     `json:"description,omitempty"`
	CreatedOn   *time.Time `json:"created_on,omitempty"`
	UpdatedOn   *time.Time `json:"updated_on,omitempty"`
	Links       *Links     `json:"links,omitempty"`
}

// PullRequestCheckState represents the state of a check
type PullRequestCheckState string

const (
	CheckStateInProgress PullRequestCheckState = "INPROGRESS"
	CheckStateSuccessful PullRequestCheckState = "SUCCESSFUL"
	CheckStateFailed     PullRequestCheckState = "FAILED"
	CheckStateStopped    PullRequestCheckState = "STOPPED"
)

// String returns the string representation of PullRequestCheckState
func (p PullRequestCheckState) String() string {
	return string(p)
}