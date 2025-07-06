package integration

import (
	"context"
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

// PullRequestIntegrationSuite provides integration tests for pull request API
type PullRequestIntegrationSuite struct {
	suite.Suite
	client      *api.Client
	authManager auth.AuthManager
	testCtx     *utils.TestContext
	workspace   string
	repository  string
}

// SetupSuite initializes the test suite with real authentication
func (suite *PullRequestIntegrationSuite) SetupSuite() {
	// Skip if integration tests are not enabled
	utils.RequireIntegrationEnv(suite.T())
	
	suite.testCtx = utils.NewTestContext(suite.T())
	
	// Load configuration
	cfg, err := config.Load()
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
	
	// Get test repository information
	suite.workspace = suite.getTestWorkspace()
	suite.repository = suite.getTestRepository()
}

// setupAuthentication sets up the auth manager for testing
func (suite *PullRequestIntegrationSuite) setupAuthentication(cfg *config.Config) auth.AuthManager {
	// Try environment variables first
	if email := suite.testCtx.GetEnv("BITBUCKET_EMAIL"); email != "" {
		if token := suite.testCtx.GetEnv("BITBUCKET_API_TOKEN"); token != "" {
			suite.T().Logf("Using API token authentication for integration tests")
			authManager, err := auth.NewAccessTokenAuth(email, token, "")
			require.NoError(suite.T(), err)
			return authManager
		}
	}
	
	// Fallback to app password authentication
	if username := suite.testCtx.GetEnv("BITBUCKET_USERNAME"); username != "" {
		if password := suite.testCtx.GetEnv("BITBUCKET_PASSWORD"); password != "" {
			suite.T().Logf("Using app password authentication for integration tests")
			authManager, err := auth.NewAppPasswordAuth(username, password, "")
			require.NoError(suite.T(), err)
			return authManager
		}
	}
	
	suite.T().Skip("No authentication credentials available for integration tests")
	return nil
}

// isVerbose returns true if verbose testing is enabled
func (suite *PullRequestIntegrationSuite) isVerbose() bool {
	return suite.testCtx.GetEnv("VERBOSE") == "1" || suite.testCtx.GetEnv("VERBOSE") == "true"
}

// getTestWorkspace returns the workspace to use for testing
func (suite *PullRequestIntegrationSuite) getTestWorkspace() string {
	workspace := suite.testCtx.GetEnv("TEST_WORKSPACE")
	if workspace == "" {
		suite.T().Skip("TEST_WORKSPACE environment variable not set")
	}
	return workspace
}

// getTestRepository returns the repository to use for testing
func (suite *PullRequestIntegrationSuite) getTestRepository() string {
	repo := suite.testCtx.GetEnv("TEST_REPOSITORY")
	if repo == "" {
		suite.T().Skip("TEST_REPOSITORY environment variable not set")
	}
	return repo
}

// TearDownSuite cleans up after the test suite
func (suite *PullRequestIntegrationSuite) TearDownSuite() {
	// Cleanup if needed
}

// TestListPullRequests tests listing pull requests with real API
func (suite *PullRequestIntegrationSuite) TestListPullRequests() {
	ctx := context.Background()
	
	// Test basic listing
	result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, nil)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	suite.T().Logf("Found %d pull requests", result.Size)
	
	// Test listing with filters
	options := &api.PullRequestListOptions{
		State: "OPEN",
		Sort:  "-updated_on",
	}
	
	openResult, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, options)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), openResult)
	
	suite.T().Logf("Found %d open pull requests", openResult.Size)
	
	// Verify all returned PRs are in OPEN state
	if openResult.Size > 0 {
		for _, value := range openResult.Values {
			prMap, ok := value.(map[string]interface{})
			require.True(suite.T(), ok)
			state, exists := prMap["state"]
			require.True(suite.T(), exists)
			assert.Equal(suite.T(), "OPEN", state)
		}
	}
}

