package shared

import (
	"fmt"

	"github.com/carlosarraes/bt/pkg/auth"
)

func CreateAuthManager() (auth.AuthManager, error) {
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		return nil, err
	}

	if !storage.Exists("auth") {
		return nil, fmt.Errorf("no stored credentials found. Please run 'bt auth login' first")
	}

	var credentials auth.StoredCredentials
	if err := storage.Retrieve("auth", &credentials); err != nil {
		return nil, fmt.Errorf("failed to load stored credentials: %w", err)
	}

	config := auth.DefaultConfig()
	config.Method = credentials.Method

	return auth.NewAuthManager(config, storage)
}

func CreateAuthManagerWithMethod(method auth.AuthMethod) (auth.AuthManager, error) {
	storage, err := auth.NewFileCredentialStorage()
	if err != nil {
		return nil, err
	}

	config := auth.DefaultConfig()
	config.Method = method

	return auth.NewAuthManager(config, storage)
}
