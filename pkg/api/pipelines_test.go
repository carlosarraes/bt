package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetStepLogsValidation(t *testing.T) {
	// Create a mock client
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       "https://api.bitbucket.org/2.0",
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test validation errors
	_, err = client.Pipelines.GetStepLogs(ctx, "", "repo", "pipeline-uuid", "step-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace")

	_, err = client.Pipelines.GetStepLogs(ctx, "workspace", "", "pipeline-uuid", "step-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository slug")

	_, err = client.Pipelines.GetStepLogs(ctx, "workspace", "repo", "", "step-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline UUID")

	_, err = client.Pipelines.GetStepLogs(ctx, "workspace", "repo", "pipeline-uuid", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step UUID")
}

func TestGetStepLogsEndpointTrying(t *testing.T) {
	// Track which endpoints were called
	var calledEndpoints []string
	
	// Create test server that tracks endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledEndpoints = append(calledEndpoints, r.URL.Path)
		
		// Return 404 for all requests to test fallback behavior
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"message": "Not found"}}`))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	mockAuth.On("SetHTTPHeaders", mock.AnythingOfType("*http.Request")).Return(nil)
	
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 0, // No retries for cleaner testing
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// This should fail but we want to verify which endpoints were tried
	_, err = client.Pipelines.GetStepLogs(ctx, "workspace", "repo", "pipeline-uuid", "step-uuid")
	
	// Should get an error (expected since all endpoints return 404)
	assert.Error(t, err)

	// Verify that both direct endpoints were tried before falling back
	assert.GreaterOrEqual(t, len(calledEndpoints), 2, "Should try at least the two direct endpoints")
	
	// Check that both direct endpoints were tried
	found := 0
	for _, endpoint := range calledEndpoints {
		if strings.Contains(endpoint, "/log") || strings.Contains(endpoint, "/logs") {
			found++
		}
	}
	assert.GreaterOrEqual(t, found, 2, "Should try both singular and plural log endpoints")

	t.Logf("Called endpoints: %v", calledEndpoints)
}