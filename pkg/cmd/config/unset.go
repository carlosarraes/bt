package config

import (
	"context"
	"fmt"
)

// UnsetCmd handles the config unset command
type UnsetCmd struct {
	Key string `arg:"" help:"Configuration key to remove (e.g., auth.default_workspace)"`
}

// Run executes the config unset command
func (cmd *UnsetCmd) Run(ctx context.Context) error {
	// Create config manager
	cm, err := NewConfigManager()
	if err != nil {
		return err
	}

	// Unset the value
	if err := cm.UnsetValue(cmd.Key); err != nil {
		return err
	}

	// Save the configuration
	if err := cm.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Confirm the change
	fmt.Printf("✓ Unset %s\n", cmd.Key)
	return nil
}
