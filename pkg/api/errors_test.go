package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedType   ErrorType
		expectedMsg    string
		expectedDetail string
	}{
		{
			name:           "Bitbucket API error with detail",
			statusCode:     400,
			body:           `{"error": {"message": "Invalid request", "detail": "Missing required field"}}`,
			expectedType:   ErrorTypeValidation,
			expectedMsg:    "Invalid request",
			expectedDetail: "Missing required field",
		},
		{
			name:         "Authentication error",
			statusCode:   401,
			body:         `{"error": {"message": "Authentication required"}}`,
			expectedType: ErrorTypeAuthentication,
			expectedMsg:  "Authentication required",
		},
		{
			name:         "Permission error",
			statusCode:   403,
			body:         `{"error": {"message": "Access denied"}}`,
			expectedType: ErrorTypePermission,
			expectedMsg:  "Access denied",
		},
		{
			name:         "Not found error",
			statusCode:   404,
			body:         `{"error": {"message": "Repository not found"}}`,
			expectedType: ErrorTypeNotFound,
			expectedMsg:  "Repository not found",
		},
		{
			name:         "Rate limit error",
			statusCode:   429,
			body:         `{"error": {"message": "Rate limit exceeded"}}`,
			expectedType: ErrorTypeRateLimit,
			expectedMsg:  "Rate limit exceeded",
		},
		{
			name:         "Server error",
			statusCode:   500,
			body:         `{"error": {"message": "Internal server error"}}`,
			expectedType: ErrorTypeServer,
			expectedMsg:  "Internal server error",
		},
		{
			name:           "Non-JSON error response",
			statusCode:     400,
			body:           "Bad Request",
			expectedType:   ErrorTypeValidation,
			expectedMsg:    "Bad request",
			expectedDetail: "Bad Request",
		},
		{
			name:           "Empty response body",
			statusCode:     500,
			body:           "",
			expectedType:   ErrorTypeServer,
			expectedMsg:    "Internal server error",
			expectedDetail: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Status:     http.StatusText(tt.statusCode),
				Body:       io.NopCloser(bytes.NewReader([]byte(tt.body))),
				Header:     make(http.Header),
			}

			// Parse error
			err := ParseError(resp)
			require.Error(t, err)

			// Check error type
			bbErr, ok := err.(*BitbucketError)
			require.True(t, ok, "Expected BitbucketError")

			assert.Equal(t, tt.expectedType, bbErr.Type)
			assert.Equal(t, tt.expectedMsg, bbErr.Message)
			assert.Equal(t, tt.statusCode, bbErr.StatusCode)

			if tt.expectedDetail != "" {
				assert.Equal(t, tt.expectedDetail, bbErr.Detail)
			}
		})
	}
}

func TestParseErrorWithNilResponse(t *testing.T) {
	err := ParseError(nil)
	require.Error(t, err)

	bbErr, ok := err.(*BitbucketError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeNetwork, bbErr.Type)
	assert.Equal(t, "No response received", bbErr.Message)
}

func TestParseErrorWithRequestID(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Status:     "400 Bad Request",
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": {"message": "Bad request"}}`))),
		Header:     http.Header{"X-Request-Id": []string{"abc123"}},
	}

	err := ParseError(resp)
	require.Error(t, err)

	bbErr, ok := err.(*BitbucketError)
	require.True(t, ok)
	assert.Equal(t, "abc123", bbErr.RequestID)
}

func TestBitbucketErrorMethods(t *testing.T) {
	tests := []struct {
		name       string
		errorType  ErrorType
		statusCode int
		retryable  bool
		rateLimit  bool
	}{
		{
			name:       "Authentication error",
			errorType:  ErrorTypeAuthentication,
			statusCode: 401,
			retryable:  false,
			rateLimit:  false,
		},
		{
			name:       "Rate limit error",
			errorType:  ErrorTypeRateLimit,
			statusCode: 429,
			retryable:  true,
			rateLimit:  true,
		},
		{
			name:       "Server error",
			errorType:  ErrorTypeServer,
			statusCode: 500,
			retryable:  true,
			rateLimit:  false,
		},
		{
			name:       "Network error",
			errorType:  ErrorTypeNetwork,
			statusCode: 0,
			retryable:  true,
			rateLimit:  false,
		},
		{
			name:       "Validation error",
			errorType:  ErrorTypeValidation,
			statusCode: 422,
			retryable:  false,
			rateLimit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BitbucketError{
				Type:       tt.errorType,
				Message:    "Test error",
				StatusCode: tt.statusCode,
			}

			assert.Equal(t, tt.retryable, err.IsRetryable())
			assert.Equal(t, tt.rateLimit, err.IsRateLimit())
		})
	}
}

