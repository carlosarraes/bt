package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PaginationTestSuite provides unit tests for pagination functionality
type PaginationTestSuite struct {
	suite.Suite
	server *httptest.Server
	client *Client
}

func (suite *PaginationTestSuite) SetupTest() {
	// Create test server
	suite.server = httptest.NewServer(http.HandlerFunc(suite.paginationHandler))
	
	// Create client
	config := &ClientConfig{
		BaseURL: suite.server.URL,
	}
	
	var err error
	suite.client, err = NewClient(nil, config)
	require.NoError(suite.T(), err)
}

func (suite *PaginationTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *PaginationTestSuite) paginationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	path := r.URL.Path
	page := r.URL.Query().Get("page")
	pageLen := r.URL.Query().Get("pagelen")
	
	switch path {
	case "/items":
		suite.handleItemsPagination(w, r, page, pageLen)
	case "/empty":
		suite.handleEmptyPagination(w, r)
	case "/single-page":
		suite.handleSinglePagePagination(w, r)
	case "/error":
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": {"message": "Server error"}}`)
	default:
		http.NotFound(w, r)
	}
}

func (suite *PaginationTestSuite) handleItemsPagination(w http.ResponseWriter, r *http.Request, page, pageLen string) {
	currentPage := 1
	if page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			currentPage = p
		}
	}
	
	itemsPerPage := 2
	if pageLen != "" {
		if pl, err := strconv.Atoi(pageLen); err == nil {
			itemsPerPage = pl
		}
	}
	
	// Generate items for current page
	startID := (currentPage-1)*itemsPerPage + 1
	endID := startID + itemsPerPage - 1
	
	var items []map[string]interface{}
	for i := startID; i <= endID && i <= 10; i++ { // Total of 10 items
		items = append(items, map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("Item %d", i),
		})
	}
	
	response := PaginatedResponse{
		Size:    len(items),
		Page:    currentPage,
		PageLen: itemsPerPage,
	}
	
	// Add next link if there are more pages
	if endID < 10 {
		response.Next = fmt.Sprintf("%s/items?page=%d&pagelen=%d", 
			suite.server.URL, currentPage+1, itemsPerPage)
	}
	
	// Add previous link if not on first page
	if currentPage > 1 {
		response.Previous = fmt.Sprintf("%s/items?page=%d&pagelen=%d", 
			suite.server.URL, currentPage-1, itemsPerPage)
	}
	
	// Convert items to JSON
	itemsJSON, _ := json.Marshal(items)
	response.Values = json.RawMessage(itemsJSON)
	
	json.NewEncoder(w).Encode(response)
}

func (suite *PaginationTestSuite) handleEmptyPagination(w http.ResponseWriter, r *http.Request) {
	response := PaginatedResponse{
		Size:    0,
		Page:    1,
		PageLen: 50,
		Values:  json.RawMessage("[]"),
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *PaginationTestSuite) handleSinglePagePagination(w http.ResponseWriter, r *http.Request) {
	items := []map[string]interface{}{
		{"id": 1, "name": "Only Item"},
	}
	
	response := PaginatedResponse{
		Size:    1,
		Page:    1,
		PageLen: 50,
	}
	
	itemsJSON, _ := json.Marshal(items)
	response.Values = json.RawMessage(itemsJSON)
	
	json.NewEncoder(w).Encode(response)
}

func (suite *PaginationTestSuite) TestDefaultPageOptions() {
	options := DefaultPageOptions()
	assert.NotNil(suite.T(), options)
	assert.Equal(suite.T(), 1, options.Page)
	assert.Equal(suite.T(), 50, options.PageLen)
	assert.Equal(suite.T(), 0, options.Limit)
}

func (suite *PaginationTestSuite) TestNewPaginator() {
	options := &PageOptions{Page: 2, PageLen: 10, Limit: 100}
	paginator := NewPaginator(suite.client, "/items", options)
	
	assert.NotNil(suite.T(), paginator)
	assert.Equal(suite.T(), suite.client, paginator.client)
	assert.Equal(suite.T(), "/items", paginator.baseURL)
	assert.Equal(suite.T(), options, paginator.options)
	
	pageInfo := paginator.GetPageInfo()
	assert.Equal(suite.T(), 2, pageInfo.Page)
	assert.Equal(suite.T(), 10, pageInfo.PageLen)
}

func (suite *PaginationTestSuite) TestPaginatorNextPage() {
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 2})
	
	ctx := context.Background()
	
	// Get first page
	page1, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page1)
	
	assert.Equal(suite.T(), 1, page1.Page)
	assert.Equal(suite.T(), 2, page1.Size)
	assert.True(suite.T(), paginator.HasNextPage())
	
	// Parse first page items
	var items1 []map[string]interface{}
	err = json.Unmarshal(page1.Values, &items1)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), items1, 2)
	assert.Equal(suite.T(), float64(1), items1[0]["id"]) // JSON numbers are float64
	assert.Equal(suite.T(), float64(2), items1[1]["id"])
	
	// Get second page
	page2, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page2)
	
	assert.Equal(suite.T(), 2, page2.Page)
	assert.Equal(suite.T(), 2, page2.Size)
	assert.True(suite.T(), paginator.HasNextPage())
	
	// Parse second page items
	var items2 []map[string]interface{}
	err = json.Unmarshal(page2.Values, &items2)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), items2, 2)
	assert.Equal(suite.T(), float64(3), items2[0]["id"])
	assert.Equal(suite.T(), float64(4), items2[1]["id"])
}

