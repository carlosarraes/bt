package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/version"
)

const (
	// DefaultBaseURL is the default Bitbucket API base URL
	DefaultBaseURL = "https://api.bitbucket.org/2.0"

	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second

	// DefaultRetryAttempts is the default number of retry attempts
	DefaultRetryAttempts = 3

	// DefaultPageSize is the default page size for paginated requests
	DefaultPageSize = 50

	// MaxPageSize is the maximum page size allowed by Bitbucket
	MaxPageSize = 100
)

// ClientConfig contains configuration options for the API client
type ClientConfig struct {
	BaseURL       string
	Timeout       time.Duration
	RetryAttempts int
	EnableLogging bool
	Logger        *log.Logger
	UserAgent     string
}

// DefaultClientConfig returns a configuration with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:       DefaultBaseURL,
		Timeout:       DefaultTimeout,
		RetryAttempts: DefaultRetryAttempts,
		EnableLogging: false,
		UserAgent:     fmt.Sprintf("bt/%s", version.Version),
	}
}

// Client is the main Bitbucket API client
type Client struct {
	httpClient  *http.Client
	authManager auth.AuthManager
	config      *ClientConfig
	baseURL     *url.URL

	// Services
	Pipelines    *PipelineService
	PullRequests *PullRequestService
	Repositories *RepositoryService
}

// NewClient creates a new Bitbucket API client
func NewClient(authManager auth.AuthManager, config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	client := &Client{
		httpClient:  httpClient,
		authManager: authManager,
		config:      config,
		baseURL:     baseURL,
	}

	// Initialize services
	client.Pipelines = NewPipelineService(client)
	client.PullRequests = NewPullRequestService(client)
	client.Repositories = NewRepositoryService(client)

	return client, nil
}

// Get performs a GET request to the specified endpoint
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Request(ctx, "GET", endpoint, nil)
}

// Post performs a POST request to the specified endpoint with JSON body
func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.Request(ctx, "POST", endpoint, body)
}

// Put performs a PUT request to the specified endpoint with JSON body
func (c *Client) Put(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.Request(ctx, "PUT", endpoint, body)
}

// Delete performs a DELETE request to the specified endpoint
func (c *Client) Delete(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Request(ctx, "DELETE", endpoint, nil)
}

// getLogsRequest performs a GET request for logs with appropriate headers (mimicking xh/HTTPie)
func (c *Client) getLogsRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	// Build the full URL
	fullURL, err := c.buildURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers to match xh/HTTPie defaults (which work for the user)
	req.Header.Set("Accept", "*/*") // xh uses */* by default, not text/plain
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("User-Agent", "bt/1.0.0") // Simple user agent like xh

	// Add authentication headers
	if c.authManager != nil {
		if err := c.authManager.SetHTTPHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	// Perform the request with retries
	return c.doRequestWithRetry(req)
}

// Request performs an HTTP request with automatic retries and error handling
func (c *Client) Request(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	// Build the full URL
	fullURL, err := c.buildURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create the request
	req, err := c.createRequest(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Perform the request with retries
	return c.doRequestWithRetry(req)
}

// GetJSON performs a GET request and unmarshals the JSON response
func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

// PostJSON performs a POST request with JSON body and unmarshals the JSON response
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result interface{}) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// PutJSON performs a PUT request with JSON body and unmarshals the JSON response
func (c *Client) PutJSON(ctx context.Context, endpoint string, body, result interface{}) error {
	resp, err := c.Put(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// Paginate creates a new paginator for the given endpoint
func (c *Client) Paginate(endpoint string, options *PageOptions) *Paginator {
	fullURL, _ := c.buildURL(endpoint)
	return NewPaginator(c, fullURL, options)
}

// buildURL constructs the full URL for an endpoint
func (c *Client) buildURL(endpoint string) (string, error) {
	// Remove leading slash if present
	endpoint = strings.TrimPrefix(endpoint, "/")

	// Ensure base URL ends with slash for proper URL joining
	baseURLStr := c.baseURL.String()
	if !strings.HasSuffix(baseURLStr, "/") {
		baseURLStr += "/"
	}

	// Parse endpoint as URL to handle query parameters
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	// Parse the base URL with trailing slash
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Resolve against base URL
	fullURL := baseURL.ResolveReference(endpointURL)
	return fullURL.String(), nil
}

// createRequest creates an HTTP request with proper headers
func (c *Client) createRequest(ctx context.Context, method, url string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader

	// Handle request body
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	// Add authentication headers
	if c.authManager != nil {
		if err := c.authManager.SetHTTPHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	return req, nil
}

// doRequestWithRetry performs the HTTP request with retry logic
func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		// Clone request for retry (body may be consumed)
		reqClone, err := c.cloneRequest(req)
		if err != nil {
			return nil, fmt.Errorf("failed to clone request: %w", err)
		}

		// Log request if enabled
		c.logRequest(reqClone)

		// Perform the request
		resp, err := c.doRequest(reqClone)
		if err != nil {
			lastErr = err

			// Don't retry certain network errors
			if !isRetryableNetworkError(err) {
				return nil, err
			}

			c.logError(fmt.Sprintf("Request failed (attempt %d/%d): %v",
				attempt+1, c.config.RetryAttempts+1, err))

			if attempt < c.config.RetryAttempts {
				c.waitBeforeRetry(attempt)
				continue
			}

			return nil, lastErr
		}

		// Log response if enabled
		c.logResponse(resp)

		// Check if we should retry based on status code
		if c.shouldRetry(resp, attempt) {
			resp.Body.Close()

			// Handle rate limiting with special backoff
			if resp.StatusCode == 429 {
				c.handleRateLimit(resp, attempt)
			} else {
				c.waitBeforeRetry(attempt)
			}

			continue
		}

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			err := ParseError(resp)
			resp.Body.Close()
			return nil, err
		}

		return resp, nil
	}

	return nil, lastErr
}

