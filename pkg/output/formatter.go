package output

import (
	"fmt"
	"io"
	"os"
)

// Formatter defines the interface for output formatting
type Formatter interface {
	// Format formats the given data and writes it to the writer
	Format(data interface{}) error
	// SetWriter sets the output writer
	SetWriter(w io.Writer)
	// SetNoColor disables color output
	SetNoColor(noColor bool)
}

// Format represents the output format type
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// FormatterOptions holds configuration for formatters
type FormatterOptions struct {
	NoColor bool
	Writer  io.Writer
}

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format Format, opts *FormatterOptions) (Formatter, error) {
	if opts == nil {
		opts = &FormatterOptions{
			Writer: os.Stdout,
		}
	}

	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	switch format {
	case FormatTable:
		return NewTableFormatter(opts), nil
	case FormatJSON:
		return NewJSONFormatter(opts), nil
	case FormatYAML:
		return NewYAMLFormatter(opts), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}

// ValidateFormat checks if the given format is valid
func ValidateFormat(format string) error {
	switch Format(format) {
	case FormatTable, FormatJSON, FormatYAML:
		return nil
	default:
		return fmt.Errorf("invalid output format: %s (supported: table, json, yaml)", format)
	}
}

// GetSupportedFormats returns a list of supported output formats
func GetSupportedFormats() []string {
	return []string{
		string(FormatTable),
		string(FormatJSON),
		string(FormatYAML),
	}
}

// BaseFormatter provides common functionality for all formatters
type BaseFormatter struct {
	writer  io.Writer
	noColor bool
}

// NewBaseFormatter creates a new base formatter
func NewBaseFormatter(opts *FormatterOptions) *BaseFormatter {
	return &BaseFormatter{
		writer:  opts.Writer,
		noColor: opts.NoColor,
	}
}

// SetWriter sets the output writer
func (b *BaseFormatter) SetWriter(w io.Writer) {
	b.writer = w
}

// SetNoColor disables color output
func (b *BaseFormatter) SetNoColor(noColor bool) {
	b.noColor = noColor
}

// Write writes data to the configured writer
func (b *BaseFormatter) Write(data []byte) (int, error) {
	return b.writer.Write(data)
}

// WriteString writes a string to the configured writer
func (b *BaseFormatter) WriteString(s string) (int, error) {
	return b.writer.Write([]byte(s))
}

// ShouldUseColor returns true if color output should be used
func (b *BaseFormatter) ShouldUseColor() bool {
	if b.noColor {
		return false
	}
	
	// Check if we're writing to a terminal
	if f, ok := b.writer.(*os.File); ok {
		return isTerminal(f)
	}
	
	return false
}

// isTerminal checks if the given file is a terminal
func isTerminal(f *os.File) bool {
	// Simple check for terminal - in a real implementation you might want to use
	// a more sophisticated method like checking file info or using a library
	return f == os.Stdout || f == os.Stderr
}