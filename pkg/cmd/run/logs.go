package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
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
		outputFormat = "table" // Use table formatter for context, but we'll output raw text
	}

	// Create run context with authentication and configuration
	runCtx, err := shared.NewCommandContext(ctx, outputFormat, cmd.NoColor)
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
	pipelineUUID, err := resolvePipelineUUID(ctx, runCtx, cmd.PipelineID)
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
		filteredSteps = filterStepsByName(steps, cmd.Step)
		if len(filteredSteps) == 0 {
			return fmt.Errorf("step '%s' not found. Available steps: %s", cmd.Step, getAvailableStepNames(steps))
		}
	}

	// Process logs for each step
	allResults := make([]*utils.LogAnalysisResult, 0, len(filteredSteps))

	for _, step := range filteredSteps {
		// Try to get step logs first, or use --tests flag to show test results
		if cmd.Tests {
			// Show test results instead of logs
			if cmd.Output == "text" {
				displayStepInfo(step)
				displayTestResults(ctx, runCtx, pipeline, step)
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
				displayStepInfo(step)
				fmt.Printf("Note: Raw logs not available through API for step '%s': %v\n", step.Name, err)

				fmt.Printf("Checking for test results...\n")
				displayTestResults(ctx, runCtx, pipeline, step)
			}
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

		parser := utils.NewLogParser()
		parser.SetContextLines(cmd.Context)

		result, err := parser.AnalyzeLog(logReader, step.Name)
		logReader.Close()
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
				if cmd.Step != "" && !matchesStepName(step.Name, cmd.Step) {
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
