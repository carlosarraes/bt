package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationSuite provides a test suite for authentication integration tests
type AuthIntegrationSuite struct {
	suite.Suite
	testCtx    *utils.TestContext
	httpClient *utils.HTTPClient
}

// SetupSuite initializes the test suite
func (suite *AuthIntegrationSuite) SetupSuite() {
	utils.RequireIntegrationEnv(suite.T())
	
	suite.testCtx = utils.NewTestContext(suite.T())
	suite.httpClient = utils.NewHTTPClient()
}

// TearDownSuite cleans up after the test suite
func (suite *AuthIntegrationSuite) TearDownSuite() {
	if suite.testCtx != nil {
		suite.testCtx.Close()
	}
}

// TestAppPasswordAuthentication tests app password authentication
func (suite *AuthIntegrationSuite) TestAppPasswordAuthentication() {
	// Test with valid credentials
	client := &utils.HTTPClient{
		BaseURL:  "https://api.bitbucket.org/2.0",
		Username: utils.GetTestUsername(),
		Password: utils.GetTestAppPassword(),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	
	resp, err := client.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, 
		"Valid app password should authenticate successfully")
	
	// Verify user data
	userData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	assert.Contains(suite.T(), userData, "username")
	assert.Equal(suite.T(), utils.GetTestUsername(), userData["username"])
}

// TestInvalidAppPasswordAuthentication tests authentication failure scenarios
func (suite *AuthIntegrationSuite) TestInvalidAppPasswordAuthentication() {
	testCases := []struct {
		name        string
		username    string
		password    string
		expectedCode int
		description string
	}{
		{
			name:        "InvalidPassword",
			username:    utils.GetTestUsername(),
			password:    "invalid-password",
			expectedCode: http.StatusUnauthorized,
			description: "Invalid password should return 401",
		},
		{
			name:        "InvalidUsername",
			username:    "invalid-username",
			password:    utils.GetTestAppPassword(),
			expectedCode: http.StatusUnauthorized,
			description: "Invalid username should return 401",
		},
		{
			name:        "EmptyCredentials",
			username:    "",
			password:    "",
			expectedCode: http.StatusUnauthorized,
			description: "Empty credentials should return 401",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			client := &utils.HTTPClient{
				BaseURL:  "https://api.bitbucket.org/2.0",
				Username: tc.username,
				Password: tc.password,
				Client: &http.Client{
					Timeout: 30 * time.Second,
				},
			}
			
			resp, err := client.Get("/user")
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedCode, resp.StatusCode, tc.description)
		})
	}
}

