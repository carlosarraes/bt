package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/carlosarraes/bt/pkg/auth"
)

// LogoutCmd handles auth logout command
type LogoutCmd struct {
	Force bool `short:"f" help:"Force logout without confirmation"`
}

// Run executes the auth logout command
func (cmd *LogoutCmd) Run(ctx context.Context) error {
	// First, check if user is currently authenticated
	currentUser, currentMethod, err := cmd.getCurrentAuthStatus(ctx)
	if err != nil {
		fmt.Println("❌ No active authentication found")
		return nil
	}

	fmt.Printf("🔍 Found authentication for: %s (%s)\n", currentUser.DisplayName, currentUser.Username)
	fmt.Printf("🔐 Method: %s\n", currentMethod)
	fmt.Println()

	// Confirm logout unless --force flag is used
	if !cmd.Force {
		if !cmd.confirmLogout() {
			fmt.Println("❌ Logout cancelled")
			return nil
		}
	}

	// Perform logout for all authentication methods
	fmt.Println("🔄 Clearing authentication credentials...")

	var errors []string

	// Clear all possible authentication methods
	methods := []auth.AuthMethod{
		auth.AuthMethodAppPassword,
		auth.AuthMethodOAuth,
		auth.AuthMethodAccessToken,
	}

	for _, method := range methods {
		if err := cmd.clearAuthMethod(method); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", method, err))
		}
	}

	// Clear environment variables (these won't persist, but clear for current session)
	os.Unsetenv("BITBUCKET_TOKEN")
	os.Unsetenv("BITBUCKET_USERNAME")
	os.Unsetenv("BITBUCKET_PASSWORD")

	if len(errors) > 0 {
		fmt.Println("⚠️  Some errors occurred during logout:")
		for _, err := range errors {
			fmt.Printf("   • %s\n", err)
		}
		fmt.Println()
	}

	fmt.Println("✅ Successfully logged out from Bitbucket")
	fmt.Println("💡 Run 'bt auth login' to authenticate again")

	return nil
}

// getCurrentAuthStatus checks current authentication status
func (cmd *LogoutCmd) getCurrentAuthStatus(ctx context.Context) (*auth.User, auth.AuthMethod, error) {
	// Try each authentication method to find the current one
	methods := []auth.AuthMethod{
		auth.AuthMethodAccessToken,
		auth.AuthMethodAppPassword,
		auth.AuthMethodOAuth,
	}

	for _, method := range methods {
		manager, err := createAuthManager(method)
		if err != nil {
			continue
		}

		user, err := manager.GetAuthenticatedUser(ctx)
		if err == nil {
			return user, method, nil
		}
	}

	return nil, "", fmt.Errorf("no active authentication found")
}

// confirmLogout prompts the user to confirm logout
func (cmd *LogoutCmd) confirmLogout() bool {
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print("🤔 Are you sure you want to log out? This will remove all stored credentials [y/N]: ")
		
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Failed to read input: %v\n", err)
			return false
		}
		
		response := strings.ToLower(strings.TrimSpace(input))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Printf("❌ Invalid response '%s'. Please enter 'y' for yes or 'n' for no.\n", response)
		}
	}
}

// clearAuthMethod clears stored credentials for a specific authentication method
func (cmd *LogoutCmd) clearAuthMethod(method auth.AuthMethod) error {
	manager, err := createAuthManager(method)
	if err != nil {
		return err
	}

	return manager.Logout()
}