// TestListPullRequestsConvenienceMethods tests convenience methods
func (suite *PullRequestIntegrationSuite) TestListPullRequestsConvenienceMethods() {
	ctx := context.Background()
	
	// Test ListOpenPullRequests
	result, err := suite.client.PullRequests.ListOpenPullRequests(ctx, suite.workspace, suite.repository)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	suite.T().Logf("ListOpenPullRequests found %d pull requests", result.Size)
	
	// If we have a known author, test ListPullRequestsByAuthor
	if testAuthor := suite.testCtx.GetEnv("TEST_AUTHOR"); testAuthor != "" {
		authorResult, err := suite.client.PullRequests.ListPullRequestsByAuthor(ctx, suite.workspace, suite.repository, testAuthor)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), authorResult)
		
		suite.T().Logf("Found %d pull requests by author %s", authorResult.Size, testAuthor)
	}
}

// TestGetPullRequest tests getting a specific pull request
func (suite *PullRequestIntegrationSuite) TestGetPullRequest() {
	ctx := context.Background()
	
	// First, get a list to find an existing PR
	result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, &api.PullRequestListOptions{
		PageLen: 1, // Just get one
	})
	require.NoError(suite.T(), err)
	
	if result.Size == 0 {
		suite.T().Skip("No pull requests found to test with")
		return
	}
	
	// Extract the PR ID from the first result
	firstPR, ok := result.Values[0].(map[string]interface{})
	require.True(suite.T(), ok)
	
	prIDFloat, exists := firstPR["id"].(float64)
	require.True(suite.T(), exists)
	prID := int(prIDFloat)
	
	// Test getting the specific PR
	pr, err := suite.client.PullRequests.GetPullRequest(ctx, suite.workspace, suite.repository, prID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), pr)
	assert.Equal(suite.T(), prID, pr.ID)
	assert.NotEmpty(suite.T(), pr.Title)
	assert.NotEmpty(suite.T(), pr.State)
	
	suite.T().Logf("Retrieved PR #%d: %s (State: %s)", pr.ID, pr.Title, pr.State)
	
	// Test GetPullRequestByID convenience method
	pr2, err := suite.client.PullRequests.GetPullRequestByID(ctx, suite.workspace, suite.repository, prID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), pr.ID, pr2.ID)
	assert.Equal(suite.T(), pr.Title, pr2.Title)
}

// TestGetPullRequestDiff tests getting the diff for a pull request
func (suite *PullRequestIntegrationSuite) TestGetPullRequestDiff() {
	ctx := context.Background()
	
	// Get a PR to test with
	result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, &api.PullRequestListOptions{
		PageLen: 1,
	})
	require.NoError(suite.T(), err)
	
	if result.Size == 0 {
		suite.T().Skip("No pull requests found to test diff with")
		return
	}
	
	firstPR, ok := result.Values[0].(map[string]interface{})
	require.True(suite.T(), ok)
	
	prIDFloat, exists := firstPR["id"].(float64)
	require.True(suite.T(), exists)
	prID := int(prIDFloat)
	
	// Test getting the diff
	diff, err := suite.client.PullRequests.GetPullRequestDiff(ctx, suite.workspace, suite.repository, prID)
	require.NoError(suite.T(), err)
	
	// Diff might be empty for some PRs, but should not error
	suite.T().Logf("Retrieved diff for PR #%d (length: %d)", prID, len(diff))
	
	// If diff is not empty, it should contain typical diff markers
	if len(diff) > 0 {
		// Should contain some diff-like content
		suite.T().Logf("Diff preview (first 200 chars): %s", diff[:min(200, len(diff))])
	}
}

// TestGetPullRequestFiles tests getting the files changed in a pull request
func (suite *PullRequestIntegrationSuite) TestGetPullRequestFiles() {
	ctx := context.Background()
	
	// Get a PR to test with
	result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, &api.PullRequestListOptions{
		PageLen: 1,
	})
	require.NoError(suite.T(), err)
	
	if result.Size == 0 {
		suite.T().Skip("No pull requests found to test files with")
		return
	}
	
	firstPR, ok := result.Values[0].(map[string]interface{})
	require.True(suite.T(), ok)
	
	prIDFloat, exists := firstPR["id"].(float64)
	require.True(suite.T(), exists)
	prID := int(prIDFloat)
	
	// Test getting the files
	diffStat, err := suite.client.PullRequests.GetPullRequestFiles(ctx, suite.workspace, suite.repository, prID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), diffStat)
	
	suite.T().Logf("PR #%d changed %d files (+%d -%d lines)", 
		prID, diffStat.FilesChanged, diffStat.LinesAdded, diffStat.LinesRemoved)
	
	// Log file details if available
	for i, file := range diffStat.Files {
		if i >= 5 { // Limit output
			suite.T().Logf("... and %d more files", len(diffStat.Files)-i)
			break
		}
		suite.T().Logf("  %s: %s (+%d -%d)", file.Status, file.NewPath, file.LinesAdded, file.LinesRemoved)
	}
}

