package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
)

// ListCmd handles the run list command
type ListCmd struct {
	Status     string `help:"Filter by status (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED)"`
	Branch     string `help:"Filter by branch name"`
	Limit      int    `help:"Maximum number of runs to show" default:"10"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool   // NoColor is passed from global flag
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the run list command
func (cmd *ListCmd) Run(ctx context.Context) error {
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

	// Validate status filter if provided
	if cmd.Status != "" {
		if err := validateStatus(cmd.Status); err != nil {
			return err
		}
	}

	// Validate limit
	if cmd.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}
	if cmd.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	// Prepare pipeline list options
	options := &api.PipelineListOptions{
		PageLen: cmd.Limit,
		Page:    1,
		Sort:    "-created_on", // Most recent first
	}

	// Add status filter if specified
	if cmd.Status != "" {
		options.Status = strings.ToUpper(cmd.Status)
	}

	// Add branch filter if specified
	if cmd.Branch != "" {
		options.Branch = cmd.Branch
	}

	// List pipelines
	result, err := runCtx.Client.Pipelines.ListPipelines(ctx, runCtx.Workspace, runCtx.Repository, options)
	if err != nil {
		return handlePipelineAPIError(err)
	}

	// Parse the pipeline results
	pipelines, err := parsePipelineResults(result)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline results: %w", err)
	}

	// Format and display output
	return cmd.formatOutput(runCtx, pipelines)
}

// formatOutput formats and displays the pipeline results
func (cmd *ListCmd) formatOutput(runCtx *RunContext, pipelines []*api.Pipeline) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(runCtx, pipelines)
	case "json":
		return cmd.formatJSON(runCtx, pipelines)
	case "yaml":
		return cmd.formatYAML(runCtx, pipelines)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

// formatTable formats pipelines as a table
func (cmd *ListCmd) formatTable(runCtx *RunContext, pipelines []*api.Pipeline) error {
	if len(pipelines) == 0 {
		fmt.Println("No pipeline runs found")
		return nil
	}

	// Custom table rendering for better control
	headers := []string{"ID", "Status", "Branch", "Started By", "Duration", "Started"}
	rows := make([][]string, len(pipelines))
	
	for i, pipeline := range pipelines {
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

		startedTime := FormatRelativeTime(pipeline.CreatedOn)

		duration := "-"
		if pipeline.BuildSecondsUsed > 0 {
			duration = FormatDuration(pipeline.BuildSecondsUsed)
		}

		branch := "-"
		if pipeline.Target != nil && pipeline.Target.RefName != "" {
			branch = pipeline.Target.RefName
			if len(branch) > 15 {
				branch = branch[:12] + "..."
			}
		}

		startedBy := "-"
		if pipeline.Creator != nil {
			if pipeline.Creator.DisplayName != "" {
				startedBy = pipeline.Creator.DisplayName
			} else if pipeline.Creator.Username != "" {
				startedBy = pipeline.Creator.Username
			}
			// Truncate long names
			if len(startedBy) > 15 {
				startedBy = startedBy[:12] + "..."
			}
		}

		rows[i] = []string{
			fmt.Sprintf("#%d", pipeline.BuildNumber),
			status,
			branch,
			startedBy,
			duration,
			startedTime,
		}
	}

	return renderCustomTable(headers, rows)
}

// formatJSON formats pipelines as JSON
func (cmd *ListCmd) formatJSON(runCtx *RunContext, pipelines []*api.Pipeline) error {
	// Create a simplified structure for JSON output
	output := map[string]interface{}{
		"total_count": len(pipelines),
		"pipelines":   pipelines,
	}

	return runCtx.Formatter.Format(output)
}

// formatYAML formats pipelines as YAML
func (cmd *ListCmd) formatYAML(runCtx *RunContext, pipelines []*api.Pipeline) error {
	// Create a simplified structure for YAML output
	output := map[string]interface{}{
		"total_count": len(pipelines),
		"pipelines":   pipelines,
	}

	return runCtx.Formatter.Format(output)
}

// parsePipelineResults parses the paginated response into Pipeline structs
func parsePipelineResults(result *api.PaginatedResponse) ([]*api.Pipeline, error) {
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

// validateStatus validates the status filter
func validateStatus(status string) error {
	validStatuses := []string{
		"PENDING", "IN_PROGRESS", "SUCCESSFUL", "FAILED", "ERROR", "STOPPED",
	}

	statusUpper := strings.ToUpper(status)
	for _, validStatus := range validStatuses {
		if statusUpper == validStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid status '%s'. Valid statuses are: %s", 
		status, strings.Join(validStatuses, ", "))
}

// handlePipelineAPIError provides user-friendly error messages for API errors
func handlePipelineAPIError(err error) error {
	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		switch bitbucketErr.Type {
		case api.ErrorTypeNotFound:
			return fmt.Errorf("repository not found or pipelines not enabled. Verify the repository exists and has Bitbucket Pipelines enabled")
		case api.ErrorTypeAuthentication:
			return fmt.Errorf("authentication failed. Please run 'bt auth login' to authenticate")
		case api.ErrorTypePermission:
			return fmt.Errorf("permission denied. You may not have access to this repository")
		case api.ErrorTypeRateLimit:
			return fmt.Errorf("rate limit exceeded. Please wait before making more requests")
		default:
			return fmt.Errorf("API error: %s", bitbucketErr.Message)
		}
	}

	return fmt.Errorf("failed to list pipelines: %w", err)
}

// renderCustomTable renders a table with proper alignment
func renderCustomTable(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Render header
	for i, header := range headers {
		fmt.Printf("%-*s", colWidths[i], header)
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	// Render separator
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	// Render rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Printf("%-*s", colWidths[i], cell)
				if i < len(row)-1 {
					fmt.Print("  ")
				}
			}
		}
		fmt.Println()
	}

	return nil
}