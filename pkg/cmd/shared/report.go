package shared

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

// SeverityIcon returns an emoji icon for a severity level.
func SeverityIcon(severity string) string {
	switch severity {
	case "BLOCKER":
		return "🚫"
	case "HIGH", "CRITICAL":
		return "🔴"
	case "MEDIUM", "MAJOR":
		return "🟠"
	case "LOW", "MINOR":
		return "🟡"
	case "INFO":
		return "🔵"
	default:
		return "⚪"
	}
}

// FormatImpact formats the first impact entry with icon, quality, and severity.
func FormatImpact(impacts []sonarcloud.IssueImpact) string {
	if len(impacts) == 0 {
		return "-"
	}
	i := impacts[0]
	icon := SeverityIcon(i.Severity)
	qual := "Unknown"
	if i.SoftwareQuality != "" {
		qual = strings.ToTitle(strings.ToLower(i.SoftwareQuality[:1])) + strings.ToLower(i.SoftwareQuality[1:])
		if len(qual) > 8 {
			qual = qual[:8]
		}
	}
	sev := i.Severity
	if sev == "" {
		sev = "-"
	}
	return fmt.Sprintf("%s %s %s", icon, qual, sev)
}

// RatingFromMetrics extracts a rating letter from metrics data.
func RatingFromMetrics(metrics *sonarcloud.MetricsData, key string) string {
	if metrics == nil {
		return "?"
	}
	if r, ok := metrics.Ratings[key]; ok {
		return RatingNumberToLetter(r)
	}
	return "?"
}

// RatingNumberToLetter converts SonarCloud numeric rating to letter grade.
func RatingNumberToLetter(value string) string {
	switch value {
	case "1.0", "1":
		return "A"
	case "2.0", "2":
		return "B"
	case "3.0", "3":
		return "C"
	case "4.0", "4":
		return "D"
	case "5.0", "5":
		return "E"
	default:
		return value
	}
}

// FormatDebtMinutes formats technical debt minutes into human-readable form.
func FormatDebtMinutes(minutes int) string {
	if minutes == 0 {
		return "0min"
	}
	days := minutes / (8 * 60)
	remaining := minutes % (8 * 60)
	hours := remaining / 60
	mins := remaining % 60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours > 0 && mins > 0:
		return fmt.Sprintf("%dh%dmin", hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dmin", mins)
	}
}

// CreateSonarCloudService creates a SonarCloud service with token validation and connection test.
func CreateSonarCloudService(ctx context.Context, cmdCtx *CommandContext) (*sonarcloud.Service, error) {
	sonarConfig := sonarcloud.DefaultClientConfig()
	if sonarConfig.Token == "" {
		return nil, &sonarcloud.SonarCloudError{
			StatusCode:  0,
			UserMessage: "SonarCloud access denied. Set SONARCLOUD_TOKEN environment variable.",
			SuggestedActions: []string{
				"Go to https://sonarcloud.io/account/security/",
				"Generate a new token with a descriptive name",
				"Export it: export SONARCLOUD_TOKEN=\"your_token_here\"",
				"Add to ~/.zshenv or ~/.bashrc for persistence",
			},
			HelpLinks: []string{
				"https://sonarcloud.io/account/security/",
			},
		}
	}

	client, err := sonarcloud.NewClient(sonarConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SonarCloud client: %w", err)
	}

	service := sonarcloud.NewService(client, cmdCtx.Client)

	if err := service.TestConnection(ctx); err != nil {
		if scErr, ok := err.(*sonarcloud.SonarCloudError); ok {
			return nil, scErr
		}
		return nil, fmt.Errorf("failed to connect to SonarCloud: %w", err)
	}

	return service, nil
}

func statusIcon(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}

// ReportFormatter handles shared report table formatting.
type ReportFormatter struct {
	ShowAllLines  bool
	LinesPerFile  int
	TruncateLines int
}

