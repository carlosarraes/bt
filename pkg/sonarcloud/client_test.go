package sonarcloud

import (
	"testing"
	"time"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()
	
	if config.BaseURL != DefaultBaseURL {
		t.Errorf("Expected BaseURL %s, got %s", DefaultBaseURL, config.BaseURL)
	}
	
	if config.Timeout != DefaultTimeout {
		t.Errorf("Expected Timeout %v, got %v", DefaultTimeout, config.Timeout)
	}
	
	if config.RetryAttempts != DefaultRetryAttempts {
		t.Errorf("Expected RetryAttempts %d, got %d", DefaultRetryAttempts, config.RetryAttempts)
	}
	
	if !config.EnableCache {
		t.Error("Expected EnableCache to be true")
	}
	
	if config.Jitter != true {
		t.Error("Expected Jitter to be true")
	}
}

func TestNewCache(t *testing.T) {
	cache := NewCache(true, 1*time.Hour)
	
	if !cache.enabled {
		t.Error("Expected cache to be enabled")
	}
	
	if cache.defaultTTL != 1*time.Hour {
		t.Errorf("Expected defaultTTL %v, got %v", 1*time.Hour, cache.defaultTTL)
	}
}

func TestCacheOperations(t *testing.T) {
	cache := NewCache(true, 1*time.Hour)
	
	key := "test-key"
	value := "test-value"
	
	cache.Set(key, value, 1*time.Second)
	
	result, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached value")
	}
	
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}
	
	time.Sleep(2 * time.Second)
	_, found = cache.Get(key)
	if found {
		t.Error("Expected cached value to be expired")
	}
}

func TestCacheDisabled(t *testing.T) {
	cache := NewCache(false, 1*time.Hour)
	
	key := "test-key"
	value := "test-value"
	
	cache.Set(key, value, 1*time.Second)
	
	_, found := cache.Get(key)
	if found {
		t.Error("Expected not to find value when cache is disabled")
	}
}

func TestSonarCloudError(t *testing.T) {
	err := &SonarCloudError{
		StatusCode:   401,
		UserMessage:  "Test error",
		TechnicalDetails: "Technical details",
		SuggestedActions: []string{"Action 1", "Action 2"},
		HelpLinks: []string{"https://example.com"},
	}
	
	if err.Error() != "Test error" {
		t.Errorf("Expected 'Test error', got '%s'", err.Error())
	}
	
	formatted := err.Format(false)
	if !contains(formatted, "Test error") {
		t.Error("Expected formatted error to contain user message")
	}
	
	if !contains(formatted, "Action 1") {
		t.Error("Expected formatted error to contain suggested actions")
	}
	
	if contains(formatted, "Technical details") {
		t.Error("Expected formatted error not to contain technical details when verbose is false")
	}
	
	formattedVerbose := err.Format(true)
	if !contains(formattedVerbose, "Technical details") {
		t.Error("Expected verbose formatted error to contain technical details")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) > len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
