package shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

// Pure-function regression: a line is a branch gap iff it has at least one
// condition AND not all conditions are covered. Anything else is a line gap.
func TestSplitCoverageGaps_PartitionsLinesAndBranches(t *testing.T) {
	lines := []sonarcloud.UncoveredLine{
		{Line: 10, Code: "untouched"},                                   // line gap (no conditions)
		{Line: 20, Code: "a ?? b", Conditions: 4, CoveredConditions: 3}, // branch gap
		{Line: 30, Code: "a || b", Conditions: 7, CoveredConditions: 6}, // branch gap
		{Line: 40, Code: "fully", Conditions: 2, CoveredConditions: 2},  // not a gap, but split routes to lineGaps (caller filters)
		{Line: 50, Code: "no cond branch", Conditions: 0},               // line gap
	}

	lineGaps, branchGaps := SplitCoverageGaps(lines)

	branchLineNumbers := []int{}
	for _, b := range branchGaps {
		branchLineNumbers = append(branchLineNumbers, b.Line)
	}
	assert.ElementsMatch(t, []int{20, 30}, branchLineNumbers,
		"only lines with covered<total go to branch section")

	lineGapNumbers := []int{}
	for _, l := range lineGaps {
		lineGapNumbers = append(lineGapNumbers, l.Line)
	}
	assert.ElementsMatch(t, []int{10, 40, 50}, lineGapNumbers,
		"lines without uncovered conditions go to line section")
}

// Display regression: the report must surface branch coverage gaps in their
// own section with a "(M/N branches)" annotation. Without this, users chasing
// a quality gate failure see only line-level uncovered code and miss the
// real cause when it's a partial-branch issue (e.g. ?? or || operators).
func TestDisplayUncoveredLinesDetails_SeparatesLineAndBranchSections(t *testing.T) {
	f := &ReportFormatter{ShowAllLines: true, LinesPerFile: 100}

	details := []sonarcloud.CoverageDetails{{
		FilePath:        "src/foo.ts",
		FileName:        "foo.ts",
		CoveragePercent: 75,
		UncoveredLines: []sonarcloud.UncoveredLine{
			{File: "src/foo.ts", Line: 10, Code: "this.flowStepValue += 1;", IsNew: true},
			{File: "src/foo.ts", Line: 807, Code: "const x = a ?? b;", IsNew: true, Conditions: 4, CoveredConditions: 3},
		},
	}}

	out, _ := captureStdoutStderr(t, func() {
		f.DisplayUncoveredLinesDetails(details, sonarcloud.FilterOptions{})
	})

	assert.Contains(t, out, "Uncovered Lines:", "line section header missing")
	assert.Contains(t, out, "Partial Branch Coverage:", "branch section header missing")
	assert.Contains(t, out, "(3/4 branches)", "branch count annotation missing")

	branchIdx := strings.Index(out, "Partial Branch Coverage:")
	lineIdx := strings.Index(out, "Uncovered Lines:")
	require := assert.New(t)
	require.True(branchIdx > 0 && lineIdx > 0, "both sections must render")

	branchSection := out[branchIdx:]
	assert.NotContains(t, branchSection, "this.flowStepValue += 1;",
		"line-only entries must not appear in the branch section")
	assert.Contains(t, branchSection, "Line 807",
		"branch entry must appear in the branch section")
}

// Display regression: lines with no uncovered conditions don't render the
// "(M/N branches)" suffix.
func TestDisplayUncoveredLinesDetails_OmitsBranchCountForPureLineGaps(t *testing.T) {
	f := &ReportFormatter{ShowAllLines: true, LinesPerFile: 100}

	details := []sonarcloud.CoverageDetails{{
		FilePath:        "src/bar.ts",
		FileName:        "bar.ts",
		CoveragePercent: 50,
		UncoveredLines: []sonarcloud.UncoveredLine{
			{File: "src/bar.ts", Line: 5, Code: "untouched", IsNew: true},
		},
	}}

	out, _ := captureStdoutStderr(t, func() {
		f.DisplayUncoveredLinesDetails(details, sonarcloud.FilterOptions{})
	})

	assert.NotContains(t, out, "branches)",
		"pure line gaps must not render the branch annotation")
	assert.NotContains(t, out, "Partial Branch Coverage:",
		"empty branch section must not render its header")
}
