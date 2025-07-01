package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/output"
)

// StatusCmd handles auth status command
type StatusCmd struct{}

// Run executes the auth status command
func (cmd *StatusCmd) Run(ctx context.Context) error {
	// Try to determine the current authentication method and get status
	status, err := cmd.getAuthStatus(ctx)
	if err != nil {
		return err
	}

	// Get output format from global context
	outputFormat := ctx.Value("output-format")
	if outputFormat == nil {
		outputFormat = "table"
	}

	// Create formatter for output
	formatter, err := output.NewFormatter(output.Format(outputFormat.(string)), &output.FormatterOptions{
		Writer:  os.Stdout,
		NoColor: ctx.Value("no-color") == true,
	})
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	// Format and display the status
	return formatter.Format(status)
}

// AuthStatus represents the authentication status information
type AuthStatus struct {
	Authenticated bool              `json:"authenticated" yaml:"authenticated"`
	Method        auth.AuthMethod   `json:"method,omitempty" yaml:"method,omitempty"`
	User          *auth.User        `json:"user,omitempty" yaml:"user,omitempty"`
	Host          string            `json:"host" yaml:"host"`
	TokenSource   string            `json:"token_source,omitempty" yaml:"token_source,omitempty"`
	Error         string            `json:"error,omitempty" yaml:"error,omitempty"`
	Scopes        []string          `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// getAuthStatus attempts to get the current authentication status
func (cmd *StatusCmd) getAuthStatus(ctx context.Context) (*AuthStatus, error) {
	status := &AuthStatus{
		Authenticated: false,
		Host:          "bitbucket.org",
	}

	// Try to determine authentication method from environment variables or stored credentials
	method, tokenSource := cmd.detectAuthMethod()
	if method != "" {
		status.Method = method
		status.TokenSource = tokenSource

		// Try to authenticate and get user info
		manager, err := createAuthManager(method)
		if err != nil {
			status.Error = fmt.Sprintf("Failed to create auth manager: %v", err)
			return status, nil
		}

		// Check if authentication is valid
		user, err := manager.GetAuthenticatedUser(ctx)
		if err != nil {
			status.Error = fmt.Sprintf("Authentication invalid: %v", err)
			return status, nil
		}

		// Authentication successful
		status.Authenticated = true
		status.User = user

		// Add scope information based on method
		switch method {
		case auth.AuthMethodAppPassword:
			status.Scopes = []string{"repository", "pullrequest", "pipeline"}
		case auth.AuthMethodOAuth:
			status.Scopes = []string{"repository", "pullrequest", "pipeline", "account"}
		case auth.AuthMethodAccessToken:
			status.Scopes = []string{"varies by token configuration"}
		}
	} else {
		status.Error = "No authentication credentials found"
	}

	return status, nil
}

// detectAuthMethod tries to detect the current authentication method
func (cmd *StatusCmd) detectAuthMethod() (auth.AuthMethod, string) {
	// Check for access token (highest precedence)
	if token := os.Getenv("BITBUCKET_TOKEN"); token != "" {
		return auth.AuthMethodAccessToken, "environment variable (BITBUCKET_TOKEN)"
	}

	// Check for app password credentials
	if username := os.Getenv("BITBUCKET_USERNAME"); username != "" {
		if password := os.Getenv("BITBUCKET_PASSWORD"); password != "" {
			return auth.AuthMethodAppPassword, "environment variables (BITBUCKET_USERNAME/BITBUCKET_PASSWORD)"
		}
	}

	// Try to load from stored credentials
	// First try access token from storage
	if tokenManager, err := createAuthManager(auth.AuthMethodAccessToken); err == nil {
		if _, err := tokenManager.GetAuthenticatedUser(context.Background()); err == nil {
			return auth.AuthMethodAccessToken, "stored credentials"
		}
	}

	// Try app password from storage
	if appManager, err := createAuthManager(auth.AuthMethodAppPassword); err == nil {
		if _, err := appManager.GetAuthenticatedUser(context.Background()); err == nil {
			return auth.AuthMethodAppPassword, "stored credentials"
		}
	}

	// Try OAuth from storage
	if oauthManager, err := createAuthManager(auth.AuthMethodOAuth); err == nil {
		if _, err := oauthManager.GetAuthenticatedUser(context.Background()); err == nil {
			return auth.AuthMethodOAuth, "stored credentials"
		}
	}

	return "", ""
}

// String implements fmt.Stringer for table output
func (s *AuthStatus) String() string {
	if !s.Authenticated {
		return fmt.Sprintf("❌ Not authenticated to bitbucket.org\n💡 Run 'bt auth login' to authenticate")
	}

	result := fmt.Sprintf("✅ Authenticated to %s\n", s.Host)
	result += fmt.Sprintf("👤 User: %s (%s)\n", s.User.DisplayName, s.User.Username)
	if s.User.Email != "" {
		result += fmt.Sprintf("📧 Email: %s\n", s.User.Email)
	}
	result += fmt.Sprintf("🔐 Method: %s\n", s.Method)
	result += fmt.Sprintf("📍 Source: %s\n", s.TokenSource)
	
	if len(s.Scopes) > 0 {
		result += fmt.Sprintf("🔓 Scopes: %v\n", s.Scopes)
	}

	return result
}