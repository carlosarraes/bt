package api

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// PullRequestService provides pull request-related API operations
type PullRequestService struct {
	client *Client
}

// NewPullRequestService creates a new pull request service
func NewPullRequestService(client *Client) *PullRequestService {
	return &PullRequestService{
		client: client,
	}
}

// ListPullRequests retrieves a paginated list of pull requests for a repository
func (p *PullRequestService) ListPullRequests(ctx context.Context, workspace, repoSlug string, options *PullRequestListOptions) (*PaginatedResponse, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", workspace, repoSlug)
	
	// Add query parameters if options are provided
	if options != nil {
		queryParams := url.Values{}
		var filterParts []string
		
		if options.State != "" {
			filterParts = append(filterParts, fmt.Sprintf("state=\"%s\"", options.State))
		}
		
		if options.Author != "" {
			filterParts = append(filterParts, fmt.Sprintf("author.username=\"%s\"", options.Author))
		}
		
		if options.Reviewer != "" {
			filterParts = append(filterParts, fmt.Sprintf("reviewers.username=\"%s\"", options.Reviewer))
		}
		
		if len(filterParts) > 0 {
			queryParams.Set("q", strings.Join(filterParts, " AND "))
		}
		
		if options.Sort != "" {
			queryParams.Set("sort", options.Sort)
		}
		
		if encodedParams := queryParams.Encode(); encodedParams != "" {
			endpoint += "?" + encodedParams
		}
	}

	// Use the paginator for large result sets
	pageOptions := &PageOptions{
		Page:    1,
		PageLen: 50,
	}
	
	if options != nil {
		if options.Page > 0 {
			pageOptions.Page = options.Page
		}
		if options.PageLen > 0 {
			pageOptions.PageLen = options.PageLen
		}
	}
	
	paginator := p.client.Paginate(endpoint, pageOptions)

	return paginator.NextPage(ctx)
}

