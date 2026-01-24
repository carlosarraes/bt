package sonarcloud

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/carlosarraes/bt/pkg/version"
)

const (
	DefaultBaseURL = "https://sonarcloud.io/api"

	DefaultTimeout = 30 * time.Second

	DefaultRetryAttempts = 3

	DefaultCacheTTL = 1 * time.Hour
)

type ClientConfig struct {
	BaseURL       string
	Token         string
	Timeout       time.Duration
	RetryAttempts int
	MaxDelay      time.Duration
	BaseDelay     time.Duration
	Jitter        bool
	EnableLogging bool
	EnableCache   bool
	UserAgent     string
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:       DefaultBaseURL,
		Token:         os.Getenv("SONARCLOUD_TOKEN"),
		Timeout:       DefaultTimeout,
		RetryAttempts: DefaultRetryAttempts,
		MaxDelay:      60 * time.Second,
		BaseDelay:     1 * time.Second,
		Jitter:        true,
		EnableLogging: false,
		EnableCache:   true,
		UserAgent:     fmt.Sprintf("bt/%s", version.Version),
	}
}

type APIContext struct {
	ProjectKey        string
	BaseParams        map[string]string
	IsPullRequest     bool
	PullRequestID     int
	PreferredMetrics  []string
	PipelineCompleted bool
	PipelineRunning   bool
}

type CacheEntry struct {
	Data        interface{}
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ETag        string
	RequestHash string
}

type Cache struct {
	entries    map[string]*CacheEntry
	mutex      sync.RWMutex
	enabled    bool
	defaultTTL time.Duration
}

func NewCache(enabled bool, defaultTTL time.Duration) *Cache {
	return &Cache{
		entries:    make(map[string]*CacheEntry),
		enabled:    enabled,
		defaultTTL: defaultTTL,
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		go func() {
			c.mutex.Lock()
			delete(c.entries, key)
			c.mutex.Unlock()
		}()
		return nil, false
	}

	return entry.Data, true
}

func (c *Cache) Set(key string, data interface{}, ttl time.Duration) {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	c.entries[key] = &CacheEntry{
		Data:      data,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

type Client struct {
	httpClient *http.Client
	config     *ClientConfig
	baseURL    *url.URL
	cache      *Cache
}

func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	if config.Token == "" {
		return nil, fmt.Errorf("SonarCloud token is required. Set SONARCLOUD_TOKEN environment variable")
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	cache := NewCache(config.EnableCache, DefaultCacheTTL)

	return &Client{
		httpClient: httpClient,
		config:     config,
		baseURL:    baseURL,
		cache:      cache,
	}, nil
}

func (c *Client) buildCacheKey(endpoint string, params map[string]string, context APIContext) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var paramStr strings.Builder
	for _, k := range keys {
		paramStr.WriteString(fmt.Sprintf("%s=%s&", k, params[k]))
	}

	contextStr := fmt.Sprintf("pr=%t,prId=%d", context.IsPullRequest, context.PullRequestID)

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s?%s&ctx=%s", endpoint, paramStr.String(), contextStr)))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))[:16]

	return fmt.Sprintf("%s:%s:%s", context.ProjectKey, endpoint, hash)
}

func (c *Client) calculateCacheTTL(context APIContext) time.Duration {
	if context.PipelineCompleted {
		return 24 * time.Hour
	}

	if context.PipelineRunning {
		return 5 * time.Minute
	}

	return 1 * time.Hour
}

func (c *Client) Request(ctx context.Context, method, endpoint string, params map[string]string, apiContext APIContext) ([]byte, error) {
	if method == "GET" {
		cacheKey := c.buildCacheKey(endpoint, params, apiContext)
		if data, hit := c.cache.Get(cacheKey); hit {
			if c.config.EnableLogging {
				fmt.Printf("SonarCloud cache hit: %s\n", cacheKey)
			}
			return data.([]byte), nil
		}
	}

	fullURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseError(resp.StatusCode, body)
	}

	if method == "GET" && c.cache.enabled {
		cacheKey := c.buildCacheKey(endpoint, params, apiContext)
		ttl := c.calculateCacheTTL(apiContext)
		c.cache.Set(cacheKey, body, ttl)

		if c.config.EnableLogging {
			fmt.Printf("SonarCloud cache set: %s (TTL: %v)\n", cacheKey, ttl)
		}
	}

	return body, nil
}

func (c *Client) buildURL(endpoint string, params map[string]string) (string, error) {
	endpoint = strings.TrimPrefix(endpoint, "/")

	u, err := url.Parse(c.baseURL.String() + "/" + endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint: %w", err)
	}

	if len(params) > 0 {
		query := u.Query()
		for key, value := range params {
			query.Set(key, value)
		}
		u.RawQuery = query.Encode()
	}

	return u.String(), nil
}

