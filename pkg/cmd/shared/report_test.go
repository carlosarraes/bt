package shared

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

// captureStdoutStderr runs fn while capturing both streams. Returns (stdout, stderr).
func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	done := make(chan struct{})
	var outBuf, errBuf bytes.Buffer
	go func() {
		_, _ = io.Copy(&outBuf, rOut)
		_, _ = io.Copy(&errBuf, rErr)
		close(done)
	}()

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	<-done
	return outBuf.String(), errBuf.String()
}

func intPtr(n int) *int { return &n }

func TestFormatActionableIssuesSection_UnassignedNewCodeSmell(t *testing.T) {
	f := &ReportFormatter{}
	actionable := &sonarcloud.IssuesData{
		Available: true,
		Issues: []sonarcloud.ProcessedIssue{{
			Key: "K1", Type: "CODE_SMELL", Severity: "MAJOR", Status: "OPEN",
			Rule: "python:S1192", File: "pagamentos/models.py", Line: intPtr(570),
			Message: "Define a constant instead of duplicating this literal.",
			IsNew:   true,
		}},
	}
	out, _ := captureStdoutStderr(t, func() {
		n := f.FormatActionableIssuesSection(actionable, nil)
		require.Equal(t, 1, n)
	})
	assert.Contains(t, out, "Actionable New Issues: 1")
	assert.Contains(t, out, "[MAJOR] Code Smell pagamentos/models.py:570 unassigned")
	assert.Contains(t, out, "Define a constant instead of duplicating this literal.")
	assert.Contains(t, out, "Rule: python:S1192")
	assert.Contains(t, out, "In PR diff: unknown")
}

func TestFormatActionableIssuesSection_AssignedInDiff(t *testing.T) {
	f := &ReportFormatter{}
	actionable := &sonarcloud.IssuesData{
		Available: true,
		Issues: []sonarcloud.ProcessedIssue{{
			Key: "K2", Type: "BUG", Severity: "CRITICAL", Status: "OPEN",
			Rule: "go:S1", File: "internal/svc.go", Line: intPtr(42),
			Message:  "Nil dereference possible.",
			Assignee: "alice", IsNew: true,
		}},
	}
	diffLines := map[string]map[int]bool{"internal/svc.go": {42: true}}
	out, _ := captureStdoutStderr(t, func() {
		f.FormatActionableIssuesSection(actionable, diffLines)
	})
	assert.Contains(t, out, "[CRITICAL] Bug internal/svc.go:42 alice")
	assert.Contains(t, out, "In PR diff: yes")
}

func TestFormatActionableIssuesSection_NoIssues(t *testing.T) {
	f := &ReportFormatter{}
	out, _ := captureStdoutStderr(t, func() {
		n := f.FormatActionableIssuesSection(nil, nil)
		require.Equal(t, 0, n)
	})
	assert.Contains(t, out, "Actionable New Issues: 0")
	assert.Contains(t, out, "Nothing for the PR author to fix")
}

func TestFormatAcceptedSummary_HiddenByDefault(t *testing.T) {
	f := &ReportFormatter{}
	accepted := &sonarcloud.IssuesData{
		Issues: []sonarcloud.ProcessedIssue{
			{Key: "A1", Resolution: "FALSE_POSITIVE", Severity: "MINOR", Type: "CODE_SMELL", File: "x.py", Line: intPtr(1), Message: "m", Rule: "r"},
			{Key: "A2", Resolution: "WONT_FIX", Severity: "MAJOR", Type: "BUG", File: "y.py", Line: intPtr(2), Message: "m2", Rule: "r2"},
			{Key: "A3", Resolution: "FALSE_POSITIVE", Severity: "MINOR", Type: "CODE_SMELL", File: "z.py", Line: intPtr(3), Message: "m3", Rule: "r3"},
		},
	}
	out, _ := captureStdoutStderr(t, func() {
		f.FormatAcceptedSummary(accepted, 0, false)
	})
	assert.Contains(t, out, "Accepted / Pre-existing Issues: 3")
	assert.Contains(t, out, "Use --all-issues to show details")
	assert.NotContains(t, out, "FALSE_POSITIVE")
	assert.NotContains(t, out, "WONT_FIX")
}

