package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext provides a shared context for tests with common utilities
type TestContext struct {
	T           *testing.T
	TempDir     string
	ConfigFile  string
	TestRepo    string
	TestRepoURL string
	Cleanup     []func()
}

// NewTestContext creates a new test context with temporary directory and config
func NewTestContext(t *testing.T) *TestContext {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")
	
	tc := &TestContext{
		T:          t,
		TempDir:    tempDir,
		ConfigFile: configFile,
		TestRepo:   GetTestRepo(),
		Cleanup:    make([]func(), 0),
	}
	
	// Create default test config
	tc.CreateTestConfig()
	
	return tc
}

// Close runs cleanup functions
func (tc *TestContext) Close() {
	for _, cleanup := range tc.Cleanup {
		cleanup()
	}
}

// CreateTestConfig creates a test configuration file
func (tc *TestContext) CreateTestConfig() {
	config := `
bitbucket:
  base_url: https://api.bitbucket.org/2.0
  auth_method: app_password
  username: ` + GetTestUsername() + `
  app_password: ` + GetTestAppPassword() + `
  
output:
  format: table
  no_color: false
  
repository:
  default_workspace: ` + GetTestWorkspace() + `
  default_repo: ` + GetTestRepo() + `
`
	
	err := os.WriteFile(tc.ConfigFile, []byte(config), 0644)
	require.NoError(tc.T, err, "Failed to create test config")
}

// GetTestUsername returns the test username from environment
func GetTestUsername() string {
	username := os.Getenv("BT_TEST_USERNAME")
	if username == "" {
		return "test-user"
	}
	return username
}

// GetTestAppPassword returns the test app password from environment
func GetTestAppPassword() string {
	password := os.Getenv("BT_TEST_APP_PASSWORD")
	if password == "" {
		return "test-password"
	}
	return password
}

// GetTestWorkspace returns the test workspace from environment
func GetTestWorkspace() string {
	workspace := os.Getenv("BT_TEST_WORKSPACE")
	if workspace == "" {
		return "test-workspace"
	}
	return workspace
}

// GetTestRepo returns the test repository from environment
func GetTestRepo() string {
	repo := os.Getenv("BT_TEST_REPO")
	if repo == "" {
		return "test-repo"
	}
	return repo
}

// IsIntegrationTest checks if integration tests should run
func IsIntegrationTest() bool {
	return os.Getenv("BT_INTEGRATION_TESTS") == "1"
}

// SkipIfNoIntegration skips the test if integration tests are disabled
func SkipIfNoIntegration(t *testing.T) {
	if !IsIntegrationTest() {
		t.Skip("Skipping integration test. Set BT_INTEGRATION_TESTS=1 to run.")
	}
}

// RequireIntegrationEnv ensures required environment variables are set for integration tests
func RequireIntegrationEnv(t *testing.T) {
	SkipIfNoIntegration(t)
	
	required := []string{
		"BT_TEST_USERNAME",
		"BT_TEST_APP_PASSWORD",
		"BT_TEST_WORKSPACE",
		"BT_TEST_REPO",
	}
	
	for _, env := range required {
		if os.Getenv(env) == "" {
			t.Fatalf("Required environment variable %s is not set", env)
		}
	}
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
}

// RunCommand executes a command and returns the result
func RunCommand(name string, args ...string) *CommandResult {
	cmd := exec.Command(name, args...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}
	
	return &CommandResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Error:    err,
	}
}

// RunBTCommand executes the bt command with given arguments
func RunBTCommand(args ...string) *CommandResult {
	// Use the built binary from the build directory
	btPath := filepath.Join("..", "..", "build", "bt")
	if _, err := os.Stat(btPath); os.IsNotExist(err) {
		// Fallback to current directory bt binary
		btPath = filepath.Join("..", "..", "bt")
	}
	
	return RunCommand(btPath, args...)
}

// RunBTCommandWithConfig executes the bt command with a specific config file
func RunBTCommandWithConfig(configPath string, args ...string) *CommandResult {
	allArgs := append([]string{"--config", configPath}, args...)
	return RunBTCommand(allArgs...)
}

