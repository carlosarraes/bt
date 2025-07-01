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
	"github.com/carlosarraes/bt/pkg/config"
)

// LoginCmd handles auth login command
type LoginCmd struct {
	WithToken string `help:"Authenticate with a token instead of interactive flow"`
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
	if token := os.Getenv("BITBUCKET_TOKEN"); token != "" {
		return cmd.authenticateWithToken(ctx, token, "environment variable")
	}

	// Check for --with-token flag
	if cmd.WithToken != "" {
		return cmd.authenticateWithToken(ctx, cmd.WithToken, "command line flag")
	}

	// Interactive authentication flow
	return cmd.interactiveLogin(ctx, cfg)
}

// authenticateWithToken handles token-based authentication
func (cmd *LoginCmd) authenticateWithToken(ctx context.Context, token, source string) error {
	fmt.Printf("🔑 Authenticating with token from %s...\n", source)

	// Create access token authenticator
	manager, err := createAuthManager(auth.AuthMethodAccessToken)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	// Set environment variable for the auth system to use
	os.Setenv("BITBUCKET_TOKEN", token)

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
	fmt.Printf("🔐 Method: Access Token\n")

	return nil
}

// interactiveLogin handles interactive authentication flow
func (cmd *LoginCmd) interactiveLogin(ctx context.Context, cfg *config.Config) error {
	fmt.Println("🚀 Welcome to Bitbucket CLI Authentication")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Ask for authentication method
	method, err := cmd.selectAuthMethod()
	if err != nil {
		return err
	}

	fmt.Printf("\n🔑 Setting up %s authentication...\n", method)

	switch method {
	case auth.AuthMethodAppPassword:
		return cmd.setupAppPassword(ctx)
	case auth.AuthMethodOAuth:
		return cmd.setupOAuth(ctx)
	case auth.AuthMethodAccessToken:
		return cmd.setupAccessToken(ctx)
	default:
		return fmt.Errorf("unsupported authentication method: %s", method)
	}
}

// selectAuthMethod prompts user to select authentication method
func (cmd *LoginCmd) selectAuthMethod() (auth.AuthMethod, error) {
	fmt.Println("📝 How would you like to authenticate?")
	fmt.Println()
	fmt.Println("  1) App Password (username + app password)")
	fmt.Println("  2) OAuth 2.0 (browser-based)")
	fmt.Println("  3) Access Token (repository/workspace scoped)")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("🤔 Select authentication method [1-3]: ")
		
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		
		choice := strings.TrimSpace(input)
		switch choice {
		case "1":
			return auth.AuthMethodAppPassword, nil
		case "2":
			return auth.AuthMethodOAuth, nil
		case "3":
			return auth.AuthMethodAccessToken, nil
		default:
			fmt.Printf("❌ Invalid choice '%s'. Please enter 1, 2, or 3.\n", choice)
		}
	}
}

// setupAppPassword handles app password authentication setup
func (cmd *LoginCmd) setupAppPassword(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// Get username
	fmt.Print("👤 Bitbucket username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Get app password (hidden input)
	fmt.Print("🔒 App password (hidden): ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password := strings.TrimSpace(string(passwordBytes))
	fmt.Println() // New line after hidden input

	if password == "" {
		return fmt.Errorf("app password cannot be empty")
	}

	// Set environment variables for authentication
	os.Setenv("BITBUCKET_USERNAME", username)
	os.Setenv("BITBUCKET_PASSWORD", password)

	// Create app password authenticator
	manager, err := createAuthManager(auth.AuthMethodAppPassword)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	fmt.Println("🔄 Authenticating with Bitbucket...")

	// Authenticate
	if err := manager.Authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get user information
	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	fmt.Printf("✅ Authentication successful!\n")
	fmt.Printf("👤 Logged in as: %s (%s)\n", user.DisplayName, user.Username)
	fmt.Printf("📧 Email: %s\n", user.Email)
	fmt.Printf("🔐 Method: App Password\n")

	return nil
}

// setupOAuth handles OAuth 2.0 authentication setup
func (cmd *LoginCmd) setupOAuth(ctx context.Context) error {
	fmt.Println("🌐 Starting OAuth 2.0 authentication flow...")
	fmt.Println("📖 This will open your browser to authenticate with Bitbucket.")
	fmt.Println()

	// Create OAuth authenticator
	manager, err := createAuthManager(auth.AuthMethodOAuth)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	fmt.Println("🔄 Starting OAuth flow...")

	// Authenticate (this will open browser and start local server)
	if err := manager.Authenticate(ctx); err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Get user information
	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	fmt.Printf("✅ OAuth authentication successful!\n")
	fmt.Printf("👤 Logged in as: %s (%s)\n", user.DisplayName, user.Username)
	fmt.Printf("📧 Email: %s\n", user.Email)
	fmt.Printf("🔐 Method: OAuth 2.0\n")

	return nil
}

// setupAccessToken handles access token authentication setup
func (cmd *LoginCmd) setupAccessToken(ctx context.Context) error {
	fmt.Println("🎟️  Access tokens can be scoped to specific repositories, workspaces, or projects.")
	fmt.Println("📋 Create one at: https://bitbucket.org/account/settings/app-passwords/")
	fmt.Println()

	// Get access token (hidden input)
	fmt.Print("🔑 Access token (hidden): ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println() // New line after hidden input

	if token == "" {
		return fmt.Errorf("access token cannot be empty")
	}

	return cmd.authenticateWithToken(ctx, token, "interactive input")
}