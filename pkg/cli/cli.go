// Package cli implements the bhaus-util command-line interface: argument
// dispatch, the individual subcommands (lint, scaffold, ls) and log setup.
//
// The package owns all of the CLI's domain logic. This keeps
// cmd/bhaus-util/main.go a thin shim. Handlers return a process exit code.
// They do not call os.Exit directly. Run is the single owner of the process
// lifecycle. This design keeps the subcommands testable, since tests can
// call them without terminating the test binary.
package cli

import (
	"fmt"
	"os"

	"github.com/sjsone/bhaus-util/pkg/scaffold"
	"github.com/sjsone/bhaus-util/pkg/version"
)

// Run dispatches args to a subcommand. args must start with the program
// name (i.e. os.Args). Run returns the process exit code. main should pass
// the result straight to os.Exit.
func Run(args []string) int {
	if len(args) < 2 {
		printUsage()
		return 1
	}

	command := args[1]

	switch command {
	case "lint":
		return handleLint(args[2:])
	case "scaffold":
		return handleScaffold(args[2:])
	case "ls":
		return handleLS(args[2:])
	case "skills":
		fmt.Printf("  using `skill` command instead\n")
		fallthrough
	case "skill":
		return handleSkill(args[2:])
	case "version", "--version", "-v":
		fmt.Printf("bhaus-util %s\n", version.String())
		return 0
	case "help", "-h", "--help":
		// `bhaus-util help <command>` shows that command's detailed help.
		if len(args) >= 3 {
			return helpFor(args[2])
		}
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "bhaus-util: unknown command %q\n\n", command)
		printUsage()
		return 1
	}
}

// isHelpFlag reports whether an argument is a request for help.
func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

// helpFor prints the detailed help for a single subcommand. It backs
// `bhaus-util help <command>` (equivalent to `bhaus-util <command> --help`).
func helpFor(command string) int {
	switch command {
	case "lint":
		lintUsage(os.Stdout)
	case "scaffold":
		scaffoldUsage(os.Stdout)
	case "ls":
		lsUsage(os.Stdout)
	case "skill":
		skillUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "bhaus-util: unknown command %q\n\n", command)
		printUsage()
		return 1
	}
	return 0
}

// printUsage prints the top-level overview: what the tool is, the command map,
// and worked examples. Per-command flag detail lives in each subcommand's own
// usage (run `bhaus-util <command> --help`). This keeps the overview a
// scannable index.
func printUsage() {
	fmt.Printf(`bhaus-util — tooling for the BHaus modelling language

Parses .bhaus files into a typed AST and drives everything built on it: the
editor language server, linting and target-language scaffolding.

Usage:
  bhaus-util <command> [flags] [arguments]

Commands:
  ls          Run the language server over stdio (started by editors, not humans)
  lint        Check a .bhaus file for problems
  scaffold    Generate rough target-language source from .bhaus declarations
  skill       Manage agentic skill for BHaus support
  version     Print the build version
  help        Show this message or "help <command>" for one command's details

Examples:
  # See what a command expects
  bhaus-util scaffold --help

  # Generate Go from a model and print it to stdout
  bhaus-util scaffold go domain/user.bhaus

  # Generate TypeScript for every .bhaus file into a directory tree
  bhaus-util scaffold typescript --out-dir ./gen domain/*.bhaus

Scaffold languages: %v (plus any YAML defs found in --template-dir).

Run "bhaus-util <command> --help" for flags, defaults and per-command examples.
`, scaffold.Available())
}