func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if !c.isRetryableError(err) {
				return nil, err
			}

			if c.config.EnableLogging {
				fmt.Printf("SonarCloud request failed (attempt %d/%d): %v\n",
					attempt+1, c.config.RetryAttempts+1, err)
			}

			if attempt < c.config.RetryAttempts {
				c.waitBeforeRetry(attempt, 0)
				continue
			}
			return nil, lastErr
		}

		if c.shouldRetry(resp.StatusCode, attempt) {
			resp.Body.Close()

			if c.config.EnableLogging {
				fmt.Printf("SonarCloud API retry %d/%d after status %d\n",
					attempt+1, c.config.RetryAttempts, resp.StatusCode)
			}

			c.waitBeforeRetry(attempt, resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

func (c *Client) shouldRetry(statusCode, attempt int) bool {
	if attempt >= c.config.RetryAttempts {
		return false
	}

	if statusCode == 429 {
		return true
	}

	if statusCode >= 500 {
		return true
	}

	return false
}

func (c *Client) isRetryableError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "no such host")
}

func (c *Client) waitBeforeRetry(attempt int, statusCode int) {
	delay := c.config.BaseDelay * time.Duration(1<<attempt)

	if statusCode == 429 {
		delay = delay * 3
	}

	if delay > c.config.MaxDelay {
		delay = c.config.MaxDelay
	}

	if c.config.Jitter {
		jitterAmount := time.Duration(rand.Int63n(int64(delay / 4)))
		delay += jitterAmount
	}

	if c.config.EnableLogging {
		fmt.Printf("SonarCloud waiting %v before retry\n", delay)
	}

	time.Sleep(delay)
}

func (c *Client) parseError(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return &SonarCloudError{
			StatusCode:       statusCode,
			UserMessage:      "SonarCloud authentication failed",
			TechnicalDetails: "Your token may be expired or invalid",
			SuggestedActions: []string{
				"Generate a new token at https://sonarcloud.io/account/security/",
				"Set SONARCLOUD_TOKEN environment variable",
			},
			HelpLinks: []string{
				"https://sonarcloud.io/account/security/",
			},
		}
	case 403:
		return &SonarCloudError{
			StatusCode:       statusCode,
			UserMessage:      "Access denied to SonarCloud project",
			TechnicalDetails: "Insufficient permissions for the requested resource",
			SuggestedActions: []string{
				"Ensure your token has access to this project",
				"Contact your SonarCloud administrator",
			},
		}
	case 404:
		return &SonarCloudError{
			StatusCode:       statusCode,
			UserMessage:      "SonarCloud project not found",
			TechnicalDetails: "The project key may be incorrect or project doesn't exist",
			SuggestedActions: []string{
				"Verify project exists at https://sonarcloud.io/projects",
				"Check project key in SonarCloud project settings",
				"Set correct key: export SONARCLOUD_PROJECT_KEY=\"correct_key\"",
			},
		}
	case 429:
		return &SonarCloudError{
			StatusCode:       statusCode,
			UserMessage:      "SonarCloud API rate limit exceeded",
			TechnicalDetails: "Too many requests sent in a short time",
			SuggestedActions: []string{
				"Wait a few minutes before retrying",
				"Reduce concurrent requests",
			},
		}
	default:
		var errorResp map[string]interface{}
		if json.Unmarshal(body, &errorResp) == nil {
			if msg, ok := errorResp["message"].(string); ok {
				return &SonarCloudError{
					StatusCode:       statusCode,
					UserMessage:      fmt.Sprintf("SonarCloud API error (%d)", statusCode),
					TechnicalDetails: msg,
				}
			}
		}

		return &SonarCloudError{
			StatusCode:       statusCode,
			UserMessage:      fmt.Sprintf("SonarCloud API error (%d)", statusCode),
			TechnicalDetails: string(body),
		}
	}
}

type SonarCloudError struct {
	StatusCode       int
	UserMessage      string
	TechnicalDetails string
	SuggestedActions []string
	HelpLinks        []string
}

func (e *SonarCloudError) Error() string {
	return e.UserMessage
}

func (e *SonarCloudError) Format(verbose bool) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("❌ %s\n", e.UserMessage))

	if len(e.SuggestedActions) > 0 {
		output.WriteString("\nSuggested actions:\n")
		for _, action := range e.SuggestedActions {
			output.WriteString(fmt.Sprintf("  • %s\n", action))
		}
	}

	if verbose && e.TechnicalDetails != "" {
		output.WriteString(fmt.Sprintf("\nTechnical details: %s\n", e.TechnicalDetails))
	}

	if len(e.HelpLinks) > 0 {
		output.WriteString("\nFor more information:\n")
		for _, link := range e.HelpLinks {
			output.WriteString(fmt.Sprintf("  %s\n", link))
		}
	}

	return output.String()
}

func (c *Client) GetJSON(ctx context.Context, endpoint string, params map[string]string, context APIContext, result interface{}) error {
	body, err := c.Request(ctx, "GET", endpoint, params, context)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}

func (c *Client) EnableLogging(enabled bool) {
	c.config.EnableLogging = enabled
}

func (c *Client) ClearCache() {
	c.cache.Clear()
}