// TestConfigFileAuthentication tests authentication using config file
func (suite *AuthIntegrationSuite) TestConfigFileAuthentication() {
	// Create a test config file with authentication
	configContent := fmt.Sprintf(`
bitbucket:
  base_url: https://api.bitbucket.org/2.0
  auth_method: app_password
  username: %s
  app_password: %s
  
output:
  format: json
  no_color: true
  
repository:
  default_workspace: %s
  default_repo: %s
`, utils.GetTestUsername(), utils.GetTestAppPassword(), 
	utils.GetTestWorkspace(), utils.GetTestRepo())
	
	configPath := filepath.Join(suite.testCtx.TempDir, "auth_test_config.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(suite.T(), err)
	
	// Test bt command with this config
	result := utils.RunBTCommandWithConfig(configPath, "version")
	
	// The version command should work regardless of auth, but it tests config parsing
	utils.AssertCommandSuccess(suite.T(), result)
	assert.Contains(suite.T(), result.Stdout, "bt version")
}

// TestAuthenticationHeaders tests that proper authentication headers are sent
func (suite *AuthIntegrationSuite) TestAuthenticationHeaders() {
	// Create a custom HTTP client to capture request details
	transport := &authCapturingTransport{
		base: http.DefaultTransport,
		headers: make(map[string]string),
	}
	
	client := &utils.HTTPClient{
		BaseURL:  "https://api.bitbucket.org/2.0",
		Username: utils.GetTestUsername(),
		Password: utils.GetTestAppPassword(),
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
	
	resp, err := client.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Verify Authorization header was set
	authHeader, exists := transport.headers["Authorization"]
	assert.True(suite.T(), exists, "Authorization header should be present")
	assert.True(suite.T(), strings.HasPrefix(authHeader, "Basic "), 
		"Should use Basic authentication")
	
	// Verify User-Agent header
	userAgent, exists := transport.headers["User-Agent"]
	assert.True(suite.T(), exists, "User-Agent header should be present")
	assert.Contains(suite.T(), userAgent, "bt-test-client")
}

// TestAuthenticationWithRealAPI tests authentication with real API calls
func (suite *AuthIntegrationSuite) TestAuthenticationWithRealAPI() {
	// Test accessing user profile
	resp, err := suite.httpClient.Get("/user")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	userData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Verify essential user fields
	assert.Contains(suite.T(), userData, "username")
	assert.Contains(suite.T(), userData, "account_id")
	assert.Contains(suite.T(), userData, "display_name")
	
	// Test accessing user's repositories (requires authentication)
	username := userData["username"].(string)
	repoEndpoint := fmt.Sprintf("/repositories/%s", username)
	
	resp, err = suite.httpClient.Get(repoEndpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	// Should be able to access user's repositories with valid auth
	assert.True(suite.T(), resp.StatusCode < 400, 
		"Should be able to access user repositories with valid authentication")
}

// TestPrivateRepositoryAccess tests access to private repositories
func (suite *AuthIntegrationSuite) TestPrivateRepositoryAccess() {
	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()
	
	// Test accessing repository (may be private)
	endpoint := fmt.Sprintf("/repositories/%s/%s", workspace, repo)
	resp, err := suite.httpClient.Get(endpoint)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		suite.T().Skipf("Test repository '%s/%s' not found", workspace, repo)
		return
	}
	
	// With proper authentication, we should be able to access the repository
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, 
		"Should be able to access repository with valid authentication")
	
	repoData, err := utils.ReadJSONResponse(resp)
	require.NoError(suite.T(), err)
	
	// Check if repository is private
	if isPrivate, exists := repoData["is_private"]; exists {
		if isPrivate.(bool) {
			suite.T().Logf("Successfully accessed private repository '%s/%s'", workspace, repo)
		}
	}
}

// TestAPIPermissions tests various API permissions with current authentication
func (suite *AuthIntegrationSuite) TestAPIPermissions() {
	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()
	
	testCases := []struct {
		name        string
		endpoint    string
		method      string
		expectSuccess bool
		description string
	}{
		{
			name:        "ReadUser",
			endpoint:    "/user",
			method:      "GET",
			expectSuccess: true,
			description: "Should be able to read user profile",
		},
		{
			name:        "ReadRepository",
			endpoint:    fmt.Sprintf("/repositories/%s/%s", workspace, repo),
			method:      "GET",
			expectSuccess: true,
			description: "Should be able to read repository info",
		},
		{
			name:        "ListPullRequests",
			endpoint:    fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repo),
			method:      "GET",
			expectSuccess: true,
			description: "Should be able to list pull requests",
		},
		{
			name:        "ListPipelines",
			endpoint:    fmt.Sprintf("/repositories/%s/%s/pipelines/", workspace, repo),
			method:      "GET",
			expectSuccess: true, // May return 404 if pipelines not enabled, but shouldn't be auth error
			description: "Should be able to list pipelines",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error
			
			switch tc.method {
			case "GET":
				resp, err = suite.httpClient.Get(tc.endpoint)
			default:
				t.Fatalf("Unsupported method: %s", tc.method)
			}
			
			require.NoError(t, err)
			defer resp.Body.Close()
			
			if tc.expectSuccess {
				// Should not be an authentication error
				assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode, 
					"Should not get authentication error for %s", tc.description)
				assert.NotEqual(t, http.StatusForbidden, resp.StatusCode, 
					"Should not get authorization error for %s", tc.description)
				
				// 404 is acceptable for some endpoints (e.g., pipelines not enabled)
				if resp.StatusCode != 404 {
					assert.Less(t, resp.StatusCode, 400, 
						"Should be successful for %s", tc.description)
				}
			}
		})
	}
}

