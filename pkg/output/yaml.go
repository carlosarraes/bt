package output

import (
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats data as YAML
type YAMLFormatter struct {
	*BaseFormatter
	indent int
}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter(opts *FormatterOptions) *YAMLFormatter {
	return &YAMLFormatter{
		BaseFormatter: NewBaseFormatter(opts),
		indent:        2, // 2 spaces for indentation
	}
}

// Format formats the data as YAML
func (y *YAMLFormatter) Format(data interface{}) error {
	if data == nil {
		_, err := y.WriteString("null\n")
		return err
	}

	// Create encoder with custom indentation
	encoder := yaml.NewEncoder(y.writer)
	encoder.SetIndent(y.indent)

	// Encode the data
	err := encoder.Encode(data)
	if err != nil {
		return err
	}

	// Close the encoder to ensure all data is written
	return encoder.Close()
}

// SetIndent sets the indentation level (number of spaces)
func (y *YAMLFormatter) SetIndent(indent int) {
	if indent < 1 {
		indent = 2
	}
	y.indent = indent
}