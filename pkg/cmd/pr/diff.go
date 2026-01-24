package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/cmd/shared"
	"github.com/carlosarraes/bt/pkg/output"
	"github.com/carlosarraes/bt/pkg/utils"
)

type DiffCmd struct {
	PRID         string `arg:"" help:"Pull request ID (number)"`
	NameOnly     bool   `name:"name-only" help:"Show only names of changed files"`
	Patch        bool   `help:"Output in patch format suitable for git apply"`
	File         string `help:"Show diff for specific file only"`
	Color        string `help:"When to use color (always, never, auto)" enum:"always,never,auto" default:"auto"`
	Output       string `short:"o" help:"Output format (diff, json, yaml)" enum:"diff,json,yaml" default:"diff"`
	Page         bool   `help:"Page output through diff-so-fancy and less for enhanced viewing"`
	IncludeTests bool   `name:"include-tests" help:"Include test files in diff (excluded by default)"`
	NoColor      bool
	Workspace    string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository   string `help:"Repository name (defaults to git remote)"`
}

func (cmd *DiffCmd) Run(ctx context.Context) error {

	prCtx, err := shared.NewCommandContext(ctx, "table", cmd.NoColor)
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

	diff, err := prCtx.Client.PullRequests.GetPullRequestDiff(ctx, prCtx.Workspace, prCtx.Repository, prID)
	if err != nil {
		return handlePullRequestAPIError(err)
	}

	if diff == "" {
		fmt.Println("No differences found in this pull request.")
		return nil
	}

	if !cmd.IncludeTests {
		diff = cmd.filterTestFiles(diff)
		if diff == "" {
			fmt.Println("No non-test changes found in this pull request.")
			return nil
		}
	}

	switch {
	case cmd.NameOnly:
		return cmd.outputNameOnly(diff)
	case cmd.Output == "json":
		return cmd.outputJSON(prCtx, diff, prID)
	case cmd.Output == "yaml":
		return cmd.outputYAML(prCtx, diff, prID)
	case cmd.Patch:
		return cmd.outputPatch(diff)
	case cmd.Page:
		return cmd.outputWithPager(diff)
	default:
		return cmd.outputColoredDiff(diff)
	}
}

func (cmd *DiffCmd) ParsePRID() (int, error) {
	if cmd.PRID == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

	prIDStr := strings.TrimPrefix(cmd.PRID, "#")

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s': must be a positive integer", cmd.PRID)
	}

	if prID <= 0 {
		return 0, fmt.Errorf("pull request ID must be positive, got %d", prID)
	}

	return prID, nil
}

