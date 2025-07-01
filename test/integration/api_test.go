package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APIIntegrationSuite provides a test suite for API integration tests
type APIIntegrationSuite struct {
	suite.Suite
	httpClient *utils.HTTPClient
	testCtx    *utils.TestContext
}

// SetupSuite initializes the test suite
func (suite *APIIntegrationSuite) SetupSuite() {
	utils.RequireIntegrationEnv(suite.T())
	
	suite.httpClient = utils.NewHTTPClient()
	suite.testCtx = utils.NewTestContext(suite.T())
	
	// Verify API connectivity
	suite.verifyAPIConnectivity()
}

// TearDownSuite cleans up after the test suite
func (suite *APIIntegrationSuite) TearDownSuite() {
	if suite.testCtx != nil {
		suite.testCtx.Close()
	}
}

// verifyAPIConnectivity ensures we can connect to the Bitbucket API
func (suite *APIIntegrationSuite) verifyAPIConnectivity() {
	resp, err := suite.httpClient.Get("/user")
	require.NoError(suite.T(), err, "Failed to connect to Bitbucket API")
	defer resp.Body.Close()
	
	if resp.StatusCode == 401 {
		suite.T().Fatal("API authentication failed. Check BT_TEST_* environment variables.")
	}
	
	require.True(suite.T(), resp.StatusCode < 400, 
		"API connectivity check failed with status %d", resp.StatusCode)
}

// TestUserAPI tests user-related API endpoints
func (suite *APIIntegrationSuite) TestUserAPI() {
	// Test getting current user
	resp, err := suite.httpClient.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	user, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify user data structure
	assert.Contains(suite.T(), user, "username")
	assert.Contains(suite.T(), user, "display_name")
	assert.Contains(suite.T(), user, "account_id")
	assert.Contains(suite.T(), user, "created_on")
	
	// Verify username matches test configuration
	expectedUsername := utils.GetTestUsername()
	assert.Equal(suite.T(), expectedUsername, user["username"])
}

// TestWorkspaceAPI tests workspace-related API endpoints
func (suite *APIIntegrationSuite) TestWorkspaceAPI() {
	workspace := utils.GetTestWorkspace()
	
	// Test getting workspace info
	endpoint := fmt.Sprintf("/workspaces/%s", workspace)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test workspace '%s' not found. Skipping workspace API tests.", workspace)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	workspaceData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify workspace data structure
	assert.Contains(suite.T(), workspaceData, "slug")
	assert.Contains(suite.T(), workspaceData, "name")
	assert.Contains(suite.T(), workspaceData, "created_on")
	
	// Verify workspace slug matches test configuration
	assert.Equal(suite.T(), workspace, workspaceData["slug"])
}

// TestRepositoryAPI tests repository-related API endpoints
func (suite *APIIntegrationSuite) TestRepositoryAPI() {
	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()
	
	// Test getting repository info
	endpoint := fmt.Sprintf("/repositories/%s/%s", workspace, repo)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test repository '%s/%s' not found. Skipping repository API tests.", workspace, repo)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	repoData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify repository data structure
	assert.Contains(suite.T(), repoData, "name")
	assert.Contains(suite.T(), repoData, "full_name")
	assert.Contains(suite.T(), repoData, "created_on")
	assert.Contains(suite.T(), repoData, "updated_on")
	assert.Contains(suite.T(), repoData, "is_private")
	
	// Verify repository name matches test configuration
	assert.Equal(suite.T(), repo, repoData["name"])
	assert.Equal(suite.T(), fmt.Sprintf("%s/%s", workspace, repo), repoData["full_name"])
}

// TestRepositoryListAPI tests repository listing API
func (suite *APIIntegrationSuite) TestRepositoryListAPI() {
	workspace := utils.GetTestWorkspace()
	
	// Test listing repositories in workspace
	endpoint := fmt.Sprintf("/repositories/%s", workspace)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test workspace '%s' not found. Skipping repository list API tests.", workspace)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	listData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify list structure
	assert.Contains(suite.T(), listData, "values")
	assert.Contains(suite.T(), listData, "page")
	assert.Contains(suite.T(), listData, "size")
	
	// Verify we have at least one repository
	values, ok := listData["values"].([]interface{})
	require.True(suite.T(), ok, "values should be an array")
	assert.Greater(suite.T(), len(values), 0, "Should have at least one repository")
	
	// Verify repository structure
	if len(values) > 0 {
		repo, ok := values[0].(map[string]interface{})
		require.True(suite.T(), ok, "Repository should be an object")
		assert.Contains(suite.T(), repo, "name")
		assert.Contains(suite.T(), repo, "full_name")
	}
}