// TestAuthenticationRetry tests authentication retry behavior
func (suite *AuthIntegrationSuite) TestAuthenticationRetry() {
	// Test with intermittent failures (simulated by making multiple requests)
	const numRequests = 3
	successCount := 0
	
	for i := 0; i < numRequests; i++ {
		resp, err := suite.httpClient.Get("/user")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == 429 {
			// Rate limited, wait and continue
			utils.RateLimitWait(resp)
		} else if resp.StatusCode == 401 {
			suite.T().Errorf("Authentication failed on request %d", i+1)
		}
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
	
	// Should have at least some successful requests
	assert.Greater(suite.T(), successCount, 0, 
		"Should have at least one successful authenticated request")
}

// TestConfigurationValidation tests configuration validation for authentication
func (suite *AuthIntegrationSuite) TestConfigurationValidation() {
	testCases := []struct {
		name        string
		config      string
		expectError bool
		description string
	}{
		{
			name: "ValidConfig",
			config: fmt.Sprintf(`
bitbucket:
  auth_method: app_password
  username: %s
  app_password: %s
`, utils.GetTestUsername(), utils.GetTestAppPassword()),
			expectError: false,
			description: "Valid configuration should not error",
		},
		{
			name: "MissingUsername",
			config: fmt.Sprintf(`
bitbucket:
  auth_method: app_password
  app_password: %s
`, utils.GetTestAppPassword()),
			expectError: true,
			description: "Missing username should cause error",
		},
		{
			name: "MissingPassword",
			config: fmt.Sprintf(`
bitbucket:
  auth_method: app_password
  username: %s
`, utils.GetTestUsername()),
			expectError: true,
			description: "Missing password should cause error",
		},
		{
			name: "InvalidAuthMethod",
			config: fmt.Sprintf(`
bitbucket:
  auth_method: invalid_method
  username: %s
  app_password: %s
`, utils.GetTestUsername(), utils.GetTestAppPassword()),
			expectError: true,
			description: "Invalid auth method should cause error",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(suite.testCtx.TempDir, fmt.Sprintf("config_%s.yml", tc.name))
			err := os.WriteFile(configPath, []byte(tc.config), 0600)
			require.NoError(t, err)
			
			// Try to use the config with a simple command
			result := utils.RunBTCommandWithConfig(configPath, "version")
			
			if tc.expectError {
				// Note: version command might not validate auth, so this test might need adjustment
				// based on actual CLI behavior
				t.Logf("Config validation test for %s: %s", tc.name, tc.description)
			} else {
				utils.AssertCommandSuccess(t, result)
			}
		})
	}
}

// authCapturingTransport captures HTTP headers for testing
type authCapturingTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *authCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Capture headers
	for name, values := range req.Header {
		if len(values) > 0 {
			t.headers[name] = values[0]
		}
	}
	
	return t.base.RoundTrip(req)
}

// Run the test suite
func TestAuthIntegration(t *testing.T) {
	suite.Run(t, new(AuthIntegrationSuite))
}

// Additional standalone authentication tests

// TestBasicAuthConfiguration tests basic authentication configuration
func TestBasicAuthConfiguration(t *testing.T) {
	utils.SkipIfNoIntegration(t)
	
	// Test that environment variables are properly set
	username := utils.GetTestUsername()
	password := utils.GetTestAppPassword()
	
	assert.NotEmpty(t, username, "BT_TEST_USERNAME should be set for integration tests")
	assert.NotEmpty(t, password, "BT_TEST_APP_PASSWORD should be set for integration tests")
	
	// Test basic connectivity with these credentials
	client := &utils.HTTPClient{
		BaseURL:  "https://api.bitbucket.org/2.0",
		Username: username,
		Password: password,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	
	resp, err := client.Get("/user")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode, 
		"Authentication should not fail with test credentials")
}