package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	yamlv3 "gopkg.in/yaml.v3"
)

// Loader handles configuration loading and management
type Loader struct {
	k          *koanf.Koanf
	configPath string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		k: koanf.New("."),
	}
}

// Load loads configuration from file and environment variables
// Priority: Environment Variables > Config File > Defaults
func (l *Loader) Load() (*Config, error) {
	// Start with default configuration
	config := NewDefaultConfig()

	// Get config file path
	configPath, err := l.getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigLoad, err)
	}
	l.configPath = configPath

	// Load from config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := l.k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("%w: failed to load config file: %v", ErrConfigLoad, err)
		}
	}

	// Load environment variables with BT_ prefix
	if err := l.k.Load(env.Provider("BT_", ".", func(s string) string {
		// Convert BT_API_BASE_URL to api.base_url
		return l.transformEnvKey(s)
	}), nil); err != nil {
		return nil, fmt.Errorf("%w: failed to load environment variables: %v", ErrConfigLoad, err)
	}

	// Unmarshal into config struct
	if err := l.k.Unmarshal("", config); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal config: %v", ErrConfigLoad, err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigLoad, err)
	}

	return config, nil
}

// Save saves the configuration to file
func (l *Loader) Save(config *Config) error {
	if l.configPath == "" {
		configPath, err := l.getConfigPath()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrConfigSave, err)
		}
		l.configPath = configPath
	}

	// Ensure config directory exists
	configDir := filepath.Dir(l.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("%w: failed to create config directory: %v", ErrConfigSave, err)
	}

	// Marshal config to YAML using gopkg.in/yaml.v3
	yamlData, err := yamlv3.Marshal(config)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal config: %v", ErrConfigSave, err)
	}

	// Write to file
	if err := os.WriteFile(l.configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("%w: failed to write config file: %v", ErrConfigSave, err)
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func (l *Loader) GetConfigPath() (string, error) {
	return l.getConfigPath()
}

// getConfigPath determines the configuration file path
func (l *Loader) getConfigPath() (string, error) {
	// Check if BT_CONFIG_PATH environment variable is set
	if configPath := os.Getenv("BT_CONFIG_PATH"); configPath != "" {
		return configPath, nil
	}

	// Use XDG config directory or fallback to home directory
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %v", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "bt", "config.yml"), nil
}

// transformEnvKey transforms environment variable names to config keys
// BT_API_BASE_URL -> api.base_url
// BT_AUTH_METHOD -> auth.method
func (l *Loader) transformEnvKey(key string) string {
	// Remove BT_ prefix
	if len(key) > 3 && key[:3] == "BT_" {
		key = key[3:]
	}

	// Transform specific keys
	switch key {
	case "API_BASE_URL":
		return "api.base_url"
	case "API_TIMEOUT":
		return "api.timeout"
	case "AUTH_METHOD":
		return "auth.method"
	case "AUTH_DEFAULT_WORKSPACE":
		return "auth.default_workspace"
	case "DEFAULTS_OUTPUT_FORMAT":
		return "defaults.output_format"
	case "LLM_MODEL":
		return "llm.model"
	default:
		return key
	}
}

// Environment variable names for easy reference
const (
	EnvConfigPath          = "BT_CONFIG_PATH"
	EnvAPIBaseURL          = "BT_API_BASE_URL"
	EnvAPITimeout          = "BT_API_TIMEOUT"
	EnvAuthMethod          = "BT_AUTH_METHOD"
	EnvDefaultWorkspace    = "BT_AUTH_DEFAULT_WORKSPACE"
	EnvDefaultOutputFormat = "BT_DEFAULTS_OUTPUT_FORMAT"
	EnvLLMModel            = "BT_LLM_MODEL"
)