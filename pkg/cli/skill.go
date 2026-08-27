// Package cli implements the bhaus-util command-line interface: argument
// dispatch, the individual subcommands (lint, scaffold, ls, skill) and log setup.

package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/skill"
)

// handleSkill dispatches the "skill" subcommand. args are the arguments
// following "skill" (i.e. os.Args[2:]).
func handleSkill(args []string) int {
	if len(args) == 0 {
		skillUsage(os.Stderr)
		return 1
	}

	if isHelpFlag(args[0]) {
		skillUsage(os.Stdout)
		return 0
	}

	switch args[0] {
	case "install":
		return installSkill(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "bhaus-util skill: unknown subcommand %q\n\n", args[0])
		skillUsage(os.Stderr)
		return 1
	}
}

// resolveTargetDirectoryPath decides where a skill is installed. An explicit
// dir always wins. Otherwise the agent name decides the location. An empty
// agent falls back to the generic ~/.agents/skills directory.
func resolveTargetDirectoryPath(dir, agent string) (string, error) {
	if dir != "" {
		return dir, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	agentsSkills := filepath.Join(u.HomeDir, ".agents", "skills")

	switch agent {
	case "", "zed":
		return agentsSkills, nil
	case "claude":
		return filepath.Join(u.HomeDir, ".claude", "skills"), nil
	default:
		return "", fmt.Errorf("unknown agent %q; known agents: claude, zed", agent)
	}
}

// installSkill copies one embedded skill (or every skill, for "all") into the
// target directory. args are the arguments following "install".
func installSkill(args []string) int {
	fs := flag.NewFlagSet("skill install", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { installSkillUsage(os.Stderr) }
	dir := fs.String("dir", "", "target directory to install the skill into")
	agent := fs.String("agent", "", "install into this agent's skill directory (mutually exclusive with --dir)")

	if len(args) == 0 {
		installSkillUsage(os.Stderr)
		return 1
	}
	if isHelpFlag(args[0]) {
		installSkillUsage(os.Stdout)
		return 0
	}

	name := args[0]
	if name == "" {
		fmt.Fprint(os.Stderr, "bhaus-util skill install: no skill name given\n\n")
		installSkillUsage(os.Stderr)
		return 1
	}

	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	if *agent != "" && *dir != "" {
		fmt.Fprint(os.Stderr, "bhaus-util skill install: pass either --dir or --agent, not both\n")
		return 1
	}

	targetDirectory, err := resolveTargetDirectoryPath(*dir, *agent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util skill install: %v\n", err)
		return 1
	}

	available := skill.GetAvailableSkills()
	if len(available) == 0 {
		fmt.Fprint(os.Stderr, "bhaus-util skill install: no skills are bundled with this build\n")
		return 1
	}

	var selected []*skill.Skill
	if name == "all" {
		selected = available
	} else {
		for _, s := range available {
			if s.Name == name {
				selected = append(selected, s)
				break
			}
		}
	}

	if len(selected) == 0 {
		fmt.Fprintf(os.Stderr, "bhaus-util skill install: unknown skill %q; available: %s\n",
			name, strings.Join(skillNames(available), ", "))
		return 1
	}

	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util skill install: cannot create %s: %v\n", targetDirectory, err)
		return 1
	}

	for _, s := range selected {
		if err := skill.InstallSkill(s, targetDirectory); err != nil {
			fmt.Fprintf(os.Stderr, "bhaus-util skill install: %s: %v\n", s.Name, err)
			return 1
		}
		fmt.Printf("installed skill %q into %s\n", s.Name, filepath.Join(targetDirectory, s.Name))
	}

	return 0
}

// skillNames returns the skill names, for use in error messages.
func skillNames(skills []*skill.Skill) []string {
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}
	return names
}

func skillUsage(w io.Writer) {
	fmt.Fprint(w, `bhaus-util skill — manage the bundled BHaus agent skills

Usage:
  bhaus-util skill <command> [flags]

A skill teaches an AI coding agent to generate and update source code from
.bhaus design files. The skills are embedded in this binary, so installing one
only copies files — it needs no network access.

Commands:
  install   Copy a bundled skill into an agent's skill directory.

Examples:
  # Install the bhaus skill for Claude Code
  bhaus-util skill install bhaus --agent claude

  # Show the flags for install
  bhaus-util skill install --help
`)
}

func installSkillUsage(w io.Writer) {
	fmt.Fprint(w, `bhaus-util skill install — install a bundled BHaus skill

Usage:
  bhaus-util skill install <skill|all> [flags]

Copies the named skill into a target directory, creating it if needed. Pass
"all" to install every bundled skill. Existing files are overwritten.

Flags:
  --dir <path>      Install into this directory.
  --agent <name>    Install into a known agent's skill directory instead.
                    One of: claude (~/.claude/skills), zed (~/.agents/skills).
                    Mutually exclusive with --dir.

With neither flag, skills go to ~/.agents/skills.

Examples:
  # Install for Claude Code
  bhaus-util skill install bhaus --agent claude

  # Install every skill into a project-local directory
  bhaus-util skill install all --dir ./.claude/skills
`)
}
