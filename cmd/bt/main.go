package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/carlosarraes/bt/pkg/cmd"
	"github.com/carlosarraes/bt/pkg/version"
)

var cli struct {
	// Global flags
	Verbose     bool   `short:"v"`
	ConfigFile  string `default:"~/.config/bt/config.yml"`
	NoColor     bool
	Help        bool `short:"h"`
	VersionFlag bool `name:"version" help:"Show version information"`
	LLM         bool `help:"Show LLM-optimized usage guide and examples"`

	// Commands
	Version cmd.VersionCmd `cmd:""`
	Auth    cmd.AuthCmd    `cmd:""`
	Run     cmd.RunCmd     `cmd:""`
	Config  cmd.ConfigCmd  `cmd:""`
	Repo    cmd.RepoCmd    `cmd:""`
	PR      cmd.PRCmd      `cmd:""`
}

func main() {
	// Set up global context with configuration
	appCtx := context.Background()

	// Intercept help, version, and LLM requests before Kong processes them
	originalArgs := os.Args
	args := os.Args[1:]

	// Check for --version flag
	if len(args) >= 1 && args[0] == "--version" {
		fmt.Println(version.GetBuildInfo().String())
		return
	}

	// Check for global --llm flag
	if len(args) >= 1 && args[0] == "--llm" {
		showLLMHelp("")
		return
	}

	// Check for command-specific --llm flag
	if len(args) >= 2 && args[1] == "--llm" {
		showLLMHelp(args[0])
		return
	}

	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		showMainHelp()
		return
	}

	// Handle subcommand help
	if len(args) == 2 && args[1] == "--help" {
		switch args[0] {
		case "auth":
			showAuthHelp()
			return
		case "run":
			showRunHelp()
			return
		case "pr":
			showPRHelp()
			return
		case "config":
			showConfigHelp()
			return
		}
	}

	// Temporarily remove help, version, and llm flags from args to prevent Kong from intercepting
	filteredArgs := []string{originalArgs[0]}
	for _, arg := range args {
		if arg != "--help" && arg != "-h" && arg != "--version" && arg != "--llm" {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	os.Args = filteredArgs

	ctx := kong.Parse(&cli,
		kong.Name("bt"),
		kong.Description("Work seamlessly with Bitbucket from the command line."),
		kong.NoDefaultHelp(),
		kong.Vars{
			"version": version.Version,
		},
		kong.BindTo(appCtx, (*context.Context)(nil)),
	)

	// Update context with global flags
	if cli.Verbose {
		appCtx = context.WithValue(appCtx, "verbose", true)
	}
	if cli.NoColor {
		appCtx = context.WithValue(appCtx, "no-color", true)
	}
	appCtx = context.WithValue(appCtx, "config-path", cli.ConfigFile)

	// Check if help flag was set after Kong parsing
	if cli.Help {
		showMainHelp()
		return
	}

	// Check if version flag was set after Kong parsing
	if cli.VersionFlag {
		fmt.Println(version.GetBuildInfo().String())
		return
	}

	// Execute the selected command
	err := ctx.Run(appCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showMainHelp() {
	fmt.Print(`Work seamlessly with Bitbucket from the command line.

USAGE
  bt <command> <subcommand> [flags]

CORE COMMANDS
  auth:          Authenticate bt and git with Bitbucket
  pr:            Manage pull requests
  repo:          Manage repositories (not yet implemented)
  run:           View and manage pipeline runs

ADDITIONAL COMMANDS
  config:        Manage configuration for bt
  version:       Show bt version

FLAGS
  --help              Show help for command
  --version           Show bt version
  -v, --verbose       Enable verbose output
  --config-file=PATH  Config file path
  --no-color          Disable colored output
  --llm               Show LLM-optimized usage guide and examples

EXAMPLES
  $ bt pr create
  $ bt pr list --state open
  $ bt run view 123
  $ bt auth login

LEARN MORE
  Use 'bt <command> <subcommand> --help' for more information about a command.
  Run 'bt --llm' for LLM-optimized usage guidance and examples.
`)
}

func showAuthHelp() {
	fmt.Print(`Authenticate bt and git with Bitbucket.

USAGE
  bt auth <command> [flags]

AVAILABLE COMMANDS
  login:         Authenticate with Bitbucket
  logout:        Log out of Bitbucket
  status:        View authentication status
  refresh:       Refresh stored authentication credentials

FLAGS
  --help   Show help for command

EXAMPLES
  $ bt auth login
  $ bt auth status
  $ bt auth login --with-token YOUR_TOKEN

LEARN MORE
  Use 'bt auth <command> --help' for more information about a command.
`)
}

func showRunHelp() {
	fmt.Print(`View and manage pipeline runs.

USAGE
  bt run <command> [flags]

AVAILABLE COMMANDS
  list:          List pipeline runs
  view:          View details about a specific pipeline run
  watch:         Watch a pipeline run in real-time
  logs:          View logs for a pipeline run
  cancel:        Cancel a running pipeline
  rerun:         Rerun a pipeline (optionally failed steps only)
  report:        SonarCloud coverage/issues report for a pipeline

FLAGS
  -R, --repo [HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format
  --help                         Show help for command

INHERITED FLAGS
  -o, --output=FORMAT   Output format (table, json, yaml)
  --no-color           Disable colored output

EXAMPLES
  $ bt run list
  $ bt run view 123
  $ bt run report 123 --coverage
  $ bt run logs 123 --errors-only
  $ bt run watch 123

LEARN MORE
  Use 'bt run <command> --help' for more information about a command.
`)
}

func showPRHelp() {
	fmt.Print(`Work with Bitbucket pull requests.

USAGE
  bt pr <command> [flags]

GENERAL COMMANDS
  create:        Create a pull request
  list:          List pull requests in a repository
  list-all:      List pull requests across repositories in a workspace
  status:        Show status of relevant pull requests
  open:          Open pull requests in a browser
  report:        Pull request report

TARGETED COMMANDS
  checkout:      Check out a pull request in git
  checks:        Show CI status for a single pull request
  close:         Close a pull request
  comment:       Add a comment to a pull request
  diff:          View changes in a pull request
  edit:          Edit a pull request
  files:         List files changed in a pull request
  lock:          Lock pull request conversation
  merge:         Merge a pull request
  ready:         Mark a pull request as ready for review
  reopen:        Reopen a pull request
  review:        Add a review to a pull request
  unlock:        Unlock pull request conversation
  update-branch: Update a pull request branch
  view:          View a pull request

FLAGS
  -R, --repo [HOST/]OWNER/REPO   Select another repository using the [HOST/]OWNER/REPO format
  --help                         Show help for command

INHERITED FLAGS
  -o, --output=FORMAT   Output format (table, json, yaml)
  --no-color           Disable colored output

ARGUMENTS
  A pull request can be supplied as argument in any of the following formats:
  - by number, e.g. "123";
  - by URL, e.g. "https://bitbucket.org/WORKSPACE/REPO/pull-requests/123"; or
  - by the name of its head branch, e.g. "feature-branch".

EXAMPLES
  $ bt pr create
  $ bt pr create --fill
  $ bt pr create --ai --template portuguese
  $ bt pr list --state open
  $ bt pr view 123
  $ bt pr checkout 123
  $ bt pr merge 123

LEARN MORE
  Use 'bt pr <command> --help' for more information about a command.
`)
}

func showConfigHelp() {
	fmt.Print(`Manage configuration for bt.

USAGE
  bt config <command> [flags]

AVAILABLE COMMANDS
  get:           Get configuration values
  set:           Set configuration values
  list:          List configuration settings
  unset:         Remove configuration values

FLAGS
  --help   Show help for command

EXAMPLES
  $ bt config list
  $ bt config get auth.default_workspace
  $ bt config set auth.default_workspace myworkspace
  $ bt config unset auth.default_workspace

LEARN MORE
  Use 'bt config <command> --help' for more information about a command.
`)
}

func showLLMHelp(commandPath string) {
	// Parse command path to extract the main command
	parts := strings.Split(commandPath, " ")
	mainCommand := ""
	if len(parts) > 0 && parts[0] != "bt" {
		mainCommand = parts[0]
	}

	// Create and run LLM help
	llmHelp := &cmd.LLMHelp{
		Command: mainCommand,
	}

	// Use background context since we don't need specific configuration for help
	ctx := context.Background()
	llmHelp.Run(ctx)
}
