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
	config      *Config
	storage     CredentialStorage
	credentials *StoredCredentials
	httpClient  *http.Client
}

// NewAPITokenAuth creates a new API Token authenticator
func NewAPITokenAuth(config *Config, storage CredentialStorage) (Authenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}

	auth := &APITokenAuth{
		config:  config,
		storage: storage,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}

	// Try to load existing credentials
	var creds StoredCredentials
	if storage.Exists("auth") {
		if err := storage.Retrieve("auth", &creds); err == nil {
			auth.credentials = &creds
		}
	}

	return auth, nil
}

func (a *APITokenAuth) Authenticate(ctx context.Context) error {
	// Check for environment variables first
	email := os.Getenv("BITBUCKET_EMAIL")
	token := os.Getenv("BITBUCKET_API_TOKEN")

	// Also check legacy environment variables for backward compatibility
	if email == "" {
		email = os.Getenv("BITBUCKET_USERNAME")
	}
	if token == "" {
		token = os.Getenv("BITBUCKET_PASSWORD")
	}

	if email == "" || token == "" {
		// Check if we have stored credentials
		if a.credentials != nil && a.credentials.Username != "" && a.credentials.Password != "" {
			email = a.credentials.Username
			token = a.credentials.Password
		} else {
			return fmt.Errorf("no API token credentials found - set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN environment variables or use 'bt auth login'")
		}
	}

	// Validate credentials by making a test API call
	if err := a.validateCredentials(ctx, email, token); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store credentials after successful authentication
	a.credentials = &StoredCredentials{
		Method:    AuthMethodAPIToken,
		Username:  email,
		Password:  token,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := a.storage.Store("auth", a.credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

func (a *APITokenAuth) SetHTTPHeaders(req *http.Request) error {
	if a.credentials == nil {
		return fmt.Errorf("not authenticated")
	}

	email := a.credentials.Username
	token := a.credentials.Password

	// Check environment variables for runtime override
	if envEmail := os.Getenv("BITBUCKET_EMAIL"); envEmail != "" {
		email = envEmail
	}
	if envToken := os.Getenv("BITBUCKET_API_TOKEN"); envToken != "" {
		token = envToken
	}

	// Also check legacy environment variables for backward compatibility
	if email == "" {
		if envEmail := os.Getenv("BITBUCKET_USERNAME"); envEmail != "" {
			email = envEmail
		}
	}
	if token == "" {
		if envToken := os.Getenv("BITBUCKET_PASSWORD"); envToken != "" {
			token = envToken
		}
	}

	if email == "" || token == "" {
		return fmt.Errorf("missing email or API token")
	}

	// Create Basic Auth header (email:token)
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")

	return nil
}

func (a *APITokenAuth) IsValid(ctx context.Context) (bool, error) {
	if a.credentials == nil {
		return false, nil
	}

	email := a.credentials.Username
	token := a.credentials.Password

	// Check environment variables for runtime override
	if envEmail := os.Getenv("BITBUCKET_EMAIL"); envEmail != "" {
		email = envEmail
	}
	if envToken := os.Getenv("BITBUCKET_API_TOKEN"); envToken != "" {
		token = envToken
	}

	// Also check legacy environment variables for backward compatibility
	if email == "" {
		if envEmail := os.Getenv("BITBUCKET_USERNAME"); envEmail != "" {
			email = envEmail
		}
	}
	if token == "" {
		if envToken := os.Getenv("BITBUCKET_PASSWORD"); envToken != "" {
			token = envToken
		}
	}

	if email == "" || token == "" {
		return false, nil
	}

	// Validate by making a test API call
	err := a.validateCredentials(ctx, email, token)
	return err == nil, nil
}

func (a *APITokenAuth) Refresh(ctx context.Context) error {
	// API tokens don't need refreshing
	return nil
}

func (a *APITokenAuth) GetUser(ctx context.Context) (*User, error) {
	if a.credentials == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	email := a.credentials.Username
	token := a.credentials.Password

	// Check environment variables for runtime override
	if envEmail := os.Getenv("BITBUCKET_EMAIL"); envEmail != "" {
		email = envEmail
	}
	if envToken := os.Getenv("BITBUCKET_API_TOKEN"); envToken != "" {
		token = envToken
	}

	// Also check legacy environment variables for backward compatibility
	if email == "" {
		if envEmail := os.Getenv("BITBUCKET_USERNAME"); envEmail != "" {
			email = envEmail
		}
	}
	if token == "" {
		if envToken := os.Getenv("BITBUCKET_PASSWORD"); envToken != "" {
			token = envToken
		}
	}

	return a.fetchUser(ctx, email, token)
}

func (a *APITokenAuth) Clear() error {
	if err := a.storage.Delete("auth"); err != nil {
		return fmt.Errorf("failed to clear stored credentials: %w", err)
	}

	a.credentials = nil
	return nil
}

// validateCredentials makes a test API call to validate the credentials
func (a *APITokenAuth) validateCredentials(ctx context.Context, email, token string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Basic Auth header (email:token)
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

	// Set Basic Auth header (email:token)
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
