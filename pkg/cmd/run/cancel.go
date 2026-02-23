package run

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type CancelCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Force      bool   `short:"f" help:"Force cancellation without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *CancelCmd) Run(ctx context.Context) error {
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

	pipelineUUID, err := resolvePipelineUUID(ctx, runCtx, cmd.PipelineID)
	if err != nil {
		return err
	}

	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline details: %w", err)
	}

	if err := cmd.validateCancellable(pipeline); err != nil {
		return err
	}

	if !cmd.Force {
		if !cmd.confirmCancellation(pipeline) {
			fmt.Println("Cancellation aborted.")
			return nil
		}
	}

	if err := runCtx.Client.Pipelines.CancelPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID); err != nil {
		return fmt.Errorf("failed to cancel pipeline: %w", err)
	}

	return cmd.outputSuccess(runCtx, pipeline)
}

func (cmd *CancelCmd) validateCancellable(pipeline *api.Pipeline) error {
	switch pipeline.State.Name {
	case "PENDING", "IN_PROGRESS":
		return nil
	case "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
		return fmt.Errorf("pipeline #%d is already completed (state: %s) and cannot be cancelled",
			pipeline.BuildNumber, pipeline.State.Name)
	default:
		return fmt.Errorf("pipeline #%d is in an unknown state (%s) and may not be cancellable",
			pipeline.BuildNumber, pipeline.State.Name)
	}
}

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
