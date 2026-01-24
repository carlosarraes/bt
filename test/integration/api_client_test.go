package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APIClientIntegrationSuite provides integration tests for the API client
type APIClientIntegrationSuite struct {
	suite.Suite
	client      *api.Client
	authManager auth.AuthManager
	testCtx     *utils.TestContext
}

// SetupSuite initializes the test suite with real authentication
func (suite *APIClientIntegrationSuite) SetupSuite() {
	// Skip if integration tests are not enabled
	utils.RequireIntegrationEnv(suite.T())

	suite.testCtx = utils.NewTestContext(suite.T())

	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load()
	require.NoError(suite.T(), err)

	// Set up authentication manager
	suite.authManager = suite.setupAuthentication(cfg)

	// Create API client
	clientConfig := &api.ClientConfig{
		BaseURL:       api.DefaultBaseURL,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		EnableLogging: suite.isVerbose(),
	}

	suite.client, err = api.NewClient(suite.authManager, clientConfig)
	require.NoError(suite.T(), err)
}

// TearDownSuite cleans up the test suite
func (suite *APIClientIntegrationSuite) TearDownSuite() {
	if suite.testCtx != nil {
		suite.testCtx.Close()
	}
}

// setupAuthentication configures authentication for testing
func (suite *APIClientIntegrationSuite) setupAuthentication(cfg *config.Config) auth.AuthManager {
	authManager, err := utils.CreateTestAuthManager()
	require.NoError(suite.T(), err)

	// Validate authentication
	ctx := context.Background()
	authenticated, err := authManager.IsAuthenticated(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), authenticated, "Authentication failed")

	return authManager
}

// isVerbose returns true if verbose logging is enabled
func (suite *APIClientIntegrationSuite) isVerbose() bool {
	return os.Getenv("VERBOSE") == "1" || os.Getenv("BT_VERBOSE") == "1"
}

// TestClientCreation tests that the client can be created successfully
func (suite *APIClientIntegrationSuite) TestClientCreation() {
	assert.NotNil(suite.T(), suite.client)
	assert.Equal(suite.T(), api.DefaultBaseURL, suite.client.BaseURL())
	assert.Equal(suite.T(), suite.authManager, suite.client.GetAuthManager())
}

// TestAuthenticatedUser tests getting the authenticated user
func (suite *APIClientIntegrationSuite) TestAuthenticatedUser() {
	ctx := context.Background()

	var user struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		AccountID   string `json:"account_id"`
		UUID        string `json:"uuid"`
	}

	err := suite.client.GetJSON(ctx, "/user", &user)
	require.NoError(suite.T(), err)

	assert.NotEmpty(suite.T(), user.Username)
	assert.NotEmpty(suite.T(), user.DisplayName)
	assert.NotEmpty(suite.T(), user.AccountID)
	assert.NotEmpty(suite.T(), user.UUID)

	suite.T().Logf("Authenticated user: %s (%s)", user.Username, user.DisplayName)
}

// TestGetRepositories tests listing repositories
func (suite *APIClientIntegrationSuite) TestGetRepositories() {
	ctx := context.Background()

	// Get the authenticated user first to use their workspace
	var user struct {
		Username string `json:"username"`
	}
	err := suite.client.GetJSON(ctx, "/user", &user)
	require.NoError(suite.T(), err)

	// List repositories for the user's workspace
	endpoint := "/repositories/" + user.Username

	var repoResponse struct {
		Size   int `json:"size"`
		Values []struct {
			Name      string `json:"name"`
			FullName  string `json:"full_name"`
			IsPrivate bool   `json:"is_private"`
			Language  string `json:"language"`
		} `json:"values"`
	}

	err = suite.client.GetJSON(ctx, endpoint+"?pagelen=10", &repoResponse)
	require.NoError(suite.T(), err)

	suite.T().Logf("Found %d repositories", repoResponse.Size)

	// If there are repositories, verify the structure
	if len(repoResponse.Values) > 0 {
		repo := repoResponse.Values[0]
		assert.NotEmpty(suite.T(), repo.Name)
		assert.NotEmpty(suite.T(), repo.FullName)
		suite.T().Logf("First repository: %s", repo.FullName)
	}
}

