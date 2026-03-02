package shared

import (
	"fmt"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"", 10, ""},
		{"short", 10, "short"},
		{"exact len!", 10, "exact len!"},
		{"one char over", 12, "one char ..."},
		{"way too long string here", 10, "way too..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abcd"},
		{"hello", 0, "hello"},
		{"hello", -1, "hello"},
		{"hello", 1, "hello"},
		{"hello", 2, "hello"},
		{"hello", 4, "h..."},
		{"hello world this is long", 15, "hello world ..."},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q_max%d", tt.s, tt.maxLen), func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
