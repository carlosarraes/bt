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

// AppPasswordAuth implements authentication using Bitbucket App Passwords
type AppPasswordAuth struct {
	config      *Config
	storage     CredentialStorage
	credentials *StoredCredentials
	httpClient  *http.Client
}

// NewAppPasswordAuth creates a new App Password authenticator
func NewAppPasswordAuth(config *Config, storage CredentialStorage) (Authenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}
	
	auth := &AppPasswordAuth{
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

func (a *AppPasswordAuth) Authenticate(ctx context.Context) error {
	// Check for environment variables first
	username := os.Getenv("BITBUCKET_USERNAME")
	password := os.Getenv("BITBUCKET_PASSWORD")
	
	if username == "" || password == "" {
		// Check if we have stored credentials
		if a.credentials != nil && a.credentials.Username != "" && a.credentials.Password != "" {
			username = a.credentials.Username
			password = a.credentials.Password
		} else {
			return fmt.Errorf("no app password credentials found - set BITBUCKET_USERNAME and BITBUCKET_PASSWORD environment variables or use 'bt auth login'")
		}
	}
	
	// Validate credentials by making a test API call
	if err := a.validateCredentials(ctx, username, password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Store credentials if they came from environment variables
	if os.Getenv("BITBUCKET_USERNAME") != "" && os.Getenv("BITBUCKET_PASSWORD") != "" {
		a.credentials = &StoredCredentials{
			Method:    AuthMethodAppPassword,
			Username:  username,
			Password:  password,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		if err := a.storage.Store("auth", a.credentials); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}
	
	return nil
}

func (a *AppPasswordAuth) SetHTTPHeaders(req *http.Request) error {
	if a.credentials == nil {
		return fmt.Errorf("not authenticated")
	}
	
	username := a.credentials.Username
	password := a.credentials.Password
	
	// Check environment variables for runtime override
	if envUsername := os.Getenv("BITBUCKET_USERNAME"); envUsername != "" {
		username = envUsername
	}
	if envPassword := os.Getenv("BITBUCKET_PASSWORD"); envPassword != "" {
		password = envPassword
	}
	
	if username == "" || password == "" {
		return fmt.Errorf("missing username or password")
	}
	
	// Create Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	return nil
}

func (a *AppPasswordAuth) IsValid(ctx context.Context) (bool, error) {
	if a.credentials == nil {
		return false, nil
	}
	
	username := a.credentials.Username
	password := a.credentials.Password
	
	// Check environment variables for runtime override
	if envUsername := os.Getenv("BITBUCKET_USERNAME"); envUsername != "" {
		username = envUsername
	}
	if envPassword := os.Getenv("BITBUCKET_PASSWORD"); envPassword != "" {
		password = envPassword
	}
	
	if username == "" || password == "" {
		return false, nil
	}
	
	// Validate by making a test API call
	err := a.validateCredentials(ctx, username, password)
	return err == nil, nil
}

func (a *AppPasswordAuth) Refresh(ctx context.Context) error {
	// App passwords don't need refreshing
	return nil
}

func (a *AppPasswordAuth) GetUser(ctx context.Context) (*User, error) {
	if a.credentials == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	
	username := a.credentials.Username
	password := a.credentials.Password
	
	// Check environment variables for runtime override
	if envUsername := os.Getenv("BITBUCKET_USERNAME"); envUsername != "" {
		username = envUsername
	}
	if envPassword := os.Getenv("BITBUCKET_PASSWORD"); envPassword != "" {
		password = envPassword
	}
	
	return a.fetchUser(ctx, username, password)
}

func (a *AppPasswordAuth) Clear() error {
	if err := a.storage.Delete("auth"); err != nil {
		return fmt.Errorf("failed to clear stored credentials: %w", err)
	}
	
	a.credentials = nil
	return nil
}

// validateCredentials makes a test API call to validate the credentials
func (a *AppPasswordAuth) validateCredentials(ctx context.Context, username, password string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid username or app password")
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
	
	return nil
}

// fetchUser retrieves user information from the Bitbucket API
func (a *AppPasswordAuth) fetchUser(ctx context.Context, username, password string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
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

// InteractiveLogin prompts the user for app password credentials
func (a *AppPasswordAuth) InteractiveLogin() error {
	fmt.Print("Username: ")
	var username string
	if _, err := fmt.Scanln(&username); err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	
	fmt.Print("App Password: ")
	var password string
	if _, err := fmt.Scanln(&password); err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	
	// Validate credentials
	ctx := context.Background()
	if err := a.validateCredentials(ctx, username, password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Store credentials
	a.credentials = &StoredCredentials{
		Method:    AuthMethodAppPassword,
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := a.storage.Store("auth", a.credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}
	
	return nil
}