// TestPaginationWithRealAPI tests pagination with real API responses
func (suite *APIClientIntegrationSuite) TestPaginationWithRealAPI() {
	ctx := context.Background()

	// Get authenticated user
	var user struct {
		Username string `json:"username"`
	}
	err := suite.client.GetJSON(ctx, "/user", &user)
	require.NoError(suite.T(), err)

	// Create paginator for repositories
	endpoint := "/repositories/" + user.Username
	options := &api.PageOptions{
		Page:    1,
		PageLen: 5,  // Small page size to test pagination
		Limit:   10, // Limit total results
	}

	paginator := suite.client.Paginate(endpoint, options)

	totalItems := 0
	pageCount := 0

	// Iterate through pages
	for paginator.HasNextPage() {
		page, err := paginator.NextPage(ctx)
		require.NoError(suite.T(), err)

		if page == nil {
			break
		}

		pageCount++
		totalItems += page.Size

		suite.T().Logf("Page %d: %d items", pageCount, page.Size)

		// Don't test too many pages in CI
		if pageCount >= 3 {
			break
		}
	}

	suite.T().Logf("Pagination test completed: %d pages, %d total items", pageCount, totalItems)
}

// TestErrorHandlingWithRealAPI tests error handling with invalid requests
func (suite *APIClientIntegrationSuite) TestErrorHandlingWithRealAPI() {
	ctx := context.Background()

	// Try to access a non-existent repository
	_, err := suite.client.Get(ctx, "/repositories/nonexistent-user/nonexistent-repo")
	require.Error(suite.T(), err)

	// Verify it's a BitbucketError
	bbErr, ok := err.(*api.BitbucketError)
	require.True(suite.T(), ok, "Expected BitbucketError, got %T", err)

	assert.Equal(suite.T(), api.ErrorTypeNotFound, bbErr.Type)
	assert.Equal(suite.T(), 404, bbErr.StatusCode)

	suite.T().Logf("Error handling test - Type: %s, Message: %s", bbErr.Type, bbErr.Message)
}

// TestRateLimitHandling tests rate limit handling (if we hit it)
func (suite *APIClientIntegrationSuite) TestRateLimitHandling() {
	// This test is mainly for manual testing when rate limits are hit
	// In normal CI, we shouldn't hit rate limits

	ctx := context.Background()

	// Make a simple request that should succeed
	var user struct {
		Username string `json:"username"`
	}

	err := suite.client.GetJSON(ctx, "/user", &user)
	require.NoError(suite.T(), err)

	suite.T().Logf("Rate limit test passed - User: %s", user.Username)
}

// TestConcurrentRequests tests making multiple concurrent requests
func (suite *APIClientIntegrationSuite) TestConcurrentRequests() {
	ctx := context.Background()

	// Make 5 concurrent requests
	numRequests := 5
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			var user struct {
				Username string `json:"username"`
			}
			err := suite.client.GetJSON(ctx, "/user", &user)
			errors <- err
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-errors
		assert.NoError(suite.T(), err, "Concurrent request %d failed", i+1)
	}

	suite.T().Logf("Concurrent requests test completed: %d requests", numRequests)
}

// TestRequestTimeout tests timeout handling
func (suite *APIClientIntegrationSuite) TestRequestTimeout() {
	// Create client with very short timeout
	shortTimeoutConfig := &api.ClientConfig{
		BaseURL: api.DefaultBaseURL,
		Timeout: 1 * time.Millisecond, // Very short timeout
	}

	shortTimeoutClient, err := api.NewClient(suite.authManager, shortTimeoutConfig)
	require.NoError(suite.T(), err)

	ctx := context.Background()

	// This should timeout
	_, err = shortTimeoutClient.Get(ctx, "/user")
	require.Error(suite.T(), err)

	// Should be a network error
	assert.True(suite.T(), api.IsNetworkError(err), "Expected network error, got %T: %v", err, err)

	suite.T().Logf("Timeout test passed - Error: %v", err)
}

// TestClientWithContext tests context cancellation
func (suite *APIClientIntegrationSuite) TestClientWithContext() {
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	// Request should fail due to cancelled context
	_, err := suite.client.Get(ctx, "/user")
	require.Error(suite.T(), err)

	assert.Contains(suite.T(), err.Error(), "context canceled")

	suite.T().Logf("Context cancellation test passed - Error: %v", err)
}

// TestAPIClientIntegration runs the integration test suite
func TestAPIClientIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=1 to run.")
	}

	suite.Run(t, new(APIClientIntegrationSuite))
}

// BenchmarkRealAPICall benchmarks real API calls
func BenchmarkRealAPICall(b *testing.B) {
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		b.Skip("Skipping integration benchmarks. Set INTEGRATION_TESTS=1 to run.")
	}

	// Setup
	authManager, err := utils.CreateTestAuthManager()
	require.NoError(b, err)

	client, err := api.NewClient(authManager, api.DefaultClientConfig())
	require.NoError(b, err)

	ctx := context.Background()

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user struct {
			Username string `json:"username"`
		}
		err := client.GetJSON(ctx, "/user", &user)
		if err != nil {
			b.Fatal(err)
		}
	}
}
