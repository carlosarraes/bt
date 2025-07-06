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
	currentUser, err := cmd.getCurrentAuthStatus(ctx)
	if err != nil {
		fmt.Println("❌ No active authentication found")
		return nil
	}

	fmt.Printf("🔍 Found authentication for: %s (%s)\n", currentUser.DisplayName, currentUser.Username)
	fmt.Printf("🔐 Method: API Token\n")
	fmt.Println()

	// Confirm logout unless --force flag is used
	if !cmd.Force {
		if !cmd.confirmLogout() {
			fmt.Println("❌ Logout cancelled")
			return nil
		}
	}

	// Perform logout
	fmt.Println("🔄 Clearing authentication credentials...")

	// Clear stored credentials
	if err := cmd.clearAuthMethod(); err != nil {
		fmt.Printf("⚠️  Error clearing stored credentials: %v\n", err)
	}

	// Clear environment variables (these won't persist, but clear for current session)
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_USERNAME") // legacy
	os.Unsetenv("BITBUCKET_PASSWORD") // legacy

	fmt.Println("✅ Successfully logged out from Bitbucket")
	fmt.Println("💡 Run 'bt auth login' to authenticate again")

	return nil
}

// getCurrentAuthStatus checks current authentication status
func (cmd *LogoutCmd) getCurrentAuthStatus(ctx context.Context) (*auth.User, error) {
	manager, err := createAuthManager(auth.AuthMethodAPIToken)
	if err != nil {
		return nil, err
	}

	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("no active authentication found")
	}

	return user, nil
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

// clearAuthMethod clears stored credentials
func (cmd *LogoutCmd) clearAuthMethod() error {
	manager, err := createAuthManager(auth.AuthMethodAPIToken)
	if err != nil {
		return err
	}

	return manager.Logout()
}