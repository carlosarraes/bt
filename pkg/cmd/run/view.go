package run

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
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
	runCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor)
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

	// Watch mode for running pipelines
	if cmd.Watch {
		return cmd.watchPipeline(ctx, runCtx, pipelineUUID)
	}

	// Handle different view modes based on flags
	if cmd.Web {
		return cmd.openInBrowser(ctx, runCtx, pipelineUUID)
	}

	if cmd.Log || cmd.LogFailed || cmd.Tests || cmd.Step != "" {
		return cmd.viewLogs(ctx, runCtx, pipelineUUID)
	}

	// Default: show pipeline summary
	return cmd.viewPipeline(ctx, runCtx, pipelineUUID)
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
		duration = output.FormatDuration(pipeline.BuildSecondsUsed)
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
			stepDuration = output.FormatDuration(step.BuildSecondsUsed)
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
		fmt.Printf("Duration:    %s\n", output.FormatDuration(pipeline.BuildSecondsUsed))
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
				stepDuration = output.FormatDuration(step.BuildSecondsUsed)
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
		filteredSteps = filterStepsByName(steps, cmd.Step)
		if len(filteredSteps) == 0 {
			return fmt.Errorf("step '%s' not found. Available steps: %s", cmd.Step, getAvailableStepNames(steps))
		}
	}

	// Filter to failed steps only if --log-failed
	isTable := cmd.Output == "table"
	if cmd.LogFailed {
		if isTable {
			fmt.Printf("Looking for failed steps in pipeline...\n")
		}
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
			if isTable {
				fmt.Printf("  Step: '%s' - Status: %s, Result: %s, UUID: %s\n", step.Name, stepStatus, stepResult, step.UUID)
			}

			if step.State != nil && (step.State.Name == "FAILED" || (step.State.Result != nil && step.State.Result.Name == "FAILED")) {
				failedSteps = append(failedSteps, step)
				if isTable {
					fmt.Printf("    ✓ This step failed - will attempt to get logs\n")
				}
			}
		}
		filteredSteps = failedSteps

		if len(filteredSteps) == 0 {
			if isTable {
				fmt.Printf("No failed steps found in this pipeline\n")
			}
			return nil
		}

		if isTable {
			fmt.Printf("\nFound %d failed step(s). Attempting to retrieve logs...\n\n", len(filteredSteps))
		}
	}

	type stepLog struct {
		Step     *api.PipelineStep `json:"step"`
		Lines    []string          `json:"lines,omitempty"`
		Error    string            `json:"error,omitempty"`
		Truncated bool            `json:"truncated,omitempty"`
	}
	var stepLogs []stepLog

	for _, step := range filteredSteps {
		if cmd.Tests {
			if isTable {
				displayStepInfo(step)
				displayTestResults(ctx, runCtx, pipeline, step)
			}
			stepLogs = append(stepLogs, stepLog{Step: step})
			continue
		}

		if isTable {
			fmt.Printf("Attempting to fetch logs for step: %s (UUID: %s)\n", step.Name, step.UUID)
			fmt.Printf("Pipeline UUID: %s\n", pipeline.UUID)
			fmt.Printf("Repository: %s/%s\n", runCtx.Workspace, runCtx.Repository)
		}

		logReader, err := runCtx.Client.Pipelines.GetStepLogs(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
		if err != nil {
			if isTable {
				displayStepInfo(step)
				fmt.Printf("Logs not available: %v\n", err)
				displayTestResults(ctx, runCtx, pipeline, step)
			}
			stepLogs = append(stepLogs, stepLog{Step: step, Error: err.Error()})
			continue
		}

		logContent, err := io.ReadAll(logReader)
		logReader.Close()
		if err != nil {
			stepLogs = append(stepLogs, stepLog{Step: step, Error: err.Error()})
			continue
		}

		logLines := strings.Split(string(logContent), "\n")
		truncated := false

		if cmd.LogFailed && !cmd.FullOutput {
			startLine := len(logLines) - 100
			if startLine < 0 {
				startLine = 0
			}
			truncated = startLine > 0
			logLines = logLines[startLine:]
		}

		if isTable {
			if truncated {
				fmt.Printf("Showing last %d lines (use --full-output for complete logs)\n", len(logLines))
			} else {
				fmt.Printf("Showing all %d lines\n", len(logLines))
			}
			fmt.Println(strings.Repeat("=", 80))
			for _, line := range logLines {
				fmt.Println(line)
			}
			fmt.Println(strings.Repeat("=", 80))
		}

		stepLogs = append(stepLogs, stepLog{Step: step, Lines: logLines, Truncated: truncated})
	}

	if !isTable {
		output := map[string]interface{}{
			"pipeline": pipeline,
			"steps":    stepLogs,
		}
		return runCtx.Formatter.Format(output)
	}

	return nil
}