// TestGetPullRequestComments tests getting comments for a pull request
func (suite *PullRequestIntegrationSuite) TestGetPullRequestComments() {
	ctx := context.Background()
	
	// Get a PR to test with
	result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, &api.PullRequestListOptions{
		PageLen: 10, // Get a few to find one with comments
	})
	require.NoError(suite.T(), err)
	
	if result.Size == 0 {
		suite.T().Skip("No pull requests found to test comments with")
		return
	}
	
	// Look for a PR with comments
	var testPRID int
	var found bool
	
	for _, value := range result.Values {
		prMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		
		commentCount, exists := prMap["comment_count"].(float64)
		if exists && commentCount > 0 {
			prIDFloat, exists := prMap["id"].(float64)
			if exists {
				testPRID = int(prIDFloat)
				found = true
				break
			}
		}
	}
	
	if !found {
		// Just use the first PR, even if it has no comments
		firstPR, ok := result.Values[0].(map[string]interface{})
		require.True(suite.T(), ok)
		prIDFloat, exists := firstPR["id"].(float64)
		require.True(suite.T(), exists)
		testPRID = int(prIDFloat)
	}
	
	// Test getting comments
	comments, err := suite.client.PullRequests.GetComments(ctx, suite.workspace, suite.repository, testPRID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), comments)
	
	suite.T().Logf("PR #%d has %d comments", testPRID, comments.Size)
}

// TestPullRequestValidation tests validation errors
func (suite *PullRequestIntegrationSuite) TestPullRequestValidation() {
	ctx := context.Background()
	
	// Test invalid workspace
	_, err := suite.client.PullRequests.ListPullRequests(ctx, "", suite.repository, nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "workspace")
	
	// Test invalid repository
	_, err = suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, "", nil)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "repository slug")
	
	// Test invalid PR ID
	_, err = suite.client.PullRequests.GetPullRequest(ctx, suite.workspace, suite.repository, 0)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "positive")
	
	_, err = suite.client.PullRequests.GetPullRequest(ctx, suite.workspace, suite.repository, -1)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "positive")
}

// TestNonExistentPullRequest tests handling of non-existent pull requests
func (suite *PullRequestIntegrationSuite) TestNonExistentPullRequest() {
	ctx := context.Background()
	
	// Use a very high PR ID that's unlikely to exist
	nonExistentID := 999999
	
	// Test getting non-existent PR
	_, err := suite.client.PullRequests.GetPullRequest(ctx, suite.workspace, suite.repository, nonExistentID)
	assert.Error(suite.T(), err)
	
	// Test GetPullRequestByID convenience method (should convert to NotFoundError)
	_, err = suite.client.PullRequests.GetPullRequestByID(ctx, suite.workspace, suite.repository, nonExistentID)
	assert.Error(suite.T(), err)
	
	// Check if it's properly converted to BitbucketError with NotFound type
	var bbErr *api.BitbucketError
	if assert.ErrorAs(suite.T(), err, &bbErr) {
		assert.Equal(suite.T(), api.ErrorTypeNotFound, bbErr.Type)
		suite.T().Logf("Correctly converted to BitbucketError with NotFound type: %s", bbErr.Error())
	}
}

// TestAPIRateLimiting tests that the client handles rate limiting properly
func (suite *PullRequestIntegrationSuite) TestAPIRateLimiting() {
	ctx := context.Background()
	
	// Make several requests quickly to test rate limiting handling
	for i := 0; i < 5; i++ {
		result, err := suite.client.PullRequests.ListPullRequests(ctx, suite.workspace, suite.repository, &api.PullRequestListOptions{
			PageLen: 1,
		})
		
		// Should not error due to rate limiting (client should handle it)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		
		suite.T().Logf("Request %d completed successfully", i+1)
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSuite runner
func TestPullRequestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PullRequestIntegrationSuite))
}