// AssertCommandSuccess asserts that a command executed successfully
func AssertCommandSuccess(t *testing.T, result *CommandResult, msgAndArgs ...interface{}) {
	if result.ExitCode != 0 {
		t.Errorf("Command failed with exit code %d\nStdout: %s\nStderr: %s\nError: %v", 
			result.ExitCode, result.Stdout, result.Stderr, result.Error)
		if len(msgAndArgs) > 0 {
			t.Errorf("Additional info: %v", msgAndArgs)
		}
		t.FailNow()
	}
}

// AssertCommandFailure asserts that a command failed with expected exit code
func AssertCommandFailure(t *testing.T, result *CommandResult, expectedExitCode int, msgAndArgs ...interface{}) {
	if result.ExitCode != expectedExitCode {
		t.Errorf("Expected exit code %d, got %d\nStdout: %s\nStderr: %s\nError: %v", 
			expectedExitCode, result.ExitCode, result.Stdout, result.Stderr, result.Error)
		if len(msgAndArgs) > 0 {
			t.Errorf("Additional info: %v", msgAndArgs)
		}
		t.FailNow()
	}
}

// AssertJSONOutput validates that the output is valid JSON and matches expected structure
func AssertJSONOutput(t *testing.T, output string, msgAndArgs ...interface{}) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Output is not valid JSON: %s", output)
	return result
}

// HTTPClient provides a configured HTTP client for API testing
type HTTPClient struct {
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
}

// NewHTTPClient creates a new HTTP client for API testing
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		BaseURL:  "https://api.bitbucket.org/2.0",
		Username: GetTestUsername(),
		Password: GetTestAppPassword(),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request to the API
func (c *HTTPClient) Get(endpoint string) (*http.Response, error) {
	return c.Request("GET", endpoint, nil)
}

// Post performs a POST request to the API
func (c *HTTPClient) Post(endpoint string, body io.Reader) (*http.Response, error) {
	return c.Request("POST", endpoint, body)
}

// Put performs a PUT request to the API
func (c *HTTPClient) Put(endpoint string, body io.Reader) (*http.Response, error) {
	return c.Request("PUT", endpoint, body)
}

// Delete performs a DELETE request to the API
func (c *HTTPClient) Delete(endpoint string) (*http.Response, error) {
	return c.Request("DELETE", endpoint, nil)
}

// Request performs a generic HTTP request to the API
func (c *HTTPClient) Request(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bt-test-client/1.0")
	
	return c.Client.Do(req)
}

// ReadJSONResponse reads and parses a JSON response
func ReadJSONResponse(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nBody: %s", err, string(body))
	}
	
	return result, nil
}

// RateLimitWait waits for rate limit reset if needed
func RateLimitWait(resp *http.Response) {
	if resp.StatusCode == 429 {
		resetHeader := resp.Header.Get("X-RateLimit-Reset")
		if resetHeader != "" {
			// Parse reset time and wait
			time.Sleep(1 * time.Second) // Simple fallback
		}
	}
}

// CreateGitRepository creates a temporary git repository for testing
func CreateGitRepository(t *testing.T, dir string) {
	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
		{"git", "add", "."},
		{"git", "commit", "-m", "Initial commit", "--allow-empty"},
	}
	
	for _, cmd := range commands {
		result := RunCommand(cmd[0], cmd[1:]...)
		if result.ExitCode != 0 {
			t.Fatalf("Failed to run git command %v: %s", cmd, result.Stderr)
		}
	}
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file and returns its content
func ReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read file %s", path)
	return string(content)
}

// WriteFile writes content to a file
func WriteFile(t *testing.T, path, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to write file %s", path)
}

// AssertFileContent asserts that a file contains expected content
func AssertFileContent(t *testing.T, path, expected string) {
	actual := ReadFile(t, path)
	assert.Equal(t, expected, actual, "File content mismatch for %s", path)
}

// AssertFileContains asserts that a file contains a substring
func AssertFileContains(t *testing.T, path, substring string) {
	content := ReadFile(t, path)
	assert.Contains(t, content, substring, "File %s should contain %s", path, substring)
}

