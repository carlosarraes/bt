package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/config"
	"github.com/carlosarraes/bt/pkg/output"
)

// ConfigManager provides operations for managing configuration
type ConfigManager struct {
	loader *config.Loader
	config *config.Config
}

// NewConfigManager creates a new config manager
func NewConfigManager() (*ConfigManager, error) {
	loader := config.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &ConfigManager{
		loader: loader,
		config: cfg,
	}, nil
}

// GetValue retrieves a configuration value by key (supports nested keys like "auth.default_workspace")
func (cm *ConfigManager) GetValue(key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	value := reflect.ValueOf(cm.config).Elem()

	for _, part := range parts {
		if !value.IsValid() {
			return nil, fmt.Errorf("invalid configuration path: %s", key)
		}

		// Convert part to proper field name (camelCase)
		fieldName := toCamelCase(part)
		field := value.FieldByName(fieldName)
		
		if !field.IsValid() {
			return nil, fmt.Errorf("configuration key not found: %s", key)
		}
		
		value = field
	}

	if !value.IsValid() {
		return nil, fmt.Errorf("invalid configuration value: %s", key)
	}

	return value.Interface(), nil
}

// SetValue sets a configuration value by key with validation
func (cm *ConfigManager) SetValue(key, valueStr string) error {
	parts := strings.Split(key, ".")
	value := reflect.ValueOf(cm.config).Elem()
	
	// Navigate to the parent of the target field
	for i, part := range parts[:len(parts)-1] {
		fieldName := toCamelCase(part)
		field := value.FieldByName(fieldName)
		
		if !field.IsValid() {
			return fmt.Errorf("invalid configuration path: %s", strings.Join(parts[:i+1], "."))
		}
		
		value = field
	}

	// Set the final field
	finalFieldName := toCamelCase(parts[len(parts)-1])
	field := value.FieldByName(finalFieldName)
	
	if !field.IsValid() {
		return fmt.Errorf("configuration key not found: %s", key)
	}
	
	if !field.CanSet() {
		return fmt.Errorf("configuration key is read-only: %s", key)
	}

	// Convert and set the value based on the field type
	if err := setFieldValue(field, valueStr); err != nil {
		return fmt.Errorf("failed to set %s: %w", key, err)
	}

	// Validate the updated configuration
	if err := cm.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// UnsetValue removes a configuration value by setting it to its zero value
func (cm *ConfigManager) UnsetValue(key string) error {
	parts := strings.Split(key, ".")
	value := reflect.ValueOf(cm.config).Elem()
	
	// Navigate to the parent of the target field
	for i, part := range parts[:len(parts)-1] {
		fieldName := toCamelCase(part)
		field := value.FieldByName(fieldName)
		
		if !field.IsValid() {
			return fmt.Errorf("invalid configuration path: %s", strings.Join(parts[:i+1], "."))
		}
		
		value = field
	}

	// Unset the final field
	finalFieldName := toCamelCase(parts[len(parts)-1])
	field := value.FieldByName(finalFieldName)
	
	if !field.IsValid() {
		return fmt.Errorf("configuration key not found: %s", key)
	}
	
	if !field.CanSet() {
		return fmt.Errorf("configuration key is read-only: %s", key)
	}

	// Set to zero value
	field.Set(reflect.Zero(field.Type()))

	// Validate the updated configuration
	if err := cm.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// Save saves the current configuration to file
func (cm *ConfigManager) Save() error {
	return cm.loader.Save(cm.config)
}

// GetAllValues returns all configuration as a map for listing
func (cm *ConfigManager) GetAllValues() map[string]interface{} {
	result := make(map[string]interface{})

	// Auth section
	result["auth.method"] = cm.config.Auth.Method
	result["auth.default_workspace"] = cm.config.Auth.DefaultWorkspace

	// API section
	result["api.base_url"] = cm.config.API.BaseURL
	result["api.timeout"] = cm.config.API.Timeout.String()

	// Defaults section
	result["defaults.output_format"] = cm.config.Defaults.OutputFormat

	result["llm.model"] = cm.config.LLM.Model

	// Version
	result["version"] = cm.config.Version

	return result
}

// setFieldValue sets a reflect.Value field from a string
func setFieldValue(field reflect.Value, valueStr string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(valueStr)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration specially
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(valueStr)
			if err != nil {
				return fmt.Errorf("invalid duration format: %v", err)
			}
			field.Set(reflect.ValueOf(duration))
		} else {
			intVal, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer: %v", err)
			}
			field.SetInt(intVal)
		}
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(valueStr)
		if err != nil {
			return fmt.Errorf("invalid boolean: %v", err)
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type())
	}
	return nil
}

// toCamelCase converts snake_case to CamelCase for struct field names
func toCamelCase(s string) string {
	// Handle special cases for struct field names
	switch s {
	case "api":
		return "API"
	case "auth":
		return "Auth"
	case "defaults":
		return "Defaults"
	case "version":
		return "Version"
	case "method":
		return "Method"
	case "default_workspace":
		return "DefaultWorkspace"
	case "base_url":
		return "BaseURL"
	case "timeout":
		return "Timeout"
	case "output_format":
		return "OutputFormat"
	case "llm":
		return "LLM"
	case "model":
		return "Model"
	}
	
	// Generic conversion for other cases
	parts := strings.Split(s, "_")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return result
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case time.Duration:
		return val.String()
	case string:
		if val == "" {
			return "(unset)"
		}
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// createFormatter creates an output formatter
func createFormatter(format string, noColor bool) (output.Formatter, error) {
	opts := &output.FormatterOptions{
		NoColor: noColor,
	}
	return output.NewFormatter(output.Format(format), opts)
}
