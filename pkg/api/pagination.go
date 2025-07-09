package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PaginatedResponse represents a paginated response from the Bitbucket API
type PaginatedResponse struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Values   json.RawMessage `json:"values"`
}

// PageInfo contains pagination metadata
type PageInfo struct {
	Size        int
	Page        int
	PageLen     int
	HasNext     bool
	NextURL     string
	HasPrevious bool
	PreviousURL string
	TotalItems  int
}

// PageOptions contains options for paginated requests
type PageOptions struct {
	Page    int
	PageLen int
	Limit   int // Maximum total items to fetch (0 = no limit)
}

// DefaultPageOptions returns sensible defaults for pagination
func DefaultPageOptions() *PageOptions {
	return &PageOptions{
		Page:    1,
		PageLen: 50, // Bitbucket default
		Limit:   0,  // No limit
	}
}

// Paginator handles paginated API requests
type Paginator struct {
	client     *Client
	baseURL    string
	options    *PageOptions
	pageInfo   *PageInfo
	totalFetched int
}

// NewPaginator creates a new paginator for the given URL
func NewPaginator(client *Client, baseURL string, options *PageOptions) *Paginator {
	if options == nil {
		options = DefaultPageOptions()
	}

	return &Paginator{
		client:  client,
		baseURL: baseURL,
		options: options,
		pageInfo: &PageInfo{
			Page:    options.Page,
			PageLen: options.PageLen,
		},
	}
}

// NextPage fetches the next page of results
func (p *Paginator) NextPage(ctx context.Context) (*PaginatedResponse, error) {
	// Check if we've reached our limit
	if p.options.Limit > 0 && p.totalFetched >= p.options.Limit {
		return nil, nil // No more pages to fetch
	}
	
	// Check if we have no more pages to fetch
	if p.totalFetched > 0 && !p.pageInfo.HasNext {
		return nil, nil // No more pages available
	}

	// Determine the URL to fetch
	var fetchURL string
	if p.pageInfo.HasNext && p.pageInfo.NextURL != "" {
		fetchURL = p.pageInfo.NextURL
	} else {
		// Build URL with pagination parameters using client's buildURL method
		endpoint := p.baseURL
		if !strings.Contains(endpoint, "?") {
			endpoint += "?"
		} else {
			endpoint += "&"
		}
		endpoint += fmt.Sprintf("page=%d&pagelen=%d", p.pageInfo.Page, p.pageInfo.PageLen)
		
		var err error
		fetchURL, err = p.client.buildURL(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to build URL: %w", err)
		}
	}

	// fmt.Fprintf(os.Stderr, "DEBUG: Full request URL: %s\n", fetchURL)
	
	// Make the request
	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "bt/1.0.0")
	
	if p.client.authManager != nil {
		if err := p.client.authManager.SetHTTPHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	// fmt.Fprintf(os.Stderr, "DEBUG: Request headers: %+v\n", req.Header)

	resp, err := p.client.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// fmt.Fprintf(os.Stderr, "DEBUG: Response status: %d %s\n", resp.StatusCode, resp.Status)

	// Parse the paginated response
	var paginatedResp PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, fmt.Errorf("failed to decode paginated response: %w", err)
	}

	// Update pagination info
	p.updatePageInfo(&paginatedResp)

	// Update total fetched counter
	p.totalFetched += paginatedResp.Size

	return &paginatedResp, nil
}

// HasNextPage returns true if there are more pages available
func (p *Paginator) HasNextPage() bool {
	// Check limit constraint
	if p.options.Limit > 0 && p.totalFetched >= p.options.Limit {
		return false
	}

	// If we haven't fetched any pages yet, we should try to fetch the first page
	if p.totalFetched == 0 {
		return true
	}

	return p.pageInfo.HasNext
}

// Reset resets the paginator to the first page
func (p *Paginator) Reset() {
	p.pageInfo = &PageInfo{
		Page:    p.options.Page,
		PageLen: p.options.PageLen,
	}
	p.totalFetched = 0
}

// GetPageInfo returns the current pagination information
func (p *Paginator) GetPageInfo() *PageInfo {
	return p.pageInfo
}

