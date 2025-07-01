package output

import (
	"encoding/json"
)

// JSONFormatter formats data as JSON
type JSONFormatter struct {
	*BaseFormatter
	indent string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(opts *FormatterOptions) *JSONFormatter {
	return &JSONFormatter{
		BaseFormatter: NewBaseFormatter(opts),
		indent:        "  ", // 2 spaces for indentation
	}
}

// Format formats the data as JSON
func (j *JSONFormatter) Format(data interface{}) error {
	if data == nil {
		_, err := j.WriteString("null\n")
		return err
	}

	// Marshal with indentation for readability
	jsonData, err := json.MarshalIndent(data, "", j.indent)
	if err != nil {
		return err
	}

	// Write JSON data
	_, err = j.Write(jsonData)
	if err != nil {
		return err
	}

	// Add newline for better terminal output
	_, err = j.WriteString("\n")
	return err
}

// SetIndent sets the indentation string (default is "  ")
func (j *JSONFormatter) SetIndent(indent string) {
	j.indent = indent
}

// SetCompact sets whether to use compact JSON output (no indentation)
func (j *JSONFormatter) SetCompact(compact bool) {
	if compact {
		j.indent = ""
	} else {
		j.indent = "  "
	}
}

// FormatCompact formats the data as compact JSON (single line)
func (j *JSONFormatter) FormatCompact(data interface{}) error {
	if data == nil {
		_, err := j.WriteString("null\n")
		return err
	}

	// Marshal without indentation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write JSON data
	_, err = j.Write(jsonData)
	if err != nil {
		return err
	}

	// Add newline for better terminal output
	_, err = j.WriteString("\n")
	return err
}