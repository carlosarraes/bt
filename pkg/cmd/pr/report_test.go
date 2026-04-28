package pr

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

func TestFilterActionable_KeepsOnlyActionableIssues(t *testing.T) {
	in := &sonarcloud.IssuesData{
		Available: true,
		Issues: []sonarcloud.ProcessedIssue{
			{Key: "open", IsNew: true, Status: "OPEN"},
			{Key: "fp", IsNew: true, Status: "RESOLVED", Resolution: "FALSE_POSITIVE"},
			{Key: "old", IsNew: false, Status: "OPEN"},
			{Key: "confirmed", IsNew: true, Status: "CONFIRMED"},
			{Key: "wontfix", IsNew: true, Status: "RESOLVED", Resolution: "WONT_FIX"},
		},
	}
	out := filterActionable(in)
	keys := make([]string, 0, len(out.Issues))
	for _, iss := range out.Issues {
		keys = append(keys, iss.Key)
	}
	assert.ElementsMatch(t, []string{"open", "confirmed"}, keys)
	assert.Equal(t, 2, out.TotalIssues)
}

func TestFilterActionable_NilInput(t *testing.T) {
	out := filterActionable(nil)
	assert.NotNil(t, out)
	assert.Empty(t, out.Issues)
	assert.True(t, out.Available)
}