func (suite *PaginationTestSuite) TestPaginatorFetchAll() {
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 3})
	
	ctx := context.Background()
	allValues, err := paginator.FetchAll(ctx)
	require.NoError(suite.T(), err)
	
	// Should get all 10 items
	assert.Len(suite.T(), allValues, 10)
	
	// Parse first item to verify structure
	var firstItem map[string]interface{}
	err = json.Unmarshal(allValues[0], &firstItem)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(1), firstItem["id"])
	assert.Equal(suite.T(), "Item 1", firstItem["name"])
}

func (suite *PaginationTestSuite) TestPaginatorFetchAllWithLimit() {
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 3, Limit: 5})
	
	ctx := context.Background()
	allValues, err := paginator.FetchAll(ctx)
	require.NoError(suite.T(), err)
	
	// Should only get 5 items due to limit
	assert.Len(suite.T(), allValues, 5)
}

func (suite *PaginationTestSuite) TestPaginatorFetchAllTyped() {
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 5})
	
	ctx := context.Background()
	
	var items []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	
	err := paginator.FetchAllTyped(ctx, &items)
	require.NoError(suite.T(), err)
	
	assert.Len(suite.T(), items, 10)
	assert.Equal(suite.T(), 1, items[0].ID)
	assert.Equal(suite.T(), "Item 1", items[0].Name)
	assert.Equal(suite.T(), 10, items[9].ID)
	assert.Equal(suite.T(), "Item 10", items[9].Name)
}

func (suite *PaginationTestSuite) TestPaginatorReset() {
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 2})
	
	ctx := context.Background()
	
	// Fetch some pages
	_, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	_, err = paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	
	// Reset and verify
	paginator.Reset()
	
	pageInfo := paginator.GetPageInfo()
	assert.Equal(suite.T(), 1, pageInfo.Page)
	assert.Equal(suite.T(), 2, pageInfo.PageLen)
	assert.Equal(suite.T(), 0, paginator.totalFetched)
}

func (suite *PaginationTestSuite) TestIterator() {
	iterator := NewIterator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 3})
	
	ctx := context.Background()
	iterator = iterator.WithContext(ctx)
	
	var items []map[string]interface{}
	
	// Iterate through all items
	for iterator.HasNext() {
		item, err := iterator.Next()
		require.NoError(suite.T(), err)
		
		if item == nil {
			break
		}
		
		var parsedItem map[string]interface{}
		err = json.Unmarshal(item, &parsedItem)
		require.NoError(suite.T(), err)
		
		items = append(items, parsedItem)
	}
	
	// Should get all 10 items
	assert.Len(suite.T(), items, 10)
	assert.Equal(suite.T(), float64(1), items[0]["id"])
	assert.Equal(suite.T(), float64(10), items[9]["id"])
}