// FormatQualityGateHeader prints quality gate status, project info, and failed conditions.
func (f *ReportFormatter) FormatQualityGateHeader(report *sonarcloud.Report) {
	if report.QualityGate == nil {
		return
	}

	status := "❌ FAILED"
	if report.QualityGate.Passed {
		status = "✅ PASSED"
	}
	fmt.Printf("Quality Gate: %s\n", status)

	if report.PullRequestID != nil {
		fmt.Printf("Project: %s | Pull Request: #%d\n", report.ProjectKey, *report.PullRequestID)
	} else {
		fmt.Printf("Project: %s\n", report.ProjectKey)
	}
	fmt.Printf("Analysis: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

	if len(report.QualityGate.FailedConditions) > 0 {
		fmt.Printf("❌ Quality Gate Failures:\n")
		for _, condition := range report.QualityGate.FailedConditions {
			fmt.Printf("  • %s: %s (required %s %s)\n",
				condition.MetricName, condition.ActualValue,
				condition.Comparator, condition.Threshold)
		}
		fmt.Println()
	}

	if report.QualityGate.Summary != nil {
		s := report.QualityGate.Summary
		fmt.Printf("📋 Quality Gate Summary:\n")
		fmt.Printf("┌──────────────┬───────────────────┬───────────────────┬──────────┬──────────────┐\n")
		fmt.Printf("│ New Issues   │ Accepted Issues   │ Security Hotspots │ Coverage │ Duplications │\n")
		fmt.Printf("├──────────────┼───────────────────┼───────────────────┼──────────┼──────────────┤\n")
		fmt.Printf("│ %12d │ %17d │ %17d │ %7.1f%% │ %11.1f%% │\n",
			s.NewIssues, s.AcceptedIssues, s.NewSecurityHotspots, s.NewCoverage, s.NewDuplicatedDensity)
		fmt.Printf("└──────────────┴───────────────────┴───────────────────┴──────────┴──────────────┘\n\n")
	}
}

// FormatCoverageSection prints the coverage summary table, uncovered line details, and goals.
func (f *ReportFormatter) FormatCoverageSection(coverage *sonarcloud.CoverageData, filters sonarcloud.FilterOptions) {
	fmt.Printf("📊 Coverage Summary:\n")
	fmt.Printf("┌─────────────────────────────────────────┬──────────┬─────────────────┬──────────────┐\n")
	fmt.Printf("│ File                                    │ Coverage │ Uncovered Lines │ New Coverage │\n")
	fmt.Printf("├─────────────────────────────────────────┼──────────┼─────────────────┼──────────────┤\n")

	displayedFiles := 0
	for _, file := range coverage.Files {
		if displayedFiles >= filters.Limit {
			break
		}

		if filters.NewLinesOnly && file.NewUncoveredLines == 0 {
			continue
		}

		fileName := Truncate(file.Name, 39)

		newCovStr := "-"
		if file.NewUncoveredLines > 0 || file.NewCoverage > 0 {
			newCovStr = fmt.Sprintf("%.1f%% (%d/%d)",
				file.NewCoverage, file.NewUncoveredLines, file.NewUncoveredLines)
		}

		fmt.Printf("│ %-39s │ %7.1f%% │ %15d │ %12s │\n",
			fileName, file.Coverage, file.UncoveredLines, newCovStr)
		displayedFiles++
	}

	fmt.Printf("└─────────────────────────────────────────┴──────────┴─────────────────┴──────────────┘\n\n")

	if len(coverage.CoverageDetails) > 0 {
		f.DisplayUncoveredLinesDetails(coverage.CoverageDetails, filters)
	}

	fmt.Printf("🎯 Coverage Goals:\n")

	if coverage.NewCodeCoverage > 0 {
		fmt.Printf("  • New Code (PR): %.1f%%", coverage.NewCodeCoverage)
		if coverage.NewCodeCoverage >= 90 {
			fmt.Printf(" ✅")
		} else {
			fmt.Printf(" → Required: 90%% ❌")
		}
		fmt.Println()

		fmt.Printf("  • Overall Project: %.1f%% (for context)", coverage.OverallCoverage)
		fmt.Println()
	} else {
		fmt.Printf("  • Overall Project: %.1f%%", coverage.OverallCoverage)
		if coverage.OverallCoverage >= 80 {
			fmt.Printf(" ✅")
		} else {
			fmt.Printf(" → Target: 80%%")
		}
		fmt.Println()
	}
	fmt.Println()
}

// DisplayUncoveredLinesDetails prints per-file uncovered line breakdowns and focus areas.
func (f *ReportFormatter) DisplayUncoveredLinesDetails(coverageDetails []sonarcloud.CoverageDetails, filters sonarcloud.FilterOptions) {
	fmt.Printf("🔍 Uncovered Lines Details:\n\n")

	displayedFiles := 0
	maxFilesToShow := filters.Limit
	if maxFilesToShow <= 0 {
		maxFilesToShow = 10
	}

	for _, details := range coverageDetails {
		if displayedFiles >= maxFilesToShow {
			break
		}

		if len(details.UncoveredLines) == 0 {
			continue
		}

		fmt.Printf("%s (%.1f%% coverage):\n", details.FilePath, details.CoveragePercent)

		lineGaps, branchGaps := SplitCoverageGaps(details.UncoveredLines)

		linesShown := f.printGapSection("  Uncovered Lines:", lineGaps, false)
		branchShown := f.printGapSection("  Partial Branch Coverage:", branchGaps, true)

		hidden := (len(lineGaps) - linesShown) + (len(branchGaps) - branchShown)
		if hidden > 0 {
			fmt.Printf("  ... (%d more uncovered entries) Use --show-all-lines to see complete list\n", hidden)
		}

		fmt.Println()
		displayedFiles++
	}

	f.formatFocusAreas(coverageDetails, maxFilesToShow)
}

// SplitCoverageGaps separates uncovered entries into untouched lines and
// partial-branch lines so they can be rendered in distinct sections.
// A line with conditions defined and CoveredConditions < Conditions is a
// branch gap; otherwise it's a line gap. This lives outside the formatter
// so report consumers (and tests) can rely on it as a stable helper.
func SplitCoverageGaps(lines []sonarcloud.UncoveredLine) (lineGaps, branchGaps []sonarcloud.UncoveredLine) {
	for _, line := range lines {
		if line.Conditions > 0 && line.CoveredConditions < line.Conditions {
			branchGaps = append(branchGaps, line)
		} else {
			lineGaps = append(lineGaps, line)
		}
	}
	return lineGaps, branchGaps
}

// printGapSection renders a labelled subsection of uncovered entries and
// returns how many were actually printed (so the caller can compute the
// "N more" hint correctly across both sections).
func (f *ReportFormatter) printGapSection(label string, lines []sonarcloud.UncoveredLine, showBranchCount bool) int {
	if len(lines) == 0 {
		return 0
	}

	toShow := lines
	if !f.ShowAllLines && len(toShow) > f.LinesPerFile {
		toShow = toShow[:f.LinesPerFile]
	}

	fmt.Println(label)
	for _, line := range toShow {
		marker := ""
		if line.IsNew {
			marker = " [NEW]"
		}

		code := line.Code
		if f.TruncateLines > 0 {
			code = Truncate(code, f.TruncateLines)
		}

		if showBranchCount && line.Conditions > 0 {
			fmt.Printf("    Line %d: %s%s (%d/%d branches)\n",
				line.Line, code, marker, line.CoveredConditions, line.Conditions)
		} else {
			fmt.Printf("    Line %d: %s%s\n", line.Line, code, marker)
		}
	}
	return len(toShow)
}

func (f *ReportFormatter) formatFocusAreas(coverageDetails []sonarcloud.CoverageDetails, maxFilesToShow int) {
	totalNewLines := 0
	totalFiles := 0
	quickWins := 0

	for _, details := range coverageDetails {
		totalFiles++
		totalNewLines += details.NewUncovered
		if details.TotalUncovered <= 10 && details.CoveragePercent < 80 {
			quickWins++
		}
	}

	if totalNewLines > 0 || quickWins > 0 {
		fmt.Printf("🎯 Focus Areas:\n")
		if totalNewLines > 0 {
			fmt.Printf("  • NEW uncovered lines: %d lines added in this PR need test coverage\n", totalNewLines)
		}
		if quickWins > 0 {
			fmt.Printf("  • Quick wins: %d files need <10 lines of coverage to reach 80%%\n", quickWins)
		}
		if totalFiles > maxFilesToShow {
			fmt.Printf("  • %d more files have uncovered lines (use --limit %d to see more)\n",
				totalFiles-maxFilesToShow, totalFiles)
		}
		fmt.Println()
	}
}

// FormatIssuesSection prints software quality, severity breakdown, issues table, and debt.
func (f *ReportFormatter) FormatIssuesSection(issues *sonarcloud.IssuesData, metrics *sonarcloud.MetricsData, filters sonarcloud.FilterOptions) {
	if len(issues.Summary.BySoftwareQuality) > 0 {
		fmt.Printf("🏗  Software Quality:\n")
		fmt.Printf("┌─────────────────────┬───────┐\n")
		fmt.Printf("│ Quality             │ Count │\n")
		fmt.Printf("├─────────────────────┼───────┤\n")
		for _, quality := range []string{"SECURITY", "RELIABILITY", "MAINTAINABILITY"} {
			count := issues.Summary.BySoftwareQuality[quality]
			fmt.Printf("│ %-19s │ %5d │\n", quality, count)
		}
		fmt.Printf("└─────────────────────┴───────┘\n\n")
	}

	fmt.Printf("🐛 Severity Breakdown:\n")
	fmt.Printf("┌──────────────┬───────┐\n")
	fmt.Printf("│ Severity     │ Count │\n")
	fmt.Printf("├──────────────┼───────┤\n")
	for _, sev := range []string{"BLOCKER", "HIGH", "MEDIUM", "LOW", "INFO"} {
		count := issues.Summary.BySeverity[sev]
		icon := SeverityIcon(sev)
		fmt.Printf("│ %s %-8s │ %5d │\n", icon, sev, count)
	}
	fmt.Printf("└──────────────┴───────┘\n\n")

	if len(issues.Issues) > 0 {
		fmt.Printf("🔥 Issues:\n")
		fmt.Printf("┌────────────────────┬─────────────────────────────────────────┬──────┬──────────┬──────────────────────────────────────────┐\n")
		fmt.Printf("│ Impact             │ File                                    │ Line │ Effort   │ Description                              │\n")
		fmt.Printf("├────────────────────┼─────────────────────────────────────────┼──────┼──────────┼──────────────────────────────────────────┤\n")

		displayedIssues := 0
		for _, issue := range issues.Issues {
			if displayedIssues >= filters.Limit {
				break
			}

			fileName := Truncate(issue.File, 39)

			lineStr := "-"
			if issue.Line != nil {
				lineStr = strconv.Itoa(*issue.Line)
			}

			description := Truncate(issue.Message, 40)

			effort := issue.Effort
			if effort == "" {
				effort = "-"
			}
			if len(effort) > 8 {
				effort = effort[:8]
			}

			impact := FormatImpact(issue.Impacts)

			fmt.Printf("│ %-18s │ %-39s │ %4s │ %-8s │ %-40s │\n",
				impact, fileName, lineStr, effort, description)
			displayedIssues++
		}

		fmt.Printf("└────────────────────┴─────────────────────────────────────────┴──────┴──────────┴──────────────────────────────────────────┘\n\n")
	}

	if metrics != nil {
		fmt.Printf("💰 Technical Debt: %s\n", FormatDebtMinutes(metrics.TechnicalDebtMinutes))

		rating := RatingFromMetrics(metrics, "sqale_rating")
		status := "❌"
		if rating == "A" {
			status = "✅"
		}
		fmt.Printf("📈 Maintainability Rating: %s (target: A) %s\n\n", rating, status)
	}
}

// FormatDuplicationsSection prints duplication summary, file table, and block details.
func (f *ReportFormatter) FormatDuplicationsSection(duplications *sonarcloud.DuplicationData, filters sonarcloud.FilterOptions) {
	fmt.Printf("📋 Duplications Summary:\n")
	fmt.Printf("  Overall: %.1f%% | New Code: %.1f%%\n", duplications.OverallDuplication, duplications.NewCodeDuplication)
	fmt.Printf("  Duplicated Lines: %d | Duplicated Blocks: %d\n\n", duplications.DuplicatedLines, duplications.DuplicatedBlocks)

	if len(duplications.Files) > 0 {
		fmt.Printf("📁 Files with Duplications:\n")
		fmt.Printf("┌─────────────────────────────────────────┬──────────────┬─────────┬────────┐\n")
		fmt.Printf("│ File                                    │ Duplication  │ Lines   │ Blocks │\n")
		fmt.Printf("├─────────────────────────────────────────┼──────────────┼─────────┼────────┤\n")

		displayed := 0
		for _, file := range duplications.Files {
			if displayed >= filters.Limit {
				break
			}
			fileName := Truncate(file.Name, 39)
			fmt.Printf("│ %-39s │ %11.1f%% │ %7d │ %6d │\n",
				fileName, file.DuplicatedDensity, file.DuplicatedLines, file.DuplicatedBlocks)
			displayed++
		}

		fmt.Printf("└─────────────────────────────────────────┴──────────────┴─────────┴────────┘\n\n")
	}

	if len(duplications.Details) > 0 {
		fmt.Printf("🔍 Duplicated Blocks:\n\n")

		displayed := 0
		for _, detail := range duplications.Details {
			if displayed >= filters.Limit {
				break
			}
			if len(detail.Blocks) == 0 {
				continue
			}

			fmt.Printf("%s (%.1f%% duplication):\n", detail.FilePath, detail.DuplicatedDensity)
			for _, block := range detail.Blocks {
				fmt.Printf("  ▶ Lines %d-%d (%d lines) → %s lines %d-%d\n",
					block.From, block.From+block.Size-1, block.Size,
					block.TargetFile, block.TargetFrom, block.TargetFrom+block.TargetSize-1)
			}
			fmt.Println()
			displayed++
		}
	}
}

// FormatOverviewSection prints the summary overview table with all metrics.
func (f *ReportFormatter) FormatOverviewSection(report *sonarcloud.Report) {
	fmt.Printf("📊 Overview:\n")
	fmt.Printf("┌─────────────────────┬─────────┬─────────────┬────────┐\n")
	fmt.Printf("│ Metric              │ Overall │ New Code    │ Status │\n")
	fmt.Printf("├─────────────────────┼─────────┼─────────────┼────────┤\n")

	if report.Coverage != nil {
		overallCov := fmt.Sprintf("%.1f%%", report.Coverage.OverallCoverage)
		newCov := "-"
		if report.Coverage.NewCodeCoverage > 0 {
			newCov = fmt.Sprintf("%.1f%%", report.Coverage.NewCodeCoverage)
		}
		status := "✅"
		if report.Coverage.OverallCoverage < 80 || report.Coverage.NewCodeCoverage < 90 {
			status = "❌"
		}
		fmt.Printf("│ Coverage            │ %7s │ %-11s │ %6s │\n", overallCov, newCov, status)
	}

	if report.Issues != nil {
		newBugs, newVulns, newSmells, newHotspots := 0, 0, 0, 0
		if report.Metrics != nil {
			newBugs = report.Metrics.NewBugs
			newVulns = report.Metrics.NewVulnerabilities
			newSmells = report.Metrics.NewCodeSmells
			newHotspots = report.Metrics.NewSecurityHotspots
		}
		fmt.Printf("│ Bugs                │ %7d │ %-11d │ %6s │\n",
			report.Issues.Bugs, newBugs, statusIcon(newBugs == 0))
		fmt.Printf("│ Vulnerabilities     │ %7d │ %-11d │ %6s │\n",
			report.Issues.Vulnerabilities, newVulns, statusIcon(newVulns == 0))
		fmt.Printf("│ Code Smells         │ %7d │ %-11d │ %6s │\n",
			report.Issues.CodeSmells, newSmells, statusIcon(newSmells == 0))
		fmt.Printf("│ Security Hotspots   │ %7d │ %-11d │ %6s │\n",
			report.Issues.SecurityHotspots, newHotspots, statusIcon(newHotspots == 0))
	}

	if report.Metrics != nil {
		newDup := fmt.Sprintf("%.1f%%", report.Metrics.NewDuplicatedDensity)
		dupStatus := "✅"
		if report.Metrics.Duplication > 3.0 {
			dupStatus = "❌"
		}
		fmt.Printf("│ Duplications        │ %6.1f%% │ %-11s │ %6s │\n",
			report.Metrics.Duplication, newDup, dupStatus)
	}

	fmt.Printf("└─────────────────────┴─────────┴─────────────┴────────┘\n\n")
}

// FormatLinksSection prints the SonarCloud dashboard link.
func (f *ReportFormatter) FormatLinksSection(report *sonarcloud.Report) {
	var url string
	if report.PullRequestID != nil {
		url = fmt.Sprintf("https://sonarcloud.io/dashboard?id=%s&pullRequest=%d",
			report.ProjectKey, *report.PullRequestID)
	} else {
		url = fmt.Sprintf("https://sonarcloud.io/project/overview?id=%s", report.ProjectKey)
	}

	fmt.Printf("🔗 Full Analysis: %s\n", url)
}

// FormatWarnings prints report warnings if any.
func (f *ReportFormatter) FormatWarnings(report *sonarcloud.Report) {
	if len(report.Warnings) > 0 {
		fmt.Printf("\n⚠ Warnings:\n")
		for _, warning := range report.Warnings {
			fmt.Printf("  • %s\n", warning.Error())
		}
	}
}

// IssueTypeLabel converts SonarCloud type constants to a human-readable label.
func IssueTypeLabel(t string) string {
	switch t {
	case "BUG":
		return "Bug"
	case "VULNERABILITY":
		return "Vulnerability"
	case "CODE_SMELL":
		return "Code Smell"
	case "SECURITY_HOTSPOT":
		return "Security Hotspot"
	default:
		return t
	}
}

func assigneeOrUnassigned(a string) string {
	if strings.TrimSpace(a) == "" {
		return "unassigned"
	}
	return a
}

func inDiffLabel(file string, line *int, diffLines map[string]map[int]bool) string {
	if diffLines == nil || line == nil {
		return "unknown"
	}
	if lines, ok := diffLines[file]; ok && lines[*line] {
		return "yes"
	}
	return "no"
}

func severityOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func lineOrDash(line *int) string {
	if line == nil {
		return "-"
	}
	return strconv.Itoa(*line)
}

// FormatActionableIssuesSection prints the bullet list of actionable PR issues.
// Returns the count rendered.
func (f *ReportFormatter) FormatActionableIssuesSection(actionable *sonarcloud.IssuesData, diffLines map[string]map[int]bool) int {
	if actionable == nil || len(actionable.Issues) == 0 {
		fmt.Printf("Actionable New Issues: 0\n")
		fmt.Printf("  ✅ Nothing for the PR author to fix.\n\n")
		return 0
	}
	n := len(actionable.Issues)
	fmt.Printf("Actionable New Issues: %d\n", n)
	for _, iss := range actionable.Issues {
		fmt.Printf("- [%s] %s %s:%s %s\n",
			severityOrDash(iss.Severity),
			IssueTypeLabel(iss.Type),
			iss.File,
			lineOrDash(iss.Line),
			assigneeOrUnassigned(iss.Assignee),
		)
		fmt.Printf("  %s\n", iss.Message)
		fmt.Printf("  Rule: %s\n", iss.Rule)
		fmt.Printf("  In PR diff: %s\n", inDiffLabel(iss.File, iss.Line, diffLines))
	}
	fmt.Println()
	return n
}

// FormatAcceptedSummary renders the Accepted/Pre-existing block. When showAll
// is false, only counts are shown. When true, full details are printed.
func (f *ReportFormatter) FormatAcceptedSummary(accepted *sonarcloud.IssuesData, qgAcceptedCount int, showAll bool) {
	detailCount := 0
	if accepted != nil {
		detailCount = len(accepted.Issues)
	}
	total := detailCount
	if qgAcceptedCount > total {
		total = qgAcceptedCount
	}
	fmt.Printf("Accepted / Pre-existing Issues: %d\n", total)
	if !showAll {
		fmt.Printf("Hidden by default. Use --all-issues to show details.\n\n")
		return
	}
	if detailCount == 0 {
		fmt.Printf("  (no detail records returned)\n\n")
		return
	}
	for _, iss := range accepted.Issues {
		res := iss.Resolution
		if res == "" {
			res = "ACCEPTED"
		}
		fmt.Printf("- [%s] %s %s:%s (%s) %s\n",
			severityOrDash(iss.Severity),
			IssueTypeLabel(iss.Type),
			iss.File,
			lineOrDash(iss.Line),
			res,
			assigneeOrUnassigned(iss.Assignee),
		)
		fmt.Printf("  %s\n", iss.Message)
		fmt.Printf("  Rule: %s\n", iss.Rule)
	}
	fmt.Println()
}

// expectedNewIssueCount sums new bugs, vulnerabilities, and code smells from
// metrics. Hotspots are intentionally excluded — they are not "issues" in the
// SonarCloud taxonomy.
func expectedNewIssueCount(metrics *sonarcloud.MetricsData, summary *sonarcloud.QualityGateSummary) int {
	if summary != nil && summary.NewIssues > 0 {
		return summary.NewIssues
	}
	if metrics == nil {
		return 0
	}
	return metrics.NewBugs + metrics.NewVulnerabilities + metrics.NewCodeSmells
}

// WarnGateMismatch compares quality gate metric counts against the actionable
// issue list size. When the gate reports more new issues than the issue list
// returned, it prints a warning to stderr and returns (expected, true) so the
// caller can retry with relaxed filters. Returns (expected, false) otherwise.
func WarnGateMismatch(metrics *sonarcloud.MetricsData, summary *sonarcloud.QualityGateSummary, actionableCount int) (int, bool) {
	expected := expectedNewIssueCount(metrics, summary)
	if expected <= actionableCount {
		return expected, false
	}
	fmt.Fprintf(os.Stderr,
		"Quality gate reports %d new code smells, but only %d were returned by the default issue query. Retrying with new-code filters...\n",
		expected, actionableCount,
	)
	return expected, true
}
