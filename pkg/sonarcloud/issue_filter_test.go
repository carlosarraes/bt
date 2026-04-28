package sonarcloud

import "testing"

func TestIsActionable(t *testing.T) {
	cases := []struct {
		name string
		in   ProcessedIssue
		want bool
	}{
		{"open new code smell", ProcessedIssue{IsNew: true, Status: "OPEN"}, true},
		{"confirmed new bug", ProcessedIssue{IsNew: true, Status: "CONFIRMED"}, true},
		{"reopened new", ProcessedIssue{IsNew: true, Status: "REOPENED"}, true},
		{"accepted false positive", ProcessedIssue{IsNew: true, Status: "RESOLVED", Resolution: "FALSE_POSITIVE"}, false},
		{"wontfix", ProcessedIssue{IsNew: true, Status: "RESOLVED", Resolution: "WONT_FIX"}, false},
		{"closed/fixed", ProcessedIssue{IsNew: true, Status: "CLOSED", Resolution: "FIXED"}, false},
		{"removed", ProcessedIssue{IsNew: true, Status: "CLOSED", Resolution: "REMOVED"}, false},
		{"old (not new)", ProcessedIssue{IsNew: false, Status: "OPEN"}, false},
		{"open but stray resolution", ProcessedIssue{IsNew: true, Status: "OPEN", Resolution: "FALSE_POSITIVE"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsActionable(tc.in); got != tc.want {
				t.Errorf("IsActionable(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
