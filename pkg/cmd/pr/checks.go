package pr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type ChecksCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	Watch      bool   `short:"w" help:"Watch for live updates"`
	Output     string `short:"o" help:"Output format (table, json, yaml)" enum:"table,json,yaml" default:"table"`
	NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}

func (cmd *ChecksCmd) Run(ctx context.Context) error {
	prCtx, err := shared.NewCommandContext(ctx, cmd.Output, cmd.NoColor)
	if err != nil {
		return err
	}

	if cmd.Workspace != "" {
		prCtx.Workspace = cmd.Workspace
	}
	if cmd.Repository != "" {
		prCtx.Repository = cmd.Repository
	}

	if err := prCtx.ValidateWorkspaceAndRepo(); err != nil {
		return err
	}

	prID, err := cmd.ParsePRID()
	if err != nil {
		return err
	}

	if cmd.Watch {
		return cmd.watchChecks(ctx, prCtx, prID)
	}

	checks, err := cmd.getChecks(ctx, prCtx, prID)
	if err != nil {
		return err
	}

	return cmd.formatOutput(prCtx, checks)
}

func (cmd *ChecksCmd) ParsePRID() (int, error) {
	return ParsePRID(cmd.PRID)
}

func (cmd *ChecksCmd) getChecks(ctx context.Context, prCtx *PRContext, prID int) ([]*api.Pipeline, error) {
	pr, err := prCtx.Client.PullRequests.GetPullRequest(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return nil, handlePullRequestAPIError(err)
	}

	commitSHA := ""
	if pr.Source != nil && pr.Source.Commit != nil {
		commitSHA = pr.Source.Commit.Hash
	}

	if commitSHA == "" {
		return nil, fmt.Errorf("unable to find commit SHA for pull request #%d", prID)
	}

	pipelines, err := prCtx.Client.Pipelines.GetPipelinesByCommit(ctx, prCtx.Workspace, prCtx.Repository, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipelines for commit %s: %w", commitSHA, err)
	}

	return pipelines, nil
}

func (cmd *ChecksCmd) watchChecks(ctx context.Context, prCtx *PRContext, prID int) error {
	checks, err := cmd.getChecks(ctx, prCtx, prID)
	if err != nil {
		return err
	}

	if err := cmd.formatOutput(prCtx, checks); err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newChecks, err := cmd.getChecks(ctx, prCtx, prID)
			if err != nil {
				return err
			}

			if cmd.hasStatusChanged(checks, newChecks) {
				fmt.Printf("\n--- Updated at %s ---\n", time.Now().Format("15:04:05"))
				if err := cmd.formatOutput(prCtx, newChecks); err != nil {
					return err
				}
				checks = newChecks
			}

			if cmd.allChecksCompleted(checks) {
				fmt.Println("\nAll checks completed.")
				return nil
			}
		}
	}
}

func (cmd *ChecksCmd) hasStatusChanged(oldChecks, newChecks []*api.Pipeline) bool {
	if len(oldChecks) != len(newChecks) {
		return true
	}

	oldStatusMap := make(map[string]string)
	for _, pipeline := range oldChecks {
		status := "unknown"
		if pipeline.State != nil {
			status = pipeline.State.Name
		}
		oldStatusMap[pipeline.UUID] = status
	}

	for _, pipeline := range newChecks {
		status := "unknown"
		if pipeline.State != nil {
			status = pipeline.State.Name
		}
		if oldStatusMap[pipeline.UUID] != status {
			return true
		}
	}

	return false
}

func (cmd *ChecksCmd) allChecksCompleted(checks []*api.Pipeline) bool {
	for _, pipeline := range checks {
		if pipeline.State != nil {
			switch pipeline.State.Name {
			case "PENDING", "IN_PROGRESS":
				return false
			}
		}
	}
	return true
}

func (cmd *ChecksCmd) formatOutput(prCtx *PRContext, checks []*api.Pipeline) error {
	switch cmd.Output {
	case "table":
		return cmd.formatTable(prCtx, checks)
	case "json":
		return cmd.formatJSON(prCtx, checks)
	case "yaml":
		return cmd.formatYAML(prCtx, checks)
	default:
		return fmt.Errorf("unsupported output format: %s", cmd.Output)
	}
}

func (cmd *ChecksCmd) formatTable(prCtx *PRContext, checks []*api.Pipeline) error {
	if len(checks) == 0 {
		fmt.Println("No CI checks found for this pull request.")
		return nil
	}

	sortedChecks := cmd.sortChecksByPriority(checks)

	summary := cmd.getChecksSummary(checks)
	fmt.Printf("Checks for pull request #%s: %s\n\n", cmd.PRID, summary)

	for _, pipeline := range sortedChecks {
		status := cmd.getStatusIndicator(pipeline)
		name := cmd.getPipelineName(pipeline)
		duration := cmd.getPipelineDuration(pipeline)

		fmt.Printf("%s %s", status, name)
		if duration != "" {
			fmt.Printf(" (%s)", duration)
		}
		fmt.Println()

		if pipeline.State != nil && pipeline.State.Name == "FAILED" {
			if err := cmd.showFailureDetails(prCtx, pipeline); err != nil {
				fmt.Printf("  └─ Unable to get failure details: %v\n", err)
			}
		}
	}

	return nil
}

