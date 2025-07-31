package run

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/utils"
)

// ViewCmd handles the run view command
type ViewCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Watch      bool   `short:"w" help:"Watch for live updates (running pipelines only)"`
	Log        bool   `help:"View full logs for all steps"`
	LogFailed  bool   `name:"log-failed" help:"View logs only for failed steps (last 100 lines)"`
	FullOutput bool   `name:"full-output" help:"Show complete logs (use with --log-failed for full failure logs)"`
	Tests      bool   `short:"t" help:"Show test results and failures"`
	Step       string `help:"View specific step only"`
	Web        bool   `help:"Open pipeline in browser"`
	URL        bool   `help:"Print pipeline URL instead of opening in browser (use with --web)"`
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the run view command
func (cmd *ViewCmd) Run(ctx context.Context) error {
	// Create run context with authentication and configuration
	runCtx, err := NewRunContext(ctx, cmd.Output, cmd.NoColor)
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

	// Watch mode for running pipelines
	if cmd.Watch {
		return cmd.watchPipeline(ctx, runCtx, pipelineUUID)
	}

	// Handle different view modes based on flags
	if cmd.Web {
		return cmd.openInBrowser(ctx, runCtx, pipelineUUID)
	}
	
	if cmd.Log || cmd.LogFailed || cmd.Tests {
		return cmd.viewLogs(ctx, runCtx, pipelineUUID)
	}
	
	// Default: show pipeline summary
	return cmd.viewPipeline(ctx, runCtx, pipelineUUID)
}

// resolvePipelineUUID converts build number to UUID or validates UUID
func (cmd *ViewCmd) resolvePipelineUUID(ctx context.Context, runCtx *RunContext) (string, error) {
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

// viewPipeline displays detailed information about a single pipeline
func (cmd *ViewCmd) viewPipeline(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	// Get pipeline details
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Get pipeline steps
	steps, err := runCtx.Client.Pipelines.GetPipelineSteps(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Format output
	return cmd.formatOutput(runCtx, pipeline, steps)
}

// watchPipeline monitors a running pipeline for live updates
func (cmd *ViewCmd) watchPipeline(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	// First, check if pipeline is running
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Check if pipeline is in a state that can be watched
	if pipeline.State == nil || (pipeline.State.Name != "IN_PROGRESS" && pipeline.State.Name != "PENDING") {
		fmt.Printf("Pipeline #%d is %s - watching is only available for running pipelines\n", 
			pipeline.BuildNumber, pipeline.State.Name)
		// Show current state and exit
		return cmd.viewPipeline(ctx, runCtx, pipelineUUID)
	}

	fmt.Printf("Watching pipeline #%d (Ctrl+C to exit)...\n\n", pipeline.BuildNumber)

	// Watch loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Show initial state
	if err := cmd.displayPipelineUpdate(ctx, runCtx, pipelineUUID); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Get updated pipeline status
			updatedPipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
			if err != nil {
				return handlePipelineAPIError(err)
			}

			// Display update
			if err := cmd.displayPipelineUpdate(ctx, runCtx, pipelineUUID); err != nil {
				return err
			}

			// Check if pipeline completed
			if updatedPipeline.State != nil && 
			   updatedPipeline.State.Name != "IN_PROGRESS" && 
			   updatedPipeline.State.Name != "PENDING" {
				fmt.Printf("\n🏁 Pipeline completed with status: %s\n", updatedPipeline.State.Name)
				return nil
			}
		}
	}
}

