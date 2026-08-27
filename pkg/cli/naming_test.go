package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// binaryName is the one name the tool ships under. Every piece of user-facing
// text must use it: usage, error messages and examples. Users copy and paste
// these strings. A stale name sends them to a binary that does not exist.
const binaryName = "bhaus-util"

// staleNames are the names this tool used to be documented under. They must not
// reappear in user-facing output.
//
// The literals are split. This stops a repo-wide search-and-replace of an old
// name from rewriting this list and turning the guard into a tautology.
// That exact failure happened during the rename that introduced this guard.
var staleNames = []string{"bhaus-" + "cli", "bhaus-" + "ls"}

func TestSubcommandUsageUsesCanonicalName(t *testing.T) {
	usages := map[string]func(io.Writer){
		"lint":          lintUsage,
		"scaffold":      scaffoldUsage,
		"ls":            lsUsage,
		"skill":         skillUsage,
		"skill install": installSkillUsage,
	}

	for name, write := range usages {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			write(&buf)
			out := buf.String()

			if !strings.Contains(out, binaryName) {
				t.Errorf("%s usage never mentions %q", name, binaryName)
			}
			for _, stale := range staleNames {
				if strings.Contains(out, stale) {
					t.Errorf("%s usage still refers to %q", name, stale)
				}
			}
		})
	}
}

func TestTopLevelUsageUsesCanonicalName(t *testing.T) {
	out := captureStdout(t, func() { printUsage() })

	if !strings.Contains(out, binaryName) {
		t.Errorf("top-level usage never mentions %q", binaryName)
	}
	for _, stale := range staleNames {
		if strings.Contains(out, stale) {
			t.Errorf("top-level usage still refers to %q", stale)
		}
	}
}

// TestErrorMessagesUseCanonicalName covers the prefixes on the failure paths.
// These prefixes are what a user actually sees when something goes wrong.
func TestErrorMessagesUseCanonicalName(t *testing.T) {
	cases := map[string][]string{
		"unknown command":    {"frobnicate"},
		"unknown skill":      {"skill", "install", "no-such-skill"},
		"unknown subcommand": {"skill", "bogus"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			out := captureStderr(t, func() { Run(append([]string{binaryName}, args...)) })

			for _, stale := range staleNames {
				if strings.Contains(out, stale) {
					t.Errorf("error output still refers to %q:\n%s", stale, out)
				}
			}
			if !strings.Contains(out, binaryName) {
				t.Errorf("error output never mentions %q:\n%s", binaryName, out)
			}
		})
	}
}
