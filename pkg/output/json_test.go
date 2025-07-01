package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONFormatter_Format(t *testing.T) {
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
			expected: `{
  "id": 123,
  "name": "test"
}
`,
		},
		{
			name: "slice of maps",
			data: []map[string]interface{}{
				{"id": "1", "status": "success"},
				{"id": "2", "status": "failed"},
			},
			expected: `[
  {
    "id": "1",
    "status": "success"
  },
  {
    "id": "2",
    "status": "failed"
  }
]
`,
		},
		{
			name: "simple slice",
			data: []string{"item1", "item2", "item3"},
			expected: `[
  "item1",
  "item2",
  "item3"
]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			formatter := NewJSONFormatter(&FormatterOptions{
				Writer: buf,
			})

			err := formatter.Format(tt.data)
			if err != nil {
				t.Errorf("JSONFormatter.Format() error = %v", err)
				return
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("JSONFormatter.Format() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestJSONFormatter_FormatCompact(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewJSONFormatter(&FormatterOptions{
		Writer: buf,
	})

	data := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	err := formatter.FormatCompact(data)
	if err != nil {
		t.Errorf("JSONFormatter.FormatCompact() error = %v", err)
		return
	}

	result := buf.String()
	// Should be single line JSON
	if strings.Contains(result, "\n  ") {
		t.Errorf("JSONFormatter.FormatCompact() should not contain indentation, got %q", result)
	}

	if !strings.HasSuffix(result, "\n") {
		t.Errorf("JSONFormatter.FormatCompact() should end with newline, got %q", result)
	}
}

func TestJSONFormatter_SetIndent(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewJSONFormatter(&FormatterOptions{
		Writer: buf,
	})

	// Set custom indent
	formatter.SetIndent("    ") // 4 spaces

	data := map[string]interface{}{
		"name": "test",
	}

	err := formatter.Format(data)
	if err != nil {
		t.Errorf("JSONFormatter.Format() error = %v", err)
		return
	}

	result := buf.String()
	expected := `{
    "name": "test"
}
`
	if result != expected {
		t.Errorf("JSONFormatter with custom indent = %q, expected %q", result, expected)
	}
}

func TestJSONFormatter_SetCompact(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := NewJSONFormatter(&FormatterOptions{
		Writer: buf,
	})

	// Set compact mode
	formatter.SetCompact(true)

	data := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	err := formatter.Format(data)
	if err != nil {
		t.Errorf("JSONFormatter.Format() error = %v", err)
		return
	}

	result := buf.String()
	// Should not contain pretty-printing spaces
	if strings.Contains(result, "  ") {
		t.Errorf("JSONFormatter in compact mode should not contain indentation, got %q", result)
	}

	// Test turning compact mode off
	buf.Reset()
	formatter.SetCompact(false)

	err = formatter.Format(data)
	if err != nil {
		t.Errorf("JSONFormatter.Format() error = %v", err)
		return
	}

	result = buf.String()
	// Should contain pretty-printing spaces
	if !strings.Contains(result, "  ") {
		t.Errorf("JSONFormatter with compact=false should contain indentation, got %q", result)
	}
}