// GetProjectRoot returns the project root directory
func GetProjectRoot() string {
	// Walk up the directory tree to find go.mod
	dir, _ := os.Getwd()
	for {
		if FileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// BuildBinary builds the bt binary for testing
func BuildBinary(t *testing.T) string {
	projectRoot := GetProjectRoot()
	if projectRoot == "" {
		t.Fatal("Could not find project root")
	}
	
	buildDir := filepath.Join(projectRoot, "build")
	err := os.MkdirAll(buildDir, 0755)
	require.NoError(t, err, "Failed to create build directory")
	
	binaryPath := filepath.Join(buildDir, "bt")
	cmdDir := filepath.Join(projectRoot, "cmd", "bt")
	
	result := RunCommand("go", "build", "-o", binaryPath, cmdDir)
	require.Equal(t, 0, result.ExitCode, "Failed to build binary: %s", result.Stderr)
	
	return binaryPath
}

// Performance measurement utilities

// BenchmarkResult represents a benchmark measurement
type BenchmarkResult struct {
	Name     string
	Duration time.Duration
	Memory   int64
	Allocs   int64
}

// BenchmarkFunc runs a function and measures its performance
func BenchmarkFunc(name string, fn func()) *BenchmarkResult {
	// Simple benchmark without runtime.GC() calls for basic measurement
	start := time.Now()
	fn()
	duration := time.Since(start)
	
	return &BenchmarkResult{
		Name:     name,
		Duration: duration,
		Memory:   0, // Would need runtime.MemStats for actual memory measurement
		Allocs:   0, // Would need runtime.MemStats for actual allocation measurement
	}
}

// AssertPerformance asserts that a benchmark meets performance requirements
func AssertPerformance(t *testing.T, result *BenchmarkResult, maxDuration time.Duration) {
	if result.Duration > maxDuration {
		t.Errorf("Performance requirement failed for %s: took %v, expected <= %v", 
			result.Name, result.Duration, maxDuration)
	}
}

// Test data generation utilities

// GenerateTestData generates test data for various scenarios
func GenerateTestData() map[string]interface{} {
	return map[string]interface{}{
		"repository": map[string]interface{}{
			"name":        "test-repo",
			"full_name":   "test-workspace/test-repo",
			"description": "Test repository for bt CLI",
			"is_private":  false,
			"created_on":  "2023-01-01T00:00:00.000000+00:00",
			"updated_on":  "2023-01-01T00:00:00.000000+00:00",
			"language":    "Go",
			"size":        1024,
		},
		"pull_request": map[string]interface{}{
			"id":          1,
			"title":       "Test PR",
			"description": "Test pull request",
			"state":       "OPEN",
			"created_on":  "2023-01-01T00:00:00.000000+00:00",
			"updated_on":  "2023-01-01T00:00:00.000000+00:00",
			"source": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "feature-branch",
				},
			},
			"destination": map[string]interface{}{
				"branch": map[string]interface{}{
					"name": "main",
				},
			},
		},
		"pipeline": map[string]interface{}{
			"uuid":       "{12345678-1234-1234-1234-123456789012}",
			"build_number": 1,
			"state": map[string]interface{}{
				"name":   "SUCCESSFUL",
				"type":   "pipeline_state",
				"result": map[string]interface{}{
					"name": "SUCCESSFUL",
					"type": "pipeline_state_result",
				},
			},
			"created_on": "2023-01-01T00:00:00.000000+00:00",
			"completed_on": "2023-01-01T00:05:00.000000+00:00",
		},
	}
}

// CleanupTestData provides cleanup for test data
func CleanupTestData(t *testing.T, cleanup func()) {
	t.Cleanup(cleanup)
}

// CreateTestAuthManager creates an auth manager for testing
func CreateTestAuthManager() (auth.AuthManager, error) {
	email := GetTestUsername()
	token := GetTestAppPassword()
	
	// Set environment variables for API token auth
	os.Setenv("BITBUCKET_EMAIL", email)
	os.Setenv("BITBUCKET_API_TOKEN", token)
	
	// Create storage
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		return nil, err
	}
	
	// Create config
	config := auth.DefaultConfig()
	
	// Create auth manager
	return auth.NewAuthManager(config, storage)
}

// GetTestLogger returns a logger for testing
func GetTestLogger() *log.Logger {
	return log.New(os.Stdout, "[TEST] ", log.LstdFlags)
}