func TestFormatAcceptedSummary_VisibleWithAllIssues(t *testing.T) {
	f := &ReportFormatter{}
	accepted := &sonarcloud.IssuesData{
		Issues: []sonarcloud.ProcessedIssue{
			{Key: "A1", Resolution: "FALSE_POSITIVE", Severity: "MINOR", Type: "CODE_SMELL", File: "x.py", Line: intPtr(1), Message: "msg fp", Rule: "r"},
			{Key: "A2", Resolution: "WONT_FIX", Severity: "MAJOR", Type: "BUG", File: "y.py", Line: intPtr(2), Message: "msg wf", Rule: "r2"},
		},
	}
	out, _ := captureStdoutStderr(t, func() {
		f.FormatAcceptedSummary(accepted, 0, true)
	})
	assert.Contains(t, out, "Accepted / Pre-existing Issues: 2")
	assert.Contains(t, out, "(FALSE_POSITIVE)")
	assert.Contains(t, out, "(WONT_FIX)")
	assert.Contains(t, out, "msg fp")
	assert.NotContains(t, out, "Use --all-issues")
}

func TestFormatAcceptedSummary_QGCountUsedWhenLarger(t *testing.T) {
	f := &ReportFormatter{}
	out, _ := captureStdoutStderr(t, func() {
		f.FormatAcceptedSummary(&sonarcloud.IssuesData{}, 69, false)
	})
	assert.Contains(t, out, "Accepted / Pre-existing Issues: 69")
}

func TestWarnGateMismatch_TriggersWhenGateOver(t *testing.T) {
	metrics := &sonarcloud.MetricsData{NewCodeSmells: 2}
	_, stderr := captureStdoutStderr(t, func() {
		expected, mismatch := WarnGateMismatch(metrics, nil, 0)
		require.Equal(t, 2, expected)
		require.True(t, mismatch)
	})
	assert.Contains(t, stderr, "Quality gate reports 2 new code smells")
	assert.Contains(t, stderr, "Retrying with new-code filters")
}

func TestWarnGateMismatch_QuietWhenAligned(t *testing.T) {
	metrics := &sonarcloud.MetricsData{NewCodeSmells: 2}
	_, stderr := captureStdoutStderr(t, func() {
		_, mismatch := WarnGateMismatch(metrics, nil, 2)
		require.False(t, mismatch)
	})
	assert.Empty(t, strings.TrimSpace(stderr))
}

func TestWarnGateMismatch_PrefersSummaryWhenSet(t *testing.T) {
	summary := &sonarcloud.QualityGateSummary{NewIssues: 5}
	metrics := &sonarcloud.MetricsData{NewBugs: 1}
	_, _ = captureStdoutStderr(t, func() {
		expected, mismatch := WarnGateMismatch(metrics, summary, 0)
		require.Equal(t, 5, expected)
		require.True(t, mismatch)
	})
}

func TestFormatDuplicationsSection_RendersPerLineDetail(t *testing.T) {
	f := &ReportFormatter{LinesPerFile: 5, TruncateLines: 80}
	dup := &sonarcloud.DuplicationData{
		Available:          true,
		OverallDuplication: 4.2,
		NewCodeDuplication: 12.5,
		DuplicatedLines:    20,
		DuplicatedBlocks:   2,
		Files: []sonarcloud.DuplicatedFile{
			{Name: "pkg/api/pullrequests.go", DuplicatedDensity: 12.4, DuplicatedLines: 15, DuplicatedBlocks: 1},
		},
		Details: []sonarcloud.DuplicationDetail{{
			FilePath:          "pkg/api/pullrequests.go",
			FileName:          "pullrequests.go",
			DuplicatedDensity: 12.4,
			Blocks: []sonarcloud.DuplicatedBlock{{
				From: 142, Size: 4,
				TargetFile: "pkg/api/pipelines.go",
				TargetFrom: 88, TargetSize: 4,
				Lines: []sonarcloud.DuplicatedLine{
					{Line: 142, Code: "func (s *Service) buildURL(workspace, repo string) string {"},
					{Line: 143, Code: "    base := s.client.baseURL"},
					{Line: 144, Code: "    return fmt.Sprintf(\"%s/repos/%s/%s\", base, workspace, repo)"},
					{Line: 145, Code: "}", IsNew: true},
				},
			}},
		}},
	}
	filters := sonarcloud.FilterOptions{Limit: 10}

	out, _ := captureStdoutStderr(t, func() {
		f.FormatDuplicationsSection(dup, filters)
	})

	assert.Contains(t, out, "Duplications Summary")
	assert.Contains(t, out, "Duplicated Lines Details")
	assert.Contains(t, out, "pkg/api/pullrequests.go (12.4% duplication)")
	assert.Contains(t, out, "Block 1 (4 lines) → also in pkg/api/pipelines.go lines 88-91")
	assert.Contains(t, out, "Line 142: func (s *Service) buildURL")
	assert.Contains(t, out, "Line 145: }")
	assert.Contains(t, out, "[NEW]")
}

