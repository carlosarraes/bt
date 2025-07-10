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
	
	logBuffer        *LogBuffer
}

type LogBuffer struct {
	AllLines     []string
	LastLogCount int
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		AllLines:     make([]string, 0),
		LastLogCount: 0,
	}
}

func (lb *LogBuffer) GetNewLines(allLines []string) []string {
	if len(allLines) <= lb.LastLogCount {
		return nil
	}
	
	newLines := allLines[lb.LastLogCount:]
	lb.LastLogCount = len(allLines)
	
	var validLines []string
	for _, line := range newLines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}
	
	return validLines
}

func (lb *LogBuffer) Reset() {
	lb.LastLogCount = 0
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

	fmt.Printf("🔍 Watching pipeline #%d (Ctrl+C to exit)...\n", pipeline.BuildNumber)

	cmd.logBuffer = NewLogBuffer()

	updateTicker := time.NewTicker(2 * time.Second)
	defer updateTicker.Stop()

	var currentStepUUID string
	var currentStepName string
	var displayedInitialStatus bool

	for {
		select {
		case <-watchCtx.Done():
			return watchCtx.Err()
			
		case <-updateTicker.C:
			updatedPipeline, err := runCtx.Client.Pipelines.GetPipeline(watchCtx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
			if err != nil {
				return handlePipelineAPIError(err)
			}

			steps, err := runCtx.Client.Pipelines.GetPipelineSteps(watchCtx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
			if err != nil {
				return err
			}

			var activeStep *api.PipelineStep
			completedSteps := 0
			totalSteps := len(steps)
			
			for _, step := range steps {
				if step.State != nil {
					switch step.State.Name {
					case "IN_PROGRESS":
						activeStep = step
					case "COMPLETED", "SUCCESSFUL":
						completedSteps++
					case "FAILED":
						completedSteps++
					}
				}
			}

			newStepUUID := ""
			newStepName := ""
			if activeStep != nil {
				newStepUUID = activeStep.UUID
				newStepName = activeStep.Name
			}

			if newStepUUID != currentStepUUID {
				if currentStepUUID != "" && currentStepName != "" && newStepUUID != "" {
					fmt.Printf("✅ Step completed: %s\n", currentStepName)
				}
				
				currentStepUUID = newStepUUID
				currentStepName = newStepName
				cmd.logBuffer.Reset()
				
				if activeStep != nil {
					fmt.Printf("🔄 Pipeline #%d: IN_PROGRESS | 🔄 %s [%d/%d steps]\n", 
						updatedPipeline.BuildNumber, activeStep.Name, completedSteps+1, totalSteps)
					fmt.Printf("📋 Streaming output from \"%s\":\n", activeStep.Name)
					displayedInitialStatus = true
				}
			} else if !displayedInitialStatus && activeStep != nil {
				status := ""
				if updatedPipeline.State != nil {
					status = updatedPipeline.State.Name
				}
				fmt.Printf("%s Pipeline #%d: %s | 🔄 %s [%d/%d steps]\n", 
					cmd.getStatusIcon(status), updatedPipeline.BuildNumber, status, 
					activeStep.Name, completedSteps+1, totalSteps)
				fmt.Printf("📋 Streaming output from \"%s\":\n", activeStep.Name)
				displayedInitialStatus = true
			}

			if currentStepUUID != "" {
				allLogs, err := cmd.getAllLogs(watchCtx, runCtx, pipelineUUID, currentStepUUID)
				if err == nil {
					newLines := cmd.logBuffer.GetNewLines(allLogs)
					dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
					if cmd.NoColor {
						dimStyle = lipgloss.NewStyle()
					}
					for _, line := range newLines {
						fmt.Printf("   %s\n", dimStyle.Render(line))
					}
				}
			}

			// Check if pipeline completed
			if updatedPipeline.State != nil && 
			   updatedPipeline.State.Name != "IN_PROGRESS" && 
			   updatedPipeline.State.Name != "PENDING" {
				
				fmt.Printf("🏁 Pipeline #%d completed with status: %s\n", 
					updatedPipeline.BuildNumber, updatedPipeline.State.Name)
				
				if cmd.Output == "json" {
					return cmd.formatJSONOutput(runCtx, updatedPipeline)
				}
				return nil
			}
		}
	}
}

func (cmd *WatchCmd) getAllLogs(ctx context.Context, runCtx *RunContext, pipelineUUID, stepUUID string) ([]string, error) {
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

	return lines, nil
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
