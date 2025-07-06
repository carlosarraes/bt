package config

import (
	"context"
	"fmt"
)

// GetCmd handles the config get command
type GetCmd struct {
	Key    string `arg:"" help:"Configuration key to retrieve (e.g., auth.default_workspace)"`
	Output string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor bool  // Passed from global flag
}

// Run executes the config get command
func (cmd *GetCmd) Run(ctx context.Context) error {
	// Create config manager
	cm, err := NewConfigManager()
	if err != nil {
		return err
	}

	// Get the value
	value, err := cm.GetValue(cmd.Key)
	if err != nil {
		return err
	}

	// Format and display the result
	return cmd.formatOutput(value)
}

// formatOutput formats and displays the retrieved value
func (cmd *GetCmd) formatOutput(value interface{}) error {
	switch cmd.Output {
	case "json":
		return cmd.formatJSON(value)
	case "yaml":
		return cmd.formatYAML(value)
	default:
		return cmd.formatTable(value)
	}
}

// formatTable outputs the value in table format
func (cmd *GetCmd) formatTable(value interface{}) error {
	fmt.Printf("%s: %s\n", cmd.Key, formatValue(value))
	return nil
}

// formatJSON outputs the value in JSON format
func (cmd *GetCmd) formatJSON(value interface{}) error {
	formatter, err := createFormatter("json", cmd.NoColor)
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"key":   cmd.Key,
		"value": value,
	}

	return formatter.Format(result)
}

// formatYAML outputs the value in YAML format
func (cmd *GetCmd) formatYAML(value interface{}) error {
	formatter, err := createFormatter("yaml", cmd.NoColor)
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"key":   cmd.Key,
		"value": value,
	}

	return formatter.Format(result)
}