func (cmd *DiffCmd) outputNameOnly(diff string) error {
	files := utils.ExtractChangedFiles(diff)

	if cmd.File != "" {
		filteredFiles := make([]string, 0)
		for _, file := range files {
			if strings.Contains(file, cmd.File) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
	}

	for _, file := range files {
		fmt.Println(file)
	}
	return nil
}

func (cmd *DiffCmd) outputPatch(diff string) error {

	if cmd.File != "" {
		diff = utils.FilterDiffByFile(diff, cmd.File)
	}

	cleanDiff := utils.CleanDiffForPatch(diff)
	fmt.Print(cleanDiff)
	return nil
}

func (cmd *DiffCmd) outputColoredDiff(diff string) error {

	if cmd.File != "" {
		diff = utils.FilterDiffByFile(diff, cmd.File)
		if diff == "" {
			fmt.Printf("No differences found for file: %s\n", cmd.File)
			return nil
		}
	}

	useColors := cmd.shouldUseColors()

	formattedDiff := utils.FormatDiff(diff, useColors)
	fmt.Print(formattedDiff)
	return nil
}

func (cmd *DiffCmd) outputJSON(prCtx *PRContext, diff string, prID int) error {

	if cmd.File != "" {
		diff = utils.FilterDiffByFile(diff, cmd.File)
	}

	diffData := map[string]interface{}{
		"pull_request_id": prID,
		"workspace":       prCtx.Workspace,
		"repository":      prCtx.Repository,
		"diff":            diff,
		"files_changed":   utils.ExtractChangedFiles(diff),
		"stats":           utils.CalculateDiffStats(diff),
	}

	if cmd.File != "" {
		diffData["filtered_file"] = cmd.File
	}

	return prCtx.Formatter.Format(diffData)
}

func (cmd *DiffCmd) outputYAML(prCtx *PRContext, diff string, prID int) error {

	yamlFormatter, err := output.NewFormatter(output.FormatYAML, &output.FormatterOptions{
		NoColor: cmd.NoColor,
	})
	if err != nil {
		return fmt.Errorf("failed to create YAML formatter: %w", err)
	}

	if cmd.File != "" {
		diff = utils.FilterDiffByFile(diff, cmd.File)
	}

	diffData := map[string]interface{}{
		"pull_request_id": prID,
		"workspace":       prCtx.Workspace,
		"repository":      prCtx.Repository,
		"diff":            diff,
		"files_changed":   utils.ExtractChangedFiles(diff),
		"stats":           utils.CalculateDiffStats(diff),
	}

	if cmd.File != "" {
		diffData["filtered_file"] = cmd.File
	}

	return yamlFormatter.Format(diffData)
}

func (cmd *DiffCmd) shouldUseColors() bool {
	switch cmd.Color {
	case "always":
		return true
	case "never":
		return false
	case "auto":

		return !cmd.NoColor && utils.IsTerminal(os.Stdout)
	default:
		return false
	}
}

func (cmd *DiffCmd) outputWithPager(diff string) error {
	if cmd.File != "" {
		diff = utils.FilterDiffByFile(diff, cmd.File)
		if diff == "" {
			fmt.Printf("No differences found for file: %s\n", cmd.File)
			return nil
		}
	}

	if _, err := exec.LookPath("less"); err != nil {
		return fmt.Errorf("less is not installed or not in PATH")
	}

	coloredDiff := utils.FormatDiff(diff, true)

	hasDiffSoFancy := false
	if _, err := exec.LookPath("diff-so-fancy"); err == nil {
		hasDiffSoFancy = true
	}

	if hasDiffSoFancy {
		diffSoFancyCmd := exec.Command("diff-so-fancy")
		diffSoFancyCmd.Stdin = strings.NewReader(coloredDiff)
		diffSoFancyCmd.Env = append(os.Environ(), "FORCE_COLOR=1")

		lessCmd := exec.Command("less", "--tabs=2", "-RFX")

		pipe, err := diffSoFancyCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create pipe: %w", err)
		}
		lessCmd.Stdin = pipe

		lessCmd.Stdout = os.Stdout
		lessCmd.Stderr = os.Stderr

		if err := lessCmd.Start(); err != nil {
			return fmt.Errorf("failed to start less: %w", err)
		}

		if err := diffSoFancyCmd.Start(); err != nil {
			return fmt.Errorf("failed to start diff-so-fancy: %w", err)
		}

		if err := diffSoFancyCmd.Wait(); err != nil {
			return fmt.Errorf("diff-so-fancy failed: %w", err)
		}

		pipe.Close()

		if err := lessCmd.Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() == 1 {
					return nil
				}
			}
			return fmt.Errorf("less failed: %w", err)
		}
	} else {
		lessCmd := exec.Command("less", "--tabs=2", "-RFX")
		lessCmd.Stdin = strings.NewReader(coloredDiff)
		lessCmd.Stdout = os.Stdout
		lessCmd.Stderr = os.Stderr

		if err := lessCmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() == 1 {
					return nil
				}
			}
			return fmt.Errorf("less failed: %w", err)
		}
	}

	return nil
}

func (cmd *DiffCmd) filterTestFiles(diff string) string {
	var result strings.Builder
	lines := strings.Split(diff, "\n")
	inTestFile := false

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filename := strings.TrimPrefix(parts[3], "b/")

				inTestFile = isTestFile(filename)
			}
		}

		if !inTestFile {
			if result.Len() > 0 && !strings.HasSuffix(result.String(), "\n") {
				result.WriteString("\n")
			}
			result.WriteString(line)
		}
	}

	return strings.TrimSpace(result.String())
}

func isTestFile(filename string) bool {
	testPatterns := []string{
		"_test.go",
		".test.js",
		".test.ts",
		".test.jsx",
		".test.tsx",
		".spec.js",
		".spec.ts",
		".spec.jsx",
		".spec.tsx",
		"_spec.rb",
		"test_",
		"Test.java",
		".test.py",
		"_test.py",
		"test/",
		"tests/",
		"__tests__/",
		"spec/",
		"specs/",
	}

	lowerFilename := strings.ToLower(filename)

	for _, pattern := range testPatterns {
		lowerPattern := strings.ToLower(pattern)
		if strings.HasSuffix(lowerFilename, lowerPattern) ||
			strings.Contains(lowerFilename, lowerPattern) ||
			strings.HasPrefix(lowerFilename, lowerPattern) {
			return true
		}
	}

	return false
}