// TestPullRequestAPI tests pull request related API endpoints
func (suite *APIIntegrationSuite) TestPullRequestAPI() {
	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()
	
	// Test listing pull requests
	endpoint := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repo)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test repository '%s/%s' not found. Skipping PR API tests.", workspace, repo)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	prData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify PR list structure
	assert.Contains(suite.T(), prData, "values")
	assert.Contains(suite.T(), prData, "page")
	assert.Contains(suite.T(), prData, "size")
	
	// If there are PRs, verify their structure
	values, ok := prData["values"].([]interface{})
	require.True(suite.T(), ok, "values should be an array")
	
	if len(values) > 0 {
		pr, ok := values[0].(map[string]interface{})
		require.True(suite.T(), ok, "PR should be an object")
		assert.Contains(suite.T(), pr, "id")
		assert.Contains(suite.T(), pr, "title")
		assert.Contains(suite.T(), pr, "state")
		assert.Contains(suite.T(), pr, "created_on")
		assert.Contains(suite.T(), pr, "source")
		assert.Contains(suite.T(), pr, "destination")
	}
}

// TestPipelineAPI tests pipeline related API endpoints
func (suite *APIIntegrationSuite) TestPipelineAPI() {
	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()
	
	// Test listing pipelines
	endpoint := fmt.Sprintf("/repositories/%s/%s/pipelines/", workspace, repo)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test repository '%s/%s' not found. Skipping pipeline API tests.", workspace, repo)
		return
	}
	
	// Pipelines might not be enabled, so 404 is acceptable
	if resp.StatusCode == 404 {
		suite.T().Logf("Pipelines not enabled for repository '%s/%s'", workspace, repo)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	pipelineData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify pipeline list structure
	assert.Contains(suite.T(), pipelineData, "values")
	assert.Contains(suite.T(), pipelineData, "page")
	assert.Contains(suite.T(), pipelineData, "size")
	
	// If there are pipelines, verify their structure
	values, ok := pipelineData["values"].([]interface{})
	require.True(suite.T(), ok, "values should be an array")
	
	if len(values) > 0 {
		pipeline, ok := values[0].(map[string]interface{})
		require.True(suite.T(), ok, "Pipeline should be an object")
		assert.Contains(suite.T(), pipeline, "uuid")
		assert.Contains(suite.T(), pipeline, "build_number")
		assert.Contains(suite.T(), pipeline, "state")
		assert.Contains(suite.T(), pipeline, "created_on")
	}
}

// TestAPIRateLimit tests that we handle rate limiting properly
func (suite *APIIntegrationSuite) TestAPIRateLimit() {
	// Make several rapid requests to test rate limiting behavior
	const numRequests = 5
	
	for i := 0; i < numRequests; i++ {
		resp, err := suite.httpClient.Get("/user")
		require.NoError(suite.T(), err)
		
		// Check if we hit rate limit
		if resp.StatusCode == 429 {
			suite.T().Logf("Hit rate limit on request %d", i+1)
			
			// Verify rate limit headers
			assert.NotEmpty(suite.T(), resp.Header.Get("X-RateLimit-Limit"))
			assert.NotEmpty(suite.T(), resp.Header.Get("X-RateLimit-Remaining"))
			
			// Wait for rate limit reset
			utils.RateLimitWait(resp)
		} else {
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		}
		
		resp.Body.Close()
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
}

// TestAPIErrorHandling tests various error scenarios
func (suite *APIIntegrationSuite) TestAPIErrorHandling() {
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
		description    string
	}{
		{
			name:           "NotFound",
			endpoint:       "/repositories/nonexistent/nonexistent",
			expectedStatus: 404,
			description:    "Should return 404 for non-existent repository",
		},
		{
			name:           "InvalidEndpoint",
			endpoint:       "/invalid/endpoint/that/does/not/exist",
			expectedStatus: 404,
			description:    "Should return 404 for invalid endpoint",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			resp, err := suite.httpClient.Get(tc.endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, tc.description)
			
			// Verify error response structure for API errors
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				errorData, err := utils.ReadJSONResponse(resp)
				if err == nil {
					// Bitbucket API error responses typically have an "error" field
					assert.Contains(t, errorData, "error")
				}
			}
		})
	}
}

