package config

import (
	"fmt"
	"strings"
	"time"

	yamlv3 "gopkg.in/yaml.v3"
)

// Config represents the main configuration structure for bt CLI
type Config struct {
	Version  int           `koanf:"version" yaml:"version"`
	Auth     AuthConfig    `koanf:"auth" yaml:"auth"`
	API      APIConfig     `koanf:"api" yaml:"api"`
	Defaults DefaultConfig `koanf:"defaults" yaml:"defaults"`
	PR       PRConfig      `koanf:"pr" yaml:"pr"`
	LLM      LLMConfig     `koanf:"llm" yaml:"llm"`
	Pick     PickConfig    `koanf:"pick" yaml:"pick"`
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

type PRConfig struct {
	BranchSuffixMapping map[string]string `koanf:"branch_suffix_mapping" yaml:"branch_suffix_mapping"`
}

type LLMConfig struct {
	Model string `koanf:"model" yaml:"model"`
}

// Prefixes holds one or more branch prefixes. Supports both a single string
// and a YAML array for backwards compatibility. Comma-separated strings are
// split automatically (e.g. "ZEX-,ZGR-").
type Prefixes []string

func (p *Prefixes) UnmarshalYAML(value *yamlv3.Node) error {
	switch value.Kind {
	case yamlv3.ScalarNode:
		*p = ParsePrefixes(value.Value)
		return nil
	case yamlv3.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*p = items
		return nil
	default:
		return fmt.Errorf("prefix must be a string or list of strings")
	}
}

// ParsePrefixes splits a comma-separated string into individual prefixes.
func ParsePrefixes(s string) Prefixes {
	var result Prefixes
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

type PickConfig struct {
	Prefix    Prefixes `koanf:"prefix" yaml:"prefix"`
	SuffixPrd string   `koanf:"suffix_prd" yaml:"suffix_prd"`
	SuffixHml string   `koanf:"suffix_hml" yaml:"suffix_hml"`
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
		PR: PRConfig{
			BranchSuffixMapping: map[string]string{
				"hml": "homolog",
				"prd": "main",
			},
		},
		LLM: LLMConfig{
			Model: "gpt-5.4-mini",
		},
		Pick: PickConfig{
			Prefix:    Prefixes{"ZUP-"},
			SuffixPrd: "-prd",
			SuffixHml: "-hml",
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

	if len(c.Pick.Prefix) == 0 {
		return ErrEmptyPickPrefix
	}
	for _, p := range c.Pick.Prefix {
		if p == "" {
			return ErrEmptyPickPrefix
		}
	}
	if c.Pick.SuffixPrd == "" {
		return ErrEmptyPickSuffixPrd
	}
	if c.Pick.SuffixHml == "" {
		return ErrEmptyPickSuffixHml
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