func (cmd *ChecksCmd) getStatusIndicator(pipeline *api.Pipeline) string {
	if pipeline.State == nil {
		return "○ "
	}

	switch pipeline.State.Name {
	case "SUCCESSFUL":
		return "✓ "
	case "FAILED", "ERROR":
		return "✗ "
	case "IN_PROGRESS":
		return "● "
	case "PENDING":
		return "○ "
	case "STOPPED":
		return "◐ "
	default:
		return "○ "
	}
}

func (cmd *ChecksCmd) getPipelineName(pipeline *api.Pipeline) string {
	if pipeline.Target != nil {
		if pipeline.Target.RefName != "" {
			return fmt.Sprintf("Pipeline #%d (%s)", pipeline.BuildNumber, pipeline.Target.RefName)
		}
	}
	return fmt.Sprintf("Pipeline #%d", pipeline.BuildNumber)
}

func (cmd *ChecksCmd) getPipelineDuration(pipeline *api.Pipeline) string {
	if pipeline.CreatedOn == nil {
		return ""
	}

	if pipeline.CompletedOn != nil {
		duration := pipeline.CompletedOn.Sub(*pipeline.CreatedOn)
		return formatDuration(duration)
	}

	if pipeline.State != nil && pipeline.State.Name == "IN_PROGRESS" {
		duration := time.Since(*pipeline.CreatedOn)
		return formatDuration(duration)
	}

	return ""
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func (cmd *ChecksCmd) sortChecksByPriority(checks []*api.Pipeline) []*api.Pipeline {
	sorted := make([]*api.Pipeline, len(checks))
	copy(sorted, checks)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if cmd.getPipelinePriority(sorted[i]) > cmd.getPipelinePriority(sorted[j]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

func (cmd *ChecksCmd) getPipelinePriority(pipeline *api.Pipeline) int {
	if pipeline.State == nil {
		return 4
	}

	switch pipeline.State.Name {
	case "FAILED", "ERROR":
		return 0
	case "IN_PROGRESS":
		return 1
	case "PENDING":
		return 2
	case "SUCCESSFUL":
		return 3
	case "STOPPED":
		return 4
	default:
		return 4
	}
}

func (cmd *ChecksCmd) getChecksSummary(checks []*api.Pipeline) string {
	var successful, failed, running, pending, stopped int

	for _, pipeline := range checks {
		if pipeline.State == nil {
			pending++
			continue
		}

		switch pipeline.State.Name {
		case "SUCCESSFUL":
			successful++
		case "FAILED", "ERROR":
			failed++
		case "IN_PROGRESS":
			running++
		case "PENDING":
			pending++
		case "STOPPED":
			stopped++
		}
	}

	var parts []string
	if successful > 0 {
		parts = append(parts, fmt.Sprintf("%d successful", successful))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if stopped > 0 {
		parts = append(parts, fmt.Sprintf("%d stopped", stopped))
	}

	if len(parts) == 0 {
		return "no checks"
	}

	return strings.Join(parts, ", ")
}

func (cmd *ChecksCmd) showFailureDetails(prCtx *PRContext, pipeline *api.Pipeline) error {
	steps, err := prCtx.Client.Pipelines.GetPipelineSteps(context.Background(), prCtx.Workspace, prCtx.Repository, pipeline.UUID)
	if err != nil {
		return err
	}

	var failedSteps []*api.PipelineStep
	for _, step := range steps {
		if step.State != nil && (step.State.Name == "FAILED" || step.State.Name == "ERROR") {
			failedSteps = append(failedSteps, step)
		}
	}

	if len(failedSteps) > 0 {
		fmt.Printf("  └─ Failed steps:\n")
		for _, step := range failedSteps {
			stepName := step.Name
			if stepName == "" {
				stepName = "Unnamed step"
			}
			fmt.Printf("     • %s\n", stepName)
		}
	}

	return nil
}

func (cmd *ChecksCmd) formatJSON(prCtx *PRContext, checks []*api.Pipeline) error {
	output := map[string]interface{}{
		"checks": checks,
		"summary": map[string]interface{}{
			"total":      len(checks),
			"successful": cmd.countByStatus(checks, "SUCCESSFUL"),
			"failed":     cmd.countByStatus(checks, "FAILED") + cmd.countByStatus(checks, "ERROR"),
			"running":    cmd.countByStatus(checks, "IN_PROGRESS"),
			"pending":    cmd.countByStatus(checks, "PENDING"),
			"stopped":    cmd.countByStatus(checks, "STOPPED"),
		},
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ChecksCmd) formatYAML(prCtx *PRContext, checks []*api.Pipeline) error {
	output := map[string]interface{}{
		"checks": checks,
		"summary": map[string]interface{}{
			"total":      len(checks),
			"successful": cmd.countByStatus(checks, "SUCCESSFUL"),
			"failed":     cmd.countByStatus(checks, "FAILED") + cmd.countByStatus(checks, "ERROR"),
			"running":    cmd.countByStatus(checks, "IN_PROGRESS"),
			"pending":    cmd.countByStatus(checks, "PENDING"),
			"stopped":    cmd.countByStatus(checks, "STOPPED"),
		},
	}

	return prCtx.Formatter.Format(output)
}

func (cmd *ChecksCmd) countByStatus(checks []*api.Pipeline, status string) int {
	count := 0
	for _, pipeline := range checks {
		if pipeline.State != nil && pipeline.State.Name == status {
			count++
		}
	}
	return count
}
