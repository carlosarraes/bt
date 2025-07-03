package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockAuthManager implements auth.AuthManager for testing
type MockAuthManager struct {
	mock.Mock
}

func (m *MockAuthManager) Authenticate(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAuthManager) GetAuthenticatedUser(ctx context.Context) (*auth.User, error) {
	args := m.Called(ctx)
	if user := args.Get(0); user != nil {
		if mockUser, ok := user.(*MockUser); ok {
			return &auth.User{
				Username:    mockUser.Username,
				DisplayName: mockUser.DisplayName,
				AccountID:   mockUser.AccountID,
				UUID:        mockUser.UUID,
				Email:       mockUser.Email,
			}, args.Error(1)
		}
		return user.(*auth.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthManager) SetHTTPHeaders(req *http.Request) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthManager) IsAuthenticated(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthManager) Refresh(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAuthManager) Logout() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuthManager) GetMethod() auth.AuthMethod {
	args := m.Called()
	return auth.AuthMethod(args.String(0))
}

// MockUser represents a test user
type MockUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
	UUID        string `json:"uuid"`
	Email       string `json:"email,omitempty"`
}

// APIClientTestSuite provides unit tests for the API client
type APIClientTestSuite struct {
	suite.Suite
	mockAuth *MockAuthManager
	server   *httptest.Server
	client   *Client
}

func (suite *APIClientTestSuite) SetupTest() {
	suite.mockAuth = &MockAuthManager{}
	
	// Create test server
	suite.server = httptest.NewServer(http.HandlerFunc(suite.testHandler))
	
	// Create client with test server URL
	config := &ClientConfig{
		BaseURL:       suite.server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 2,
		EnableLogging: false,
		UserAgent:     "bt/test",
	}
	
	var err error
	suite.client, err = NewClient(suite.mockAuth, config)
	require.NoError(suite.T(), err)
}

func (suite *APIClientTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *APIClientTestSuite) testHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	path := r.URL.Path
	method := r.Method
	
	switch {
	case method == "GET" && path == "/test":
		suite.handleGetTest(w, r)
	case method == "POST" && path == "/test":
		suite.handlePostTest(w, r)
	case method == "GET" && path == "/error":
		suite.handleError(w, r)
	case method == "GET" && path == "/rate-limit":
		suite.handleRateLimit(w, r)
	case method == "GET" && path == "/server-error":
		suite.handleServerError(w, r)
	case method == "GET" && path == "/paginated":
		suite.handlePaginated(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *APIClientTestSuite) handleGetTest(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "success",
		"method":  r.Method,
		"path":    r.URL.Path,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *APIClientTestSuite) handlePostTest(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	
	response := map[string]interface{}{
		"message": "created",
		"body":    body,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *APIClientTestSuite) handleError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Bad request",
			"detail":  "Invalid parameters provided",
		},
	}
	json.NewEncoder(w).Encode(errorResp)
}

func (suite *APIClientTestSuite) handleRateLimit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Retry-After", "2")
	w.WriteHeader(http.StatusTooManyRequests)
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Rate limit exceeded",
		},
	}
	json.NewEncoder(w).Encode(errorResp)
}

func (suite *APIClientTestSuite) handleServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Internal server error",
		},
	}
	json.NewEncoder(w).Encode(errorResp)
}

func (suite *APIClientTestSuite) handlePaginated(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" || page == "1" {
		response := PaginatedResponse{
			Size:    2,
			Page:    1,
			PageLen: 50,
			Next:    suite.server.URL + "/paginated?page=2",
			Values:  json.RawMessage(`[{"id": 1}, {"id": 2}]`),
		}
		json.NewEncoder(w).Encode(response)
	} else {
		response := PaginatedResponse{
			Size:     1,
			Page:     2,
			PageLen:  50,
			Previous: suite.server.URL + "/paginated?page=1",
			Values:   json.RawMessage(`[{"id": 3}]`),
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *APIClientTestSuite) TestNewClient() {
	config := DefaultClientConfig()
	client, err := NewClient(suite.mockAuth, config)
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), client)
	assert.Equal(suite.T(), DefaultBaseURL, client.BaseURL())
	assert.Equal(suite.T(), suite.mockAuth, client.GetAuthManager())
}

