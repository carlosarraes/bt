package auth

import (
	"context"
	"fmt"
)

// RefreshCmd handles auth refresh command
type RefreshCmd struct{}

// Run executes the auth refresh command
func (cmd *RefreshCmd) Run(ctx context.Context) error {
	return fmt.Errorf("❌ Token refresh is not needed for API tokens\n💡 API tokens don't expire and don't need refresh\n🔄 If you need to update your token, run 'bt auth login' with a new token")
}