func TestFormatDuplicationsSection_FallsBackWhenSourceMissing(t *testing.T) {
	f := &ReportFormatter{LinesPerFile: 5}
	dup := &sonarcloud.DuplicationData{
		Available: true,
		Details: []sonarcloud.DuplicationDetail{{
			FilePath:          "pkg/api/pullrequests.go",
			DuplicatedDensity: 5.0,
			Blocks: []sonarcloud.DuplicatedBlock{{
				From: 10, Size: 5,
				TargetFile: "pkg/api/pipelines.go",
				TargetFrom: 30, TargetSize: 5,
			}},
		}},
	}
	filters := sonarcloud.FilterOptions{Limit: 10}

	out, _ := captureStdoutStderr(t, func() {
		f.FormatDuplicationsSection(dup, filters)
	})

	assert.Contains(t, out, "Block 1 (5 lines) → also in pkg/api/pipelines.go lines 30-34")
	assert.Contains(t, out, "Lines 10-14 (source unavailable)")
}

func TestFormatDuplicationsSection_TruncatesWithShowAllLinesHint(t *testing.T) {
	f := &ReportFormatter{LinesPerFile: 2}
	lines := []sonarcloud.DuplicatedLine{
		{Line: 1, Code: "a"},
		{Line: 2, Code: "b"},
		{Line: 3, Code: "c"},
		{Line: 4, Code: "d"},
	}
	dup := &sonarcloud.DuplicationData{
		Available: true,
		Details: []sonarcloud.DuplicationDetail{{
			FilePath: "pkg/x.go",
			Blocks: []sonarcloud.DuplicatedBlock{{
				From: 1, Size: 4,
				TargetFile: "pkg/y.go", TargetFrom: 1, TargetSize: 4,
				Lines: lines,
			}},
		}},
	}

	out, _ := captureStdoutStderr(t, func() {
		f.FormatDuplicationsSection(dup, sonarcloud.FilterOptions{Limit: 10})
	})

	assert.Contains(t, out, "Line 1: a")
	assert.Contains(t, out, "Line 2: b")
	assert.NotContains(t, out, "Line 3: c")
	assert.Contains(t, out, "... (2 more) Use --show-all-lines")
}

func TestFormatDuplicationsSection_ShowAllLinesPrintsEverything(t *testing.T) {
	f := &ReportFormatter{LinesPerFile: 2, ShowAllLines: true}
	lines := []sonarcloud.DuplicatedLine{
		{Line: 1, Code: "a"},
		{Line: 2, Code: "b"},
		{Line: 3, Code: "c"},
		{Line: 4, Code: "d"},
	}
	dup := &sonarcloud.DuplicationData{
		Available: true,
		Details: []sonarcloud.DuplicationDetail{{
			FilePath: "pkg/x.go",
			Blocks: []sonarcloud.DuplicatedBlock{{
				From: 1, Size: 4,
				TargetFile: "pkg/y.go", TargetFrom: 1, TargetSize: 4,
				Lines: lines,
			}},
		}},
	}

	out, _ := captureStdoutStderr(t, func() {
		f.FormatDuplicationsSection(dup, sonarcloud.FilterOptions{Limit: 10})
	})

	for _, expected := range []string{"Line 1: a", "Line 2: b", "Line 3: c", "Line 4: d"} {
		assert.Contains(t, out, expected)
	}
	assert.NotContains(t, out, "more) Use --show-all-lines")
}
