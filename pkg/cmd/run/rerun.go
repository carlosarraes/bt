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
)

type RerunCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Failed     bool   `help:"Rerun only failed steps"`
	Step       string `help:"Rerun specific step"`
	Force      bool   `short:"f" help:"Force rerun without confirmation"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *RerunCmd) Run(ctx context.Context) error {
	runCtx, err := NewRunContext(ctx, cmd.Output, cmd.NoColor)
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

	pipelineUUID, err := cmd.resolvePipelineUUID(ctx, runCtx)
	if err != nil {
		return err
	}

	pipeline, err := runCtx.Client.Pipelines.GetPipeline(ctx, runCtx.Workspace, runCtx.Repository, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline details: %w", err)
	}

	if err := cmd.validateRerunnable(pipeline); err != nil {
		return err
	}

	if !cmd.Force {
		if !cmd.confirmRerun(pipeline) {
			fmt.Println("Rerun aborted.")
			return nil
		}
	}

	triggerRequest, err := cmd.buildTriggerRequest(ctx, runCtx, pipeline)
	if err != nil {
		return err
	}

	newPipeline, err := runCtx.Client.Pipelines.TriggerPipeline(ctx, runCtx.Workspace, runCtx.Repository, triggerRequest)
	if err != nil {
		return fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	return cmd.outputSuccess(runCtx, pipeline, newPipeline)
}

func (cmd *RerunCmd) resolvePipelineUUID(ctx context.Context, runCtx *RunContext) (string, error) {
	pipelineID := strings.TrimSpace(cmd.PipelineID)
	
	if strings.Contains(pipelineID, "-") {
		return pipelineID, nil
	}

	if strings.HasPrefix(pipelineID, "#") {
		pipelineID = pipelineID[1:]
	}

	buildNumber, err := strconv.Atoi(pipelineID)
	if err != nil {
		return "", fmt.Errorf("invalid pipeline ID '%s'. Expected build number (e.g., 123, #123) or UUID", cmd.PipelineID)
	}

	options := &api.PipelineListOptions{
		PageLen: 100,
	}
	
	fmt.Printf("🔍 Searching for pipeline #%d in workspace '%s', repository '%s'\n", buildNumber, runCtx.Workspace, runCtx.Repository)
	
	resp, err := runCtx.Client.Pipelines.ListPipelines(ctx, runCtx.Workspace, runCtx.Repository, options)
	if err != nil {
		return "", fmt.Errorf("failed to search for pipeline: %w", err)
	}

	pipelines, err := cmd.parsePipelineResults(resp)
	if err != nil {
		return "", fmt.Errorf("failed to parse pipeline results: %w", err)
	}

	fmt.Printf("📋 Found %d pipelines. Looking for build number %d:\n", len(pipelines), buildNumber)
	
	for i, pipeline := range pipelines {
		fmt.Printf("  [%d] Pipeline #%d (UUID: %s, State: %s)\n", i+1, pipeline.BuildNumber, pipeline.UUID, pipeline.State.Name)
		if pipeline.BuildNumber == buildNumber {
			fmt.Printf("✅ Found matching pipeline: #%d -> %s\n", buildNumber, pipeline.UUID)
			return pipeline.UUID, nil
		}
	}

	fmt.Printf("❌ Pipeline #%d not found in the %d recent pipelines\n", buildNumber, len(pipelines))
	fmt.Printf("💡 Try using the full UUID instead, or check if the pipeline is older than the %d most recent ones.\n", len(pipelines))
	return "", fmt.Errorf("pipeline with build number %d not found in the %d most recent pipelines", buildNumber, len(pipelines))
}

func (cmd *RerunCmd) parsePipelineResults(result *api.PaginatedResponse) ([]*api.Pipeline, error) {
	var pipelines []*api.Pipeline

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

func (cmd *RerunCmd) validateRerunnable(pipeline *api.Pipeline) error {
	switch pipeline.State.Name {
	case "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
		return nil
	case "PENDING", "IN_PROGRESS":
		return fmt.Errorf("pipeline #%d is still running (state: %s) and cannot be rerun. Use 'bt run cancel' to stop it first", 
			pipeline.BuildNumber, pipeline.State.Name)
	default:
		return fmt.Errorf("pipeline #%d is in an unknown state (%s) and may not be rerunnable", 
			pipeline.BuildNumber, pipeline.State.Name)
	}
}

func (cmd *RerunCmd) confirmRerun(pipeline *api.Pipeline) bool {
	action := "rerun"
	if cmd.Failed {
		action = "rerun failed steps of"
	} else if cmd.Step != "" {
		action = fmt.Sprintf("rerun step '%s' of", cmd.Step)
	}

	fmt.Printf("Are you sure you want to %s pipeline #%d (%s)?\n", 
		action, pipeline.BuildNumber, pipeline.State.Name)
	
	if pipeline.Target != nil {
		fmt.Printf("  Branch: %s\n", pipeline.Target.RefName)
		if pipeline.Target.Commit != nil {
			fmt.Printf("  Commit: %s\n", pipeline.Target.Commit.Hash[:8])
		}
	}
	
	fmt.Print("Type 'yes' to confirm: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "yes"
}

func (cmd *RerunCmd) buildTriggerRequest(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline) (*api.TriggerPipelineRequest, error) {
	if pipeline.Target == nil {
		return nil, fmt.Errorf("pipeline #%d has no target information", pipeline.BuildNumber)
	}

	request := &api.TriggerPipelineRequest{
		Target: &api.PipelineTarget{
			Type:     pipeline.Target.Type,
			RefType:  pipeline.Target.RefType,
			RefName:  pipeline.Target.RefName,
			Selector: pipeline.Target.Selector,
			Commit:   pipeline.Target.Commit,
		},
	}

	if cmd.Failed {
		fmt.Printf("⚠️  Bitbucket doesn't support rerunning only failed steps. The entire pipeline will be rerun.\n")
	}

	if cmd.Step != "" {
		fmt.Printf("⚠️  Bitbucket doesn't support rerunning individual steps. The entire pipeline will be rerun.\n")
	}

	return request, nil
}

func (cmd *RerunCmd) outputSuccess(runCtx *RunContext, originalPipeline *api.Pipeline, newPipeline *api.Pipeline) error {
	switch cmd.Output {
	case "json":
		return cmd.outputJSON(runCtx, originalPipeline, newPipeline)
	case "yaml":
		return cmd.outputYAML(runCtx, originalPipeline, newPipeline)
	default:
		return cmd.outputTable(originalPipeline, newPipeline)
	}
}

func (cmd *RerunCmd) outputTable(originalPipeline *api.Pipeline, newPipeline *api.Pipeline) error {
	fmt.Printf("✓ Pipeline #%d has been triggered successfully.\n", newPipeline.BuildNumber)
	fmt.Printf("  Original pipeline: #%d (%s)\n", originalPipeline.BuildNumber, originalPipeline.State.Name)
	fmt.Printf("  New pipeline: #%d (%s)\n", newPipeline.BuildNumber, newPipeline.State.Name)
	
	if newPipeline.Repository != nil {
		fmt.Printf("  Repository: %s\n", newPipeline.Repository.FullName)
	}
	if newPipeline.Target != nil {
		fmt.Printf("  Branch: %s\n", newPipeline.Target.RefName)
		if newPipeline.Target.Commit != nil {
			fmt.Printf("  Commit: %s\n", newPipeline.Target.Commit.Hash[:8])
		}
	}
	
	fmt.Printf("\nUse 'bt run watch %d' to monitor the new pipeline.\n", newPipeline.BuildNumber)
	return nil
}

func (cmd *RerunCmd) outputJSON(runCtx *RunContext, originalPipeline *api.Pipeline, newPipeline *api.Pipeline) error {
	originalData := map[string]interface{}{
		"build_number": originalPipeline.BuildNumber,
		"uuid":         originalPipeline.UUID,
	}
	
	if originalPipeline.State != nil {
		originalData["state"] = originalPipeline.State.Name
	}
	
	newData := map[string]interface{}{
		"build_number": newPipeline.BuildNumber,
		"uuid":         newPipeline.UUID,
	}
	
	if newPipeline.State != nil {
		newData["state"] = newPipeline.State.Name
	}
	
	if newPipeline.Repository != nil {
		newData["repository"] = newPipeline.Repository.FullName
	}
	
	if newPipeline.Target != nil {
		newData["branch"] = newPipeline.Target.RefName
		if newPipeline.Target.Commit != nil {
			newData["commit"] = newPipeline.Target.Commit.Hash
		}
	}
	
	result := map[string]interface{}{
		"success":           true,
		"message":           "Pipeline rerun triggered successfully",
		"original_pipeline": originalData,
		"new_pipeline":      newData,
	}
	
	return runCtx.Formatter.Format(result)
}

func (cmd *RerunCmd) outputYAML(runCtx *RunContext, originalPipeline *api.Pipeline, newPipeline *api.Pipeline) error {
	originalData := map[string]interface{}{
		"build_number": originalPipeline.BuildNumber,
		"uuid":         originalPipeline.UUID,
	}
	
	if originalPipeline.State != nil {
		originalData["state"] = originalPipeline.State.Name
	}
	
	newData := map[string]interface{}{
		"build_number": newPipeline.BuildNumber,
		"uuid":         newPipeline.UUID,
	}
	
	if newPipeline.State != nil {
		newData["state"] = newPipeline.State.Name
	}
	
	if newPipeline.Repository != nil {
		newData["repository"] = newPipeline.Repository.FullName
	}
	
	if newPipeline.Target != nil {
		newData["branch"] = newPipeline.Target.RefName
		if newPipeline.Target.Commit != nil {
			newData["commit"] = newPipeline.Target.Commit.Hash
		}
	}
	
	result := map[string]interface{}{
		"success":           true,
		"message":           "Pipeline rerun triggered successfully",
		"original_pipeline": originalData,
		"new_pipeline":      newData,
	}
	
	return runCtx.Formatter.Format(result)
}
