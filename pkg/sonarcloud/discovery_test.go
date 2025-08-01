package sonarcloud

import (
	"testing"
)

func TestSanitizeProjectKey(t *testing.T) {
	discovery := &ProjectKeyDiscovery{}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"workspace_repo", "workspace_repo"},
		{"workspace-repo", "workspace-repo"},
		{"workspace.repo", "workspace.repo"},
		{"workspace@repo", "workspace_repo"},
		{"workspace#repo", "workspace_repo"},
		{"workspace__repo", "workspace_repo"},
		{"_workspace_repo_", "workspace_repo"},
		{"", ""},
		{"a", "a"},
	}
	
	for _, test := range tests {
		result := discovery.sanitizeProjectKey(test.input)
		if result != test.expected {
			t.Errorf("sanitizeProjectKey(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsValidProjectKey(t *testing.T) {
	discovery := &ProjectKeyDiscovery{}
	
	tests := []struct {
		key   string
		valid bool
	}{
		{"workspace_repo", true},
		{"workspace-repo", true},
		{"workspace.repo", true},
		{"workspace123", true},
		{"", false},
		{"a", true},
		{"workspace@repo", false},
		{"workspace#repo", false},
		{"very_long_" + string(make([]byte, 400)), false},
	}
	
	for _, test := range tests {
		result := discovery.isValidProjectKey(test.key)
		if result != test.valid {
			t.Errorf("isValidProjectKey(%q) = %v, expected %v", test.key, result, test.valid)
		}
	}
}

func TestExtractProjectKeyFromURL(t *testing.T) {
	discovery := &ProjectKeyDiscovery{}
	
	tests := []struct {
		url      string
		expected string
	}{
		{"https://sonarcloud.io/dashboard?id=truora_api&pullRequest=949", "truora_api"},
		{"https://sonarcloud.io/project/overview?id=project_key", "project_key"},
		{"https://sonarcloud.io/summary/overall?id=another_key", "another_key"},
		{"https://sonarcloud.io/dashboard/project_name", "project_name"},
		{"https://example.com/no-match", ""},
		{"", ""},
	}
	
	for _, test := range tests {
		result := discovery.extractProjectKeyFromURL(test.url)
		if result != test.expected {
			t.Errorf("extractProjectKeyFromURL(%q) = %q, expected %q", test.url, result, test.expected)
		}
	}
}

func TestGetProjectKeyStrategies(t *testing.T) {
	discovery := &ProjectKeyDiscovery{}
	
	strategies := discovery.GetProjectKeyStrategies()
	if len(strategies) != 5 {
		t.Errorf("Expected 5 strategies, got %d", len(strategies))
	}
	
	expectedStrategies := []string{
		"Bitbucket Reports API",
		"Environment Variable", 
		"Configuration File",
		"Git Repository",
		"Heuristic Naming",
	}
	
	for i, expected := range expectedStrategies {
		if !containsSubstring(strategies[i], expected) {
			t.Errorf("Strategy %d should contain %q, got %q", i, expected, strategies[i])
		}
	}
}