func (suite *PaginationTestSuite) TestEmptyPagination() {
	paginator := NewPaginator(suite.client, "/empty", DefaultPageOptions())
	
	ctx := context.Background()
	page, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page)
	
	assert.Equal(suite.T(), 0, page.Size)
	assert.False(suite.T(), paginator.HasNextPage())
	
	// FetchAll should return empty slice
	allValues, err := paginator.FetchAll(ctx)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), allValues, 0)
}

func (suite *PaginationTestSuite) TestSinglePagePagination() {
	paginator := NewPaginator(suite.client, "/single-page", DefaultPageOptions())
	
	ctx := context.Background()
	page, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page)
	
	assert.Equal(suite.T(), 1, page.Size)
	assert.False(suite.T(), paginator.HasNextPage())
	
	// Should not be able to get another page
	page2, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), page2)
}

func (suite *PaginationTestSuite) TestPaginateRequest() {
	tests := []struct {
		name     string
		baseURL  string
		options  *PageOptions
		expected string
	}{
		{
			name:     "Default options",
			baseURL:  "https://api.example.com/items",
			options:  DefaultPageOptions(),
			expected: "https://api.example.com/items?page=1&pagelen=50",
		},
		{
			name:     "Custom options",
			baseURL:  "https://api.example.com/items",
			options:  &PageOptions{Page: 2, PageLen: 10},
			expected: "https://api.example.com/items?page=2&pagelen=10",
		},
		{
			name:     "URL with existing query params",
			baseURL:  "https://api.example.com/items?q=test",
			options:  &PageOptions{Page: 1, PageLen: 25},
			expected: "https://api.example.com/items?page=1&pagelen=25&q=test",
		},
		{
			name:     "Nil options",
			baseURL:  "https://api.example.com/items",
			options:  nil,
			expected: "https://api.example.com/items?page=1&pagelen=50",
		},
	}
	
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			result, err := PaginateRequest(tt.baseURL, tt.options)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func (suite *PaginationTestSuite) TestPaginateRequestInvalidURL() {
	_, err := PaginateRequest("://invalid-url", nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid URL")
}

func (suite *PaginationTestSuite) TestPaginatorWithLimitReached() {
	// Create paginator with limit of 3 items
	paginator := NewPaginator(suite.client, "/items", &PageOptions{Page: 1, PageLen: 2, Limit: 3})
	
	ctx := context.Background()
	
	// Get first page (2 items)
	page1, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page1)
	assert.Equal(suite.T(), 2, page1.Size)
	
	// Should still have next page
	assert.True(suite.T(), paginator.HasNextPage())
	
	// Get second page (should only get 1 item due to limit)
	page2, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), page2) // Should return nil when limit is reached
	
	// Should not have next page after reaching limit
	assert.False(suite.T(), paginator.HasNextPage())
}

// TestPagination runs the pagination test suite
func TestPagination(t *testing.T) {
	suite.Run(t, new(PaginationTestSuite))
}

// Benchmark tests for pagination
func BenchmarkPaginatorNextPage(b *testing.B) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := PaginatedResponse{
			Size:    50,
			Page:    1,
			PageLen: 50,
			Values:  json.RawMessage(`[{"id": 1}, {"id": 2}]`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := &ClientConfig{BaseURL: server.URL}
	client, err := NewClient(nil, config)
	require.NoError(b, err)
	
	paginator := NewPaginator(client, "/items", DefaultPageOptions())
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paginator.Reset()
		_, err := paginator.NextPage(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIteratorNext(b *testing.B) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		items := make([]map[string]int, 50)
		for i := 0; i < 50; i++ {
			items[i] = map[string]int{"id": i + 1}
		}
		
		response := PaginatedResponse{
			Size:    50,
			Page:    1,
			PageLen: 50,
			Values:  json.RawMessage(fmt.Sprintf(`%v`, items)[1:]), // Remove first '['
		}
		response.Values = json.RawMessage(fmt.Sprintf(`%v`, items))
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := &ClientConfig{BaseURL: server.URL}
	client, err := NewClient(nil, config)
	require.NoError(b, err)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iterator := NewIterator(client, "/items", DefaultPageOptions()).WithContext(ctx)
		for iterator.HasNext() {
			_, err := iterator.Next()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}