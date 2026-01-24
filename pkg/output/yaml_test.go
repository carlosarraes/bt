package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestYAMLFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: "null\n",
		},
		{
			name: "simple map",
			data: map[string]interface{}{
				"name": "test",
				"id":   123,
			},
			expected: "id: 123\nname: test\n",
		},
		{
			name: "slice of maps",
			data: []map[string]interface{}{
				{"id": "1", "status": "success"},
				{"id": "2", "status": "failed"},
			},
			expected: "- id: \"1\"\n  status: success\n- id: \"2\"\n  status: failed\n",
		},
		{
			name:     "simple slice",
			data:     []string{"item1", "item2", "item3"},
			expected: "- item1\n- item2\n- item3\n",
		},
		{
			name: "nested structure",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
				"active": true,
			},
			expected: "active: true\nuser:\n  age: 30\n  name: John\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			formatter := NewYAMLFormatter(&FormatterOptions{
				Writer: buf,
			})

			err := formatter.Format(tt.data)
			if err != nil {
				t.Errorf("YAMLFormatter.Format() error = %v", err)
				return
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("YAMLFormatter.Format() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestYAMLFormatter_SetIndent(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewYAMLFormatter(&FormatterOptions{
		Writer: buf,
	})

	// Set custom indent (4 spaces)
	formatter.SetIndent(4)

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John",
		},
	}

	err := formatter.Format(data)
	if err != nil {
		t.Errorf("YAMLFormatter.Format() error = %v", err)
		return
	}

	result := buf.String()
	expected := "user:\n    name: John\n" // 4 spaces indentation

	if result != expected {
		t.Errorf("YAMLFormatter with 4-space indent = %q, expected %q", result, expected)
	}
}

func TestYAMLFormatter_SetIndentInvalid(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewYAMLFormatter(&FormatterOptions{
		Writer: buf,
	})

	// Set invalid indent (should default to 2)
	formatter.SetIndent(0)

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John",
		},
	}

	err := formatter.Format(data)
	if err != nil {
		t.Errorf("YAMLFormatter.Format() error = %v", err)
		return
	}

	result := buf.String()
	expected := "user:\n  name: John\n" // Should default to 2 spaces

	if result != expected {
		t.Errorf("YAMLFormatter with invalid indent should default to 2 spaces, got %q, expected %q", result, expected)
	}
}

func TestYAMLFormatter_ComplexData(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewYAMLFormatter(&FormatterOptions{
		Writer: buf,
	})

	// Test with more complex data structure
	data := map[string]interface{}{
		"pipeline": map[string]interface{}{
			"id":       "12345",
			"status":   "success",
			"duration": 150,
			"steps": []map[string]interface{}{
				{"name": "build", "status": "success", "duration": 60},
				{"name": "test", "status": "success", "duration": 90},
			},
		},
		"metadata": map[string]interface{}{
			"branch":  "main",
			"commit":  "abc123",
			"trigger": "push",
		},
	}

	err := formatter.Format(data)
	if err != nil {
		t.Errorf("YAMLFormatter.Format() error = %v", err)
		return
	}

	result := buf.String()

	// Check that the output contains expected structure
	expectedParts := []string{
		"pipeline:",
		"id: \"12345\"",
		"status: success",
		"duration: 150",
		"steps:",
		"- duration: 60",
		"  name: build",
		"  status: success",
		"metadata:",
		"branch: main",
		"commit: abc123",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("YAMLFormatter output missing expected part %q, got:\n%s", part, result)
		}
	}
}