// FetchAll fetches all pages and returns all values as a single slice
func (p *Paginator) FetchAll(ctx context.Context) ([]json.RawMessage, error) {
	var allValues []json.RawMessage

	for p.HasNextPage() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		if page == nil {
			break
		}

		// Parse the values array
		var values []json.RawMessage
		if err := json.Unmarshal(page.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal values: %w", err)
		}

		allValues = append(allValues, values...)

		// Check if we've reached our limit
		if p.options.Limit > 0 && len(allValues) >= p.options.Limit {
			// Trim to exact limit
			if len(allValues) > p.options.Limit {
				allValues = allValues[:p.options.Limit]
			}
			break
		}
	}

	return allValues, nil
}

// FetchAllTyped fetches all pages and unmarshals values into the provided slice
func (p *Paginator) FetchAllTyped(ctx context.Context, result interface{}) error {
	allValues, err := p.FetchAll(ctx)
	if err != nil {
		return err
	}

	// Marshal all values as a JSON array
	valuesJSON, err := json.Marshal(allValues)
	if err != nil {
		return fmt.Errorf("failed to marshal values: %w", err)
	}

	// Unmarshal into the result
	if err := json.Unmarshal(valuesJSON, result); err != nil {
		return fmt.Errorf("failed to unmarshal into result: %w", err)
	}

	return nil
}

// Iterator provides an iterator interface for paginated results
type Iterator struct {
	paginator  *Paginator
	currentPage *PaginatedResponse
	currentValues []json.RawMessage
	currentIndex  int
	ctx           context.Context
}

// NewIterator creates a new iterator for paginated results
func NewIterator(client *Client, baseURL string, options *PageOptions) *Iterator {
	return &Iterator{
		paginator: NewPaginator(client, baseURL, options),
		ctx:       context.Background(),
	}
}

// WithContext sets the context for the iterator
func (it *Iterator) WithContext(ctx context.Context) *Iterator {
	it.ctx = ctx
	return it
}

// Next returns the next item in the paginated results
func (it *Iterator) Next() (json.RawMessage, error) {
	// Check if we need to fetch a new page
	if it.currentValues == nil || it.currentIndex >= len(it.currentValues) {
		if !it.paginator.HasNextPage() {
			return nil, nil // No more items
		}

		page, err := it.paginator.NextPage(it.ctx)
		if err != nil {
			return nil, err
		}

		if page == nil {
			return nil, nil // No more items
		}

		// Parse values
		var values []json.RawMessage
		if err := json.Unmarshal(page.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal values: %w", err)
		}

		it.currentPage = page
		it.currentValues = values
		it.currentIndex = 0
	}

	// Return the current item and advance
	if it.currentIndex < len(it.currentValues) {
		item := it.currentValues[it.currentIndex]
		it.currentIndex++
		return item, nil
	}

	return nil, nil // No more items
}

// HasNext returns true if there are more items available
func (it *Iterator) HasNext() bool {
	// If we have items in the current page
	if it.currentValues != nil && it.currentIndex < len(it.currentValues) {
		return true
	}

	// Check if there are more pages
	return it.paginator.HasNextPage()
}

// updatePageInfo updates the pagination info from a response
func (p *Paginator) updatePageInfo(resp *PaginatedResponse) {
	p.pageInfo.Size = resp.Size
	p.pageInfo.Page = resp.Page
	p.pageInfo.PageLen = resp.PageLen
	p.pageInfo.NextURL = resp.Next
	p.pageInfo.PreviousURL = resp.Previous
	p.pageInfo.HasNext = resp.Next != ""
	p.pageInfo.HasPrevious = resp.Previous != ""
}

// PaginateRequest is a helper function to add pagination parameters to a URL
func PaginateRequest(baseURL string, options *PageOptions) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if options == nil {
		options = DefaultPageOptions()
	}

	query := u.Query()
	if options.Page > 0 {
		query.Set("page", strconv.Itoa(options.Page))
	}
	if options.PageLen > 0 {
		query.Set("pagelen", strconv.Itoa(options.PageLen))
	}

	u.RawQuery = query.Encode()
	return u.String(), nil
}