func TestErrorHelperFunctions(t *testing.T) {
	networkErr := NewNetworkError("Connection failed", nil)
	validationErr := NewValidationError("Invalid input", "Field is required")

	// Test network error
	bbErr, ok := networkErr.(*BitbucketError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeNetwork, bbErr.Type)
	assert.Equal(t, "Connection failed", bbErr.Message)

	// Test validation error
	bbErr, ok = validationErr.(*BitbucketError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeValidation, bbErr.Type)
	assert.Equal(t, "Invalid input", bbErr.Message)
	assert.Equal(t, "Field is required", bbErr.Detail)

	// Test error type checking functions
	assert.True(t, IsNetworkError(networkErr))
	assert.False(t, IsNetworkError(validationErr))

	authErr := &BitbucketError{Type: ErrorTypeAuthentication}
	assert.True(t, IsAuthenticationError(authErr))
	assert.False(t, IsAuthenticationError(networkErr))

	rateLimitErr := &BitbucketError{Type: ErrorTypeRateLimit}
	assert.True(t, IsRateLimitError(rateLimitErr))
	assert.False(t, IsRateLimitError(networkErr))

	assert.True(t, IsRetryableError(networkErr))
	assert.False(t, IsRetryableError(validationErr))
}

func TestErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      *BitbucketError
		expected string
	}{
		{
			name: "Error with detail",
			err: &BitbucketError{
				Type:    ErrorTypeValidation,
				Message: "Invalid request",
				Detail:  "Missing field",
			},
			expected: "validation: Invalid request (Missing field)",
		},
		{
			name: "Error without detail",
			err: &BitbucketError{
				Type:    ErrorTypeAuthentication,
				Message: "Auth required",
			},
			expected: "authentication: Auth required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestCategorizeStatusCode(t *testing.T) {
	tests := []struct {
		statusCode   int
		expectedType ErrorType
	}{
		{400, ErrorTypeValidation},
		{401, ErrorTypeAuthentication},
		{403, ErrorTypePermission},
		{404, ErrorTypeNotFound},
		{422, ErrorTypeValidation},
		{429, ErrorTypeRateLimit},
		{500, ErrorTypeServer},
		{502, ErrorTypeServer},
		{503, ErrorTypeServer},
		{999, ErrorTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			result := categorizeStatusCode(tt.statusCode)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestGetGenericErrorMessage(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   string
	}{
		{400, "Bad request"},
		{401, "Authentication required"},
		{403, "Permission denied"},
		{404, "Resource not found"},
		{422, "Validation failed"},
		{429, "Rate limit exceeded"},
		{500, "Internal server error"},
		{502, "Bad gateway"},
		{503, "Service unavailable"},
		{504, "Gateway timeout"},
		{418, "HTTP 418 error"}, // Non-standard code
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			result := getGenericErrorMessage(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "X-Request-Id header",
			headers:  map[string]string{"X-Request-Id": "abc123"},
			expected: "abc123",
		},
		{
			name:     "X-Request-ID header",
			headers:  map[string]string{"X-Request-ID": "def456"},
			expected: "def456",
		},
		{
			name:     "Request-Id header",
			headers:  map[string]string{"Request-Id": "ghi789"},
			expected: "ghi789",
		},
		{
			name:     "Request-ID header",
			headers:  map[string]string{"Request-ID": "jkl012"},
			expected: "jkl012",
		},
		{
			name:     "No request ID header",
			headers:  map[string]string{"Other-Header": "value"},
			expected: "",
		},
		{
			name:     "Multiple headers, first one wins",
			headers:  map[string]string{"X-Request-Id": "first", "Request-Id": "second"},
			expected: "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
			}

			for key, value := range tt.headers {
				resp.Header.Set(key, value)
			}

			result := getRequestID(resp)
			assert.Equal(t, tt.expected, result)
		})
	}
}
