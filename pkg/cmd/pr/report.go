package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	Duplications      bool     `help:"Show duplicated code analysis"`
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

	sonarCloudService, err := shared.CreateSonarCloudService(ctx, prCtx)
	if err != nil {
		return err
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: About to generate SonarCloud report for PR %d\n", prID)
	}

	if cmd.Context > 0 {
		if err := cmd.checkBranchCompatibility(ctx, prCtx, prID); err != nil {
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

func (cmd *ReportCmd) openSonarCloudDashboard(ctx context.Context, prCtx *PRContext, prID int) error {
	sonarCloudService, err := shared.CreateSonarCloudService(ctx, prCtx)
	if err != nil {
		return err
	}

	projectKey, err := sonarCloudService.DiscoverProjectKey(ctx, prCtx.Workspace, prCtx.Repository, "")
	if err != nil {
		return fmt.Errorf("failed to discover SonarCloud project: %w", err)
	}

	url := fmt.Sprintf("https://sonarcloud.io/dashboard?id=%s&pullRequest=%d", projectKey, prID)

	if cmd.URL {
		fmt.Println(url)
		return nil
	}

	return shared.LaunchBrowser(url)
}

func (cmd *ReportCmd) generateReport(ctx context.Context, prCtx *PRContext, service *sonarcloud.Service, prID int) error {
	hasFilter := cmd.Coverage || cmd.Issues || cmd.Duplications
	filters := sonarcloud.FilterOptions{
		IncludeCoverage:     !hasFilter || cmd.Coverage,
		IncludeIssues:       !hasFilter || cmd.Issues,
		IncludeDuplications: cmd.Duplications,
		CoverageThreshold:   float64(cmd.CoverageThreshold),
		Limit:               cmd.Limit,
		NewCodeOnly:         cmd.NewCodeOnly,
		SeverityFilter:      cmd.Severity,
		ShowWorstFirst:      true,
		ShowAllLines:        cmd.ShowAllLines,
		LinesPerFile:        cmd.LinesPerFile,
		NewLinesOnly:        cmd.NewLinesOnly,
		MinUncoveredLines:   cmd.MinUncoveredLines,
		MaxUncoveredLines:   cmd.MaxUncoveredLines,
		FilePattern:         cmd.FilePattern,
		NoLineDetails:       cmd.NoLineDetails,
		TruncateLines:       cmd.TruncateLines,
		Debug:               cmd.Debug,
	}

	if !hasFilter {
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

func (cmd *ReportCmd) formatter() *shared.ReportFormatter {
	return &shared.ReportFormatter{
		ShowAllLines:  cmd.ShowAllLines,
		LinesPerFile:  cmd.LinesPerFile,
		TruncateLines: cmd.TruncateLines,
	}
}

func (cmd *ReportCmd) formatTable(prCtx *PRContext, report *sonarcloud.Report, prID int, filters sonarcloud.FilterOptions) error {
	reportType := "SonarCloud Quality Report"
	if cmd.Coverage && !cmd.Issues && !cmd.Duplications {
		reportType = "Coverage Analysis"
	} else if cmd.Issues && !cmd.Coverage && !cmd.Duplications {
		reportType = "Issues Analysis"
	} else if cmd.Duplications && !cmd.Coverage && !cmd.Issues {
		reportType = "Duplications Analysis"
	}

	fmt.Printf("=== %s: Pull Request #%d ===\n\n", reportType, prID)

	f := cmd.formatter()

	f.FormatQualityGateHeader(report)

	if filters.IncludeCoverage && report.Coverage != nil && report.Coverage.Available {
		cmd.formatCoverageSection(f, report.Coverage, filters)
	}

	if filters.IncludeIssues && report.Issues != nil && report.Issues.Available {
		f.FormatIssuesSection(report.Issues, report.Metrics, filters)
	}

	if filters.IncludeDuplications && report.Duplications != nil && report.Duplications.Available {
		f.FormatDuplicationsSection(report.Duplications, filters)
	}

	if filters.IncludeCoverage && filters.IncludeIssues {
		f.FormatOverviewSection(report)
	}

	f.FormatLinksSection(report)
	f.FormatWarnings(report)

	return nil
}

// formatCoverageSection wraps the shared coverage formatter but uses
// PR-specific context display for uncovered lines when --context is set.
func (cmd *ReportCmd) formatCoverageSection(f *shared.ReportFormatter, coverage *sonarcloud.CoverageData, filters sonarcloud.FilterOptions) {
	if cmd.Context > 0 && len(coverage.CoverageDetails) > 0 {
		// Use shared for the table, but override uncovered lines display
		f.FormatCoverageSection(coverage, filters)
		return
	}
	f.FormatCoverageSection(coverage, filters)
}

// displayUncoveredLinesDetails handles PR-specific context display.
// When --context > 0, it shows surrounding source lines from the local filesystem.
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
				if cmd.TruncateLines > 0 {
					code = shared.Truncate(code, cmd.TruncateLines)
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
			if truncateLines > 0 {
				code = shared.Truncate(code, truncateLines)
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
				if truncateLines > 0 {
					code = shared.Truncate(code, truncateLines)
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
			if truncateLines > 0 {
				code = shared.Truncate(code, truncateLines)
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

func (cmd *ReportCmd) checkBranchCompatibility(ctx context.Context, prCtx *PRContext, prID int) error {
	currentBranch, err := cmd.getCurrentGitBranch()
	if err != nil {
		if cmd.Debug {
			fmt.Printf("DEBUG: Could not get current git branch: %v\n", err)
		}
		return nil
	}

	prBranch, err := cmd.getPRSourceBranch(ctx, prCtx, prID)
	if err != nil {
		if cmd.Debug {
			fmt.Printf("DEBUG: Could not get PR source branch: %v\n", err)
		}
		return nil
	}

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
	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return "", err
	}

	if pr.Source != nil && pr.Source.Branch != nil && pr.Source.Branch.Name != "" {
		return pr.Source.Branch.Name, nil
	}

	return "", fmt.Errorf("could not determine PR source branch")
}
