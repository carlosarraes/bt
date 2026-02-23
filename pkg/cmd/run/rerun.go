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

type RerunCmd struct {
	PipelineID string `arg:"" help:"Pipeline ID (build number or UUID)"`
	Failed     bool   `help:"Rerun only failed steps"`
	Step       string `help:"Rerun specific step"`
	Force      bool   `short:"f" help:"Force rerun without confirmation"`
	Debug      bool   `help:"Show debug information"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *RerunCmd) Run(ctx context.Context) error {
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

func (cmd *RerunCmd) findPullRequestByCommit(ctx context.Context, runCtx *RunContext, commitHash string) (*int, error) {
	if cmd.Debug {
		fmt.Printf("🐛 Debug: Looking for PR with commit hash: %s\n", commitHash)
	}

	options := &api.PullRequestListOptions{
		State:   "OPEN,MERGED,DECLINED",
		PageLen: 50,
	}

	resp, err := runCtx.Client.PullRequests.ListPullRequests(ctx, runCtx.Workspace, runCtx.Repository, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	pullRequests, err := cmd.parsePullRequestResults(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pull request results: %w", err)
	}

	for _, pr := range pullRequests {
		if pr.Source != nil && pr.Source.Commit != nil && pr.Source.Commit.Hash == commitHash {
			if cmd.Debug {
				fmt.Printf("🐛 Debug: Found matching PR #%d for commit %s (source)\n", pr.ID, commitHash)
			}
			return &pr.ID, nil
		}

		if pr.Destination != nil && pr.Destination.Commit != nil && pr.Destination.Commit.Hash == commitHash {
			if cmd.Debug {
				fmt.Printf("🐛 Debug: Found matching PR #%d for commit %s (destination)\n", pr.ID, commitHash)
			}
			return &pr.ID, nil
		}

		if cmd.Debug && pr.Source != nil && pr.Source.Commit != nil {
			fmt.Printf("🐛 Debug: PR #%d source commit: %s\n", pr.ID, pr.Source.Commit.Hash)
		}
		if cmd.Debug && pr.Destination != nil && pr.Destination.Commit != nil {
			fmt.Printf("🐛 Debug: PR #%d destination commit: %s\n", pr.ID, pr.Destination.Commit.Hash)
		}
	}

	if cmd.Debug {
		fmt.Printf("🐛 Debug: No PR found for commit %s\n", commitHash)
	}
	return nil, fmt.Errorf("no pull request found for commit %s", commitHash)
}

func (cmd *RerunCmd) parsePullRequestResults(result *api.PaginatedResponse) ([]*api.PullRequest, error) {
	return shared.ParsePaginatedResults[api.PullRequest](result)
}

func (cmd *RerunCmd) validateRerunnable(pipeline *api.Pipeline) error {
	if pipeline.State == nil {
		return fmt.Errorf("pipeline #%d has no state information", pipeline.BuildNumber)
	}

	switch pipeline.State.Name {
	case "PENDING", "IN_PROGRESS":
		return fmt.Errorf("pipeline #%d is still running (state: %s) and cannot be rerun. Use 'bt run cancel' to stop it first",
			pipeline.BuildNumber, pipeline.State.Name)
	}

	switch pipeline.State.Name {
	case "COMPLETED":
		return nil
	case "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
		return nil
	default:
		if cmd.Debug {
			resultName := "UNKNOWN"
			if pipeline.State.Result != nil {
				resultName = pipeline.State.Result.Name
			}
			fmt.Printf("🐛 Debug: State=%s, Result=%s\n", pipeline.State.Name, resultName)
		}
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

	displayStatus := pipeline.State.Name
	if pipeline.State.Result != nil && pipeline.State.Result.Name != "" {
		displayStatus = pipeline.State.Result.Name
	}

	fmt.Printf("Are you sure you want to %s pipeline #%d (%s)?\n",
		action, pipeline.BuildNumber, displayStatus)

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

	if cmd.Debug {
		fmt.Printf("🐛 Debug: Original pipeline target:\n")
		fmt.Printf("  Type: %s\n", pipeline.Target.Type)
		fmt.Printf("  RefType: %s\n", pipeline.Target.RefType)
		fmt.Printf("  RefName: %s\n", pipeline.Target.RefName)
		if pipeline.Target.PullRequestId != nil {
			fmt.Printf("  PullRequestId: %d\n", *pipeline.Target.PullRequestId)
		} else {
			fmt.Printf("  PullRequestId: nil\n")
		}
		if pipeline.Target.Selector != nil {
			fmt.Printf("  Selector: %+v\n", pipeline.Target.Selector)
		} else {
			fmt.Printf("  Selector: nil\n")
		}
		if pipeline.Target.Commit != nil {
			fmt.Printf("  Commit: %s\n", pipeline.Target.Commit.Hash)
		} else {
			fmt.Printf("  Commit: nil\n")
		}
	}

	pullRequestId := pipeline.Target.PullRequestId
	if pipeline.Target.Type == "pipeline_pullrequest_target" && pullRequestId == nil {
		if pipeline.Target.Commit != nil {
			if cmd.Debug {
				fmt.Printf("🐛 Debug: PR pipeline missing pullRequestId, attempting to find by commit hash\n")
			}

			foundPRId, err := cmd.findPullRequestByCommit(ctx, runCtx, pipeline.Target.Commit.Hash)
			if err != nil {
				return nil, fmt.Errorf("failed to find pull request for commit %s: %w", pipeline.Target.Commit.Hash, err)
			}
			pullRequestId = foundPRId

			if cmd.Debug {
				fmt.Printf("🐛 Debug: Found PR ID: %d\n", *pullRequestId)
			}
		} else {
			return nil, fmt.Errorf("pull request pipeline missing both pullRequestId and commit hash")
		}
	}

	request := &api.TriggerPipelineRequest{
		Target: &api.PipelineTarget{
			Type:          pipeline.Target.Type,
			RefType:       pipeline.Target.RefType,
			RefName:       pipeline.Target.RefName,
			Selector:      pipeline.Target.Selector,
			Commit:        pipeline.Target.Commit,
			PullRequestId: pullRequestId,
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
