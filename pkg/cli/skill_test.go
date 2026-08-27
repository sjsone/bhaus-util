package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// withStderr redirects os.Stderr for the duration of f. The skill subcommand
// writes its diagnostics there. Exit-code tests should not spray them over
// the test output.
func withStderr(t *testing.T, f func()) {
	t.Helper()
	orig := os.Stderr
	devnull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	os.Stderr = devnull
	defer func() { os.Stderr = orig; devnull.Close() }()
	f()
}

// quiet silences both streams. handleSkill uses stdout for help and stderr
// for errors.
func quiet(t *testing.T, f func()) {
	t.Helper()
	withStdout(t, func() { withStderr(t, f) })
}

// TestHandleSkillNoSubcommand is the regression guard for the reported panic:
// `bhaus-util skill` with no subcommand indexed args[0] on an empty slice.
func TestHandleSkillNoSubcommand(t *testing.T) {
	var code int
	quiet(t, func() { code = handleSkill(nil) })
	if code != 1 {
		t.Fatalf("handleSkill(nil) = %d, want 1", code)
	}
}

func TestHandleSkillExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no subcommand", []string{}, 1},
		{"unknown subcommand", []string{"bogus"}, 1},
		{"install without skill name", []string{"install"}, 1},
		{"install unknown skill", []string{"install", "no-such-skill"}, 1},
		{"help", []string{"--help"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var code int
			quiet(t, func() { code = handleSkill(tc.args) })
			if code != tc.want {
				t.Fatalf("handleSkill(%v) = %d, want %d", tc.args, code, tc.want)
			}
		})
	}
}

// TestHandleSkillInstallWritesSkill covers the success path end to end: the
// embedded skill tree is copied under <dir>/<skill name>/.
func TestHandleSkillInstallWritesSkill(t *testing.T) {
	dir := t.TempDir()

	var code int
	quiet(t, func() { code = handleSkill([]string{"install", "bhaus", "--dir", dir}) })
	if code != 0 {
		t.Fatalf("install exit = %d, want 0", code)
	}

	got := filepath.Join(dir, "bhaus", "SKILL.md")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("expected %s to exist: %v", got, err)
	}
}

// TestHandleSkillInstallRejectsBothTargets guards the mutually-exclusive flags.
func TestHandleSkillInstallRejectsBothTargets(t *testing.T) {
	var code int
	quiet(t, func() {
		code = handleSkill([]string{"install", "bhaus", "--dir", t.TempDir(), "--agent", "claude"})
	})
	if code != 1 {
		t.Fatalf("install with both --dir and --agent = %d, want 1", code)
	}
}

// TestHandleSkillInstallUnwritableTargetFails proves the install error is
// propagated rather than swallowed: a target under a regular file can never be
// created.
func TestHandleSkillInstallUnwritableTargetFails(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var code int
	quiet(t, func() {
		code = handleSkill([]string{"install", "bhaus", "--dir", filepath.Join(blocker, "sub")})
	})
	if code != 1 {
		t.Fatalf("install into unwritable target = %d, want 1", code)
	}
}
