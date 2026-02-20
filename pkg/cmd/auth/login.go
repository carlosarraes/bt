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
)

// LoginCmd handles auth login command
type LoginCmd struct {
	WithToken string `help:"Authenticate with a token (format: email:token)"`
}

// Run executes the auth login command
func (cmd *LoginCmd) Run(ctx context.Context) error {
	if email := os.Getenv("BITBUCKET_EMAIL"); email != "" {
		if token := os.Getenv("BITBUCKET_API_TOKEN"); token != "" {
			return cmd.authenticateAndSave(ctx, email, token, "environment variables")
		}
	}

	if email := os.Getenv("BITBUCKET_USERNAME"); email != "" {
		if token := os.Getenv("BITBUCKET_PASSWORD"); token != "" {
			return cmd.authenticateAndSave(ctx, email, token, "environment variables")
		}
	}

	if cmd.WithToken != "" {
		parts := strings.SplitN(cmd.WithToken, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--with-token flag requires format: email:token")
		}
		return cmd.authenticateAndSave(ctx, parts[0], parts[1], "command line flag")
	}

	// Interactive authentication flow
	return cmd.interactiveLogin(ctx)
}

func (cmd *LoginCmd) authenticateAndSave(ctx context.Context, email, token, source string) error {
	fmt.Printf("🔑 Authenticating with API token from %s...\n", source)

	os.Setenv("BITBUCKET_EMAIL", email)
	os.Setenv("BITBUCKET_API_TOKEN", token)

	manager, err := shared.CreateAuthManagerWithMethod(auth.AuthMethodAPIToken)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	if err := manager.Authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	fmt.Printf("✅ Authentication successful!\n")
	fmt.Printf("👤 Logged in as: %s (%s)\n", user.DisplayName, user.Username)
	fmt.Printf("📧 Email: %s\n", user.Email)
	fmt.Printf("🔐 Method: API Token\n\n")

	return cmd.saveToProfile(email, token)
}

func (cmd *LoginCmd) interactiveLogin(ctx context.Context) error {
	fmt.Println("🚀 Welcome to Bitbucket CLI Authentication")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("🔑 Authentication uses API tokens (email + token)")
	fmt.Println("📋 Create an API token at: https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("📧 Atlassian account email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	fmt.Print("🔑 API token (hidden): ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Println()
	if token == "" {
		return fmt.Errorf("API token cannot be empty")
	}

	return cmd.authenticateAndSave(ctx, email, token, "interactive input")
}

func (cmd *LoginCmd) saveToProfile(email, token string) error {
	profile, err := detectShellProfile()
	if err != nil {
		fmt.Printf("⚠️  %v\n", err)
		cmd.printManualInstructions(email, token)
		return nil
	}

	if err := writeEnvsToProfile(profile, [][2]string{
		{"BITBUCKET_EMAIL", email},
		{"BITBUCKET_API_TOKEN", token},
	}); err != nil {
		return fmt.Errorf("failed to write to %s: %w", profile, err)
	}

	fmt.Printf("💾 Credentials saved to %s\n", profile)

	cmd.promptSonarCloudToken(profile)

	fmt.Printf("\n💡 Run 'source %s' or open a new terminal for changes to take effect\n", profile)
	return nil
}

func (cmd *LoginCmd) promptSonarCloudToken(profile string) {
	if os.Getenv("SONARCLOUD_TOKEN") != "" {
		fmt.Println("☁️  SonarCloud token already set in environment")
		return
	}

	fmt.Print("\n☁️  SonarCloud token (optional, press Enter to skip): ")
	sonarBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Printf("⚠️  Could not read SonarCloud token: %v (skipping)\n", err)
		return
	}
	sonarToken := strings.TrimSpace(string(sonarBytes))

	if sonarToken == "" {
		return
	}

	if err := writeEnvToProfile(profile, "SONARCLOUD_TOKEN", sonarToken); err != nil {
		fmt.Printf("⚠️  Failed to save SonarCloud token: %v\n", err)
		return
	}
	fmt.Println("☁️  SonarCloud token saved")
}

func (cmd *LoginCmd) printManualInstructions(email, token string) {
	fmt.Println("\n💡 Add these to your shell profile manually:")
	fmt.Printf("   export BITBUCKET_EMAIL=%q\n", email)
	fmt.Printf("   export BITBUCKET_API_TOKEN=%q\n", token)
}
