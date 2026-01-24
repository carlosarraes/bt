package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/config"
)

// LoginCmd handles auth login command
type LoginCmd struct {
	WithToken string `help:"Authenticate with a token (format: email:token)"`
}

// Run executes the auth login command
func (cmd *LoginCmd) Run(ctx context.Context) error {
	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check for environment variables first (highest precedence)
	if email := os.Getenv("BITBUCKET_EMAIL"); email != "" {
		if token := os.Getenv("BITBUCKET_API_TOKEN"); token != "" {
			return cmd.authenticateWithAPIToken(ctx, email, token, "environment variables")
		}
	}

	// Also check legacy environment variables for backward compatibility
	if email := os.Getenv("BITBUCKET_USERNAME"); email != "" {
		if token := os.Getenv("BITBUCKET_PASSWORD"); token != "" {
			return cmd.authenticateWithAPIToken(ctx, email, token, "environment variables")
		}
	}

	// Check for --with-token flag (expects email:token format)
	if cmd.WithToken != "" {
		parts := strings.SplitN(cmd.WithToken, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--with-token flag requires format: email:token")
		}
		return cmd.authenticateWithAPIToken(ctx, parts[0], parts[1], "command line flag")
	}

	// Interactive authentication flow
	return cmd.interactiveLogin(ctx, cfg)
}

// authenticateWithAPIToken handles API token authentication (email + token)
func (cmd *LoginCmd) authenticateWithAPIToken(ctx context.Context, email, token, source string) error {
	fmt.Printf("🔑 Authenticating with API token from %s...\n", source)

	// Set environment variables for authentication
	os.Setenv("BITBUCKET_EMAIL", email)
	os.Setenv("BITBUCKET_API_TOKEN", token)

	// Create API token authenticator
	manager, err := shared.CreateAuthManagerWithMethod(auth.AuthMethodAPIToken)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	// Authenticate
	if err := manager.Authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get user information to confirm authentication
	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	fmt.Printf("✅ Authentication successful!\n")
	fmt.Printf("👤 Logged in as: %s (%s)\n", user.DisplayName, user.Username)
	fmt.Printf("📧 Email: %s\n", user.Email)
	fmt.Printf("🔐 Method: API Token\n")
	fmt.Printf("\n💡 Tip: You can set environment variables to avoid re-entering credentials:\n")
	fmt.Printf("   export BITBUCKET_EMAIL=\"%s\"\n", email)
	fmt.Printf("   export BITBUCKET_API_TOKEN=\"your-api-token\"\n")

	return nil
}

// interactiveLogin handles interactive authentication flow
func (cmd *LoginCmd) interactiveLogin(ctx context.Context, cfg *config.Config) error {
	fmt.Println("🚀 Welcome to Bitbucket CLI Authentication")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("🔑 Authentication uses API tokens (email + token)")
	fmt.Println("📋 Create an API token at: https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()

	// Setup API token authentication
	return cmd.setupAPIToken(ctx)
}

// setupAPIToken handles API token authentication setup (email + token)
func (cmd *LoginCmd) setupAPIToken(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// Get email
	fmt.Print("📧 Atlassian account email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	// Get API token (hidden input)
	fmt.Print("🔑 API token (hidden): ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println() // New line after hidden input

	if token == "" {
		return fmt.Errorf("API token cannot be empty")
	}

	return cmd.authenticateWithAPIToken(ctx, email, token, "interactive input")
}

