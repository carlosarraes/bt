package output

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableFormatter formats data as a table using lipgloss
type TableFormatter struct {
	*BaseFormatter
	style TableStyle
}

// TableStyle defines the styling for tables
type TableStyle struct {
	Header    lipgloss.Style
	Row       lipgloss.Style
	AltRow    lipgloss.Style
	Border    lipgloss.Style
	Separator string
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(opts *FormatterOptions) *TableFormatter {
	tf := &TableFormatter{
		BaseFormatter: NewBaseFormatter(opts),
	}
	tf.initializeStyle()
	return tf
}

// initializeStyle sets up the default table styling
func (t *TableFormatter) initializeStyle() {
	if t.ShouldUseColor() {
		t.style = TableStyle{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1),
			Row: lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Padding(0, 1),
			AltRow: lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("235")).
				Padding(0, 1),
			Border: lipgloss.NewStyle().
				Foreground(lipgloss.Color("238")),
			Separator: "│",
		}
	} else {
		// No color styles
		t.style = TableStyle{
			Header: lipgloss.NewStyle().
				Bold(false).
				Padding(0, 1),
			Row: lipgloss.NewStyle().
				Padding(0, 1),
			AltRow: lipgloss.NewStyle().
				Padding(0, 1),
			Border:    lipgloss.NewStyle(),
			Separator: "|",
		}
	}
}

// Format formats the data as a table
func (t *TableFormatter) Format(data interface{}) error {
	if data == nil {
		return nil
	}

	// Handle different data types
	switch v := data.(type) {
	case []map[string]interface{}:
		return t.formatMapSlice(v)
	case []interface{}:
		return t.formatInterfaceSlice(v)
	case map[string]interface{}:
		return t.formatSingleMap(v)
	default:
		return t.formatStruct(data)
	}
}

// formatMapSlice formats a slice of maps as a table
func (t *TableFormatter) formatMapSlice(data []map[string]interface{}) error {
	if len(data) == 0 {
		t.WriteString("No data to display\n")
		return nil
	}

	// Extract headers from the first map
	headers := make([]string, 0)
	for key := range data[0] {
		headers = append(headers, key)
	}

	// Calculate column widths
	colWidths := t.calculateColumnWidths(headers, data)

	// Render header
	t.renderHeader(headers, colWidths)

	// Render rows
	for i, row := range data {
		t.renderRow(headers, row, colWidths, i%2 == 1)
	}

	return nil
}

// formatInterfaceSlice formats a slice of interfaces
func (t *TableFormatter) formatInterfaceSlice(data []interface{}) error {
	if len(data) == 0 {
		t.WriteString("No data to display\n")
		return nil
	}

	// Convert to map slice if possible
	mapSlice := make([]map[string]interface{}, 0, len(data))
	for _, item := range data {
		if m, ok := t.toMap(item); ok {
			mapSlice = append(mapSlice, m)
		}
	}

	if len(mapSlice) > 0 {
		return t.formatMapSlice(mapSlice)
	}

	// Fallback to simple list
	for _, item := range data {
		t.WriteString(fmt.Sprintf("%v\n", item))
	}

	return nil
}

// formatSingleMap formats a single map as a two-column table
func (t *TableFormatter) formatSingleMap(data map[string]interface{}) error {
	if len(data) == 0 {
		t.WriteString("No data to display\n")
		return nil
	}

	// Calculate column widths
	maxKeyWidth := 0
	maxValueWidth := 0
	
	for key, value := range data {
		if len(key) > maxKeyWidth {
			maxKeyWidth = len(key)
		}
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > maxValueWidth {
			maxValueWidth = len(valueStr)
		}
	}

	// Render header
	headers := []string{"Key", "Value"}
	colWidths := []int{maxKeyWidth, maxValueWidth}
	t.renderHeader(headers, colWidths)

	// Render rows
	i := 0
	for key, value := range data {
		row := map[string]interface{}{
			"Key":   key,
			"Value": value,
		}
		t.renderRow(headers, row, colWidths, i%2 == 1)
		i++
	}

	return nil
}

// formatStruct formats a struct using reflection
func (t *TableFormatter) formatStruct(data interface{}) error {
	if m, ok := t.toMap(data); ok {
		return t.formatSingleMap(m)
	}

	// Fallback to string representation
	t.WriteString(fmt.Sprintf("%+v\n", data))
	return nil
}

// toMap converts a struct to a map using reflection
func (t *TableFormatter) toMap(data interface{}) (map[string]interface{}, bool) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, false
	}

	result := make(map[string]interface{})
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Use json tag if available, otherwise use field name
		name := fieldType.Name
		if tag := fieldType.Tag.Get("json"); tag != "" && tag != "-" {
			if comma := strings.Index(tag, ","); comma != -1 {
				name = tag[:comma]
			} else {
				name = tag
			}
		}

		result[name] = field.Interface()
	}

	return result, true
}

// calculateColumnWidths calculates the optimal width for each column
func (t *TableFormatter) calculateColumnWidths(headers []string, data []map[string]interface{}) []int {
	widths := make([]int, len(headers))

	// Initialize with header widths
	for i, header := range headers {
		widths[i] = len(header)
	}

	// Check data widths
	for _, row := range data {
		for i, header := range headers {
			if value, exists := row[header]; exists {
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > widths[i] {
					widths[i] = len(valueStr)
				}
			}
		}
	}

	return widths
}

// renderHeader renders the table header
func (t *TableFormatter) renderHeader(headers []string, colWidths []int) {
	row := make([]string, len(headers))
	for i, header := range headers {
		row[i] = t.style.Header.Width(colWidths[i]).Render(header)
	}
	t.WriteString(strings.Join(row, t.style.Separator) + "\n")

	// Render separator line
	if t.ShouldUseColor() {
		separatorRow := make([]string, len(headers))
		for i, width := range colWidths {
			separatorRow[i] = t.style.Border.Render(strings.Repeat("─", width+2))
		}
		t.WriteString(strings.Join(separatorRow, "┼") + "\n")
	}
}

// renderRow renders a single table row
func (t *TableFormatter) renderRow(headers []string, rowData map[string]interface{}, colWidths []int, isAltRow bool) {
	row := make([]string, len(headers))
	style := t.style.Row
	if isAltRow {
		style = t.style.AltRow
	}

	for i, header := range headers {
		value := ""
		if val, exists := rowData[header]; exists {
			value = fmt.Sprintf("%v", val)
		}
		row[i] = style.Width(colWidths[i]).Render(value)
	}

	t.WriteString(strings.Join(row, t.style.Separator) + "\n")
}

// SetNoColor updates the styling based on color preference
func (t *TableFormatter) SetNoColor(noColor bool) {
	t.BaseFormatter.SetNoColor(noColor)
	t.initializeStyle()
}