// displayPipelineUpdate shows a compact update during watch mode
func (cmd *ViewCmd) displayPipelineUpdate(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	// Get current pipeline state
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return err
	}

	// Get current steps
	steps, err := runCtx.Client.Pipelines.GetPipelineSteps(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return err
	}

	// Display compact status
	status := "UNKNOWN"
	if pipeline.State != nil {
		// Use the result if available (SUCCESSFUL, FAILED, etc.)
		if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
			status = pipeline.State.Result.Name
		} else {
			// Fall back to state name (PENDING, IN_PROGRESS, COMPLETED, etc.)
			status = pipeline.State.Name
		}
	}

	duration := ""
	if pipeline.BuildSecondsUsed > 0 {
		duration = FormatDuration(pipeline.BuildSecondsUsed)
	}

	fmt.Printf("[%s] Pipeline #%d: %s", 
		time.Now().Format("15:04:05"), pipeline.BuildNumber, status)
	
	if duration != "" {
		fmt.Printf(" (%s)", duration)
	}
	fmt.Println()

	// Show step progress
	for _, step := range steps {
		stepStatus := "UNKNOWN"
		if step.State != nil {
			stepStatus = step.State.Name
		}

		stepDuration := ""
		if step.BuildSecondsUsed > 0 {
			stepDuration = FormatDuration(step.BuildSecondsUsed)
		}

		statusIcon := cmd.getStatusIcon(stepStatus)
		fmt.Printf("  %s %-15s %s", statusIcon, step.Name, stepStatus)
		
		if stepDuration != "" {
			fmt.Printf(" (%s)", stepDuration)
		}
		fmt.Println()
	}
	
	fmt.Println("---")
	return nil
}

// formatOutput formats and displays the pipeline and step information
func (cmd *ViewCmd) formatOutput(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(runCtx, pipeline, steps)
	case "json":
		return cmd.formatJSON(runCtx, pipeline, steps)
	case "yaml":
		return cmd.formatYAML(runCtx, pipeline, steps)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

// formatTable formats the pipeline information as a detailed table
func (cmd *ViewCmd) formatTable(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep) error {
	// Pipeline header
	status := "UNKNOWN"
	if pipeline.State != nil {
		// Use the result if available (SUCCESSFUL, FAILED, etc.)
		if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
			status = pipeline.State.Result.Name
		} else {
			// Fall back to state name (PENDING, IN_PROGRESS, COMPLETED, etc.)
			status = pipeline.State.Name
		}
	}

	branch := ""
	if pipeline.Target != nil && pipeline.Target.RefName != "" {
		branch = pipeline.Target.RefName
	}

	fmt.Printf("Pipeline #%d: %s (%s)\n", pipeline.BuildNumber, branch, status)
	fmt.Println(strings.Repeat("━", 60))
	fmt.Println()

	// Repository information
	if pipeline.Repository != nil {
		fmt.Printf("Repository:  %s\n", pipeline.Repository.FullName)
	}
	
	if branch != "" {
		fmt.Printf("Branch:      %s\n", branch)
	}

	// Commit information
	if pipeline.Target != nil && pipeline.Target.Commit != nil {
		commit := pipeline.Target.Commit
		commitMsg := commit.Message
		if len(commitMsg) > 50 {
			commitMsg = commitMsg[:47] + "..."
		}
		fmt.Printf("Commit:      %s (%s)\n", commit.Hash[:8], commitMsg)
	}

	// Trigger information
	if pipeline.Trigger != nil {
		fmt.Printf("Trigger:     %s\n", pipeline.Trigger.Name)
	}

	// Timing information
	if pipeline.CreatedOn != nil {
		fmt.Printf("Started:     %s\n", pipeline.CreatedOn.Format("2006-01-02 15:04:05"))
	}

	if pipeline.CompletedOn != nil {
		fmt.Printf("Completed:   %s\n", pipeline.CompletedOn.Format("2006-01-02 15:04:05"))
	}

	if pipeline.BuildSecondsUsed > 0 {
		fmt.Printf("Duration:    %s\n", FormatDuration(pipeline.BuildSecondsUsed))
	}

	// Variables (if any)
	if len(pipeline.Variables) > 0 {
		fmt.Println("\nVariables:")
		for _, variable := range pipeline.Variables {
			if variable.Secured {
				fmt.Printf("  %s: [SECURED]\n", variable.Key)
			} else {
				fmt.Printf("  %s: %s\n", variable.Key, variable.Value)
			}
		}
	}

	// Steps section
	if len(steps) > 0 {
		fmt.Println("\nSteps:")
		for _, step := range steps {
			stepStatus := "UNKNOWN"
			if step.State != nil {
				stepStatus = step.State.Name
			}

			stepDuration := ""
			if step.BuildSecondsUsed > 0 {
				stepDuration = FormatDuration(step.BuildSecondsUsed)
			}

			statusIcon := cmd.getStatusIcon(stepStatus)
			
			fmt.Printf("  %s %-15s", statusIcon, step.Name)
			
			if stepDuration != "" {
				fmt.Printf(" %8s", stepDuration)
			}
			
			fmt.Printf("   %s\n", stepStatus)
		}
	}

	return nil
}

