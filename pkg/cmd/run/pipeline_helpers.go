package run

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/output"
)

func resolvePipelineUUID(ctx context.Context, runCtx *RunContext, pipelineID string) (string, error) {
	pipelineID = strings.TrimSpace(pipelineID)

	if strings.Contains(pipelineID, "-") {
		return pipelineID, nil
	}

	if strings.HasPrefix(pipelineID, "#") {
		pipelineID = pipelineID[1:]
	}

	buildNumber, err := strconv.Atoi(pipelineID)
	if err != nil {
		return "", fmt.Errorf("invalid pipeline ID '%s'. Expected build number (e.g., 123, #123) or UUID", pipelineID)
	}

	options := &api.PipelineListOptions{
		PageLen: 100,
		Page:    1,
		Sort:    "-created_on",
	}

	for options.Page <= 5 {
		result, err := runCtx.Client.Pipelines.ListPipelines(ctx, runCtx.Workspace, runCtx.Repository, options)
		if err != nil {
			return "", handlePipelineAPIError(err)
		}

		pipelines, err := parsePipelineResults(result)
		if err != nil {
			return "", fmt.Errorf("failed to parse pipeline results: %w", err)
		}

		for _, pipeline := range pipelines {
			if pipeline.BuildNumber == buildNumber {
				return pipeline.UUID, nil
			}
		}

		if result.Next == "" {
			break
		}
		options.Page++
	}

	return "", fmt.Errorf("pipeline with build number %d not found", buildNumber)
}

func displayStepInfo(step *api.PipelineStep) {
	fmt.Printf("\n=== Step: %s ===\n", step.Name)

	if step.State != nil {
		status := step.State.Name
		if step.State.Result != nil && step.State.Result.Name != "" {
			status = step.State.Result.Name
		}
		fmt.Printf("Status: %s\n", status)
	}

	if step.StartedOn != nil {
		fmt.Printf("Started: %s\n", step.StartedOn.Format("2006-01-02 15:04:05"))
	}

	if step.CompletedOn != nil {
		fmt.Printf("Completed: %s\n", step.CompletedOn.Format("2006-01-02 15:04:05"))
	}

	if step.BuildSecondsUsed > 0 {
		fmt.Printf("Duration: %s\n", output.FormatDuration(step.BuildSecondsUsed))
	}

	if step.Image != nil {
		fmt.Printf("Image: %s\n", step.Image.Name)
	}

	if len(step.SetupCommands) > 0 {
		fmt.Printf("\nSetup Commands:\n")
		for _, cmd := range step.SetupCommands {
			fmt.Printf("  - %s\n", cmd.Command)
		}
	}

	if len(step.ScriptCommands) > 0 {
		fmt.Printf("\nScript Commands:\n")
		for _, cmd := range step.ScriptCommands {
			fmt.Printf("  - %s\n", cmd.Command)
		}
	}

	fmt.Println()
}

func displayTestResults(ctx context.Context, runCtx *RunContext, pipeline *api.Pipeline, step *api.PipelineStep) {
	reports, err := runCtx.Client.Pipelines.GetStepTestReports(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
	if err != nil {
		fmt.Printf("No test reports available: %v\n", err)
		return
	}

	if len(reports) == 0 {
		fmt.Printf("No test reports found for this step\n")
		return
	}

	fmt.Printf("\n🧪 Test Reports Summary:\n")
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0

	for _, report := range reports {
		fmt.Printf("  Report: %s\n", report.Name)
		fmt.Printf("    Status: %s\n", report.Status)
		if report.Total > 0 {
			fmt.Printf("    Tests: %d total, %d passed, %d failed, %d skipped\n",
				report.Total, report.Passed, report.Failed, report.Skipped)
			totalPassed += report.Passed
			totalFailed += report.Failed
			totalSkipped += report.Skipped
		}
		if report.Duration > 0 {
			fmt.Printf("    Duration: %.2fs\n", report.Duration)
		}
		fmt.Println()
	}

	_ = totalPassed
	_ = totalSkipped

	if totalFailed > 0 {
		fmt.Printf("❌ Getting details for %d failed test(s)...\n\n", totalFailed)

		testCases, err := runCtx.Client.Pipelines.GetStepTestCases(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID)
		if err != nil {
			fmt.Printf("Could not get detailed test cases: %v\n", err)
			return
		}

		failedTests := 0
		for _, testCase := range testCases {
			if testCase.Status == "FAILED" || testCase.Result == "FAILED" {
				failedTests++
				fmt.Printf("❌ Test Failed: %s\n", testCase.Name)
				if testCase.ClassName != "" {
					fmt.Printf("   Class: %s\n", testCase.ClassName)
				}
				if testCase.TestSuite != "" {
					fmt.Printf("   Suite: %s\n", testCase.TestSuite)
				}
				if testCase.Duration > 0 {
					fmt.Printf("   Duration: %.2fs\n", testCase.Duration)
				}
				if testCase.Message != "" {
					fmt.Printf("   Message: %s\n", testCase.Message)
				}
				if testCase.Stacktrace != "" {
					fmt.Printf("   Stacktrace:\n%s\n", testCase.Stacktrace)
				}

				reasons, err := runCtx.Client.Pipelines.GetTestCaseReasons(ctx, runCtx.Workspace, runCtx.Repository, pipeline.UUID, step.UUID, testCase.UUID)
				if err == nil && len(reasons) > 0 {
					fmt.Printf("   Detailed Output:\n")
					for _, reason := range reasons {
						if reason.Message != "" {
							fmt.Printf("     %s\n", reason.Message)
						}
						if reason.Output != "" {
							fmt.Printf("     %s\n", reason.Output)
						}
					}
				}
				fmt.Println()
			}
		}

		if failedTests == 0 {
			fmt.Printf("Could not find detailed information for failed tests\n")
		}
	} else {
		fmt.Printf("✅ All tests passed!\n")
	}
}

func filterStepsByName(steps []*api.PipelineStep, stepName string) []*api.PipelineStep {
	var filtered []*api.PipelineStep
	for _, step := range steps {
		if matchesStepName(step.Name, stepName) {
			filtered = append(filtered, step)
		}
	}
	return filtered
}

func matchesStepName(stepName, requestedName string) bool {
	stepNameLower := strings.ToLower(stepName)
	requestedLower := strings.ToLower(requestedName)

	if stepNameLower == requestedLower {
		return true
	}

	if strings.Contains(stepNameLower, requestedLower) {
		return true
	}

	if strings.HasPrefix(stepNameLower, requestedLower) {
		return true
	}

	return false
}

func getAvailableStepNames(steps []*api.PipelineStep) string {
	names := make([]string, len(steps))
	for i, step := range steps {
		names[i] = step.Name
	}
	return strings.Join(names, ", ")
}