func (suite *APIClientTestSuite) TestGetRequest() {
	// Setup mock auth
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Make GET request
	ctx := context.Background()
	resp, err := suite.client.Get(ctx, "/test")
	
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), 200, resp.StatusCode)
	
	// Verify response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	resp.Body.Close()
	
	assert.Equal(suite.T(), "success", result["message"])
	assert.Equal(suite.T(), "GET", result["method"])
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestPostRequest() {
	// Setup mock auth
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Make POST request with body
	ctx := context.Background()
	body := map[string]string{"test": "data"}
	resp, err := suite.client.Post(ctx, "/test", body)
	
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	assert.Equal(suite.T(), 200, resp.StatusCode)
	
	// Verify response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	resp.Body.Close()
	
	assert.Equal(suite.T(), "created", result["message"])
	assert.Equal(suite.T(), "data", result["body"].(map[string]interface{})["test"])
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestGetJSON() {
	// Setup mock auth
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Make GET request with JSON unmarshaling
	ctx := context.Background()
	var result map[string]interface{}
	err := suite.client.GetJSON(ctx, "/test", &result)
	
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "success", result["message"])
	assert.Equal(suite.T(), "GET", result["method"])
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestErrorHandling() {
	// Setup mock auth
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Make request that returns an error
	ctx := context.Background()
	_, err := suite.client.Get(ctx, "/error")
	
	require.Error(suite.T(), err)
	
	// Check that it's a BitbucketError
	bbErr, ok := err.(*BitbucketError)
	require.True(suite.T(), ok)
	assert.Equal(suite.T(), ErrorTypeValidation, bbErr.Type)
	assert.Equal(suite.T(), "Bad request", bbErr.Message)
	assert.Equal(suite.T(), 400, bbErr.StatusCode)
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestAuthenticationIntegration() {
	// Setup mock auth to add Authorization header
	suite.mockAuth.On("SetHTTPHeaders", mock.MatchedBy(func(req *http.Request) bool {
		req.Header.Set("Authorization", "Bearer test-token")
		return true
	})).Return(nil)
	
	// Make request
	ctx := context.Background()
	resp, err := suite.client.Get(ctx, "/test")
	
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	resp.Body.Close()
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestBuildURL() {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "simple endpoint",
			endpoint: "repositories",
			expected: suite.server.URL + "/repositories",
		},
		{
			name:     "endpoint with leading slash",
			endpoint: "/repositories",
			expected: suite.server.URL + "/repositories",
		},
		{
			name:     "endpoint with query params",
			endpoint: "repositories?q=test",
			expected: suite.server.URL + "/repositories?q=test",
		},
	}
	
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			url, err := suite.client.buildURL(tt.endpoint)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func (suite *APIClientTestSuite) TestUserAgent() {
	// Setup mock auth and capture request
	var capturedReq *http.Request
	suite.mockAuth.On("SetHTTPHeaders", mock.MatchedBy(func(req *http.Request) bool {
		capturedReq = req
		return true
	})).Return(nil)
	
	// Make request
	ctx := context.Background()
	resp, err := suite.client.Get(ctx, "/test")
	require.NoError(suite.T(), err)
	resp.Body.Close()
	
	// Check User-Agent header
	userAgent := capturedReq.Header.Get("User-Agent")
	suite.T().Logf("User-Agent: %s", userAgent)
	assert.True(suite.T(), strings.HasPrefix(userAgent, "bt/"), "User-Agent should start with 'bt/', got: %s", userAgent)
	
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *APIClientTestSuite) TestTimeout() {
	// Create client with short timeout
	config := &ClientConfig{
		BaseURL: suite.server.URL,
		Timeout: 10 * time.Millisecond, // Short timeout but not too short
	}
	
	client, err := NewClient(suite.mockAuth, config)
	require.NoError(suite.T(), err)
	
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Create a slow server that will cause timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Sleep longer than timeout
		w.WriteHeader(200)
	}))
	defer slowServer.Close()
	
	// Update client to use slow server
	client.config.BaseURL = slowServer.URL
	client.baseURL, _ = url.Parse(slowServer.URL)
	
	// Request should timeout
	ctx := context.Background()
	_, err = client.Get(ctx, "/test")
	
	// Should get a timeout or context error
	assert.Error(suite.T(), err)
}

func (suite *APIClientTestSuite) TestPagination() {
	// Setup mock auth
	suite.mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	// Create paginator
	paginator := suite.client.Paginate("/paginated", DefaultPageOptions())
	
	// Get first page
	ctx := context.Background()
	page1, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page1)
	
	assert.Equal(suite.T(), 1, page1.Page)
	assert.Equal(suite.T(), 2, page1.Size)
	assert.True(suite.T(), paginator.HasNextPage())
	
	// Get second page
	page2, err := paginator.NextPage(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), page2)
	
	assert.Equal(suite.T(), 2, page2.Page)
	assert.Equal(suite.T(), 1, page2.Size)
	assert.False(suite.T(), paginator.HasNextPage())
}

// TestAPIClient runs the test suite
func TestAPIClient(t *testing.T) {
	suite.Run(t, new(APIClientTestSuite))
}

// Benchmark tests
func BenchmarkClientGet(b *testing.B) {
	// Setup
	mockAuth := &MockAuthManager{}
	mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "success"}`)
	}))
	defer server.Close()
	
	config := &ClientConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}
	
	client, err := NewClient(mockAuth, config)
	require.NoError(b, err)
	
	ctx := context.Background()
	
	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(ctx, "/test")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}