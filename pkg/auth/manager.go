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
	Authenticate(ctx context.Context) error
	GetAuthenticatedUser(ctx context.Context) (*User, error)
	SetHTTPHeaders(req *http.Request) error
	IsAuthenticated(ctx context.Context) (bool, error)
	Refresh(ctx context.Context) error
	Logout() error
	GetMethod() AuthMethod
}

// Authenticator is the interface that specific auth implementations must satisfy
type Authenticator interface {
	Authenticate(ctx context.Context) error
	SetHTTPHeaders(req *http.Request) error
	IsValid(ctx context.Context) (bool, error)
	Refresh(ctx context.Context) error
	GetUser(ctx context.Context) (*User, error)
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
func NewAuthManager(config *Config) (AuthManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	auth, err := NewAPITokenAuth(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API token authenticator: %w", err)
	}

	return &authManager{
		config:        config,
		authenticator: auth,
	}, nil
}

// authManager is the concrete implementation of AuthManager
type authManager struct {
	config        *Config
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
