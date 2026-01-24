package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

type ReportCmd struct {
	PRID              string   `arg:"" help:"Pull request ID (number)"`
	Output            string   `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	Coverage          bool     `help:"Show only coverage-related information"`
	Issues            bool     `help:"Show only code quality issues"`
	Web               bool     `help:"Open SonarCloud dashboard in browser"`
	URL               bool     `help:"Print SonarCloud URL instead of opening browser"`
	CoverageThreshold int      `name:"coverage-threshold" help:"Show only files below N% coverage"`
	Limit             int      `help:"Limit number of files/issues shown" default:"10"`
	NewCodeOnly       bool     `name:"new-code-only" help:"Focus on new code analysis"`
	Severity          []string `help:"Filter issues by severity level (BLOCKER,CRITICAL,MAJOR,MINOR,INFO)"`
	ShowAllLines      bool     `name:"show-all-lines" help:"Show all uncovered lines (not just top 5 per file)"`
	LinesPerFile      int      `name:"lines-per-file" help:"Max lines to show per file" default:"5"`
	NewLinesOnly      bool     `name:"new-lines-only" help:"Only show NEW uncovered lines from this PR"`
	MinUncoveredLines int      `name:"min-uncovered-lines" help:"Only show files with N+ uncovered lines"`
	MaxUncoveredLines int      `name:"max-uncovered-lines" help:"Only show files with ≤N uncovered lines (quick wins)"`
	FilePattern       string   `name:"file" help:"Filter to specific files (glob pattern)"`
	NoLineDetails     bool     `name:"no-line-details" help:"Skip line-by-line breakdown (performance)"`
	TruncateLines     int      `name:"truncate-lines" help:"Truncate code lines after N characters" default:"80"`
	Context           int      `name:"context" help:"Show N lines of context around each uncovered line"`
	Debug             bool     `help:"Enable debug output for troubleshooting"`
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

func (cmd *ReportCmd) Run(ctx context.Context) error {
	noColor := false
	if v := ctx.Value("no-color"); v != nil {
		noColor = v.(bool)
	}

	prCtx, err := shared.NewCommandContext(ctx, cmd.Output, noColor)
	if err != nil {
		return err
	}

	if cmd.Workspace != "" {
		prCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		prCtx.Repository = cmd.Repository
	}

	if err := prCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	if strings.TrimSpace(cmd.PRID) == "" {
		return fmt.Errorf("pull request ID is required")
	}

	prID, err := cmd.parsePRID()
	if err != nil {
		return err
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: Pull Request ID: %d\n", prID)
	}

	if cmd.Web || cmd.URL {
		return cmd.openSonarCloudDashboard(ctx, prCtx, prID)
	}

	sonarCloudService, err := cmd.createSonarCloudService(ctx, prCtx)
	if err != nil {
		return err
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: About to generate SonarCloud report for PR %d\n", prID)
	}

	// Check branch compatibility for context feature
	if cmd.Context > 0 {
		if err := cmd.checkBranchCompatibility(ctx, prCtx, prID); err != nil {
			// Don't fail, just warn
			fmt.Printf("⚠️  %s\n\n", err.Error())
		}
	}

	return cmd.generateReport(ctx, prCtx, sonarCloudService, prID)
}

func (cmd *ReportCmd) parsePRID() (int, error) {
	prid := strings.TrimSpace(cmd.PRID)
	if strings.HasPrefix(prid, "#") {
		prid = prid[1:]
	}

	prID, err := strconv.Atoi(prid)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s'. Expected number (e.g., 123, #123)", cmd.PRID)
	}

	return prID, nil
}

func (cmd *ReportCmd) createSonarCloudService(ctx context.Context, prCtx *PRContext) (*sonarcloud.Service, error) {
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

	service := sonarcloud.NewService(client, prCtx.Client)

	if err := service.TestConnection(ctx); err != nil {
		if scErr, ok := err.(*sonarcloud.SonarCloudError); ok {
			return nil, scErr
		}
		return nil, fmt.Errorf("failed to connect to SonarCloud: %w", err)
	}

	return service, nil
}

func (cmd *ReportCmd) openSonarCloudDashboard(ctx context.Context, prCtx *PRContext, prID int) error {
	sonarCloudService, err := cmd.createSonarCloudService(ctx, prCtx)
	if err != nil {
		return err
	}

	filters := sonarcloud.FilterOptions{
		IncludeCoverage: false,
		IncludeIssues:   false,
	}

	report, err := sonarCloudService.GenerateReportForPR(ctx, prID, prCtx.Workspace, prCtx.Repository, filters)
	if err != nil {
		return fmt.Errorf("failed to discover SonarCloud project: %w", err)
	}

	var url string
	if report.PullRequestID != nil {
		url = fmt.Sprintf("https://sonarcloud.io/dashboard?id=%s&pullRequest=%d",
			report.ProjectKey, *report.PullRequestID)
	} else {
		url = fmt.Sprintf("https://sonarcloud.io/project/overview?id=%s", report.ProjectKey)
	}

	if cmd.URL {
		fmt.Println(url)
		return nil
	}

	return cmd.launchBrowser(url)
}

func (cmd *ReportCmd) launchBrowser(url string) error {
	var cmdName string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmdName = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmdName = "open"
		args = []string{url}
	case "linux":
		cmdName = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	execCmd := exec.Command(cmdName, args...)
	return execCmd.Start()
}

func (cmd *ReportCmd) generateReport(ctx context.Context, prCtx *PRContext, service *sonarcloud.Service, prID int) error {
	filters := sonarcloud.FilterOptions{
		IncludeCoverage:   !cmd.Issues || cmd.Coverage,
		IncludeIssues:     !cmd.Coverage || cmd.Issues,
		CoverageThreshold: float64(cmd.CoverageThreshold),
		Limit:             cmd.Limit,
		NewCodeOnly:       cmd.NewCodeOnly,
		SeverityFilter:    cmd.Severity,
		ShowWorstFirst:    true,
		ShowAllLines:      cmd.ShowAllLines,
		LinesPerFile:      cmd.LinesPerFile,
		NewLinesOnly:      cmd.NewLinesOnly,
		MinUncoveredLines: cmd.MinUncoveredLines,
		MaxUncoveredLines: cmd.MaxUncoveredLines,
		FilePattern:       cmd.FilePattern,
		NoLineDetails:     cmd.NoLineDetails,
		TruncateLines:     cmd.TruncateLines,
		Debug:             cmd.Debug,
	}

	if !cmd.Coverage && !cmd.Issues {
		filters.IncludeCoverage = true
		filters.IncludeIssues = true
	}

	report, err := service.GenerateReportForPR(ctx, prID, prCtx.Workspace, prCtx.Repository, filters)
	if err != nil {
		return err
	}

	return cmd.formatOutput(prCtx, report, prID, filters)
}

func (cmd *ReportCmd) formatOutput(prCtx *PRContext, report *sonarcloud.Report, prID int, filters sonarcloud.FilterOptions) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, report, prID, filters)
	case "json":
		return prCtx.Formatter.Format(report)
	case "yaml":
		return prCtx.Formatter.Format(report)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ReportCmd) formatTable(prCtx *PRContext, report *sonarcloud.Report, prID int, filters sonarcloud.FilterOptions) error {
	reportType := "SonarCloud Quality Report"
	if cmd.Coverage && !cmd.Issues {
		reportType = "Coverage Analysis"
	} else if cmd.Issues && !cmd.Coverage {
		reportType = "Issues Analysis"
	}

	fmt.Printf("=== %s: Pull Request #%d ===\n\n", reportType, prID)

	if report.QualityGate != nil {
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
	}

	if filters.IncludeCoverage && report.Coverage != nil && report.Coverage.Available {
		cmd.formatCoverageSection(report.Coverage, filters)
	}

	if filters.IncludeIssues && report.Issues != nil && report.Issues.Available {
		cmd.formatIssuesSection(report.Issues, filters)
	}

	if filters.IncludeCoverage && filters.IncludeIssues {
		cmd.formatOverviewSection(report, prID)
	}

	cmd.formatLinksSection(report)

	if len(report.Warnings) > 0 {
		fmt.Printf("\n⚠ Warnings:\n")
		for _, warning := range report.Warnings {
			fmt.Printf("  • %s\n", warning.Error())
		}
	}

	return nil
}

func (cmd *ReportCmd) formatCoverageSection(coverage *sonarcloud.CoverageData, filters sonarcloud.FilterOptions) {
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

		fileName := file.Name
		if len(fileName) > 39 {
			fileName = fileName[:36] + "..."
		}

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
		cmd.displayUncoveredLinesDetails(coverage.CoverageDetails, filters)
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

func (cmd *ReportCmd) displayUncoveredLinesDetails(coverageDetails []sonarcloud.CoverageDetails, filters sonarcloud.FilterOptions) {
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

		linesToShow := details.UncoveredLines
		if !cmd.ShowAllLines && len(linesToShow) > cmd.LinesPerFile {
			linesToShow = linesToShow[:cmd.LinesPerFile]
		}

		if cmd.Context > 0 {
			cmd.displayLinesWithContext(details.FilePath, linesToShow, cmd.Context, cmd.NewLinesOnly, cmd.TruncateLines)
		} else {
			for _, line := range linesToShow {
				marker := ""
				if line.IsNew {
					marker = " [NEW]"
				}

				code := line.Code
				if cmd.TruncateLines > 0 && len(code) > cmd.TruncateLines {
					code = code[:cmd.TruncateLines-3] + "..."
				}

				fmt.Printf("▶ %d %s%s\n", line.Line, code, marker)
			}
		}

		remaining := len(details.UncoveredLines) - len(linesToShow)
		if remaining > 0 {
			fmt.Printf("  ... (%d more uncovered lines) Use --show-all-lines to see complete list\n", remaining)
		}

		fmt.Println()
		displayedFiles++
	}

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

func (cmd *ReportCmd) formatIssuesSection(issues *sonarcloud.IssuesData, filters sonarcloud.FilterOptions) {
	fmt.Printf("🐛 Issues Breakdown:\n")
	fmt.Printf("┌──────────────┬───────┬─────────────────┐\n")
	fmt.Printf("│ Type         │ Count │ New in PR       │\n")
	fmt.Printf("├──────────────┼───────┼─────────────────┤\n")

	fmt.Printf("│ Bugs         │ %5d │ %15d │\n", issues.Bugs, issues.NewIssues)
	fmt.Printf("│ Vulnerabilities │ %2d │ %15d │\n", issues.Vulnerabilities, 0)
	fmt.Printf("│ Code Smells  │ %5d │ %15d │\n", issues.CodeSmells, 0)
	fmt.Printf("│ Security Hotspots │ %2d │ %15d │\n", issues.SecurityHotspots, 0)
	fmt.Printf("└──────────────┴───────┴─────────────────┘\n\n")

	if len(issues.Issues) > 0 {
		fmt.Printf("🔥 Critical Issues:\n")
		fmt.Printf("┌──────────────┬─────────────────────────────────────────┬──────┬─────────────────────────────────────────────────────┐\n")
		fmt.Printf("│ Severity     │ File                                    │ Line │ Description                                         │\n")
		fmt.Printf("├──────────────┼─────────────────────────────────────────┼──────┼─────────────────────────────────────────────────────┤\n")

		displayedIssues := 0
		for _, issue := range issues.Issues {
			if displayedIssues >= filters.Limit {
				break
			}

			fileName := issue.File
			if len(fileName) > 39 {
				fileName = fileName[:36] + "..."
			}

			lineStr := "-"
			if issue.Line != nil {
				lineStr = strconv.Itoa(*issue.Line)
			}

			description := issue.Message
			if len(description) > 51 {
				description = description[:48] + "..."
			}

			severityIcon := ""
			switch issue.Severity {
			case "BLOCKER":
				severityIcon = "🚫 BLOCKER"
			case "CRITICAL":
				severityIcon = "🔴 CRITICAL"
			case "MAJOR":
				severityIcon = "🟠 MAJOR"
			case "MINOR":
				severityIcon = "🟡 MINOR"
			default:
				severityIcon = issue.Severity
			}

			fmt.Printf("│ %-12s │ %-39s │ %4s │ %-51s │\n",
				severityIcon, fileName, lineStr, description)
			displayedIssues++
		}

		fmt.Printf("└──────────────┴─────────────────────────────────────────┴──────┴─────────────────────────────────────────────────────┘\n\n")
	}

	if len(issues.Issues) > 0 {
		totalDebt := "0"
		for _, issue := range issues.Issues {
			if issue.TechnicalDebt != "" {
				totalDebt = issue.TechnicalDebt
				break
			}
		}

		fmt.Printf("💰 Technical Debt: %s\n", totalDebt)

		fmt.Printf("📈 Maintainability Rating: B (target: A) ❌\n\n")
	}
}

func (cmd *ReportCmd) formatOverviewSection(report *sonarcloud.Report, prID int) {
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
		fmt.Printf("│ Bugs                │ %7d │ %-11d │ %6s │\n",
			report.Issues.Bugs, 0, "✅")
		fmt.Printf("│ Vulnerabilities     │ %7d │ %-11d │ %6s │\n",
			report.Issues.Vulnerabilities, 0, "✅")
		fmt.Printf("│ Code Smells         │ %7d │ %-11d │ %6s │\n",
			report.Issues.CodeSmells, 0, "❌")
		fmt.Printf("│ Security Hotspots   │ %7d │ %-11d │ %6s │\n",
			report.Issues.SecurityHotspots, 0, "✅")
	}

	if report.Metrics != nil {
		fmt.Printf("│ Duplications        │ %6.1f%% │ %-11s │ %6s │\n",
			report.Metrics.Duplication, "0.0%", "✅")
	}

	fmt.Printf("└─────────────────────┴─────────┴─────────────┴────────┘\n\n")
}

func (cmd *ReportCmd) colorizeNewLine(text string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", text)
}

func (cmd *ReportCmd) displayLinesWithContext(filePath string, uncoveredLines []sonarcloud.UncoveredLine, contextLines int, newLinesOnly bool, truncateLines int) {
	if len(uncoveredLines) == 0 {
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		for _, line := range uncoveredLines {
			code := line.Code
			if truncateLines > 0 && len(code) > truncateLines {
				code = code[:truncateLines-3] + "..."
			}

			if line.IsNew {
				fmt.Printf("%s\n", cmd.colorizeNewLine(fmt.Sprintf("▶ %d %s [NEW]", line.Line, code)))
			} else {
				fmt.Printf("▶ %d %s\n", line.Line, code)
			}
		}
		return
	}

	ranges := cmd.buildContextRanges(uncoveredLines, contextLines)

	for i, lineRange := range ranges {
		if i > 0 {
			fmt.Println()
		}

		cmd.displayContextRange(filePath, lineRange, uncoveredLines, newLinesOnly, truncateLines)
	}
}

func (cmd *ReportCmd) buildContextRanges(uncoveredLines []sonarcloud.UncoveredLine, contextLines int) [][2]int {
	if len(uncoveredLines) == 0 {
		return nil
	}

	sortedLines := make([]int, len(uncoveredLines))
	for i, line := range uncoveredLines {
		sortedLines[i] = line.Line
	}
	sort.Ints(sortedLines)

	var ranges [][2]int
	start := sortedLines[0] - contextLines
	if start < 1 {
		start = 1
	}
	end := sortedLines[0] + contextLines

	for i := 1; i < len(sortedLines); i++ {
		lineStart := sortedLines[i] - contextLines
		lineEnd := sortedLines[i] + contextLines

		if lineStart <= end+1 {
			end = lineEnd
		} else {
			ranges = append(ranges, [2]int{start, end})
			start = lineStart
			if start < 1 {
				start = 1
			}
			end = lineEnd
		}
	}

	ranges = append(ranges, [2]int{start, end})

	return ranges
}

func (cmd *ReportCmd) displayContextRange(filePath string, lineRange [2]int, uncoveredLines []sonarcloud.UncoveredLine, newLinesOnly bool, truncateLines int) {
	uncoveredMap := make(map[int]sonarcloud.UncoveredLine)
	for _, line := range uncoveredLines {
		uncoveredMap[line.Line] = line
	}

	allLinesCmd := exec.Command("rg", "--line-number", "--no-heading", "--color", "never", ".*", filePath)
	output, err := allLinesCmd.Output()
	if err != nil {
		for _, line := range uncoveredLines {
			if line.Line >= lineRange[0] && line.Line <= lineRange[1] {
				code := line.Code
				if truncateLines > 0 && len(code) > truncateLines {
					code = code[:truncateLines-3] + "..."
				}

				if line.IsNew {
					fmt.Printf("%s\n", cmd.colorizeNewLine(fmt.Sprintf("▶ %d %s", line.Line, code)))
				} else {
					fmt.Printf("▶ %d %s\n", line.Line, code)
				}
			}
		}
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		lineNum, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		if lineNum >= lineRange[0] && lineNum <= lineRange[1] {
			code := parts[1]
			if truncateLines > 0 && len(code) > truncateLines {
				code = code[:truncateLines-3] + "..."
			}

			if uncoveredLine, isUncovered := uncoveredMap[lineNum]; isUncovered {
				if uncoveredLine.IsNew {
					fmt.Printf("%s\n", cmd.colorizeNewLine(fmt.Sprintf("▶ %d %s [NEW]", lineNum, code)))
				} else {
					fmt.Printf("▶ %d %s\n", lineNum, code)
				}
			} else {
				fmt.Printf("  %d %s\n", lineNum, code)
			}
		}
	}
}

func (cmd *ReportCmd) formatLinksSection(report *sonarcloud.Report) {
	var url string
	if report.PullRequestID != nil {
		url = fmt.Sprintf("https://sonarcloud.io/dashboard?id=%s&pullRequest=%d",
			report.ProjectKey, *report.PullRequestID)
	} else {
		url = fmt.Sprintf("https://sonarcloud.io/project/overview?id=%s", report.ProjectKey)
	}

	fmt.Printf("🔗 Full Analysis: %s\n", url)
}

func (cmd *ReportCmd) checkBranchCompatibility(ctx context.Context, prCtx *PRContext, prID int) error {
	// Get current git branch
	currentBranch, err := cmd.getCurrentGitBranch()
	if err != nil {
		if cmd.Debug {
			fmt.Printf("DEBUG: Could not get current git branch: %v\n", err)
		}
		return nil // Don't warn if we can't detect branches
	}

	// Get PR source branch
	prBranch, err := cmd.getPRSourceBranch(ctx, prCtx, prID)
	if err != nil {
		if cmd.Debug {
			fmt.Printf("DEBUG: Could not get PR source branch: %v\n", err)
		}
		return nil // Don't warn if we can't detect PR branch
	}

	// Compare branches
	if currentBranch != prBranch {
		return fmt.Errorf("Context may be inaccurate: You're on branch '%s' but PR #%d is from branch '%s'.\n"+
			"   Run: git checkout %s", currentBranch, prID, prBranch, prBranch)
	}

	return nil
}

func (cmd *ReportCmd) getCurrentGitBranch() (string, error) {
	gitCmd := exec.Command("git", "branch", "--show-current")
	output, err := gitCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (cmd *ReportCmd) getPRSourceBranch(ctx context.Context, prCtx *PRContext, prID int) (string, error) {
	// Use the PR API to get the source branch
	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return "", err
	}

	if pr.Source != nil && pr.Source.Branch != nil && pr.Source.Branch.Name != "" {
		return pr.Source.Branch.Name, nil
	}

	return "", fmt.Errorf("could not determine PR source branch")
}
