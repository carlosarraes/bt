package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
)

type ListCmd struct {
	Status     string `help:"Filter by status (PENDING, IN_PROGRESS, SUCCESSFUL, FAILED, ERROR, STOPPED)"`
	Branch     string `help:"Filter by branch name"`
	Limit      int    `help:"Maximum number of runs to show" default:"10"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

// Run executes the run list command
func (cmd *ListCmd) Run(ctx context.Context) error {
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
		Sort:    "-created_on",
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
	headers := []string{"ID", "Status", "Ref", "Started By", "Duration", "Started"}
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

		startedTime := output.FormatRelativeTime(pipeline.CreatedOn)

		duration := "-"
		if pipeline.BuildSecondsUsed > 0 {
			duration = output.FormatDuration(pipeline.BuildSecondsUsed)
		}

		ref := "-"
		if pipeline.Target != nil {
			// Check if this is a PR-triggered pipeline
			if pipeline.Target.Type == "pipeline_pullrequest_target" {
				ref = "PR"
			} else if pipeline.Target.PullRequestId != nil {
				ref = fmt.Sprintf("PR #%d", *pipeline.Target.PullRequestId)
			} else if pipeline.Target.RefName != "" {
				ref = pipeline.Target.RefName
				if len(ref) > 15 {
					ref = ref[:12] + "..."
				}
			} else if pipeline.Target.Type == "pipeline_branch_target" {
				// This is a branch pipeline but no ref_name, try to infer from trigger
				ref = "branch"
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
			ref,
			startedBy,
			duration,
			startedTime,
		}
	}

	return output.RenderSimpleTable(headers, rows)
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

func parsePipelineResults(result *api.PaginatedResponse) ([]*api.Pipeline, error) {
	return shared.ParsePaginatedResults[api.Pipeline](result)
}

func validateStatus(status string) error {
	return shared.ValidateAllowedValue(status, shared.AllowedPipelineStatuses, "status")
}

func handlePipelineAPIError(err error) error {
	return shared.HandleAPIError(err, shared.DomainPipeline)
}