// doRequest performs the actual HTTP request (used internally)
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// cloneRequest creates a copy of the HTTP request for retries
func (c *Client) cloneRequest(req *http.Request) (*http.Request, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	clonedReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for key, values := range req.Header {
		for _, value := range values {
			clonedReq.Header.Add(key, value)
		}
	}

	return clonedReq, nil
}

// shouldRetry determines if a request should be retried based on the response
func (c *Client) shouldRetry(resp *http.Response, attempt int) bool {
	if attempt >= c.config.RetryAttempts {
		return false
	}

	// Retry on rate limit
	if resp.StatusCode == 429 {
		return true
	}

	// Retry on server errors
	if resp.StatusCode >= 500 {
		return true
	}

	return false
}

// handleRateLimit handles rate limiting with proper backoff
func (c *Client) handleRateLimit(resp *http.Response, attempt int) {
	// Check for Retry-After header
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			duration := time.Duration(seconds) * time.Second
			c.logError(fmt.Sprintf("Rate limited, waiting %v (Retry-After header)", duration))
			time.Sleep(duration)
			return
		}
	}

	// Fallback to exponential backoff
	c.waitBeforeRetry(attempt)
}

// waitBeforeRetry implements exponential backoff with jitter
func (c *Client) waitBeforeRetry(attempt int) {
	// Exponential backoff: 1s, 2s, 4s, 8s...
	baseDelay := time.Duration(math.Pow(2, float64(attempt))) * time.Second

	// Add jitter (±25%)
	jitter := time.Duration(rand.Float64() * 0.5 * float64(baseDelay))
	delay := baseDelay + jitter - time.Duration(0.25*float64(baseDelay))

	// Cap at 30 seconds
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	c.logError(fmt.Sprintf("Waiting %v before retry", delay))
	time.Sleep(delay)
}

// isRetryableNetworkError checks if a network error is retryable
func isRetryableNetworkError(err error) bool {
	// In a real implementation, you'd check for specific network errors
	// For now, we'll be conservative and only retry on timeout errors
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "temporary failure")
}

// logRequest logs HTTP requests if logging is enabled
func (c *Client) logRequest(req *http.Request) {
	if !c.config.EnableLogging || c.config.Logger == nil {
		return
	}

	c.config.Logger.Printf("Request: %s %s", req.Method, req.URL.String())
}

// logResponse logs HTTP responses if logging is enabled
func (c *Client) logResponse(resp *http.Response) {
	if !c.config.EnableLogging || c.config.Logger == nil {
		return
	}

	c.config.Logger.Printf("Response: %d %s", resp.StatusCode, resp.Status)
}

// logError logs errors if logging is enabled
func (c *Client) logError(message string) {
	if !c.config.EnableLogging || c.config.Logger == nil {
		return
	}

	c.config.Logger.Printf("Error: %s", message)
}

// SetAuthManager updates the authentication manager
func (c *Client) SetAuthManager(authManager auth.AuthManager) {
	c.authManager = authManager
}

// GetAuthManager returns the current authentication manager
func (c *Client) GetAuthManager() auth.AuthManager {
	return c.authManager
}

// SetTimeout updates the request timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
	c.config.Timeout = timeout
}

// EnableLogging enables or disables request/response logging
func (c *Client) EnableLogging(enabled bool, logger *log.Logger) {
	c.config.EnableLogging = enabled
	c.config.Logger = logger
}

// BaseURL returns the base URL being used
func (c *Client) BaseURL() string {
	return c.baseURL.String()
}