// formatJSON formats the pipeline and steps as JSON
func (cmd *ViewCmd) formatJSON(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep) error {
	output := map[string]interface{}{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return runCtx.Formatter.Format(output)
}

// formatYAML formats the pipeline and steps as YAML
func (cmd *ViewCmd) formatYAML(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep) error {
	output := map[string]interface{}{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return runCtx.Formatter.Format(output)
}

// getStatusIcon returns an appropriate icon for the step status
func (cmd *ViewCmd) getStatusIcon(status string) string {
	switch status {
	case "SUCCESSFUL":
		return "✓"
	case "FAILED":
		return "✗"
	case "ERROR":
		return "✗"
	case "STOPPED":
		return "⏸"
	case "IN_PROGRESS":
		return "⚙"
	case "PENDING":
		return "⏳"
	default:
		return "?"
	}
}

func (cmd *ViewCmd) openInBrowser(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	url := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", 
		runCtx.Workspace, runCtx.Repository, pipeline.BuildNumber)
	
	if cmd.URL {
		fmt.Println(url)
		return nil
	}
	
	return cmd.launchBrowser(url)
}

func (cmd *ViewCmd) launchBrowser(url string) error {
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

// viewLogs displays logs and test results for pipeline steps
func (cmd *ViewCmd) viewLogs(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	// Get pipeline details
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

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

	// Filter to failed steps only if --log-failed
	if cmd.LogFailed {
		fmt.Printf("Looking for failed steps in pipeline...\n")
		var failedSteps []*api.PipelineStep
		for _, step := range filteredSteps {
			stepStatus := "unknown"
			stepResult := "unknown"
			if step.State != nil {
				stepStatus = step.State.Name
				if step.State.Result != nil {
					stepResult = step.State.Result.Name
				}
			}
			fmt.Printf("  Step: '%s' - Status: %s, Result: %s, UUID: %s\n", step.Name, stepStatus, stepResult, step.UUID)
			
			if step.State != nil && (step.State.Name == "FAILED" || (step.State.Result != nil && step.State.Result.Name == "FAILED")) {
				failedSteps = append(failedSteps, step)
				fmt.Printf("    ✓ This step failed - will attempt to get logs\n")
			}
		}
		filteredSteps = failedSteps
		
		if len(filteredSteps) == 0 {
			fmt.Printf("No failed steps found in this pipeline\n")
			return nil
		}
		
		fmt.Printf("\nFound %d failed step(s). Attempting to retrieve logs...\n\n", len(filteredSteps))
	}

	// Process logs for each step
	allResults := make([]*utils.LogAnalysisResult, 0, len(filteredSteps))
	
	for _, step := range filteredSteps {
		if cmd.Tests {
			// Show test results instead of logs
			if cmd.Output == "table" {
				cmd.displayStepInfo(step)
				cmd.displayTestResults(ctx, runCtx, pipeline, step)
			}
		} else {
			// Try to get step logs
			fmt.Printf("Attempting to fetch logs for step: %s (UUID: %s)\n", step.Name, step.UUID)
			fmt.Printf("Pipeline UUID: %s\n", pipeline.UUID)
			fmt.Printf("Repository: %s/%s\n", runCtx.Workspace, runCtx.Repository)
			
			logReader, err := runCtx.Client.Pipelines.GetStepLogs(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
			if err != nil {
				if cmd.Output == "table" {
					cmd.displayStepInfo(step)
					fmt.Printf("Logs not available: %v\n", err)
					
					// Try to show test results as fallback
					cmd.displayTestResults(ctx, runCtx, pipeline, step)
				}
			} else {
				defer logReader.Close()
				
				fmt.Printf("✅ Successfully retrieved logs for step: %s\n", step.Name)
				
				// Read all log content
				logContent, err := io.ReadAll(logReader)
				if err != nil {
					fmt.Printf("Error reading logs: %v\n", err)
					continue
				}
				
				logLines := strings.Split(string(logContent), "\n")
				
				// Show appropriate output based on flags
				if cmd.LogFailed && !cmd.FullOutput {
					// Show only last 100 lines for failed steps (unless --full-output)
					startLine := len(logLines) - 100
					if startLine < 0 {
						startLine = 0
					}
					
					fmt.Printf("Showing last %d lines (use --full-output for complete logs)\n", len(logLines)-startLine)
					fmt.Println(strings.Repeat("=", 80))
					
					for i := startLine; i < len(logLines); i++ {
						fmt.Println(logLines[i])
					}
				} else {
					// Show full logs
					fmt.Printf("Showing all %d lines\n", len(logLines))
					fmt.Println(strings.Repeat("=", 80))
					
					for _, line := range logLines {
						fmt.Println(line)
					}
				}
				
				fmt.Println(strings.Repeat("=", 80))
				return nil // Exit after showing logs
			}
		}
		
		// Create a dummy result for consistency
		if len(allResults) <= len(filteredSteps)-1 {
			result := &utils.LogAnalysisResult{
				TotalLines:   0,
				ErrorCount:   0,
				WarningCount: 0,
				Errors:       []utils.ExtractedError{},
				Summary:      make(map[string]int),
				ProcessedAt:  time.Now(),
			}
			allResults = append(allResults, result)
		}
	}

	// Format and display output
	return cmd.formatLogOutput(runCtx, pipeline, filteredSteps, allResults)
}

// Helper methods copied from logs.go
func (cmd *ViewCmd) displayStepInfo(step *api.PipelineStep) {
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

func (cmd *ViewCmd) displayTestResults(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline, step *api.PipelineStep) {
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

func (cmd *ViewCmd) filterStepsByName(steps []*api.PipelineStep, stepName string) []*api.PipelineStep {
	var filtered []*api.PipelineStep
	
	for _, step := range steps {
		if cmd.matchesStepName(step.Name, stepName) {
			filtered = append(filtered, step)
		}
	}
	
	return filtered
}

func (cmd *ViewCmd) matchesStepName(stepName, requestedName string) bool {
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

func (cmd *ViewCmd) getAvailableStepNames(steps []*api.PipelineStep) string {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name
	}
	return strings.Join(names, ", ")
}

func (cmd *ViewCmd) formatLogOutput(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTextLogs(runCtx, pipeline, steps, results)
	case "json":
		return cmd.formatJSONLogs(runCtx, pipeline, steps, results)
	case "yaml":
		return cmd.formatYAMLLogs(runCtx, pipeline, steps, results)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ViewCmd) formatTextLogs(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	// Pipeline header
	status := "UNKNOWN"
	if pipeline.State != nil {
		// Use the result if available (SUCCESSFUL, FAILED, etc.)
		if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
			status = pipeline.State.Result.Name
		} else {
			// Fall back to state name (PENDING, IN_PROGRESS, COMPLETED, etc.)
			status = pipeline.State.Name
		}
	}

	branch := ""
	if pipeline.Target != nil && pipeline.Target.RefName != "" {
		branch = pipeline.Target.RefName
	}

	fmt.Printf("=== Pipeline #%d: %s (%s) ===\n", pipeline.BuildNumber, branch, status)
	fmt.Println()

	// Summary
	if len(results) > 0 {
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

func (cmd *ViewCmd) formatJSONLogs(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	output := map[string]interface{}{
		"pipeline":     pipeline,
		"steps":        steps,
		"log_analysis": results,
		"summary": map[string]interface{}{
			"total_steps": len(steps),
			"analyzed_at": time.Now(),
		},
	}

	return runCtx.Formatter.Format(output)
}

func (cmd *ViewCmd) formatYAMLLogs(runCtx *RunContext, pipeline *api.Pipeline, steps []*api.PipelineStep, results []*utils.LogAnalysisResult) error {
	output := map[string]interface{}{
		"pipeline":     pipeline,
		"steps":        steps,
		"log_analysis": results,
		"summary": map[string]interface{}{
			"total_steps": len(steps),
			"analyzed_at": time.Now(),
		},
	}

	return runCtx.Formatter.Format(output)
}

