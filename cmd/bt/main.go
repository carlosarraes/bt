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
	Verbose    bool   `short:"v"`
	ConfigFile string `default:"~/.config/bt/config.yml"`
	NoColor    bool
	Help       bool   `short:"h"`
	VersionFlag bool  `name:"version" help:"Show version information"`
	LLM        bool   `help:"Show LLM-optimized usage guide and examples"`

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
	fmt.Print(`Usage: bt <command> [flags]

Work seamlessly with Bitbucket from the command line.

Flags:
  -h, --help              Show context-sensitive help.
  -v, --verbose           Enable verbose output
      --version           Show version information
      --config=PATH       Config file path
  -o, --output=FORMAT     Output format (table,json,yaml)
      --no-color          Disable colored output
      --llm               Show LLM-optimized usage guide and examples

Commands:
  auth
  run
  repo
  pr
  version

Run "bt <command> --help" for more information on a command.
Run "bt --llm" for LLM-optimized usage guidance.
`)
}

func showAuthHelp() {
	fmt.Print(`Usage: bt auth <command>

Authenticate bt and git with Bitbucket.

Commands:
  login
  logout
  status
  refresh

Run "bt auth <command> --help" for more information on a command.
`)
}

func showRunHelp() {
	fmt.Print(`Usage: bt run <command>

View and manage pipeline runs.

Commands:
  list
  view
  watch
  logs
  cancel

Run "bt run <command> --help" for more information on a command.
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
