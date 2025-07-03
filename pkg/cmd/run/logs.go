package run

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/utils"
)

// LogsCmd handles the run logs command - the killer feature for 5x faster pipeline debugging
type LogsCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Step       string `help:"Show logs for specific step only"`
	ErrorsOnly bool   `help:"Extract and show errors only"`
	Follow     bool   `short:"f" help:"Follow live logs for running pipelines"`
	Output     string `short:"o" help:"Output format (text, json, yaml)" enum:"text,json,yaml" default:"text"`
	NoColor    bool   // NoColor is passed from global flag
	Context    int    `help:"Number of context lines around errors" default:"3"`
	Tests      bool   `short:"t" help:"Show test results and failures instead of raw logs"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the run logs command
func (cmd *LogsCmd) Run(ctx context.Context) error {
	// For logs, we handle text output specially - just use table format for the context
	outputFormat := cmd.Output
	if outputFormat == "text" {
		outputFormat = "table"  // Use table formatter for context, but we'll output raw text
	}
	
	// Create run context with authentication and configuration
	runCtx, err := NewRunContext(ctx, outputFormat, cmd.NoColor)
	if err != nil {
		return err
	}

	// Override workspace and repository if provided via flags
	if cmd.Workspace != "" {
		runCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		runCtx.Repository = cmd.Repository
	}

	// Validate workspace and repository are available
	if err := runCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	// Validate pipeline ID
	if strings.TrimSpace(cmd.PipelineID) == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	// Convert pipeline ID to UUID if it's a build number
	pipelineUUID, err := cmd.resolvePipelineUUID(ctx, runCtx)
	if err != nil {
		return err
	}

	// Get pipeline details to check status
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Follow mode for running pipelines
	if cmd.Follow {
		return cmd.followLogs(ctx, runCtx, pipeline)
	}

	// Static log viewing
	return cmd.viewLogs(ctx, runCtx, pipeline)
}

// resolvePipelineUUID converts build number to UUID or validates UUID (reused from view.go)
func (cmd *LogsCmd) resolvePipelineUUID(ctx context.Context, runCtx *RunContext) (string, error) {
	pipelineID := strings.TrimSpace(cmd.PipelineID)
	
	// If it's already a UUID (contains hyphens), return as-is
	if strings.Contains(pipelineID, "-") {
		return pipelineID, nil
	}

	// If it starts with #, remove it
	if strings.HasPrefix(pipelineID, "#") {
		pipelineID = pipelineID[1:]
	}

	// Try to parse as build number
	buildNumber, err := strconv.Atoi(pipelineID)
	if err != nil {
		return "", fmt.Errorf("invalid pipeline ID '%s'. Expected build number (e.g., 123, #123) or UUID", cmd.PipelineID)
	}

	// Search for pipeline by build number
	options := &api.PipelineListOptions{
		PageLen: 100, // Search recent pipelines
		Page:    1,
		Sort:    "-created_on",
	}

	result, err := runCtx.Client.Pipelines.ListPipelines(ctx, runCtx.Workspace, runCtx.Repository, options)
	if err != nil {
		return "", handlePipelineAPIError(err)
	}

	// Parse and search through pipelines
	pipelines, err := parsePipelineResults(result)
	if err != nil {
		return "", fmt.Errorf("failed to parse pipeline results: %w", err)
	}

	for _, pipeline := range pipelines {
		if pipeline.BuildNumber == buildNumber {
			return pipeline.UUID, nil
		}
	}

	return "", fmt.Errorf("pipeline with build number %d not found", buildNumber)
}

// displayStepInfo shows useful information about a step
func (cmd *LogsCmd) displayStepInfo(step *api.PipelineStep) {
	fmt.Printf("\n=== Step: %s ===\n", step.Name)
	
	if step.State != nil {
		status := step.State.Name
		if step.State.Result != nil && step.State.Result.Name != "" {
			status = step.State.Result.Name
		}
		fmt.Printf("Status: %s\n", status)
	}
	
	if step.StartedOn != nil {
		fmt.Printf("Started: %s\n", step.StartedOn.Format("2006-01-02 15:04:05"))
	}
	
	if step.CompletedOn != nil {
		fmt.Printf("Completed: %s\n", step.CompletedOn.Format("2006-01-02 15:04:05"))
	}
	
	if step.BuildSecondsUsed > 0 {
		fmt.Printf("Duration: %s\n", FormatDuration(step.BuildSecondsUsed))
	}
	
	if step.Image != nil {
		fmt.Printf("Image: %s\n", step.Image.Name)
	}
	
	// Show setup commands
	if len(step.SetupCommands) > 0 {
		fmt.Printf("\nSetup Commands:\n")
		for _, cmd := range step.SetupCommands {
			fmt.Printf("  - %s\n", cmd.Command)
		}
	}
	
	// Show script commands
	if len(step.ScriptCommands) > 0 {
		fmt.Printf("\nScript Commands:\n")
		for _, cmd := range step.ScriptCommands {
			fmt.Printf("  - %s\n", cmd.Command)
		}
	}
	
	fmt.Println()
}

// displayTestResults shows test results and failures for a step
func (cmd *LogsCmd) displayTestResults(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline, step *api.PipelineStep) {
	// Get test reports summary
	reports, err := runCtx.Client.Pipelines.GetStepTestReports(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
	if err != nil {
		fmt.Printf("No test reports available: %v\n", err)
		return
	}

	if len(reports) == 0 {
		fmt.Printf("No test reports found for this step\n")
		return
	}

	// Display test summary
	fmt.Printf("\n🧪 Test Reports Summary:\n")
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	
	for _, report := range reports {
		fmt.Printf("  Report: %s\n", report.Name)
		fmt.Printf("    Status: %s\n", report.Status)
		if report.Total > 0 {
			fmt.Printf("    Tests: %d total, %d passed, %d failed, %d skipped\n", 
				report.Total, report.Passed, report.Failed, report.Skipped)
			totalPassed += report.Passed
			totalFailed += report.Failed
			totalSkipped += report.Skipped
		}
		if report.Duration > 0 {
			fmt.Printf("    Duration: %.2fs\n", report.Duration)
		}
		fmt.Println()
	}

	// If there are failures, get detailed test cases
	if totalFailed > 0 {
		fmt.Printf("❌ Getting details for %d failed test(s)...\n\n", totalFailed)
		
		testCases, err := runCtx.Client.Pipelines.GetStepTestCases(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
		if err != nil {
			fmt.Printf("Could not get detailed test cases: %v\n", err)
			return
		}

		failedTests := 0
		for _, testCase := range testCases {
			if testCase.Status == "FAILED" || testCase.Result == "FAILED" {
				failedTests++
				fmt.Printf("❌ Test Failed: %s\n", testCase.Name)
				if testCase.ClassName != "" {
					fmt.Printf("   Class: %s\n", testCase.ClassName)
				}
				if testCase.TestSuite != "" {
					fmt.Printf("   Suite: %s\n", testCase.TestSuite)
				}
				if testCase.Duration > 0 {
					fmt.Printf("   Duration: %.2fs\n", testCase.Duration)
				}
				if testCase.Message != "" {
					fmt.Printf("   Message: %s\n", testCase.Message)
				}
				if testCase.Stacktrace != "" {
					fmt.Printf("   Stacktrace:\n%s\n", testCase.Stacktrace)
				}

				// Try to get more detailed reasons
				reasons, err := runCtx.Client.Pipelines.GetTestCaseReasons(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID, testCase.UUID)
				if err == nil && len(reasons) > 0 {
					fmt.Printf("   Detailed Output:\n")
					for _, reason := range reasons {
						if reason.Message != "" {
							fmt.Printf("     %s\n", reason.Message)
						}
						if reason.Output != "" {
							fmt.Printf("     %s\n", reason.Output)
						}
					}
				}
				fmt.Println()
			}
		}

		if failedTests == 0 {
			fmt.Printf("Could not find detailed information for failed tests\n")
		}
	} else {
		fmt.Printf("✅ All tests passed!\n")
	}
}

// viewLogs displays logs for a completed or stopped pipeline
func (cmd *LogsCmd) viewLogs(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline) error {
	// Get pipeline steps
	steps, err := runCtx.Client.Pipelines.GetPipelineSteps(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Filter steps if specific step requested
	filteredSteps := steps
	if cmd.Step != "" {
		filteredSteps = cmd.filterStepsByName(steps, cmd.Step)
		if len(filteredSteps) == 0 {
			return fmt.Errorf("step '%s' not found. Available steps: %s", cmd.Step, cmd.getAvailableStepNames(steps))
		}
	}

	// Process logs for each step
	allResults := make([]*utils.LogAnalysisResult, 0, len(filteredSteps))
	
	for _, step := range filteredSteps {
		// Try to get step logs first, or use --tests flag to show test results
		if cmd.Tests {
			// Show test results instead of logs
			if cmd.Output == "text" {
				cmd.displayStepInfo(step)
				cmd.displayTestResults(ctx, runCtx, pipeline, step)
			}
			// Create a dummy result for this step
			result := &utils.LogAnalysisResult{
				TotalLines:   0,
				ErrorCount:   0,
				WarningCount: 0,
				Errors:       []utils.ExtractedError{},
				Summary:      make(map[string]int),
				ProcessedAt:  time.Now(),
			}
			allResults = append(allResults, result)
			continue
		}

		logReader, err := runCtx.Client.Pipelines.GetStepLogs(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
		if err != nil {
			if cmd.Output == "text" {
				cmd.displayStepInfo(step)
				fmt.Printf("Note: Raw logs not available through API for step '%s': %v\n", step.Name, err)
				
				// Try to show test results as fallback
				fmt.Printf("Checking for test results...\n")
				cmd.displayTestResults(ctx, runCtx, pipeline, step)
			}
			// Create a dummy result for this step
			result := &utils.LogAnalysisResult{
				TotalLines:   0,
				ErrorCount:   0,
				WarningCount: 0,
				Errors:       []utils.ExtractedError{},
				Summary:      make(map[string]int),
				ProcessedAt:  time.Now(),
			}
			allResults = append(allResults, result)
			continue
		}
		defer logReader.Close()

		// Analyze logs with error detection
		parser := utils.NewLogParser()
		parser.SetContextLines(cmd.Context)
		
		result, err := parser.AnalyzeLog(logReader, step.Name)
		if err != nil {
			fmt.Printf("Warning: Could not analyze logs for step '%s': %v\n", step.Name, err)
			continue
		}

		// Apply error-only filter if requested
		if cmd.ErrorsOnly {
			result = parser.FilterErrorsOnly(result)
		}

		allResults = append(allResults, result)
	}

	// Format and display output
	return cmd.formatOutput(runCtx, pipeline, filteredSteps, allResults)
}

// followLogs provides real-time log streaming for running pipelines
func (cmd *LogsCmd) followLogs(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline) error {
	// Check if pipeline is in a state that can be followed
	if pipeline.State == nil || (pipeline.State.Name != "IN_PROGRESS" && pipeline.State.Name != "PENDING") {
		fmt.Printf("Pipeline #%d is %s - following is only available for running pipelines\n", 
			pipeline.BuildNumber, pipeline.State.Name)
		// Fall back to static view
		return cmd.viewLogs(ctx, runCtx, pipeline)
	}

	fmt.Printf("Following logs for pipeline #%d (Ctrl+C to exit)...\n\n", pipeline.BuildNumber)

	// Create log parser for real-time analysis
	parser := utils.NewLogParser()
	parser.SetContextLines(cmd.Context)

	// Follow loop - check for new steps and stream their logs
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	seenSteps := make(map[string]bool)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Get current pipeline steps
			steps, err := runCtx.Client.Pipelines.GetPipelineSteps(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID)
			if err != nil {
				fmt.Printf("Error getting pipeline steps: %v\n", err)
				continue
			}

			// Process new or updated steps
			for _, step := range steps {
				if cmd.Step != "" && !cmd.matchesStepName(step.Name, cmd.Step) {
					continue
				}

				// Check if this is a new step or step we should re-process
				stepKey := fmt.Sprintf("%s-%s", step.UUID, step.State.Name)
				if seenSteps[stepKey] {
					continue
				}
				seenSteps[stepKey] = true

				// Display step header
				fmt.Printf("=== Step: %s (%s) ===\n", step.Name, step.State.Name)

				// Stream logs for this step
				if err := cmd.streamStepLogs(ctx, runCtx, pipeline, step, parser); err != nil {
					fmt.Printf("Error streaming logs for step '%s': %v\n", step.Name, err)
				}
			}

			// Check if pipeline completed
			updatedPipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID)
			if err != nil {
				fmt.Printf("Error checking pipeline status: %v\n", err)
				continue
			}

			if updatedPipeline.State != nil && 
			   updatedPipeline.State.Name != "IN_PROGRESS" && 
			   updatedPipeline.State.Name != "PENDING" {
				fmt.Printf("\n🏁 Pipeline completed with status: %s\n", updatedPipeline.State.Name)
				return nil
			}
		}
	}
}

// streamStepLogs streams logs for a single step with real-time error analysis
func (cmd *LogsCmd) streamStepLogs(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline, step *api.PipelineStep, parser *utils.LogParser) error {
	// Use streaming API for real-time logs
	logChan, errChan := runCtx.Client.Pipelines.StreamStepLogs(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)

	lineNumber := 0
	var logLines []string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			if err != nil {
				return err
			}
			// Channel closed, process accumulated logs
			return cmd.processAccumulatedLogs(logLines, step.Name, parser)
		case line, ok := <-logChan:
			if !ok {
				// Channel closed, process accumulated logs
				return cmd.processAccumulatedLogs(logLines, step.Name, parser)
			}
			
			lineNumber++
			logLines = append(logLines, line)

			// For real-time display, show the line immediately unless errors-only mode
			if !cmd.ErrorsOnly {
				timestamp := time.Now().Format("15:04:05")
				fmt.Printf("[%s] %s\n", timestamp, line)
			} else {
				// In errors-only mode, analyze each line for errors
				if cmd.containsError(line, parser) {
					timestamp := time.Now().Format("15:04:05")
					fmt.Printf("[%s] ❌ %s\n", timestamp, line)
				}
			}
		}
	}
}

// processAccumulatedLogs analyzes all accumulated log lines for errors
func (cmd *LogsCmd) processAccumulatedLogs(logLines []string, stepName string, parser *utils.LogParser) error {
	if cmd.ErrorsOnly && len(logLines) > 0 {
		// Analyze all logs for comprehensive error detection
		logContent := strings.Join(logLines, "\n")
		result, err := parser.AnalyzeLog(strings.NewReader(logContent), stepName)
		if err != nil {
			return err
		}

		filtered := parser.FilterErrorsOnly(result)
		if len(filtered.Errors) > 0 {
			fmt.Printf("\n📋 Error Summary for %s:\n", stepName)
			for _, logError := range filtered.Errors {
				fmt.Printf("  Line %d: %s\n", logError.Line, logError.Content)
			}
		}
	}
	return nil
}

// containsError quickly checks if a log line contains error patterns
func (cmd *LogsCmd) containsError(line string, parser *utils.LogParser) bool {
	for _, pattern := range parser.ErrorPatterns {
		if pattern.Severity == "error" || pattern.Severity == "critical" {
			if pattern.Regex.MatchString(line) {
				return true
			}
		}
	}
	return false
}

// filterStepsByName filters steps by name with fuzzy matching
func (cmd *LogsCmd) filterStepsByName(steps []*api.PipelineStep, stepName string) []*api.PipelineStep {
	var filtered []*api.PipelineStep
	
	for _, step := range steps {
		if cmd.matchesStepName(step.Name, stepName) {
			filtered = append(filtered, step)
		}
	}
	
	return filtered
}

// matchesStepName checks if a step name matches the requested name (with fuzzy matching)
func (cmd *LogsCmd) matchesStepName(stepName, requestedName string) bool {
	stepNameLower := strings.ToLower(stepName)
	requestedLower := strings.ToLower(requestedName)
	
	// Exact match
	if stepNameLower == requestedLower {
		return true
	}
	
	// Contains match
	if strings.Contains(stepNameLower, requestedLower) {
		return true
	}
	
	// Prefix match
	if strings.HasPrefix(stepNameLower, requestedLower) {
		return true
	}
	
	return false
}

// getAvailableStepNames returns a comma-separated list of available step names
func (cmd *LogsCmd) getAvailableStepNames(steps []*api.PipelineStep) string {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name
	}
	return strings.Join(names, ", ")
}

// formatOutput formats and displays the log analysis results
func (cmd *LogsCmd) formatOutput(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	switch cmd.Output {
	case "text":
		return cmd.formatText(runCtx, pipeline, steps, results)
	case "json":
		return cmd.formatJSON(runCtx, pipeline, steps, results)
	case "yaml":
		return cmd.formatYAML(runCtx, pipeline, steps, results)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

// formatText formats logs as human-readable text with error highlighting
func (cmd *LogsCmd) formatText(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	// Pipeline header
	status := "UNKNOWN"
	if pipeline.State != nil {
		status = pipeline.State.Name
	}

	branch := ""
	if pipeline.Target != nil && pipeline.Target.RefName != "" {
		branch = pipeline.Target.RefName
	}

	fmt.Printf("=== Pipeline #%d: %s (%s) ===\n", pipeline.BuildNumber, branch, status)
	fmt.Println()

	// Process each step's logs
	for i, result := range results {
		if i < len(steps) {
			step := steps[i]
			stepStatus := "UNKNOWN"
			if step.State != nil {
				stepStatus = step.State.Name
			}

			fmt.Printf("=== Step: %s (%s) ===\n", step.Name, stepStatus)
			
			if cmd.ErrorsOnly {
				// Show only errors with context
				if len(result.Errors) > 0 {
					fmt.Printf("❌ Found %d error(s):\n\n", len(result.Errors))
					for _, logError := range result.Errors {
						fmt.Printf("Line %d [%s]: %s\n", logError.Line, logError.Category, logError.Content)
						if len(logError.Context) > 0 {
							fmt.Printf("Context:\n")
							for _, contextLine := range logError.Context {
								fmt.Printf("  %s\n", contextLine)
							}
						}
						fmt.Println()
					}
				} else {
					fmt.Printf("✅ No errors found\n\n")
				}
			} else {
				// Show summary with error highlights
				fmt.Printf("Total lines: %d, Errors: %d, Warnings: %d\n", 
					result.TotalLines, result.ErrorCount, result.WarningCount)
				
				if len(result.Errors) > 0 {
					fmt.Printf("\n❌ Errors found:\n")
					for _, logError := range result.Errors {
						if logError.Severity == "error" || logError.Severity == "critical" {
							fmt.Printf("  Line %d [%s]: %s\n", logError.Line, logError.Category, logError.Content)
						}
					}
				}
				fmt.Println()
			}
		}
	}

	// Summary
	if len(results) > 0 && !cmd.ErrorsOnly {
		totalErrors := 0
		totalWarnings := 0
		for _, result := range results {
			totalErrors += result.ErrorCount
			totalWarnings += result.WarningCount
		}
		
		fmt.Printf("📊 Overall Summary: %d error(s), %d warning(s) across %d step(s)\n", 
			totalErrors, totalWarnings, len(results))
	}

	return nil
}

// formatJSON formats logs as structured JSON for AI/automation
func (cmd *LogsCmd) formatJSON(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	output := map[string]interface{}{
		"pipeline":    pipeline,
		"steps":       steps,
		"log_analysis": results,
		"summary": map[string]interface{}{
			"total_steps": len(steps),
			"total_errors": func() int {
				total := 0
				for _, r := range results {
					total += r.ErrorCount
				}
				return total
			}(),
			"total_warnings": func() int {
				total := 0
				for _, r := range results {
					total += r.WarningCount
				}
				return total
			}(),
			"analyzed_at": time.Now(),
		},
	}

	return runCtx.Formatter.Format(output)
}

// formatYAML formats logs as YAML for alternative structured output
func (cmd *LogsCmd) formatYAML(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	output := map[string]interface{}{
		"pipeline":     pipeline,
		"steps":        steps,
		"log_analysis": results,
		"summary": map[string]interface{}{
			"total_steps": len(steps),
			"total_errors": func() int {
				total := 0
				for _, r := range results {
					total += r.ErrorCount
				}
				return total
			}(),
			"total_warnings": func() int {
				total := 0
				for _, r := range results {
					total += r.WarningCount
				}
				return total
			}(),
			"analyzed_at": time.Now(),
		},
	}

	return runCtx.Formatter.Format(output)
}