package auth

import (
	"context"
	"fmt"
	"net/http"
)

// AuthMethod represents the type of authentication being used
type AuthMethod string

const (
	AuthMethodAPIToken AuthMethod = "api_token"
)

// User represents the authenticated Bitbucket user
type User struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
	UUID        string `json:"uuid"`
	Email       string `json:"email,omitempty"`
}

// AuthManager is the main interface for handling all authentication methods
type AuthManager interface {
	// Authenticate performs authentication using the configured method
	Authenticate(ctx context.Context) error

	// GetAuthenticatedUser returns the current authenticated user information
	GetAuthenticatedUser(ctx context.Context) (*User, error)

	// SetHTTPHeaders adds authentication headers to HTTP requests
	SetHTTPHeaders(req *http.Request) error

	// IsAuthenticated checks if the user is currently authenticated
	IsAuthenticated(ctx context.Context) (bool, error)

	// Refresh refreshes the authentication token if applicable
	Refresh(ctx context.Context) error

	// Logout clears all stored authentication data
	Logout() error

	// GetMethod returns the authentication method being used
	GetMethod() AuthMethod
}

// Authenticator is the interface that specific auth implementations must satisfy
type Authenticator interface {
	// Authenticate performs the authentication flow
	Authenticate(ctx context.Context) error

	// SetHTTPHeaders adds the appropriate auth headers to requests
	SetHTTPHeaders(req *http.Request) error

	// IsValid checks if the current authentication is still valid
	IsValid(ctx context.Context) (bool, error)

	// Refresh refreshes the authentication if supported
	Refresh(ctx context.Context) error

	// GetUser returns the authenticated user info
	GetUser(ctx context.Context) (*User, error)

	// Clear removes all stored authentication data
	Clear() error
}

// Config holds authentication configuration
type Config struct {
	Method  AuthMethod `yaml:"method"`
	BaseURL string     `yaml:"base_url"`
	Timeout int        `yaml:"timeout_seconds"`
}

// DefaultConfig returns the default authentication configuration
func DefaultConfig() *Config {
	return &Config{
		Method:  AuthMethodAPIToken,
		BaseURL: "https://api.bitbucket.org/2.0",
		Timeout: 30,
	}
}

// NewAuthManager creates a new AuthManager based on the provided configuration
func NewAuthManager(config *Config, storage CredentialStorage) (AuthManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	manager := &authManager{
		config:  config,
		storage: storage,
	}

	// Create the API token authenticator
	auth, err := NewAPITokenAuth(config, storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create API token authenticator: %w", err)
	}

	manager.authenticator = auth
	return manager, nil
}

// authManager is the concrete implementation of AuthManager
type authManager struct {
	config        *Config
	storage       CredentialStorage
	authenticator Authenticator
}

func (m *authManager) Authenticate(ctx context.Context) error {
	return m.authenticator.Authenticate(ctx)
}

func (m *authManager) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	return m.authenticator.GetUser(ctx)
}

func (m *authManager) SetHTTPHeaders(req *http.Request) error {
	return m.authenticator.SetHTTPHeaders(req)
}

func (m *authManager) IsAuthenticated(ctx context.Context) (bool, error) {
	return m.authenticator.IsValid(ctx)
}

func (m *authManager) Refresh(ctx context.Context) error {
	return m.authenticator.Refresh(ctx)
}

func (m *authManager) Logout() error {
	return m.authenticator.Clear()
}

func (m *authManager) GetMethod() AuthMethod {
	return m.config.Method
}
