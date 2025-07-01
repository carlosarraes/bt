package config

import (
	"time"
)

// Config represents the main configuration structure for bt CLI
type Config struct {
	Version  int           `koanf:"version" yaml:"version"`
	Auth     AuthConfig    `koanf:"auth" yaml:"auth"`
	API      APIConfig     `koanf:"api" yaml:"api"`
	Defaults DefaultConfig `koanf:"defaults" yaml:"defaults"`
}

// AuthConfig holds authentication-related configuration
type AuthConfig struct {
	Method           string `koanf:"method" yaml:"method"`
	DefaultWorkspace string `koanf:"default_workspace" yaml:"default_workspace"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	BaseURL string        `koanf:"base_url" yaml:"base_url"`
	Timeout time.Duration `koanf:"timeout" yaml:"timeout"`
}

// DefaultConfig holds default preferences
type DefaultConfig struct {
	OutputFormat string `koanf:"output_format" yaml:"output_format"`
}

// NewDefaultConfig returns a new Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		Version: 1,
		Auth: AuthConfig{
			Method:           "app_password",
			DefaultWorkspace: "",
		},
		API: APIConfig{
			BaseURL: "https://api.bitbucket.org/2.0",
			Timeout: 30 * time.Second,
		},
		Defaults: DefaultConfig{
			OutputFormat: "table",
		},
	}
}

// Validate ensures the configuration is valid
func (c *Config) Validate() error {
	if c.Version < 1 {
		return ErrInvalidVersion
	}

	if c.Auth.Method != "" {
		if !isValidAuthMethod(c.Auth.Method) {
			return ErrInvalidAuthMethod
		}
	}

	if c.API.BaseURL == "" {
		return ErrEmptyBaseURL
	}

	if c.API.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	if c.Defaults.OutputFormat != "" {
		if !isValidOutputFormat(c.Defaults.OutputFormat) {
			return ErrInvalidOutputFormat
		}
	}

	return nil
}

// AuthMethod constants
const (
	AuthMethodAppPassword = "app_password"
	AuthMethodOAuth       = "oauth"
	AuthMethodAccessToken = "access_token"
)

// OutputFormat constants
const (
	OutputFormatTable = "table"
	OutputFormatJSON  = "json"
	OutputFormatYAML  = "yaml"
)

// isValidAuthMethod checks if the provided auth method is valid
func isValidAuthMethod(method string) bool {
	switch method {
	case AuthMethodAppPassword, AuthMethodOAuth, AuthMethodAccessToken:
		return true
	default:
		return false
	}
}

// isValidOutputFormat checks if the provided output format is valid
func isValidOutputFormat(format string) bool {
	switch format {
	case OutputFormatTable, OutputFormatJSON, OutputFormatYAML:
		return true
	default:
		return false
	}
}