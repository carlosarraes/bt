package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/carlosarraes/bt/pkg/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/auth-demo/main.go <method>")
		fmt.Println("Methods: app_password, access_token, oauth")
		fmt.Println("")
		fmt.Println("Environment variables:")
		fmt.Println("  BITBUCKET_TOKEN - for access_token method")
		fmt.Println("  BITBUCKET_USERNAME - for app_password method")
		fmt.Println("  BITBUCKET_PASSWORD - for app_password method")
		os.Exit(1)
	}

	method := os.Args[1]
	var authMethod auth.AuthMethod

	switch method {
	case "app_password":
		authMethod = auth.AuthMethodAppPassword
	case "access_token":
		authMethod = auth.AuthMethodAccessToken
	case "oauth":
		authMethod = auth.AuthMethodOAuth
	default:
		log.Fatalf("Unknown method: %s", method)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create storage
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create config
	config := &auth.Config{
		Method:        authMethod,
		BaseURL:       "https://api.bitbucket.org/2.0",
		Timeout:       30,
		OAuthClientID: "bt-cli", // This would need to be registered with Bitbucket
	}

	// Create auth manager
	manager, err := auth.NewAuthManager(config, storage)
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	fmt.Printf("Testing authentication method: %s\n", method)

	// Test authentication
	fmt.Println("Authenticating...")
	if err := manager.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("✓ Authentication successful!")

	// Test user retrieval
	fmt.Println("Retrieving user information...")
	user, err := manager.GetAuthenticatedUser(ctx)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("✓ User: %s (%s)\n", user.Username, user.DisplayName)
	fmt.Printf("  Account ID: %s\n", user.AccountID)
	fmt.Printf("  UUID: %s\n", user.UUID)
	if user.Email != "" {
		fmt.Printf("  Email: %s\n", user.Email)
	}

	// Test authentication status
	fmt.Println("Checking authentication status...")
	isAuth, err := manager.IsAuthenticated(ctx)
	if err != nil {
		log.Fatalf("Failed to check auth status: %v", err)
	}

	if isAuth {
		fmt.Println("✓ Authentication is valid")
	} else {
		fmt.Println("✗ Authentication is invalid")
	}

	// Test logout
	fmt.Println("Testing logout...")
	if err := manager.Logout(); err != nil {
		log.Fatalf("Failed to logout: %v", err)
	}

	fmt.Println("✓ Logout successful")

	// Verify logout
	isAuth, err = manager.IsAuthenticated(ctx)
	if err != nil {
		log.Fatalf("Failed to check auth status after logout: %v", err)
	}

	if !isAuth {
		fmt.Println("✓ Logout verified - no longer authenticated")
	} else {
		fmt.Println("✗ Logout failed - still authenticated")
	}

	fmt.Println("\nAuthentication demo completed successfully!")
}