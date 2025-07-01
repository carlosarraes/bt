package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// AccessTokenAuth implements authentication using Bitbucket Access Tokens
type AccessTokenAuth struct {
	config      *Config
	storage     CredentialStorage
	credentials *StoredCredentials
	httpClient  *http.Client
}

// NewAccessTokenAuth creates a new Access Token authenticator
func NewAccessTokenAuth(config *Config, storage CredentialStorage) (Authenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}
	
	auth := &AccessTokenAuth{
		config:  config,
		storage: storage,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
	
	// Try to load existing credentials
	var creds StoredCredentials
	if storage.Exists("auth") {
		if err := storage.Retrieve("auth", &creds); err == nil && creds.Method == AuthMethodAccessToken {
			auth.credentials = &creds
		}
	}
	
	return auth, nil
}

func (a *AccessTokenAuth) Authenticate(ctx context.Context) error {
	// Check for environment variable first
	token := os.Getenv("BITBUCKET_TOKEN")
	
	if token == "" {
		// Check if we have stored credentials
		if a.credentials != nil && a.credentials.AccessToken != "" {
			token = a.credentials.AccessToken
		} else {
			return fmt.Errorf("no access token found - set BITBUCKET_TOKEN environment variable or use 'bt auth login --with-token'")
		}
	}
	
	// Validate token by making a test API call
	if err := a.validateToken(ctx, token); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Store token if it came from environment variable
	if os.Getenv("BITBUCKET_TOKEN") != "" {
		a.credentials = &StoredCredentials{
			Method:      AuthMethodAccessToken,
			AccessToken: token,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		
		if err := a.storage.Store("auth", a.credentials); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}
	}
	
	return nil
}

func (a *AccessTokenAuth) SetHTTPHeaders(req *http.Request) error {
	token := a.getToken()
	if token == "" {
		return fmt.Errorf("no access token available")
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	return nil
}

func (a *AccessTokenAuth) IsValid(ctx context.Context) (bool, error) {
	token := a.getToken()
	if token == "" {
		return false, nil
	}
	
	// Validate by making a test API call
	err := a.validateToken(ctx, token)
	return err == nil, nil
}

func (a *AccessTokenAuth) Refresh(ctx context.Context) error {
	// Access tokens don't support automatic refresh
	// User needs to provide a new token
	return fmt.Errorf("access tokens cannot be automatically refreshed - please provide a new token")
}

func (a *AccessTokenAuth) GetUser(ctx context.Context) (*User, error) {
	token := a.getToken()
	if token == "" {
		return nil, fmt.Errorf("no access token available")
	}
	
	return a.fetchUser(ctx, token)
}

func (a *AccessTokenAuth) Clear() error {
	if err := a.storage.Delete("auth"); err != nil {
		return fmt.Errorf("failed to clear stored token: %w", err)
	}
	
	a.credentials = nil
	return nil
}

// getToken returns the access token, preferring environment variable
func (a *AccessTokenAuth) getToken() string {
	// Environment variable takes precedence
	if token := os.Getenv("BITBUCKET_TOKEN"); token != "" {
		return token
	}
	
	// Fall back to stored credentials
	if a.credentials != nil {
		return a.credentials.AccessToken
	}
	
	return ""
}

// validateToken makes a test API call to validate the token
func (a *AccessTokenAuth) validateToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()
	
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid or expired access token")
	case http.StatusForbidden:
		return fmt.Errorf("access token does not have required permissions")
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
}

// fetchUser retrieves user information from the Bitbucket API
func (a *AccessTokenAuth) fetchUser(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
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

// InteractiveLogin prompts the user for an access token
func (a *AccessTokenAuth) InteractiveLogin() error {
	fmt.Println("Please provide your Bitbucket Access Token.")
	fmt.Println("You can create one at: https://bitbucket.org/account/settings/app-passwords/")
	fmt.Print("Access Token: ")
	
	var token string
	if _, err := fmt.Scanln(&token); err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	
	// Validate token
	ctx := context.Background()
	if err := a.validateToken(ctx, token); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Store token
	a.credentials = &StoredCredentials{
		Method:      AuthMethodAccessToken,
		AccessToken: token,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := a.storage.Store("auth", a.credentials); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}
	
	return nil
}