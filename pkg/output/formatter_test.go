package output

import (
	"bytes"
	"testing"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		opts   *FormatterOptions
		want   string // type name for verification
	}{
		{
			name:   "table formatter",
			format: FormatTable,
			opts:   &FormatterOptions{Writer: &bytes.Buffer{}},
			want:   "*output.TableFormatter",
		},
		{
			name:   "json formatter",
			format: FormatJSON,
			opts:   &FormatterOptions{Writer: &bytes.Buffer{}},
			want:   "*output.JSONFormatter",
		},
		{
			name:   "yaml formatter",
			format: FormatYAML,
			opts:   &FormatterOptions{Writer: &bytes.Buffer{}},
			want:   "*output.YAMLFormatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := NewFormatter(tt.format, tt.opts)
			if err != nil {
				t.Errorf("NewFormatter() error = %v", err)
				return
			}
			if formatter == nil {
				t.Error("NewFormatter() returned nil formatter")
			}
		})
	}
}

func TestNewFormatterInvalidFormat(t *testing.T) {
	_, err := NewFormatter("invalid", nil)
	if err == nil {
		t.Error("NewFormatter() should return error for invalid format")
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"table", false},
		{"json", false},
		{"yaml", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSupportedFormats(t *testing.T) {
	formats := GetSupportedFormats()
	expected := []string{"table", "json", "yaml"}

	if len(formats) != len(expected) {
		t.Errorf("GetSupportedFormats() returned %d formats, expected %d", len(formats), len(expected))
	}

	for i, format := range formats {
		if format != expected[i] {
			t.Errorf("GetSupportedFormats()[%d] = %s, expected %s", i, format, expected[i])
		}
	}
}

func TestBaseFormatter(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &FormatterOptions{
		Writer:  buf,
		NoColor: true,
	}

	base := NewBaseFormatter(opts)

	// Test SetWriter
	newBuf := &bytes.Buffer{}
	base.SetWriter(newBuf)
	base.WriteString("test")

	if newBuf.String() != "test" {
		t.Errorf("SetWriter() failed, expected 'test', got %s", newBuf.String())
	}

	// Test SetNoColor
	base.SetNoColor(false)
	if base.noColor != false {
		t.Error("SetNoColor(false) failed")
	}

	base.SetNoColor(true)
	if base.noColor != true {
		t.Error("SetNoColor(true) failed")
	}
}
