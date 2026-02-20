package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

// LogoutCmd handles auth logout command
type LogoutCmd struct {
	Force bool `short:"f" help:"Force logout without confirmation"`
}

// Run executes the auth logout command
func (cmd *LogoutCmd) Run(ctx context.Context) error {
	currentUser, err := cmd.getCurrentAuthStatus(ctx)
	if err != nil {
		fmt.Println("❌ No active authentication found")
		return nil
	}

	fmt.Printf("🔍 Found authentication for: %s (%s)\n", currentUser.DisplayName, currentUser.Username)
	fmt.Printf("🔐 Method: API Token\n\n")

	if !cmd.Force {
		if !cmd.confirmLogout() {
			fmt.Println("❌ Logout cancelled")
			return nil
		}
	}

	fmt.Println("🔄 Clearing authentication credentials...")

	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_USERNAME")
	os.Unsetenv("BITBUCKET_PASSWORD")

	profile, err := detectShellProfile()
	if err != nil {
		fmt.Printf("⚠️  %v\n", err)
		fmt.Println("💡 Remove BITBUCKET_EMAIL and BITBUCKET_API_TOKEN from your shell profile manually")
	} else {
		keys := []string{
			"BITBUCKET_EMAIL", "BITBUCKET_API_TOKEN",
			"BITBUCKET_USERNAME", "BITBUCKET_PASSWORD",
		}
		if err := removeEnvsFromProfile(profile, keys); err != nil {
			fmt.Printf("⚠️  Could not remove credentials from %s: %v\n", profile, err)
			fmt.Println("💡 Please remove BITBUCKET_EMAIL and BITBUCKET_API_TOKEN manually")
		} else {
			fmt.Printf("💾 Credentials removed from %s\n", profile)
		}
		fmt.Printf("💡 Run 'source %s' or open a new terminal for changes to take effect\n", profile)
	}

	fmt.Println("✅ Successfully logged out from Bitbucket")
	fmt.Println("💡 Run 'bt auth login' to authenticate again")

	return nil
}

func (cmd *LogoutCmd) getCurrentAuthStatus(ctx context.Context) (*auth.User, error) {
	manager, err := shared.CreateAuthManagerWithMethod(auth.AuthMethodAPIToken)
	if err != nil {
		return nil, err
	}

	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("no active authentication found")
	}

	return user, nil
}

func (cmd *LogoutCmd) confirmLogout() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("🤔 Are you sure you want to log out? This will remove stored credentials [y/N]: ")

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
