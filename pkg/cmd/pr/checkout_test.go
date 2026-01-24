package pr

import (
	"testing"
)

func TestCheckoutCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid number",
			input:    "123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "valid number with hash",
			input:    "#123",
			expected: 123,
			wantErr:  false,
		},
		{
			name:     "zero is invalid",
			input:    "0",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "negative is invalid",
			input:    "-1",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "non-numeric is invalid",
			input:    "abc",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty string is invalid",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "just hash is invalid",
			input:    "#",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePRID(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePRID(%q) expected error but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParsePRID(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParsePRID(%q) = %d, expected %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestIsSameBranch(t *testing.T) {

	t.Skip("Skipping nil repository test - requires valid git repository for testing")
}
