package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// APITokenAuth implements authentication using Bitbucket API Tokens
type APITokenAuth struct {
	config     *Config
	httpClient *http.Client
}

// NewAPITokenAuth creates a new API Token authenticator
func NewAPITokenAuth(config *Config) (Authenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	return &APITokenAuth{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

func GetCredentials() (email, token string) {
	email = os.Getenv("BITBUCKET_EMAIL")
	token = os.Getenv("BITBUCKET_API_TOKEN")

	if email == "" {
		email = os.Getenv("BITBUCKET_USERNAME")
	}
	if token == "" {
		token = os.Getenv("BITBUCKET_PASSWORD")
	}

	return email, token
}

func (a *APITokenAuth) Authenticate(ctx context.Context) error {
	email, token := GetCredentials()

	if email == "" || token == "" {
		return fmt.Errorf("no API token credentials found - set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN environment variables or use 'bt auth login'")
	}

	if err := a.validateCredentials(ctx, email, token); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

func (a *APITokenAuth) SetHTTPHeaders(req *http.Request) error {
	email, token := GetCredentials()

	if email == "" || token == "" {
		return fmt.Errorf("missing email or API token - set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN environment variables or use 'bt auth login'")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")

	return nil
}

func (a *APITokenAuth) IsValid(ctx context.Context) (bool, error) {
	email, token := GetCredentials()

	if email == "" || token == "" {
		return false, nil
	}

	err := a.validateCredentials(ctx, email, token)
	return err == nil, nil
}

func (a *APITokenAuth) Refresh(ctx context.Context) error {
	// API tokens don't need refreshing
	return nil
}

func (a *APITokenAuth) GetUser(ctx context.Context) (*User, error) {
	email, token := GetCredentials()

	if email == "" || token == "" {
		return nil, fmt.Errorf("not authenticated - set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN environment variables or use 'bt auth login'")
	}

	return a.fetchUser(ctx, email, token)
}

func (a *APITokenAuth) Clear() error {
	return nil
}

// validateCredentials makes a test API call to validate the credentials
func (a *APITokenAuth) validateCredentials(ctx context.Context, email, token string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid email or API token")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// fetchUser retrieves user information from the Bitbucket API
func (a *APITokenAuth) fetchUser(ctx context.Context, email, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		AccountID   string `json:"account_id"`
		UUID        string `json:"uuid"`
		Email       string `json:"email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	return &User{
		Username:    apiResponse.Username,
		DisplayName: apiResponse.DisplayName,
		AccountID:   apiResponse.AccountID,
		UUID:        strings.Trim(apiResponse.UUID, "{}"),
		Email:       apiResponse.Email,
	}, nil
}
