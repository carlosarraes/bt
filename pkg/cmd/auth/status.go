package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
)

// StatusCmd handles auth status command
type StatusCmd struct{}

// Run executes the auth status command
func (cmd *StatusCmd) Run(ctx context.Context) error {
	status, err := cmd.getAuthStatus(ctx)
	if err != nil {
		return err
	}

	outputFormat := ctx.Value("output-format")
	if outputFormat == nil {
		outputFormat = "table"
	}

	formatter, err := output.NewFormatter(output.Format(outputFormat.(string)), &output.FormatterOptions{
		Writer:  os.Stdout,
		NoColor: ctx.Value("no-color") == true,
	})
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	return formatter.Format(status)
}

// AuthStatus represents the authentication status information
type AuthStatus struct {
	Authenticated bool            `json:"authenticated" yaml:"authenticated"`
	Method        auth.AuthMethod `json:"method,omitempty" yaml:"method,omitempty"`
	User          *auth.User      `json:"user,omitempty" yaml:"user,omitempty"`
	Host          string          `json:"host" yaml:"host"`
	TokenSource   string          `json:"token_source,omitempty" yaml:"token_source,omitempty"`
	Error         string          `json:"error,omitempty" yaml:"error,omitempty"`
	Scopes        []string        `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

func (cmd *StatusCmd) getAuthStatus(ctx context.Context) (*AuthStatus, error) {
	status := &AuthStatus{
		Authenticated: false,
		Host:          "bitbucket.org",
	}

	tokenSource := cmd.detectAuthMethod()
	if tokenSource == "" {
		status.Error = "No API token credentials found"
		return status, nil
	}

	status.Method = auth.AuthMethodAPIToken
	status.TokenSource = tokenSource

	manager, err := shared.CreateAuthManagerWithMethod(auth.AuthMethodAPIToken)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create auth manager: %v", err)
		return status, nil
	}

	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		status.Error = fmt.Sprintf("Authentication invalid: %v", err)
		return status, nil
	}

	status.Authenticated = true
	status.User = user
	status.Scopes = []string{"repository", "pullrequest", "pipeline", "account"}

	return status, nil
}

func (cmd *StatusCmd) detectAuthMethod() string {
	if email := os.Getenv("BITBUCKET_EMAIL"); email != "" {
		if token := os.Getenv("BITBUCKET_API_TOKEN"); token != "" {
			return "environment variables (BITBUCKET_EMAIL/BITBUCKET_API_TOKEN)"
		}
	}

	if username := os.Getenv("BITBUCKET_USERNAME"); username != "" {
		if password := os.Getenv("BITBUCKET_PASSWORD"); password != "" {
			return "environment variables (BITBUCKET_USERNAME/BITBUCKET_PASSWORD)"
		}
	}

	return ""
}

// String implements fmt.Stringer for table output
func (s *AuthStatus) String() string {
	if !s.Authenticated {
		return fmt.Sprintf("❌ Not authenticated to bitbucket.org\n💡 Run 'bt auth login' to authenticate with your API token")
	}

	result := fmt.Sprintf("✅ Authenticated to %s\n", s.Host)
	result += fmt.Sprintf("👤 User: %s (%s)\n", s.User.DisplayName, s.User.Username)
	if s.User.Email != "" {
		result += fmt.Sprintf("📧 Email: %s\n", s.User.Email)
	}
	result += fmt.Sprintf("🔐 Method: API Token\n")
	result += fmt.Sprintf("📍 Source: %s\n", s.TokenSource)

	if len(s.Scopes) > 0 {
		result += fmt.Sprintf("🔓 Scopes: %v\n", s.Scopes)
	}

	return result
}