// TestAPIPerformance tests API response times
func (suite *APIIntegrationSuite) TestAPIPerformance() {
	testCases := []struct {
		name        string
		endpoint    string
		maxDuration time.Duration
	}{
		{
			name:        "UserEndpoint",
			endpoint:    "/user",
			maxDuration: 2 * time.Second,
		},
		{
			name:        "WorkspaceEndpoint",
			endpoint:    fmt.Sprintf("/workspaces/%s", utils.GetTestWorkspace()),
			maxDuration: 2 * time.Second,
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			start := time.Now()
			
			resp, err := suite.httpClient.Get(tc.endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			duration := time.Since(start)
			
			// Only check performance if the request was successful
			if resp.StatusCode < 400 {
				assert.Less(t, duration, tc.maxDuration, 
					"API response time for %s should be less than %v, got %v", 
					tc.name, tc.maxDuration, duration)
			}
		})
	}
}

// TestAPIHeaders tests that proper headers are sent and received
func (suite *APIIntegrationSuite) TestAPIHeaders() {
	resp, err := suite.httpClient.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Check response headers
	assert.Equal(suite.T(), "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.NotEmpty(suite.T(), resp.Header.Get("X-Request-Id"))
	
	// Check for security headers
	assert.NotEmpty(suite.T(), resp.Header.Get("X-Frame-Options"))
	assert.NotEmpty(suite.T(), resp.Header.Get("X-Content-Type-Options"))
}

// TestAPIAuthentication tests different authentication scenarios
func (suite *APIIntegrationSuite) TestAPIAuthentication() {
	// Test valid authentication (already tested in other methods)
	resp, err := suite.httpClient.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	// Test invalid authentication
	invalidClient := &utils.HTTPClient{
		BaseURL:  suite.httpClient.BaseURL,
		Username: "invalid",
		Password: "invalid",
		Client:   suite.httpClient.Client,
	}
	
	resp, err = invalidClient.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

// TestPaginationAPI tests API pagination functionality
func (suite *APIIntegrationSuite) TestPaginationAPI() {
	workspace := utils.GetTestWorkspace()
	
	// Test repository listing with pagination
	endpoint := fmt.Sprintf("/repositories/%s?pagelen=1", workspace)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test workspace '%s' not found. Skipping pagination API tests.", workspace)
		return
	}
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	data, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify pagination structure
	assert.Contains(suite.T(), data, "values")
	assert.Contains(suite.T(), data, "page")
	assert.Contains(suite.T(), data, "size")
	assert.Contains(suite.T(), data, "pagelen")
	
	// Verify page size limit
	pagelen, ok := data["pagelen"].(float64)
	require.True(suite.T(), ok, "pagelen should be a number")
	assert.Equal(suite.T(), float64(1), pagelen, "pagelen should be 1")
	
	// Verify values array respects pagination
	values, ok := data["values"].([]interface{})
	require.True(suite.T(), ok, "values should be an array")
	assert.LessOrEqual(suite.T(), len(values), 1, "Should have at most 1 item per page")
}

// Run the test suite
func TestAPIIntegration(t *testing.T) {
	suite.Run(t, new(APIIntegrationSuite))
}

// Additional standalone integration tests

// TestAPIBasicConnectivity is a basic connectivity test that can run independently
func TestAPIBasicConnectivity(t *testing.T) {
	utils.SkipIfNoIntegration(t)
	
	client := utils.NewHTTPClient()
	
	resp, err := client.Get("/user")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.True(t, resp.StatusCode < 500, 
		"API should be reachable (got status %d)", resp.StatusCode)
		
	if resp.StatusCode == 401 {
		t.Fatal("API authentication failed. Check BT_TEST_* environment variables.")
	}
}