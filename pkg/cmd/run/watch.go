package run

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/charmbracelet/lipgloss"
)

// WatchCmd handles the run watch command for real-time pipeline monitoring
type WatchCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Output     string `short:"o" help:"Output format (table, json)" enum:"table,json" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
	
	lastDisplayLines int
}

type LogBuffer struct {
	Lines []string
	Size  int
}

func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		Lines: make([]string, 0, size),
		Size:  size,
	}
}

func (lb *LogBuffer) Add(line string) {
	if len(lb.Lines) >= lb.Size {
		lb.Lines = lb.Lines[1:]
	}
	lb.Lines = append(lb.Lines, line)
}

func (lb *LogBuffer) GetLines() []string {
	return lb.Lines
}

// Run executes the run watch command
func (cmd *WatchCmd) Run(ctx context.Context) error {
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

	// Resolve pipeline ID to UUID
	pipelineUUID, err := cmd.resolvePipelineUUID(ctx, runCtx)
	if err != nil {
		return err
	}

	// Start watching the pipeline
	return cmd.watchPipeline(ctx, runCtx, pipelineUUID)
}

// resolvePipelineUUID resolves a pipeline ID (build number or UUID) to a full UUID
func (cmd *WatchCmd) resolvePipelineUUID(ctx context.Context, runCtx *RunContext) (string, error) {
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

// watchPipeline monitors a running pipeline for live updates
func (cmd *WatchCmd) watchPipeline(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
	// Create context that can be cancelled by signal
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Watch interrupted by user")
		cancel()
	}()

	// First, check if pipeline exists and get initial state
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(watchCtx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Check if pipeline is in a state that can be watched
	if pipeline.State == nil || (pipeline.State.Name != "IN_PROGRESS" && pipeline.State.Name != "PENDING") {
		fmt.Printf("Pipeline #%d is %s - watching is only available for running pipelines\n", 
			pipeline.BuildNumber, pipeline.State.Name)
		
		// Show current state and exit for completed pipelines
		if cmd.Output == "json" {
			return cmd.formatJSONOutput(runCtx, pipeline)
		} else {
			return cmd.displayFinalStatus(pipeline)
		}
	}

	fmt.Printf("🔍 Watching pipeline #%d (Ctrl+C to exit)...\n\n", pipeline.BuildNumber)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Show initial state
	if err := cmd.displayWatchUpdate(watchCtx, runCtx, pipelineUUID); err != nil {
		return err
	}

	// Track previous state for change detection
	var previousState string
	if pipeline.State != nil {
		previousState = pipeline.State.Name
	}

	for {
		select {
		case <-watchCtx.Done():
			return watchCtx.Err()
		case <-ticker.C:
			// Get updated pipeline status
			updatedPipeline, err := runCtx.Client.Pipelines.GetPipeline(watchCtx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
			if err != nil {
				return handlePipelineAPIError(err)
			}

			// Display update
			if err := cmd.displayWatchUpdate(watchCtx, runCtx, pipelineUUID); err != nil {
				return err
			}

			// Check for state changes and notify
			currentState := ""
			if updatedPipeline.State != nil {
				currentState = updatedPipeline.State.Name
			}

			if currentState != previousState && previousState != "" {
				cmd.notifyStateChange(previousState, currentState, updatedPipeline.BuildNumber)
			}
			previousState = currentState

			// Check if pipeline completed
			if updatedPipeline.State != nil && 
			   updatedPipeline.State.Name != "IN_PROGRESS" && 
			   updatedPipeline.State.Name != "PENDING" {
				fmt.Printf("\n🏁 Pipeline #%d completed with status: %s\n", 
					updatedPipeline.BuildNumber, updatedPipeline.State.Name)
				
				if cmd.Output == "json" {
					return cmd.formatJSONOutput(runCtx, updatedPipeline)
				}
				return nil
			}
		}
	}
}

func (cmd *WatchCmd) getRecentLogs(ctx context.Context, runCtx *RunContext, pipelineUUID, stepUUID string, lineCount int) ([]string, error) {
	logReader, err := runCtx.Client.Pipelines.GetStepLogs(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID, stepUUID)
	if err != nil {
		return nil, err
	}
	defer logReader.Close()

	var lines []string
	scanner := bufio.NewScanner(logReader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) > lineCount {
		return lines[len(lines)-lineCount:], nil
	}
	return lines, nil
}

func (cmd *WatchCmd) displayWatchUpdate(ctx context.Context, runCtx *RunContext, pipelineUUID string) error {
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

	if cmd.lastDisplayLines > 0 {
		fmt.Printf("\033[%dA", cmd.lastDisplayLines)
		fmt.Print("\033[J")
	}

	lineCount := 0
	
	// Pipeline status
	status := "UNKNOWN"
	if pipeline.State != nil {
		if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
			status = pipeline.State.Result.Name
		} else {
			status = pipeline.State.Name
		}
	}

	// Duration
	duration := ""
	if pipeline.BuildSecondsUsed > 0 {
		duration = FormatDuration(pipeline.BuildSecondsUsed)
	}

	fmt.Printf("[%s] %s Pipeline #%d: %s", 
		time.Now().Format("15:04:05"), 
		cmd.getStatusIcon(status), 
		pipeline.BuildNumber, 
		status)
	
	if duration != "" {
		fmt.Printf(" (%s)", duration)
	}

	// Show current step progress
	activeSteps := 0
	totalSteps := len(steps)
	completedSteps := 0
	var currentStep *api.PipelineStep
	
	for _, step := range steps {
		if step.State != nil {
			switch step.State.Name {
			case "COMPLETED":
				completedSteps++
			case "IN_PROGRESS":
				activeSteps++
				currentStep = step
				// Show which step is currently running
				fmt.Printf(" | 🔄 %s", step.Name)
			case "FAILED":
				completedSteps++
			}
		}
	}

	// Progress indicator
	if totalSteps > 0 {
		fmt.Printf(" [%d/%d steps]", completedSteps+activeSteps, totalSteps)
	}

	fmt.Println()
	lineCount++

	if currentStep != nil {
		fmt.Printf("\n📋 Recent output from \"%s\":\n", currentStep.Name)
		lineCount += 2
		
		recentLogs, err := cmd.getRecentLogs(ctx, runCtx, pipelineUUID, currentStep.UUID, 10)
		if err != nil {
			fmt.Printf("   (Unable to fetch logs: %v)\n", err)
			lineCount++
		} else if len(recentLogs) > 0 {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			if cmd.NoColor {
				dimStyle = lipgloss.NewStyle()
			}
			
			for _, line := range recentLogs {
				if line != "" {
					fmt.Printf("   %s\n", dimStyle.Render(line))
					lineCount++
				}
			}
		} else {
			fmt.Printf("   (No recent output)\n")
			lineCount++
		}
	} else {
		fmt.Printf("\n💤 No steps currently running\n")
		lineCount += 2
	}

	cmd.lastDisplayLines = lineCount

	return nil
}

// notifyStateChange shows a notification when pipeline state changes
func (cmd *WatchCmd) notifyStateChange(previousState, currentState string, buildNumber int) {
	fmt.Printf("\n📢 Pipeline #%d: %s → %s\n", buildNumber, previousState, currentState)
}

// getStatusIcon returns an icon for the given status
func (cmd *WatchCmd) getStatusIcon(status string) string {
	switch status {
	case "SUCCESSFUL":
		return "✅"
	case "FAILED":
		return "❌"
	case "IN_PROGRESS":
		return "🔄"
	case "PENDING":
		return "⏳"
	case "STOPPED":
		return "🛑"
	case "ERROR":
		return "💥"
	default:
		return "❓"
	}
}

// displayFinalStatus shows the final status for completed pipelines
func (cmd *WatchCmd) displayFinalStatus(pipeline *api.Pipeline) error {
	status := "UNKNOWN"
	if pipeline.State != nil {
		if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
			status = pipeline.State.Result.Name
		} else {
			status = pipeline.State.Name
		}
	}

	duration := ""
	if pipeline.BuildSecondsUsed > 0 {
		duration = FormatDuration(pipeline.BuildSecondsUsed)
	}

	fmt.Printf("%s Pipeline #%d: %s", 
		cmd.getStatusIcon(status), 
		pipeline.BuildNumber, 
		status)
	
	if duration != "" {
		fmt.Printf(" (%s)", duration)
	}
	fmt.Println()

	return nil
}

// formatJSONOutput formats the pipeline status as JSON
func (cmd *WatchCmd) formatJSONOutput(runCtx *RunContext, pipeline *api.Pipeline) error {
	// Use the same JSON formatting as the view command for consistency
	steps, err := runCtx.Client.Pipelines.GetPipelineSteps(context.Background(), 
		runCtx.Workspace, runCtx.Repository, pipeline.UUID)
	if err != nil {
		return err
	}

	// Create ViewCmd temporarily to reuse JSON formatting
	viewCmd := &ViewCmd{Output: "json"}
	return viewCmd.formatJSON(runCtx, pipeline, steps)
}
