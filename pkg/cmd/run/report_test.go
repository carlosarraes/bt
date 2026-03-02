package run

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRunSeverityIcon(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"BLOCKER", "🚫"},
		{"HIGH", "🔴"},
		{"CRITICAL", "🔴"},
		{"MEDIUM", "🟠"},
		{"MAJOR", "🟠"},
		{"LOW", "🟡"},
		{"MINOR", "🟡"},
		{"INFO", "🔵"},
		{"UNKNOWN", "⚪"},
		{"", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := shared.SeverityIcon(tt.severity)
			if got != tt.want {
				t.Errorf("SeverityIcon(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestRunRatingNumberToLetter(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"1.0", "A"},
		{"1", "A"},
		{"2.0", "B"},
		{"2", "B"},
		{"3.0", "C"},
		{"3", "C"},
		{"4.0", "D"},
		{"4", "D"},
		{"5.0", "E"},
		{"5", "E"},
		{"unknown", "unknown"},
		{"", ""},
		{"6.0", "6.0"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("value_%s", tt.value), func(t *testing.T) {
			got := shared.RatingNumberToLetter(tt.value)
			if got != tt.want {
				t.Errorf("RatingNumberToLetter(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestRunRatingFromMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics *sonarcloud.MetricsData
		key     string
		want    string
	}{
		{"nil metrics", nil, "sqale_rating", "?"},
		{"missing key", &sonarcloud.MetricsData{Ratings: map[string]string{}}, "sqale_rating", "?"},
		{"rating A", &sonarcloud.MetricsData{Ratings: map[string]string{"sqale_rating": "1.0"}}, "sqale_rating", "A"},
		{"rating C", &sonarcloud.MetricsData{Ratings: map[string]string{"sqale_rating": "3"}}, "sqale_rating", "C"},
		{"nil ratings map", &sonarcloud.MetricsData{}, "sqale_rating", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.RatingFromMetrics(tt.metrics, tt.key)
			if got != tt.want {
				t.Errorf("RatingFromMetrics(%v, %q) = %q, want %q", tt.metrics, tt.key, got, tt.want)
			}
		})
	}
}

func TestRunFormatDebtMinutes(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{0, "0min"},
		{5, "5min"},
		{30, "30min"},
		{60, "1h"},
		{90, "1h30min"},
		{120, "2h"},
		{480, "1d"},
		{600, "1d2h"},
		{510, "1d"},   // 8h30min = 1d (30min lost — known limitation)
		{960, "2d"},
		{1440, "3d"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_minutes", tt.minutes), func(t *testing.T) {
			got := shared.FormatDebtMinutes(tt.minutes)
			if got != tt.want {
				t.Errorf("FormatDebtMinutes(%d) = %q, want %q", tt.minutes, got, tt.want)
			}
		})
	}
}

func TestRunFormatImpact(t *testing.T) {
	tests := []struct {
		name    string
		impacts []sonarcloud.IssueImpact
		want    string
	}{
		{
			"empty impacts",
			nil,
			"-",
		},
		{
			"single high security",
			[]sonarcloud.IssueImpact{{SoftwareQuality: "SECURITY", Severity: "HIGH"}},
			"🔴 Security HIGH",
		},
		{
			"single medium maintainability",
			[]sonarcloud.IssueImpact{{SoftwareQuality: "MAINTAINABILITY", Severity: "MEDIUM"}},
			"🟠 Maintain MEDIUM",
		},
		{
			"empty severity",
			[]sonarcloud.IssueImpact{{SoftwareQuality: "RELIABILITY", Severity: ""}},
			"⚪ Reliabil -",
		},
		{
			"empty quality",
			[]sonarcloud.IssueImpact{{SoftwareQuality: "", Severity: "LOW"}},
			"🟡 Unknown LOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared.FormatImpact(tt.impacts)
			if got != tt.want {
				t.Errorf("FormatImpact() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatCoverageSection(t *testing.T) {
	cmd := &ReportCmd{TruncateLines: 80}
	coverage := &sonarcloud.CoverageData{
		Available:       true,
		OverallCoverage: 75.5,
		NewCodeCoverage: 82.3,
		Files: []sonarcloud.CoverageFile{
			{Name: "src/main.go", Coverage: 90.0, UncoveredLines: 5, NewCoverage: 95.0},
			{Name: "src/handler.go", Coverage: 60.0, UncoveredLines: 20, NewCoverage: 70.0},
		},
	}
	filters := sonarcloud.FilterOptions{Limit: 10}

	output := captureStdout(func() {
		cmd.formatter().FormatCoverageSection(coverage, filters)
	})

	checks := []string{
		"Coverage Summary",
		"src/main.go",
		"src/handler.go",
		"90.0%",
		"60.0%",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("formatCoverageSection output missing %q", check)
		}
	}
}

func TestFormatIssuesSection(t *testing.T) {
	line := 42
	cmd := &ReportCmd{}
	issues := &sonarcloud.IssuesData{
		Available: true,
		Issues: []sonarcloud.ProcessedIssue{
			{
				File:     "src/main.go",
				Line:     &line,
				Message:  "Remove this unused variable",
				Severity: "MAJOR",
				Effort:   "5min",
				Impacts:  []sonarcloud.IssueImpact{{SoftwareQuality: "MAINTAINABILITY", Severity: "MEDIUM"}},
			},
		},
		Summary: sonarcloud.IssuesSummary{
			BySeverity:        map[string]int{"MEDIUM": 1},
			BySoftwareQuality: map[string]int{"MAINTAINABILITY": 1},
		},
	}
	metrics := &sonarcloud.MetricsData{
		TechnicalDebtMinutes: 5,
		Ratings:              map[string]string{"sqale_rating": "1.0"},
	}
	filters := sonarcloud.FilterOptions{Limit: 10}

	output := captureStdout(func() {
		cmd.formatter().FormatIssuesSection(issues, metrics, filters)
	})

	checks := []string{
		"Software Quality",
		"MAINTAINABILITY",
		"Severity Breakdown",
		"MEDIUM",
		"src/main.go",
		"Remove this unused variable",
		"Technical Debt",
		"5min",
		"Maintainability Rating: A",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("formatIssuesSection output missing %q", check)
		}
	}
}

func TestFormatDuplicationsSection(t *testing.T) {
	cmd := &ReportCmd{}
	duplications := &sonarcloud.DuplicationData{
		Available:          true,
		OverallDuplication: 3.5,
		NewCodeDuplication: 1.2,
		DuplicatedLines:    150,
		DuplicatedBlocks:   12,
		Files: []sonarcloud.DuplicatedFile{
			{Name: "src/utils.go", DuplicatedDensity: 15.0, DuplicatedLines: 30, DuplicatedBlocks: 3},
		},
	}
	filters := sonarcloud.FilterOptions{Limit: 10}

	output := captureStdout(func() {
		cmd.formatter().FormatDuplicationsSection(duplications, filters)
	})

	checks := []string{
		"Duplications Summary",
		"3.5%",
		"1.2%",
		"src/utils.go",
		"15.0%",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("formatDuplicationsSection output missing %q", check)
		}
	}
}

func TestFormatQualityGateHeader(t *testing.T) {
	t.Run("passed", func(t *testing.T) {
		cmd := &ReportCmd{Coverage: true, Issues: true}
		report := &sonarcloud.Report{
			ProjectKey: "my-project",
			QualityGate: &sonarcloud.QualityGateInfo{
				Passed: true,
			},
		}
		pipeline := &api.Pipeline{BuildNumber: 42}
		filters := sonarcloud.FilterOptions{IncludeCoverage: true, IncludeIssues: true}

		output := captureStdout(func() {
			cmd.formatTable(nil, report, pipeline, filters)
		})

		if !strings.Contains(output, "✅ PASSED") {
			t.Error("expected PASSED in quality gate header")
		}
		if !strings.Contains(output, "Pipeline #42") {
			t.Error("expected Pipeline #42 in header")
		}
	})

	t.Run("failed with conditions", func(t *testing.T) {
		cmd := &ReportCmd{}
		report := &sonarcloud.Report{
			ProjectKey: "my-project",
			QualityGate: &sonarcloud.QualityGateInfo{
				Passed: false,
				FailedConditions: []sonarcloud.QualityGateCondition{
					{MetricName: "Coverage", ActualValue: "50%", Comparator: ">", Threshold: "80%"},
				},
			},
		}
		pipeline := &api.Pipeline{BuildNumber: 99}
		filters := sonarcloud.FilterOptions{}

		output := captureStdout(func() {
			cmd.formatTable(nil, report, pipeline, filters)
		})

		if !strings.Contains(output, "❌ FAILED") {
			t.Error("expected FAILED in quality gate header")
		}
		if !strings.Contains(output, "Coverage") {
			t.Error("expected failed condition in output")
		}
	})
}
