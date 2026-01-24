package api

import (
	"context"
	"fmt"
	"net/url"
)

type RepositoryService struct {
	client *Client
}

func NewRepositoryService(client *Client) *RepositoryService {
	return &RepositoryService{
		client: client,
	}
}

type RepositoryListOptions struct {
	Role    string `url:"role,omitempty"`
	Query   string `url:"q,omitempty"`
	Sort    string `url:"sort,omitempty"`
	PageLen int    `url:"pagelen,omitempty"`
	Page    int    `url:"page,omitempty"`
}

func (r *RepositoryService) ListRepositories(ctx context.Context, workspace string, options *RepositoryListOptions) (*PaginatedResponse, error) {
	if workspace == "" {
		return nil, NewValidationError("workspace is required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s", workspace)

	if options != nil {
		queryParams := url.Values{}

		if options.Role != "" {
			queryParams.Set("role", options.Role)
		}
		if options.Query != "" {
			queryParams.Set("q", options.Query)
		}
		if options.Sort != "" {
			queryParams.Set("sort", options.Sort)
		}

		if len(queryParams) > 0 {
			endpoint += "?" + queryParams.Encode()
		}
	}

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

	paginator := r.client.Paginate(endpoint, pageOptions)
	return paginator.NextPage(ctx)
}
