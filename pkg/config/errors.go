package config

import "errors"

// Configuration validation errors
var (
	ErrInvalidVersion      = errors.New("invalid configuration version")
	ErrInvalidAuthMethod   = errors.New("invalid authentication method")
	ErrEmptyBaseURL        = errors.New("API base URL cannot be empty")
	ErrInvalidTimeout      = errors.New("API timeout must be positive")
	ErrInvalidOutputFormat = errors.New("invalid output format")
	ErrConfigNotFound      = errors.New("configuration file not found")
	ErrConfigLoad          = errors.New("failed to load configuration")
	ErrConfigSave          = errors.New("failed to save configuration")
)
