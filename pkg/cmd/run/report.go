package run

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/sonarcloud"
)

type ReportCmd struct {
	PipelineID        string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output            string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor           bool
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
	Debug             bool     `help:"Enable debug output for troubleshooting"`
	Workspace         string   `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository        string   `help:"Repository name (defaults to git remote)"`
}

func (cmd *ReportCmd) Run(ctx context.Context) error {
	runCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor)
	if err != nil {
		return err
	}

	if cmd.Workspace != "" {
		runCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		runCtx.Repository = cmd.Repository
	}

	if err := runCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	if strings.TrimSpace(cmd.PipelineID) == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	pipelineUUID, err := resolvePipelineUUID(ctx, runCtx, cmd.PipelineID)
	if err != nil {
		return err
	}

	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: Pipeline UUID: %s\n", pipelineUUID)
		fmt.Printf("DEBUG: Pipeline BuildNumber: %d\n", pipeline.BuildNumber)
		if pipeline.Target != nil {
			fmt.Printf("DEBUG: Pipeline Target Type: %s\n", pipeline.Target.Type)
			if pipeline.Target.PullRequestId != nil {
				fmt.Printf("DEBUG: Pipeline PR ID: %d\n", *pipeline.Target.PullRequestId)
			} else {
				fmt.Printf("DEBUG: Pipeline PR ID: nil\n")
			}
			if pipeline.Target.Commit != nil {
				fmt.Printf("DEBUG: Pipeline Commit: %s\n", pipeline.Target.Commit.Hash)
			}
		} else {
			fmt.Printf("DEBUG: Pipeline Target: nil\n")
		}
	}

	if cmd.Web || cmd.URL {
		return cmd.openSonarCloudDashboard(ctx, runCtx, pipeline)
	}

	sonarCloudService, err := cmd.createSonarCloudService(ctx, runCtx)
	if err != nil {
		return err
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: About to generate SonarCloud report\n")
	}

	return cmd.generateReport(ctx, runCtx, sonarCloudService, pipeline)
}

func (cmd *ReportCmd) createSonarCloudService(ctx context.Context, runCtx *RunContext) (*sonarcloud.Service, error) {
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

	service := sonarcloud.NewService(client, runCtx.Client)

	if err := service.TestConnection(ctx); err != nil {
		if scErr, ok := err.(*sonarcloud.SonarCloudError); ok {
			return nil, scErr
		}
		return nil, fmt.Errorf("failed to connect to SonarCloud: %w", err)
	}

	return service, nil
}

func (cmd *ReportCmd) openSonarCloudDashboard(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline) error {
	sonarCloudService, err := cmd.createSonarCloudService(ctx, runCtx)
	if err != nil {
		return err
	}

	filters := sonarcloud.FilterOptions{
		IncludeCoverage: false,
		IncludeIssues:   false,
	}

	report, err := sonarCloudService.GenerateReport(ctx, pipeline, runCtx.Workspace, runCtx.Repository, filters)
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

func (cmd *ReportCmd) generateReport(ctx context.Context, runCtx *RunContext, service *sonarcloud.Service, pipeline *api.Pipeline) error {
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

	report, err := service.GenerateReport(ctx, pipeline, runCtx.Workspace, runCtx.Repository, filters)
	if err != nil {
		return err
	}

	return cmd.formatOutput(runCtx, report, pipeline, filters)
}

func (cmd *ReportCmd) formatOutput(runCtx *RunContext, report *sonarcloud.Report, pipeline *api.Pipeline, filters sonarcloud.FilterOptions) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(runCtx, report, pipeline, filters)
	case "json":
		return runCtx.Formatter.Format(report)
	case "yaml":
		return runCtx.Formatter.Format(report)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ReportCmd) formatTable(runCtx *RunContext, report *sonarcloud.Report, pipeline *api.Pipeline, filters sonarcloud.FilterOptions) error {
	reportType := "SonarCloud Quality Report"
	if cmd.Coverage && !cmd.Issues {
		reportType = "Coverage Analysis"
	} else if cmd.Issues && !cmd.Coverage {
		reportType = "Issues Analysis"
	}

	fmt.Printf("=== %s: Pipeline #%d ===\n\n", reportType, pipeline.BuildNumber)

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
		cmd.formatOverviewSection(report, pipeline)
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

		for _, line := range linesToShow {
			marker := ""
			if line.IsNew {
				marker = " [NEW]"
			}

			code := line.Code
			if cmd.TruncateLines > 0 && len(code) > cmd.TruncateLines {
				code = code[:cmd.TruncateLines-3] + "..."
			}

			fmt.Printf("  Line %d: %s%s\n", line.Line, code, marker)
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

func (cmd *ReportCmd) formatOverviewSection(report *sonarcloud.Report, pipeline *api.Pipeline) {
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
