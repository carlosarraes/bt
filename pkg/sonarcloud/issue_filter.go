package sonarcloud

// IsActionable reports whether an issue requires the PR author to act on it:
// new-code, no resolution, and status in OPEN/CONFIRMED/REOPENED.
func IsActionable(issue ProcessedIssue) bool {
	if !issue.IsNew {
		return false
	}
	if issue.Resolution != "" {
		return false
	}
	switch issue.Status {
	case "OPEN", "CONFIRMED", "REOPENED":
		return true
	}
	return false
}
