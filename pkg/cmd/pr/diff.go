package pr

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/output"
	"github.com/carlosarraes/bt/pkg/utils"
)


type DiffCmd struct {
	PRID       string `arg:"" help:"Pull request ID (number)"`
	NameOnly   bool   `name:"name-only" help:"Show only names of changed files"`
	Patch      bool   `help:"Output in patch format suitable for git apply"`
	File       string `help:"Show diff for specific file only"`
	Color      string `help:"When to use color (always, never, auto)" enum:"always,never,auto" default:"auto"`
	Output     string `short:"o" help:"Output format (diff, json, yaml)" enum:"diff,json,yaml" default:"diff"`
NoColor    bool
	Workspace  string `help:"Bitbucket workspace (defaults to git remote or config)"`
	Repository string `help:"Repository name (defaults to git remote)"`
}


func (cmd *DiffCmd) Run(ctx context.Context) error {

prCtx, err := NewPRContext(ctx, "table", cmd.NoColor)
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


	switch {
	case cmd.NameOnly:
		return cmd.outputNameOnly(diff)
	case cmd.Output == "json":
		return cmd.outputJSON(prCtx, diff, prID)
	case cmd.Output == "yaml":
		return cmd.outputYAML(prCtx, diff, prID)
	case cmd.Patch:
		return cmd.outputPatch(diff)
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
