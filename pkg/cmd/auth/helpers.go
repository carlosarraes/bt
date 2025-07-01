package auth

import (
	"github.com/carlosarraes/bt/pkg/auth"
)

// createAuthManager is a helper function to create an auth manager with the specified method
func createAuthManager(method auth.AuthMethod) (auth.AuthManager, error) {
	// Create file-based credential storage
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		return nil, err
	}

	// Create config with the specified method
	config := auth.DefaultConfig()
	config.Method = method

	// Create and return the auth manager
	return auth.NewAuthManager(config, storage)
}