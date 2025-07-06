package config

import (
	"context"
	"fmt"
)

// SetCmd handles the config set command
type SetCmd struct {
	Key   string `arg:"" help:"Configuration key to set (e.g., auth.default_workspace)"`
	Value string `arg:"" help:"Configuration value to set"`
}

// Run executes the config set command
func (cmd *SetCmd) Run(ctx context.Context) error {
	// Create config manager
	cm, err := NewConfigManager()
	if err != nil {
		return err
	}

	// Set the value
	if err := cm.SetValue(cmd.Key, cmd.Value); err != nil {
		return err
	}

	// Save the configuration
	if err := cm.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Confirm the change
	fmt.Printf("✓ Set %s to %s\n", cmd.Key, cmd.Value)
	return nil
}