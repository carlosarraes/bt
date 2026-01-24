package run

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

// CancelCmd handles the run cancel command
type CancelCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Force      bool   `short:"f" help:"Force cancellation without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the run cancel command
func (cmd *CancelCmd) Run(ctx context.Context) error {
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

	// Resolve pipeline ID (build number to UUID)
	pipelineUUID, err := cmd.resolvePipelineUUID(ctx, runCtx)
	if err != nil {
		return err
	}

	// Get pipeline details to check if it's cancellable
	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline details: %w", err)
	}

	// Check if pipeline can be cancelled
	if err := cmd.validateCancellable(pipeline); err != nil {
		return err
	}

	// Show confirmation prompt unless --force is used
	if !cmd.Force {
		if !cmd.confirmCancellation(pipeline) {
			fmt.Println("Cancellation aborted.")
			return nil
		}
	}

	// Cancel the pipeline
	if err := runCtx.Client.Pipelines.CancelPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID); err != nil {
		return fmt.Errorf("failed to cancel pipeline: %w", err)
	}

	// Output success message
	return cmd.outputSuccess(runCtx, pipeline)
}

// resolvePipelineUUID converts build number to UUID or validates UUID
func (cmd *CancelCmd) resolvePipelineUUID(ctx context.Context, runCtx *RunContext) (string, error) {
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
		PageLen: 50, // Search recent pipelines
	}
	
	resp, err := runCtx.Client.Pipelines.ListPipelines(ctx, runCtx.Workspace, runCtx.Repository, options)
	if err != nil {
		return "", fmt.Errorf("failed to search for pipeline: %w", err)
	}

	// Parse pipeline results
	pipelines, err := cmd.parsePipelineResults(resp)
	if err != nil {
		return "", fmt.Errorf("failed to parse pipeline results: %w", err)
	}

	// Look for matching build number
	for _, pipeline := range pipelines {
		if pipeline.BuildNumber == buildNumber {
			return pipeline.UUID, nil
		}
	}

	return "", fmt.Errorf("pipeline with build number %d not found", buildNumber)
}

// parsePipelineResults parses the paginated response into Pipeline structs
func (cmd *CancelCmd) parsePipelineResults(result *api.PaginatedResponse) ([]*api.Pipeline, error) {
	var pipelines []*api.Pipeline

	// Parse the Values field (raw JSON) into Pipeline structs
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pipeline values: %w", err)
		}

		pipelines = make([]*api.Pipeline, len(values))
		for i, rawPipeline := range values {
			var pipeline api.Pipeline
			if err := json.Unmarshal(rawPipeline, &pipeline); err != nil {
				return nil, fmt.Errorf("failed to unmarshal pipeline %d: %w", i, err)
			}
			pipelines[i] = &pipeline
		}
	}

	return pipelines, nil
}

// validateCancellable checks if the pipeline can be cancelled
func (cmd *CancelCmd) validateCancellable(pipeline *api.Pipeline) error {
	// Check if pipeline is in a cancellable state
	switch pipeline.State.Name {
	case "PENDING", "IN_PROGRESS":
		// These states are cancellable
		return nil
	case "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
		return fmt.Errorf("pipeline #%d is already completed (state: %s) and cannot be cancelled", 
			pipeline.BuildNumber, pipeline.State.Name)
	default:
		return fmt.Errorf("pipeline #%d is in an unknown state (%s) and may not be cancellable", 
			pipeline.BuildNumber, pipeline.State.Name)
	}
}

// confirmCancellation prompts the user for confirmation
func (cmd *CancelCmd) confirmCancellation(pipeline *api.Pipeline) bool {
	fmt.Printf("Are you sure you want to cancel pipeline #%d (%s)? This action cannot be undone.\n", 
		pipeline.BuildNumber, pipeline.State.Name)
	fmt.Print("Type 'yes' to confirm: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "yes"
}

// outputSuccess outputs the success message in the requested format
func (cmd *CancelCmd) outputSuccess(runCtx *RunContext, pipeline *api.Pipeline) error {
	switch cmd.Output {
	case "json":
		return cmd.outputJSON(runCtx, pipeline)
	case "yaml":
		return cmd.outputYAML(runCtx, pipeline)
	default:
		return cmd.outputTable(pipeline)
	}
}

// outputTable outputs success in table format
func (cmd *CancelCmd) outputTable(pipeline *api.Pipeline) error {
	fmt.Printf("✓ Pipeline #%d has been cancelled successfully.\n", pipeline.BuildNumber)
	if pipeline.Repository != nil {
		fmt.Printf("  Repository: %s\n", pipeline.Repository.FullName)
	}
	if pipeline.Target != nil {
		fmt.Printf("  Branch: %s\n", pipeline.Target.RefName)
		if pipeline.Target.Commit != nil {
			fmt.Printf("  Commit: %s\n", pipeline.Target.Commit.Hash[:8])
		}
	}
	return nil
}

// outputJSON outputs success in JSON format
func (cmd *CancelCmd) outputJSON(runCtx *RunContext, pipeline *api.Pipeline) error {
	pipelineData := map[string]interface{}{
		"build_number": pipeline.BuildNumber,
		"uuid":         pipeline.UUID,
	}
	
	if pipeline.State != nil {
		pipelineData["state"] = pipeline.State.Name
	}
	
	if pipeline.Repository != nil {
		pipelineData["repository"] = pipeline.Repository.FullName
	}
	
	if pipeline.Target != nil {
		pipelineData["branch"] = pipeline.Target.RefName
		if pipeline.Target.Commit != nil {
			pipelineData["commit"] = pipeline.Target.Commit.Hash
		}
	}
	
	result := map[string]interface{}{
		"success":  true,
		"message":  "Pipeline cancelled successfully",
		"pipeline": pipelineData,
	}
	
	return runCtx.Formatter.Format(result)
}

// outputYAML outputs success in YAML format  
func (cmd *CancelCmd) outputYAML(runCtx *RunContext, pipeline *api.Pipeline) error {
	pipelineData := map[string]interface{}{
		"build_number": pipeline.BuildNumber,
		"uuid":         pipeline.UUID,
	}
	
	if pipeline.State != nil {
		pipelineData["state"] = pipeline.State.Name
	}
	
	if pipeline.Repository != nil {
		pipelineData["repository"] = pipeline.Repository.FullName
	}
	
	if pipeline.Target != nil {
		pipelineData["branch"] = pipeline.Target.RefName
		if pipeline.Target.Commit != nil {
			pipelineData["commit"] = pipeline.Target.Commit.Hash
		}
	}
	
	result := map[string]interface{}{
		"success":  true,
		"message":  "Pipeline cancelled successfully",
		"pipeline": pipelineData,
	}
	
	return runCtx.Formatter.Format(result)
}