package config

import (
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	config := NewDefaultConfig()

	if config.Version != 1 {
		t.Errorf("Expected version 1, got %d", config.Version)
	}

	if config.Auth.Method != AuthMethodAppPassword {
		t.Errorf("Expected auth method %s, got %s", AuthMethodAppPassword, config.Auth.Method)
	}

	if config.API.BaseURL != "https://api.bitbucket.org/2.0" {
		t.Errorf("Expected base URL https://api.bitbucket.org/2.0, got %s", config.API.BaseURL)
	}

	if config.API.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.API.Timeout)
	}

	if config.Defaults.OutputFormat != OutputFormatTable {
		t.Errorf("Expected output format %s, got %s", OutputFormatTable, config.Defaults.OutputFormat)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errType error
	}{
		{
			name:    "valid default config",
			config:  NewDefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid version",
			config: &Config{
				Version: 0,
				API:     APIConfig{BaseURL: "https://api.bitbucket.org/2.0", Timeout: 30 * time.Second},
			},
			wantErr: true,
			errType: ErrInvalidVersion,
		},
		{
			name: "invalid auth method",
			config: &Config{
				Version: 1,
				Auth:    AuthConfig{Method: "invalid"},
				API:     APIConfig{BaseURL: "https://api.bitbucket.org/2.0", Timeout: 30 * time.Second},
			},
			wantErr: true,
			errType: ErrInvalidAuthMethod,
		},
		{
			name: "empty base URL",
			config: &Config{
				Version: 1,
				Auth:    AuthConfig{Method: AuthMethodAppPassword},
				API:     APIConfig{BaseURL: "", Timeout: 30 * time.Second},
			},
			wantErr: true,
			errType: ErrEmptyBaseURL,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Version: 1,
				Auth:    AuthConfig{Method: AuthMethodAppPassword},
				API:     APIConfig{BaseURL: "https://api.bitbucket.org/2.0", Timeout: -1},
			},
			wantErr: true,
			errType: ErrInvalidTimeout,
		},
		{
			name: "invalid output format",
			config: &Config{
				Version:  1,
				Auth:     AuthConfig{Method: AuthMethodAppPassword},
				API:      APIConfig{BaseURL: "https://api.bitbucket.org/2.0", Timeout: 30 * time.Second},
				Defaults: DefaultConfig{OutputFormat: "invalid"},
			},
			wantErr: true,
			errType: ErrInvalidOutputFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("Config.Validate() error = %v, expected %v", err, tt.errType)
			}
		})
	}
}

func TestIsValidAuthMethod(t *testing.T) {
	tests := []struct {
		method string
		valid  bool
	}{
		{AuthMethodAppPassword, true},
		{AuthMethodOAuth, true},
		{AuthMethodAccessToken, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := isValidAuthMethod(tt.method); got != tt.valid {
				t.Errorf("isValidAuthMethod(%s) = %v, want %v", tt.method, got, tt.valid)
			}
		})
	}
}

func TestIsValidOutputFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{OutputFormatTable, true},
		{OutputFormatJSON, true},
		{OutputFormatYAML, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			if got := isValidOutputFormat(tt.format); got != tt.valid {
				t.Errorf("isValidOutputFormat(%s) = %v, want %v", tt.format, got, tt.valid)
			}
		})
	}
}