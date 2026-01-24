package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrorType represents the category of API error
type ErrorType string

const (
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypePermission     ErrorType = "permission"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeServer         ErrorType = "server"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeUnknown        ErrorType = "unknown"
)

// BitbucketError represents a structured error from the Bitbucket API
type BitbucketError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Detail     string    `json:"detail,omitempty"`
	StatusCode int       `json:"status_code"`
	RequestID  string    `json:"request_id,omitempty"`
	Raw        string    `json:"raw,omitempty"`
}

// Error implements the error interface
func (e *BitbucketError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// IsRetryable returns true if the error might be resolved by retrying
func (e *BitbucketError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit, ErrorTypeServer, ErrorTypeNetwork:
		return true
	case ErrorTypeAuthentication, ErrorTypePermission, ErrorTypeNotFound, ErrorTypeValidation:
		return false
	default:
		return e.StatusCode >= 500
	}
}

// IsRateLimit returns true if the error is due to rate limiting
func (e *BitbucketError) IsRateLimit() bool {
	return e.Type == ErrorTypeRateLimit || e.StatusCode == 429
}

// BitbucketAPIError represents the error structure returned by Bitbucket API
type BitbucketAPIError struct {
	Error struct {
		Message string                 `json:"message"`
		Detail  string                 `json:"detail,omitempty"`
		Data    map[string]interface{} `json:"data,omitempty"`
	} `json:"error"`
	Type      string `json:"type,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// ParseError creates a structured error from an HTTP response
func ParseError(resp *http.Response) error {
	if resp == nil {
		return &BitbucketError{
			Type:    ErrorTypeNetwork,
			Message: "No response received",
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &BitbucketError{
			Type:       ErrorTypeNetwork,
			Message:    "Failed to read error response",
			Detail:     err.Error(),
			StatusCode: resp.StatusCode,
		}
	}

	// Determine error type based on status code
	errorType := categorizeStatusCode(resp.StatusCode)

	// Try to parse as Bitbucket API error JSON
	var apiError BitbucketAPIError
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Error.Message != "" {
		return &BitbucketError{
			Type:       errorType,
			Message:    apiError.Error.Message,
			Detail:     apiError.Error.Detail,
			StatusCode: resp.StatusCode,
			RequestID:  getRequestID(resp),
			Raw:        string(body),
		}
	}

	// Fallback to generic error message
	message := getGenericErrorMessage(resp.StatusCode)
	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = resp.Status
	}

	return &BitbucketError{
		Type:       errorType,
		Message:    message,
		Detail:     detail,
		StatusCode: resp.StatusCode,
		RequestID:  getRequestID(resp),
		Raw:        string(body),
	}
}

// categorizeStatusCode determines the error type based on HTTP status code
func categorizeStatusCode(statusCode int) ErrorType {
	switch {
	case statusCode == 401:
		return ErrorTypeAuthentication
	case statusCode == 403:
		return ErrorTypePermission
	case statusCode == 404:
		return ErrorTypeNotFound
	case statusCode == 422:
		return ErrorTypeValidation
	case statusCode == 429:
		return ErrorTypeRateLimit
	case statusCode >= 500 && statusCode < 600:
		return ErrorTypeServer
	case statusCode >= 400 && statusCode < 500:
		return ErrorTypeValidation
	default:
		return ErrorTypeUnknown
	}
}

// getGenericErrorMessage returns a user-friendly error message for HTTP status codes
func getGenericErrorMessage(statusCode int) string {
	switch statusCode {
	case 400:
		return "Bad request"
	case 401:
		return "Authentication required"
	case 403:
		return "Permission denied"
	case 404:
		return "Resource not found"
	case 422:
		return "Validation failed"
	case 429:
		return "Rate limit exceeded"
	case 500:
		return "Internal server error"
	case 502:
		return "Bad gateway"
	case 503:
		return "Service unavailable"
	case 504:
		return "Gateway timeout"
	default:
		return fmt.Sprintf("HTTP %d error", statusCode)
	}
}

// getRequestID extracts the request ID from response headers
func getRequestID(resp *http.Response) string {
	// Common header names for request IDs
	headers := []string{
		"X-Request-Id",
		"X-Request-ID",
		"Request-Id",
		"Request-ID",
	}

	for _, header := range headers {
		if id := resp.Header.Get(header); id != "" {
			return id
		}
	}

	return ""
}

// NewNetworkError creates a network-related error
func NewNetworkError(message string, err error) error {
	detail := ""
	if err != nil {
		detail = err.Error()
	}

	return &BitbucketError{
		Type:    ErrorTypeNetwork,
		Message: message,
		Detail:  detail,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message, detail string) error {
	return &BitbucketError{
		Type:    ErrorTypeValidation,
		Message: message,
		Detail:  detail,
	}
}

// IsNetworkError returns true if the error is network-related
func IsNetworkError(err error) bool {
	if bbErr, ok := err.(*BitbucketError); ok {
		return bbErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsAuthenticationError returns true if the error is authentication-related
func IsAuthenticationError(err error) bool {
	if bbErr, ok := err.(*BitbucketError); ok {
		return bbErr.Type == ErrorTypeAuthentication
	}
	return false
}

// IsRateLimitError returns true if the error is due to rate limiting
func IsRateLimitError(err error) bool {
	if bbErr, ok := err.(*BitbucketError); ok {
		return bbErr.IsRateLimit()
	}
	return false
}

// IsRetryableError returns true if the error might be resolved by retrying
func IsRetryableError(err error) bool {
	if bbErr, ok := err.(*BitbucketError); ok {
		return bbErr.IsRetryable()
	}
	return false
}
