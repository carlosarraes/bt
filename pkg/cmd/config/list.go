package config

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ListCmd handles the config list command
type ListCmd struct {
	Output  string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor bool   // Passed from global flag
}

// Run executes the config list command
func (cmd *ListCmd) Run(ctx context.Context) error {
	// Create config manager
	cm, err := NewConfigManager()
	if err != nil {
		return err
	}

	// Get all configuration values
	values := cm.GetAllValues()

	// Format and display the result
	return cmd.formatOutput(values)
}

// formatOutput formats and displays all configuration values
func (cmd *ListCmd) formatOutput(values map[string]interface{}) error {
	switch cmd.Output {
	case "json":
		return cmd.formatJSON(values)
	case "yaml":
		return cmd.formatYAML(values)
	default:
		return cmd.formatTable(values)
	}
}

// formatTable outputs configuration in table format
func (cmd *ListCmd) formatTable(values map[string]interface{}) error {
	if len(values) == 0 {
		fmt.Println("No configuration found")
		return nil
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Calculate max key width for alignment
	maxKeyWidth := 0
	for _, key := range keys {
		if len(key) > maxKeyWidth {
			maxKeyWidth = len(key)
		}
	}

	// Display configuration
	fmt.Println("Configuration:")
	fmt.Println(strings.Repeat("─", maxKeyWidth+20))

	for _, key := range keys {
		value := values[key]
		formattedValue := formatValue(value)
		fmt.Printf("%-*s  %s\n", maxKeyWidth, key, formattedValue)
	}

	return nil
}

// formatJSON outputs configuration in JSON format
func (cmd *ListCmd) formatJSON(values map[string]interface{}) error {
	formatter, err := createFormatter("json", cmd.NoColor)
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"configuration": values,
	}

	return formatter.Format(result)
}

// formatYAML outputs configuration in YAML format
func (cmd *ListCmd) formatYAML(values map[string]interface{}) error {
	formatter, err := createFormatter("yaml", cmd.NoColor)
	if err != nil {
		return err
	}

	result := map[string]interface{}{
		"configuration": values,
	}

	return formatter.Format(result)
}
