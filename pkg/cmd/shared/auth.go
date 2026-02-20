package shared

import (
	"fmt"

	"github.com/carlosarraes/bt/pkg/auth"
)

func CreateAuthManager() (auth.AuthManager, error) {
	email, token := auth.GetCredentials()

	if email == "" || token == "" {
		return nil, fmt.Errorf("no credentials found. Run 'bt auth login' or set BITBUCKET_EMAIL and BITBUCKET_API_TOKEN")
	}

	config := auth.DefaultConfig()
	return auth.NewAuthManager(config)
}

func CreateAuthManagerWithMethod(method auth.AuthMethod) (auth.AuthManager, error) {
	config := auth.DefaultConfig()
	config.Method = method
	return auth.NewAuthManager(config)
}
