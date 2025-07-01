package auth

import (
	"context"
	"fmt"

	"github.com/carlosarraes/bt/pkg/auth"
)

// RefreshCmd handles auth refresh command
type RefreshCmd struct{}

// Run executes the auth refresh command
func (cmd *RefreshCmd) Run(ctx context.Context) error {
	fmt.Println("🔄 Checking authentication status...")

	// Try to determine the current authentication method
	currentMethod, err := cmd.detectCurrentAuthMethod(ctx)
	if err != nil {
		return fmt.Errorf("❌ No active authentication found: %v\nRun 'bt auth login' to authenticate", err)
	}

	fmt.Printf("🔍 Found %s authentication\n", currentMethod)

	// Check if the method supports refresh
	if currentMethod != auth.AuthMethodOAuth {
		return fmt.Errorf("❌ Token refresh is only supported for OAuth authentication\nCurrent method: %s\n💡 App passwords and access tokens don't expire and don't need refresh", currentMethod)
	}

	// Refresh OAuth tokens
	fmt.Println("🔄 Refreshing OAuth tokens...")

	manager, err := createAuthManager(auth.AuthMethodOAuth)
	if err != nil {
		return fmt.Errorf("failed to create OAuth manager: %w", err)
	}

	// Attempt to refresh the token
	if err := manager.Refresh(ctx); err != nil {
		return fmt.Errorf("❌ Failed to refresh OAuth token: %w\n💡 Try running 'bt auth login' to re-authenticate", err)
	}

	// Verify the refresh worked by getting user info
	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("❌ Token refresh appeared to succeed but authentication is still invalid: %w", err)
	}

	fmt.Println("✅ OAuth tokens refreshed successfully!")
	fmt.Printf("👤 Authenticated as: %s (%s)\n", user.DisplayName, user.Username)
	fmt.Printf("📧 Email: %s\n", user.Email)
	fmt.Println("🕒 New tokens are valid for another 2 hours")

	return nil
}

// detectCurrentAuthMethod tries to detect the current authentication method
func (cmd *RefreshCmd) detectCurrentAuthMethod(ctx context.Context) (auth.AuthMethod, error) {
	// Try each authentication method to find the active one
	methods := []auth.AuthMethod{
		auth.AuthMethodOAuth,      // Try OAuth first since that's what we're refreshing
		auth.AuthMethodAccessToken,
		auth.AuthMethodAppPassword,
	}

	for _, method := range methods {
		manager, err := createAuthManager(method)
		if err != nil {
			continue
		}

		// Try to get user info to verify authentication works
		_, err = manager.GetAuthenticatedUser(ctx)
		if err == nil {
			return method, nil
		}
	}

	return "", fmt.Errorf("no valid authentication found")
}