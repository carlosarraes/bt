package run

import (
	"context"
	"fmt"
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
		return cmd.openSonarCloudDashboard(ctx, runCtx)
	}

	sonarCloudService, err := shared.CreateSonarCloudService(ctx, runCtx)
	if err != nil {
		return err
	}

	if cmd.Debug {
		fmt.Printf("DEBUG: About to generate SonarCloud report\n")
	}

	return cmd.generateReport(ctx, runCtx, sonarCloudService, pipeline)
}

func (cmd *ReportCmd) openSonarCloudDashboard(ctx context.Context, runCtx *RunContext) error {
	sonarCloudService, err := shared.CreateSonarCloudService(ctx, runCtx)
	if err != nil {
		return err
	}

	projectKey, err := sonarCloudService.DiscoverProjectKey(ctx, runCtx.Workspace, runCtx.Repository, "")
	if err != nil {
		return fmt.Errorf("failed to discover SonarCloud project: %w", err)
	}

	url := fmt.Sprintf("https://sonarcloud.io/project/overview?id=%s", projectKey)

	if cmd.URL {
		fmt.Println(url)
		return nil
	}

	return shared.LaunchBrowser(url)
}

func (cmd *ReportCmd) generateReport(ctx context.Context, runCtx *RunContext, service *sonarcloud.Service, pipeline *api.Pipeline) error {
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

func (cmd *ReportCmd) formatter() *shared.ReportFormatter {
	return &shared.ReportFormatter{
		ShowAllLines:  cmd.ShowAllLines,
		LinesPerFile:  cmd.LinesPerFile,
		TruncateLines: cmd.TruncateLines,
	}
}

func (cmd *ReportCmd) formatTable(runCtx *RunContext, report *sonarcloud.Report, pipeline *api.Pipeline, filters sonarcloud.FilterOptions) error {
	reportType := "SonarCloud Quality Report"
	if cmd.Coverage && !cmd.Issues && !cmd.Duplications {
		reportType = "Coverage Analysis"
	} else if cmd.Issues && !cmd.Coverage && !cmd.Duplications {
		reportType = "Issues Analysis"
	} else if cmd.Duplications && !cmd.Coverage && !cmd.Issues {
		reportType = "Duplications Analysis"
	}

	fmt.Printf("=== %s: Pipeline #%d ===\n\n", reportType, pipeline.BuildNumber)

	f := cmd.formatter()

	f.FormatQualityGateHeader(report)

	if filters.IncludeCoverage && report.Coverage != nil && report.Coverage.Available {
		f.FormatCoverageSection(report.Coverage, filters)
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
