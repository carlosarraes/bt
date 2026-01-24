package pr

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type FilesCmd struct {
	PRID string `arg:"" name:"pr-id" help:"Pull request ID or number (e.g., 123 or #123)"`

	NameOnly bool   `help:"Show only file names"`
	Filter   string `help:"Filter files by pattern (e.g., '*.go', 'src/**/*.js')"`

	Output     string `short:"o" help:"Output format (table, json, yaml)" default:"table"`
	Workspace  string `help:"Bitbucket workspace (defaults to current repository)"`
	Repository string `help:"Repository name (defaults to current repository)"`
}

func (c *FilesCmd) Run(ctx context.Context) error {
	prCtx, err := shared.NewCommandContext(ctx, c.Output, false)
	if err != nil {
		return fmt.Errorf("failed to create PR context: %w", err)
	}

	prID := strings.TrimPrefix(c.PRID, "#")
	id, err := strconv.Atoi(prID)
	if err != nil {
		return fmt.Errorf("invalid PR ID '%s': must be a number", c.PRID)
	}

	workspace := c.Workspace
	repository := c.Repository
	
	if workspace == "" {
		workspace = prCtx.Workspace
	}
	if repository == "" {
		repository = prCtx.Repository
	}

	if workspace == "" || repository == "" {
		return fmt.Errorf("workspace and repository are required (use --workspace and --repository or run from a git repository)")
	}

	diffStat, err := prCtx.Client.PullRequests.GetPullRequestFiles(ctx, workspace, repository, id)
	if err != nil {
		return fmt.Errorf("failed to get PR files: %w", err)
	}

	files := diffStat.Files

	if c.Filter != "" {
		filtered := make([]*api.PullRequestFile, 0)
		for _, file := range files {
			path := file.NewPath
			if path == "" {
				path = file.OldPath
			}
			if c.matchesFilter(path, c.Filter) {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}

	return c.formatOutput(prCtx, files)
}

func (c *FilesCmd) matchesFilter(path, pattern string) bool {
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err != nil {
		return true
	}
	if matched {
		return true
	}

	matched, _ = filepath.Match(pattern, path)
	return matched
}

func (c *FilesCmd) formatOutput(prCtx *PRContext, files []*api.PullRequestFile) error {
	if c.Output == "json" || c.Output == "yaml" {
		output := struct {
			Files       []*api.PullRequestFile `json:"files" yaml:"files"`
			TotalFiles  int                    `json:"total_files" yaml:"total_files"`
			TotalAdded  int                    `json:"total_added" yaml:"total_added"`
			TotalRemoved int                   `json:"total_removed" yaml:"total_removed"`
		}{
			Files:      files,
			TotalFiles: len(files),
		}

		for _, file := range files {
			output.TotalAdded += file.LinesAdded
			output.TotalRemoved += file.LinesRemoved
		}

		return prCtx.Formatter.Format(output)
	}

	if c.NameOnly {
		for _, file := range files {
			path := file.NewPath
			if path == "" {
				path = file.OldPath
			}
			fmt.Println(path)
		}
		return nil
	}

	return c.formatTable(prCtx, files)
}

func (c *FilesCmd) formatTable(prCtx *PRContext, files []*api.PullRequestFile) error {
	if len(files) == 0 {
		fmt.Println("No files changed in this pull request")
		return nil
	}

	totalAdded := 0
	totalRemoved := 0

	headers := []string{"STATUS", "FILE", "+LINES", "-LINES"}
	var rows [][]string

	for _, file := range files {
		status := c.getFileStatus(file)
		added := fmt.Sprintf("%d", file.LinesAdded)
		removed := fmt.Sprintf("%d", file.LinesRemoved)
		
		totalAdded += file.LinesAdded
		totalRemoved += file.LinesRemoved

		path := file.NewPath
		if path == "" {
			path = file.OldPath
		}

		rows = append(rows, []string{
			status,
			path,
			added,
			removed,
		})
	}

	rows = append(rows, []string{
		"",
		fmt.Sprintf("Total: %d files", len(files)),
		fmt.Sprintf("%d", totalAdded),
		fmt.Sprintf("%d", totalRemoved),
	})

	return c.renderTable(headers, rows)
}

func (c *FilesCmd) renderTable(headers []string, rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

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

	for i, header := range headers {
		fmt.Printf("%-*s", colWidths[i], header)
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	for i := range headers {
		fmt.Print(strings.Repeat("-", colWidths[i]))
		if i < len(headers)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()

	for _, row := range rows {
		for i, cell := range row {
			if i == 0 && len(cell) > 0 {
				switch cell {
				case "A":
					cell = "\033[32m" + cell + "\033[0m"
				case "M":
					cell = "\033[33m" + cell + "\033[0m"
				case "D":
					cell = "\033[31m" + cell + "\033[0m"
				}
			}
			if (i == 2 || i == 3) && cell != "0" && cell != "" {
				if i == 2 {
					cell = "\033[32m+" + cell + "\033[0m"
				} else {
					cell = "\033[31m-" + cell + "\033[0m"
				}
			}
			
			fmt.Printf("%-*s", colWidths[i], cell)
			if i < len(row)-1 {
				fmt.Print("  ")
			}
		}
		fmt.Println()
	}

	return nil
}

func (c *FilesCmd) getFileStatus(file *api.PullRequestFile) string {
	hasAdditions := file.LinesAdded > 0
	hasRemovals := file.LinesRemoved > 0

	if file.Status != "" {
		switch strings.ToLower(file.Status) {
		case "added":
			return "A"
		case "removed", "deleted":
			return "D"
		case "modified":
			return "M"
		case "renamed":
			return "R"
		default:
			return "M"
		}
	}

	if hasAdditions && !hasRemovals {
		return "A"
	} else if !hasAdditions && hasRemovals {
		return "D"
	} else {
		return "M"
	}
}