// GetPullRequest retrieves detailed information about a specific pull request
func (p *PullRequestService) GetPullRequest(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d", workspace, repoSlug, id)
	
	var pullRequest PullRequest
	err := p.client.GetJSON(ctx, endpoint, &pullRequest)
	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

// GetPullRequestDiff retrieves the unified diff for a pull request
func (p *PullRequestService) GetPullRequestDiff(ctx context.Context, workspace, repoSlug string, id int) (string, error) {
	if workspace == "" || repoSlug == "" {
		return "", NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return "", NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/diff", workspace, repoSlug, id)
	
	resp, err := p.client.Get(ctx, endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the raw diff content
	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff content: %w", err)
	}

	return string(diffBytes), nil
}

// GetPullRequestFiles retrieves the diffstat (list of changed files) for a pull request
func (p *PullRequestService) GetPullRequestFiles(ctx context.Context, workspace, repoSlug string, id int) (*PullRequestDiffStat, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/diffstat", workspace, repoSlug, id)
	
	var diffStat PullRequestDiffStat
	err := p.client.GetJSON(ctx, endpoint, &diffStat)
	if err != nil {
		return nil, err
	}

	return &diffStat, nil
}

// ApprovePullRequest approves a pull request
func (p *PullRequestService) ApprovePullRequest(ctx context.Context, workspace, repoSlug string, id int) (*PullRequestApproval, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, id)
	
	var approval PullRequestApproval
	err := p.client.PostJSON(ctx, endpoint, nil, &approval)
	if err != nil {
		return nil, err
	}

	return &approval, nil
}

// UnapproveePullRequest removes approval from a pull request
func (p *PullRequestService) UnapprovePullRequest(ctx context.Context, workspace, repoSlug string, id int) error {
	if workspace == "" || repoSlug == "" {
		return NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, id)
	
	resp, err := p.client.Delete(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// RequestChanges requests changes on a pull request with a required comment
func (p *PullRequestService) RequestChanges(ctx context.Context, workspace, repoSlug string, id int, comment string) (*PullRequestComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}
	
	if comment == "" {
		return nil, NewValidationError("comment is required when requesting changes", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/request-changes", workspace, repoSlug, id)
	
	request := &RequestChangesRequest{
		Type: "pullrequest_comment",
		Content: &PullRequestCommentContent{
			Type: "text",
			Raw:  comment,
		},
	}

	var result PullRequestComment
	err := p.client.PostJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// AddComment adds a comment to a pull request
func (p *PullRequestService) AddComment(ctx context.Context, workspace, repoSlug string, id int, comment string, inline *PullRequestCommentInline) (*PullRequestComment, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}
	
	if comment == "" {
		return nil, NewValidationError("comment content is required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, id)
	
	request := &AddCommentRequest{
		Type: "pullrequest_comment",
		Content: &PullRequestCommentContent{
			Type: "text",
			Raw:  comment,
		},
		Inline: inline,
	}

	var result PullRequestComment
	err := p.client.PostJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetComments retrieves comments for a pull request
func (p *PullRequestService) GetComments(ctx context.Context, workspace, repoSlug string, id int) (*PaginatedResponse, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, id)
	
	// Use the paginator for potentially large comment lists
	paginator := p.client.Paginate(endpoint, nil)
	return paginator.NextPage(ctx)
}

// CreatePullRequest creates a new pull request
func (p *PullRequestService) CreatePullRequest(ctx context.Context, workspace, repoSlug string, request *CreatePullRequestRequest) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if request == nil {
		return nil, NewValidationError("create request is required", "")
	}
	
	if request.Title == "" {
		return nil, NewValidationError("pull request title is required", "")
	}
	
	if request.Source == nil || request.Destination == nil {
		return nil, NewValidationError("source and destination branches are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests", workspace, repoSlug)
	
	// Ensure the type is set
	if request.Type == "" {
		request.Type = "pullrequest"
	}

	var result PullRequest
	err := p.client.PostJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdatePullRequest updates an existing pull request
func (p *PullRequestService) UpdatePullRequest(ctx context.Context, workspace, repoSlug string, id int, request *UpdatePullRequestRequest) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}
	
	if request == nil {
		return nil, NewValidationError("update request is required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d", workspace, repoSlug, id)
	
	var result PullRequest
	err := p.client.PutJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// MergePullRequest merges a pull request
func (p *PullRequestService) MergePullRequest(ctx context.Context, workspace, repoSlug string, id int, request *PullRequestMerge) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/merge", workspace, repoSlug, id)
	
	// Set default merge request if none provided
	if request == nil {
		request = &PullRequestMerge{
			Type: "pullrequest_merge",
		}
	}
	
	if request.Type == "" {
		request.Type = "pullrequest_merge"
	}

	var result PullRequest
	err := p.client.PostJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeclinePullRequest declines (closes) a pull request
func (p *PullRequestService) DeclinePullRequest(ctx context.Context, workspace, repoSlug string, id int, reason string) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/decline", workspace, repoSlug, id)
	
	request := map[string]interface{}{
		"type": "pullrequest",
	}
	
	if reason != "" {
		request["reason"] = reason
	}

	var result PullRequest
	err := p.client.PostJSON(ctx, endpoint, request, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (p *PullRequestService) ReopenPullRequest(ctx context.Context, workspace, repoSlug string, id int, comment string) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	request := &UpdatePullRequestRequest{
		State: "OPEN",
	}

	if comment != "" {
		_, err := p.AddComment(ctx, workspace, repoSlug, id, comment, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to add comment: %w", err)
		}
	}

	return p.UpdatePullRequest(ctx, workspace, repoSlug, id, request)
}

// GetPullRequestActivity retrieves activity for a pull request
func (p *PullRequestService) GetPullRequestActivity(ctx context.Context, workspace, repoSlug string, id int) (*PaginatedResponse, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pullrequests/%d/activity", workspace, repoSlug, id)
	
	// Use the paginator for potentially large activity lists
	paginator := p.client.Paginate(endpoint, nil)
	return paginator.NextPage(ctx)
}

// Convenience methods for common operations

// ListOpenPullRequests retrieves all open pull requests
func (p *PullRequestService) ListOpenPullRequests(ctx context.Context, workspace, repoSlug string) (*PaginatedResponse, error) {
	return p.ListPullRequests(ctx, workspace, repoSlug, &PullRequestListOptions{
		State: "OPEN",
		Sort:  "-updated_on", // Most recently updated first
	})
}

// ListPullRequestsByAuthor retrieves pull requests by a specific author
func (p *PullRequestService) ListPullRequestsByAuthor(ctx context.Context, workspace, repoSlug, author string) (*PaginatedResponse, error) {
	return p.ListPullRequests(ctx, workspace, repoSlug, &PullRequestListOptions{
		Author: author,
		Sort:   "-updated_on",
	})
}

// ListPullRequestsForReviewer retrieves pull requests where the user is a reviewer
func (p *PullRequestService) ListPullRequestsForReviewer(ctx context.Context, workspace, repoSlug, reviewer string) (*PaginatedResponse, error) {
	return p.ListPullRequests(ctx, workspace, repoSlug, &PullRequestListOptions{
		Reviewer: reviewer,
		State:    "OPEN", // Usually only interested in open PRs for review
		Sort:     "-updated_on",
	})
}

// GetPullRequestByID retrieves a pull request by ID with error handling for common issues
func (p *PullRequestService) GetPullRequestByID(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	pr, err := p.GetPullRequest(ctx, workspace, repoSlug, id)
	if err != nil {
		// Check for common error cases
		if bbErr, ok := err.(*BitbucketError); ok {
			if bbErr.StatusCode == 404 {
				return nil, &BitbucketError{
					Type:       ErrorTypeNotFound,
					Message:    fmt.Sprintf("pull request #%d not found", id),
					StatusCode: 404,
					RequestID:  bbErr.RequestID,
				}
			}
		}
		return nil, err
	}
	return pr, nil
}

// AddInlineComment adds an inline comment to a specific line in a pull request
func (p *PullRequestService) AddInlineComment(ctx context.Context, workspace, repoSlug string, id int, comment, filePath string, lineNumber int) (*PullRequestComment, error) {
	inline := &PullRequestCommentInline{
		Type: "inline",
		Path: filePath,
		To:   lineNumber,
	}
	
	return p.AddComment(ctx, workspace, repoSlug, id, comment, inline)
}

func (p *PullRequestService) LockPullRequestConversation(ctx context.Context, workspace, repoSlug string, id int, reason string) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	pr, err := p.GetPullRequest(ctx, workspace, repoSlug, id)
	if err != nil {
		return nil, err
	}

	lockMessage := "🔒 **Conversation locked**\n\nThis pull request's conversation has been locked to prevent further comments."
	
	if reason != "" {
		var reasonText string
		switch reason {
		case "off_topic":
			reasonText = "off-topic"
		case "resolved":
			reasonText = "resolved"
		case "spam":
			reasonText = "spam"
		case "too_heated":
			reasonText = "too heated"
		default:
			reasonText = reason
		}
		lockMessage += fmt.Sprintf("\n\n**Reason:** %s", reasonText)
	}
	
	lockMessage += "\n\nIf you have questions about this decision, please contact the repository administrators."

	_, err = p.AddComment(ctx, workspace, repoSlug, id, lockMessage, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add lock comment: %w", err)
	}

	return pr, nil
}

func (p *PullRequestService) UnlockPullRequestConversation(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}
	
	if id <= 0 {
		return nil, NewValidationError("pull request ID must be positive", "")
	}

	pr, err := p.GetPullRequest(ctx, workspace, repoSlug, id)
	if err != nil {
		return nil, err
	}

	unlockMessage := "🔓 **Conversation unlocked**\n\nThis pull request's conversation has been unlocked and comments are now enabled again."

	_, err = p.AddComment(ctx, workspace, repoSlug, id, unlockMessage, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add unlock comment: %w", err)
	}

	return pr, nil
}
